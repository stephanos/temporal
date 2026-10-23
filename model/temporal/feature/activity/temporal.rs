//! Temporal protobuf realization for the independently authored mini activity Model.

// The fixture runner compiles the Model independently of this realization.
#[path = "model.rs"]
mod included;
mod phase;

pub use included::activity;

use activity::{Client, Phase, Worker};
use umpire_dsl::{Machine, Outcome};
use umpire_runtime::{Candidate, Projection, google::protobuf::Any, protocol::Refusal};
use umpire_temporal_api::temporal::api::{
    common::v1::Payloads,
    failure::v1::Failure,
    workflowservice::v1::{
        DescribeActivityExecutionResponse, PollActivityTaskQueueResponse,
        RequestCancelActivityExecutionResponse, RespondActivityTaskCompletedRequest,
        RespondActivityTaskFailedRequest,
    },
};

use crate::{
    Reply, realization,
    support::{ApplicationFailure, plain_text, rejection},
};

type Action = <activity::World as Machine>::Action;

/// Creates one independently owned guest run.
pub fn runtime() -> umpire_runtime::Runtime<activity::World, Realization> {
    umpire_runtime::Runtime::new(activity::World::new(3), Realization)
}

/// The strict mini activity observation and callback registry.
pub struct Realization;

#[realization(activity)]
impl Realization {
    #[observe(WorkflowService::RequestCancelActivityExecution)]
    fn cancel(
        response: Result<RequestCancelActivityExecutionResponse, i32>,
    ) -> Result<Projection<Action>, Refusal> {
        let outcome = if let Err(code) = response {
            let Some(rejection) = rejection(code) else {
                return Ok(Projection::steps([]));
            };
            Outcome::Rejected(rejection)
        } else {
            Outcome::Accepted
        };
        Ok(Projection::Candidates(vec![Candidate {
            path: vec![Action::from(Client::Cancel).into()],
            outcome: Some(outcome),
        }]))
    }

    #[observe(WorkflowService::DescribeActivityExecution,
        reads(info.status, info.run_state, info.attempt),
        ignores(
            run_id = "Run identity is outside the lifecycle projection",
            input = "Describe input does not evidence callback delivery",
            outcome = "Lifecycle projection reads status rather than result contents",
            long_poll_token = "Transport polling state is outside the Model",
            callbacks = "Callback registration is outside the Model",
            info.activity_id = "Activity identity is bound by the owning run",
            info.run_id = "Run identity is outside the lifecycle projection",
            info.activity_type = "Activity type is outside the lifecycle projection",
            info.task_queue = "Queue routing is outside the lifecycle projection",
            info.schedule_to_close_timeout = "Timing policy is outside the lifecycle projection",
            info.schedule_to_start_timeout = "Timing policy is outside the lifecycle projection",
            info.start_to_close_timeout = "Timing policy is outside the lifecycle projection",
            info.heartbeat_timeout = "Heartbeat timing is outside the lifecycle projection",
            info.retry_policy = "Retry policy is outside the six-phase Model",
            info.heartbeat_details = "Heartbeat payloads are outside the lifecycle projection",
            info.last_heartbeat_time = "Observed timestamps are outside the lifecycle projection",
            info.last_started_time = "Observed timestamps are outside the lifecycle projection",
            info.execution_duration = "Measured duration is outside the lifecycle projection",
            info.schedule_time = "Observed timestamps are outside the lifecycle projection",
            info.expiration_time = "Observed timestamps are outside the lifecycle projection",
            info.close_time = "Observed timestamps are outside the lifecycle projection",
            info.last_failure = "Lifecycle status alone supplies terminal evidence",
            info.last_worker_identity = "Worker identity is outside the lifecycle projection",
            info.current_retry_interval = "Retry policy is outside the six-phase Model",
            info.last_attempt_complete_time = "Retry timing is outside the six-phase Model",
            info.next_attempt_schedule_time = "Retry timing is outside the six-phase Model",
            info.last_deployment_version = "Deployment routing is outside the Model",
            info.priority = "Scheduling priority is outside the Model",
            info.state_transition_count = "Server counters are outside the lifecycle projection",
            info.state_size_bytes = "Server storage size is outside the lifecycle projection",
            info.search_attributes = "Search indexing is outside the Model",
            info.header = "Application headers are outside the lifecycle projection",
            info.user_metadata = "Application metadata is outside the lifecycle projection",
            info.canceled_reason = "Cancellation reasons are outside the lifecycle projection",
            info.links = "Linked entities are outside the Model",
            info.total_heartbeat_count = "Heartbeat counters are outside the lifecycle projection",
            info.sdk_name = "SDK identity is outside the lifecycle projection",
            info.sdk_version = "SDK identity is outside the lifecycle projection",
            info.start_delay = "Timing policy is outside the lifecycle projection",
            info.execution_time = "Observed timestamps are outside the lifecycle projection"
        )
    )]
    fn describe(
        response: DescribeActivityExecutionResponse,
        state: &activity::Model,
    ) -> Result<Projection<Action>, Refusal> {
        let info = response.info.ok_or_else(|| Refusal::malformed("activity info is missing"))?;
        let raw_attempt = info.attempt;
        let phase = Phase::try_from(info)?;
        let attempt = u8::try_from(raw_attempt)
            .map_err(|_| Refusal::unsupported("Describe attempt is outside maximum_attempts"))?;
        let retry = state.phase == Phase::Started
            && phase == Phase::Scheduled
            && state.attempt.checked_add(1) == Some(attempt);
        if state.attempt != attempt && !retry {
            return Err(Refusal::new(
                umpire_runtime::protocol::RefusalCode::State,
                "Describe attempt does not match the current attempt",
            ));
        }
        Ok(Self::project_phase(state, &activity::Model { phase, attempt }))
    }

    fn project_phase(state: &activity::Model, observed: &activity::Model) -> Projection<Action> {
        if state == observed {
            Projection::Current
        } else {
            if state.phase == Phase::Started
                && observed.phase == Phase::Scheduled
                && state.attempt.checked_add(1) == Some(observed.attempt)
            {
                return Projection::step(Worker::RespondFailed { retryable: true }.into());
            }
            if state.attempt != observed.attempt {
                return Projection::steps([]);
            }
            match observed.phase {
                Phase::Completed => Projection::step(Worker::RespondCompleted.into()),
                Phase::Failed => {
                    Projection::step(Worker::RespondFailed { retryable: false }.into())
                }
                _ => Projection::steps([]),
            }
        }
    }

    #[answer(PollActivityTaskQueueResponse,
        reads(input.payloads.metadata, input.payloads.data, attempt),
        ignores(
            input.payloads.external_payloads = "External references are accepted then removed from completed results",
            task_token = "The SDK owns task-token transport",
            workflow_namespace = "The SDK owns namespace transport",
            workflow_type = "Workflow identity is outside this callback projection",
            workflow_execution = "Workflow identity is outside this callback projection",
            activity_type = "Activity type is outside the callback policy",
            activity_id = "Activity identity is bound by the owning run",
            header = "Application headers are outside the callback policy",
            heartbeat_details = "Heartbeat payloads are outside the callback policy",
            scheduled_time = "Server timing is outside the callback policy",
            current_attempt_scheduled_time = "Retry timing is outside the callback policy",
            started_time = "Server timing is outside the callback policy",
            schedule_to_close_timeout = "Timing policy is outside the callback projection",
            start_to_close_timeout = "Timing policy is outside the callback projection",
            heartbeat_timeout = "Heartbeat timing is outside the callback projection",
            retry_policy = "Retry policy is outside the six-phase Model",
            poller_scaling_decision = "Worker scaling is outside the Model",
            priority = "Scheduling priority is outside the Model",
            activity_run_id = "Run identity is outside the callback policy",
            poller_group_infos = "Poller routing is outside the Model",
            poller_groups_info = "Poller routing is outside the Model"
        ),
        produces(RespondActivityTaskCompletedRequest, fields(result), ignores(
            task_token = "The SDK supplies the task token",
            identity = "The SDK supplies worker identity",
            namespace = "The SDK supplies the namespace",
            resource_id = "The SDK owns resource transport",
            worker_version = "The SDK supplies worker version metadata",
            deployment = "The SDK supplies deployment metadata",
            deployment_options = "The SDK supplies deployment routing"
        )),
        produces(RespondActivityTaskFailedRequest, fields(failure), ignores(
            task_token = "The SDK supplies the task token",
            identity = "The SDK supplies worker identity",
            namespace = "The SDK supplies the namespace",
            resource_id = "The SDK owns resource transport",
            last_heartbeat_details = "The callback emits no heartbeat payloads",
            worker_version = "The SDK supplies worker version metadata",
            deployment = "The SDK supplies deployment metadata",
            deployment_options = "The SDK supplies deployment routing"
        ))
    )]
    fn respond(
        task: PollActivityTaskQueueResponse,
        state: &activity::Model,
    ) -> Result<Reply<Worker>, Refusal> {
        let input = task.input.ok_or_else(|| Refusal::malformed("callback input is missing"))?;
        let [payload] = input.payloads.as_slice() else {
            return Err(Refusal::malformed("callback requires exactly one input payload"));
        };
        let (action, response) = match plain_text(payload)? {
            b"RetryThenComplete" if task.attempt == 1 => (
                Worker::RespondFailed { retryable: true },
                Any::pack(&RespondActivityTaskFailedRequest {
                    failure: Some(Failure::application("MiniActivityFailure", "Retry", false)),
                    ..Default::default()
                }),
            ),
            b"Complete" | b"RetryThenComplete" => {
                let mut payload = payload.clone();
                payload.external_payloads.clear();
                payload.data = b"Complete".to_vec();
                (
                    Worker::RespondCompleted,
                    Any::pack(&RespondActivityTaskCompletedRequest {
                        result: Some(Payloads { payloads: vec![payload] }),
                        ..Default::default()
                    }),
                )
            }
            b"Fail" => (
                Worker::RespondFailed { retryable: false },
                Any::pack(&RespondActivityTaskFailedRequest {
                    failure: Some(Failure::application("MiniActivityFailure", "Fail", true)),
                    ..Default::default()
                }),
            ),
            _ => return Err(Refusal::unsupported("unsupported callback policy")),
        };
        let attempt = u8::try_from(task.attempt)
            .map_err(|_| Refusal::unsupported("callback attempt is outside maximum_attempts"))?;
        if attempt != state.attempt {
            return Err(Refusal::new(
                umpire_runtime::protocol::RefusalCode::State,
                "callback attempt does not match the current attempt",
            ));
        }
        Ok(Reply { delivery: vec![Worker::Poll(attempt)], action, request: response })
    }
}
