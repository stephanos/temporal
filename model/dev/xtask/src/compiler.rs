//! Native compiler wrapper.
use std::ffi::OsString;
/// Prepare and run a compiler, preserving its outcome.
pub fn run(_: Vec<OsString>) -> crate::Result<i32> {
    Err(crate::Error::Invalid("compiler implementation pending".into()))
}
