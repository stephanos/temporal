//! Host-only native development tools.

pub mod apalache_pins;
pub mod compiler;
/// SHA-pinned installer with private staging and serialized publication.
pub mod fetch_apalache;
pub mod guest;
pub mod lock;
pub mod retired_names;
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
