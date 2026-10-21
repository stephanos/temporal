use umpire_temporal_annotate::{Catalog, kinds::*};
mod foundation { pub(crate) mod dc { pub(crate) struct Limits; } }
const VALIDATOR: &str = "server.validateRequestID";
#[derive(Clone, Default)]
struct Wire { value: String }
#[derive(Catalog)]
#[api(request = "test.Wire", type = Wire, supplemental, covers = VALIDATOR)]
struct Supplemental { #[api(required)] value: Text }
fn main() {}
