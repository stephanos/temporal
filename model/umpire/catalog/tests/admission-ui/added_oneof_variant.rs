#[path = "../support/mod.rs"]
mod foundation;
use umpire_catalog::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { choice: Option<Choice> }
#[derive(Clone)]
enum Choice { Known(String), Added(bool) }
#[derive(Catalog)]
#[mirror(response = "test.Wire", type = Wire)]
struct Mirror {
    #[mirror(ignored = "opaque", oneof(Known = "known"))]
    choice: OneOf<Choice>,
}
fn main() {}

type CatalogContext = foundation::dc::Limits;
