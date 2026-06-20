use std::fmt;

use if_addrs::IfAddr;

use crate::error::{HestiaError, Result};

/// A network address value.
///
/// Mirrors Go's `net.Addr` implementation used by the original module.
#[derive(Debug, Clone)]
pub struct NetAddr {
    network: String,
    address: String,
}

impl NetAddr {
    /// Creates a new `NetAddr`.
    pub fn new(network: impl Into<String>, address: impl Into<String>) -> Self {
        Self {
            network: network.into(),
            address: address.into(),
        }
    }

    /// Returns the network name.
    pub fn network(&self) -> &str {
        &self.network
    }

    /// Returns the address string.
    pub fn address(&self) -> &str {
        &self.address
    }
}

impl fmt::Display for NetAddr {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.address)
    }
}

/// Resolves an address to a canonical `host:port` string.
///
/// If the host portion is empty or `"::"`, the first non-loopback IPv4 address
/// of the local machine is used.
pub fn resolve(address: &str) -> Result<String> {
    let pos = address.rfind(':').ok_or_else(|| {
        HestiaError::InvalidAddress(format!("missing port in address: {}", address))
    })?;

    let (host, port_str) = address.split_at(pos);
    let port_str = &port_str[1..];

    let host = if host.is_empty() || host == "::" {
        local_ipv4_host()?
    } else {
        host.to_string()
    };

    let port: u16 = port_str
        .parse()
        .map_err(|_| HestiaError::InvalidAddress(format!("invalid port: {}", port_str)))?;

    Ok(format!("{}:{}", host, port))
}

/// Returns the local non-loopback IPv4 address.
pub fn local_addr() -> Result<String> {
    local_ipv4_host()
}

fn local_ipv4_host() -> Result<String> {
    for iface in if_addrs::get_if_addrs()? {
        if iface.is_loopback() {
            continue;
        }
        if let IfAddr::V4(v4) = iface.addr {
            let ip = v4.ip;
            if !ip.is_loopback() {
                return Ok(ip.to_string());
            }
        }
    }

    Err(HestiaError::Other("not found ipv4 address".to_string()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_net_addr() {
        let n = NetAddr::new("tcp", ":8090");
        assert_eq!(n.network(), "tcp");
        assert_eq!(n.address(), ":8090");
        assert_eq!(n.to_string(), ":8090");
    }

    #[test]
    fn test_resolve_localhost() {
        // localhost is kept as-is; Resolve only fills empty or "::" hosts.
        let addr = resolve("localhost:8090").unwrap();
        assert_eq!(addr, "localhost:8090");
    }

    #[test]
    fn test_resolve_dns() {
        let addr = resolve("my-service.default.svc.cluster.local:8080").unwrap();
        assert_eq!(addr, "my-service.default.svc.cluster.local:8080");
    }

    #[test]
    fn test_resolve_empty_host() {
        let addr = resolve(":8090").unwrap();
        assert!(addr.ends_with(":8090"));
        assert_ne!(addr, ":8090");
    }

    #[test]
    fn test_resolve_ipv6_wildcard() {
        let addr = resolve(":::8090").unwrap();
        assert!(addr.ends_with(":8090"));
        assert_ne!(addr, ":::8090");
    }

    #[test]
    fn test_resolve_invalid_address() {
        assert!(resolve("8090").is_err());
        assert!(resolve("host:not-a-port").is_err());
    }

    #[test]
    fn test_local_addr() {
        let ip = local_addr().unwrap();
        assert!(!ip.is_empty());
        assert!(!ip.starts_with("127."));
    }
}
