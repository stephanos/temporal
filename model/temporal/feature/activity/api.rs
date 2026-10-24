//! Exhaustive activity API mirrors, independent of the Model's runtime state.
#![allow(dead_code, reason = "Mirror fields describe the API and are not instantiated")]

use crate::activity::{self, Client, Deadline, Worker};
use crate::foundation::dc;
use umpire_temporal_annotate::{
    Catalog,
    kinds::{
        Counter, Duration, Enum, Flag, Identifier, Ignored, Message, OneOf, RawDuration, Secret,
        Text, Timestamp, Uuid,
    },
};
use umpire_temporal_api::temporal::api::{
    activity::v1 as a, common::v1 as c, enums::v1 as e, failure::v1 as f, taskqueue::v1 as q,
    workflowservice::v1 as w,
};

type ModelAction = <activity::World as umpire_dsl::Machine>::Action;

#[path = "api_transport.rs"]
mod transport;
pub use transport::{AnnotationResult, annotate};

/// Success and failure are mutually exclusive outcome alternatives.
#[derive(Catalog)]
#[mirror(response = "temporal.api.activity.v1.ActivityExecutionOutcome")]
#[rule(one_of(result, failure))]
pub struct ActivityExecutionOutcome {
    #[mirror(decided)]
    #[mirror(oneof(Result = "result", Failure = "failure"))]
    value: OneOf<a::activity_execution_outcome::Value>,
    retry_state: Enum<e::RetryState>,
}

/// Server lifecycle metadata keeps externally supplied message contents Decided.
#[derive(Catalog)]
#[mirror(response = "temporal.api.activity.v1.ActivityExecutionInfo")]
pub struct ActivityExecutionInfo {
    activity_id: Text,
    run_id: Uuid,
    #[mirror(decided)]
    activity_type: Message<c::ActivityType>,
    status: Enum<e::ActivityExecutionStatus>,
    run_state: Enum<e::PendingActivityState>,
    task_queue: Text,
    schedule_to_close_timeout: RawDuration,
    schedule_to_start_timeout: RawDuration,
    start_to_close_timeout: RawDuration,
    heartbeat_timeout: RawDuration,
    #[mirror(decided)]
    retry_policy: Message<c::RetryPolicy>,
    #[mirror(decided)]
    heartbeat_details: Message<c::Payloads>,
    last_heartbeat_time: Timestamp,
    last_started_time: Timestamp,
    #[rule(gte = 0)]
    attempt: Counter,
    execution_duration: RawDuration,
    schedule_time: Timestamp,
    expiration_time: Timestamp,
    close_time: Timestamp,
    #[mirror(decided)]
    last_failure: Message<f::Failure>,
    last_worker_identity: Text,
    current_retry_interval: RawDuration,
    last_attempt_complete_time: Timestamp,
    next_attempt_schedule_time: Timestamp,
    #[mirror(ignored = "This activity Model does not project \
        ActivityExecutionInfo.last_deployment_version; no semantic validation is claimed")]
    last_deployment_version: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        ActivityExecutionInfo.priority; no semantic validation is claimed")]
    priority: Ignored,
    state_transition_count: Counter,
    state_size_bytes: Counter,
    #[mirror(ignored = "This activity Model does not project \
        ActivityExecutionInfo.search_attributes; no semantic validation is claimed")]
    search_attributes: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        ActivityExecutionInfo.header; no semantic validation is claimed")]
    header: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        ActivityExecutionInfo.user_metadata; no semantic validation is claimed")]
    user_metadata: Ignored,
    canceled_reason: Text,
    #[mirror(ignored = "This activity Model does not project \
        ActivityExecutionInfo.links; no semantic validation is claimed")]
    links: Ignored,
    total_heartbeat_count: Counter,
    sdk_name: Text,
    sdk_version: Text,
    start_delay: RawDuration,
    execution_time: Timestamp,
}

/// Worker polling validates queue identity without modeling version routing.
#[derive(Catalog)]
#[rule(required(task_queue), rejects = "taskQueue|task queue")]
#[rule(required(namespace), rejects = "Namespace is empty")]
#[mirror(request = "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest")]
pub struct PollActivityTaskQueueRequest {
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "Namespace length exceeds limit")]
    namespace: Text,

    task_queue: Message<q::TaskQueue>,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueRequest.poller_group_id; no semantic validation is claimed")]
    poller_group_id: Ignored,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity")]
    identity: Text,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueRequest.worker_instance_key; no semantic validation is claimed")]
    worker_instance_key: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueRequest.worker_control_task_queue; no semantic validation is claimed")]
    worker_control_task_queue: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueRequest.task_queue_metadata; no semantic validation is claimed")]
    task_queue_metadata: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueRequest.worker_version_capabilities; \
        no semantic validation is claimed")]
    worker_version_capabilities: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueRequest.deployment_options; no semantic validation is claimed")]
    deployment_options: Ignored,
}

/// Polled tasks carry opaque tokens and externally supplied activity payloads.
#[derive(Catalog)]
#[allow(clippy::duplicated_attributes, reason = "Presence rules may share dependency operands")]
#[rule(requires(activity_id, task_token))]
#[rule(requires(input, task_token))]
#[mirror(response = "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse")]
pub struct PollActivityTaskQueueResponse {
    task_token: Secret,
    workflow_namespace: Text,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.workflow_type; no semantic validation is claimed")]
    workflow_type: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.workflow_execution; no semantic validation is claimed")]
    workflow_execution: Ignored,
    #[mirror(decided)]
    activity_type: Message<c::ActivityType>,

    activity_id: Text,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.header; no semantic validation is claimed")]
    header: Ignored,

    #[mirror(decided)]
    input: Message<c::Payloads>,
    #[mirror(decided)]
    heartbeat_details: Message<c::Payloads>,
    scheduled_time: Timestamp,
    current_attempt_scheduled_time: Timestamp,
    started_time: Timestamp,
    #[rule(gte = 0)]
    attempt: Counter,
    schedule_to_close_timeout: RawDuration,
    start_to_close_timeout: RawDuration,
    heartbeat_timeout: RawDuration,
    #[mirror(decided)]
    retry_policy: Message<c::RetryPolicy>,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.poller_scaling_decision; no semantic validation is claimed")]
    poller_scaling_decision: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.priority; no semantic validation is claimed")]
    priority: Ignored,
    activity_run_id: Uuid,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.poller_group_infos; no semantic validation is claimed")]
    poller_group_infos: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.poller_groups_info; no semantic validation is claimed")]
    poller_groups_info: Ignored,
}

/// Completion transports worker results under the frontend identity limit.
#[derive(Catalog)]
#[rule(required(task_token), rejects = "Task token not set")]
#[mirror(request = "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest")]
pub struct RespondActivityTaskCompletedRequest {
    task_token: Secret,
    result: Message<c::Payloads>,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity")]
    identity: Text,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "Namespace length exceeds limit")]
    namespace: Text,
    resource_id: Text,
    #[mirror(ignored = "This activity Model does not project \
        RespondActivityTaskCompletedRequest.worker_version; no semantic validation is claimed")]
    worker_version: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        RespondActivityTaskCompletedRequest.deployment; no semantic validation is claimed")]
    deployment: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        RespondActivityTaskCompletedRequest.deployment_options; no semantic validation is claimed")]
    deployment_options: Ignored,
}

/// Completion acknowledgement carries no additional lifecycle evidence.
#[derive(Catalog)]
#[mirror(response = "temporal.api.workflowservice.v1.RespondActivityTaskCompletedResponse")]
pub struct RespondActivityTaskCompletedResponse {}

/// Worker failures require root application-failure information, unlike nested server failures.
#[derive(Catalog)]
#[rule(required(task_token), rejects = "Task token not set")]
#[mirror(request = "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest")]
pub struct RespondActivityTaskFailedRequest {
    task_token: Secret,
    failure: Message<f::Failure>,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity")]
    identity: Text,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "Namespace length exceeds limit")]
    namespace: Text,
    resource_id: Text,
    last_heartbeat_details: Message<c::Payloads>,
    #[mirror(ignored = "This activity Model does not project \
        RespondActivityTaskFailedRequest.worker_version; no semantic validation is claimed")]
    worker_version: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        RespondActivityTaskFailedRequest.deployment; no semantic validation is claimed")]
    deployment: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        RespondActivityTaskFailedRequest.deployment_options; no semantic validation is claimed")]
    deployment_options: Ignored,
    #[rule(accepts)]
    cause: Enum<e::ActivityTaskFailedCause>,
}

/// Failure-response diagnostics are retained without rewriting their contents.
#[derive(Catalog)]
#[mirror(response = "temporal.api.workflowservice.v1.RespondActivityTaskFailedResponse")]
pub struct RespondActivityTaskFailedResponse {
    #[mirror(decided)]
    failures: Vec<Message<f::Failure>>,
}

/// Worker cancellation acknowledgement retains opaque details and validates identity.
#[derive(Catalog)]
#[rule(required(task_token), rejects = "Task token not set")]
#[mirror(request = "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest")]
pub struct RespondActivityTaskCanceledRequest {
    task_token: Secret,
    details: Message<c::Payloads>,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity")]
    identity: Text,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "Namespace length exceeds limit")]
    namespace: Text,
    resource_id: Text,
    #[mirror(ignored = "This activity Model does not project \
        RespondActivityTaskCanceledRequest.worker_version; no semantic validation is claimed")]
    worker_version: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        RespondActivityTaskCanceledRequest.deployment; no semantic validation is claimed")]
    deployment: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        RespondActivityTaskCanceledRequest.deployment_options; no semantic validation is claimed")]
    deployment_options: Ignored,
}

/// Cancellation acknowledgement carries no additional lifecycle evidence.
#[derive(Catalog)]
#[mirror(response = "temporal.api.workflowservice.v1.RespondActivityTaskCanceledResponse")]
pub struct RespondActivityTaskCanceledResponse {}

/// Timeout and payload limits apply before the standalone activity is scheduled.
#[derive(Catalog)]
#[rule(required(activity_type, task_queue), rejects = "activityType|taskQueue|task queue")]
#[rule(required(namespace), rejects = "Namespace is empty")]
#[mirror(request = "temporal.api.workflowservice.v1.StartActivityExecutionRequest")]
#[rule(
    any_of(schedule_to_close_timeout, start_to_close_timeout),
    rejects = "a valid StartToCloseTimeout or ScheduleToCloseTimeout"
)]
pub struct StartActivityExecutionRequest {
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "Namespace length exceeds limit")]
    namespace: Text,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity")]
    identity: Text,
    #[mirror(idempotency_key)]
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "request ID exceeds length limit")]
    #[mirror(also(uuid, accepts))]
    request_id: Text,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "activity[Ii][Dd]|activity ID")]
    activity_id: Identifier,

    activity_type: Message<c::ActivityType>,

    task_queue: Message<q::TaskQueue>,
    #[rule(gte = 0, rejects = "invalid ScheduleToCloseTimeout")]
    schedule_to_close_timeout: Duration,
    #[rule(gte = 0, rejects = "invalid ScheduleToStartTimeout")]
    schedule_to_start_timeout: Duration,
    #[rule(gte = 0, rejects = "invalid StartToCloseTimeout")]
    start_to_close_timeout: Duration,
    #[rule(gte = 0, rejects = "invalid HeartbeatTimeout")]
    heartbeat_timeout: Duration,
    retry_policy: Message<c::RetryPolicy>,
    #[rule(max_len = dc::BLOB_SIZE_ERROR, rejects = "input exceeds length limit")]
    input: Message<c::Payloads>,
    #[rule(rejects = "unsupported ID reuse policy")]
    id_reuse_policy: Enum<e::ActivityIdReusePolicy>,
    #[rule(rejects = "unsupported ID conflict policy")]
    id_conflict_policy: Enum<e::ActivityIdConflictPolicy>,
    #[mirror(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.search_attributes; no semantic validation is claimed")]
    search_attributes: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.header; no semantic validation is claimed")]
    header: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.user_metadata; no semantic validation is claimed")]
    user_metadata: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.priority; no semantic validation is claimed")]
    priority: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.completion_callbacks; no semantic validation is claimed")]
    completion_callbacks: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.links; no semantic validation is claimed")]
    links: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.on_conflict_options; no semantic validation is claimed")]
    on_conflict_options: Ignored,
    #[rule(gte = 0, rejects = "invalid StartDelay")]
    start_delay: Duration,
}

/// The assigned run identifier is observed; execution links remain outside this Model.
#[derive(Catalog)]
#[rule(required(run_id))]
#[mirror(response = "temporal.api.workflowservice.v1.StartActivityExecutionResponse")]
pub struct StartActivityExecutionResponse {
    run_id: Uuid,
    started: Flag,
    #[mirror(ignored = "This activity Model does not project \
        StartActivityExecutionResponse.link; no semantic validation is claimed")]
    link: Ignored,
}

/// Long-poll tokens require an explicit run identifier.
#[derive(Catalog)]
#[rule(requires(long_poll_token, run_id), rejects = "run id is required")]
#[rule(required(namespace), rejects = "Namespace is empty")]
#[mirror(request = "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest")]
pub struct DescribeActivityExecutionRequest {
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "Namespace length exceeds limit")]
    namespace: Text,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "activity[Ii][Dd]|activity ID")]
    activity_id: Identifier,
    #[rule(rejects = "invalid run id")]
    run_id: Uuid,
    #[mirror(ignored = "This activity Model does not project \
        DescribeActivityExecutionRequest.include_input; no semantic validation is claimed")]
    include_input: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        DescribeActivityExecutionRequest.include_outcome; no semantic validation is claimed")]
    include_outcome: Ignored,

    long_poll_token: Secret,
    #[mirror(ignored = "This activity Model does not project \
        DescribeActivityExecutionRequest.include_heartbeat_details; \
        no semantic validation is claimed")]
    include_heartbeat_details: Ignored,
    #[mirror(ignored = "This activity Model does not project \
        DescribeActivityExecutionRequest.include_last_failure; no semantic validation is claimed")]
    include_last_failure: Ignored,
}

/// Lifecycle metadata is required; optional external payloads remain Decided.
#[derive(Catalog)]
#[rule(required(info))]
#[mirror(response = "temporal.api.workflowservice.v1.DescribeActivityExecutionResponse")]
pub struct DescribeActivityExecutionResponse {
    run_id: Uuid,

    #[mirror(decided)]
    info: Message<a::ActivityExecutionInfo>,
    #[mirror(decided)]
    input: Message<c::Payloads>,
    #[mirror(decided)]
    outcome: Message<a::ActivityExecutionOutcome>,
    long_poll_token: Secret,
    #[mirror(ignored = "This activity Model does not project \
        DescribeActivityExecutionResponse.callbacks; no semantic validation is claimed")]
    callbacks: Ignored,
}

/// Outcome polling accepts an optional normalized run UUID.
#[derive(Catalog)]
#[rule(required(namespace), rejects = "Namespace is empty")]
#[mirror(request = "temporal.api.workflowservice.v1.PollActivityExecutionRequest")]
pub struct PollActivityExecutionRequest {
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "Namespace length exceeds limit")]
    namespace: Text,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "activity[Ii][Dd]|activity ID")]
    activity_id: Identifier,
    #[rule(rejects = "invalid run id")]
    run_id: Uuid,
}

/// The observed run identifier scopes the optional external outcome.
#[derive(Catalog)]
#[mirror(response = "temporal.api.workflowservice.v1.PollActivityExecutionResponse")]
pub struct PollActivityExecutionResponse {
    run_id: Uuid,
    #[mirror(decided)]
    outcome: Message<a::ActivityExecutionOutcome>,
}

/// Cancellation intent is separate from the worker's eventual cancellation acknowledgement.
#[derive(Catalog)]
#[rule(required(namespace), rejects = "Namespace is empty")]
#[mirror(request = "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest")]
pub struct RequestCancelActivityExecutionRequest {
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "Namespace length exceeds limit")]
    namespace: Text,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "activity[Ii][Dd]|activity ID")]
    activity_id: Identifier,
    #[rule(rejects = "invalid run id")]
    run_id: Uuid,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity")]
    identity: Text,
    #[mirror(idempotency_key, also(uuid, accepts))]
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "request ID exceeds length limit")]
    request_id: Text,
    #[rule(max_bytes = dc::BLOB_SIZE_ERROR, rejects = "reason exceeds length limit")]
    reason: Text,
}

/// Cancellation intent acknowledgement carries no additional lifecycle evidence.
#[derive(Catalog)]
#[mirror(response = "temporal.api.workflowservice.v1.RequestCancelActivityExecutionResponse")]
pub struct RequestCancelActivityExecutionResponse {}

use crate::foundation::service::{self, RpcRole};
use umpire_temporal_annotate::Binding;
use umpire_temporal_api::WorkflowServiceMethod;

/// The RPCs this feature claims; every other RPC is out of scope with its resource family.
fn classify(method: WorkflowServiceMethod) -> RpcRole<ModelAction> {
    match method {
        WorkflowServiceMethod::PollActivityTaskQueue => {
            RpcRole::Action(ModelAction::Worker(Worker::Poll(1)))
        }
        WorkflowServiceMethod::RespondActivityTaskCompleted => {
            RpcRole::Action(ModelAction::Worker(Worker::RespondCompleted))
        }
        WorkflowServiceMethod::RespondActivityTaskFailed => {
            RpcRole::Action(ModelAction::Worker(Worker::RespondFailed { retryable: false }))
        }
        WorkflowServiceMethod::RespondActivityTaskCanceled => {
            RpcRole::Observed("worker acknowledges client cancellation")
        }
        WorkflowServiceMethod::StartActivityExecution => {
            RpcRole::Observed("successful start establishes the Model initial state")
        }
        WorkflowServiceMethod::DescribeActivityExecution => {
            RpcRole::Observed("activity lifecycle phase")
        }
        WorkflowServiceMethod::PollActivityExecution => {
            RpcRole::Observed("SDK activity handle outcome")
        }
        WorkflowServiceMethod::RequestCancelActivityExecution => {
            RpcRole::Action(ModelAction::Client(Client::Cancel))
        }
        _ => RpcRole::OutOfScope(service::family(method)),
    }
}

/// Executable factories and exhaustive runtime bindings.
pub mod catalog {
    use super::{Binding, Client, Deadline, ModelAction, Worker, a, c, dc, f, q, w};
    pub use crate::foundation::enum_inventory::enum_inventory::*;
    use umpire_catalog::{AdmissionErrors, Catalog, ProbeCase};
    pub use umpire_temporal_api::RPC_BINDINGS;

    use crate::foundation::api as foundation;

    macro_rules! factories {
        ($($source:ident => $target:ident:
            [$($mirror:ident => $factory:ident),* $(,)?]);* $(;)?) => {
            $($(
                #[doc = concat!("Admits the catalog for ", stringify!($mirror), ".")]
                pub fn $factory(limits: &dc::Limits)
                -> Result<Catalog<$target::$mirror>, AdmissionErrors> {
                    $source::$mirror::catalog(limits)
                }
            )*)*
            /// Every declared primary and supplemental expectation.
            pub fn expected_probe_cases(limits: &dc::Limits)
            -> Result<Vec<ProbeCase>, AdmissionErrors> {
                let mut cases = Vec::new();
                $($(for catalog in $source::$mirror::catalogs(limits)? {
                    cases.extend(catalog.probe_cases());
                })*)*
                cases.sort_by(|a, b| a.id.cmp(&b.id));
                Ok(cases)
            }
        };
    }
    factories! {
        foundation => c: [
            WorkflowExecution => workflow_execution,
            WorkflowType => workflow_type,
            Payloads => payloads,
            Payload => payload,
            ActivityType => activity_type,
            RetryPolicy => retry_policy,
        ];
        super => a: [
            ActivityExecutionOutcome => activity_execution_outcome,
            ActivityExecutionInfo => activity_execution_info,
        ];
        foundation => f: [
            ApplicationFailureInfo => application_failure_info,
            Failure => failure,
            TimeoutFailureInfo => timeout_failure_info,
            CanceledFailureInfo => canceled_failure_info,
            TerminatedFailureInfo => terminated_failure_info,
            ServerFailureInfo => server_failure_info,
            ResetWorkflowFailureInfo => reset_workflow_failure_info,
            ActivityFailureInfo => activity_failure_info,
            ChildWorkflowExecutionFailureInfo => child_workflow_execution_failure_info,
            NexusOperationFailureInfo => nexus_operation_failure_info,
            NexusHandlerFailureInfo => nexus_handler_failure_info,
        ];
        foundation => q: [
            TaskQueue => task_queue,
        ];
        super => w: [
            PollActivityTaskQueueRequest => poll_activity_task_queue_request,
            PollActivityTaskQueueResponse => poll_activity_task_queue_response,
            RespondActivityTaskCompletedRequest => respond_activity_task_completed_request,
            RespondActivityTaskCompletedResponse => respond_activity_task_completed_response,
            RespondActivityTaskFailedRequest => respond_activity_task_failed_request,
            RespondActivityTaskFailedResponse => respond_activity_task_failed_response,
            RespondActivityTaskCanceledRequest => respond_activity_task_canceled_request,
            RespondActivityTaskCanceledResponse => respond_activity_task_canceled_response,
            StartActivityExecutionRequest => start_activity_execution_request,
            StartActivityExecutionResponse => start_activity_execution_response,
            DescribeActivityExecutionRequest => describe_activity_execution_request,
            DescribeActivityExecutionResponse => describe_activity_execution_response,
            PollActivityExecutionRequest => poll_activity_execution_request,
            PollActivityExecutionResponse => poll_activity_execution_response,
            RequestCancelActivityExecutionRequest => request_cancel_activity_execution_request,
            RequestCancelActivityExecutionResponse => request_cancel_activity_execution_response,
        ];
    }

    /// Binds every Model action, including explicitly environmental deadlines.
    pub fn bind(action: &ModelAction) -> Binding {
        umpire_temporal_api::bind_workflow_action!(action,
            ModelAction::Deadline(Deadline::Tick)
            | ModelAction::Worker(Worker::Heartbeat | Worker::RespondCanceled(_)) => {
                if matches!(action, ModelAction::Deadline(Deadline::Tick)) {
                    Binding::Environmental("abstract deadline tick")
                } else if matches!(action, ModelAction::Worker(Worker::Heartbeat)) {
                    Binding::Rpc("temporal.api.workflowservice.v1.WorkflowService.RecordActivityTaskHeartbeat")
                } else {
                    Binding::Rpc("temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskCanceled")
                }
            }
        )
    }
}

/// UUID intent for request IDs accepted without UUID syntax by the server.
pub fn start_request_id_intent(
    limits: &dc::Limits,
) -> Result<Catalog<w::StartActivityExecutionRequest>, umpire_catalog::AdmissionErrors> {
    StartActivityExecutionRequest::__request_id_also_0(limits)
}

/// UUID intent for cancellation request IDs.
pub fn cancel_request_id_intent(
    limits: &dc::Limits,
) -> Result<Catalog<w::RequestCancelActivityExecutionRequest>, umpire_catalog::AdmissionErrors> {
    RequestCancelActivityExecutionRequest::__request_id_also_0(limits)
}

/// Worker-supplied failures require application failure information.
pub fn worker_failure_presence(
    limits: &dc::Limits,
) -> Result<Catalog<f::Failure>, umpire_catalog::AdmissionErrors> {
    crate::foundation::api::WorkerFailure::catalog(limits)
}

/// Source-reviewed dispositions keyed by complete contextual probe identity.
pub(crate) fn accepted_probe_gaps() -> Vec<(String, umpire_catalog::AcceptedGap)> {
    Vec::new()
}
