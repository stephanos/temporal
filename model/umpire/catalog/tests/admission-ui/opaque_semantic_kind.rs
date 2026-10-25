#[path = "../support/mod.rs"]
mod foundation;
use umpire_catalog::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { value: i32 }

#[derive(Catalog)]
#[mirror(response = "test.Wire", type = Wire)]
struct Mirror {
    #[mirror(ignored = "outside")]
    #[rule(max_len = 2)]
    value: Ignored,
}
fn main() {}

type CatalogContext = foundation::dc::Limits;
