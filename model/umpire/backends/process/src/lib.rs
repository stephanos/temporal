//! Shared bounded subprocess and workspace scratch support for lifted backends.
mod fingerprint;
mod jar;
mod support;
mod syscall;
/// Isolated mise provisioning and exact release path resolution.
pub mod tool_provider;
pub mod work;
pub use fingerprint::{bytes_sha256, structural_sha256};
pub use jar::PinnedJar;
pub use support::*;
