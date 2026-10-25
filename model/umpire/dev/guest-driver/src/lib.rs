//! Host-only native development tools.

#![forbid(unsafe_code)]

pub mod compiler;
pub mod context;
pub mod guest;
pub mod lock;
pub mod support;

/// A native tool failure. Compiler exit statuses are returned separately.
#[derive(Debug, thiserror::Error)]
pub enum Error {
    /// An operating-system operation failed.
    #[error(transparent)]
    Io(#[from] std::io::Error),
    /// Native JSON decoding failed.
    #[error(transparent)]
    Json(#[from] serde_json::Error),
    /// Native TOML decoding failed.
    #[error(transparent)]
    Toml(#[from] toml::de::Error),
    /// A driver contract was violated.
    #[error("{0}")]
    Invalid(String),
}

/// Result returned by native tool operations.
pub type Result<T> = std::result::Result<T, Error>;
