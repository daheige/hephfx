pub mod discovery;
pub mod options;
pub mod registry;
pub mod resolver;

pub use discovery::new_discovery;
pub use options::Options;
pub use registry::new_registry;
pub use resolver::{
    ConsulResolverBuilder, build_channel, new_resolver_builder, register_consul_resolver,
};

use crate::error::{HestiaError, Result};

pub(crate) fn new_http_client(_opt: &Options) -> Result<reqwest::Client> {
    reqwest::Client::builder()
        .build()
        .map_err(HestiaError::Consul)
}

pub(crate) fn base_url(opt: &Options) -> String {
    opt.endpoints
        .first()
        .cloned()
        .unwrap_or_else(|| "http://127.0.0.1:8500".to_string())
}

pub(crate) fn normalize_prefix(prefix: &str) -> String {
    prefix
        .trim_start_matches('/')
        .trim_end_matches('/')
        .to_string()
}
