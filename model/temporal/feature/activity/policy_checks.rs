use umpire_testing::{replace_once, replace_in};
use super::{bound_for, checked_world, configured};
use crate::activity::activity::{
    Action, Client, Deadline, ExpiredDeadline, Model, Phase, Remaining, Ticks, Worker, World,
};
use umpire_dsl::{Machine, Outcome, Rejection, __lifted::ConcreteOf};

type TestResult = umpire_testing::TestResult;

fn accepted(world: &World, before: &Model, action: Action) -> Model {
    let steps = world.checked_step(before, action).unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(steps.len(), 1);
    assert_eq!(steps[0].0, Outcome::Accepted);
    assert_eq!(steps[0].1, Vec::<ExpiredDeadline>::new());
    steps[0].2.clone()
}

fn attempt(world: &World, number: u8, started: bool, pending: bool) -> Model {
    let mut state = world.starts().remove(0);
    for current in 1..=number {
        if current == number && !started {
            assert!(!pending);
            return state;
        }
        state = accepted(world, &state, Worker::Poll(current).into());
        if current < number {
            state = accepted(world, &state, Worker::RespondFailed { retryable: true }.into());
        }
    }
    if pending {
        state = accepted(world, &state, Client::Cancel.into());
    }
    assert_eq!(state.attempt, number);
    state
}

fn cleared(before: &Model, phase: Phase) -> Model {
    Model {
        phase,
        attempt: before.attempt,
        cancel_requested: false,
        schedule_to_start_remaining: None,
        start_to_close_remaining: None,
        schedule_to_close_remaining: None,
        heartbeat_remaining: None,
    }
}

fn assert_closed(world: &World, before: &Model, action: Action, phase: Phase) {
    assert_eq!(
        world.checked_step(before, action).unwrap_or_else(|error| panic!("{error}")).into_vec(),
        [(Outcome::Accepted, vec![], cleared(before, phase))]
    );
}

#[test]
fn configured_deadlines_reject_zero_and_binding_bypass() {
    for (index, field) in
        ["schedule_to_start", "start_to_close", "schedule_to_close", "heartbeat_timeout"]
            .into_iter()
            .enumerate()
    {
        for invalid in [0, 4] {
            let mut timeouts = [None; 4];
            timeouts[index] = Some(invalid);
            assert_eq!(checked_world(3, true, timeouts).err(), Some(field));
            let mut raw = [None; 4];
            raw[index] = Some(Ticks(invalid));
            let world = World::new(3, raw[0], raw[1], raw[2], raw[3], true);
            let error = bound_for(world)
                .lift_source("model.rs", SOURCE)
                .err()
                .unwrap_or_else(|| panic!("invalid {field} admitted"));
            assert!(error.contains(field), "{error}");
        }
    }
    for timeout in [None, Some(1), Some(3)] {
        assert!(checked_world(3, true, [timeout; 4]).is_ok());
    }
}

#[test]
fn cancellation_and_worker_races_have_exact_reachable_results() {
    for maximum in [1, 3] {
        for asynchronous in [false, true] {
            let world = configured(maximum, asynchronous, [Some(3); 4]);
            for number in 1..=maximum {
                let scheduled = attempt(&world, number, false, false);
                assert_closed(&world, &scheduled, Client::Cancel.into(), Phase::Canceled);
                let started = attempt(&world, number, true, false);
                if asynchronous {
                    let pending = accepted(&world, &started, Client::Cancel.into());
                    let mut expected = started.clone();
                    expected.cancel_requested = true;
                    assert_eq!(pending, expected);
                    assert_eq!(
                        world
                            .checked_step(&pending, Client::Cancel.into())
                            .unwrap_or_default()
                            .into_vec(),
                        [(
                            Outcome::Rejected(Rejection::FailedPrecondition),
                            vec![],
                            pending.clone()
                        )]
                    );
                    assert_closed(
                        &world,
                        &pending,
                        Worker::RespondCanceled(number).into(),
                        Phase::Canceled,
                    );
                    assert_closed(
                        &world,
                        &pending,
                        Worker::RespondCompleted.into(),
                        Phase::Completed,
                    );
                    for (retryable, phase) in [(false, Phase::Failed), (true, Phase::Canceled)] {
                        assert_closed(
                            &world,
                            &pending,
                            Worker::RespondFailed { retryable }.into(),
                            phase,
                        );
                    }
                    let ticked =
                        world.checked_step(&pending, Deadline::Tick.into()).unwrap_or_default();
                    assert_eq!(ticked.len(), 1);
                    let consumed = &ticked[0].2;
                    assert_eq!(consumed.phase, Phase::Started);
                    let heartbeat = accepted(&world, consumed, Worker::Heartbeat.into());
                    let mut expected = consumed.clone();
                    expected.heartbeat_remaining = Some(Remaining { ticks: 3 });
                    assert_eq!(heartbeat, expected);
                } else {
                    assert_closed(&world, &started, Client::Cancel.into(), Phase::Canceled);
                }
                assert_closed(&world, &started, Worker::RespondCompleted.into(), Phase::Completed);
                assert_closed(
                    &world,
                    &started,
                    Worker::RespondFailed { retryable: false }.into(),
                    Phase::Failed,
                );
                let failed =
                    accepted(&world, &started, Worker::RespondFailed { retryable: true }.into());
                if number < maximum {
                    let mut expected = scheduled;
                    expected.attempt += 1;
                    assert_eq!(failed, expected);
                } else {
                    assert_eq!(failed, cleared(&started, Phase::Failed));
                }
            }
        }
    }
}

#[test]
fn current_and_stale_worker_tokens_preserve_exact_availability() -> TestResult {
    let world = configured(3, true, [None; 4]);
    for number in 1..=3 {
        for (started, pending) in [(false, false), (true, false), (true, true)] {
            let before = attempt(&world, number, started, pending);
            for token in 1..=3 {
                let ack =
                    world.checked_step(&before, Worker::RespondCanceled(token).into())?.into_vec();
                let poll = world.checked_step(&before, Worker::Poll(token).into())?.into_vec();
                if token == number {
                    assert_eq!(
                        ack,
                        if pending {
                            vec![(Outcome::Accepted, vec![], cleared(&before, Phase::Canceled))]
                        } else {
                            vec![]
                        }
                    );
                    assert_eq!(
                        poll,
                        if started {
                            vec![]
                        } else {
                            let mut destination = before.clone();
                            destination.phase = Phase::Started;
                            vec![(Outcome::Accepted, vec![], destination)]
                        }
                    );
                } else {
                    assert_eq!(
                        ack,
                        [(Outcome::Rejected(Rejection::NotFound), vec![], before.clone())]
                    );
                    assert_eq!(
                        poll,
                        [(
                            Outcome::Rejected(Rejection::FailedPrecondition),
                            vec![],
                            before.clone()
                        )]
                    );
                }
            }
            assert!(world.checked_step(&before, Deadline::Tick.into())?.is_empty());
        }
        let started = attempt(&world, number, true, false);
        let pending = attempt(&world, number, true, true);
        for closed in [
            accepted(&world, &started, Worker::RespondCompleted.into()),
            accepted(&world, &started, Worker::RespondFailed { retryable: false }.into()),
            accepted(&world, &pending, Worker::RespondCanceled(number).into()),
        ] {
            for action in world.actions() {
                let actual = world.checked_step(&closed, action.clone())?.into_vec();
                let expected = if action == Client::Cancel.into() {
                    vec![(Outcome::Rejected(Rejection::NotFound), vec![], closed.clone())]
                } else {
                    vec![]
                };
                assert_eq!(actual, expected, "{action} after {closed:?}");
            }
        }
    }
    Ok(())
}

#[test]
fn each_deadline_expires_once_with_its_exact_fact() -> TestResult {
    for ticks in [1, 3] {
        for (index, fact) in [
            ExpiredDeadline::ScheduleToStart,
            ExpiredDeadline::StartToClose,
            ExpiredDeadline::ScheduleToClose,
            ExpiredDeadline::Heartbeat,
        ]
        .into_iter()
        .enumerate()
        {
            for pending in [false, true] {
                if index == 0 && pending {
                    continue;
                }
                let mut timeouts = [None; 4];
                timeouts[index] = Some(ticks);
                let world = configured(3, true, timeouts);
                let mut state = attempt(&world, 1, index != 0, pending);
                for remaining in (0..ticks).rev() {
                    let steps = world.checked_step(&state, Deadline::Tick.into())?;
                    assert_eq!(steps.len(), 1);
                    assert_eq!(steps[0].0, Outcome::Accepted);
                    if remaining == 0 {
                        let phase = if pending && matches!(index, 1 | 3) {
                            Phase::Canceled
                        } else {
                            Phase::TimedOut
                        };
                        assert_eq!(steps[0].1, [fact]);
                        assert_eq!(steps[0].2, cleared(&state, phase));
                    } else {
                        assert_eq!(steps[0].1, Vec::<ExpiredDeadline>::new());
                        let mut expected = state.clone();
                        match index {
                            0 => {
                                expected.schedule_to_start_remaining =
                                    Some(Remaining { ticks: remaining });
                            }
                            1 => {
                                expected.start_to_close_remaining =
                                    Some(Remaining { ticks: remaining });
                            }
                            2 => {
                                expected.schedule_to_close_remaining =
                                    Some(Remaining { ticks: remaining });
                            }
                            _ => {
                                expected.heartbeat_remaining = Some(Remaining { ticks: remaining });
                            }
                        }
                        assert_eq!(steps[0].2, expected);
                    }
                    state = steps[0].2.clone();
                }
                assert!(world.checked_step(&state, Deadline::Tick.into())?.is_empty());
            }
        }
    }
    Ok(())
}

#[test]
fn simultaneous_expiry_obeys_global_then_attempt_then_heartbeat_priority() -> TestResult {
    for (timeouts, started, fact) in [
        ([Some(1), None, Some(1), None], false, ExpiredDeadline::ScheduleToClose),
        ([None, Some(1), Some(1), Some(1)], true, ExpiredDeadline::ScheduleToClose),
        ([None, Some(1), None, Some(1)], true, ExpiredDeadline::StartToClose),
    ] {
        for pending in [false, true] {
            if pending && !started {
                continue;
            }
            let world = configured(3, true, timeouts);
            let before = attempt(&world, 1, started, pending);
            let phase = if pending && fact == ExpiredDeadline::StartToClose {
                Phase::Canceled
            } else {
                Phase::TimedOut
            };
            assert_eq!(
                world.checked_step(&before, Deadline::Tick.into())?.into_vec(),
                [(Outcome::Accepted, vec![fact], cleared(&before, phase))]
            );
        }
    }
    Ok(())
}

#[test]
fn poll_retry_and_heartbeat_reset_only_their_applicable_budgets() -> TestResult {
    let world = configured(3, true, [Some(3); 4]);
    let start = world.starts().remove(0);
    assert_eq!(start.schedule_to_start_remaining, Some(Remaining { ticks: 3 }));
    assert_eq!(start.schedule_to_close_remaining, Some(Remaining { ticks: 3 }));
    assert_eq!(start.start_to_close_remaining, None);
    assert_eq!(start.heartbeat_remaining, None);
    let steps = world.checked_step(&start, Deadline::Tick.into())?;
    assert_eq!(steps.len(), 1);
    let scheduled = &steps[0].2;
    let started = accepted(&world, scheduled, Worker::Poll(1).into());
    assert_eq!(started.schedule_to_start_remaining, None);
    assert_eq!(started.schedule_to_close_remaining, Some(Remaining { ticks: 2 }));
    assert_eq!(started.start_to_close_remaining, Some(Remaining { ticks: 3 }));
    assert_eq!(started.heartbeat_remaining, Some(Remaining { ticks: 3 }));
    let steps = world.checked_step(&started, Deadline::Tick.into())?;
    assert_eq!(steps.len(), 1);
    let consumed = &steps[0].2;
    let heartbeat = accepted(&world, consumed, Worker::Heartbeat.into());
    let mut expected = consumed.clone();
    expected.heartbeat_remaining = Some(Remaining { ticks: 3 });
    assert_eq!(heartbeat, expected);
    let retried = accepted(&world, &heartbeat, Worker::RespondFailed { retryable: true }.into());
    assert_eq!(retried.phase, Phase::Scheduled);
    assert_eq!(retried.attempt, 2);
    assert_eq!(retried.schedule_to_start_remaining, Some(Remaining { ticks: 3 }));
    assert_eq!(retried.schedule_to_close_remaining, Some(Remaining { ticks: 1 }));
    assert_eq!(retried.start_to_close_remaining, None);
    assert_eq!(retried.heartbeat_remaining, None);
    let repolled = accepted(&world, &retried, Worker::Poll(2).into());
    assert_eq!(repolled.schedule_to_close_remaining, Some(Remaining { ticks: 1 }));
    assert_eq!(
        world.checked_step(&repolled, Deadline::Tick.into())?.into_vec(),
        [(
            Outcome::Accepted,
            vec![ExpiredDeadline::ScheduleToClose],
            cleared(&repolled, Phase::TimedOut)
        )]
    );
    Ok(())
}

const SOURCE: &str = include_str!("model.rs");

fn folder() -> std::path::PathBuf {
    std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../feature/activity")
}


fn extra_method(method: &str) -> String {
    if method.contains("#[postcondition") {
        replace_once(SOURCE, "        #[assumption]", &format!("{method}\n        #[assumption]"))
    } else {
        replace_once(SOURCE, "\n    }\n}", &format!("\n{method}\n    }}\n}}"))
    }
}

fn check_trace_state(native: &Model, state: &umpire_tlc::State) {
    use umpire_tlc::Value;
    let remaining =
        |value: Option<Remaining>| Value::Int(value.map_or(0, |value| i128::from(value.ticks) + 1));
    assert_eq!(state.len(), 7);
    assert_eq!(state["phase"], Value::Text(format!("Phase::{:?}", native.phase)));
    assert_eq!(state["attempt"], Value::Int(i128::from(native.attempt)));
    assert_eq!(state["cancel_requested"], Value::Bool(native.cancel_requested));
    for (field, expected) in [
        ("schedule_to_start_remaining", native.schedule_to_start_remaining),
        ("start_to_close_remaining", native.start_to_close_remaining),
        ("schedule_to_close_remaining", native.schedule_to_close_remaining),
        ("heartbeat_remaining", native.heartbeat_remaining),
    ] {
        assert_eq!(state[field], remaining(expected), "{field}");
    }
}

type Replayed = (Model, Action, Outcome, Vec<ExpiredDeadline>, Model);

fn replay(
    world: &World,
    module: &umpire_tla::Module,
    trace: &umpire_tlc::Trace,
) -> Result<Vec<Replayed>, Box<dyn std::error::Error>> {
    use umpire_tlc::Value;
    let mut current = world.starts().remove(0);
    check_trace_state(&current, &trace.states.first().ok_or("empty trace")?.state);
    let mut replayed = Vec::new();
    for pair in trace.states.windows(2) {
        let before = &pair[0];
        let after = &pair[1];
        check_trace_state(&current, &before.state);
        if before.observers["__umpire_edge_tick"] == after.observers["__umpire_edge_tick"] {
            assert_eq!(before.state, after.state);
            continue;
        }
        let Value::Int(ordinal) = after.observers["__umpire_input"] else {
            return Err("missing input ordinal".into());
        };
        let ordinal = usize::try_from(ordinal)?;
        let action = world.actions().get(ordinal).ok_or("invalid action ordinal")?.clone();
        let identity = umpire_tla::Concrete::of::<World>(&action);
        assert_eq!(after.observers["__umpire_action"], Value::Text(identity.action.clone()));
        assert_eq!(module.metadata.catalog[ordinal], identity);
        let Value::Int(choice) = after.observers["__umpire_choice"] else {
            return Err("missing choice".into());
        };
        let steps = world.checked_step(&current, action.clone())?;
        let (outcome, facts, destination) =
            steps.get(usize::try_from(choice)?).ok_or("unavailable native edge")?;
        check_trace_state(destination, &after.state);
        let expected_outcome = match outcome {
            Outcome::Accepted => "Outcome::Accepted",
            Outcome::Rejected(Rejection::NotFound) => "Outcome::Rejected(NotFound)",
            Outcome::Rejected(Rejection::FailedPrecondition) => {
                "Outcome::Rejected(FailedPrecondition)"
            }
            other @ Outcome::Rejected(_) => panic!("unexpected outcome: {other:?}"),
        };
        assert_eq!(after.observers["__umpire_outcome"], Value::Text(expected_outcome.into()));
        let expected_facts =
            facts.iter().map(|fact| Value::Text(format!("ExpiredDeadline::{fact:?}"))).collect();
        if after.observers["__umpire_facts"] != Value::Seq(expected_facts) {
            return Err("outcome/facts differ from exact native edge".into());
        }
        replayed.push((current, action, *outcome, facts.clone(), destination.clone()));
        current = destination.clone();
    }
    Ok(replayed)
}

#[test]
#[allow(clippy::print_stderr)]
fn required_tlc_replays_both_pending_failure_payloads_at_every_attempt() -> TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../..");
    let tlc = umpire_tlc::Tlc::discover(&root)?;
    for maximum in [1, 3] {
        for number in 1..=maximum {
            for retryable in [false, true] {
                let world = configured(maximum, true, [None; 4]);
                let method = format!(
                    r"
        #[progress]
        fn false_pending_failure(&self) -> LeadsTo {{
            path![self.phase == Phase::Started && self.cancel_requested && self.attempt == {number}, ..=1, self.phase == Phase::Completed]
                .assuming(only([Worker::RespondFailed {{ retryable: {retryable} }}]))
        }}
"
                );
                let source = extra_method(&method);
                let module = bound_for(world.clone()).lift_source("model.rs", &source)?;
                for (name, value) in [
                    ("pending_retryable_failure_cancels", true),
                    ("pending_nonretryable_failure_fails", false),
                ] {
                    let action = Worker::RespondFailed { retryable: value }.into();
                    let ordinal = world
                        .actions()
                        .iter()
                        .position(|candidate| *candidate == action)
                        .ok_or("missing concrete failure")?;
                    let claim = module
                        .metadata
                        .progress
                        .iter()
                        .find(|claim| claim.name == name)
                        .ok_or("missing pending failure claim")?;
                    assert_eq!(module.metadata.catalog[ordinal], umpire_tla::Concrete::of::<World>(&action));
                    assert_eq!(claim.allowed_inputs, [ordinal]);
                    assert_eq!(claim.fair_inputs, []);
                }
                let module = only_property(module, "Progress_false_pending_failure");
                let result = tlc.check_module(&module)?;
                let property = &result.properties["Progress_false_pending_failure"];
                assert_eq!(property.verdict, umpire_tlc::Verdict::Refuted);
                eprintln!(
                    "activity pending failure maximum={maximum} attempt={number} retryable={retryable}: {:#?}",
                    property.trace
                );
                let edges = replay(
                    &world,
                    &module,
                    property.trace.as_ref().ok_or("missing failure trace")?,
                )?;
                let edge = edges
                    .iter()
                    .find(|(before, action, _, _, _)| {
                        before.cancel_requested
                            && before.attempt == number
                            && *action == Worker::RespondFailed { retryable }.into()
                    })
                    .ok_or("missing concrete pending failure edge")?;
                assert_eq!(edge.2, Outcome::Accepted);
                assert_eq!(edge.3, Vec::<ExpiredDeadline>::new());
                assert_eq!(
                    edge.4,
                    cleared(&edge.0, if retryable { Phase::Canceled } else { Phase::Failed })
                );
            }
        }
    }
    Ok(())
}

#[test]
#[allow(clippy::print_stderr)]
fn required_tlc_replays_both_canceled_attempt_expiry_facts() -> TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../..");
    let tlc = umpire_tlc::Tlc::discover(&root)?;
    let source = extra_method(
        r"
        #[postcondition(after = Deadline::Tick)]
        fn false_pending_tick(&self, before: &Self) {
            check!(!before.cancel_requested || self.phase != Phase::Canceled);
        }
",
    );
    for (index, fact) in [(1, ExpiredDeadline::StartToClose), (3, ExpiredDeadline::Heartbeat)] {
        let mut timeouts = [None; 4];
        timeouts[index] = Some(1);
        let world = configured(3, true, timeouts);
        let module = only_property(
            bound_for(world.clone()).lift_source("model.rs", &source)?,
            "Post_false_pending_tick",
        );
        let result = tlc.check_module(&module)?;
        let property = &result.properties["Post_false_pending_tick"];
        assert_eq!(property.verdict, umpire_tlc::Verdict::Refuted);
        eprintln!(
            "activity pending expiry timeout index={index}, fact={fact:?}: {:#?}",
            property.trace
        );
        let edges =
            replay(&world, &module, property.trace.as_ref().ok_or("missing expiry trace")?)?;
        let edge = edges
            .iter()
            .find(|(before, action, _, facts, _)| {
                before.cancel_requested && *action == Deadline::Tick.into() && facts == &[fact]
            })
            .ok_or("missing exact canceled expiry edge")?;
        assert_eq!(edge.2, Outcome::Accepted);
        assert_eq!(edge.4, cleared(&edge.0, Phase::Canceled));
        let trace = property.trace.as_ref().ok_or("missing expiry trace")?;
        let exact = umpire_tlc::Value::Text(format!("ExpiredDeadline::{fact:?}"));
        for facts in [vec![], vec![exact.clone(), exact]] {
            let mut forged = trace.clone();
            let index = forged
                .states
                .windows(2)
                .position(|pair| {
                    pair[0].observers["__umpire_edge_tick"]
                        != pair[1].observers["__umpire_edge_tick"]
                        && pair[0].state["cancel_requested"] == umpire_tlc::Value::Bool(true)
                        && pair[1].observers.get("__umpire_action")
                            == Some(&umpire_tlc::Value::Text("Deadline::Tick".into()))
                })
                .ok_or("missing same pending Tick observer")?
                + 1;
            forged.states[index]
                .observers
                .insert("__umpire_facts".into(), umpire_tlc::Value::Seq(facts));
            let error =
                replay(&world, &module, &forged).err().ok_or("forged exact-edge facts replayed")?;
            assert!(error.to_string().contains("outcome/facts differ"), "{error}");
        }
    }
    Ok(())
}


#[test]
#[allow(clippy::print_stderr)]
fn required_tlc_refutes_source_only_boundary_policy_mutants() -> TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../..");
    let tlc = umpire_tlc::Tlc::discover(&root)?;
    let pending_at_cap = replace_once(
        SOURCE,
        "if retryable && self.cancel_requested {",
        "if retryable && self.cancel_requested && self.attempt < world.maximum_attempts {",
    );
    let wrong_failure_scope = replace_in(
        SOURCE,
        "fn pending_retryable_failure_cancels(",
        "fn pending_nonretryable_failure_fails(",
        "only([Worker::RespondFailed { retryable: true }])",
        "only([Worker::RespondFailed { retryable: false }])",
    );
    let nonretryable_loses = replace_once(
        SOURCE,
        "if retryable && self.cancel_requested {",
        "if self.cancel_requested {",
    );
    let heartbeat_closes = replace_in(
        SOURCE,
        "Worker::Heartbeat if self.phase == Phase::Started =>",
        "Worker::RespondCanceled(expected)",
        "stay!();",
        "stay!(); self.phase = Phase::Canceled; self.clear_terminal();",
    );
    let global_loses = replace_in(
        SOURCE,
        "self.decrement();",
        "Some(ExpiredDeadline::ScheduleToClose)",
        "self.phase = Phase::TimedOut;",
        "if self.cancel_requested { self.phase = Phase::Canceled; } else { self.phase = Phase::TimedOut; }",
    );
    for (source, world, property) in [
        (
            pending_at_cap,
            configured(1, true, [None; 4]),
            "Progress_pending_retryable_failure_cancels",
        ),
        (
            wrong_failure_scope,
            configured(3, true, [None; 4]),
            "Progress_pending_retryable_failure_cancels",
        ),
        (
            nonretryable_loses,
            configured(3, true, [None; 4]),
            "Progress_pending_nonretryable_failure_fails",
        ),
        (
            heartbeat_closes,
            configured(3, true, [None; 4]),
            "Post_heartbeat_preserves_attempt_and_global",
        ),
        (
            global_loses,
            configured(3, true, [None, Some(1), Some(1), Some(1)]),
            "Post_tick_priority",
        ),
    ] {
        let module = only_property(bound_for(world).lift_source("model.rs", &source)?, property);
        let result = tlc.check_module(&module)?;
        assert_eq!(result.properties[property].verdict, umpire_tlc::Verdict::Refuted, "{property}");
        assert!(result.properties[property].trace.is_some(), "{property}");
        eprintln!(
            "activity source policy mutant {property}: {:#?}",
            result.properties[property].trace
        );
    }
    Ok(())
}

#[test]
fn required_tlc_fact_only_mutants_fail_replay_on_the_same_pending_tick() -> TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let witness = extra_method(
        r"
        #[postcondition(after = Deadline::Tick)]
        fn false_pending_tick(&self, before: &Self) {
            check!(!before.cancel_requested || self.phase != Phase::Canceled);
        }
",
    );
    for (index, original, replacement) in [
        (1, "Some(ExpiredDeadline::StartToClose)", "Some(ExpiredDeadline::Heartbeat)"),
        (3, "Some(ExpiredDeadline::Heartbeat)", "Some(ExpiredDeadline::StartToClose)"),
        (1, "Some(ExpiredDeadline::StartToClose)", "None"),
    ] {
        let mut timeouts = [None; 4];
        timeouts[index] = Some(1);
        let world = configured(3, true, timeouts);
        let source = replace_once(&witness, original, replacement);
        rejects_fact_mutant(&world, &source, "Post_false_pending_tick")?;
    }
    Ok(())
}

#[test]
fn required_tlc_nonexpiring_tick_fact_leak_fails_exact_replay() -> TestResult {
    if std::env::var("UMPIRE_BACKENDS").as_deref() != Ok("require") {
        return Ok(());
    }
    let witness = extra_method(
        r"
        #[postcondition(after = Deadline::Tick)]
        fn false_nonexpiring_tick(&self, before: &Self) {
            check!(before.schedule_to_close_remaining.unwrap_or(Remaining { ticks: 0 }).ticks <= 1 || self.phase != before.phase);
        }
",
    );
    let begin = witness.find("fn deadline(").ok_or("missing deadline handler")?;
    let end =
        begin + witness[begin..].find("fn clear_terminal(").ok_or("missing terminal helper")?;
    let tail = witness[begin..end]
        .match_indices("else {")
        .find_map(|(position, _)| {
            let start = begin + position + "else {".len();
            let end = start + witness[start..].find('}')?;
            (witness[start..end].trim() == "None").then_some((start, end))
        })
        .ok_or("missing nonexpiring fact leaf")?;
    let source =
        format!("{} Some(ExpiredDeadline::Heartbeat) {}", &witness[..tail.0], &witness[tail.1..]);
    let world = configured(1, true, [None, None, Some(3), None]);
    rejects_fact_mutant(&world, &source, "Post_false_nonexpiring_tick")
}

fn rejects_fact_mutant(world: &World, source: &str, property: &str) -> TestResult {
    let module = only_property(bound_for(world.clone()).lift_source("model.rs", source)?, property);
    let graph = umpire_testing::Checker::explore(world, 4096)?;
    let native = umpire_testing::render(&graph)?;
    let error = umpire_testing::backend::check_exploration(
        world,
        &world.starts(),
        &graph,
        &module,
        &folder(),
        &native,
        Some(&umpire_tla::Concrete::of::<World>),
    )
    .err()
    .ok_or("fact-only mutant agreed")?;
    assert!(error.to_string().contains("outcome/facts differ"), "{error}");
    Ok(())
}

fn only_property(mut module: umpire_tla::Module, operator: &str) -> umpire_tla::Module {
    module.metadata.invariants.retain(|claim| claim.operator == operator);
    module.metadata.postconditions.retain(|claim| claim.operator == operator);
    module.metadata.negative.retain(|claim| claim.operator == operator);
    module.metadata.progress.retain(|claim| claim.operator == operator);
    module.cfg = module
        .cfg
        .lines()
        .filter(|line| !line.starts_with("INVARIANT") && !line.starts_with("PROPERTY"))
        .flat_map(|line| [line, "\n"])
        .collect();
    for claim in &module.metadata.invariants {
        module.cfg.push_str("INVARIANT ");
        module.cfg.push_str(&claim.operator);
        module.cfg.push('\n');
    }
    for claim in &module.metadata.postconditions {
        module.cfg.push_str("PROPERTY ");
        module.cfg.push_str(&claim.operator);
        module.cfg.push('\n');
    }
    module
}

#[test]
#[allow(clippy::print_stderr)]
fn selected_deadline_worlds_agree_compiled_interpreted_and_required_tlc() -> TestResult {
    let mut selections = vec![(1, false, [None; 4]), (3, true, [None; 4])];
    for (maximum, asynchronous, ticks) in [(1, false, 1), (3, true, 3)] {
        for index in 0..4 {
            let mut deadlines = [None; 4];
            deadlines[index] = Some(ticks);
            selections.push((maximum, asynchronous, deadlines));
        }
    }
    selections.extend([
        (1, false, [Some(1), None, Some(1), None]),
        (3, true, [None, Some(1), Some(1), Some(1)]),
        (3, true, [None, Some(1), None, Some(1)]),
    ]);
    for (maximum, asynchronous, deadlines) in selections {
        let world = configured(maximum, asynchronous, deadlines);
        let graph = umpire_testing::Checker::explore(&world, 4096)?;
        assert!(!graph.counts.cut);
        assert!(graph.counts.reached > 0 && graph.counts.emitted > 0);
        let report = graph.verify_all()?;
        assert!(!report.claims.is_empty());
        assert_eq!(report.outcome(), umpire_dsl::CheckOutcome::Verified, "{report}");
        for name in [
            "state_bounds",
            "deadline_arming",
            "pending_is_cooperative",
            "tick_priority",
            "pending_retryable_failure_cancels",
            "pending_nonretryable_failure_fails",
        ] {
            assert!(
                report.claims.iter().any(|claim| claim.name == name
                    && claim.outcome == umpire_dsl::CheckOutcome::Verified),
                "{name}: {report}"
            );
        }
        let agreement = umpire_testing::interpreted(|| world.clone(), None, 4096, None)?;
        assert_eq!(
            (agreement.states, agreement.edges),
            (graph.counts.reached, graph.counts.emitted)
        );
        eprintln!(
            "activity policy max={maximum} cooperative={asynchronous} timeouts={deadlines:?}: states={} edges={} cut={}",
            graph.counts.reached, graph.counts.emitted, graph.counts.cut
        );
        if std::env::var("UMPIRE_BACKENDS").as_deref() == Ok("require") {
            let module = bound_for(world.clone()).lift_source("model.rs", SOURCE)?;
            let files = bound_for(world).check_module(&folder(), &module)?;
            eprintln!(
                "activity policy max={maximum} cooperative={asynchronous} timeouts={deadlines:?}: {}",
                files.get("tlc.txt").ok_or("missing TLC receipt")?
            );
        }
    }
    Ok(())
}
