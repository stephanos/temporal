#[path = "../support/mod.rs"]
mod foundation;
use umpire_catalog::{Catalog, kinds::*};

#[derive(Clone, Default)]
struct Wire { id: String }

#[derive(Catalog)]
#[mirror(request = "test.Wire", type = Wire)]
struct Mirror { #[rule(accepts, covers = retired::VALIDATOR)] id: Uuid }
fn main() {}

type CatalogContext = foundation::dc::Limits;
