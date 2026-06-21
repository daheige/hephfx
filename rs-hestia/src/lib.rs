//! `rs-hestia` — Rust port of the `hestia` service registry/discovery module.
//!
//! The crate mirrors the public shape of the Go `github.com/daheige/hephfx/hestia`
//! module: a [`Service`] entity, [`Registry`] / [`Discovery`] traits, built-in
//! selection strategies, and an etcd-backed implementation with a tonic gRPC
//! resolver.

pub mod discovery;
pub mod error;
pub mod netaddr;
pub mod registry;
pub mod service;

// consul and etcd registry module
pub mod consul;
pub mod etcd;

pub use discovery::Discovery;
pub use error::{HestiaError, Result};
pub use netaddr::{NetAddr, local_addr, resolve};
pub use registry::Registry;
pub use service::{ProtocolType, Service, StrategyHandler, random_handler, round_robin_handler};

/// Context alias used by [`Registry`] and [`Discovery`] methods.
///
/// It is implemented as a cancellation token so callers can cancel long-running
/// operations and so that the interface stays close to Go's `context.Context`.
pub type Context = tokio_util::sync::CancellationToken;
