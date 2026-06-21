use std::any::Any;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;

use async_trait::async_trait;
use serde::Deserialize;
use tokio::sync::RwLock;

use crate::Context;
use crate::discovery::Discovery;
use crate::error::{HestiaError, Result};
use crate::service::{ProtocolType, Service, StrategyHandler};

use super::{Options, base_url, new_http_client};

/// Consul-backed [`Discovery`] implementation.
#[derive(Clone)]
pub struct ConsulDiscovery {
    client: reqwest::Client,
    options: Options,
    service_list: Arc<RwLock<HashMap<String, Vec<Service>>>>,
}

/// Creates a new Consul-backed discovery client.
pub async fn new_discovery(options: Options) -> Result<Arc<dyn Discovery>> {
    let client = new_http_client(&options)?;
    Ok(Arc::new(ConsulDiscovery {
        client,
        options,
        service_list: Arc::new(RwLock::new(HashMap::with_capacity(20))),
    }))
}

#[async_trait]
impl Discovery for ConsulDiscovery {
    async fn get_services(&self, ctx: &Context, name: &str, version: &str) -> Result<Vec<Service>> {
        if !self.options.disable_watch {
            let list = self.service_list.read().await;
            if let Some(services) = list.get(name) {
                return Ok(services.clone());
            }
        }

        let services = self.get_services_by_name(ctx, name, version).await?;
        if services.is_empty() {
            return Err(HestiaError::ServicesNotFound);
        }

        if !self.options.disable_watch {
            let mut list = self.service_list.write().await;
            list.insert(name.to_string(), services.clone());

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

    fn string(&self) -> &str {
        "consul"
    }

    fn as_any(&self) -> &dyn Any {
        self
    }
}

impl ConsulDiscovery {
    async fn watch(&self, name: &str, version: &str) {
        let name = name.to_string();
        let version = version.to_string();
        self.watch_with_callback(&name, &version, move |_services| {
            // Internal cache update is intentionally a no-op here because
            // `get_services` already writes to the cache.
        })
        .await;
    }

    pub(crate) async fn watch_with_callback(
        &self,
        name: &str,
        version: &str,
        callback: impl Fn(Vec<Service>) + Send + 'static,
    ) {
        let wait = "30s";
        let mut index: Option<String> = None;

        loop {
            let url = health_url(&self.options, name, version, index.as_deref(), Some(wait));
            let mut req = self.client.get(&url);
            if !self.options.token.is_empty() {
                req = req.header("X-Consul-Token", &self.options.token);
            }
            // Blocking queries may wait up to `wait`; allow extra headroom.
            req = req.timeout(Duration::from_secs(45));

            match req.send().await {
                Ok(resp) => {
                    if resp.status().is_success() {
                        let new_index = resp
                            .headers()
                            .get("x-consul-index")
                            .and_then(|v| v.to_str().ok())
                            .map(String::from);

                        match resp.json::<Vec<HealthEntry>>().await {
                            Ok(entries) => {
                                let services = entries_to_services(entries, version);
                                {
                                    let mut list = self.service_list.write().await;
                                    list.insert(name.to_string(), services.clone());
                                }
                                callback(services);

                                index = new_index;
                                continue;
                            }
                            Err(e) => {
                                log::error!("consul watch decode error: {}", e);
                            }
                        }
                    } else {
                        log::error!("consul watch returned status: {}", resp.status());
                    }
                }
                Err(e) => {
                    if e.is_timeout() {
                        // Timeout likely means no change within the wait window.
                        // Re-fetch with the same index.
                        continue;
                    }
                    log::error!("consul watch request error: {}", e);
                }
            }

            // Back off on error before retrying.
            tokio::time::sleep(Duration::from_secs(3)).await;
        }
    }

    async fn get_services_by_name(
        &self,
        _ctx: &Context,
        name: &str,
        version: &str,
    ) -> Result<Vec<Service>> {
        let url = health_url(&self.options, name, version, None, None);
        let mut req = self.client.get(&url);
        if !self.options.token.is_empty() {
            req = req.header("X-Consul-Token", &self.options.token);
        }
        req = req.timeout(self.options.dial_timeout);

        let resp = req
            .send()
            .await
            .map_err(HestiaError::Consul)?
            .error_for_status()
            .map_err(HestiaError::Consul)?;

        let entries: Vec<HealthEntry> = resp.json().await.map_err(HestiaError::Consul)?;
        Ok(entries_to_services(entries, version))
    }
}

fn health_url(
    options: &Options,
    name: &str,
    version: &str,
    index: Option<&str>,
    wait: Option<&str>,
) -> String {
    let mut url = format!("{}/v1/health/service/{}", base_url(options), name);
    let mut params = Vec::new();
    params.push("passing=true".to_string());
    if !options.datacenter.is_empty() {
        params.push(format!("dc={}", options.datacenter));
    }
    if !version.is_empty() {
        params.push(format!("tag=version:{}", version));
    }
    if let Some(idx) = index {
        params.push(format!("index={}", idx));
    }
    if let Some(w) = wait {
        params.push(format!("wait={}", w));
    }
    if !params.is_empty() {
        url.push('?');
        url.push_str(&params.join("&"));
    }
    url
}

fn entries_to_services(entries: Vec<HealthEntry>, version: &str) -> Vec<Service> {
    let mut services = Vec::with_capacity(entries.len());
    for entry in entries {
        if let Some(svc) = entry_to_service(entry, version) {
            services.push(svc);
        }
    }
    services
}

fn entry_to_service(entry: HealthEntry, version: &str) -> Option<Service> {
    let service = entry.service;
    let svc_version = tag_value(&service.tags, "version:").unwrap_or_default();

    if !version.is_empty() && svc_version != version {
        return None;
    }

    let host = if service.address.is_empty() {
        entry.node.address
    } else {
        service.address
    };
    let address = format!("{}:{}", host, service.port);

    let protocol = tag_value(&service.tags, "protocol:")
        .map(ProtocolType::from)
        .unwrap_or_default();

    let instance_id =
        tag_value(&service.tags, "instance_id:").unwrap_or_else(|| service.id.clone());

    Some(Service {
        network: "tcp".to_string(),
        name: service.service,
        address,
        naming_address: String::new(),
        instance_id,
        version: svc_version,
        weight: 100,
        protocol,
        healthy: true,
        created: String::new(),
        metadata: service
            .meta
            .into_iter()
            .map(|(k, v)| (k, serde_json::Value::String(v)))
            .collect(),
        tags: HashMap::new(),
    })
}

fn tag_value(tags: &[String], prefix: &str) -> Option<String> {
    tags.iter()
        .find(|t| t.starts_with(prefix))
        .map(|t| t[prefix.len()..].to_string())
}

#[derive(Debug, Deserialize)]
struct HealthEntry {
    #[serde(rename = "Node")]
    node: NodeEntry,
    #[serde(rename = "Service")]
    service: ServiceEntry,
}

#[derive(Debug, Deserialize)]
struct NodeEntry {
    #[serde(rename = "Address")]
    address: String,
}

#[derive(Debug, Deserialize)]
struct ServiceEntry {
    #[serde(rename = "ID")]
    id: String,
    #[serde(rename = "Service")]
    service: String,
    #[serde(rename = "Tags")]
    tags: Vec<String>,
    #[serde(rename = "Address")]
    address: String,
    #[serde(rename = "Port")]
    port: u16,
    #[serde(rename = "Meta")]
    meta: HashMap<String, String>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_health_url() {
        let opt = Options::new(vec!["http://localhost:8500".to_string()]);
        let url = health_url(&opt, "order", "v1", Some("42"), Some("30s"));
        assert!(url.contains("/v1/health/service/order"));
        assert!(url.contains("passing=true"));
        assert!(url.contains("tag=version:v1"));
        assert!(url.contains("index=42"));
        assert!(url.contains("wait=30s"));
    }

    #[test]
    fn test_tag_value() {
        let tags = vec!["version:v1".to_string(), "protocol:GRPC".to_string()];
        assert_eq!(tag_value(&tags, "version:"), Some("v1".to_string()));
        assert_eq!(tag_value(&tags, "protocol:"), Some("GRPC".to_string()));
        assert_eq!(tag_value(&tags, "missing:"), None);
    }

    #[test]
    fn test_entry_to_service() {
        let entry = HealthEntry {
            node: NodeEntry {
                address: "10.0.0.1".to_string(),
            },
            service: ServiceEntry {
                id: "uuid-1".to_string(),
                service: "order".to_string(),
                tags: vec![
                    "version:v1".to_string(),
                    "protocol:GRPC".to_string(),
                    "instance_id:uuid-1".to_string(),
                ],
                address: "127.0.0.1".to_string(),
                port: 8080,
                meta: HashMap::new(),
            },
        };
        let svc = entry_to_service(entry, "v1").unwrap();
        assert_eq!(svc.name, "order");
        assert_eq!(svc.address, "127.0.0.1:8080");
        assert_eq!(svc.version, "v1");
        assert_eq!(svc.protocol, ProtocolType::Grpc);
    }
}
