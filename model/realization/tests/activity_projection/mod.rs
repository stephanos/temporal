//! Feature World adapter, Describe target decoder and full-state map controls.
use super::{
    ActivityTarget, DescribeActivityExecutionResponse, Failure, FailureInfo, TimeoutType, Value,
    View, activity, feature_state, protocol, world,
};
use crate::activity_protocol::{
    adapter,
    parameters::{Parameters, WorldSource, validate_effective_world},
    view::{Observed, RetainedFailure},
};
use std::collections::{HashSet, VecDeque};
use umpire_dsl::Machine as _;
use umpire_runtime::ReachTarget as _;
use umpire_temporal_api::temporal::api::{
    activity::v1::{ActivityExecutionInfo, ActivityExecutionOutcome},
    failure::v1::{ApplicationFailureInfo, CanceledFailureInfo, TimeoutFailureInfo},
};

type Error = Box<dyn std::error::Error>;

/// Source-lifted World admission recurses deeply in unoptimized builds.
fn lifted(check: fn() -> Result<(), Error>) -> Result<(), Error> {
    std::thread::Builder::new()
        .stack_size(64 << 20)
        .spawn(move || check().map_err(|error| error.to_string()))?
        .join()
        .map_err(|_| "lifted control panicked")??;
    Ok(())
}

fn parameters(maximum: u8, timeouts: [Option<u8>; 4]) -> Result<Parameters, Error> {
    Ok(Parameters::new(maximum, timeouts)?)
}

fn reachable<M: umpire_dsl::Machine>(world: &M) -> Result<Vec<M::State>, Error>
where
    M::State: Clone + Eq + std::hash::Hash,
{
    let catalog = world.checked_catalog(umpire_dsl::CATALOG_LIMIT)?;
    let mut seen = HashSet::new();
    let mut queue: VecDeque<_> = world.starts().into_iter().collect();
    while let Some(state) = queue.pop_front() {
        if !seen.insert(state.clone()) {
            continue;
        }
        for action in catalog.actions() {
            for (_, _, next) in world.checked_step(&state, action.clone())? {
                queue.push_back(next);
            }
        }
    }
    Ok(seen.into_iter().collect())
}

const WORLDS: [(u8, [Option<u8>; 4]); 5] = [
    (1, [None; 4]),
    (3, [None; 4]),
    (2, [Some(1), Some(1), None, None]),
    (3, [None, Some(3), Some(3), Some(2)]),
    (2, [Some(2), None, Some(3), Some(1)]),
];

#[test]
fn feature_and_protocol_worlds_share_the_exact_neutral_parameters() -> Result<(), Error> {
    lifted(feature_and_protocol_worlds_share_the_exact_neutral_parameters_checks)
}

fn feature_and_protocol_worlds_share_the_exact_neutral_parameters_checks() -> Result<(), Error> {
    for (maximum, timeouts) in WORLDS {
        let exact = parameters(maximum, timeouts)?;
        let feature = world(exact)?;
        assert_eq!(validate_effective_world(&feature, WorldSource::Feature)?, exact);
        assert_eq!(
            validate_effective_world(&adapter::world(exact)?, WorldSource::Protocol)?,
            exact
        );
        assert!(validate_effective_world(&feature, WorldSource::Protocol).is_err());
    }
    let absent = world(parameters(2, [None; 4])?)?;
    let present = world(parameters(2, [Some(1), None, None, None])?)?;
    assert!(absent.schedule_to_start.is_none() && present.schedule_to_start.is_some());
    for invalid in [
        (0, [None; 4]),
        (4, [None; 4]),
        (2, [Some(0), None, None, None]),
        (2, [None, None, None, Some(4)]),
    ] {
        assert!(Parameters::new(invalid.0, invalid.1).is_err(), "{invalid:?}");
        let unchecked = Parameters {
            maximum_attempts: invalid.0,
            schedule_to_start: invalid.1[0],
            start_to_close: invalid.1[1],
            schedule_to_close: invalid.1[2],
            heartbeat_timeout: invalid.1[3],
        };
        assert!(world(unchecked).is_err(), "{invalid:?} must not clamp or default");
    }
    Ok(())
}

#[test]
fn full_state_map_commutes_with_the_observable_view_and_keeps_every_budget() -> Result<(), Error> {
    lifted(full_state_map_commutes_with_the_observable_view_and_keeps_every_budget_checks)
}

fn full_state_map_commutes_with_the_observable_view_and_keeps_every_budget_checks()
-> Result<(), Error> {
    for (maximum, timeouts) in WORLDS {
        let exact = parameters(maximum, timeouts)?;
        let protocol_world = adapter::world(exact)?;
        let feature_world = world(exact)?;
        let images: HashSet<_> = reachable(&feature_world)?.into_iter().collect();
        for state in reachable(&protocol_world)? {
            let mapped = feature_state(&state).ok_or("reachable state has no feature image")?;
            assert!(images.contains(&mapped), "{state:?} maps outside the feature graph");
            let target = ActivityTarget::observe(&View::from(&state));
            assert!(target.matches(&mapped), "{state:?}");
            let budgets = |state: &activity::Model| {
                [
                    state.schedule_to_start_remaining,
                    state.start_to_close_remaining,
                    state.schedule_to_close_remaining,
                    state.heartbeat_remaining,
                ]
                .map(|budget| budget.map(|budget| budget.ticks))
            };
            assert_eq!(
                budgets(&mapped),
                [
                    state.queued_budget,
                    state.attempt_budget,
                    state.execution_budget,
                    state.heartbeat_budget
                ]
                .map(|budget| budget.map(|budget| budget.ticks)),
                "{state:?}"
            );
            let mut wrong = mapped.clone();
            wrong.attempt = wrong.attempt % maximum + 1;
            assert_eq!(target.matches(&wrong), maximum == 1, "{state:?}");
            let mut flipped = mapped.clone();
            flipped.cancel_requested = !flipped.cancel_requested;
            assert!(!target.matches(&flipped), "{state:?}");
        }
    }
    Ok(())
}

#[test]
fn targets_never_distinguish_latent_budgets_or_authorize_a_tick() -> Result<(), Error> {
    lifted(targets_never_distinguish_latent_budgets_or_authorize_a_tick_checks)
}

fn targets_never_distinguish_latent_budgets_or_authorize_a_tick_checks() -> Result<(), Error> {
    let exact = parameters(3, [None, Some(3), Some(3), Some(2)])?;
    let feature_world = world(exact)?;
    let started = feature_world
        .checked_step(&feature_world.starts()[0], activity::Worker::Poll(1).into())?
        .remove(0)
        .2;
    let ticked = feature_world.checked_step(&started, activity::Deadline::Tick.into())?.remove(0).2;
    assert_ne!(started, ticked);
    let protocol_world = adapter::world(exact)?;
    let delivered = protocol_world
        .checked_step(&protocol_world.starts()[0], protocol::Worker::Deliver(1).into())?
        .remove(0)
        .2;
    let target = ActivityTarget::observe(&View::from(&delivered));
    assert!(target.matches(&started) && target.matches(&ticked));
    let mut erased = started.clone();
    erased.heartbeat_remaining = None;
    assert!(target.matches(&erased), "a view match is not budget evidence");
    assert_ne!(feature_state(&delivered), Some(erased));
    Ok(())
}

/// A protocol budget field and its feature image.
type Budget = fn(&mut protocol::Model) -> &mut Option<protocol::Remaining>;
type Image = fn(&activity::Model) -> Option<u8>;

#[test]
fn invariant_violating_protocol_states_have_no_feature_image() -> Result<(), Error> {
    lifted(|| {
        let exact = parameters(2, [Some(2), None, Some(3), Some(1)])?;
        let protocol_world = adapter::world(exact)?;
        let start = protocol_world.starts().remove(0);
        assert!(feature_state(&start).is_some());
        let mut open_with_outcome = start.clone();
        open_with_outcome.terminal_outcome = protocol::Terminal::Completed;
        assert_eq!(feature_state(&open_with_outcome), None);
        let mut closed_without_outcome = start.clone();
        closed_without_outcome.lifecycle = protocol::Lifecycle::Closed;
        assert_eq!(feature_state(&closed_without_outcome), None);
        let delivered =
            protocol_world.checked_step(&start, protocol::Worker::Deliver(1).into())?.remove(0).2;
        let mut executing_with_outcome = delivered;
        executing_with_outcome.terminal_outcome = protocol::Terminal::Completed;
        assert_eq!(feature_state(&executing_with_outcome), None);
        let budgets: [(Budget, Image); 4] = [
            (
                |state| &mut state.queued_budget,
                |state| state.schedule_to_start_remaining.map(|budget| budget.ticks),
            ),
            (
                |state| &mut state.attempt_budget,
                |state| state.start_to_close_remaining.map(|budget| budget.ticks),
            ),
            (
                |state| &mut state.execution_budget,
                |state| state.schedule_to_close_remaining.map(|budget| budget.ticks),
            ),
            (
                |state| &mut state.heartbeat_budget,
                |state| state.heartbeat_remaining.map(|budget| budget.ticks),
            ),
        ];
        for (budget, image) in budgets {
            let mut over = start.clone();
            *budget(&mut over) = Some(protocol::Remaining { ticks: 4 });
            assert_eq!(feature_state(&over), None);
            let mut zero = start.clone();
            *budget(&mut zero) = Some(protocol::Remaining { ticks: 0 });
            let mapped = feature_state(&zero).ok_or("present zero has an image")?;
            assert_eq!(image(&mapped), Some(0));
        }
        Ok(())
    })
}

fn describe(status: i32, run_state: i32, attempt: i32) -> DescribeActivityExecutionResponse {
    DescribeActivityExecutionResponse {
        info: Some(ActivityExecutionInfo { status, run_state, attempt, ..Default::default() }),
        ..Default::default()
    }
}

#[test]
fn describe_target_decodes_only_actual_fields() -> Result<(), Error> {
    let pending = ActivityTarget::decode(&describe(1, 3, 1)).map_err(|refusal| refusal.message)?;
    assert_eq!(pending.0.cancellation_pending, Observed::Known(true));
    assert_eq!(pending.0.retained_failure, Observed::Unknown);
    let mut failed = describe(1, 1, 2);
    failed.info.as_mut().ok_or("info")?.last_failure = Some(Failure {
        failure_info: Some(FailureInfo::ApplicationFailureInfo(ApplicationFailureInfo::default())),
        ..Default::default()
    });
    let failed = ActivityTarget::decode(&failed).map_err(|refusal| refusal.message)?;
    assert_eq!(failed.0.retained_failure, Observed::Known(RetainedFailure::RetryableWorker));
    let mut timed = describe(6, 0, 1);
    timed.info.as_mut().ok_or("info")?.last_failure = Some(Failure {
        failure_info: Some(FailureInfo::TimeoutFailureInfo(TimeoutFailureInfo {
            timeout_type: TimeoutType::Heartbeat as i32,
            ..Default::default()
        })),
        ..Default::default()
    });
    assert_eq!(
        ActivityTarget::decode(&timed).map_err(|refusal| refusal.message)?.0.retained_failure,
        Observed::Known(RetainedFailure::Heartbeat)
    );
    let canceled = |info| {
        let mut response = describe(4, 0, 1);
        response.outcome = Some(ActivityExecutionOutcome {
            value: Some(Value::Failure(Failure { failure_info: info, ..Default::default() })),
            ..Default::default()
        });
        response
    };
    assert!(
        ActivityTarget::decode(&canceled(Some(FailureInfo::CanceledFailureInfo(
            CanceledFailureInfo::default()
        ))))
        .is_ok()
    );
    assert!(ActivityTarget::decode(&canceled(None)).is_err(), "Canceled without its cause");
    for (status, run_state, attempt) in
        [(1, 0, 1), (2, 2, 1), (0, 0, 1), (1, 1, 0), (7, 0, 1), (1, 4, 1)]
    {
        assert!(
            ActivityTarget::decode(&describe(status, run_state, attempt)).is_err(),
            "{status} {run_state} {attempt}"
        );
    }
    assert!(ActivityTarget::decode(&DescribeActivityExecutionResponse::default()).is_err());
    Ok(())
}
