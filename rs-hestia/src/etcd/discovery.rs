use std::any::Any;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;

use async_trait::async_trait;
use etcd_client::{Client, EventType, GetOptions, WatchOptions};
use tokio::sync::RwLock;

use crate::Context;
use crate::discovery::Discovery;
use crate::error::{HestiaError, Result};
use crate::service::{Service, StrategyHandler};

use super::{Options, new_etcd_client, normalize_prefix};

/// etcd-backed [`Discovery`] implementation.
#[derive(Clone)]
pub struct EtcdDiscovery {
    client: Client,
    prefix: String,
    disable_watch: bool,
    service_list: Arc<RwLock<HashMap<String, Vec<Service>>>>,
}

/// Creates a new etcd-backed discovery client.
pub async fn new_discovery(options: Options) -> Result<Arc<dyn Discovery>> {
    let client = new_etcd_client(&options).await?;
    let prefix = normalize_prefix(&options.prefix);

    Ok(Arc::new(EtcdDiscovery {
        client,
        prefix,
        disable_watch: options.disable_watch,
        service_list: Arc::new(RwLock::new(HashMap::with_capacity(20))),
    }))
}

#[async_trait]
impl Discovery for EtcdDiscovery {
    async fn get_services(&self, ctx: &Context, name: &str, version: &str) -> Result<Vec<Service>> {
        if !self.disable_watch {
            let list = self.service_list.read().await;
            if let Some(services) = list.get(name) {
                return Ok(services.clone());
            }
        }

        let services = self.get_services_by_name(ctx, name, version).await?;
        if services.is_empty() {
            return Err(HestiaError::ServicesNotFound);
        }

        if !self.disable_watch {
            let mut list = self.service_list.write().await;
            list.insert(name.to_string(), services.clone());

            // Spawn a detached watch so that it survives caller cancellation.
            let this = self.clone();
            let name = name.to_string();
            let version = version.to_string();
            tokio::spawn(async move {
                this.watch(&name, &version).await;
            });
        }

        Ok(services)
    }

    async fn get(
        &self,
        ctx: &Context,
        name: &str,
        version: &str,
        strategy: Option<StrategyHandler>,
    ) -> Result<Service> {
        let services = self.get_services(ctx, name, version).await?;
        let handler = strategy.unwrap_or_else(crate::service::round_robin_handler);
        handler(&services).ok_or(HestiaError::ServicesNotFound)
    }

    fn name(&self) -> &str {
        "etcd"
    }

    fn as_any(&self) -> &dyn Any {
        self
    }
}

impl EtcdDiscovery {
    /// Watches the service prefix and updates the local cache on every change.
    async fn watch(&self, name: &str, version: &str) {
        let name = name.to_string();
        let version = version.to_string();
        self.watch_with_callback(&name, &version, move |_services| {
            // Internal cache update is intentionally a no-op here because
            // `get_services` already writes to the cache. This callback
            // exists so the gRPC resolver can share the same watch path.
        })
        .await;
    }

    /// Watches the service prefix and invokes `callback` on every change.
    ///
    /// This is `pub(crate)` so that the etcd resolver can receive real-time
    /// updates without exposing watch semantics in the public `Discovery` API.
    pub(crate) async fn watch_with_callback(
        &self,
        name: &str,
        version: &str,
        callback: impl Fn(Vec<Service>) + Send + 'static,
    ) {
        let key = discovery_key(&self.prefix, name, version);
        let mut watch_client = self.client.watch_client();
        let mut stream = match watch_client
            .watch(key.as_str(), Some(WatchOptions::new().with_prefix()))
            .await
        {
            Ok(ws) => ws,
            Err(e) => {
                tracing::error!("etcd watch failed: {}", e);
                return;
            }
        };

        loop {
            match stream.message().await {
                Ok(Some(resp)) => {
                    for event in resp.events() {
                        match event.event_type() {
                            EventType::Put | EventType::Delete => {
                                let ctx = Context::new();
                                match self.get_services_by_name(&ctx, name, version).await {
                                    Ok(services) => {
                                        let mut list = self.service_list.write().await;
                                        list.insert(name.to_string(), services.clone());
                                        callback(services);
                                    }
                                    Err(e) => {
                                        tracing::error!(
                                            "reload etcd prefix:{} services error:{}",
                                            key,
                                            e
                                        );
                                    }
                                }
                            }
                        }
                    }
                }
                Ok(None) => break,
                Err(e) => {
                    tracing::error!("etcd watch stream error: {}", e);
                    break;
                }
            }
        }
    }

    async fn get_services_by_name(
        &self,
        _ctx: &Context,
        name: &str,
        version: &str,
    ) -> Result<Vec<Service>> {
        let key = discovery_key(&self.prefix, name, version);
        let mut kv = self.client.kv_client();
        let resp = tokio::time::timeout(
            Duration::from_secs(15),
            kv.get(key.as_str(), Some(GetOptions::new().with_prefix())),
        )
        .await
        .map_err(|e| HestiaError::Other(format!("get services timeout: {}", e)))??;

        let kvs = resp.kvs();
        let mut services = Vec::with_capacity(kvs.len());
        for kv in kvs {
            let entry: Service = match serde_json::from_slice(kv.value()) {
                Ok(s) => s,
                Err(e) => {
                    tracing::warn!("unmarshal service failed, error: {}", e);
                    continue;
                }
            };
            if entry.healthy {
                services.push(entry);
            }
        }

        Ok(services)
    }
}

fn discovery_key(prefix: &str, name: &str, version: &str) -> String {
    if version.is_empty() {
        format!("{}/{}/", prefix, name)
    } else {
        format!("{}/{}/{}/", prefix, name, version)
    }
}
