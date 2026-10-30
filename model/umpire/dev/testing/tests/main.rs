//! Shared testing controls without process-global fixtures.

#![forbid(unsafe_code)]

#[path = "main/declared_values.rs"]
mod declared_values;
#[path = "main/discovery.rs"]
mod discovery;
#[path = "main/empty_control.rs"]
mod empty_control;
#[path = "main/fixture_composition.rs"]
mod fixture_composition;
#[path = "main/orphan_control.rs"]
mod orphan_control;
