#[path = "../support/mod.rs"]
mod foundation;
use umpire_temporal_annotate::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { id: String }
#[derive(Catalog)]
#[mirror(request = "test.Wire", type = Wire)]
#[rule(any_of(id))]
struct Mirror { #[rule(accepts)] id: Uuid }
fn main() {}
