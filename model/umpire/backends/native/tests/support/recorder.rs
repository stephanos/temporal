//! Recorded check evidence follows its actual invocation and counterexample path.

use std::cell::Cell;

use umpire_checker::Checker;
use umpire_dsl::{CheckOutcome, Claim, Machine, StepClaim, Steps, Verdict, check, claim};

struct World {
    calls: Cell<usize>,
}

impl World {
    fn checks(&self, state: u8) {
        self.calls.set(self.calls.get() + 1);
        check!(state == 0);
    }
}

impl Machine for World {
    type State = u8;
    type Action = u8;
    type Outcome = bool;
    type Fact = ();

    fn starts(&self) -> Vec<u8> {
        vec![0]
    }
    fn actions(&self) -> Vec<u8> {
        vec![0]
    }
    fn internal(&self, _: &u8) -> bool {
        false
    }
    fn step(&self, state: &u8, _: u8) -> Steps<u8, bool, ()> {
        if *state == 0 {
            [(true, vec![], 1), (false, vec![], 1)].into_iter().collect()
        } else {
            Steps::new()
        }
    }
    fn invariants(&self) -> Vec<Claim<Self>> {
        vec![
            Claim::with_checks(
                "state_check",
                |world, state| world.checks(*state),
                |world, state| claim::evaluate(|| world.checks(*state)).verdict,
            ),
            Claim::new("manual", |world, state| {
                let generated = Claim::<Self>::with_checks(
                    "discarded",
                    |world, state| world.checks(*state),
                    |world, state| claim::evaluate(|| world.checks(*state)).verdict,
                );
                let _ = (generated.predicate)(world, state);
                Verdict::Fails
            }),
            Claim::new("panic", |_, _| {
                claim::evaluate(|| {
                    check!(false);
                    panic!("broken predicate");
                })
                .verdict
            }),
        ]
    }
    fn postconditions(&self) -> Vec<StepClaim<Self>> {
        vec![StepClaim::<Self>::after_checks("second_choice", 0, |_, _, _, _, outcome, _| {
            check!(*outcome);
        })]
    }
}

#[test]
fn evidence_belongs_to_actual_evaluation_and_counterexample_state()
-> Result<(), Box<dyn std::error::Error>> {
    let world = World { calls: Cell::new(0) };
    let result = Checker::explore(&world, 10)?;
    let report = result.verify_all()?;
    assert!(report.claims.iter().all(|claim| claim.verdict.is_none() && claim.hint.is_none()));
    assert_eq!(world.calls.get(), 3);
    let state = &report.claims[0];
    assert_eq!(state.outcome, CheckOutcome::Refuted);
    let path = state.path.as_ref().ok_or("state counterexample missing")?;
    assert_eq!(path.steps.last().ok_or("state edge missing")?.after, 1);
    assert_eq!(path.failure.as_deref(), Some("check!(state == 0) failed at 1"));
    assert_eq!(report.claims[1].path.as_ref().ok_or("manual path missing")?.failure, None);
    assert_eq!(
        report.claims[2].path.as_ref().ok_or("panic path missing")?.failure.as_deref(),
        Some("claim panic panicked: broken predicate")
    );
    let edge = report.claims[3].path.as_ref().ok_or("edge counterexample missing")?;
    assert_eq!(edge.steps.last().ok_or("edge missing")?.choice, 1);
    assert_eq!(edge.failure.as_deref(), Some("check!(*outcome) failed at 1"));
    let mut text = report.to_string();
    text.push_str(&path.to_string());
    assert_eq!(world.calls.get(), 3);
    let again = Checker::explore(&world, 10)?.verify_all()?;
    assert_eq!(again.claims[1].path.as_ref().ok_or("repeated manual path missing")?.failure, None);
    Ok(())
}
