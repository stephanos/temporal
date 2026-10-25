use umpire_catalog::{Catalog, kinds::*};
mod foundation { pub(crate) mod dc { pub(crate) struct Limits; } }
#[derive(Clone, Default)]
struct Wire { value: String }
#[derive(Catalog)]
#[mirror(request = "test.Wire", type = Wire, supplemental)]
struct Supplemental { value: Uuid }
fn main() {}

type CatalogContext = foundation::dc::Limits;
