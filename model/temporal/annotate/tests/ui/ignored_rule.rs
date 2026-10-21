#[path = "../support/mod.rs"]
mod foundation;
use umpire_temporal_annotate::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { opaque: bool }

#[derive(Catalog)]
#[api(response = "test.Wire", type = Wire)]
struct Mirror { #[api(ignored = "opaque boolean", required)] opaque: Ignored }
fn main() {}
