//! Compiles the trait sketch with an explicit progress state type for Rust inference.

mod umpire {
    pub(super) use framework_checker as Checker;
    pub(super) use framework_dsl::{Claim, Machine, Outcome, Progress, Scope, StepClaim};
}

#[allow(unused_imports, missing_docs, unreachable_pub)]
mod kettle {
    use super::umpire;
    use umpire::StepClaim;

    include!("fixtures/kettle_trait.rs");
}

use kettle::Kettle;
use umpire::Machine;

#[test]
fn trait_sketch_compiles() {
    fn accepts_machine<M: Machine>() {}
    accepts_machine::<Kettle>();
}
