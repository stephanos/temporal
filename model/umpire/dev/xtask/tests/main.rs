//! Native CLI controls.
#![forbid(unsafe_code)]
#[path = "apalache_install.rs"]
mod apalache_install;
#[path = "gate_wiring.rs"]
mod gate_wiring;
#[path = "retired_names.rs"]
mod retired_names;
pub use xtask::{Error, Result, apalache_pins, support};
#[path = "standalone/mod.rs"]
mod standalone;
