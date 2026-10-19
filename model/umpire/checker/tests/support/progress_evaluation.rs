//! Combined progress evaluation and fixed assumption contracts.

use std::cell::Cell;

use umpire_checker::{Checker, ProgressVerdict, Refusal};
use umpire_dsl::{
    Assumption, AssumptionExt, CheckOutcome, Machine, Progress, ProgressEvaluation, Steps, Verdict,
};

struct World {
    starts: Vec<u8>,
    excluded: u8,
    changing: u8,
    manual: bool,
    second: bool,
    duplicate: bool,
    actions: Vec<u8>,
    steps: Cell<usize>,
    calls: Cell<usize>,
}

impl World {
    fn new(starts: Vec<u8>, excluded: u8) -> Self {
        Self {
            starts,
            excluded,
            changing: 0,
            manual: false,
            second: false,
            duplicate: false,
            actions: vec![0, 1],
            steps: Cell::new(0),
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
        self.actions.clone()
    }

    fn internal(&self, action: &u8) -> bool {
        *action == 0
    }

    fn step(&self, state: &u8, action: u8) -> Steps<u8, (), ()> {
        self.steps.set(self.steps.get() + 1);
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
                    bound: None,
                    trigger: Verdict::Holds,
                    target: (*state == 1).into(),
                    assumption: match world.changing {
                        1 => Assumption::without(&[*state]),
                        2 => Assumption::only(&[*state]),
                        3 => Assumption::without(&[world.excluded]).fair([*state]),
                        _ => Assumption::without(&[world.excluded]),
                    },
                }
            })
        }];
        if self.second {
            claims.push(Progress::evaluated("other_assumption", |_, state| ProgressEvaluation {
                bound: None,
                trigger: Verdict::Holds,
                target: (*state == 1).into(),
                assumption: Assumption::without(&[0]),
            }));
        }
        if self.duplicate {
            claims.push(Progress::new("reaches_one"));
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
    assert_eq!(report.claims[0].outcome, CheckOutcome::Verified);
    assert_eq!(report.claims[1].outcome, CheckOutcome::Refuted);
    assert!(report.claims.iter().all(|claim| claim.holes.is_empty()));
}

#[test]
fn combined_evaluation_runs_once_per_reached_state_and_rendering_reuses_results() {
    let world = World::new(vec![0, 0], 1);
    let result = Checker::explore(&world, 8).unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(world.calls.get(), 2);
    assert_eq!(
        result.verify_all().unwrap_or_else(|error| panic!("{error}")).outcome(),
        CheckOutcome::Verified
    );
    assert_eq!(
        result.verify_all().unwrap_or_else(|error| panic!("{error}")).outcome(),
        CheckOutcome::Verified
    );
    assert_eq!(world.calls.get(), 2);
}

#[test]
fn assumptions_are_fixed_per_exploration_rather_than_across_worlds() {
    for (excluded, expected) in [(1, CheckOutcome::Verified), (0, CheckOutcome::Refuted)] {
        let world = World::new(vec![0], excluded);
        let result = Checker::explore(&world, 8).unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(
            result.verify_all().unwrap_or_else(|error| panic!("{error}")).outcome(),
            expected
        );
    }
    for changing in [1, 2, 3] {
        let mut world = World::new(vec![0], 0);
        world.changing = changing;
        let result = Checker::explore(&world, 8).unwrap_or_else(|error| panic!("{error}"));
        let report = result.verify_all().unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(report.outcome(), CheckOutcome::Unknown);
        assert!(report.claims[0].holes.iter().any(|reason| reason.contains("assumption")));
        assert_eq!(world.calls.get(), 2);
    }
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
                CheckOutcome::Verified
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

#[test]
fn registered_queries_reuse_graph_and_canonical_callbacks_without_mutating_cache()
-> umpire_testing::TestResult {
    for manual in [false, true] {
        let mut world = World::new(vec![0], 1);
        world.manual = manual;
        let result = Checker::explore(&world, 8)?;
        let before = umpire_testing::render(&result)?;
        let counts = result.counts;
        let steps = world.steps.get();
        let calls = world.calls.get();
        let mut selector =
            Progress::evaluated("reaches_one", |_, _| panic!("fabricated evaluator"))
                .whenever(|_, _| panic!("fabricated trigger"))
                .eventually(|_, _| panic!("fabricated target"));
        selector.scope = Assumption::only(&[9]);
        selector.bound = Some(0);
        for round in 1..=2 {
            let queried = result.verify_progress(&world, &selector)?;
            assert_eq!(queried.verdict, Some(ProgressVerdict::Verified { longest: Some(1) }));
            assert_eq!(world.calls.get(), calls + if manual { 0 } else { round * 2 });
            assert_eq!(world.steps.get(), steps);
            assert_eq!(result.counts, counts);
            assert_eq!(umpire_testing::render(&result)?, before);
        }
    }
    Ok(())
}

#[test]
fn registered_queries_refuse_missing_ambiguous_and_changed_catalogs() -> umpire_testing::TestResult
{
    let mut world = World::new(vec![0], 1);
    let result = Checker::explore(&world, 8)?;
    assert!(
        matches!(result.verify_progress(&world, &Progress::new("absent")), Err(Refusal::ProgressRegistrationMissing { claim }) if claim == "absent")
    );
    let selector = Progress::new("reaches_one");
    let calls = world.calls.get();
    for actions in [vec![], vec![0], vec![1, 0], vec![0, 1, 2], vec![0, 2]] {
        world.actions = actions;
        assert!(matches!(
            result.verify_progress(&world, &selector),
            Err(Refusal::ProgressCatalogChanged)
        ));
    }
    assert_eq!(world.calls.get(), calls);
    world.actions = vec![0, 1];
    world.duplicate = true;
    let ambiguous = Checker::explore(&world, 8)?;
    assert!(
        matches!(ambiguous.verify_progress(&world, &selector), Err(Refusal::ProgressRegistrationAmbiguous { claim }) if claim == "reaches_one")
    );
    Ok(())
}

#[test]
fn zero_start_queries_validate_overrides_without_evaluating_a_state() -> umpire_testing::TestResult
{
    for manual in [false, true] {
        let mut world = World::new(vec![], 1);
        world.manual = manual;
        let result = Checker::explore(&world, 8)?;
        let claim = result.verify_progress(&world, &Progress::new("reaches_one"))?;
        assert_eq!(claim.verdict, Some(ProgressVerdict::Verified { longest: Some(0) }));
        for assumption in
            [Assumption::only(&[9]), Assumption::without(&[9]), Assumption::fair(&[9])]
        {
            let selector = Progress::new("reaches_one").assuming(assumption);
            assert!(matches!(
                result.verify_progress(&world, &selector),
                Err(Refusal::ScopeAction { .. })
            ));
        }
        assert!(matches!(
            result.verify_progress(
                &world,
                &Progress::new("reaches_one").assuming(Assumption::only(&[] as &[u8]))
            ),
            Err(Refusal::EmptyAssumption)
        ));
        assert!(matches!(
            result.verify_progress(&world, &Progress::new("reaches_one").within_steps(0)),
            Err(Refusal::ProgressBound { .. })
        ));
        assert_eq!(world.calls.get(), 0);
        assert_eq!(world.steps.get(), 0);
    }
    Ok(())
}

#[test]
fn query_assumption_override_replaces_inconsistent_metadata() -> umpire_testing::TestResult {
    for changing in [1, 2, 3] {
        let mut world = World::new(vec![0], 1);
        world.changing = changing;
        let result = Checker::explore(&world, 8)?;
        let selector = Progress::new("reaches_one").assuming(Assumption::only(&[0]));
        assert_eq!(
            result.verify_progress(&world, &selector)?.verdict,
            Some(ProgressVerdict::Verified { longest: Some(1) })
        );
        assert_eq!(result.verify_all()?.claims[0].verdict, Some(ProgressVerdict::Unknown));
    }
    Ok(())
}
