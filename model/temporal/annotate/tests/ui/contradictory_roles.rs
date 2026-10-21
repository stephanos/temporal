#[path = "../support/mod.rs"]
mod foundation;
use umpire_temporal_annotate::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { id: String }

#[derive(Catalog)]
#[api(response = "test.Wire", type = Wire)]
struct Mirror { #[api(decided, observed)] id: Uuid }
fn main() {}
