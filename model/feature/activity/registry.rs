//! The activity action registry: typed fillers, evidence predicates and entity/read hooks.

use super::{Action, ActivityTarget, ProposalContext, Realization, activity};
use crate::support::{ApplicationFailure, rejection};
use crate::{
    activity_api::catalog,
    activity_protocol::{protocol, view::View},
    foundation::dc::Limits,
};
use activity::{Client, Worker};
use umpire_catalog::{Catalog, validate};
use umpire_dsl::Outcome;
use umpire_runtime::{
    AnyMessage as _,
    prost_types::Any,
    protocol::{Refusal, RefusalCode},
};
use umpire_temporal_api::temporal::api::common::v1::link;
use umpire_temporal_api::temporal::api::{
    activity::v1::activity_execution_outcome::Value,
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
        StartActivityExecutionRequest, StartActivityExecutionResponse,
    },
};

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
                    RefusalCode::State,
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
    #[entity(rpc = WorkflowService::StartActivityExecution,
        request = StartActivityExecutionRequest, response = StartActivityExecutionResponse,
        receipt = ProposalContext, admit = admit_start, make_receipt = start_receipt,
        correlate = correlate_start)]
    #[read(rpc = WorkflowService::DescribeActivityExecution,
        request = DescribeActivityExecutionRequest, response = DescribeActivityExecutionResponse,
        target = ActivityTarget, fill = describe_request, project = describe_target,
        finality = is_final, continue_polling = polls, correlate = correlate_describe,
        response_reads(info.status, info.run_state, info.attempt, info.last_failure, outcome))]
    #[observed(rpc = WorkflowService::PollActivityExecution, reason = "SDK activity handle outcome")]
    #[refines(protocol = crate::activity_protocol::protocol, state = refined_state,
        action = refined_action, outcome = refined_outcome, fact = refined_fact,
        view = refined_view)]
    #[allow(
        clippy::trivially_copy_pass_by_ref,
        reason = "registry fillers receive the actual action by reference"
    )]
    impl Realization {
        fn admit_start(
            &self,
            request: &StartActivityExecutionRequest,
            response: &StartActivityExecutionResponse,
        ) -> Result<ActivityIdentity, Refusal> {
            validate_start(request, response)?;
            if !response.started {
                return Err(state("initial Start must create a new run"));
            }
            Ok(ActivityIdentity {
                namespace: request.namespace.clone(),
                activity_id: request.activity_id.clone(),
                run_id: response.run_id.clone(),
                task_queue: request.task_queue.as_ref().map(|queue| queue.name.clone()).unwrap_or_default(),
                task_token: Vec::new(),
                caller: String::new(),
            })
        }

        fn start_receipt(
            &self,
            identity: &ActivityIdentity,
            request: &StartActivityExecutionRequest,
            _: &StartActivityExecutionResponse,
        ) -> Result<ProposalContext, Refusal> {
            Ok(ProposalContext { start: request.clone(), run_id: identity.run_id.clone() })
        }

        fn correlate_start(
            &self,
            receipt: &ProposalContext,
            request: &StartActivityExecutionRequest,
            response: &StartActivityExecutionResponse,
        ) -> Result<(), Refusal> {
            validate_start(request, response)?;
            receipt.matches(&request.namespace, &request.activity_id, &response.run_id)?;
            if receipt.start.request_id != request.request_id {
                return Err(state("Start request contradicts the original receipt"));
            }
            Ok(())
        }

        fn describe_request(
            &self,
            _: &activity::World,
            receipt: &ProposalContext,
        ) -> Result<DescribeActivityExecutionRequest, Refusal> {
            Ok(DescribeActivityExecutionRequest {
                namespace: receipt.start.namespace.clone(),
                activity_id: receipt.start.activity_id.clone(),
                run_id: receipt.run_id.clone(),
                include_outcome: true,
                include_last_failure: true,
                ..Default::default()
            })
        }

        fn correlate_describe(
            &self,
            receipt: &ProposalContext,
            request: &DescribeActivityExecutionRequest,
            response: &DescribeActivityExecutionResponse,
        ) -> Result<(), Refusal> {
            let info = response.info.as_ref().ok_or_else(|| Refusal::malformed("activity info is missing"))?;
            receipt.matches(&request.namespace, &request.activity_id, &request.run_id)?;
            receipt.matches(&request.namespace, &info.activity_id, &info.run_id)?;
            receipt.matches(&request.namespace, &request.activity_id, &response.run_id)?;
            if info.attempt < 1
                || receipt.maximum_attempts().is_some_and(|maximum| info.attempt > i32::from(maximum))
            {
                return Err(state("Describe attempt is outside the admitted attempts"));
            }
            Ok(())
        }

        fn describe_target(&self, response: &DescribeActivityExecutionResponse) -> Result<ActivityTarget, Refusal> {
            ActivityTarget::decode(response)
        }

        fn refined_state(state: &protocol::Model) -> Option<activity::Model> {
            super::projection::feature_state(state)
        }

        fn refined_action(action: &protocol::Action) -> Action {
            super::projection::feature_action(action)
        }

        fn refined_outcome(outcome: &Outcome) -> Outcome {
            *outcome
        }

        fn refined_fact(fact: &protocol::Expired) -> activity::ExpiredDeadline {
            super::projection::feature_fact(*fact)
        }

        fn refined_view(state: &protocol::Model) -> ActivityTarget {
            ActivityTarget::observe(&View::from(state))
        }

        fn is_final(&self, target: &ActivityTarget) -> bool {
            target.closed()
        }

        fn polls(&self, target: &ActivityTarget) -> bool {
            !target.closed()
        }

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
            Ok(Self::edge(path, Self::outcome(&response)?, |action| {
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
            let outcome = Self::outcome(&response)?;
            let delivered = if let Ok(task) = &response {
                // An empty long-poll delivers no task and evidences no Poll edge.
                if task.task_token.is_empty() {
                    return Ok(umpire_runtime::EvidenceMatch::Reject);
                }
                self.correlated(&request.namespace, Some(&task.activity_id), None)?;
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
            Ok(Self::edge(path, Self::outcome(&response)?, |action| {
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
                return Err(Refusal::unsupported(
                    "worker failure without application retryability is outside the Model",
                ));
            };
            Ok(Self::edge(path, Self::outcome(&response)?, |action| {
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
            Ok(Self::edge(path, Self::outcome(&response)?, |action| {
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
            Ok(Self::edge(path, Self::outcome(&response)?, |action| {
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

const CANCEL: &str =
    "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution";
const DESCRIBE: &str = "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution";

/// The live Reach path: every observation projects to the read's typed target or an unobserved
/// target, and only the registry's cancellation predicate or the Describe snapshot reconciled
/// with the Model-derived worker program authorizes an edge.
///
/// Worker RPC predicates stay declared but off this path: the guest receives no worker RPC
/// observations, and no receipt holds the polled task token they would need to correlate.
impl Realization {
    /// Methods this realization projects through Reach, in coverage order.
    pub const REACH: [&'static str; 2] = [CANCEL, DESCRIBE];

    pub(super) fn reach_projection(
        &self,
        _: &activity::World,
        observation: &Any,
        state: &activity::Model,
    ) -> Result<umpire_runtime::Projection<Action, Outcome, ActivityTarget>, Refusal> {
        let observed = crate::support::Observed::new(observation)?;
        match observed.method.as_str() {
            CANCEL => {
                if let Some(context) = &self.context {
                    let request: RequestCancelActivityExecutionRequest = observed
                        .request()?
                        .ok_or_else(|| Refusal::malformed("actual request is missing"))?;
                    context.matches(&request.namespace, &request.activity_id, &request.run_id)?;
                }
                let response = if let Some(code) = observed.code()? {
                    Err(code)
                } else {
                    let response: RequestCancelActivityExecutionResponse =
                        observed.response().map_err(|error| Refusal::malformed(error.message))?;
                    Ok(response)
                };
                Self::outcome(&response)?;
                Ok(umpire_runtime::Projection::Reach(ActivityTarget::unobserved()))
            }
            DESCRIBE => {
                let response: DescribeActivityExecutionResponse =
                    observed.response().map_err(|error| Refusal::malformed(error.message))?;
                let info = response
                    .info
                    .as_ref()
                    .ok_or_else(|| Refusal::malformed("activity info is missing"))?;
                let phase = activity::Phase::try_from(info.clone())?;
                let attempt = u8::try_from(info.attempt).map_err(|_| {
                    Refusal::unsupported("Describe attempt is outside maximum_attempts")
                })?;
                if !(state.attempt..=3).contains(&attempt) {
                    return Err(state_refusal(
                        "Describe attempt does not match the current attempt",
                    ));
                }
                if let Some(context) = &self.context {
                    let scenario = Self::scenario(context)?;
                    self.manifest
                        .as_ref()
                        .ok_or_else(|| {
                            Refusal::unsupported("admitted policy manifest is required")
                        })?
                        .admit_name(scenario)?;
                    for failure in [info.last_failure.as_ref(), Self::outcome_failure(&response)]
                        .into_iter()
                        .flatten()
                    {
                        Self::programmed_failure(failure)?;
                    }
                    // Every programmed response the evidence consults is selected here, so an
                    // unknown scenario or an attempt beyond the World's cap refuses before
                    // search; evidence matching then cannot fail. An inconsistent snapshot is
                    // Unexplained before any response is selected.
                    if Self::consistent(phase, attempt, &response) {
                        self.select_programmed(scenario, state.attempt, phase, &response)?;
                    }
                }
                Ok(umpire_runtime::Projection::Reach((Self::read().project)(self, &response)?))
            }
            _ => Err(Refusal::unsupported("unsupported observation method")),
        }
    }

    /// Select, in the pre-Reach order, every programmed response a Describe's evidence
    /// consults, stopping where that evidence is already contradicted.
    fn select_programmed(
        &self,
        scenario: &str,
        current: u8,
        phase: activity::Phase,
        response: &DescribeActivityExecutionResponse,
    ) -> Result<(), Refusal> {
        let info =
            response.info.as_ref().ok_or_else(|| Refusal::malformed("activity info is missing"))?;
        let attempt = u8::try_from(info.attempt)
            .map_err(|_| Refusal::unsupported("Describe attempt is outside maximum_attempts"))?;
        let retry = Worker::RespondFailed { retryable: true };
        if attempt > 1 && phase != activity::Phase::Failed {
            let Some((action, expected)) = self.snapshot_response(scenario, attempt - 1)? else {
                return Ok(());
            };
            if action != retry {
                return Ok(());
            }
            let expected: RespondActivityTaskFailedRequest = expected.unpack()?;
            if !Self::same_failure(info.last_failure.as_ref(), expected.failure.as_ref()) {
                return Ok(());
            }
        }
        for skipped in current..attempt {
            if self.snapshot_response(scenario, skipped)?.is_none_or(|(action, _)| action != retry)
            {
                return Ok(());
            }
        }
        let _ = self.snapshot_response(scenario, attempt)?;
        Ok(())
    }

    pub(super) fn reach_evidence(
        &self,
        observed: &crate::support::Observed,
        path: &Witness<'_>,
    ) -> umpire_runtime::EvidenceMatch {
        let evidence = match observed.method.as_str() {
            CANCEL if path.is_empty() => Ok(umpire_runtime::EvidenceMatch::Continue { cursor: 0 }),
            CANCEL => (|| {
                let response: Result<RequestCancelActivityExecutionResponse, i32> =
                    if let Some(code) = observed.code()? {
                        Err(code)
                    } else {
                        Ok(observed.response()?)
                    };
                if self.context.is_none() {
                    // An unadmitted run has no receipt to correlate with; the same edge rule holds.
                    return Ok(Self::edge(path, Self::outcome(&response)?, |action| {
                        *action == Action::from(Client::Cancel)
                    }));
                }
                let request = observed
                    .request()?
                    .ok_or_else(|| Refusal::malformed("actual request is missing"))?;
                self.cancel_evidence(request, response, path)
            })(),
            DESCRIBE => self.describe_evidence(observed, path),
            _ => Ok(umpire_runtime::EvidenceMatch::Reject),
        };
        // Projection already raised every refusal these predicates can reach: correlation, the
        // status code, the scenario and every programmed response. A failure here is no evidence.
        evidence.unwrap_or(umpire_runtime::EvidenceMatch::Reject)
    }

    /// Reconcile a Describe snapshot with the Model-derived worker program: each hidden worker
    /// edge must be the programmed response for its attempt and be evidenced by the snapshot's
    /// attempt, run state, outcome or retained failure.
    fn describe_evidence(
        &self,
        observed: &crate::support::Observed,
        path: &Witness<'_>,
    ) -> Result<umpire_runtime::EvidenceMatch, Refusal> {
        let reject = Ok(umpire_runtime::EvidenceMatch::Reject);
        let response: DescribeActivityExecutionResponse = observed.response()?;
        let info =
            response.info.as_ref().ok_or_else(|| Refusal::malformed("activity info is missing"))?;
        let phase = activity::Phase::try_from(info.clone())?;
        let attempt = u8::try_from(info.attempt)
            .map_err(|_| Refusal::unsupported("Describe attempt is outside maximum_attempts"))?;
        let Some(context) = &self.context else {
            return Ok(Self::snapshot_evidence(phase, attempt, path, false));
        };
        let scenario = Self::scenario(context)?;
        if !ActivityTarget::agrees(&response) || !Self::consistent(phase, attempt, &response) {
            return reject;
        }
        if attempt > 1 && phase != activity::Phase::Failed {
            let Some((action, expected)) = self.snapshot_response(scenario, attempt - 1)? else {
                return reject;
            };
            if action != (Worker::RespondFailed { retryable: true }) {
                return reject;
            }
            let expected: RespondActivityTaskFailedRequest = expected.unpack()?;
            if !Self::same_failure(info.last_failure.as_ref(), expected.failure.as_ref()) {
                return reject;
            }
        }
        let Some((programmed, expected)) = self.snapshot_response(scenario, attempt)? else {
            return reject;
        };
        match phase {
            activity::Phase::Completed => {
                if programmed != Worker::RespondCompleted {
                    return reject;
                }
                let expected: RespondActivityTaskCompletedRequest = expected.unpack()?;
                let result = response.outcome.as_ref().and_then(|outcome| outcome.value.as_ref());
                if !matches!(result, Some(Value::Result(result)) if expected.result.as_ref() == Some(result))
                {
                    return reject;
                }
            }
            activity::Phase::Failed => {
                if !matches!(programmed, Worker::RespondFailed { .. }) {
                    return reject;
                }
                let expected: RespondActivityTaskFailedRequest = expected.unpack()?;
                if Self::outcome_failure(&response).is_none()
                    || !Self::same_failure(
                        Self::outcome_failure(&response),
                        expected.failure.as_ref(),
                    )
                    || !Self::same_failure(info.last_failure.as_ref(), expected.failure.as_ref())
                {
                    return reject;
                }
            }
            _ => {}
        }
        for edge in path {
            if let Action::Worker(Worker::RespondFailed { retryable }) = edge.action
                && self
                    .snapshot_response(scenario, edge.source.attempt)?
                    .is_none_or(|(action, _)| action != (Worker::RespondFailed { retryable }))
            {
                return reject;
            }
        }
        Ok(Self::snapshot_evidence(phase, attempt, path, true))
    }

    /// The snapshot alone evidences delivery of earlier attempts and of a running or closed
    /// current attempt, a retry before each later attempt, and the terminal response its status
    /// names. Without a program, a closing failure is the nonretryable one.
    fn snapshot_evidence(
        phase: activity::Phase,
        attempt: u8,
        path: &Witness<'_>,
        programmed: bool,
    ) -> umpire_runtime::EvidenceMatch {
        let admitted = path.iter().all(|edge| {
            let at = edge.source.attempt;
            let delivered = at < attempt
                || (at == attempt
                    && matches!(
                        phase,
                        activity::Phase::Started
                            | activity::Phase::Completed
                            | activity::Phase::Failed
                    ));
            edge.outcome == Outcome::Accepted
                && match edge.action {
                    Action::Worker(Worker::Poll(polled)) => polled == at && delivered,
                    Action::Worker(Worker::RespondFailed { retryable }) => {
                        (at < attempt && retryable)
                            || (at == attempt
                                && phase == activity::Phase::Failed
                                && (programmed || !retryable))
                    }
                    Action::Worker(Worker::RespondCompleted) => {
                        at == attempt && phase == activity::Phase::Completed
                    }
                    Action::Client(_)
                    | Action::Worker(Worker::Heartbeat | Worker::RespondCanceled(_))
                    | Action::Deadline(_) => false,
                }
        });
        if admitted {
            umpire_runtime::EvidenceMatch::Match { cursor: path.len() }
        } else {
            umpire_runtime::EvidenceMatch::Reject
        }
    }

    /// The snapshot does not contradict itself: no retained failure at a first attempt that has
    /// not failed, no outcome while open, and a closing cancellation or timeout carries its cause.
    fn consistent(
        phase: activity::Phase,
        attempt: u8,
        response: &DescribeActivityExecutionResponse,
    ) -> bool {
        use umpire_temporal_api::temporal::api::failure::v1::failure::FailureInfo;
        let retained = response.info.as_ref().is_some_and(|info| info.last_failure.is_some());
        if attempt == 1
            && !matches!(phase, activity::Phase::Failed | activity::Phase::TimedOut)
            && retained
        {
            return false;
        }
        let Some(outcome) = response.outcome.as_ref() else {
            return true;
        };
        let cause = |cause: fn(&FailureInfo) -> bool| {
            matches!(&outcome.value, Some(Value::Failure(failure))
                if failure.failure_info.as_ref().is_some_and(cause))
        };
        match phase {
            activity::Phase::Scheduled | activity::Phase::Started => false,
            activity::Phase::Canceled => {
                cause(|info| matches!(info, FailureInfo::CanceledFailureInfo(_)))
            }
            activity::Phase::TimedOut => {
                cause(|info| matches!(info, FailureInfo::TimeoutFailureInfo(_)))
            }
            activity::Phase::Completed | activity::Phase::Failed => true,
        }
    }

    /// Snapshot evidence excludes heartbeat and acknowledgment edges, so only the
    /// exact empty history for each delivered task is admissible here.
    fn snapshot_response(
        &self,
        scenario: &str,
        attempt: u8,
    ) -> Result<Option<(Worker, Any)>, Refusal> {
        self.resolve_response(
            scenario,
            &super::policy::TaskKey { attempt, heartbeat_history: Vec::new() },
        )
    }

    fn outcome_failure(response: &DescribeActivityExecutionResponse) -> Option<&Failure> {
        match response.outcome.as_ref().and_then(|outcome| outcome.value.as_ref()) {
            Some(Value::Failure(failure)) => Some(failure),
            _ => None,
        }
    }

    fn scenario(context: &ProposalContext) -> Result<&str, Refusal> {
        context
            .start
            .header
            .as_ref()
            .and_then(|header| header.fields.get(super::SCENARIO_HEADER))
            .and_then(|payload| std::str::from_utf8(&payload.data).ok())
            .ok_or_else(|| Refusal::malformed("Start scenario header is missing"))
    }
}

fn state_refusal(message: &str) -> Refusal {
    Refusal::new(RefusalCode::State, message)
}

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
    pub(super) fn program_inputs(action: Worker, attempt: u8) -> Result<FillInputs, Refusal> {
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
            Worker::Heartbeat | Worker::RespondCanceled(_) => {
                FillInputs::Worker { attempt, details: None }
            }
            Worker::Poll(_) => {
                return Err(Refusal::unsupported(
                    "Poll is task delivery, not a programmed response",
                ));
            }
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
        context.matches(
            namespace,
            activity_id.unwrap_or(&context.start.activity_id),
            run_id.unwrap_or(&context.run_id),
        )
    }

    /// The Model outcome evidenced by an RPC result. A status code outside the Model's declared
    /// rejections is evidence the Model cannot classify, so it refuses rather than judging.
    fn outcome<T>(response: &Result<T, i32>) -> Result<Outcome, Refusal> {
        match response {
            Ok(_) => Ok(Outcome::Accepted),
            Err(code) => rejection(*code).map(Outcome::Rejected).ok_or_else(|| {
                Refusal::unsupported(format!(
                    "status code {code} is outside the Model's declared rejections"
                ))
            }),
        }
    }

    /// Match the complete witness prefix's last edge; RPC evidence never admits a root, and one
    /// actual RPC evidences exactly one edge, so no earlier edge of the prefix may claim it.
    fn edge(
        path: &Witness<'_>,
        outcome: Outcome,
        action: impl Fn(&Action) -> bool,
    ) -> umpire_runtime::EvidenceMatch {
        match path.split_last() {
            Some((edge, earlier))
                if action(&edge.action)
                    && edge.outcome == outcome
                    && !earlier.iter().any(|edge| action(&edge.action)) =>
            {
                umpire_runtime::EvidenceMatch::Match { cursor: path.len() }
            }
            _ => umpire_runtime::EvidenceMatch::Reject,
        }
    }
}

fn state(message: impl Into<String>) -> Refusal {
    Refusal::new(RefusalCode::State, message)
}

fn checked<T: 'static>(
    value: &T,
    catalog: Result<Catalog<T>, umpire_catalog::AdmissionErrors>,
) -> Result<(), Refusal> {
    let catalog =
        catalog.map_err(|_| Refusal::unsupported("Start validation catalog admission failed"))?;
    if validate(value, &catalog).is_empty() {
        Ok(())
    } else {
        Err(Refusal::malformed("Start captured evidence violates its catalog"))
    }
}

fn validate_start(
    request: &StartActivityExecutionRequest,
    response: &StartActivityExecutionResponse,
) -> Result<(), Refusal> {
    for (present, field) in [
        (!request.namespace.is_empty(), "namespace"),
        (!request.activity_id.is_empty(), "activity_id"),
        (!request.request_id.is_empty(), "request_id"),
        (!response.run_id.is_empty(), "run_id"),
    ] {
        if !present {
            return Err(Refusal::malformed(format!("Start {field} is missing")));
        }
    }
    let limits = Limits::default();
    checked(request, catalog::start_activity_execution_request(&limits))?;
    checked(response, catalog::start_activity_execution_response(&limits))?;
    if let Some(value) = &request.activity_type {
        checked(value, catalog::activity_type(&limits))?;
    }
    if let Some(value) = &request.task_queue {
        checked(value, catalog::task_queue(&limits))?;
    }
    if let Some(value) = &request.retry_policy {
        let policy = catalog::retry_policy(&limits)
            .map_err(|_| Refusal::unsupported("Start validation catalog admission failed"))?;
        if validate(value, &policy)
            .iter()
            .any(|finding| !finding.path.ends_with(".maximum_attempts"))
        {
            return Err(Refusal::malformed("Start retry policy violates its catalog"));
        }
    }
    if let Some(link) = &response.link {
        let Some(link::Variant::Activity(link)) = &link.variant else {
            return Err(Refusal::malformed("Start response Link is not an activity"));
        };
        if link.namespace != request.namespace
            || link.activity_id != request.activity_id
            || link.run_id != response.run_id
        {
            return Err(state("Start response Link contradicts captured identity"));
        }
    }
    Ok(())
}

#[cfg(test)]
#[path = "../../realization/tests/activity_registry/mod.rs"]
mod registry_controls;
