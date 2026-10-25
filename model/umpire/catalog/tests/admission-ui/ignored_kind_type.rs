#[path = "../support/mod.rs"]
mod foundation;
use umpire_catalog::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { value: i32 }

#[derive(Catalog)]
#[mirror(response = "test.Wire", type = Wire)]
struct Mirror { #[mirror(ignored = "outside")] value: Uuid }
fn main() {}

type CatalogContext = foundation::dc::Limits;
