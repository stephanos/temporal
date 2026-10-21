#[path = "../support/mod.rs"]
mod foundation;
use umpire_temporal_annotate::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { id: String }
#[derive(Catalog)]
#[api(request = "test.Wire", type = Wire, covers = foundation::VALIDATOR)]
#[api(any_of(id))]
struct Mirror { #[api(accepts)] id: Uuid }
fn main() {}
