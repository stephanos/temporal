#[path = "../support/mod.rs"]
mod foundation;
use umpire_catalog::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { id: String }

#[derive(Catalog)]
#[mirror(response = "test.Wire", type = Wire)]
struct Mirror { #[mirror(decided, observed)] id: Uuid }
fn main() {}

type CatalogContext = foundation::dc::Limits;
