#[path = "../support/mod.rs"]
mod foundation;
use umpire_catalog::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { count: i32 }

#[derive(Catalog)]
#[mirror(response = "test.Wire", type = Wire)]
struct Mirror { #[mirror(idempotency_key)] count: Counter }
fn main() {}

type CatalogContext = foundation::dc::Limits;
