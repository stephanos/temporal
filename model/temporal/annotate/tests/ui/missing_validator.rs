#[path = "../support/mod.rs"]
mod foundation;
use umpire_temporal_annotate::{Catalog, kinds::*};

#[derive(Clone, Default)]
struct Wire { id: String }

#[derive(Catalog)]
#[api(request = "test.Wire", type = Wire, covers = vanished::VALIDATOR)]
struct Mirror { #[api(accepts)] id: Uuid }
fn main() {}
