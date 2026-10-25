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
    state_transition_count: Counter,
    state_size_bytes: Counter,
    canceled_reason: Text,
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
    #[mirror(also(text, via = crate::foundation::views::poll_normal_name,
        max_len = 0, rejects = "only sticky queues can have normal name"))]
    task_queue: Message<q::TaskQueue>,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity")]
    identity: Text,
}
/// Polled tasks carry opaque tokens and externally supplied activity payloads.
#[derive(Catalog)]
#[allow(
    clippy::duplicated_attributes,
    reason = "Presence rules may share dependency operands"
)]
#[rule(requires(activity_id, task_token))]
#[rule(requires(input, task_token))]
#[mirror(response = "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse")]
pub struct PollActivityTaskQueueResponse {
    task_token: Secret,
    workflow_namespace: Text,
    #[mirror(decided)]
    activity_type: Message<c::ActivityType>,
    activity_id: Text,
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
    activity_run_id: Uuid,
}
/// Completion transports worker results under the frontend identity limit.
#[derive(Catalog)]
#[rule(required(task_token), rejects = "Task token not set")]
#[mirror(
    request = "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest"
)]
pub struct RespondActivityTaskCompletedRequest {
    task_token: Secret,
    result: Message<c::Payloads>,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity")]
    identity: Text,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "Namespace length exceeds limit")]
    namespace: Text,
    resource_id: Text,
}
/// Completion acknowledgement carries no additional lifecycle evidence.
#[derive(Catalog)]
#[mirror(
    response = "temporal.api.workflowservice.v1.RespondActivityTaskCompletedResponse"
)]
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
}
/// Cancellation acknowledgement carries no additional lifecycle evidence.
#[derive(Catalog)]
#[mirror(
    response = "temporal.api.workflowservice.v1.RespondActivityTaskCanceledResponse"
)]
pub struct RespondActivityTaskCanceledResponse {}
/// Timeout and payload limits apply before the standalone activity is scheduled.
#[derive(Catalog)]
#[rule(
    required(activity_type, task_queue),
    rejects = "activityType|taskQueue|task queue"
)]
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
    #[mirror(also(counter, via = crate::foundation::views::start_retry_order,
        gte = 1, rejects = "MaximumInterval cannot be less than InitialInterval"))]
    retry_policy: Message<c::RetryPolicy>,
    #[rule(max_len = dc::BLOB_SIZE_ERROR, rejects = "input exceeds length limit")]
    input: Message<c::Payloads>,
    #[rule(rejects = "unsupported ID reuse policy")]
    id_reuse_policy: Enum<e::ActivityIdReusePolicy>,
    #[rule(rejects = "unsupported ID conflict policy")]
    id_conflict_policy: Enum<e::ActivityIdConflictPolicy>,
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
    long_poll_token: Secret,
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
#[mirror(
    request = "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest"
)]
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
#[mirror(
    response = "temporal.api.workflowservice.v1.RequestCancelActivityExecutionResponse"
)]
pub struct RequestCancelActivityExecutionResponse {}
