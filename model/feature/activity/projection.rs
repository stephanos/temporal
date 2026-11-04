//! Feature-local World adapter, observable Describe target and full-state protocol map.
//!
//! The target holds only validated actual fields; latent budgets are never observable and a
//! target match never authorizes an edge or a Tick by itself.

use super::activity::{self, Phase, Remaining, Ticks};
use crate::activity_protocol::{
    parameters::{ParameterError, Parameters},
    protocol,
    view::{Observed, RetainedFailure, RunState, Snapshot, Status, TerminalOutcome, View},
};
use umpire_runtime::{ReachTarget, protocol::Refusal};
use umpire_temporal_api::temporal::api::{
    activity::v1::activity_execution_outcome::Value,
    enums::v1::{ActivityExecutionStatus, TimeoutType},
    failure::v1::{Failure, failure::FailureInfo},
    workflowservice::v1::DescribeActivityExecutionResponse,
};

/// Construct the feature World from exact neutral parameters, without defaults or clamping.
///
/// # Errors
/// Refuses invalid values and, outside the wasm guest (which carries no source schema), any
/// disagreement with the feature's declared World schema.
pub fn world(parameters: Parameters) -> Result<activity::World, ParameterError> {
    let exact = Parameters::new(
        parameters.maximum_attempts,
        [
            parameters.schedule_to_start,
            parameters.start_to_close,
            parameters.schedule_to_close,
            parameters.heartbeat_timeout,
        ],
    )?;
    let world = activity::World::new(
        exact.maximum_attempts,
        exact.schedule_to_start.map(Ticks),
        exact.start_to_close.map(Ticks),
        exact.schedule_to_close.map(Ticks),
        exact.heartbeat_timeout.map(Ticks),
    );
    #[cfg(not(target_arch = "wasm32"))]
    exact.validate_world(&world, crate::activity_protocol::parameters::WorldSource::Feature)?;
    Ok(world)
}

/// Total map from a full protocol state to the full feature state.
///
/// Every budget maps exactly, absent stays absent and present zero stays present zero.
/// `retained_failure` has no feature field: the feature Model abstracts last-attempt history.
/// States the feature type cannot represent have no image: an open lifecycle with a terminal
/// outcome, a Closed lifecycle without one, or a budget outside 0..=3. Other protocol-invariant
/// violations, such as an armed present-zero budget, map exactly so the refinement can reject them.
pub fn feature_state(state: &protocol::Model) -> Option<activity::Model> {
    let remaining = |budget: Option<protocol::Remaining>| match budget {
        None => Some(None),
        Some(budget) if budget.ticks <= 3 => Some(Some(Remaining { ticks: budget.ticks })),
        Some(_) => None,
    };
    let open = state.terminal_outcome == protocol::Terminal::None;
    Some(activity::Model {
        phase: match (state.lifecycle, state.terminal_outcome) {
            (protocol::Lifecycle::Queued, _) if open => Phase::Scheduled,
            (protocol::Lifecycle::Executing, _) if open => Phase::Started,
            (protocol::Lifecycle::Closed, protocol::Terminal::Completed) => Phase::Completed,
            (protocol::Lifecycle::Closed, protocol::Terminal::Failed) => Phase::Failed,
            (protocol::Lifecycle::Closed, protocol::Terminal::Canceled) => Phase::Canceled,
            (protocol::Lifecycle::Closed, protocol::Terminal::TimedOut) => Phase::TimedOut,
            (
                protocol::Lifecycle::Queued
                | protocol::Lifecycle::Executing
                | protocol::Lifecycle::Closed,
                _,
            ) => return None,
        },
        attempt: state.attempt,
        cancel_requested: state.cancellation_pending,
        schedule_to_start_remaining: remaining(state.queued_budget)?,
        start_to_close_remaining: remaining(state.attempt_budget)?,
        schedule_to_close_remaining: remaining(state.execution_budget)?,
        heartbeat_remaining: remaining(state.heartbeat_budget)?,
    })
}

/// The feature action each protocol action reproduces.
pub fn feature_action(action: &protocol::Action) -> activity::Action {
    use activity::{Client, Deadline, Worker};
    match *action {
        protocol::Action::Client(protocol::Client::RequestCancel) => Client::Cancel.into(),
        protocol::Action::Worker(protocol::Worker::Deliver(attempt)) => {
            Worker::Poll(attempt).into()
        }
        protocol::Action::Worker(protocol::Worker::Complete) => Worker::RespondCompleted.into(),
        protocol::Action::Worker(protocol::Worker::Fail { retryable }) => {
            Worker::RespondFailed { retryable }.into()
        }
        protocol::Action::Worker(protocol::Worker::Heartbeat) => Worker::Heartbeat.into(),
        protocol::Action::Worker(protocol::Worker::AcknowledgeCancel(attempt)) => {
            Worker::RespondCanceled(attempt).into()
        }
        protocol::Action::Timer(protocol::Timer::Tick) => Deadline::Tick.into(),
    }
}

/// The feature's exact expiry fact for each protocol expiry cause.
pub fn feature_fact(fact: protocol::Expired) -> activity::ExpiredDeadline {
    match fact {
        protocol::Expired::ScheduleToStart => activity::ExpiredDeadline::ScheduleToStart,
        protocol::Expired::StartToClose => activity::ExpiredDeadline::StartToClose,
        protocol::Expired::ScheduleToClose => activity::ExpiredDeadline::ScheduleToClose,
        protocol::Expired::Heartbeat => activity::ExpiredDeadline::Heartbeat,
    }
}

/// The validated observable facts of one actual Describe response.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct ActivityTarget(pub Snapshot);

impl ActivityTarget {
    /// Decode only actually supplied fields; an omitted last failure stays unknown. An outcome
    /// that contradicts its status is in the Model's vocabulary, so it is no refusal here:
    /// [`ActivityTarget::agrees`] withholds evidence and the judgment is Unexplained.
    ///
    /// # Errors
    /// Refuses missing info, unsupported statuses, run states or attempts, and unsupported
    /// retained failure kinds.
    pub fn decode(response: &DescribeActivityExecutionResponse) -> Result<Self, Refusal> {
        let info =
            response.info.as_ref().ok_or_else(|| Refusal::malformed("activity info is missing"))?;
        let (status, run_state) = super::phase::lifecycle(info)?;
        let attempt =
            u8::try_from(info.attempt).ok().filter(|attempt| *attempt >= 1).ok_or_else(|| {
                Refusal::malformed("Describe attempt is outside the admitted attempts")
            })?;
        let terminal_outcome = match status {
            Status::Running => TerminalOutcome::None,
            Status::Completed => TerminalOutcome::Completed,
            Status::Failed => TerminalOutcome::Failed,
            Status::Canceled => TerminalOutcome::Canceled,
            Status::TimedOut => TerminalOutcome::TimedOut,
        };
        let retained_failure = match info.last_failure.as_ref() {
            Some(failure) => Observed::Known(retained(failure)?),
            None => Observed::Unknown,
        };
        Ok(Self(Snapshot {
            status: Observed::Known(status),
            run_state: Observed::Known(run_state),
            attempt: Observed::Known(attempt),
            terminal_outcome: Observed::Known(terminal_outcome),
            retained_failure,
            cancellation_pending: Observed::Known(run_state == Some(RunState::CancelRequested)),
        }))
    }

    /// Whether a supplied outcome agrees with the supplied status.
    pub fn agrees(response: &DescribeActivityExecutionResponse) -> bool {
        let status = response.info.as_ref().map(|info| info.status);
        let Some(value) = response.outcome.as_ref().and_then(|outcome| outcome.value.as_ref())
        else {
            return true;
        };
        let Some(status) = status.and_then(|status| ActivityExecutionStatus::try_from(status).ok())
        else {
            return false;
        };
        match (status, value) {
            (ActivityExecutionStatus::Completed, Value::Result(_))
            | (ActivityExecutionStatus::Failed, Value::Failure(_)) => true,
            (ActivityExecutionStatus::Canceled, Value::Failure(failure)) => {
                matches!(failure.failure_info, Some(FailureInfo::CanceledFailureInfo(_)))
            }
            (ActivityExecutionStatus::TimedOut, Value::Failure(failure)) => {
                matches!(failure.failure_info, Some(FailureInfo::TimeoutFailureInfo(_)))
            }
            _ => false,
        }
    }

    /// A target that observes no field, for an observation without a snapshot.
    pub fn unobserved() -> Self {
        Self(Snapshot {
            status: Observed::Unknown,
            run_state: Observed::Unknown,
            attempt: Observed::Unknown,
            terminal_outcome: Observed::Unknown,
            retained_failure: Observed::Unknown,
            cancellation_pending: Observed::Unknown,
        })
    }

    /// Every fact of a complete protocol view, as the commuting offline control uses it.
    pub fn observe(view: &View) -> Self {
        Self(Snapshot {
            status: Observed::Known(view.status),
            run_state: Observed::Known(view.run_state),
            attempt: Observed::Known(view.attempt),
            terminal_outcome: Observed::Known(view.terminal_outcome),
            retained_failure: Observed::Known(view.retained_failure),
            cancellation_pending: Observed::Known(view.cancellation_pending),
        })
    }

    /// Whether the observed status is terminal.
    pub fn closed(&self) -> bool {
        matches!(self.0.status, Observed::Known(status) if status != Status::Running)
    }
}

/// The public view of a feature state; the feature does not represent retained failure.
fn visible(state: &activity::Model, retained_failure: RetainedFailure) -> View {
    let (status, terminal_outcome) = match state.phase {
        Phase::Scheduled | Phase::Started => (Status::Running, TerminalOutcome::None),
        Phase::Completed => (Status::Completed, TerminalOutcome::Completed),
        Phase::Failed => (Status::Failed, TerminalOutcome::Failed),
        Phase::Canceled => (Status::Canceled, TerminalOutcome::Canceled),
        Phase::TimedOut => (Status::TimedOut, TerminalOutcome::TimedOut),
    };
    View {
        status,
        run_state: match state.phase {
            Phase::Scheduled => Some(RunState::Scheduled),
            Phase::Started if state.cancel_requested => Some(RunState::CancelRequested),
            Phase::Started => Some(RunState::Started),
            _ => None,
        },
        attempt: state.attempt,
        terminal_outcome,
        retained_failure,
        cancellation_pending: state.cancel_requested,
    }
}

impl ReachTarget<activity::Model> for ActivityTarget {
    fn matches(&self, candidate: &activity::Model) -> bool {
        // The feature cannot contradict a retained failure it does not represent.
        let retained = match self.0.retained_failure {
            Observed::Known(failure) => failure,
            Observed::Unknown => RetainedFailure::None,
        };
        self.0.matches(&visible(candidate, retained))
    }
}

fn retained(failure: &Failure) -> Result<RetainedFailure, Refusal> {
    Ok(match &failure.failure_info {
        Some(FailureInfo::ApplicationFailureInfo(info)) if info.non_retryable => {
            RetainedFailure::PermanentWorker
        }
        Some(FailureInfo::ApplicationFailureInfo(_)) => RetainedFailure::RetryableWorker,
        Some(FailureInfo::TimeoutFailureInfo(info)) => {
            match TimeoutType::try_from(info.timeout_type) {
                Ok(TimeoutType::ScheduleToStart) => RetainedFailure::ScheduleToStart,
                Ok(TimeoutType::StartToClose) => RetainedFailure::StartToClose,
                Ok(TimeoutType::ScheduleToClose) => RetainedFailure::ScheduleToClose,
                Ok(TimeoutType::Heartbeat) => RetainedFailure::Heartbeat,
                Ok(TimeoutType::Unspecified) | Err(_) => {
                    return Err(Refusal::malformed("timeout failure has no timeout type"));
                }
            }
        }
        _ => return Err(Refusal::unsupported("unsupported retained failure kind")),
    })
}

#[cfg(test)]
#[path = "../../realization/tests/activity_projection/mod.rs"]
mod projection_controls;
