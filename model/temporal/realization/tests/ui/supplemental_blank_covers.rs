use umpire_temporal_annotate::{Catalog, kinds::*};
mod foundation { pub(crate) mod dc { pub(crate) struct Limits; } }
const VALIDATOR: &str = "server.validateRequestID";
#[derive(Clone, Default)]
struct Wire { value: String }
const BLANK_ASCII: &str = " \t\n";
const BLANK_UNICODE: &str = "\u{2003}";
#[derive(Catalog)]
#[api(request = "test.Wire", type = Wire, supplemental, covers = BLANK_ASCII)]
struct Ascii { #[api(accepts)] value: Uuid }
#[derive(Catalog)]
#[api(request = "test.Wire", type = Wire, supplemental, covers = BLANK_UNICODE)]
struct Unicode { #[api(accepts)] value: Uuid }
fn main() {}
