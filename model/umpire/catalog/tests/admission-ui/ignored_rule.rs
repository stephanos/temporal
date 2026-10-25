#[path = "../support/mod.rs"]
mod foundation;
use umpire_catalog::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { opaque: bool }

#[derive(Catalog)]
#[rule(required(opaque))]

#[mirror(response = "test.Wire", type = Wire)]
struct Mirror { #[mirror(ignored = "opaque boolean")] opaque: Ignored }
fn main() {}

type CatalogContext = foundation::dc::Limits;
