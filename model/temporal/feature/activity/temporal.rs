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
    activity::v1::activity_execution_outcome::Value,
    common::v1::{Payload, Payloads},
    failure::v1::Failure,
    workflowservice::v1::{
        DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        RequestCancelActivityExecutionRequest, RequestCancelActivityExecutionResponse,
        RespondActivityTaskCompletedRequest, RespondActivityTaskFailedRequest,
        StartActivityExecutionRequest,
    },
};

use crate::{
    realization,
    support::{ApplicationFailure, rejection},
};

type Action = <activity::World as Machine>::Action;

/// Creates one independently owned guest run.
pub fn runtime() -> umpire_runtime::Runtime<activity::World, Realization> {
    umpire_runtime::Runtime::new(activity::World::new(3), Realization::default())
}

/// The strict mini activity observation and worker program policy.
#[derive(Default)]
pub struct Realization {
    context: Option<ProposalContext>,
}

pub(crate) struct ProposalContext {
    pub start: StartActivityExecutionRequest,
    pub run_id: String,
}

impl Realization {
    pub(crate) fn configured(context: ProposalContext) -> Self {
        Self { context: Some(context) }
    }
}

/// Header carrying the worker's scenario independently of application input.
pub const SCENARIO_HEADER: &str = "umpire-scenario";

pub(crate) const SCENARIOS: [&str; 3] = ["Complete", "RetryThenComplete", "Fail"];

impl Realization {
    pub(crate) fn worker_program(start: &StartActivityExecutionRequest) -> Result<Any, Refusal> {
        use umpire_temporal_api::umpire::temporal::v1::{ActivityTaskKey, Rule, WorkerProgram};
        let maximum = start.retry_policy.as_ref().map_or(0, |policy| policy.maximum_attempts);
        if !(1..=3).contains(&maximum) {
            return Err(Refusal::unsupported(
                "retry_policy.maximum_attempts must be explicitly in 1..=3",
            ));
        }
        let activity_type = start
            .activity_type
            .as_ref()
            .filter(|value| !value.name.is_empty())
            .ok_or_else(|| Refusal::malformed("program activity_type is required"))?;
        let task_queue = start
            .task_queue
            .as_ref()
            .filter(|value| !value.name.is_empty())
            .ok_or_else(|| Refusal::malformed("program task_queue is required"))?;
        if start.namespace.is_empty() {
            return Err(Refusal::malformed("program namespace is required"));
        }
        let mut rules = Vec::new();
        for scenario in SCENARIOS {
            for attempt in 1..=maximum {
                let (_, response) = Self::response(scenario, attempt)?;
                rules.push(Rule {
                    key: Some(Any::pack(&ActivityTaskKey {
                        scenario: scenario.into(),
                        attempt,
                        activity_type: activity_type.name.clone(),
                    })),
                    respond: vec![response],
                });
            }
        }
        Ok(Any::pack(&WorkerProgram {
            namespace: start.namespace.clone(),
            task_queue: task_queue.name.clone(),
            rules,
        }))
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
            include_outcome: true,
            include_last_failure: true,
            ..Default::default()
        });
        let read = Any::pack(&umpire_temporal_api::umpire::temporal::v1::ObserveActivity {
            request: Some(describe),
            poll_interval_ms: 10,
            read_timeout_ms: 30_000,
            backoff_ms: 10,
        });
        if !umpire_dsl::Phase::closed(&state.phase) && seed % 2 == 1 {
            return Ok(Proposal::Read { request: read });
        }
        Ok(Proposal::Action {
            action: Client::Cancel.into(),
            request: Any::pack(&umpire_temporal_api::umpire::temporal::v1::TemporalInstruction {
                request: Some(Any::pack(&RequestCancelActivityExecutionRequest {
                    namespace: context.start.namespace.clone(),
                    activity_id: context.start.activity_id.clone(),
                    run_id: context.run_id.clone(),
                    ..Default::default()
                })),
                hold_worker_polls: state.phase == Phase::Scheduled && seed & 2 != 0,
                not_found_read: Some(read),
            }),
        })
    }

    fn matches_proposal(&self, request: &Any, observation: &Any) -> Result<(), Refusal> {
        let context = self
            .context
            .as_ref()
            .ok_or_else(|| Refusal::unsupported("proposal configuration required"))?;
        let observed = crate::support::Observed::new(observation)?;
        let observation = observed.current();
        let actual = observation
            .request
            .as_ref()
            .ok_or_else(|| Refusal::malformed("actual request is missing"))?;
        if request.type_url
            == <umpire_temporal_api::umpire::temporal::v1::TemporalInstruction as prost::Name>::type_url()
        {
            let instruction: umpire_temporal_api::umpire::temporal::v1::TemporalInstruction = request.unpack()?;
            if observation.method == "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution" {
                if observed.acknowledged().is_some() { return Err(Refusal::new(umpire_runtime::protocol::RefusalCode::State, "compound evidence requires the reconciliation read")); }
                return if instruction.request.as_ref() == Some(actual) {
                    Ok(())
                } else {
                    Err(Refusal::new(umpire_runtime::protocol::RefusalCode::State,
                        "observation does not match cancellation invocation"))
                };
            }
            if observation.method == "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution" {
                let read: umpire_temporal_api::umpire::temporal::v1::ObserveActivity =
                    Any::required(instruction.not_found_read)?.unpack()?;
                let acknowledged = observed.acknowledged().ok_or_else(|| Refusal::new(umpire_runtime::protocol::RefusalCode::State, "reconciliation requires the actual cancellation acknowledgment"))?;
                if acknowledged.method != "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
                    || acknowledged.request.as_ref() != instruction.request.as_ref()
                    || !matches!(&acknowledged.result, Some(umpire_runtime::realization::observation::Result::Error(error)) if error.code == 5)
                {
                    return Err(Refusal::new(umpire_runtime::protocol::RefusalCode::State, "reconciliation acknowledgment does not match cancellation NotFound"));
                }
                return if read.request.as_ref() == Some(actual) {
                    Ok(())
                } else {
                    Err(Refusal::new(umpire_runtime::protocol::RefusalCode::State,
                        "Describe differs from cancellation reconciliation read"))
                };
            }
            return Err(Refusal::new(umpire_runtime::protocol::RefusalCode::State,
                "observation does not match cancellation invocation"));
        }
        if observed.acknowledged().is_some() {
            return Err(Refusal::new(
                umpire_runtime::protocol::RefusalCode::State,
                "compound evidence requires a pending cancellation",
            ));
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
        } else {
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
        let observed = crate::support::Observed::new(observation)?;
        Ok(self.context.is_some()
            && observed.method
                == "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"
            && (action.is_some() || !umpire_dsl::Phase::closed(&state.phase)))
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
        reads(info.status, info.run_state, info.attempt, outcome, info.last_failure),
        ignores(
            run_id = "Run identity is outside the lifecycle projection",
            input = "Application input is independent of the worker scenario",
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
        &self,
        response: DescribeActivityExecutionResponse,
        state: &activity::Model,
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
        if let Some(context) = &self.context {
            let scenario = context
                .start
                .header
                .as_ref()
                .and_then(|header| header.fields.get(SCENARIO_HEADER))
                .and_then(|payload| std::str::from_utf8(&payload.data).ok())
                .ok_or_else(|| Refusal::malformed("Start scenario header is missing"))?;
            if let Some(failure) = info.last_failure.as_ref() {
                Self::programmed_failure(failure)?;
            }
            if let Some(Value::Failure(failure)) =
                response.outcome.as_ref().and_then(|outcome| outcome.value.as_ref())
            {
                Self::programmed_failure(failure)?;
            }
            if attempt > 1 {
                let (action, expected) = Self::response(scenario, i32::from(attempt - 1))?;
                if !matches!(action, Worker::RespondFailed { retryable: true }) {
                    return Ok(Projection::steps([]));
                }
                let expected: RespondActivityTaskFailedRequest = expected.unpack()?;
                if !Self::same_failure(info.last_failure.as_ref(), expected.failure.as_ref()) {
                    return Ok(Projection::steps([]));
                }
            }
            for skipped in state.attempt..attempt {
                let (action, _) = Self::response(scenario, i32::from(skipped))?;
                if !matches!(action, Worker::RespondFailed { retryable: true }) {
                    return Ok(Projection::steps([]));
                }
            }
            let (action, expected) = Self::response(scenario, raw_attempt)?;
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
                    if !Self::same_failure(Some(failure), expected.failure.as_ref()) {
                        return Ok(Projection::steps([]));
                    }
                }
                _ => {}
            }
        }
        Ok(Self::project_phase(state, &activity::Model { phase, attempt }, terminal_retryable))
    }

    fn programmed_failure(failure: &Failure) -> Result<(), Refusal> {
        use umpire_temporal_api::temporal::api::failure::v1::failure::FailureInfo;
        if matches!(&failure.failure_info, Some(FailureInfo::ApplicationFailureInfo(info)) if info.r#type == "umpire.unprogrammed")
        {
            return Err(Refusal::unsupported("unprogrammed worker task"));
        }
        Ok(())
    }

    fn same_failure(observed: Option<&Failure>, expected: Option<&Failure>) -> bool {
        fn content(failure: &Failure) -> Failure {
            let mut failure = failure.clone();
            failure.source.clear();
            failure.stack_trace.clear();
            failure.cause = failure.cause.map(|cause| Box::new(content(&cause)));
            failure
        }
        observed.map(content) == expected.map(content)
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

    fn response(scenario: &str, attempt: i32) -> Result<(Worker, Any), Refusal> {
        let response = match scenario {
            "RetryThenComplete" if attempt == 1 => (
                Worker::RespondFailed { retryable: true },
                Any::pack(&RespondActivityTaskFailedRequest {
                    failure: Some(Failure::application("MiniActivityFailure", "Retry", false)),
                    ..Default::default()
                }),
            ),
            "Complete" | "RetryThenComplete" => (
                Worker::RespondCompleted,
                Any::pack(&RespondActivityTaskCompletedRequest {
                    result: Some(Payloads {
                        payloads: vec![Payload {
                            metadata: std::collections::BTreeMap::from([(
                                "encoding".into(),
                                b"binary/plain".to_vec(),
                            )]),
                            data: b"Complete".to_vec(),
                            ..Default::default()
                        }],
                    }),
                    ..Default::default()
                }),
            ),
            "Fail" => (
                Worker::RespondFailed { retryable: false },
                Any::pack(&RespondActivityTaskFailedRequest {
                    failure: Some(Failure::application("MiniActivityFailure", "Fail", true)),
                    ..Default::default()
                }),
            ),
            _ => return Err(Refusal::unsupported("unsupported worker scenario")),
        };
        Ok(response)
    }
}
