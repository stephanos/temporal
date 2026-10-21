#[path = "../support/mod.rs"]
mod foundation;
use umpire_temporal_annotate::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { value: i32 }

#[derive(Catalog)]
#[api(response = "test.Wire", type = Wire)]
struct Mirror { #[api(ignored = "outside", max_len = 2)] value: Ignored }
fn main() {}
