//! Exhaustive activity API mirrors, independent of the Model's runtime state.
#![allow(dead_code, reason = "Mirror fields describe the API and are not instantiated")]

use crate::activity::{self, Client, Deadline, Worker};
use crate::foundation::{dc, validators};
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
const NORMALIZE_AND_VALIDATE: &str = validators::tqid::normalizeAndValidate;
const POLL_QUEUE: &str = validators::frontend::WorkflowHandler_PollActivityTaskQueue;
const COMPLETED: &str = validators::frontend::WorkflowHandler_RespondActivityTaskCompleted;
const FAILED: &str = validators::frontend::WorkflowHandler_RespondActivityTaskFailed;
const CANCELED: &str = validators::frontend::WorkflowHandler_RespondActivityTaskCanceled;
const TIMEOUTS: &str = validators::activity::validateAndNormalizeTimeouts;
const START: &str = validators::activity::validateAndNormalizeStartRequest;
const ATTRIBUTES: &str = validators::activity::validateAndNormalizeActivityAttributes;
const STANDALONE: &str = validators::activity::ValidateAndNormalizeStandaloneActivity;
const HANDLER: &str = validators::activity::handler_StartActivityExecution;
const START_DELAY: &str = validators::activity::validateStartDelay;
const DESCRIBE: &str = validators::activity::validateAndNormalizeDescribeActivityExecutionRequest;
const POLL: &str = validators::activity::validateAndNormalizePollActivityExecutionRequest;
const CANCEL: &str =
    validators::activity::validateAndNormalizeRequestCancelActivityExecutionRequest;
const REQUEST_ID: &str = validators::activity::validateAndNormalizeRequestID;

type ModelAction = <activity::World as umpire_dsl::Machine>::Action;

#[path = "api_transport.rs"]
mod transport;
pub use transport::{AnnotationResult, annotate};

/// Success and failure are mutually exclusive outcome alternatives.
#[derive(Catalog)]
#[api(response = "temporal.api.activity.v1.ActivityExecutionOutcome")]
#[api(one_of(result, failure))]
pub struct ActivityExecutionOutcome {
    #[api(decided)]
    #[api(oneof(Result = "result", Failure = "failure"))]
    value: OneOf<a::activity_execution_outcome::Value>,
    retry_state: Enum<e::RetryState>,
}

/// Server lifecycle metadata keeps externally supplied message contents Decided.
#[derive(Catalog)]
#[api(response = "temporal.api.activity.v1.ActivityExecutionInfo")]
pub struct ActivityExecutionInfo {
    activity_id: Text,
    run_id: Uuid,
    #[api(decided)]
    activity_type: Message<c::ActivityType>,
    status: Enum<e::ActivityExecutionStatus>,
    run_state: Enum<e::PendingActivityState>,
    task_queue: Text,
    schedule_to_close_timeout: RawDuration,
    schedule_to_start_timeout: RawDuration,
    start_to_close_timeout: RawDuration,
    heartbeat_timeout: RawDuration,
    #[api(decided)]
    retry_policy: Message<c::RetryPolicy>,
    #[api(decided)]
    heartbeat_details: Message<c::Payloads>,
    last_heartbeat_time: Timestamp,
    last_started_time: Timestamp,
    #[api(gte = 0)]
    attempt: Counter,
    execution_duration: RawDuration,
    schedule_time: Timestamp,
    expiration_time: Timestamp,
    close_time: Timestamp,
    #[api(decided)]
    last_failure: Message<f::Failure>,
    last_worker_identity: Text,
    current_retry_interval: RawDuration,
    last_attempt_complete_time: Timestamp,
    next_attempt_schedule_time: Timestamp,
    #[api(ignored = "This activity Model does not project \
        ActivityExecutionInfo.last_deployment_version; no semantic validation is claimed")]
    last_deployment_version: Ignored,
    #[api(ignored = "This activity Model does not project \
        ActivityExecutionInfo.priority; no semantic validation is claimed")]
    priority: Ignored,
    state_transition_count: Counter,
    state_size_bytes: Counter,
    #[api(ignored = "This activity Model does not project \
        ActivityExecutionInfo.search_attributes; no semantic validation is claimed")]
    search_attributes: Ignored,
    #[api(ignored = "This activity Model does not project \
        ActivityExecutionInfo.header; no semantic validation is claimed")]
    header: Ignored,
    #[api(ignored = "This activity Model does not project \
        ActivityExecutionInfo.user_metadata; no semantic validation is claimed")]
    user_metadata: Ignored,
    canceled_reason: Text,
    #[api(ignored = "This activity Model does not project \
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
#[api(request = "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest")]
pub struct PollActivityTaskQueueRequest {
    namespace: Text,
    #[api(required, rejects = "taskQueue|task queue", covers = NORMALIZE_AND_VALIDATE)]
    task_queue: Message<q::TaskQueue>,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueRequest.poller_group_id; no semantic validation is claimed")]
    poller_group_id: Ignored,
    #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity", covers = POLL_QUEUE)]
    identity: Text,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueRequest.worker_instance_key; no semantic validation is claimed")]
    worker_instance_key: Ignored,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueRequest.worker_control_task_queue; no semantic validation is claimed")]
    worker_control_task_queue: Ignored,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueRequest.task_queue_metadata; no semantic validation is claimed")]
    task_queue_metadata: Ignored,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueRequest.worker_version_capabilities; \
        no semantic validation is claimed")]
    worker_version_capabilities: Ignored,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueRequest.deployment_options; no semantic validation is claimed")]
    deployment_options: Ignored,
}

/// Polled tasks carry opaque tokens and externally supplied activity payloads.
#[derive(Catalog)]
#[api(response = "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse")]
pub struct PollActivityTaskQueueResponse {
    task_token: Secret,
    workflow_namespace: Text,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.workflow_type; no semantic validation is claimed")]
    workflow_type: Ignored,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.workflow_execution; no semantic validation is claimed")]
    workflow_execution: Ignored,
    #[api(decided)]
    activity_type: Message<c::ActivityType>,
    #[api(requires = task_token)]
    activity_id: Text,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.header; no semantic validation is claimed")]
    header: Ignored,
    #[api(requires = task_token)]
    #[api(decided)]
    input: Message<c::Payloads>,
    #[api(decided)]
    heartbeat_details: Message<c::Payloads>,
    scheduled_time: Timestamp,
    current_attempt_scheduled_time: Timestamp,
    started_time: Timestamp,
    #[api(gte = 0)]
    attempt: Counter,
    schedule_to_close_timeout: RawDuration,
    start_to_close_timeout: RawDuration,
    heartbeat_timeout: RawDuration,
    #[api(decided)]
    retry_policy: Message<c::RetryPolicy>,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.poller_scaling_decision; no semantic validation is claimed")]
    poller_scaling_decision: Ignored,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.priority; no semantic validation is claimed")]
    priority: Ignored,
    activity_run_id: Uuid,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.poller_group_infos; no semantic validation is claimed")]
    poller_group_infos: Ignored,
    #[api(ignored = "This activity Model does not project \
        PollActivityTaskQueueResponse.poller_groups_info; no semantic validation is claimed")]
    poller_groups_info: Ignored,
}

/// Completion transports worker results under the frontend identity limit.
#[derive(Catalog)]
#[api(request = "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest")]
pub struct RespondActivityTaskCompletedRequest {
    task_token: Secret,
    result: Message<c::Payloads>,
    #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity", covers = COMPLETED)]
    identity: Text,
    namespace: Text,
    resource_id: Text,
    #[api(ignored = "This activity Model does not project \
        RespondActivityTaskCompletedRequest.worker_version; no semantic validation is claimed")]
    worker_version: Ignored,
    #[api(ignored = "This activity Model does not project \
        RespondActivityTaskCompletedRequest.deployment; no semantic validation is claimed")]
    deployment: Ignored,
    #[api(ignored = "This activity Model does not project \
        RespondActivityTaskCompletedRequest.deployment_options; no semantic validation is claimed")]
    deployment_options: Ignored,
}

/// Completion acknowledgement carries no additional lifecycle evidence.
#[derive(Catalog)]
#[api(response = "temporal.api.workflowservice.v1.RespondActivityTaskCompletedResponse")]
pub struct RespondActivityTaskCompletedResponse {}

/// Worker failures require root application-failure information, unlike nested server failures.
#[derive(Catalog)]
#[api(request = "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest")]
pub struct RespondActivityTaskFailedRequest {
    task_token: Secret,
    failure: Message<f::Failure>,
    #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity", covers = FAILED)]
    identity: Text,
    namespace: Text,
    resource_id: Text,
    last_heartbeat_details: Message<c::Payloads>,
    #[api(ignored = "This activity Model does not project \
        RespondActivityTaskFailedRequest.worker_version; no semantic validation is claimed")]
    worker_version: Ignored,
    #[api(ignored = "This activity Model does not project \
        RespondActivityTaskFailedRequest.deployment; no semantic validation is claimed")]
    deployment: Ignored,
    #[api(ignored = "This activity Model does not project \
        RespondActivityTaskFailedRequest.deployment_options; no semantic validation is claimed")]
    deployment_options: Ignored,
    #[api(accepts, covers = FAILED)]
    cause: Enum<e::ActivityTaskFailedCause>,
}

/// Failure-response diagnostics are retained without rewriting their contents.
#[derive(Catalog)]
#[api(response = "temporal.api.workflowservice.v1.RespondActivityTaskFailedResponse")]
pub struct RespondActivityTaskFailedResponse {
    #[api(decided)]
    failures: Vec<Message<f::Failure>>,
}

/// Worker cancellation acknowledgement retains opaque details and validates identity.
#[derive(Catalog)]
#[api(request = "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest")]
pub struct RespondActivityTaskCanceledRequest {
    task_token: Secret,
    details: Message<c::Payloads>,
    #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity", covers = CANCELED)]
    identity: Text,
    namespace: Text,
    resource_id: Text,
    #[api(ignored = "This activity Model does not project \
        RespondActivityTaskCanceledRequest.worker_version; no semantic validation is claimed")]
    worker_version: Ignored,
    #[api(ignored = "This activity Model does not project \
        RespondActivityTaskCanceledRequest.deployment; no semantic validation is claimed")]
    deployment: Ignored,
    #[api(ignored = "This activity Model does not project \
        RespondActivityTaskCanceledRequest.deployment_options; no semantic validation is claimed")]
    deployment_options: Ignored,
}

/// Cancellation acknowledgement carries no additional lifecycle evidence.
#[derive(Catalog)]
#[api(response = "temporal.api.workflowservice.v1.RespondActivityTaskCanceledResponse")]
pub struct RespondActivityTaskCanceledResponse {}

/// Timeout and payload limits apply before the standalone activity is scheduled.
#[derive(Catalog)]
#[api(request = "temporal.api.workflowservice.v1.StartActivityExecutionRequest")]
#[api(
    any_of(schedule_to_close_timeout, start_to_close_timeout),
    rejects = "a valid StartToCloseTimeout or ScheduleToCloseTimeout",
    covers = TIMEOUTS,
)]
pub struct StartActivityExecutionRequest {
    namespace: Text,
    #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity", covers = START)]
    identity: Text,
    #[api(
        max_bytes = dc::MAX_ID_LENGTH,
        idempotency_key,
        rejects = "request ID exceeds length limit",
        covers = START,
    )]
    #[api(also(uuid, accepts))]
    request_id: Text,
    #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "activity[Ii][Dd]|activity ID")]
    #[api(covers = ATTRIBUTES)]
    activity_id: Identifier,
    #[api(required, rejects = "activityType|taskQueue|task queue", covers = STANDALONE)]
    activity_type: Message<c::ActivityType>,
    #[api(required, rejects = "activityType|taskQueue|task queue", covers = STANDALONE)]
    task_queue: Message<q::TaskQueue>,
    #[api(gte = 0, rejects = "invalid ScheduleToCloseTimeout", covers = TIMEOUTS)]
    schedule_to_close_timeout: Duration,
    #[api(gte = 0, rejects = "invalid ScheduleToStartTimeout", covers = TIMEOUTS)]
    schedule_to_start_timeout: Duration,
    #[api(gte = 0, rejects = "invalid StartToCloseTimeout", covers = TIMEOUTS)]
    start_to_close_timeout: Duration,
    #[api(gte = 0, rejects = "invalid HeartbeatTimeout", covers = TIMEOUTS)]
    heartbeat_timeout: Duration,
    retry_policy: Message<c::RetryPolicy>,
    #[api(max_len = dc::BLOB_SIZE_ERROR, rejects = "input exceeds length limit", covers = START)]
    input: Message<c::Payloads>,
    #[api(rejects = "unsupported ID reuse policy", covers = HANDLER)]
    id_reuse_policy: Enum<e::ActivityIdReusePolicy>,
    #[api(rejects = "unsupported ID conflict policy", covers = HANDLER)]
    id_conflict_policy: Enum<e::ActivityIdConflictPolicy>,
    #[api(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.search_attributes; no semantic validation is claimed")]
    search_attributes: Ignored,
    #[api(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.header; no semantic validation is claimed")]
    header: Ignored,
    #[api(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.user_metadata; no semantic validation is claimed")]
    user_metadata: Ignored,
    #[api(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.priority; no semantic validation is claimed")]
    priority: Ignored,
    #[api(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.completion_callbacks; no semantic validation is claimed")]
    completion_callbacks: Ignored,
    #[api(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.links; no semantic validation is claimed")]
    links: Ignored,
    #[api(ignored = "This activity Model does not project \
        StartActivityExecutionRequest.on_conflict_options; no semantic validation is claimed")]
    on_conflict_options: Ignored,
    #[api(gte = 0, rejects = "invalid StartDelay", covers = START_DELAY)]
    start_delay: Duration,
}

/// The assigned run identifier is observed; execution links remain outside this Model.
#[derive(Catalog)]
#[api(response = "temporal.api.workflowservice.v1.StartActivityExecutionResponse")]
pub struct StartActivityExecutionResponse {
    #[api(required)]
    run_id: Uuid,
    started: Flag,
    #[api(ignored = "This activity Model does not project \
        StartActivityExecutionResponse.link; no semantic validation is claimed")]
    link: Ignored,
}

/// Long-poll tokens require an explicit run identifier.
#[derive(Catalog)]
#[api(request = "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest")]
pub struct DescribeActivityExecutionRequest {
    namespace: Text,
    #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "activity[Ii][Dd]|activity ID")]
    #[api(covers = DESCRIBE)]
    activity_id: Identifier,
    #[api(rejects = "invalid run id", covers = DESCRIBE)]
    run_id: Uuid,
    #[api(ignored = "This activity Model does not project \
        DescribeActivityExecutionRequest.include_input; no semantic validation is claimed")]
    include_input: Ignored,
    #[api(ignored = "This activity Model does not project \
        DescribeActivityExecutionRequest.include_outcome; no semantic validation is claimed")]
    include_outcome: Ignored,
    #[api(requires = run_id, rejects = "run id is required", covers = DESCRIBE)]
    long_poll_token: Secret,
    #[api(ignored = "This activity Model does not project \
        DescribeActivityExecutionRequest.include_heartbeat_details; \
        no semantic validation is claimed")]
    include_heartbeat_details: Ignored,
    #[api(ignored = "This activity Model does not project \
        DescribeActivityExecutionRequest.include_last_failure; no semantic validation is claimed")]
    include_last_failure: Ignored,
}

/// Lifecycle metadata is required; optional external payloads remain Decided.
#[derive(Catalog)]
#[api(response = "temporal.api.workflowservice.v1.DescribeActivityExecutionResponse")]
pub struct DescribeActivityExecutionResponse {
    run_id: Uuid,
    #[api(required)]
    #[api(decided)]
    info: Message<a::ActivityExecutionInfo>,
    #[api(decided)]
    input: Message<c::Payloads>,
    #[api(decided)]
    outcome: Message<a::ActivityExecutionOutcome>,
    long_poll_token: Secret,
    #[api(ignored = "This activity Model does not project \
        DescribeActivityExecutionResponse.callbacks; no semantic validation is claimed")]
    callbacks: Ignored,
}

/// Outcome polling accepts an optional normalized run UUID.
#[derive(Catalog)]
#[api(request = "temporal.api.workflowservice.v1.PollActivityExecutionRequest")]
pub struct PollActivityExecutionRequest {
    namespace: Text,
    #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "activity[Ii][Dd]|activity ID", covers = POLL)]
    activity_id: Identifier,
    #[api(rejects = "invalid run id", covers = POLL)]
    run_id: Uuid,
}

/// The observed run identifier scopes the optional external outcome.
#[derive(Catalog)]
#[api(response = "temporal.api.workflowservice.v1.PollActivityExecutionResponse")]
pub struct PollActivityExecutionResponse {
    run_id: Uuid,
    #[api(decided)]
    outcome: Message<a::ActivityExecutionOutcome>,
}

/// Cancellation intent is separate from the worker's eventual cancellation acknowledgement.
#[derive(Catalog)]
#[api(request = "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest")]
pub struct RequestCancelActivityExecutionRequest {
    namespace: Text,
    #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "activity[Ii][Dd]|activity ID", covers = CANCEL)]
    activity_id: Identifier,
    #[api(rejects = "invalid run id", covers = CANCEL)]
    run_id: Uuid,
    #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity", covers = CANCEL)]
    identity: Text,
    #[api(
        max_bytes = dc::MAX_ID_LENGTH,
        idempotency_key,
        rejects = "request ID exceeds length limit",
        covers = REQUEST_ID,
        also(uuid, accepts),
    )]
    request_id: Text,
    #[api(max_bytes = dc::BLOB_SIZE_ERROR, rejects = "reason exceeds length limit", covers = CANCEL)]
    reason: Text,
}

/// Cancellation intent acknowledgement carries no additional lifecycle evidence.
#[derive(Catalog)]
#[api(response = "temporal.api.workflowservice.v1.RequestCancelActivityExecutionResponse")]
pub struct RequestCancelActivityExecutionResponse {}

use umpire_temporal_annotate::Binding;
use umpire_temporal_api::WorkflowServiceMethod;

enum RpcRole {
    Action(ModelAction),
    Observed(&'static str),
    OutOfScope(&'static str),
}

#[allow(
    clippy::match_same_arms,
    clippy::too_many_lines,
    reason = "Descriptor admission requires RPC declaration order, including separated role groups"
)]
fn classify(method: WorkflowServiceMethod) -> RpcRole {
    match method {
        WorkflowServiceMethod::RegisterNamespace => RpcRole::OutOfScope(
            "Namespace administration is outside this standalone activity Model",
        ),
        WorkflowServiceMethod::DescribeNamespace => RpcRole::OutOfScope(
            "SDK namespace discovery is infrastructure outside activity lifecycle validation",
        ),
        WorkflowServiceMethod::ListNamespaces
        | WorkflowServiceMethod::UpdateNamespace
        | WorkflowServiceMethod::DeprecateNamespace => RpcRole::OutOfScope(
            "Namespace administration is outside this standalone activity Model",
        ),
        WorkflowServiceMethod::StartWorkflowExecution => RpcRole::OutOfScope(
            "Workflow execution operations are outside this standalone activity Model",
        ),
        WorkflowServiceMethod::ExecuteMultiOperation => RpcRole::OutOfScope(
            "Service administration and other resource lifecycles are outside this activity Model",
        ),
        WorkflowServiceMethod::GetWorkflowExecutionHistory
        | WorkflowServiceMethod::GetWorkflowExecutionHistoryReverse
        | WorkflowServiceMethod::PollWorkflowTaskQueue
        | WorkflowServiceMethod::RespondWorkflowTaskCompleted
        | WorkflowServiceMethod::RespondWorkflowTaskFailed => RpcRole::OutOfScope(
            "Workflow execution operations are outside this standalone activity Model",
        ),
        WorkflowServiceMethod::PollActivityTaskQueue => {
            RpcRole::Action(ModelAction::Worker(Worker::Poll))
        }
        WorkflowServiceMethod::RecordActivityTaskHeartbeat
        | WorkflowServiceMethod::RecordActivityTaskHeartbeatById => RpcRole::OutOfScope(
            "This activity operation is not represented by the mini activity lifecycle",
        ),
        WorkflowServiceMethod::RespondActivityTaskCompleted => {
            RpcRole::Action(ModelAction::Worker(Worker::RespondCompleted))
        }
        WorkflowServiceMethod::RespondActivityTaskCompletedById => RpcRole::OutOfScope(
            "This activity operation is not represented by the mini activity lifecycle",
        ),
        WorkflowServiceMethod::RespondActivityTaskFailed => {
            RpcRole::Action(ModelAction::Worker(Worker::RespondFailed))
        }
        WorkflowServiceMethod::RespondActivityTaskFailedById => RpcRole::OutOfScope(
            "This activity operation is not represented by the mini activity lifecycle",
        ),
        WorkflowServiceMethod::RespondActivityTaskCanceled => {
            RpcRole::Observed("worker acknowledges client cancellation")
        }
        WorkflowServiceMethod::RespondActivityTaskCanceledById => RpcRole::OutOfScope(
            "This activity operation is not represented by the mini activity lifecycle",
        ),
        WorkflowServiceMethod::RequestCancelWorkflowExecution
        | WorkflowServiceMethod::SignalWorkflowExecution
        | WorkflowServiceMethod::SignalWithStartWorkflowExecution
        | WorkflowServiceMethod::ResetWorkflowExecution
        | WorkflowServiceMethod::TerminateWorkflowExecution
        | WorkflowServiceMethod::DeleteWorkflowExecution
        | WorkflowServiceMethod::ListOpenWorkflowExecutions
        | WorkflowServiceMethod::ListClosedWorkflowExecutions
        | WorkflowServiceMethod::ListWorkflowExecutions
        | WorkflowServiceMethod::ListArchivedWorkflowExecutions
        | WorkflowServiceMethod::ScanWorkflowExecutions
        | WorkflowServiceMethod::CountWorkflowExecutions => RpcRole::OutOfScope(
            "Workflow execution operations are outside this standalone activity Model",
        ),
        WorkflowServiceMethod::GetSearchAttributes
        | WorkflowServiceMethod::RespondQueryTaskCompleted
        | WorkflowServiceMethod::ResetStickyTaskQueue => RpcRole::OutOfScope(
            "Service administration and other resource lifecycles are outside this activity Model",
        ),
        WorkflowServiceMethod::ShutdownWorker => RpcRole::OutOfScope(
            "SDK worker shutdown bookkeeping does not change this activity Model",
        ),
        WorkflowServiceMethod::QueryWorkflow | WorkflowServiceMethod::DescribeWorkflowExecution => {
            RpcRole::OutOfScope(
                "Workflow execution operations are outside this standalone activity Model",
            )
        }
        WorkflowServiceMethod::DescribeTaskQueue | WorkflowServiceMethod::GetClusterInfo => {
            RpcRole::OutOfScope(
                "Service administration and other resource lifecycles are outside this \
                    activity Model",
            )
        }
        WorkflowServiceMethod::GetSystemInfo => RpcRole::OutOfScope(
            "SDK server capability negotiation does not observe an activity lifecycle",
        ),
        WorkflowServiceMethod::ListTaskQueuePartitions
        | WorkflowServiceMethod::CreateSchedule
        | WorkflowServiceMethod::DescribeSchedule
        | WorkflowServiceMethod::UpdateSchedule
        | WorkflowServiceMethod::PatchSchedule
        | WorkflowServiceMethod::ListScheduleMatchingTimes
        | WorkflowServiceMethod::DeleteSchedule
        | WorkflowServiceMethod::ListSchedules
        | WorkflowServiceMethod::CountSchedules
        | WorkflowServiceMethod::UpdateWorkerBuildIdCompatibility
        | WorkflowServiceMethod::GetWorkerBuildIdCompatibility
        | WorkflowServiceMethod::UpdateWorkerVersioningRules
        | WorkflowServiceMethod::GetWorkerVersioningRules
        | WorkflowServiceMethod::GetWorkerTaskReachability
        | WorkflowServiceMethod::DescribeDeployment
        | WorkflowServiceMethod::DescribeWorkerDeploymentVersion
        | WorkflowServiceMethod::ListDeployments
        | WorkflowServiceMethod::GetDeploymentReachability
        | WorkflowServiceMethod::GetCurrentDeployment
        | WorkflowServiceMethod::SetCurrentDeployment
        | WorkflowServiceMethod::SetWorkerDeploymentCurrentVersion
        | WorkflowServiceMethod::DescribeWorkerDeployment
        | WorkflowServiceMethod::DeleteWorkerDeployment
        | WorkflowServiceMethod::DeleteWorkerDeploymentVersion
        | WorkflowServiceMethod::SetWorkerDeploymentRampingVersion
        | WorkflowServiceMethod::ListWorkerDeployments
        | WorkflowServiceMethod::CreateWorkerDeployment
        | WorkflowServiceMethod::CreateWorkerDeploymentVersion
        | WorkflowServiceMethod::UpdateWorkerDeploymentVersionComputeConfig
        | WorkflowServiceMethod::ValidateWorkerDeploymentVersionComputeConfig
        | WorkflowServiceMethod::UpdateWorkerDeploymentVersionMetadata
        | WorkflowServiceMethod::SetWorkerDeploymentManager => RpcRole::OutOfScope(
            "Service administration and other resource lifecycles are outside this activity Model",
        ),
        WorkflowServiceMethod::UpdateWorkflowExecution
        | WorkflowServiceMethod::PollWorkflowExecutionUpdate => RpcRole::OutOfScope(
            "Workflow execution operations are outside this standalone activity Model",
        ),
        WorkflowServiceMethod::StartBatchOperation
        | WorkflowServiceMethod::StopBatchOperation
        | WorkflowServiceMethod::DescribeBatchOperation
        | WorkflowServiceMethod::ListBatchOperations => RpcRole::OutOfScope(
            "Service administration and other resource lifecycles are outside this activity Model",
        ),
        WorkflowServiceMethod::PollNexusTaskQueue => RpcRole::OutOfScope(
            "SDK worker control commands use Nexus polling; their contents are outside \
                this activity Model",
        ),
        WorkflowServiceMethod::RespondNexusTaskCompleted => RpcRole::OutOfScope(
            "SDK Nexus worker control completion is outside this activity Model",
        ),
        WorkflowServiceMethod::RespondNexusTaskFailed => {
            RpcRole::OutOfScope("SDK Nexus worker control failure is outside this activity Model")
        }
        WorkflowServiceMethod::UpdateActivityOptions => RpcRole::OutOfScope(
            "This activity operation is not represented by the mini activity lifecycle",
        ),
        WorkflowServiceMethod::UpdateWorkflowExecutionOptions => RpcRole::OutOfScope(
            "Workflow execution operations are outside this standalone activity Model",
        ),
        WorkflowServiceMethod::PauseActivity
        | WorkflowServiceMethod::UnpauseActivity
        | WorkflowServiceMethod::ResetActivity => RpcRole::OutOfScope(
            "This activity operation is not represented by the mini activity lifecycle",
        ),
        WorkflowServiceMethod::CreateWorkflowRule
        | WorkflowServiceMethod::DescribeWorkflowRule
        | WorkflowServiceMethod::DeleteWorkflowRule
        | WorkflowServiceMethod::ListWorkflowRules
        | WorkflowServiceMethod::TriggerWorkflowRule => RpcRole::OutOfScope(
            "Workflow execution operations are outside this standalone activity Model",
        ),
        WorkflowServiceMethod::RecordWorkerHeartbeat => RpcRole::OutOfScope(
            "SDK worker health and configuration heartbeat is infrastructure outside \
                activity lifecycle validation",
        ),
        WorkflowServiceMethod::ListWorkers
        | WorkflowServiceMethod::CountWorkers
        | WorkflowServiceMethod::UpdateTaskQueueConfig
        | WorkflowServiceMethod::FetchWorkerConfig
        | WorkflowServiceMethod::UpdateWorkerConfig
        | WorkflowServiceMethod::DescribeWorker => RpcRole::OutOfScope(
            "Service administration and other resource lifecycles are outside this activity Model",
        ),
        WorkflowServiceMethod::PauseWorkflowExecution
        | WorkflowServiceMethod::UnpauseWorkflowExecution => RpcRole::OutOfScope(
            "Workflow execution operations are outside this standalone activity Model",
        ),
        WorkflowServiceMethod::StartActivityExecution => {
            RpcRole::Observed("successful start establishes the Model initial state")
        }
        WorkflowServiceMethod::StartNexusOperationExecution => RpcRole::OutOfScope(
            "Service administration and other resource lifecycles are outside this activity Model",
        ),
        WorkflowServiceMethod::DescribeActivityExecution => {
            RpcRole::Observed("activity lifecycle phase")
        }
        WorkflowServiceMethod::DescribeNexusOperationExecution => RpcRole::OutOfScope(
            "Service administration and other resource lifecycles are outside this activity Model",
        ),
        WorkflowServiceMethod::PollActivityExecution => {
            RpcRole::Observed("SDK activity handle outcome")
        }
        WorkflowServiceMethod::PollNexusOperationExecution => RpcRole::OutOfScope(
            "Service administration and other resource lifecycles are outside this activity Model",
        ),
        WorkflowServiceMethod::ListActivityExecutions => RpcRole::OutOfScope(
            "This activity operation is not represented by the mini activity lifecycle",
        ),
        WorkflowServiceMethod::ListNexusOperationExecutions => RpcRole::OutOfScope(
            "Service administration and other resource lifecycles are outside this activity Model",
        ),
        WorkflowServiceMethod::CountActivityExecutions => RpcRole::OutOfScope(
            "This activity operation is not represented by the mini activity lifecycle",
        ),
        WorkflowServiceMethod::CountNexusOperationExecutions => RpcRole::OutOfScope(
            "Service administration and other resource lifecycles are outside this activity Model",
        ),
        WorkflowServiceMethod::RequestCancelActivityExecution => {
            RpcRole::Action(ModelAction::Client(Client::Cancel))
        }
        WorkflowServiceMethod::RequestCancelNexusOperationExecution => RpcRole::OutOfScope(
            "Service administration and other resource lifecycles are outside this activity Model",
        ),
        WorkflowServiceMethod::TerminateActivityExecution
        | WorkflowServiceMethod::DeleteActivityExecution
        | WorkflowServiceMethod::PauseActivityExecution
        | WorkflowServiceMethod::ResetActivityExecution
        | WorkflowServiceMethod::UnpauseActivityExecution
        | WorkflowServiceMethod::UpdateActivityExecutionOptions => RpcRole::OutOfScope(
            "This activity operation is not represented by the mini activity lifecycle",
        ),
        WorkflowServiceMethod::TerminateNexusOperationExecution
        | WorkflowServiceMethod::DeleteNexusOperationExecution => RpcRole::OutOfScope(
            "Service administration and other resource lifecycles are outside this activity Model",
        ),
        WorkflowServiceMethod::PollWorkflowExecutionTimeSkipping => RpcRole::OutOfScope(
            "Workflow execution operations are outside this standalone activity Model",
        ),
    }
}

/// Executable factories and exhaustive runtime bindings.
pub mod catalog {
    use super::{Binding, Client, Deadline, ModelAction, Worker, a, c, dc, f, q, w};
    pub use crate::foundation::enum_inventory::enum_inventory::*;
    use umpire_annotations::{AdmissionErrors, Catalog, ProbeCase};
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
            ModelAction::Deadline(deadline) => match deadline {
                Deadline::ScheduleToStart => Binding::Environmental("abstract \
                    schedule-to-start deadline"),
                Deadline::StartToClose => Binding::Environmental("abstract \
                    start-to-close deadline"),
            }
        )
    }
}

/// UUID intent for request IDs accepted without UUID syntax by the server.
pub fn start_request_id_intent(
    limits: &dc::Limits,
) -> Result<
    umpire_annotations::Catalog<w::StartActivityExecutionRequest>,
    umpire_annotations::AdmissionErrors,
> {
    StartActivityExecutionRequest::__request_id_also_0(limits)
}

/// UUID intent for cancellation request IDs.
pub fn cancel_request_id_intent(
    limits: &dc::Limits,
) -> Result<
    umpire_annotations::Catalog<w::RequestCancelActivityExecutionRequest>,
    umpire_annotations::AdmissionErrors,
> {
    RequestCancelActivityExecutionRequest::__request_id_also_0(limits)
}

/// Worker-supplied failures require application failure information.
pub fn worker_failure_presence(
    limits: &dc::Limits,
) -> Result<umpire_annotations::Catalog<f::Failure>, umpire_annotations::AdmissionErrors> {
    crate::foundation::api::WorkerFailure::catalog(limits)
}

/// Source-reviewed dispositions keyed by complete contextual probe identity.
pub(crate) fn accepted_probe_gaps() -> Vec<(String, umpire_annotations::AcceptedGap)> {
    Vec::new()
}
