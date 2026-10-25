#[path = "../support/mod.rs"]
mod foundation;
use umpire_catalog::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { value: String }

#[derive(Catalog)]
#[mirror(response = "test.Wire", type = Wire)]
struct Mirror { #[rule(min_items = 1)] value: Text }
fn main() {}

type CatalogContext = foundation::dc::Limits;
