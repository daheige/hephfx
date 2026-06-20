use std::any::Any;

use async_trait::async_trait;

use crate::Context;
use crate::error::Result;
use crate::service::{Service, StrategyHandler};

/// Service discovery interface.
#[async_trait]
pub trait Discovery: Send + Sync + Any {
    /// Returns a list of healthy service instances.
    ///
    /// `version` may be empty to ignore version filtering.
    async fn get_services(&self, ctx: &Context, name: &str, version: &str) -> Result<Vec<Service>>;

    /// Returns a single available instance using the supplied strategy.
    ///
    /// If no strategy is supplied, round-robin is used.
    async fn get(
        &self,
        ctx: &Context,
        name: &str,
        version: &str,
        strategy: Option<StrategyHandler>,
    ) -> Result<Service>;

    /// Returns the name of the discovery implementation.
    fn name(&self) -> &str;

    /// Returns the type as `Any` so that implementations can be downcast.
    ///
    /// This mirrors Go's type assertion and is used by the gRPC resolver to
    /// reuse etcd-specific watch capabilities.
    fn as_any(&self) -> &dyn Any;
}
