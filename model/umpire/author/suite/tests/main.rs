//! Integration tests for the umpire-suite crate.

#![forbid(unsafe_code)]

#[path = "main/alternative_search.rs"]
mod alternative_search;
#[path = "main/declared_world.rs"]
mod declared_world;
#[path = "main/explicit_world.rs"]
mod explicit_world;
#[path = "main/fixtures.rs"]
mod fixtures;
#[path = "main/full_patterns.rs"]
mod full_patterns;
#[path = "main/large_space.rs"]
mod large_space;
#[path = "main/world_domains.rs"]
mod world_domains;

use fixtures::{coin, failure, kettle, sample_kettle};
