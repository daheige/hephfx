use std::time::Duration;

/// Configuration for the Consul-backed registry/discovery.
#[derive(Debug, Clone)]
pub struct Options {
    /// Consul HTTP endpoint list. The first endpoint is used for requests.
    pub endpoints: Vec<String>,

    /// Consul datacenter. Leave empty to use the agent's default datacenter.
    pub datacenter: String,

    /// ACL token used for authenticated requests.
    pub token: String,

    /// Resolver scheme name. Defaults to `consul`.
    pub scheme: String,

    /// TTL in seconds for the service health check. Defaults to 10.
    pub health_check_ttl: u64,

    /// Key prefix used for tag/metadata conventions. Not used as a Consul KV prefix.
    pub prefix: String,

    /// Disable long-poll watch-based cache updates.
    pub disable_watch: bool,

    /// Validate service addresses during registration.
    pub validate_address: bool,

    /// HTTP request timeout for non-blocking requests.
    pub dial_timeout: Duration,
}

impl Default for Options {
    fn default() -> Self {
        Self {
            endpoints: vec!["http://127.0.0.1:8500".to_string()],
            datacenter: String::new(),
            token: String::new(),
            scheme: "consul".to_string(),
            health_check_ttl: 10,
            prefix: "/hestia/registry-consul".to_string(),
            disable_watch: true,
            validate_address: false,
            dial_timeout: Duration::from_secs(5),
        }
    }
}

impl Options {
    /// Creates a new [`Options`] with the given Consul endpoints.
    pub fn new(endpoints: Vec<String>) -> Self {
        Self {
            endpoints,
            ..Default::default()
        }
    }

    /// Sets the Consul endpoint list.
    pub fn with_endpoints(mut self, endpoints: Vec<String>) -> Self {
        self.endpoints = endpoints;
        self
    }

    /// Sets the Consul datacenter.
    pub fn with_datacenter(mut self, datacenter: impl Into<String>) -> Self {
        self.datacenter = datacenter.into();
        self
    }

    /// Sets the ACL token.
    pub fn with_token(mut self, token: impl Into<String>) -> Self {
        self.token = token.into();
        self
    }

    /// Sets the resolver scheme name.
    pub fn with_scheme(mut self, scheme: impl Into<String>) -> Self {
        self.scheme = scheme.into();
        self
    }

    /// Sets the TTL in seconds for the service health check.
    pub fn with_health_check_ttl(mut self, ttl: u64) -> Self {
        self.health_check_ttl = ttl;
        self
    }

    /// Sets the prefix used for tag/metadata conventions.
    pub fn with_prefix(mut self, prefix: impl Into<String>) -> Self {
        self.prefix = prefix.into();
        self
    }

    /// Enables long-poll watch-based real-time cache updates.
    pub fn with_enable_watch(mut self) -> Self {
        self.disable_watch = false;
        self
    }

    /// Enables address validation during registration.
    pub fn with_validate_address(mut self, validate: bool) -> Self {
        self.validate_address = validate;
        self
    }

    /// Sets the HTTP request timeout for non-blocking requests.
    pub fn with_dial_timeout(mut self, dial_timeout: Duration) -> Self {
        self.dial_timeout = dial_timeout;
        self
    }
}
