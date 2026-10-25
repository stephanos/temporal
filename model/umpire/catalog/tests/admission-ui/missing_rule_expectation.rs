#[path = "../support/mod.rs"]
mod foundation;
use umpire_catalog::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { id: String }
#[derive(Catalog)]
#[mirror(request = "test.Wire", type = Wire)]
#[rule(any_of(id))]
struct Mirror { #[rule(accepts)] id: Uuid }
fn main() {}

type CatalogContext = foundation::dc::Limits;
