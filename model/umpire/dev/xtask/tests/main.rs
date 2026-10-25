//! Native CLI controls.
#![forbid(unsafe_code)]
#[path = "apalache_install.rs"]
mod apalache_install;
#[path = "retired_names.rs"]
mod retired_names;
pub use xtask::{Error, Result, apalache_pins, support};
