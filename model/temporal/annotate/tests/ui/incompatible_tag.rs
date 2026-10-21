#[path = "../support/mod.rs"]
mod foundation;
use umpire_temporal_annotate::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { count: i32 }

#[derive(Catalog)]
#[api(response = "test.Wire", type = Wire)]
struct Mirror { #[api(idempotency_key)] count: Counter }
fn main() {}
