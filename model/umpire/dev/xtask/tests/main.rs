//! Native CLI controls.
#![forbid(unsafe_code)]
#[path = "apalache_install.rs"]
mod apalache_install;
#[path = "controller_counts.rs"]
mod controller_counts;
#[path = "gate_graph.rs"]
mod gate_graph;
#[path = "gate_wiring.rs"]
mod gate_wiring;
#[path = "retired_names.rs"]
mod retired_names;
pub use xtask::{Error, Result, apalache_pins, support};
#[path = "standalone/mod.rs"]
mod standalone;
