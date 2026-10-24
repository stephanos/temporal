use crate::activity::activity::{Client, Deadline, Model, Phase, Worker, World};
use umpire_dsl::{Machine, Outcome, Phase as _, Rejection, Verdict};

fn world() -> World { World::new(3) }

fn bound_for(world: World) -> umpire_testing::Bound<World> { umpire_testing::bound(world, 64, None) }

pub(crate) fn lifted(source: &std::path::Path) -> Result<umpire_tla::Module, String> {
    bound_for(world()).lift(source)
}

pub(crate) fn check(folder: &std::path::Path, _source: &std::path::Path, _run: fn() -> umpire_testing::RunResult) -> umpire_testing::RunResult {
    bound_for(world()).check(folder)
}

pub(crate) fn run() -> umpire_testing::RunResult {
    umpire_testing::interpreted(world, None, 64, None)?;
    let result = umpire_testing::Checker::explore(&world(), 64)?;
    let mut files = umpire_testing::render(&result)?;
    files.insert("rows.txt".into(), umpire_testing::render_rows(&result)?);
    let covered = closings()?;
    files.insert("expansion/closings.txt".into(), covered.expansion().into());
    files.insert("coverage.txt".into(), covered.coverage().to_string());
    Ok(files)
}

fn closings() -> Result<umpire_suite::Covered<Model>, umpire_suite::Refusal> {
    use crate::activity::activity::Action;
    use umpire_dsl::{ActionDomain, Gap, Path, Pattern, Waypoint};
    use umpire_suite::*;

    let start = Model { phase: Phase::Scheduled, attempt: 1 };
    let mut alternatives = Vec::new();
    for attempt in 1..=3 {
        for (phase, action) in [
            (Phase::Scheduled, Action::from(Client::Cancel)),
            (Phase::Scheduled, Deadline::ScheduleToStart.into()),
            (Phase::Started, Client::Cancel.into()),
            (Phase::Started, Worker::RespondCompleted.into()),
            (Phase::Started, Worker::RespondFailed { retryable: false }.into()),
            (Phase::Started, Worker::RespondFailed { retryable: true }.into()),
            (Phase::Started, Deadline::StartToClose.into()),
        ] {
            alternatives.push(Pattern::Sequence(vec![
                Pattern::state(Waypoint::Is(start.clone())),
                Pattern::Gap(Gap::Any, None),
                Pattern::state(Waypoint::Is(Model { phase, attempt })),
                Pattern::action(ActionDomain::new(Ok(1), move |_| action.clone()), None, None),
            ]));
        }
    }
    let covered = cover(&Space::with_world(world(), [start]), Cases(3 * 7), Path::pattern(Pattern::Either(alternatives)))?;
    assert_eq!(covered.len(), 21);
    for (index, run) in covered.runs().iter().enumerate() {
        let attempt = u8::try_from(index / 7 + 1).map_err(|_| Refusal::Empty)?;
        let retry = index % 7 == 5 && attempt < 3;
        assert_eq!(run.state().attempt, if retry { attempt + 1 } else { attempt });
        assert_eq!(run.state().phase, match index % 7 {
            0 | 2 => Phase::Canceled,
            1 | 6 => Phase::TimedOut,
            3 => Phase::Completed,
            5 if retry => Phase::Scheduled,
            _ => Phase::Failed,
        });
    }
    Ok(covered)
}

const PHASES: [Phase; 6] = [
    Phase::Scheduled,
    Phase::Started,
    Phase::Completed,
    Phase::Failed,
    Phase::Canceled,
    Phase::TimedOut,
];

fn states() -> impl Iterator<Item = Model> {
    PHASES.into_iter().flat_map(|phase| (1..=3).map(move |attempt| Model { phase, attempt }))
}

fn terminals() -> impl Iterator<Item = Model> { states().filter(|state| state.phase.closed()) }

#[test]
fn closed_cancel_emits_a_rejected_self_loop() {
    let world = world();
    for before in terminals() {
        let phase = before.phase;
        let steps = world.step(&before, Client::Cancel.into());
        assert_eq!(steps.len(), 1, "cancel after {phase:?}");
        assert_eq!(steps[0].0, Outcome::Rejected(Rejection::NotFound));
        assert_eq!(steps[0].2, before);
        assert_eq!(steps[0].1, Vec::<()>::new());
    }
}

#[test]
fn terminal_classification_covers_every_phase() {
    for (phase, terminal) in [
        (Phase::Scheduled, false),
        (Phase::Started, false),
        (Phase::Completed, true),
        (Phase::Failed, true),
        (Phase::Canceled, true),
        (Phase::TimedOut, true),
    ] {
        assert_eq!(phase.closed(), terminal, "{phase:?}");
    }
}

#[test]
fn six_phase_transition_matrix_includes_disabled_actions() {
    use Phase::{Canceled, Completed, Failed, Scheduled, Started, TimedOut};

    let world = World::new(1);
    let actions = world.actions();
    let expected_actions = [
        Client::Cancel.into(),
        Worker::Poll(1).into(),
        Worker::RespondCompleted.into(),
        Worker::RespondFailed { retryable: false }.into(),
        Worker::RespondFailed { retryable: true }.into(),
        Deadline::ScheduleToStart.into(),
        Deadline::StartToClose.into(),
    ];
    assert_eq!(actions, expected_actions);
    let starts = world.starts();
    assert_eq!(starts, [Model { phase: Scheduled, attempt: 1 }]);
    for (action, internal) in actions.iter().zip([false, false, false, false, false, true, true]) {
        assert_eq!(world.internal(action), internal);
    }
    for (phase, expected) in [
        (Scheduled, [Some(Canceled), Some(Started), None, None, None, Some(TimedOut), None]),
        (Started, [Some(Canceled), None, Some(Completed), Some(Failed), Some(Failed), None, Some(TimedOut)]),
        (Completed, [Some(Completed), None, None, None, None, None, None]),
        (Failed, [Some(Failed), None, None, None, None, None, None]),
        (Canceled, [Some(Canceled), None, None, None, None, None, None]),
        (TimedOut, [Some(TimedOut), None, None, None, None, None, None]),
    ] {
        for (action, next) in actions.iter().zip(expected) {
            let steps = world.step(&Model { phase, attempt: 1 }, action.clone());
            let outcome = if phase.closed() {
                Outcome::Rejected(Rejection::NotFound)
            } else {
                Outcome::Accepted
            };
            let expected: Vec<_> =
                next.into_iter().map(|phase| (outcome, Vec::new(), Model { phase, attempt: 1 })).collect();
            assert!(steps.as_slice() == expected, "{action} after {phase:?}");
        }
    }
}

#[test]
fn registered_closure_laws_check_every_phase_action_pair() {
    let world = world();
    let claims = world.postconditions();
    assert_eq!(
        claims.iter().map(|claim| claim.name).collect::<Vec<_>>(),
        ["terminal_remains", "closed_rejects",]
    );
    assert!(claims.iter().all(|claim| claim.action.is_none()));
    let mut checked = 0;
    for before in states() {
        for action in world.actions() {
            let steps = world.step(&before, action.clone());
            let preserved =
                (Outcome::Rejected(Rejection::NotFound), Vec::new(), before.clone());
            let emitted = steps.first().unwrap_or(&preserved);
            for claim in &claims {
                assert_eq!(claim.holds(&world, &before, &action, emitted), Verdict::Holds);
                checked += 1;
            }
        }
    }
    assert_eq!(checked, 6 * 3 * 9 * 2);
}

#[test]
fn hypothetical_reopening_and_terminal_moves_refute_registered_closure_laws() {
    let world = world();
    let claims = world.postconditions();
    for before in terminals() {
        for action in world.actions() {
            for destination in states().filter(|destination| *destination != before) {
                let changed = (
                    Outcome::Rejected(Rejection::NotFound),
                    Vec::new(),
                    destination,
                );
                for claim in &claims {
                    assert_eq!(claim.holds(&world, &before, &action, &changed),
                        if claim.name == "terminal_remains" && changed.2.phase == before.phase { Verdict::Holds } else { Verdict::Fails });
                }
            }
        }
    }
}

#[test]
fn hypothetical_accepted_and_wrong_rejection_refute_registered_closed_rejection() {
    let world = world();
    let claims = world.postconditions();
    let claim = &claims[1];
    assert_eq!(claim.name, "closed_rejects");
    for before in terminals() {
        for action in world.actions() {
            for outcome in [Outcome::Accepted, Outcome::Rejected(Rejection::FailedPrecondition)]
            {
                let emitted = (outcome, Vec::new(), before.clone());
                assert_eq!(claim.holds(&world, &before, &action, &emitted), Verdict::Fails);
            }
        }
    }
}

/// TLC refutes the closure laws the native hypothetical controls refute, on mutants of this
/// source, which are read as the macro reads them rather than compiled: a reopening, an
/// accepted step from closure and a wrong rejection.
#[test]
fn tlc_refutes_closure_law_mutants_of_this_source() -> umpire_testing::TestResult {
    use umpire_tlc::Verdict::{Refuted, Verified};
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let manifest = std::path::Path::new(env!("CARGO_MANIFEST_DIR"));
    let source = std::fs::read_to_string(manifest.join("../feature/activity/model.rs"))?;
    let tlc = umpire_tlc::Tlc::discover(&manifest.join("../../.."))?;
    for (edit, laws) in [
        (None, [Verified, Verified]),
        (
            Some((
                "Worker::Poll(_) if self.phase == Phase::Scheduled =>",
                "Worker::Poll(_) if self.phase == Phase::Scheduled || self.phase == Phase::Canceled =>",
            )),
            [Refuted, Refuted],
        ),
        (
            Some((
                "Deadline::StartToClose if self.phase == Phase::Started => {",
                "_ if self.closed() => stay!(),\n                Deadline::StartToClose if self.phase == Phase::Started => {",
            )),
            [Verified, Refuted],
        ),
        (
            Some((
                "Client::Cancel if self.closed() => reject!(Rejection::NotFound),",
                "Client::Cancel if self.closed() => reject!(Rejection::FailedPrecondition),",
            )),
            [Verified, Refuted],
        ),
    ] {
        let mutant = match edit {
            Some((from, to)) => {
                assert_eq!(source.matches(from).count(), 1, "{from}");
                source.replace(from, to)
            }
            None => source.clone(),
        };
        let module = bound_for(world()).lift_source("model.rs", &mutant)?;
        let result = tlc.check_module(&module)?;
        for (law, verdict) in ["terminal_remains", "closed_rejects"].into_iter().zip(laws) {
            let actual = result.properties[&format!("Post_{law}")].verdict;
            assert_eq!(actual, verdict, "{law} after {edit:?}");
        }
    }
    Ok(())
}

#[test]
fn retry_policy_caps_and_stale_delivery() {
    use crate::activity::activity::World;
    for maximum in [1, 3] {
        let world = World::new(maximum);
        assert_eq!(world.starts(), [Model { phase: Phase::Scheduled, attempt: 1 }]);
        for attempt in 1..=maximum {
            let scheduled = Model { phase: Phase::Scheduled, attempt };
            let poll = world.step(&scheduled, Worker::Poll(attempt).into());
            assert_eq!(poll.as_slice(), &[(Outcome::Accepted, vec![], Model { phase: Phase::Started, attempt })]);
            for expected in (1..=maximum).filter(|expected| *expected != attempt) {
                assert_eq!(world.step(&scheduled, Worker::Poll(expected).into()).as_slice(), &
                    [(Outcome::Rejected(Rejection::FailedPrecondition), vec![], scheduled.clone())]);
            }
            for retryable in [false, true] {
                let retry = retryable && attempt < maximum;
                assert_eq!(world.step(&poll[0].2, Worker::RespondFailed { retryable }.into()).as_slice(), &
                    [(Outcome::Accepted, vec![], Model {
                        phase: if retry { Phase::Scheduled } else { Phase::Failed },
                        attempt: if retry { attempt + 1 } else { attempt },
                    })]);
            }
        }
    }
}

#[path = "compatibility/checkpoint.rs"]
mod compatibility;

#[test]
#[allow(clippy::print_stderr)]
fn actual_tlc_agrees_and_replays_retry_payload_counterexamples() -> umpire_testing::TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        eprintln!("activity retry TLC control skipped; set UMPIRE_BACKENDS=require");
        return Ok(());
    }
    let folder = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../feature/activity");
    let source = std::fs::read_to_string(folder.join("model.rs"))?;
    let negative = format!("{source}\numpire_testing::fixture!(World = World::new(3), 64, Claim::new(\"never_retried\", |_, state: &Model| (state.attempt == 1).into()));");
    for (maximum, states, replayed) in [(1, 6, 0), (3, 18, 1)] {
        let bound = umpire_testing::bound(World::new(maximum), 64, Some(umpire_dsl::Claim::new("never_retried", |_, state: &Model| (state.attempt == 1).into())));
        let module = bound.lift_source("model.rs", &negative)?;
        let files = bound.check_module(&folder, &module)?;
        let report = &files["tlc.txt"];
        assert!(report.contains(&format!("states: {states}; native: {states}; agreement: yes")), "{report}");
        assert!(report.contains(&format!("counterexamples replayed: {replayed}")), "{report}");
        eprintln!("activity maximum_attempts={maximum}: {report}");
    }
    Ok(())
}

#[test]
fn actual_tlc_rejected_outcome_counterexample_replays_in_the_activity_model() -> umpire_testing::TestResult {
    use umpire_tlc::Value;
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") { return Ok(()); }
    let manifest = std::path::Path::new(env!("CARGO_MANIFEST_DIR"));
    let source = std::fs::read_to_string(manifest.join("../feature/activity/model.rs"))?;
    let negative = source.replace("        #[postcondition]\n        fn terminal_remains", "        #[postcondition]\n        fn accepts_every_step(&self, _before: &Self, outcome: &Outcome) {\n            check!(*outcome == Outcome::Accepted);\n        }\n\n        #[postcondition]\n        fn terminal_remains");
    assert_ne!(negative, source);
    let world = World::new(3);
    let module = bound_for(world.clone()).lift_source("model.rs", &negative)?;
    let result = umpire_tlc::Tlc::discover(&manifest.join("../../.."))?.check_module(&module)?;
    let property = &result.properties["Post_accepts_every_step"];
    assert_eq!(property.verdict, umpire_tlc::Verdict::Refuted);
    let trace = property.trace.as_ref().ok_or("missing outcome counterexample")?;
    let mut current = world.starts()[0].clone();
    let mut rejected = false;
    for pair in trace.states.windows(2) {
        let [before, after] = pair else { return Err("missing trace pair".into()); };
        if before.observers["__umpire_edge_tick"] == after.observers["__umpire_edge_tick"] {
            assert_eq!(before.state, after.state);
            continue;
        }
        assert_eq!(before.state["attempt"], Value::Int(i128::from(current.attempt)));
        assert_eq!(before.state["phase"], Value::Text(format!("Phase::{:?}", current.phase)));
        let Value::Int(ordinal) = after.observers["__umpire_input"] else { return Err("missing input".into()); };
        let action = world.actions().get(usize::try_from(ordinal)?).ok_or("invalid action ordinal")?.clone();
        let (name, fields) = umpire_dsl::ActionValues::<World>::action_value(&action);
        assert_eq!(after.observers["__umpire_action"], Value::Text(name.into()));
        assert_eq!(module.metadata.catalog[usize::try_from(ordinal)?].action, name);
        let native_fields: Vec<_> = fields.into_iter().map(|(name, value)| {
            let literal = match value {
                umpire_dsl::Value::Int(value) => umpire_tla::Literal::Int(value),
                umpire_dsl::Value::Bool(value) => umpire_tla::Literal::Bool(value),
                other => panic!("unexpected activity action field: {other:?}"),
            };
            (name.into(), literal)
        }).collect();
        assert_eq!(module.metadata.catalog[usize::try_from(ordinal)?].fields, native_fields);
        let Value::Int(choice) = after.observers["__umpire_choice"] else { return Err("missing choice".into()); };
        let steps = world.checked_step(&current, action)?;
        let (outcome, facts, destination) = steps.get(usize::try_from(choice)?).ok_or("missing native choice")?;
        assert_eq!(after.state["attempt"], Value::Int(i128::from(destination.attempt)));
        assert_eq!(after.state["phase"], Value::Text(format!("Phase::{:?}", destination.phase)));
        let expected = match outcome {
            Outcome::Accepted => "Outcome::Accepted",
            Outcome::Rejected(Rejection::NotFound) => { rejected = true; "Outcome::Rejected(NotFound)" },
            Outcome::Rejected(Rejection::FailedPrecondition) => { rejected = true; "Outcome::Rejected(FailedPrecondition)" },
            Outcome::Rejected(_) => return Err("unexpected activity rejection".into()),
        };
        assert_eq!(after.observers["__umpire_outcome"], Value::Text(expected.into()));
        assert_eq!(facts.as_slice(), []);
        assert_eq!(after.observers["__umpire_facts"], Value::Seq(vec![]));
        current = destination.clone();
    }
    assert!(rejected);
    Ok(())
}
