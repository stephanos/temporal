#[path = "../support/mod.rs"]
mod foundation;
use umpire_temporal_annotate::{Catalog, kinds::*};

#[derive(Clone, Default)]
struct Wire { id: String }

#[derive(Catalog)]
#[mirror(request = "test.Wire", type = Wire)]
struct Mirror { #[rule(accepts, covers = retired::VALIDATOR)] id: Uuid }
fn main() {}
