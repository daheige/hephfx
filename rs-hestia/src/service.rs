use std::collections::HashMap;
use std::sync::Arc;
use std::sync::atomic::{AtomicUsize, Ordering};

use rand::RngExt as _;
use serde::{Deserialize, Serialize};

/// Protocol type used by a downstream service.
///
/// Supports the well-known variants `Unspecified`, `Grpc`, and `Http`, plus
/// arbitrary protocol strings via `Other` (e.g. `"TCP"`, `"WEBSOCKET"`).
/// Empty strings deserialize as `Unspecified`; `GRPC`/`HTTP` are recognized
/// case-insensitively and normalized to their well-known variants.
#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
#[serde(from = "String", into = "String")]
pub enum ProtocolType {
    /// No protocol specified. Treated as gRPC-compatible by the resolver.
    /// 表示没有指定协议
    #[default]
    Unspecified,
    /// gRPC service.
    Grpc,
    /// HTTP/REST service.
    Http,
    /// Any other protocol string.
    Other(String),
}

impl From<String> for ProtocolType {
    fn from(s: String) -> Self {
        match s.to_uppercase().as_str() {
            "" => Self::Unspecified,
            "GRPC" => Self::Grpc,
            "HTTP" => Self::Http,
            _ => Self::Other(s),
        }
    }
}

impl From<ProtocolType> for String {
    fn from(p: ProtocolType) -> Self {
        match p {
            ProtocolType::Unspecified => String::new(),
            ProtocolType::Grpc => "GRPC".to_string(),
            ProtocolType::Http => "HTTP".to_string(),
            ProtocolType::Other(s) => s,
        }
    }
}

/// Service instance metadata.
///
/// Field names and JSON tags match the Go `hestia.Service` struct so that
/// Go-registered services can be read by Rust clients and vice-versa.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct Service {
    /// Network name, e.g. `"tcp"`.
    pub network: String,

    /// Service name.
    pub name: String,

    /// Service address, usually `host:port`.
    pub address: String,

    /// Naming address, e.g. a Kubernetes headless service DNS name.
    #[serde(rename = "naming_address")]
    pub naming_address: String,

    /// Unique instance identifier.
    #[serde(rename = "instance_id")]
    pub instance_id: String,

    /// Service version.
    pub version: String,

    /// Weight used by load-balancing strategies (default 100).
    pub weight: u32,

    /// Protocol type.
    pub protocol: ProtocolType,

    /// Health flag.
    pub healthy: bool,

    /// Creation timestamp.
    pub created: String,

    /// Arbitrary metadata.
    pub metadata: HashMap<String, serde_json::Value>,

    /// Tags.
    pub tags: HashMap<String, String>,
}

/// Strategy used to select one service instance from a list.
pub type StrategyHandler = Arc<dyn Fn(&[Service]) -> Option<Service> + Send + Sync>;

static ROUND_ROBIN_COUNTER: AtomicUsize = AtomicUsize::new(0);

/// Returns a strategy that selects instances in round-robin order.
///
/// Uses a global atomic counter to match the Go implementation's semantics.
pub fn round_robin_handler() -> StrategyHandler {
    Arc::new(|services| {
        if services.is_empty() {
            return None;
        }
        let idx = ROUND_ROBIN_COUNTER.fetch_add(1, Ordering::Relaxed);
        Some(services[idx % services.len()].clone())
    })
}

/// Returns a strategy that selects a random instance.
pub fn random_handler() -> StrategyHandler {
    Arc::new(|services| {
        if services.is_empty() {
            return None;
        }
        let mut rng = rand::rng();
        Some(services[rng.random_range(0..services.len())].clone())
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_round_robin_handler() {
        let services = vec![
            Service {
                address: "a:1".to_string(),
                ..Default::default()
            },
            Service {
                address: "a:2".to_string(),
                ..Default::default()
            },
            Service {
                address: "a:3".to_string(),
                ..Default::default()
            },
        ];

        let handler = round_robin_handler();
        let mut seen = HashMap::new();
        for _ in 0..services.len() * 3 {
            let svc = handler(&services).expect("got nil service");
            *seen.entry(svc.address.clone()).or_insert(0) += 1;
        }

        for svc in &services {
            assert_eq!(
                seen.get(&svc.address),
                Some(&3),
                "expected 3 selections for {}",
                svc.address
            );
        }
    }

    #[test]
    fn test_round_robin_handler_empty() {
        let handler = round_robin_handler();
        assert!(handler(&[]).is_none());
    }

    #[test]
    fn test_random_handler() {
        let services = vec![
            Service {
                address: "a:1".to_string(),
                ..Default::default()
            },
            Service {
                address: "a:2".to_string(),
                ..Default::default()
            },
        ];

        let handler = random_handler();
        let svc = handler(&services).expect("got nil service");
        assert!(services.iter().any(|s| s.address == svc.address));
    }

    #[test]
    fn test_random_handler_empty() {
        let handler = random_handler();
        assert!(handler(&[]).is_none());
    }

    #[test]
    fn test_service_json_compatibility() {
        let svc = Service {
            network: "tcp".to_string(),
            name: "my-service".to_string(),
            address: "127.0.0.1:8080".to_string(),
            instance_id: "uuid-1".to_string(),
            version: "v1".to_string(),
            weight: 100,
            protocol: ProtocolType::Grpc,
            healthy: true,
            created: "2024-01-01 00:00:00".to_string(),
            ..Default::default()
        };

        let json = serde_json::to_string(&svc).unwrap();
        assert!(json.contains("\"instance_id\":\"uuid-1\""));
        assert!(json.contains("\"naming_address\""));
        assert!(json.contains("\"GRPC\""));

        let decoded: Service = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.instance_id, "uuid-1");
        assert_eq!(decoded.protocol, ProtocolType::Grpc);
    }

    #[test]
    fn test_protocol_type_arbitrary_string() {
        let decoded: ProtocolType = serde_json::from_str("\"WEBSOCKET\"").unwrap();
        assert_eq!(decoded, ProtocolType::Other("WEBSOCKET".to_string()));

        let json = serde_json::to_string(&ProtocolType::Other("MQTT".to_string())).unwrap();
        assert_eq!(json, "\"MQTT\"");
    }

    #[test]
    fn test_protocol_type_case_insensitive_known_variants() {
        assert_eq!(
            serde_json::from_str::<ProtocolType>("\"grpc\"").unwrap(),
            ProtocolType::Grpc
        );
        assert_eq!(
            serde_json::from_str::<ProtocolType>("\"Http\"").unwrap(),
            ProtocolType::Http
        );
    }
}
