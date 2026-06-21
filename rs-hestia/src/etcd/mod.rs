pub mod discovery;
pub mod options;
pub mod registry;
pub mod resolver;

pub use discovery::new_discovery;
pub use options::Options;
pub use registry::new_registry;
pub use resolver::{
    EtcdResolverBuilder, build_channel, new_resolver_builder, register_etcd_resolver,
};

use crate::error::{HestiaError, Result};

pub(crate) async fn new_etcd_client(opt: &Options) -> Result<etcd_client::Client> {
    let mut connect_opts = etcd_client::ConnectOptions::new().with_timeout(opt.dial_timeout);
    if !opt.username.is_empty() {
        connect_opts = connect_opts.with_user(&opt.username, &opt.password);
    }

    etcd_client::Client::connect(opt.endpoints.clone(), Some(connect_opts))
        .await
        .map_err(HestiaError::Etcd)
}

pub(crate) fn normalize_prefix(prefix: &str) -> String {
    let trimmed = prefix.trim_start_matches('/').trim_end_matches('/');
    format!("/{}", trimmed)
}
