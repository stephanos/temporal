//! Compiles the world-based trait vocabulary without behavioral assertions.

#[allow(unused_imports, missing_docs, unreachable_pub)]
mod kettle {
    include!("fixtures/kettle_trait.rs");
}

use umpire_dsl::Machine;
use kettle::KettleWorld;

#[test]
fn trait_sketch_compiles() {
    fn accepts_machine<M: Machine>() {}
    accepts_machine::<KettleWorld>();
}
