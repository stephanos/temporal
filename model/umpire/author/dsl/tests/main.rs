//! Integration tests for the umpire-dsl crate.

#![forbid(unsafe_code)]

#[path = "main/action_inputs_builders.rs"]
mod action_inputs_builders;
#[path = "main/catalog.rs"]
mod catalog_tests;
#[path = "main/closable.rs"]
mod closable;
#[path = "main/countdown.rs"]
mod countdown;
#[path = "main/declared_world.rs"]
mod declared_world;
#[path = "main/finite_domains.rs"]
mod finite_domains;
#[path = "main/optional_values.rs"]
mod optional_values;
#[path = "main/outcome.rs"]
mod outcome;
#[path = "main/path_domains.rs"]
mod path_domains;
#[path = "main/path_expression_semantics.rs"]
pub mod path_expression_semantics;
#[path = "main/path_language_patterns.rs"]
pub mod path_language_patterns;
#[path = "main/path_leading_patterns.rs"]
pub mod path_leading_patterns;
#[path = "main/readme.rs"]
mod readme;
#[path = "main/replay_values.rs"]
mod replay_values;
#[path = "main/world.rs"]
mod world;
#[path = "main/world_dispatch.rs"]
mod world_dispatch;
