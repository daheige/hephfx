use std::any::Any;
use std::collections::HashMap;
use std::sync::Arc;

use async_trait::async_trait;
use serde::Deserialize;
use tokio::sync::RwLock;

use crate::Context;
use crate::discovery::Discovery;
use crate::error::{HestiaError, Result};
use crate::service::{ProtocolType, Service, StrategyHandler};

use super::{Options, base_url, new_http_client, normalize_prefix};

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
        let service_list = Arc::clone(&self.service_list);
        let name_clone = name.clone();
        self.watch_with_callback(&name, &version, move |services| {
            let service_list = Arc::clone(&service_list);
            let name = name_clone.clone();
            tokio::spawn(async move {
                let mut list = service_list.write().await;
                list.insert(name, services);
            });
        })
        .await;
    }

    /// Periodically polls the service and invokes the callback on every tick.
    pub(crate) async fn watch_with_callback(
        &self,
        name: &str,
        version: &str,
        callback: impl Fn(Vec<Service>) + Send + 'static,
    ) {
        let mut interval = tokio::time::interval(self.options.watch_interval);

        // Fetch once immediately on start
        {
            let ctx = crate::Context::new();
            match self.get_services_by_name(&ctx, name, version).await {
                Ok(services) => callback(services),
                Err(e) => log::error!("consul watch error for service {}: {}", name, e),
            }
        }

        loop {
            interval.tick().await;
            let ctx = crate::Context::new();
            match self.get_services_by_name(&ctx, name, version).await {
                Ok(services) => (&callback)(services),
                Err(e) => log::error!("consul watch error for service {}: {}", name, e),
            }
        }
    }

    async fn get_services_by_name(
        &self,
        _ctx: &Context,
        name: &str,
        version: &str,
    ) -> Result<Vec<Service>> {
        let url = health_url(&self.options, name, version);
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

        let prefix = self.options.prefix.clone();
        Ok(entries_to_services(entries, &prefix))
    }
}

fn health_url(options: &Options, name: &str, version: &str) -> String {
    let mut url = format!("{}/v1/health/service/{}", base_url(options), name);
    let mut params = Vec::new();
    params.push("passing=true".to_string());
    if !options.datacenter.is_empty() {
        params.push(format!("dc={}", options.datacenter));
    }
    if !version.is_empty() {
        params.push(format!("tag=version:{}", version));
    }
    if !params.is_empty() {
        url.push('?');
        url.push_str(&params.join("&"));
    }
    url
}

fn entries_to_services(entries: Vec<HealthEntry>, prefix: &str) -> Vec<Service> {
    let entries = filter_by_prefix(entries, prefix);
    let mut services = Vec::with_capacity(entries.len());
    for entry in entries {
        if let Some(svc) = entry_to_service(entry) {
            services.push(svc);
        }
    }
    services
}

fn filter_by_prefix(entries: Vec<HealthEntry>, prefix: &str) -> Vec<HealthEntry> {
    if prefix.is_empty() {
        return entries;
    }
    let target = format!("prefix:{}", normalize_prefix(prefix));
    entries
        .into_iter()
        .filter(|entry| entry.service.tags.iter().any(|t| t == &target))
        .collect()
}

fn entry_to_service(entry: HealthEntry) -> Option<Service> {
    let service = entry.service;

    // Read fields from tags
    let prefix = tag_value(&service.tags, "prefix:").unwrap_or_default();
    let version = tag_value(&service.tags, "version:").unwrap_or_default();
    let protocol = tag_value(&service.tags, "protocol:")
        .map(ProtocolType::from)
        .unwrap_or_default();
    let instance_id =
        tag_value(&service.tags, "instance_id:").unwrap_or_else(|| service.id.clone());
    let network = tag_value(&service.tags, "network:").unwrap_or_else(|| "tcp".to_string());
    let weight = parse_weight(&service.tags);
    let created = tag_value(&service.tags, "created:").unwrap_or_default();
    let naming_address = tag_value(&service.tags, "naming_address:").unwrap_or_default();

    // Node address fallback when service address is empty
    let host = if service.address.is_empty() {
        entry.node.address
    } else {
        service.address
    };
    let address = format!("{}:{}", host, service.port);

    // Build metadata from service meta directly
    let metadata = service
        .meta
        .unwrap_or_default()
        .into_iter()
        .map(|(k, v)| (k, serde_json::Value::String(v)))
        .collect();

    // Build tags from Consul service tags
    let mut tags = HashMap::new();
    tags.insert("prefix".to_string(), prefix.clone());
    if !version.is_empty() {
        tags.insert("version".to_string(), version.clone());
    }
    let protocol_str: String = protocol.clone().into();
    if !protocol_str.is_empty() {
        tags.insert("protocol".to_string(), protocol_str);
    }
    if !instance_id.is_empty() {
        tags.insert("instance_id".to_string(), instance_id.clone());
    }
    if !network.is_empty() {
        tags.insert("network".to_string(), network.clone());
    }
    if !created.is_empty() {
        tags.insert("created".to_string(), created.clone());
    }
    if !naming_address.is_empty() {
        tags.insert("naming_address".to_string(), naming_address.clone());
    }

    Some(Service {
        network,
        name: service.service,
        address,
        naming_address,
        instance_id,
        version,
        weight,
        protocol,
        healthy: true,
        created,
        metadata,
        tags,
    })
}

fn tag_value(tags: &[String], prefix: &str) -> Option<String> {
    tags.iter()
        .find(|t| t.starts_with(prefix))
        .map(|t| t[prefix.len()..].to_string())
}

fn parse_weight(tags: &[String]) -> u32 {
    tag_value(tags, "weight:")
        .and_then(|s| s.parse().ok())
        .unwrap_or(100)
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
    #[serde(rename = "Tags", default)]
    tags: Vec<String>,
    #[serde(rename = "Address")]
    address: String,
    #[serde(rename = "Port")]
    port: u16,
    #[serde(rename = "Meta", default)]
    meta: Option<HashMap<String, String>>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_health_url() {
        let opt = Options::new(vec!["http://localhost:8500".to_string()]);
        let url = health_url(&opt, "order", "v1");
        assert!(url.contains("/v1/health/service/order"));
        assert!(url.contains("passing=true"));
        assert!(url.contains("tag=version:v1"));
        // No longer includes index/wait params since we use ticker-based polling
        assert!(!url.contains("index="));
        assert!(!url.contains("wait="));
    }

    #[test]
    fn test_tag_value() {
        let tags = vec!["version:v1".to_string(), "protocol:GRPC".to_string()];
        assert_eq!(tag_value(&tags, "version:"), Some("v1".to_string()));
        assert_eq!(tag_value(&tags, "protocol:"), Some("GRPC".to_string()));
        assert_eq!(tag_value(&tags, "missing:"), None);
    }

    #[test]
    fn test_parse_weight() {
        let tags = vec!["weight:200".to_string()];
        assert_eq!(parse_weight(&tags), 200);

        let tags = vec!["weight:invalid".to_string()];
        assert_eq!(parse_weight(&tags), 100);

        let tags: Vec<String> = vec![];
        assert_eq!(parse_weight(&tags), 100);
    }

    #[test]
    fn test_filter_by_prefix() {
        let entries = vec![
            HealthEntry {
                node: NodeEntry {
                    address: "10.0.0.1".to_string(),
                },
                service: ServiceEntry {
                    id: "uuid-1".to_string(),
                    service: "test".to_string(),
                    tags: vec![
                        "version:v1".to_string(),
                        "prefix:hestia".to_string(),
                    ],
                    address: "127.0.0.1".to_string(),
                    port: 8080,
                    meta: Some(HashMap::new()),
                },
            },
            HealthEntry {
                node: NodeEntry {
                    address: "10.0.0.2".to_string(),
                },
                service: ServiceEntry {
                    id: "uuid-2".to_string(),
                    service: "test".to_string(),
                    tags: vec![
                        "version:v1".to_string(),
                        "prefix:other".to_string(),
                    ],
                    address: "127.0.0.2".to_string(),
                    port: 8081,
                    meta: Some(HashMap::new()),
                },
            },
        ];

        let filtered = filter_by_prefix(entries, "hestia");
        assert_eq!(filtered.len(), 1);
        assert_eq!(filtered[0].service.id, "uuid-1");
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
                    "prefix:hestia".to_string(),
                    "version:v1".to_string(),
                    "protocol:GRPC".to_string(),
                    "instance_id:uuid-1".to_string(),
                    "network:tcp".to_string(),
                    "weight:200".to_string(),
                    "created:2024-01-01".to_string(),
                    "naming_address:order.local.svc".to_string(),
                ],
                address: "127.0.0.1".to_string(),
                port: 8080,
                meta: Some(HashMap::new()),
            },
        };
        let svc = entry_to_service(entry).unwrap();
        assert_eq!(svc.name, "order");
        assert_eq!(svc.address, "127.0.0.1:8080");
        assert_eq!(svc.version, "v1");
        assert_eq!(svc.protocol, ProtocolType::Grpc);
        assert_eq!(svc.network, "tcp");
        assert_eq!(svc.weight, 200);
        assert_eq!(svc.created, "2024-01-01");
        assert_eq!(svc.naming_address, "order.local.svc");
        assert_eq!(svc.instance_id, "uuid-1");
        assert_eq!(svc.tags.get("prefix"), Some(&"hestia".to_string()));
        assert_eq!(svc.tags.get("version"), Some(&"v1".to_string()));
        assert_eq!(svc.tags.get("network"), Some(&"tcp".to_string()));
        assert_eq!(svc.tags.get("created"), Some(&"2024-01-01".to_string()));
        assert_eq!(
            svc.tags.get("naming_address"),
            Some(&"order.local.svc".to_string())
        );
    }

    #[test]
    fn test_entry_to_service_defaults() {
        let entry = HealthEntry {
            node: NodeEntry {
                address: "10.0.0.1".to_string(),
            },
            service: ServiceEntry {
                id: "uuid-1".to_string(),
                service: "order".to_string(),
                tags: vec!["version:v1".to_string(), "protocol:GRPC".to_string()],
                address: "".to_string(),
                port: 8080,
                meta: Some(HashMap::new()),
            },
        };
        let svc = entry_to_service(entry).unwrap();
        // Node address fallback
        assert_eq!(svc.address, "10.0.0.1:8080");
        // Defaults
        assert_eq!(svc.network, "tcp");
        assert_eq!(svc.weight, 100);
        assert_eq!(svc.created, "");
        assert_eq!(svc.naming_address, "");
    }

    #[test]
    fn test_health_entry_deserializes_null_meta() {
        let json = r#"[{
            "Node": {"Address": "10.0.0.1"},
            "Service": {
                "ID": "uuid-1",
                "Service": "my-test",
                "Tags": ["version:v1", "protocol:GRPC", "instance_id:uuid-1", "weight:100"],
                "Address": "127.0.0.1",
                "Port": 8080,
                "Meta": null
            }
        }]"#;
        let entries: Vec<HealthEntry> = serde_json::from_str(json).expect("deserialize health entries");
        assert_eq!(entries.len(), 1);
        assert!(entries[0].service.meta.is_none());
        let svc = entry_to_service(entries.into_iter().next().unwrap()).unwrap();
        assert_eq!(svc.name, "my-test");
        assert_eq!(svc.version, "v1");
        assert!(svc.metadata.is_empty());
    }
}
