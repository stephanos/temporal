#[path = "../support/mod.rs"]
mod foundation;
use umpire_temporal_annotate::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { value: String }

#[derive(Catalog)]
#[api(response = "test.Wire", type = Wire)]
struct Mirror { #[api(min_items = 1)] value: Text }
fn main() {}
