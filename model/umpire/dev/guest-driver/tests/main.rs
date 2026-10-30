//! Native guest driver controls.
#![forbid(unsafe_code)]
#[path = "guest_identity.rs"]
mod guest_identity;
#[path = "guest_pipeline.rs"]
mod guest_pipeline;
#[path = "linker.rs"]
mod linker;
use std::path::{Path, PathBuf};
fn framework_root() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR")).join("../..")
}
fn driver_binary() -> &'static str {
    env!("CARGO_BIN_EXE_umpire-guest-driver")
}
fn compiler_fixture() -> &'static str {
    env!("CARGO_BIN_EXE_identity-compiler-fixture")
}
