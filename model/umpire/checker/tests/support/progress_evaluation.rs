//! Combined progress evaluation and fixed assumption contracts.

use std::cell::Cell;

use umpire_checker::{Checker, Refusal};
use umpire_dsl::{Assumption, Machine, Outcome, Progress, ProgressEvaluation, Steps, Verdict};

struct World {
    starts: Vec<u8>,
    excluded: u8,
    changing: bool,
    manual: bool,
    second: bool,
    calls: Cell<usize>,
}

impl World {
    fn new(starts: Vec<u8>, excluded: u8) -> Self {
        Self {
            starts,
            excluded,
            changing: false,
            manual: false,
            second: false,
            calls: Cell::new(0),
        }
    }
}

impl Machine for World {
    type State = u8;
    type Action = u8;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<u8> {
        self.starts.clone()
    }

    fn actions(&self) -> Vec<u8> {
        vec![0, 1]
    }

    fn internal(&self, action: &u8) -> bool {
        *action == 0
    }

    fn step(&self, state: &u8, action: u8) -> Steps<u8, (), ()> {
        let mut steps = Steps::new();
        steps.push(((), vec![], if action == 0 { 1 } else { *state }));
        steps
    }

    fn progress(&self) -> Vec<Progress<Self>> {
        let mut claims = vec![if self.manual {
            Progress::new("reaches_one")
                .eventually(|_, state| (*state == 1).into())
                .within(Assumption::without(&[self.excluded]))
        } else {
            Progress::evaluated("reaches_one", |world: &Self, state| {
                world.calls.set(world.calls.get() + 1);
                ProgressEvaluation {
                    trigger: Verdict::Holds,
                    target: (*state == 1).into(),
                    assumption: Assumption::without(&[if world.changing {
                        *state
                    } else {
                        world.excluded
                    }]),
                }
            })
        }];
        if self.second {
            claims.push(Progress::evaluated("other_assumption", |_, state| ProgressEvaluation {
                trigger: Verdict::Holds,
                target: (*state == 1).into(),
                assumption: Assumption::without(&[0]),
            }));
        }
        claims
    }
}

#[test]
fn simultaneous_claims_can_use_different_fixed_assumptions() {
    let mut world = World::new(vec![0], 1);
    world.second = true;
    let result = Checker::explore(&world, 8).unwrap_or_else(|error| panic!("{error}"));
    let report = result.verify_all().unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(report.claims[0].outcome, Outcome::Verified);
    assert_eq!(report.claims[1].outcome, Outcome::Refuted);
    assert!(report.claims.iter().all(|claim| claim.holes.is_empty()));
}

#[test]
fn combined_evaluation_runs_once_per_reached_state_and_rendering_reuses_results() {
    let world = World::new(vec![0, 0], 1);
    let result = Checker::explore(&world, 8).unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(world.calls.get(), 2);
    assert_eq!(
        result.verify_all().unwrap_or_else(|error| panic!("{error}")).outcome(),
        Outcome::Verified
    );
    assert_eq!(
        result.verify_all().unwrap_or_else(|error| panic!("{error}")).outcome(),
        Outcome::Verified
    );
    assert_eq!(world.calls.get(), 2);
}

#[test]
fn assumptions_are_fixed_per_exploration_rather_than_across_worlds() {
    for (excluded, expected) in [(1, Outcome::Verified), (0, Outcome::Refuted)] {
        let world = World::new(vec![0], excluded);
        let result = Checker::explore(&world, 8).unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(
            result.verify_all().unwrap_or_else(|error| panic!("{error}")).outcome(),
            expected
        );
    }
    let mut world = World::new(vec![0], 0);
    world.changing = true;
    let result = Checker::explore(&world, 8).unwrap_or_else(|error| panic!("{error}"));
    let report = result.verify_all().unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(report.outcome(), Outcome::Unknown);
    assert!(report.claims[0].holes.iter().any(|reason| reason.contains("assumption")));
    assert_eq!(world.calls.get(), 2);
}

#[test]
fn dynamic_invalid_actions_are_refused_only_after_real_state_evaluation() {
    for empty in [false, true] {
        let world = World::new(if empty { vec![] } else { vec![0] }, 2);
        let result = Checker::explore(&world, 8);
        if empty {
            let result = result.unwrap_or_else(|error| panic!("{error}"));
            assert_eq!(
                result.verify_all().unwrap_or_else(|error| panic!("{error}")).outcome(),
                Outcome::Verified
            );
            assert_eq!(world.calls.get(), 0);
        } else {
            assert!(matches!(result, Err(Refusal::ScopeAction { .. })));
            assert_eq!(world.calls.get(), 1);
        }
    }
}

#[test]
fn manual_invalid_actions_are_refused_even_without_reached_states() {
    let mut world = World::new(vec![], 2);
    world.manual = true;
    assert!(matches!(Checker::explore(&world, 8), Err(Refusal::ScopeAction { .. })));
    assert_eq!(world.calls.get(), 0);
}
