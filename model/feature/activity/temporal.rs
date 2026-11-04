//! Temporal protobuf realization for the independently authored mini activity Model.

// The fixture runner compiles the Model independently of this realization.
#[path = "model.rs"]
mod included;
mod phase;

pub use included::activity;

use activity::{Client, Phase, Worker};
use umpire_dsl::{Machine, Outcome};
use umpire_runtime::{Candidate, Projection, Proposal, prost_types::Any, protocol::Refusal};
use umpire_temporal_api::temporal::api::{
    activity::v1::{ActivityExecutionOutcome, activity_execution_outcome::Value},
    common::v1::{Payload, Payloads},
    failure::v1::Failure,
    taskqueue::v1::TaskQueue,
    workflowservice::v1::{
        DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        PollActivityTaskQueueRequest, PollActivityTaskQueueResponse,
        RecordActivityTaskHeartbeatRequest, RecordActivityTaskHeartbeatResponse,
        RequestCancelActivityExecutionRequest, RequestCancelActivityExecutionResponse,
        RespondActivityTaskCanceledRequest, RespondActivityTaskCanceledResponse,
        RespondActivityTaskCompletedRequest, RespondActivityTaskCompletedResponse,
        RespondActivityTaskFailedRequest, RespondActivityTaskFailedResponse,
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
    umpire_runtime::Runtime::new(
        activity::World::new(3, None, None, None, None),
        Realization::default(),
    )
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
    pub(crate) fn worker_program(
        start: &StartActivityExecutionRequest,
        world: &activity::World,
    ) -> Result<Any, Refusal> {
        use umpire_temporal_api::umpire::temporal::v1::{ActivityTaskKey, Rule};
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
                let worker = Self::select_response(scenario, attempt)?;
                let action = Action::from(worker);
                let binding = Self::default();
                let identity =
                    umpire_runtime::capability::Identity::identify(&binding, &Any::pack(start))?;
                let response = umpire_runtime::capability::Request::pack_request(
                    &umpire_runtime::capability::Fill::fill(
                        &binding,
                        &action,
                        world,
                        &Self::program_inputs(
                            worker,
                            u8::try_from(attempt).map_err(|_| {
                                Refusal::unsupported("worker attempt is out of range")
                            })?,
                        )?,
                        &identity,
                    )?,
                );
                rules.push(Rule {
                    key: Some(Any::pack(&ActivityTaskKey {
                        scenario: scenario.into(),
                        attempt,
                        activity_type: activity_type.name.clone(),
                        heartbeat_history: Vec::new(),
                    })),
                    respond: vec![response],
                });
            }
        }
        Ok(Any::pack(&crate::support::worker_program(
            start.namespace.clone(),
            task_queue.name.clone(),
            rules,
            Vec::new(),
        )))
    }
}

#[realization(activity)]
impl Realization {
    #[propose(matches = matches_proposal, continue_polling = polling)]
    fn proposal(
        &self,
        world: &activity::World,
        state: &activity::Model,
        seed: u64,
    ) -> Result<Proposal<Action>, Refusal> {
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
        let read = Any::pack(&crate::support::scheduled_read(describe, 10, 30_000, 10));
        if !umpire_dsl::Phase::closed(&state.phase) && seed % 2 == 1 {
            return Ok(Proposal::Read { request: read });
        }
        let hold = state.phase == Phase::Scheduled && seed & 2 != 0;
        if !umpire_dsl::Phase::closed(&state.phase) && !hold {
            return Err(Refusal::unsupported(
                "unheld cancellation of an open activity is outside the atomic cancellation Model",
            ));
        }
        let action = Action::from(Client::Cancel);
        let identity =
            umpire_runtime::capability::Identity::identify(self, &Any::pack(&context.start))?;
        let request = umpire_runtime::capability::Fill::fill(
            self,
            &action,
            world,
            &FillInputs::Cancel { request_id: String::new() },
            &identity,
        )?;
        let instruction = crate::support::instruction(
            umpire_runtime::capability::Request::pack_request(&request),
            hold,
            read,
        );
        Ok(Proposal::Action { action, request: Any::pack(&instruction) })
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
            if !Self::consistent_evidence(
                phase,
                attempt,
                info.last_failure.as_ref(),
                response.outcome.as_ref(),
            ) {
                return Ok(Projection::steps([]));
            }
            if attempt > 1 && phase != Phase::Failed {
                let (action, expected) = self.response(world, scenario, i32::from(attempt - 1))?;
                if !matches!(action, Worker::RespondFailed { retryable: true }) {
                    return Ok(Projection::steps([]));
                }
                let expected: RespondActivityTaskFailedRequest = expected.unpack()?;
                if !Self::same_failure(info.last_failure.as_ref(), expected.failure.as_ref()) {
                    return Ok(Projection::steps([]));
                }
            }
            for skipped in state.attempt..attempt {
                let (action, _) = self.response(world, scenario, i32::from(skipped))?;
                if !matches!(action, Worker::RespondFailed { retryable: true }) {
                    return Ok(Projection::steps([]));
                }
            }
            let (action, expected) = self.response(world, scenario, raw_attempt)?;
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
                    if !Self::same_failure(Some(failure), expected.failure.as_ref())
                        || !Self::same_failure(
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

    fn select_response(scenario: &str, attempt: i32) -> Result<Worker, Refusal> {
        match scenario {
            "RetryThenComplete" if attempt == 1 => Ok(Worker::RespondFailed { retryable: true }),
            "Complete" | "RetryThenComplete" => Ok(Worker::RespondCompleted),
            "Fail" => Ok(Worker::RespondFailed { retryable: false }),
            _ => Err(Refusal::unsupported("unsupported worker scenario")),
        }
    }

    fn response(
        &self,
        world: &activity::World,
        scenario: &str,
        attempt: i32,
    ) -> Result<(Worker, Any), Refusal> {
        let action = Self::select_response(scenario, attempt)?;
        let context = self
            .context
            .as_ref()
            .ok_or_else(|| Refusal::unsupported("proposal configuration required"))?;
        let identity =
            umpire_runtime::capability::Identity::identify(self, &Any::pack(&context.start))?;
        let request = umpire_runtime::capability::Fill::fill(
            self,
            &Action::from(action),
            world,
            &Self::program_inputs(
                action,
                u8::try_from(attempt)
                    .map_err(|_| Refusal::unsupported("worker attempt is out of range"))?,
            )?,
            &identity,
        )?;
        Ok((action, umpire_runtime::capability::Request::pack_request(&request)))
    }
}

/// Actual activity identity admitted from the captured Start request and run context.
pub struct ActivityIdentity {
    namespace: String,
    activity_id: String,
    run_id: String,
    task_queue: String,
    /// Actual worker task token; empty when the program executor assigns it.
    task_token: Vec<u8>,
    /// Actual caller identity; empty when the executing host assigns it.
    caller: String,
}

impl ActivityIdentity {
    /// An actual live task and caller identity, as captured by a probe prerequisite.
    pub(crate) fn task(
        namespace: String,
        activity_id: String,
        run_id: String,
        task_queue: String,
        task_token: Vec<u8>,
        caller: String,
    ) -> Self {
        Self { namespace, activity_id, run_id, task_queue, task_token, caller }
    }
}

/// Explicit action-specific request inputs selected by policy.
pub enum FillInputs {
    /// Client cancellation of the admitted run; hold and read belong to the proposal.
    Cancel {
        /// Caller-selected request ID; empty when host annotation fills it.
        request_id: String,
    },
    /// A worker poll, heartbeat or cancellation acknowledgment.
    Worker {
        /// Actual admitted attempt used by cancellation acknowledgment.
        attempt: u8,
        /// Opaque heartbeat or cancellation details selected by policy.
        details: Option<Payloads>,
    },
    /// A worker completion and its policy-selected result.
    Completed {
        /// Actual admitted attempt.
        attempt: u8,
        /// Opaque result selected by policy.
        result: Option<Payloads>,
    },
    /// A worker failure whose retryability must agree with the action.
    Failed {
        /// Actual admitted attempt.
        attempt: u8,
        /// Application failure selected by policy.
        failure: Box<Failure>,
    },
}

impl umpire_runtime::capability::Identity for Realization {
    type Id = ActivityIdentity;
    fn identify(&self, value: &Any) -> Result<Self::Id, Refusal> {
        let actual: StartActivityExecutionRequest = value.unpack()?;
        if actual.namespace.is_empty() {
            return Err(Refusal::malformed("program namespace is required"));
        }
        let run_id = if let Some(context) = &self.context {
            if context.start != actual {
                return Err(Refusal::new(
                    umpire_runtime::protocol::RefusalCode::State,
                    "filler crosses admitted Start identity",
                ));
            }
            context.run_id.clone()
        } else {
            String::new()
        };
        Ok(ActivityIdentity {
            task_queue: actual.task_queue.map(|queue| queue.name).unwrap_or_default(),
            namespace: actual.namespace,
            activity_id: actual.activity_id,
            run_id,
            task_token: Vec::new(),
            caller: String::new(),
        })
    }
}

/// A complete checked witness prefix, matched against its last edge.
type Witness<'a> = [&'a umpire_runtime::Evidence<activity::World>];

activity::action_metadata!(crate::registry; activity; {
    #[registry(inputs = FillInputs)]
    #[allow(
        clippy::trivially_copy_pass_by_ref,
        reason = "registry fillers receive the actual action by reference"
    )]
    impl Realization {
        fn fill_cancel(
            &self,
            _: &Client,
            _: &activity::World,
            inputs: &FillInputs,
            identity: &ActivityIdentity,
        ) -> Result<RequestCancelActivityExecutionRequest, Refusal> {
            Self::admitted(identity)?;
            let FillInputs::Cancel { request_id } = inputs else {
                return Err(Refusal::unsupported("action has no activity request for these inputs"));
            };
            if identity.activity_id.is_empty() || identity.run_id.is_empty() {
                return Err(Refusal::malformed("cancellation identity is missing"));
            }
            Ok(RequestCancelActivityExecutionRequest {
                namespace: identity.namespace.clone(),
                activity_id: identity.activity_id.clone(),
                run_id: identity.run_id.clone(),
                identity: identity.caller.clone(),
                request_id: request_id.clone(),
                ..Default::default()
            })
        }

        fn fill_poll(
            &self,
            action: &Worker,
            world: &activity::World,
            inputs: &FillInputs,
            identity: &ActivityIdentity,
        ) -> Result<PollActivityTaskQueueRequest, Refusal> {
            let FillInputs::Worker { attempt, details: None } = inputs else {
                return Err(Refusal::unsupported("action has no activity request for these inputs"));
            };
            Self::attempt(world, identity, *attempt)?;
            if *action != Worker::Poll(*attempt) || identity.task_queue.is_empty() {
                return Err(Refusal::unsupported("action has no activity request for these inputs"));
            }
            Ok(PollActivityTaskQueueRequest {
                namespace: identity.namespace.clone(),
                task_queue: Some(TaskQueue { name: identity.task_queue.clone(), ..Default::default() }),
                identity: identity.caller.clone(),
                ..Default::default()
            })
        }

        fn fill_completed(
            &self,
            _: &Worker,
            world: &activity::World,
            inputs: &FillInputs,
            identity: &ActivityIdentity,
        ) -> Result<RespondActivityTaskCompletedRequest, Refusal> {
            let FillInputs::Completed { attempt, result } = inputs else {
                return Err(Refusal::unsupported("action has no activity request for these inputs"));
            };
            Self::attempt(world, identity, *attempt)?;
            let (namespace, task_token) = Self::task(identity);
            Ok(RespondActivityTaskCompletedRequest {
                namespace,
                task_token,
                identity: identity.caller.clone(),
                result: result.clone(),
                ..Default::default()
            })
        }

        fn fill_failed(
            &self,
            action: &Worker,
            world: &activity::World,
            inputs: &FillInputs,
            identity: &ActivityIdentity,
        ) -> Result<RespondActivityTaskFailedRequest, Refusal> {
            let (FillInputs::Failed { attempt, failure }, Worker::RespondFailed { retryable }) =
                (inputs, *action)
            else {
                return Err(Refusal::unsupported("action has no activity request for these inputs"));
            };
            Self::attempt(world, identity, *attempt)?;
            if Self::retryable(failure) != Some(retryable) {
                return Err(Refusal::unsupported("failure retryability contradicts the action"));
            }
            let (namespace, task_token) = Self::task(identity);
            Ok(RespondActivityTaskFailedRequest {
                namespace,
                task_token,
                identity: identity.caller.clone(),
                failure: Some(failure.as_ref().clone()),
                ..Default::default()
            })
        }

        fn fill_heartbeat(
            &self,
            _: &Worker,
            world: &activity::World,
            inputs: &FillInputs,
            identity: &ActivityIdentity,
        ) -> Result<RecordActivityTaskHeartbeatRequest, Refusal> {
            let FillInputs::Worker { attempt, details } = inputs else {
                return Err(Refusal::unsupported("action has no activity request for these inputs"));
            };
            Self::attempt(world, identity, *attempt)?;
            let (namespace, task_token) = Self::task(identity);
            Ok(RecordActivityTaskHeartbeatRequest {
                namespace,
                task_token,
                identity: identity.caller.clone(),
                details: details.clone(),
                ..Default::default()
            })
        }

        fn fill_canceled(
            &self,
            action: &Worker,
            world: &activity::World,
            inputs: &FillInputs,
            identity: &ActivityIdentity,
        ) -> Result<RespondActivityTaskCanceledRequest, Refusal> {
            let FillInputs::Worker { attempt, details } = inputs else {
                return Err(Refusal::unsupported("action has no activity request for these inputs"));
            };
            Self::attempt(world, identity, *attempt)?;
            if *action != Worker::RespondCanceled(*attempt) {
                return Err(Refusal::unsupported("action has no activity request for these inputs"));
            }
            let (namespace, task_token) = Self::task(identity);
            Ok(RespondActivityTaskCanceledRequest {
                namespace,
                task_token,
                identity: identity.caller.clone(),
                details: details.clone(),
                ..Default::default()
            })
        }

        fn cancel_evidence(
            &self,
            request: RequestCancelActivityExecutionRequest,
            response: Result<RequestCancelActivityExecutionResponse, i32>,
            path: &Witness<'_>,
        ) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
            self.correlated(&request.namespace, Some(&request.activity_id), Some(&request.run_id))?;
            Ok(Self::edge(path, Self::outcome(&response), |action| {
                *action == Action::from(Client::Cancel)
            }))
        }

        fn poll_evidence(
            &self,
            request: PollActivityTaskQueueRequest,
            response: Result<PollActivityTaskQueueResponse, i32>,
            path: &Witness<'_>,
        ) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
            self.correlated(&request.namespace, None, None)?;
            let outcome = Self::outcome(&response);
            let delivered = if let Ok(task) = &response {
                self.correlated(&request.namespace, Some(&task.activity_id), None)?;
                if task.task_token.is_empty() {
                    return Ok(umpire_runtime::EvidenceMatch::Reject);
                }
                Some(task.attempt)
            } else {
                None
            };
            Ok(Self::edge(path, outcome, |action| {
                matches!(action, Action::Worker(Worker::Poll(attempt))
                    if delivered.is_none_or(|delivered| i32::from(*attempt) == delivered))
            }))
        }

        fn completed_evidence(
            &self,
            request: RespondActivityTaskCompletedRequest,
            response: Result<RespondActivityTaskCompletedResponse, i32>,
            path: &Witness<'_>,
        ) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
            self.correlated(&request.namespace, None, None)?;
            Ok(Self::edge(path, Self::outcome(&response), |action| {
                *action == Action::from(Worker::RespondCompleted)
            }))
        }

        fn failed_evidence(
            &self,
            request: RespondActivityTaskFailedRequest,
            response: Result<RespondActivityTaskFailedResponse, i32>,
            path: &Witness<'_>,
        ) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
            self.correlated(&request.namespace, None, None)?;
            let Some(retryable) = request.failure.as_ref().and_then(Self::retryable) else {
                return Ok(umpire_runtime::EvidenceMatch::Reject);
            };
            Ok(Self::edge(path, Self::outcome(&response), |action| {
                *action == Action::from(Worker::RespondFailed { retryable })
            }))
        }

        fn heartbeat_evidence(
            &self,
            request: RecordActivityTaskHeartbeatRequest,
            response: Result<RecordActivityTaskHeartbeatResponse, i32>,
            path: &Witness<'_>,
        ) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
            self.correlated(&request.namespace, None, None)?;
            let reported = if let Ok(reply) = &response {
                if reply.activity_paused || reply.activity_reset {
                    return Err(Refusal::unsupported("paused or reset activity is outside the Model"));
                }
                Some(reply.cancel_requested)
            } else {
                None
            };
            let consistent = path.last().is_some_and(|edge| {
                reported.is_none_or(|pending| edge.target.cancel_requested == pending)
            });
            Ok(Self::edge(path, Self::outcome(&response), |action| {
                consistent && *action == Action::from(Worker::Heartbeat)
            }))
        }

        fn canceled_evidence(
            &self,
            request: RespondActivityTaskCanceledRequest,
            response: Result<RespondActivityTaskCanceledResponse, i32>,
            path: &Witness<'_>,
        ) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
            self.correlated(&request.namespace, None, None)?;
            Ok(Self::edge(path, Self::outcome(&response), |action| {
                matches!(action, Action::Worker(Worker::RespondCanceled(_)))
            }))
        }

        /// No observation evidences a Tick directly; existential admission of unobserved
        /// interleavings belongs to the declared refinement.
        fn tick_evidence(
            &self,
            _: &crate::support::Observed,
            _: &Witness<'_>,
        ) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
            Ok(umpire_runtime::EvidenceMatch::Reject)
        }
    }
});

impl Realization {
    fn admitted(identity: &ActivityIdentity) -> Result<(), Refusal> {
        if identity.namespace.is_empty() {
            return Err(Refusal::malformed("program namespace is required"));
        }
        Ok(())
    }

    fn attempt(
        world: &activity::World,
        identity: &ActivityIdentity,
        attempt: u8,
    ) -> Result<(), Refusal> {
        Self::admitted(identity)?;
        if !(1..=world.maximum_attempts).contains(&attempt) {
            return Err(Refusal::unsupported("worker attempt exceeds World.maximum_attempts"));
        }
        Ok(())
    }

    /// An application failure's declared retryability; other failures have none.
    fn retryable(failure: &Failure) -> Option<bool> {
        use umpire_temporal_api::temporal::api::failure::v1::failure::FailureInfo;
        match &failure.failure_info {
            Some(FailureInfo::ApplicationFailureInfo(info)) => Some(!info.non_retryable),
            _ => None,
        }
    }

    /// Policy-selected inputs for one programmed worker response.
    fn program_inputs(action: Worker, attempt: u8) -> Result<FillInputs, Refusal> {
        Ok(match action {
            Worker::RespondCompleted => FillInputs::Completed {
                attempt,
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
            },
            Worker::RespondFailed { retryable } => FillInputs::Failed {
                attempt,
                failure: Box::new(if retryable {
                    Failure::application("MiniActivityFailure", "Retry", false)
                } else {
                    Failure::application("MiniActivityFailure", "Fail", true)
                }),
            },
            _ => return Err(Refusal::unsupported("unsupported programmed worker response")),
        })
    }

    /// The actual task identity, or none when the program executor assigns it.
    fn task(identity: &ActivityIdentity) -> (String, Vec<u8>) {
        if identity.task_token.is_empty() {
            (String::new(), Vec::new())
        } else {
            (identity.namespace.clone(), identity.task_token.clone())
        }
    }

    /// Refuse evidence whose namespace, or any supplied activity or run, crosses the admitted Start.
    fn correlated(
        &self,
        namespace: &str,
        activity_id: Option<&str>,
        run_id: Option<&str>,
    ) -> Result<(), Refusal> {
        let context = self
            .context
            .as_ref()
            .ok_or_else(|| Refusal::unsupported("evidence correlation requires admitted Start"))?;
        let crossed = namespace != context.start.namespace
            || activity_id.is_some_and(|activity_id| activity_id != context.start.activity_id)
            || run_id.is_some_and(|run_id| run_id != context.run_id);
        if crossed {
            return Err(Refusal::new(
                umpire_runtime::protocol::RefusalCode::State,
                "evidence crosses admitted activity identity",
            ));
        }
        Ok(())
    }

    /// The Model outcome evidenced by an RPC result; unmapped status codes have none.
    fn outcome<T>(response: &Result<T, i32>) -> Option<Outcome> {
        match response {
            Ok(_) => Some(Outcome::Accepted),
            Err(code) => rejection(*code).map(Outcome::Rejected),
        }
    }

    /// Match the complete witness prefix's last edge; RPC evidence never admits a root.
    fn edge(
        path: &Witness<'_>,
        outcome: Option<Outcome>,
        action: impl Fn(&Action) -> bool,
    ) -> umpire_runtime::EvidenceMatch {
        match (path.last(), outcome) {
            (Some(edge), Some(outcome)) if action(&edge.action) && edge.outcome == outcome => {
                umpire_runtime::EvidenceMatch::Match { cursor: path.len() }
            }
            _ => umpire_runtime::EvidenceMatch::Reject,
        }
    }
}

impl Realization {
    /// Derives successful next heartbeat bits from complete compatible Model histories.
    ///
    /// # Errors
    /// Refuses invalid attempts, impossible histories, Model errors or exhausted search bounds.
    pub fn heartbeat_successors(
        world: &activity::World,
        attempt: u8,
        history: &[bool],
        max_depth: usize,
        max_work: usize,
    ) -> Result<Vec<bool>, Refusal> {
        use std::{
            collections::{HashSet, hash_map::DefaultHasher},
            hash::BuildHasherDefault,
        };
        use umpire_checker::search::Traversal;
        Self::heartbeat_world(world, attempt)?;
        let catalog = world
            .checked_catalog(umpire_dsl::CATALOG_LIMIT)
            .map_err(|error| Refusal::unsupported(format!("heartbeat Model catalog: {error}")))?;
        let mut search = Traversal::<_, ()>::new(max_depth, max_work);
        let mut visited = HashSet::<_, BuildHasherDefault<DefaultHasher>>::default();
        for state in world.starts() {
            let position = (state, 0_usize, false);
            visited
                .try_reserve(1)
                .map_err(|_| Refusal::unsupported("heartbeat successor allocation bound"))?;
            if visited.insert(position.clone()) {
                search.root(position).map_err(Self::heartbeat_search_limit)?;
            }
        }
        let mut bits = [false; 2];
        while let Some(parent) = search.next() {
            let (state, cursor, assigned) = search.label(parent).position.clone();
            if assigned && cursor > history.len() {
                bits[usize::from(state.cancel_requested)] = true;
                continue;
            }
            if state.attempt > attempt || umpire_dsl::Phase::closed(&state.phase) {
                continue;
            }
            for action in catalog.actions() {
                let heartbeat = *action == Action::from(Worker::Heartbeat);
                if assigned && matches!(action, Action::Worker(_)) && !heartbeat {
                    continue;
                }
                search.charge().map_err(Self::heartbeat_search_limit)?;
                let choices = world.checked_step(&state, action.clone()).map_err(|error| {
                    Refusal::unsupported(format!("heartbeat Model step: {error}"))
                })?;
                for (outcome, _, next) in choices {
                    if outcome != Outcome::Accepted {
                        continue;
                    }
                    if assigned && (next.phase != Phase::Started || next.attempt != attempt) {
                        continue;
                    }
                    let mut next_cursor = cursor;
                    if assigned && heartbeat {
                        if cursor < history.len() && history[cursor] != next.cancel_requested {
                            continue;
                        }
                        next_cursor += 1;
                    }
                    let next_assigned = assigned
                        || (*action == Action::from(Worker::Poll(attempt))
                            && next.phase == Phase::Started
                            && next.attempt == attempt);
                    let position = (next, next_cursor, next_assigned);
                    if visited.contains(&position) {
                        continue;
                    }
                    visited.try_reserve(1).map_err(|_| {
                        Refusal::unsupported("heartbeat successor allocation bound")
                    })?;
                    visited.insert(position.clone());
                    search
                        .push(parent, position, Some(()))
                        .map_err(Self::heartbeat_search_limit)?;
                }
            }
        }
        let successors: Vec<_> =
            [false, true].into_iter().filter(|bit| bits[usize::from(*bit)]).collect();
        if successors.is_empty() {
            return Err(Refusal::unsupported(
                "heartbeat_history has no successful Model continuation",
            ));
        }
        Ok(successors)
    }

    /// Checks a supplied observable successor declaration independently of its response rows.
    ///
    /// # Errors
    /// Refuses malformed keys, missing/duplicate/impossible bits, or incomplete Model search.
    pub fn validate_heartbeat_successors(
        world: &activity::World,
        declaration: &Any,
        max_depth: usize,
        max_work: usize,
    ) -> Result<(), Refusal> {
        let declaration: umpire_temporal_api::umpire::temporal::v1::ActivityTaskSuccessors =
            declaration.unpack()?;
        let key = declaration
            .key
            .as_ref()
            .ok_or_else(|| Refusal::malformed("heartbeat successor key is missing"))?;
        if key.scenario.is_empty() || key.activity_type.is_empty() {
            return Err(Refusal::malformed("heartbeat successor key identity is missing"));
        }
        let bits = &declaration.cancel_requested;
        if bits.is_empty() || bits.len() > 2 || (bits.len() == 2 && bits[0] == bits[1]) {
            return Err(Refusal::malformed(
                "heartbeat successor domain requires unique observable bits",
            ));
        }
        let attempt = u8::try_from(key.attempt)
            .map_err(|_| Refusal::unsupported("heartbeat attempt is outside World"))?;
        let expected = Self::heartbeat_successors(
            world,
            attempt,
            &key.heartbeat_history,
            max_depth,
            max_work,
        )?;
        let mut actual = declaration.cancel_requested.clone();
        actual.sort_unstable();
        if actual != expected {
            return Err(Refusal::unsupported(
                "observable successor domain differs from checked Model histories",
            ));
        }
        Ok(())
    }

    fn heartbeat_world(world: &activity::World, attempt: u8) -> Result<(), Refusal> {
        if !(1..=3).contains(&world.maximum_attempts)
            || !(1..=world.maximum_attempts).contains(&attempt)
        {
            return Err(Refusal::unsupported(
                "heartbeat attempt is outside World.maximum_attempts",
            ));
        }
        for (field, value) in [
            ("schedule_to_start", world.schedule_to_start),
            ("start_to_close", world.start_to_close),
            ("schedule_to_close", world.schedule_to_close),
            ("heartbeat_timeout", world.heartbeat_timeout),
        ] {
            if value.is_some_and(|ticks| !(1..=3).contains(&ticks.0)) {
                return Err(Refusal::unsupported(format!(
                    "heartbeat World.{field} is outside 1..=3"
                )));
            }
        }
        Ok(())
    }

    fn heartbeat_search_limit(error: umpire_checker::search::SearchLimit) -> Refusal {
        Refusal::unsupported(format!("heartbeat successor {error}"))
    }
}

#[cfg(test)]
#[path = "../../realization/tests/activity_registry/mod.rs"]
mod registry_controls;
