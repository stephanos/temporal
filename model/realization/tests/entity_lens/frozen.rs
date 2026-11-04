//! The frozen pre-Reach Describe and cancellation projections, kept only as the equivalence
//! oracle for the Reach cutover; no runtime path reads them.

use umpire_runtime::{Candidate, Projection, protocol::Refusal};
use umpire_temporal_api::temporal::api::{
    activity::v1::{ActivityExecutionOutcome, activity_execution_outcome::Value},
    failure::v1::Failure,
    workflowservice::v1::{
        DescribeActivityExecutionResponse, RequestCancelActivityExecutionResponse,
        RespondActivityTaskCompletedRequest, RespondActivityTaskFailedRequest,
    },
};

use crate::{Realization, SCENARIO_HEADER, activity, support::rejection};
use activity::{Client, Phase, Worker};
use umpire_runtime::AnyMessage as _;

type Action = <activity::World as umpire_dsl::Machine>::Action;

pub(super) fn cancel(
    response: Result<RequestCancelActivityExecutionResponse, i32>,
) -> Projection<Action> {
    let outcome = if let Err(code) = response {
        let Some(rejection) = rejection(code) else {
            return Projection::steps([]);
        };
        umpire_dsl::Outcome::Rejected(rejection)
    } else {
        umpire_dsl::Outcome::Accepted
    };
    Projection::Candidates(vec![Candidate {
        path: vec![Action::from(Client::Cancel).into()],
        outcome: Some(outcome),
    }])
}

pub(super) struct Frozen<'a>(pub(super) &'a Realization);

impl Frozen<'_> {
    pub(super) fn describe(
        &self,
        response: DescribeActivityExecutionResponse,
        state: &activity::Model,
        world: &activity::World,
    ) -> Result<Projection<Action>, Refusal> {
        let info = response.info.ok_or_else(|| Refusal::malformed("activity info is missing"))?;
        let raw_attempt = info.attempt;
        let phase = Phase::try_from(info.clone())?;
        let attempt = u8::try_from(raw_attempt)
            .map_err(|_| Refusal::unsupported("Describe attempt is outside maximum_attempts"))?;
        if !(state.attempt..=3).contains(&attempt) {
            return Err(Refusal::new(
                umpire_runtime::protocol::RefusalCode::State,
                "Describe attempt does not match the current attempt",
            ));
        }
        let mut terminal_retryable = false;
        if let Some(context) = self.0.context() {
            let scenario = context
                .start
                .header
                .as_ref()
                .and_then(|header| header.fields.get(SCENARIO_HEADER))
                .and_then(|payload| std::str::from_utf8(&payload.data).ok())
                .ok_or_else(|| Refusal::malformed("Start scenario header is missing"))?;
            if let Some(failure) = info.last_failure.as_ref() {
                Realization::programmed_failure(failure)?;
            }
            if let Some(Value::Failure(failure)) =
                response.outcome.as_ref().and_then(|outcome| outcome.value.as_ref())
            {
                Realization::programmed_failure(failure)?;
            }
            if !Self::consistent_evidence(
                phase,
                attempt,
                info.last_failure.as_ref(),
                response.outcome.as_ref(),
            ) {
                return Ok(Projection::steps([]));
            }
            if attempt > 1 && phase != Phase::Failed {
                let (action, expected) =
                    self.0.response(world, scenario, i32::from(attempt - 1))?;
                if !matches!(action, Worker::RespondFailed { retryable: true }) {
                    return Ok(Projection::steps([]));
                }
                let expected: RespondActivityTaskFailedRequest = expected.unpack()?;
                if !Realization::same_failure(info.last_failure.as_ref(), expected.failure.as_ref())
                {
                    return Ok(Projection::steps([]));
                }
            }
            for skipped in state.attempt..attempt {
                let (action, _) = self.0.response(world, scenario, i32::from(skipped))?;
                if !matches!(action, Worker::RespondFailed { retryable: true }) {
                    return Ok(Projection::steps([]));
                }
            }
            let (action, expected) = self.0.response(world, scenario, raw_attempt)?;
            match phase {
                Phase::Completed => {
                    let Some(Value::Result(result)) =
                        response.outcome.as_ref().and_then(|outcome| outcome.value.as_ref())
                    else {
                        return Ok(Projection::steps([]));
                    };
                    if !matches!(action, Worker::RespondCompleted) {
                        return Ok(Projection::steps([]));
                    }
                    let expected: RespondActivityTaskCompletedRequest = expected.unpack()?;
                    if expected.result.as_ref() != Some(result) {
                        return Ok(Projection::steps([]));
                    }
                }
                Phase::Failed => {
                    let Some(Value::Failure(failure)) =
                        response.outcome.as_ref().and_then(|outcome| outcome.value.as_ref())
                    else {
                        return Ok(Projection::steps([]));
                    };
                    let Worker::RespondFailed { retryable } = action else {
                        return Ok(Projection::steps([]));
                    };
                    terminal_retryable = retryable;
                    let expected: RespondActivityTaskFailedRequest = expected.unpack()?;
                    if !Realization::same_failure(Some(failure), expected.failure.as_ref())
                        || !Realization::same_failure(
                            info.last_failure.as_ref(),
                            expected.failure.as_ref(),
                        )
                    {
                        return Ok(Projection::steps([]));
                    }
                }
                _ => {}
            }
        }
        Ok(Self::project_phase(state, &Self::observed_state(phase, attempt), terminal_retryable))
    }

    fn consistent_evidence(
        phase: Phase,
        attempt: u8,
        last_failure: Option<&Failure>,
        outcome: Option<&ActivityExecutionOutcome>,
    ) -> bool {
        use umpire_temporal_api::temporal::api::failure::v1::failure::FailureInfo;
        if attempt == 1
            && !matches!(phase, Phase::Failed | Phase::TimedOut)
            && last_failure.is_some()
        {
            return false;
        }
        let Some(outcome) = outcome else {
            return true;
        };
        match phase {
            Phase::Scheduled | Phase::Started => false,
            Phase::Canceled => matches!(&outcome.value, Some(Value::Failure(failure))
                if matches!(&failure.failure_info, Some(FailureInfo::CanceledFailureInfo(_)))),
            Phase::TimedOut => matches!(&outcome.value, Some(Value::Failure(failure))
                if matches!(&failure.failure_info, Some(FailureInfo::TimeoutFailureInfo(_)))),
            Phase::Completed | Phase::Failed => true,
        }
    }

    fn observed_state(phase: Phase, attempt: u8) -> activity::Model {
        activity::Model {
            phase,
            attempt,
            cancel_requested: false,
            schedule_to_start_remaining: None,
            start_to_close_remaining: None,
            schedule_to_close_remaining: None,
            heartbeat_remaining: None,
        }
    }

    fn project_phase(
        state: &activity::Model,
        observed: &activity::Model,
        terminal_retryable: bool,
    ) -> Projection<Action> {
        if state == observed {
            return Projection::Current;
        }
        if umpire_dsl::Phase::closed(&state.phase) {
            return Projection::steps([]);
        }
        let mut phase = state.phase;
        let mut path = Vec::new();
        for attempt in state.attempt..observed.attempt {
            if phase == Phase::Scheduled {
                path.push(Action::from(Worker::Poll(attempt)).into());
            }
            path.push(Action::from(Worker::RespondFailed { retryable: true }).into());
            phase = Phase::Scheduled;
        }
        if phase == Phase::Scheduled
            && matches!(observed.phase, Phase::Started | Phase::Completed | Phase::Failed)
        {
            path.push(Action::from(Worker::Poll(observed.attempt)).into());
            phase = Phase::Started;
        }
        if phase == Phase::Started {
            match observed.phase {
                Phase::Started => {}
                Phase::Completed => {
                    path.push(Action::from(Worker::RespondCompleted).into());
                    phase = Phase::Completed;
                }
                Phase::Failed => {
                    path.push(
                        Action::from(Worker::RespondFailed { retryable: terminal_retryable })
                            .into(),
                    );
                    phase = Phase::Failed;
                }
                _ if observed.attempt == state.attempt => return Projection::steps([]),
                _ => {}
            }
        }
        if path.is_empty() || phase != observed.phase {
            Projection::steps([])
        } else {
            Projection::steps([path])
        }
    }
}
