//! Integration controls for backends/tla.

#![forbid(unsafe_code)]

#[path = "main/emission.rs"]
mod emission;
#[path = "main/inputs.rs"]
mod inputs;
#[path = "main/narrow_remainder.rs"]
mod narrow_remainder;
#[path = "main/optional_facts.rs"]
mod optional_facts;
#[path = "main/optional_source.rs"]
mod optional_source;
#[path = "main/optional_ticks.rs"]
mod optional_ticks;
#[path = "main/phased.rs"]
mod phased;
#[path = "main/safety.rs"]
mod safety;
#[path = "main/sequential_checks.rs"]
mod sequential_checks;
#[path = "main/shadowing.rs"]
mod shadowing;
#[path = "main/source.rs"]
mod source;
#[path = "main/vector.rs"]
mod vector;
#[path = "main/world.rs"]
mod world;
#[path = "main/world_fairness_binding.rs"]
mod world_fairness_binding;
