//! Shared bounded subprocess and workspace scratch support for lifted backends.
mod jar;
mod support;
mod syscall;
pub use jar::PinnedJar;
pub use support::*;
