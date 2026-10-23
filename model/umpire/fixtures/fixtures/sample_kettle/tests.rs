use crate::sample_kettle::kettle;
use umpire_dsl::Claim;

umpire_testing::fixture!(kettle::world(), 1 << 20, Claim::new("never_beeps", |_, state: &kettle::Model| (!state.beeping).into()), extra = suite_files()?);

fn suite_files() -> umpire_testing::RunResult {
    use kettle::{Heater, User};
    use umpire_suite::*;

    let space = Space::from([room()]);
    let covered = cover(&space, Cases((80 + 1) * 2), path![room(), User::PressButton, [Heater::Heat; 0..=80], .., (User::DepressButton | Heater::Failover)])?;
    assert_eq!(covered.len(), 162);
    for (index, run) in covered.runs().iter().enumerate() {
        let heats = index / 2;
        assert_eq!(run.state(), &if index % 2 == 0 { at(20 + i32::try_from(heats)?, false, false) } else { at(100, false, true) });
        assert_eq!(run.path().steps.len(), if index % 2 == 0 { heats + 2 } else { 82 });
    }
    let adjacent = cover(&space, Cases(162), path![room(), User::PressButton, [Heater::Heat; 0..=80], (User::DepressButton | Heater::Failover)]);
    assert!(matches!(adjacent, Err(Refusal::Alternative { index: 1, .. })));
    let sampled = sample(&space.clone().walk_bound(12), path![room(), User::PressButton, .., User::DepressButton], 4, 42)?;
    let repeated = sample(&space.walk_bound(12), path![room(), User::PressButton, .., User::DepressButton], 4, 42)?;
    assert_eq!(sampled.expansion(), repeated.expansion());
    let enabled = enabledness()?;
    Ok(umpire_testing::Files::from([
        ("expansion/closings.txt".into(), covered.expansion().into()),
        ("expansion/enabledness.txt".into(), enabled.expansion().into()),
        ("expansion/sample.txt".into(), sampled.expansion().into()),
        ("coverage.txt".into(), format!("closings\n{}\nenabledness\n{}\nsample (finite experiment)\n{}\nadjacent refusal: {}\n", covered.coverage(), enabled.coverage(), sampled.coverage(), adjacent.err().ok_or("expected adjacent refusal")?)),
    ]))
}

fn enabledness() -> Result<umpire_suite::Covered<kettle::Model>, umpire_suite::Refusal> {
    use kettle::{Action, Heater, User};
    use umpire_dsl::{ActionDomain, Path, Pattern, Waypoint};
    use umpire_suite::*;

    let rows: [(_, Action, _); 9] = [
        (coldest(), User::PressButton.into(), at(-40, true, false)),
        (at(-40, true, false), Heater::Heat.into(), at(-39, true, false)),
        (room(), User::PressButton.into(), at(20, true, false)),
        (at(20, true, false), Heater::Heat.into(), at(21, true, false)),
        (at(21, true, true), User::DepressButton.into(), at(21, false, false)),
        (at(99, true, false), Heater::Heat.into(), at(100, true, false)),
        (at(100, true, false), Heater::Failover.into(), at(100, false, true)),
        (at(100, false, true), User::PressButton.into(), at(100, true, false)),
        (at(101, true, false), Heater::Failover.into(), at(101, false, true)),
    ];
    let space = Space::from(rows.iter().map(|(before, _, _)| before.clone()).collect::<Vec<_>>());
    let pattern = Path::pattern(Pattern::Either(rows.into_iter().map(|(before, action, after)| Pattern::Sequence(vec![
        Pattern::state(Waypoint::Is(before)),
        Pattern::action(ActionDomain::new(Ok(1), move |_| action.clone()), None, None),
        Pattern::state(Waypoint::Is(after)),
    ])).collect()));
    for (before, action) in [
        (room(), Action::from(Heater::Heat)),
        (room(), Heater::Failover.into()),
        (room(), User::DepressButton.into()),
        (at(20, true, false), User::PressButton.into()),
        (at(21, false, false), Heater::Heat.into()),
        (at(100, true, false), Heater::Heat.into()),
        (at(100, false, true), Heater::Failover.into()),
    ] {
        assert!(matches!(Run::from(before.clone()).step(action.clone()), Err(Refusal::Disabled { action: refused, state }) if refused == action.to_string() && state == before.to_string()));
    }
    cover(&space, Cases(9), pattern)
}

/// A kettle at `temperature` degrees Celsius, heating or not, beeping or not.
fn at(temperature: i32, heating_on: bool, beeping: bool) -> kettle::Model {
    kettle::Model { temperature, heating_on, beeping }
}

/// The coldest start, which reaches every warmer one by press, heats and depress.
fn coldest() -> kettle::Model {
    at(-40, false, false)
}

/// Room temperature, the element off and quiet.
fn room() -> kettle::Model {
    at(20, false, false)
}

mod examples {
    use umpire_dsl::CheckOutcome;
    use umpire_suite::*;

    use super::{at, coldest, kettle::{Heater, Model, User}, room};

    #[test]
    fn archived_enabledness_table_keeps_every_positive_and_disabled_row() -> Result<(), Refusal> {
        assert_eq!(super::enabledness()?.len(), 9);
        Ok(())
    }

    #[test]
    fn boundary_from_99_to_100_and_the_failover_guard() -> Result<(), Refusal> {
        let at_beep = Space::from([at(100, false, true)]);
        let report = verify_all(&at_beep)?;
        let claim = report.claims.iter().find(|claim| claim.name == "beeps_only_when_off");
        assert_eq!(claim.map(|claim| claim.outcome), Some(CheckOutcome::Verified));
        assert_eq!(refuted(&at_beep, |k: &Model| !k.beeping)?.steps.len(), 0);
        Ok(())
    }

    #[test]
    fn switched_on_at_room_temperature_beeps_after_80_heats_and_the_failover() -> Result<(), Refusal> {
        let on = Run::from(room()).step(User::PressButton)?;
        let boiled = on.repeat(80, Heater::Heat)?.step(Heater::Failover)?;
        expect! { k = boiled.state(); *k == at(100, false, true); }
        assert_eq!(boiled.path().steps.len(), 82);

        // The same story told by its milestones: on, then 81 steps to the beep, none a depress.
        let story = Run::along(path![at(20, true, false), .., |k: &Model| k.beeping])?;
        let steps = &story.path().steps;
        assert_eq!(steps.len(), 81);
        assert!(steps.iter().all(|step| step.action.to_string() != "User::DepressButton"));
        expect! { k = story.state(); *k == at(100, false, true); }
        Ok(())
    }
}

mod explorations {
    use umpire_checker::{Checker, ProgressVerdict};
    use umpire_dsl::{CheckOutcome, Machine, Registered, StepClaim, StepError, StepPredicate, Steps, Verdict};
    use umpire_suite::*;

    use super::{at, coldest, kettle::{Action, Model, World}, room};

    fn outcomes(report: &umpire_checker::Report<impl Machine>) -> Vec<(&str, CheckOutcome)> {
        report.claims.iter().map(|claim| (claim.name.as_str(), claim.outcome)).collect()
    }

    fn boils(report: &umpire_checker::Report<impl Machine>) -> Option<&ProgressVerdict> {
        report.claims.iter().find(|claim| claim.name == "boils")?.verdict.as_ref()
    }

    const VERIFIED: [(&str, CheckOutcome); 4] = [
        ("beeps_only_when_off", CheckOutcome::Verified),
        ("warms_only_when_on", CheckOutcome::Verified),
        ("fails_over_at_boiling", CheckOutcome::Verified),
        ("boils", CheckOutcome::Verified),
    ];

    #[test]
    fn every_start_from_minus_40_to_40_boils_within_141_steps() -> Result<(), Refusal> {
        let starts = Model::world().starts();
        let temperatures: Vec<i32> = starts.iter().map(|k| k.temperature).collect();
        assert_eq!(temperatures, (-40..=40).collect::<Vec<_>>());
        assert!(starts.iter().all(|k| !k.heating_on && !k.beeping));
        let report = verify_all(&Space::from(starts))?;
        assert_eq!(outcomes(&report), VERIFIED);
        assert_eq!(boils(&report), Some(&ProgressVerdict::Verified { longest: Some(141) }));
        Ok(())
    }

    #[test]
    fn claims_hold_from_room_and_coldest_and_boils_in_81_steps_from_room() -> Result<(), Refusal> {
        let both = verify_all(&Space::from([room(), coldest()]))?;
        assert_eq!(outcomes(&both), VERIFIED);
        assert_eq!(both.warnings, Vec::<String>::new());
        // Switched on at the coldest start, the water boils after 140 heats and the failover.
        assert_eq!(boils(&both), Some(&ProgressVerdict::Verified { longest: Some(141) }));

        // boils assumes never_switched_off: from room temperature the beep is at most 81 steps
        // after switching on, and the exploration is breadth-first, 82 steps deep.
        let warm = verify_all(&Space::from([room()]))?;
        assert_eq!(outcomes(&warm), VERIFIED);
        assert_eq!(boils(&warm), Some(&ProgressVerdict::Verified { longest: Some(81) }));
        assert_eq!((warm.counts.depth, warm.counts.cut), (82, false));
        Ok(())
    }

    #[test]
    fn failing_over_above_boiling_refutes_fails_over_at_boiling() -> Result<(), Refusal> {
        let at_boiling = verify_all(&Space::from([at(99, true, false)]))?;
        assert_eq!(outcomes(&at_boiling), VERIFIED);
        let above = verify_all(&Space::from([at(101, true, false)]))?;
        let claim = above.claims.iter().find(|claim| claim.name == "fails_over_at_boiling");
        assert_eq!(claim.map(|claim| claim.outcome), Some(CheckOutcome::Refuted));
        let spelled = claim.and_then(|claim| claim.path.as_ref()).map(ToString::to_string);
        assert_eq!(
            spelled.as_deref().and_then(|path| path.lines().nth(1)),
            Some("1: Heater::Failover choice=0 -> Model { temperature: 101, heating_on: false, beeping: true }")
        );
        Ok(())
    }

    #[test]
    fn never_beeps_is_refuted_by_press_80_heats_and_failover() -> Result<(), Refusal> {
        let path = refuted(&Space::from([room()]), |k: &Model| !k.beeping)?;
        let actions: Vec<String> = path.steps.iter().map(|step| step.action.to_string()).collect();
        let mut expected = vec!["User::PressButton".to_owned()];
        expected.extend(std::iter::repeat_n("Heater::Heat".to_owned(), 80));
        expected.push("Heater::Failover".to_owned());
        assert_eq!(actions, expected);

        let found = find(&Space::from([room()]), path![room(), .., |k: &Model| k.beeping])?;
        assert_eq!(found.steps.len(), 82);
        Ok(())
    }

    #[test]
    fn a_budget_of_10_answers_unknown_and_a_counterexample_before_a_cut_stands() -> Result<(), Refusal> {
        let cut = verify_all(&Space::from([room()]).budget(10))?;
        assert!(cut.counts.cut);
        assert_eq!(cut.outcome(), CheckOutcome::Unknown);
        assert!(cut.claims.iter().all(|claim| claim.outcome == CheckOutcome::Unknown));
        let never_beeps = refuted(&Space::from([room()]).budget(10), |k: &Model| !k.beeping);
        assert_eq!(never_beeps.err(), Some(Refusal::Unknown { budget: 10 }));

        // From room temperature the beep is the deepest state, so any cut misses it. Started
        // heating at 99 as well, the beep is reached before a budget of 12 cuts the exploration,
        // and never_beeps is refuted all the same.
        let short = Space::from([room(), at(99, true, false)]).budget(12);
        assert_eq!(verify_all(&short)?.outcome(), CheckOutcome::Unknown);
        assert_eq!(refuted(&short, |k: &Model| !k.beeping)?.steps.len(), 2);
        Ok(())
    }

    #[test]
    fn beeps_only_when_off_holds_quiet_hot_and_heating_and_a_heating_beeping_start_refutes_it() -> Result<(), Refusal> {
        let held = verify_all(&Space::from([room(), at(100, false, true), at(20, true, false)]))?;
        let claim = held.claims.iter().find(|claim| claim.name == "beeps_only_when_off");
        assert_eq!(claim.map(|claim| claim.outcome), Some(CheckOutcome::Verified));

        let report = verify_all(&Space::from([at(100, true, true)]))?;
        let claim = report.claims.iter().find(|claim| claim.name == "beeps_only_when_off");
        assert_eq!(claim.map(|claim| claim.outcome), Some(CheckOutcome::Refuted));
        assert_eq!(claim.and_then(|claim| claim.path.as_ref()).map(|path| path.steps.len()), Some(0));
        Ok(())
    }

    #[test]
    fn without_the_heater_fails_over_at_boiling_is_never_triggered() -> Result<(), umpire_checker::Refusal> {
        let report = Checker::explore(&Unheated, Checker::BUDGET)?.verify_all()?;
        assert!(!report.counts.cut);
        assert!(report.warnings.contains(&"after claim fails_over_at_boiling never triggered".to_owned()));
        Ok(())
    }

    /// Kettle with its heater excluded: the user's actions only, under Kettle's step claims. A
    /// `Space` chooses starts, never actions, so this world is written out.
    struct Unheated;

    impl Machine for Unheated {
        type State = Model;
        type Action = Action;
        type Outcome = ();
        type Fact = ();

        fn starts(&self) -> Vec<Model> {
            World.starts()
        }

        fn actions(&self) -> Vec<Action> {
            World.actions().into_iter().filter(|action| !matches!(action, Action::Heater(_))).collect()
        }

        fn internal(&self, action: &Action) -> bool {
            World.internal(action)
        }

        fn step(&self, state: &Model, action: Action) -> Steps<Model, (), ()> {
            World.step(state, action)
        }

        fn checked_step(&self, state: &Model, action: Action) -> Result<Steps<Model, (), ()>, StepError> {
            World.checked_step(state, action)
        }

        fn postconditions(&self) -> Vec<StepClaim<Self>> {
            // A claim is a plain `fn`, so each one judges through Kettle's claim at its index; the
            // outcome is by reference because `StepPredicate` takes every outcome type that way.
            #[allow(clippy::trivially_copy_pass_by_ref)]
            fn kettles<const I: usize>(_: &Unheated, before: &Model, after: &Model, action: &Action, (): &(), facts: &[()]) -> Verdict {
                World.postconditions()[I].holds(&World, before, action, &((), facts.to_vec(), after.clone()))
            }
            let judges: [StepPredicate<Self>; 2] = [kettles::<0>, kettles::<1>];
            let claims = World.postconditions();
            assert_eq!(claims.len(), judges.len());
            claims
                .into_iter()
                .zip(judges)
                .map(|(claim, judge)| match claim.action {
                    Some(action) => StepClaim::after(claim.name, action, judge),
                    None => StepClaim::new(claim.name, judge),
                })
                .collect()
        }
    }
}
