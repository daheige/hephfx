use std::sync::Arc;
use std::time::Duration;

use async_trait::async_trait;
use etcd_client::{Client, PutOptions};
use tokio::sync::Mutex;
use tokio::task::JoinHandle;
use uuid::Uuid;

use crate::Context;
use crate::error::{HestiaError, Result};
use crate::netaddr::resolve;
use crate::registry::Registry;
use crate::service::Service;

use super::{Options, new_etcd_client, normalize_prefix};

/// etcd-backed [`Registry`] implementation.
pub struct EtcdRegistry {
    client: Client,
    lease_ttl: i64,
    prefix: String,
    validate_address: bool,
    keepalive_handle: Mutex<Option<JoinHandle<()>>>,
}

/// Creates a new etcd-backed registry.
pub async fn new_registry(options: Options) -> Result<Arc<dyn Registry>> {
    let client = new_etcd_client(&options).await?;
    let prefix = normalize_prefix(&options.prefix);

    Ok(Arc::new(EtcdRegistry {
        client,
        lease_ttl: options.lease_ttl,
        prefix,
        validate_address: options.validate_address,
        keepalive_handle: Mutex::new(None),
    }))
}

#[async_trait]
impl Registry for EtcdRegistry {
    async fn register(&self, _ctx: &Context, service: &mut Service) -> Result<()> {
        if service.instance_id.is_empty() {
            service.instance_id = Uuid::new_v4().to_string();
        }

        if self.validate_address {
            service.address = resolve(&service.address)?;
        }

        if service.weight == 0 {
            service.weight = 100;
        }
        service.healthy = true;

        let lease_id = self.grant_lease().await?;
        self.register_service(service, lease_id).await?;

        // Cancel any previous keepalive before starting a new one.
        {
            let mut handle = self.keepalive_handle.lock().await;
            if let Some(h) = handle.take() {
                h.abort();
            }

            let client = self.client.clone();
            let ttl = self.lease_ttl;
            *handle = Some(tokio::spawn(async move {
                if let Err(e) = keepalive(client, lease_id, ttl).await {
                    log::error!("etcd keepalive failed: {}", e);
                }
            }));
        }

        Ok(())
    }

    async fn deregister(&self, _ctx: &Context, service: &mut Service) -> Result<()> {
        if service.name.is_empty() {
            return Err(HestiaError::MissingServiceName);
        }

        // Go always resolves here regardless of validate_address.
        let _ = resolve(&service.address)?;

        let key = register_key(&self.prefix, service);
        let mut kv = self.client.kv_client();
        tokio::time::timeout(Duration::from_secs(15), kv.delete(key.as_str(), None))
            .await
            .map_err(|e| HestiaError::Other(format!("deregister timeout: {}", e)))??;

        {
            let mut handle = self.keepalive_handle.lock().await;
            if let Some(h) = handle.take() {
                h.abort();
            }
        }

        service.healthy = false;
        Ok(())
    }

    fn string(&self) -> &str {
        "etcd"
    }
}

impl EtcdRegistry {
    async fn grant_lease(&self) -> Result<i64> {
        let mut lease = self.client.lease_client();
        let resp = lease.grant(self.lease_ttl, None).await?;
        Ok(resp.id())
    }

    async fn register_service(&self, service: &Service, lease_id: i64) -> Result<()> {
        let value = serde_json::to_string(service)?;
        let key = register_key(&self.prefix, service);
        // println!("register key:{}",key);
        let mut kv = self.client.kv_client();
        tokio::time::timeout(
            Duration::from_secs(15),
            kv.put(
                key.as_str(),
                value,
                Some(PutOptions::new().with_lease(lease_id)),
            ),
        )
        .await
        .map_err(|e| HestiaError::Other(format!("register timeout: {}", e)))??;

        log::info!(
            "register prefix:{} service:{} version:{} instance_id:{} lease_id:{} success",
            self.prefix,
            service.name,
            service.version,
            service.instance_id,
            lease_id
        );
        Ok(())
    }
}

async fn keepalive(client: Client, lease_id: i64, ttl: i64) -> Result<()> {
    let (mut keeper, mut stream) = client.lease_client().keep_alive(lease_id).await?;
    let interval_secs = (ttl / 3).max(1);
    let mut interval = tokio::time::interval(Duration::from_secs(interval_secs as u64));

    loop {
        tokio::select! {
            _ = interval.tick() => {
                keeper.keep_alive().await?;
            }
            resp = stream.message() => {
                match resp {
                    Ok(Some(_)) => {}
                    Ok(None) => break,
                    Err(e) => return Err(HestiaError::Etcd(e)),
                }
            }
        }
    }

    Ok(())
}

fn register_key(prefix: &str, service: &Service) -> String {
    if service.version.is_empty() {
        format!("{}/{}/{}", prefix, service.name, service.instance_id)
    } else {
        format!(
            "{}/{}/{}/{}",
            prefix, service.name, service.version, service.instance_id
        )
    }
}
