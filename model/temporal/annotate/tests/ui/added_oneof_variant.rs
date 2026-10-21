#[path = "../support/mod.rs"]
mod foundation;
use umpire_temporal_annotate::{Catalog, kinds::*};
#[derive(Clone, Default)]
struct Wire { choice: Option<Choice> }
#[derive(Clone)]
enum Choice { Known(String), Added(bool) }
#[derive(Catalog)]
#[api(response = "test.Wire", type = Wire)]
struct Mirror {
    #[api(ignored = "opaque", oneof(Known = "known"))]
    choice: OneOf<Choice>,
}
fn main() {}
