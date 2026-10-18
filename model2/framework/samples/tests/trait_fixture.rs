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

use kettle::{Act, Kettle};
use umpire::Machine;

#[test]
fn trait_sketch_registers_executable_claims() -> Result<(), &'static str> {
    let start = Kettle::starts().remove(0);
    assert_eq!(Kettle::starts().len(), 81);
    assert_eq!(Kettle::actions().len(), 4);
    assert!(Kettle::internal(Act::HeaterHeat));
    assert!(!Kettle::internal(Act::UserPressButton));
    assert!(start.step(Act::HeaterHeat).is_none());

    let on = start.step(Act::UserPressButton).ok_or("kettle refused the on switch")?;
    let hot = on.step(Act::HeaterHeat).ok_or("switched-on kettle refused to heat")?;
    let invariant = Kettle::invariants().remove(0);
    assert_eq!(invariant.name, "beeps_only_when_off");
    assert!((invariant.predicate)(&start));
    let postconditions = Kettle::postconditions();
    assert!(postconditions[0].holds(&on, &hot, Act::HeaterHeat));
    assert!(!postconditions[0].holds(&start, &hot, Act::HeaterHeat));
    assert!(!postconditions[1].applies_to(Act::HeaterHeat));
    assert!(postconditions[1].applies_to(Act::HeaterFailover));
    assert!(postconditions[1].holds(&on, &hot, Act::HeaterHeat));
    assert!(!postconditions[1].holds(&on, &hot, Act::HeaterFailover));

    let progress = Kettle::progress().remove(0);
    assert_eq!(progress.name, "boils");
    assert!(!(progress.trigger)(&start));
    assert!((progress.trigger)(&on));
    assert!(!(progress.target)(&on));
    assert!(!progress.scope.allows(Act::UserDepressButton));
    assert!(progress.scope.allows(Act::HeaterHeat));

    let mut boiling = on;
    for _ in -40..100 {
        boiling = boiling.step(Act::HeaterHeat).ok_or("kettle stopped heating before boiling")?;
    }
    let beeping = boiling.step(Act::HeaterFailover).ok_or("boiling kettle refused failover")?;
    assert!(postconditions[1].holds(&boiling, &beeping, Act::HeaterFailover));
    assert!((progress.target)(&beeping));
    Ok(())
}
