//! Temporal protobuf realization for the independently authored mini activity Model.

// The fixture runner compiles the Model independently of this realization.
#[path = "model.rs"]
mod included;
mod phase;

pub use included::activity;

use activity::{Client, Phase, Worker};
use umpire_dsl::{Machine, Outcome};
use umpire_runtime::{Candidate, Projection, Proposal, google::protobuf::Any, protocol::Refusal};
use umpire_temporal_api::temporal::api::{
    common::v1::Payloads,
    failure::v1::Failure,
    workflowservice::v1::{
        DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        PollActivityTaskQueueResponse, RequestCancelActivityExecutionRequest,
        RequestCancelActivityExecutionResponse, RespondActivityTaskCompletedRequest,
        RespondActivityTaskFailedRequest, StartActivityExecutionRequest,
    },
};

use crate::{
    Reply, realization,
    support::{ApplicationFailure, plain_text, rejection},
};

type Action = <activity::World as Machine>::Action;

/// Creates one independently owned guest run.
pub fn runtime() -> umpire_runtime::Runtime<activity::World, Realization> {
    umpire_runtime::Runtime::new(activity::World::new(3), Realization::default())
}

/// The strict mini activity observation and callback registry.
#[derive(Default)]
pub struct Realization {
    context: Option<ProposalContext>,
}

pub(crate) struct ProposalContext {
    pub start: StartActivityExecutionRequest,
    pub run_id: String,
    pub response: Option<(PollActivityTaskQueueResponse, Any)>,
}

impl Realization {
    pub(crate) fn configured(context: ProposalContext) -> Self {
        Self { context: Some(context) }
    }
}

#[realization(activity)]
impl Realization {
    #[propose(matches = matches_proposal, continue_polling = polling)]
    fn proposal(&self, state: &activity::Model, seed: u64) -> Result<Proposal<Action>, Refusal> {
        let context = self
            .context
            .as_ref()
            .ok_or_else(|| Refusal::unsupported("proposal configuration required"))?;
        let describe = Any::pack(&DescribeActivityExecutionRequest {
            namespace: context.start.namespace.clone(),
            activity_id: context.start.activity_id.clone(),
            run_id: context.run_id.clone(),
            ..Default::default()
        });
        let mut eligible = vec![(
            Action::from(Client::Cancel),
            Any::pack(&RequestCancelActivityExecutionRequest {
                namespace: context.start.namespace.clone(),
                activity_id: context.start.activity_id.clone(),
                run_id: context.run_id.clone(),
                ..Default::default()
            }),
        )];
        if state.phase == Phase::Scheduled {
            eligible.push((
                Worker::Poll(state.attempt).into(),
                Any::pack(&umpire_temporal_api::umpire::temporal::v1::ObserveActivity {
                    request: Some(describe.clone()),
                    poll_interval_ms: 10,
                    read_timeout_ms: 30_000,
                    backoff_ms: 10,
                }),
            ));
        }
        if let Some((task, response)) = &context.response
            && task.attempt == i32::from(state.attempt)
            && state.phase == Phase::Started
        {
            let reply = Self::respond(task.clone(), state)?;
            if reply.request == *response {
                eligible.push((
                    reply.action.into(),
                    Any::pack(&umpire_temporal_api::umpire::temporal::v1::ObserveActivity {
                        request: Some(describe),
                        poll_interval_ms: 10,
                        read_timeout_ms: 30_000,
                        backoff_ms: 10,
                    }),
                ));
            }
        }
        let attempts = context.start.retry_policy.as_ref().map_or(0, |p| p.maximum_attempts);
        let maximum = u8::try_from(attempts)
            .map_err(|_| Refusal::unsupported("invalid configured attempts"))?;
        let world = activity::World::new(maximum);
        eligible.retain(|(action, _)| {
            !world.internal(action)
                && world.checked_step(state, action.clone()).is_ok_and(|steps| !steps.is_empty())
        });
        if eligible.is_empty() {
            return Err(Refusal::new(
                umpire_runtime::protocol::RefusalCode::Quiescent,
                "no external action",
            ));
        }
        let index = usize::try_from(seed % eligible.len() as u64)
            .map_err(|_| Refusal::malformed("seed selection overflow"))?;
        let (action, request) = eligible.swap_remove(index);
        Ok(Proposal::Action { action, request })
    }

    fn matches_proposal(&self, request: &Any, observation: &Any) -> Result<(), Refusal> {
        let context = self
            .context
            .as_ref()
            .ok_or_else(|| Refusal::unsupported("proposal configuration required"))?;
        let observation: umpire_runtime::realization::Observation = observation.unpack()?;
        let actual =
            observation.request.ok_or_else(|| Refusal::malformed("actual request is missing"))?;
        if request.type_url == <RequestCancelActivityExecutionRequest as prost::Name>::type_url()
            && observation.method
                == "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
            && actual == *request
        {
            return Ok(());
        }
        if observation.method
            != "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"
        {
            return Err(Refusal::new(
                umpire_runtime::protocol::RefusalCode::State,
                "observation RPC does not correlate with proposal",
            ));
        }
        let actual: DescribeActivityExecutionRequest = actual.unpack()?;
        if actual.namespace != context.start.namespace
            || actual.activity_id != context.start.activity_id
            || actual.run_id != context.run_id
        {
            return Err(Refusal::new(
                umpire_runtime::protocol::RefusalCode::State,
                "observation crosses proposal identity",
            ));
        }
        if request.type_url
            == <umpire_temporal_api::umpire::temporal::v1::ObserveActivity as prost::Name>::type_url(
            )
        {
            let instruction: umpire_temporal_api::umpire::temporal::v1::ObserveActivity =
                request.unpack()?;
            let expected: DescribeActivityExecutionRequest =
                Any::required(instruction.request)?.unpack()?;
            if expected != actual {
                return Err(Refusal::new(
                    umpire_runtime::protocol::RefusalCode::State,
                    "Describe differs from proposal",
                ));
            }
        } else if request.type_url
            != <RequestCancelActivityExecutionRequest as prost::Name>::type_url()
        {
            return Err(Refusal::unsupported("unknown proposal request"));
        }
        Ok(())
    }

    fn polling(
        &self,
        observation: &Any,
        state: &activity::Model,
        action: Option<&Action>,
    ) -> Result<bool, Refusal> {
        if self.context.is_none() || action.is_none() {
            return Ok(false);
        }
        let observed = umpire_runtime::Observed::new(observation)?;
        if observed.method
            != "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"
            || umpire_dsl::Phase::closed(&state.phase)
        {
            return Ok(false);
        }
        for attempt in 1..=3 {
            if action == Some(&Worker::Poll(attempt).into()) {
                return Ok(state.phase != Phase::Started || state.attempt != attempt);
            }
        }
        if action == Some(&Worker::RespondFailed { retryable: true }.into()) {
            let context = self
                .context
                .as_ref()
                .ok_or_else(|| Refusal::unsupported("proposal context required"))?;
            if let Some((task, _)) = &context.response
                && state.phase == Phase::Scheduled
                && i32::from(state.attempt) == task.attempt + 1
            {
                return Ok(false);
            }
        }
        Ok(true)
    }

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
