use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;

use async_trait::async_trait;
use serde::Serialize;
use tokio::sync::Mutex;
use tokio::task::JoinHandle;
use uuid::Uuid;

use crate::Context;
use crate::error::{HestiaError, Result};
use crate::netaddr::resolve;
use crate::registry::Registry;
use crate::service::{ProtocolType, Service};

use super::{Options, base_url, new_http_client, normalize_prefix};

/// Consul-backed [`Registry`] implementation.
pub struct ConsulRegistry {
    client: reqwest::Client,
    options: Options,
    keepalive_handle: Mutex<Option<JoinHandle<()>>>,
}

/// Creates a new Consul-backed registry.
pub async fn new_registry(options: Options) -> Result<Arc<dyn Registry>> {
    let client = new_http_client(&options)?;
    Ok(Arc::new(ConsulRegistry {
        client,
        options,
        keepalive_handle: Mutex::new(None),
    }))
}

#[async_trait]
impl Registry for ConsulRegistry {
    async fn register(&self, _ctx: &Context, service: &mut Service) -> Result<()> {
        if service.instance_id.is_empty() {
            service.instance_id = Uuid::new_v4().to_string();
        }

        if self.options.validate_address {
            service.address = resolve(&service.address)?;
        }

        if service.weight == 0 {
            service.weight = 100;
        }
        service.healthy = true;

        let check_id = format!("service:{}", service.instance_id);
        let (host, port) = parse_host_port(&service.address)?;
        let payload = RegisterPayload {
            id: service.instance_id.clone(),
            name: service.name.clone(),
            tags: build_tags(service, &self.options.prefix),
            address: host,
            port,
            meta: flatten_metadata(&service.metadata),
            check: Some(CheckPayload {
                check_id: check_id.clone(),
                name: format!("{} TTL check", service.name),
                ttl: format!("{}s", self.options.health_check_ttl),
                deregister_critical_service_after: "1m".to_string(),
            }),
        };

        let url = format!("{}/v1/agent/service/register", base_url(&self.options));
        let mut req = self.client.put(&url).json(&payload);
        if !self.options.token.is_empty() {
            req = req.header("X-Consul-Token", &self.options.token);
        }
        tokio::time::timeout(self.options.dial_timeout, req.send())
            .await
            .map_err(|e| HestiaError::Other(format!("register timeout: {}", e)))??
            .error_for_status()
            .map_err(HestiaError::Consul)?;

        log::info!(
            "register consul service:{} version:{} instance_id:{} success",
            service.name,
            service.version,
            service.instance_id
        );

        {
            let mut handle = self.keepalive_handle.lock().await;
            if let Some(h) = handle.take() {
                h.abort();
            }

            let client = self.client.clone();
            let token = self.options.token.clone();
            let base = base_url(&self.options);
            let ttl = self.options.health_check_ttl;
            *handle = Some(tokio::spawn(async move {
                if let Err(e) = keepalive(client, &base, &token, &check_id, ttl).await {
                    log::error!("consul keepalive failed: {}", e);
                }
            }));
        }

        Ok(())
    }

    async fn deregister(&self, _ctx: &Context, service: &mut Service) -> Result<()> {
        if service.name.is_empty() {
            return Err(HestiaError::MissingServiceName);
        }

        let _ = parse_host_port(&service.address)?;

        let url = format!(
            "{}/v1/agent/service/deregister/{}",
            base_url(&self.options),
            service.instance_id
        );
        let mut req = self.client.put(&url);
        if !self.options.token.is_empty() {
            req = req.header("X-Consul-Token", &self.options.token);
        }
        tokio::time::timeout(self.options.dial_timeout, req.send())
            .await
            .map_err(|e| HestiaError::Other(format!("deregister timeout: {}", e)))??
            .error_for_status()
            .map_err(HestiaError::Consul)?;

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
        "consul"
    }
}

#[derive(Debug, Serialize)]
struct RegisterPayload {
    #[serde(rename = "ID")]
    id: String,
    #[serde(rename = "Name")]
    name: String,
    #[serde(rename = "Tags")]
    tags: Vec<String>,
    #[serde(rename = "Address")]
    address: String,
    #[serde(rename = "Port")]
    port: u16,
    #[serde(rename = "Meta")]
    meta: HashMap<String, String>,
    #[serde(rename = "Check")]
    check: Option<CheckPayload>,
}

#[derive(Debug, Serialize)]
struct CheckPayload {
    #[serde(rename = "CheckID")]
    check_id: String,
    #[serde(rename = "Name")]
    name: String,
    #[serde(rename = "TTL")]
    ttl: String,
    #[serde(rename = "DeregisterCriticalServiceAfter")]
    deregister_critical_service_after: String,
}

async fn keepalive(
    client: reqwest::Client,
    base: &str,
    token: &str,
    check_id: &str,
    ttl: u64,
) -> Result<()> {
    let url = format!("{}/v1/agent/check/pass/{}", base, check_id);
    let interval_secs = (ttl / 2).max(1);
    let mut interval = tokio::time::interval(Duration::from_secs(interval_secs));

    loop {
        interval.tick().await;
        let mut req = client.put(&url);
        if !token.is_empty() {
            req = req.header("X-Consul-Token", token);
        }
        match req.send().await {
            Ok(resp) => {
                if let Err(e) = resp.error_for_status() {
                    log::error!("consul keepalive error: {}", e);
                }
            }
            Err(e) => log::error!("consul keepalive request error: {}", e),
        }
    }
}

fn build_tags(service: &Service, prefix: &str) -> Vec<String> {
    let mut tags = Vec::with_capacity(4);
    if !prefix.is_empty() {
        tags.push(format!("prefix:{}", normalize_prefix(prefix)));
    }
    if !service.version.is_empty() {
        tags.push(format!("version:{}", service.version));
    }
    let protocol: ProtocolType = service.protocol.clone();
    tags.push(format!("protocol:{}", String::from(protocol)));
    tags.push(format!("instance_id:{}", service.instance_id));
    tags
}

fn flatten_metadata(metadata: &HashMap<String, serde_json::Value>) -> HashMap<String, String> {
    metadata
        .iter()
        .map(|(k, v)| (k.clone(), v.to_string()))
        .collect()
}

fn parse_host_port(address: &str) -> Result<(String, u16)> {
    let pos = address.rfind(':').ok_or_else(|| {
        HestiaError::InvalidAddress(format!("missing port in address: {}", address))
    })?;
    let (host, port_str) = address.split_at(pos);
    let port_str = &port_str[1..];
    if host.is_empty() {
        return Err(HestiaError::InvalidAddress(format!(
            "missing host in address: {}",
            address
        )));
    }
    let port: u16 = port_str
        .parse()
        .map_err(|_| HestiaError::InvalidAddress(format!("invalid port: {}", port_str)))?;
    Ok((host.to_string(), port))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_host_port() {
        assert_eq!(
            parse_host_port("127.0.0.1:8080").unwrap(),
            ("127.0.0.1".to_string(), 8080)
        );
    }

    #[test]
    fn test_build_tags() {
        let service = Service {
            name: "order".to_string(),
            version: "v1".to_string(),
            instance_id: "uuid-1".to_string(),
            protocol: ProtocolType::Grpc,
            ..Default::default()
        };
        let tags = build_tags(&service, "hestia");
        assert!(tags.iter().any(|t| t == "version:v1"));
        assert!(tags.iter().any(|t| t == "protocol:GRPC"));
    }
}
