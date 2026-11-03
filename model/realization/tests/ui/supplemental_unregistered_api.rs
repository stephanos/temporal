use umpire_catalog::{Catalog, kinds::Uuid};
#[derive(Clone, Default)]
struct Wire { value: String }
#[derive(Catalog)]
#[mirror(request = "test.Wire", type = Wire, supplemental)]
struct Supplemental { #[api(accepts)] #[rule(accepts)] value: Uuid }
fn main() {}
type CatalogContext = ();
