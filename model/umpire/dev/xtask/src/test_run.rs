//! Framework test runner leaf, including Cargo's workspace doctests.
use std::{os::unix::process::ExitStatusExt, process::Command};

/// Run workspace tests with the caller's admitted gate environment and working directory.
pub fn run() -> crate::Result<i32> {
    let status = Command::new("mise")
        .args(["exec", "--", "cargo", "test", "--workspace", "--locked", "--quiet"])
        .status()?;
    Ok(status.code().unwrap_or_else(|| 128 + status.signal().unwrap_or(0)))
}
