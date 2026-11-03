use umpire_dsl::{
    __lifted::{Bounded, Lifts, Type},
    Machine, Outcome, Value, Values,
};
use umpire_temporal::activity_protocol::{
    adapter,
    parameters::Parameters,
    protocol::{Action, Client, Model, Remaining, Timer, Worker, World},
    view::{
        Acknowledgement, HeartbeatReply, Observed, RetainedFailure, RunState, Snapshot, Status,
        TerminalOutcome, View,
    },
};

type TestResult = umpire_testing::TestResult;
type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;

fn selected(timeouts: [Option<u8>; 4]) -> Result<World> {
    Ok(adapter::world(Parameters::new(2, timeouts, true)?)?)
}

fn initial(world: &World) -> Result<Model> {
    world.starts().into_iter().next().ok_or_else(|| "missing protocol start".into())
}

fn accepted(world: &World, before: &Model, action: Action) -> Result<Model> {
    let edges = world.checked_step(before, action)?;
    assert_eq!(edges.len(), 1);
    let (outcome, _, after) = edges.first().ok_or("missing accepted edge")?;
    assert_eq!(*outcome, Outcome::Accepted);
    Ok(after.clone())
}

#[test]
fn latent_countdowns_do_not_become_public_snapshot_fields() -> TestResult {
    let world = selected([Some(3), Some(3), Some(3), Some(3)])?;
    let running = accepted(&world, &initial(&world)?, Worker::Deliver(1).into())?;
    let elapsed = accepted(&world, &running, Timer::Tick.into())?;
    assert_ne!(running, elapsed);
    assert_ne!(running.attempt_budget, elapsed.attempt_budget);
    assert_ne!(running.execution_budget, elapsed.execution_budget);
    assert_ne!(running.heartbeat_budget, elapsed.heartbeat_budget);
    let view = View::from(&running);
    assert_eq!(view, View::from(&elapsed));
    let snapshot = Snapshot {
        status: Observed::Known(Status::Running),
        run_state: Observed::Known(Some(RunState::Started)),
        attempt: Observed::Known(1),
        terminal_outcome: Observed::Known(TerminalOutcome::None),
        retained_failure: Observed::Known(RetainedFailure::None),
        cancellation_pending: Observed::Known(false),
    };
    assert!(snapshot.matches(&view));
    assert!(snapshot.matches(&View::from(&elapsed)));
    let queued = initial(&world)?;
    let waiting = accepted(&world, &queued, Timer::Tick.into())?;
    assert_ne!(queued.queued_budget, waiting.queued_budget);
    assert_eq!(View::from(&queued), View::from(&waiting));
    Ok(())
}

#[test]
fn unknown_and_known_absence_constrain_views_differently() -> TestResult {
    let world = selected([None; 4])?;
    let queued = initial(&world)?;
    let running = accepted(&world, &queued, Worker::Deliver(1).into())?;
    let retried = accepted(&world, &running, Worker::Fail { retryable: true }.into())?;
    let no_failure = Snapshot {
        retained_failure: Observed::Known(RetainedFailure::None),
        ..Snapshot::default()
    };
    assert!(no_failure.matches(&View::from(&queued)));
    assert!(!no_failure.matches(&View::from(&retried)));
    assert!(Snapshot::default().matches(&View::from(&retried)));
    let closed_run_state = Snapshot { run_state: Observed::Known(None), ..Snapshot::default() };
    assert!(!closed_run_state.matches(&View::from(&running)));
    let completed = accepted(&world, &running, Worker::Complete.into())?;
    assert!(closed_run_state.matches(&View::from(&completed)));
    Ok(())
}

#[test]
fn completion_keeps_last_attempt_failure_separate_from_outcome() -> TestResult {
    let world = selected([None; 4])?;
    let running = accepted(&world, &initial(&world)?, Worker::Deliver(1).into())?;
    let retried = accepted(&world, &running, Worker::Fail { retryable: true }.into())?;
    let running = accepted(&world, &retried, Worker::Deliver(2).into())?;
    let completed = accepted(&world, &running, Worker::Complete.into())?;
    let view = View::from(&completed);
    assert_eq!(view.status, Status::Completed);
    assert_eq!(view.terminal_outcome, TerminalOutcome::Completed);
    assert_eq!(view.retained_failure, RetainedFailure::RetryableWorker);
    assert_eq!(view.run_state, None);
    assert_eq!(view.attempt, 2);
    let actual = Snapshot {
        terminal_outcome: Observed::Known(TerminalOutcome::Completed),
        retained_failure: Observed::Known(RetainedFailure::RetryableWorker),
        ..Snapshot::default()
    };
    assert!(actual.matches(&view));
    assert!(
        !Snapshot { terminal_outcome: Observed::Known(TerminalOutcome::Failed), ..actual }
            .matches(&view)
    );
    Ok(())
}

#[test]
fn heartbeat_notification_and_worker_acknowledgement_have_distinct_results() -> TestResult {
    let world = selected([None, Some(2), None, Some(2)])?;
    let running = accepted(&world, &initial(&world)?, Worker::Deliver(1).into())?;
    let pending = accepted(&world, &running, Client::RequestCancel.into())?;
    let heartbeat = accepted(&world, &pending, Worker::Heartbeat.into())?;
    let notification = HeartbeatReply { cancellation_pending: true };
    let observed = Snapshot {
        status: Observed::Known(Status::Running),
        run_state: Observed::Known(Some(RunState::CancelRequested)),
        cancellation_pending: Observed::Known(notification.cancellation_pending),
        ..Snapshot::default()
    };
    assert!(observed.matches(&View::from(&heartbeat)));
    assert_eq!(View::from(&heartbeat).terminal_outcome, TerminalOutcome::None);
    let acknowledgement = Acknowledgement { attempt: 1 };
    let canceled =
        accepted(&world, &heartbeat, Worker::AcknowledgeCancel(acknowledgement.attempt).into())?;
    assert_eq!(View::from(&canceled).terminal_outcome, TerminalOutcome::Canceled);
    assert!(!observed.matches(&View::from(&canceled)));
    let failure = accepted(&world, &pending, Worker::Fail { retryable: true }.into())?;
    assert_eq!(View::from(&failure).terminal_outcome, TerminalOutcome::Failed);
    Ok(())
}

fn expected_fields() -> Vec<(&'static str, Type)> {
    let enumeration = |name: &str, variants: &[&str]| Type::Enum {
        name: name.into(),
        variants: variants.iter().map(|variant| format!("{name}::{variant}")).collect(),
    };
    let remaining = Type::Optional(Bounded {
        name: "Remaining".into(),
        field: Some("ticks".into()),
        backing: "u8".into(),
        min: 0,
        max: 255,
        low: 0,
        high: 3,
    });
    vec![
        ("lifecycle", enumeration("Lifecycle", &["Queued", "Executing", "Closed"])),
        ("attempt", Type::Int { min: 0, max: 255 }),
        ("cancellation_pending", Type::Bool),
        (
            "terminal_outcome",
            enumeration("Terminal", &["None", "Completed", "Failed", "Canceled", "TimedOut"]),
        ),
        (
            "retained_failure",
            enumeration(
                "Failure",
                &[
                    "None",
                    "RetryableWorker",
                    "PermanentWorker",
                    "ScheduleToStart",
                    "StartToClose",
                    "ScheduleToClose",
                    "Heartbeat",
                ],
            ),
        ),
        ("queued_budget", remaining.clone()),
        ("attempt_budget", remaining.clone()),
        ("execution_budget", remaining.clone()),
        ("heartbeat_budget", remaining),
    ]
}

#[test]
fn full_state_schema_accounts_for_all_fields_and_rejects_erasure_or_retyping() -> TestResult {
    let lifted = World::__umpire_lifted().model.map_err(|error| format!("{error:?}"))?;
    adapter::validate_state_schema(&lifted)?;
    assert_eq!(
        lifted
            .fields
            .iter()
            .map(|field| (field.name.as_str(), field.ty.clone()))
            .collect::<Vec<_>>(),
        expected_fields()
    );
    assert!(lifted.fields.iter().all(|field| field.location.line > 0));
    for index in 0..9 {
        let mut erased = lifted.clone();
        erased.fields.remove(index);
        assert!(adapter::validate_state_schema(&erased).is_err());
        let mut renamed = lifted.clone();
        renamed.fields[index].name.push_str("_lost");
        assert!(adapter::validate_state_schema(&renamed).is_err());
        let mut retyped = lifted.clone();
        retyped.fields[index].ty = Type::Unit;
        assert!(adapter::validate_state_schema(&retyped).is_err());
    }
    for index in 5..9 {
        for mutation in 0..6 {
            let mut changed = lifted.clone();
            let Type::Optional(domain) = &mut changed.fields[index].ty else {
                return Err("expected optional Remaining".into());
            };
            match mutation {
                0 => domain.name = "Ticks".into(),
                1 => domain.field = None,
                2 => domain.backing = "u16".into(),
                3 => domain.low = 1,
                4 => domain.high = 2,
                _ => domain.max = 65535,
            }
            assert!(adapter::validate_state_schema(&changed).is_err());
        }
    }
    Ok(())
}

#[test]
fn optional_remaining_preserves_absence_and_present_zero_in_exported_values() -> TestResult {
    let world = selected([None; 4])?;
    let absent = initial(&world)?;
    for name in ["queued_budget", "attempt_budget", "execution_budget", "heartbeat_budget"] {
        let mut zero = absent.clone();
        match name {
            "queued_budget" => zero.queued_budget = Some(Remaining { ticks: 0 }),
            "attempt_budget" => zero.attempt_budget = Some(Remaining { ticks: 0 }),
            "execution_budget" => zero.execution_budget = Some(Remaining { ticks: 0 }),
            _ => zero.heartbeat_budget = Some(Remaining { ticks: 0 }),
        }
        let absent_values = absent.values();
        let zero_values = zero.values();
        let absent_value = &absent_values
            .iter()
            .find(|(field, _)| *field == name)
            .ok_or("missing absent budget export")?
            .1;
        let zero_value = &zero_values
            .iter()
            .find(|(field, _)| *field == name)
            .ok_or("missing zero budget export")?
            .1;
        assert_eq!(*absent_value, Value::Optional(None));
        assert_eq!(
            *zero_value,
            Value::Optional(Some(Box::new(Value::Bounded { name: "Remaining", raw: 0 })))
        );
        assert_ne!(absent_value, zero_value);
        assert_eq!(View::from(&absent), View::from(&zero));
    }
    Ok(())
}
