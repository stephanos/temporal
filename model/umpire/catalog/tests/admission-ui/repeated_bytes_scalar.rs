#[path = "../support/mod.rs"]
mod foundation;
use umpire_catalog::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { value: Vec<u8> }

#[derive(Catalog)]
#[mirror(response = "test.Wire", type = Wire)]
struct Mirror { #[rule(unique = true)] value: Bytes }
fn main() {}

type CatalogContext = foundation::dc::Limits;
