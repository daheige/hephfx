use std::num::ParseIntError;

use thiserror::Error;

/// Errors returned by `rs-hestia`.
#[derive(Error, Debug)]
pub enum HestiaError {
    /// The requested service has no available instances.
    #[error("services not found")]
    ServicesNotFound,

    /// The service name is missing where required.
    #[error("missing service name")]
    MissingServiceName,

    /// The supplied address is not a valid `host:port`.
    #[error("invalid address: {0}")]
    InvalidAddress(String),

    /// An etcd client error.
    #[error("etcd error: {0}")]
    Etcd(#[from] etcd_client::Error),

    /// A Consul HTTP client error.
    #[error("consul error: {0}")]
    Consul(#[from] reqwest::Error),

    /// A JSON serialization/deserialization error.
    #[error("json error: {0}")]
    Json(#[from] serde_json::Error),

    /// A tonic transport error.
    #[error("tonic error: {0}")]
    Tonic(#[from] tonic::transport::Error),

    /// A URL parse error from the gRPC resolver target.
    #[error("invalid target url: {0}")]
    InvalidTarget(String),

    /// A generic I/O error.
    #[error("io error: {0}")]
    Io(#[from] std::io::Error),

    /// A catch-all error for unexpected situations.
    #[error("{0}")]
    Other(String),
}

/// Convenience result type used throughout the crate.
pub type Result<T> = std::result::Result<T, HestiaError>;

impl From<ParseIntError> for HestiaError {
    fn from(err: ParseIntError) -> Self {
        HestiaError::InvalidAddress(err.to_string())
    }
}
