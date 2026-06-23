use std::collections::HashSet;
use std::sync::Arc;
use std::time::Duration;

use tokio::sync::mpsc::Sender;
use tonic::transport::channel::Change;
use tonic::transport::{Channel, Endpoint};

use crate::discovery::Discovery;
use crate::error::{HestiaError, Result};
use crate::service::{ProtocolType, Service};

use super::discovery::ConsulDiscovery;

/// gRPC resolver builder backed by a Consul [`Discovery`] implementation.
#[derive(Clone)]
pub struct ConsulResolverBuilder {
    discovery: Arc<dyn Discovery>,
    scheme: String,
}

impl ConsulResolverBuilder {
    /// Creates a new builder for the `consul` scheme.
    pub fn new(discovery: Arc<dyn Discovery>) -> Self {
        Self {
            discovery,
            scheme: "consul".to_string(),
        }
    }

    /// Returns the resolver scheme.
    pub fn scheme(&self) -> &str {
        &self.scheme
    }

    /// Builds a load-balanced tonic [`Channel`] for the given target.
    pub async fn build(&self, target: &str) -> Result<Channel> {
        let (name, version) = parse_target(target)?;
        let (channel, mut tx) = Channel::balance_channel::<String>(64);

        let ctx = crate::Context::new();
        let services = match self.discovery.get_services(&ctx, &name, &version).await {
            Ok(s) => s,
            Err(HestiaError::ServicesNotFound) => Vec::new(),
            Err(e) => return Err(e),
        };

        let current = Arc::new(tokio::sync::Mutex::new(HashSet::new()));
        {
            let mut guard = current.lock().await;
            apply_services(&mut tx, &services, &mut guard).await?;
        }

        if let Some(cd) = self.discovery.as_any().downcast_ref::<ConsulDiscovery>() {
            let cd = cd.clone();
            let current = Arc::clone(&current);
            let tx = tx.clone();
            let name = name.clone();
            let version = version.clone();
            tokio::spawn(async move {
                cd.watch_with_callback(&name, &version, move |services| {
                    let current = Arc::clone(&current);
                    let mut tx = tx.clone();
                    tokio::spawn(async move {
                        let mut guard = current.lock().await;
                        if let Err(e) = apply_services(&mut tx, &services, &mut guard).await {
                            log::error!("failed to update consul state err: {}", e);
                        }
                    });
                })
                .await;
            });
        } else {
            let discovery = Arc::clone(&self.discovery);
            let current = Arc::clone(&current);
            let mut tx = tx.clone();
            tokio::spawn(async move {
                let mut interval = tokio::time::interval(Duration::from_secs(10));
                loop {
                    interval.tick().await;
                    let ctx = crate::Context::new();
                    match discovery.get_services(&ctx, &name, &version).await {
                        Ok(services) => {
                            let mut guard = current.lock().await;
                            if let Err(e) = apply_services(&mut tx, &services, &mut guard).await {
                                log::error!("failed to update consul state err: {}", e);
                            }
                        }
                        Err(HestiaError::ServicesNotFound) => {
                            let mut guard = current.lock().await;
                            for key in guard.drain() {
                                let _ = tx.send(Change::Remove(key)).await;
                            }
                        }
                        Err(e) => {
                            log::error!("poll discovery error: {}", e);
                        }
                    }
                }
            });
        }

        Ok(channel)
    }
}

/// Convenience function that builds a channel for the given target and discovery.
pub async fn build_channel(target: &str, discovery: Arc<dyn Discovery>) -> Result<Channel> {
    ConsulResolverBuilder::new(discovery).build(target).await
}

/// Creates a new Consul gRPC resolver builder.
pub fn new_resolver_builder(discovery: Arc<dyn Discovery>) -> ConsulResolverBuilder {
    ConsulResolverBuilder::new(discovery)
}

/// Registers the Consul gRPC resolver.
pub fn register_consul_resolver(discovery: Arc<dyn Discovery>) -> ConsulResolverBuilder {
    ConsulResolverBuilder::new(discovery)
}

fn parse_target(target: &str) -> Result<(String, String)> {
    const PREFIX: &str = "consul:///";
    if !target.starts_with(PREFIX) {
        return Err(HestiaError::InvalidTarget(format!(
            "expected target to start with {}, got: {}",
            PREFIX, target
        )));
    }

    let path = &target[PREFIX.len()..];
    if path.is_empty() {
        return Err(HestiaError::InvalidTarget(format!(
            "consul resolver target path is empty, got: {}",
            target
        )));
    }

    let mut parts = path.split('/').filter(|s| !s.is_empty());
    let name = parts.next().map(ToString::to_string).ok_or_else(|| {
        HestiaError::InvalidTarget(format!("empty service name in target: {}", target))
    })?;
    let version = parts.next().unwrap_or("").to_string();
    Ok((name, version))
}

async fn apply_services(
    tx: &mut Sender<Change<String, Endpoint>>,
    services: &[Service],
    current: &mut HashSet<String>,
) -> Result<()> {
    let mut new_keys = HashSet::with_capacity(services.len());
    for s in services {
        if s.protocol != ProtocolType::Unspecified && s.protocol != ProtocolType::Grpc {
            continue;
        }

        new_keys.insert(s.instance_id.clone());
        if !current.contains(&s.instance_id) {
            let endpoint = Endpoint::from_shared(format!("http://{}", s.address))?;
            tx.send(Change::Insert(s.instance_id.clone(), endpoint))
                .await
                .map_err(|e| HestiaError::Other(e.to_string()))?;
        }
    }

    let to_remove: Vec<String> = current.difference(&new_keys).cloned().collect();
    for key in to_remove {
        tx.send(Change::Remove(key.clone()))
            .await
            .map_err(|e| HestiaError::Other(e.to_string()))?;
        current.remove(&key);
    }

    for key in new_keys {
        current.insert(key);
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_target_name_and_version() {
        let (name, version) = parse_target("consul:///order_service/v1").unwrap();
        assert_eq!(name, "order_service");
        assert_eq!(version, "v1");
    }

    #[test]
    fn test_parse_target_name_only() {
        let (name, version) = parse_target("consul:///order_service").unwrap();
        assert_eq!(name, "order_service");
        assert_eq!(version, "");
    }

    #[test]
    fn test_parse_target_empty_path() {
        assert!(parse_target("consul:///").is_err());
    }

    #[test]
    fn test_parse_target_wrong_scheme() {
        assert!(parse_target("dns:///order_service/v1").is_err());
    }
}
