use std::time::Duration;

/// Configuration for the etcd-backed registry/discovery.
#[derive(Debug, Clone)]
pub struct Options {
    /// etcd endpoint list.
    pub endpoints: Vec<String>,
    /// Dial timeout.
    pub dial_timeout: Duration,
    /// Lease TTL in seconds.
    pub lease_ttl: i64,
    /// Key prefix used in etcd.
    pub prefix: String,
    /// Authentication username.
    pub username: String,
    /// Authentication password.
    pub password: String,
    /// Disable watch-based cache updates.
    pub disable_watch: bool,
    /// Validate service addresses during registration.
    pub validate_address: bool,
}

impl Default for Options {
    fn default() -> Self {
        Self {
            endpoints: Vec::new(),
            dial_timeout: Duration::from_secs(5),
            lease_ttl: 60,
            prefix: "/hestia/registry-etcd".to_string(),
            username: String::new(),
            password: String::new(),
            disable_watch: true,
            validate_address: false,
        }
    }
}

impl Options {
    /// Creates a new [`Options`] with the given etcd endpoints.
    pub fn new(endpoints: Vec<String>) -> Self {
        Self {
            endpoints,
            ..Default::default()
        }
    }

    /// Sets the etcd endpoint list.
    pub fn with_endpoints(mut self, endpoints: Vec<String>) -> Self {
        self.endpoints = endpoints;
        self
    }

    /// Sets the dial timeout.
    pub fn with_dial_timeout(mut self, dial_timeout: Duration) -> Self {
        self.dial_timeout = dial_timeout;
        self
    }

    /// Sets the lease TTL in seconds.
    pub fn with_lease_ttl(mut self, ttl: i64) -> Self {
        self.lease_ttl = ttl;
        self
    }

    /// Sets the etcd key prefix.
    pub fn with_prefix(mut self, prefix: impl Into<String>) -> Self {
        self.prefix = prefix.into();
        self
    }

    /// Sets the authentication username.
    pub fn with_username(mut self, username: impl Into<String>) -> Self {
        self.username = username.into();
        self
    }

    /// Sets the authentication password.
    pub fn with_password(mut self, password: impl Into<String>) -> Self {
        self.password = password.into();
        self
    }

    /// Enables watch-based real-time cache updates.
    pub fn with_enable_watched(mut self) -> Self {
        self.disable_watch = false;
        self
    }

    /// Enables address validation during registration.
    pub fn with_validate_address(mut self, validate: bool) -> Self {
        self.validate_address = validate;
        self
    }
}
