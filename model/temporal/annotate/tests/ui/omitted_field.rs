#[path = "../support/mod.rs"]
mod foundation;
use umpire_temporal_annotate::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { id: String, omitted: String }

#[derive(Catalog)]
#[api(response = "test.Wire", type = Wire)]
struct Mirror { id: Uuid }
fn main() {}
