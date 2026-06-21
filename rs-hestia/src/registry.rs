use async_trait::async_trait;

use crate::Context;
use crate::error::Result;
use crate::service::Service;

/// Extension interface for service registry implementations.
#[async_trait]
pub trait Registry: Send + Sync {
    /// Register a service instance.
    async fn register(&self, ctx: &Context, service: &mut Service) -> Result<()>;

    /// Deregister a service instance when the application exits.
    async fn deregister(&self, ctx: &Context, service: &mut Service) -> Result<()>;

    /// Returns the string identifier of the registry implementation.
    fn string(&self) -> &str;
}
