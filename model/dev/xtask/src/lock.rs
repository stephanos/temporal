//! Portable non-development lock closures.
use std::path::{Path, PathBuf};
/// Publish a content-addressed lock map from metadata JSON and native TOML.
pub fn publish(_: &[u8], _: &Path, _: &Path) -> crate::Result<PathBuf> {
    Err(crate::Error::Invalid("lock implementation pending".into()))
}
