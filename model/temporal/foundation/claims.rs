#[derive(Catalog)]
#[mirror(response = "temporal.api.common.v1.WorkflowExecution")]
/// A workflow execution's identifiers.
pub struct WorkflowExecution {
    workflow_id: Text,
    run_id: Text,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.common.v1.WorkflowType")]
/// A workflow type name.
pub struct WorkflowType {
    name: Text,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.common.v1.Payloads")]
/// An ordered collection of encoded payloads.
pub struct Payloads {
    #[mirror(decided)]
    payloads: Vec<Message<common::Payload>>,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.common.v1.Payload")]
/// Payload metadata and data remain external secrets.
pub struct Payload {
    #[mirror(decided)]
    metadata: Map<Text, Secret>,
    data: Secret,
}
#[derive(Catalog)]
#[mirror(request = "temporal.api.common.v1.ActivityType")]
/// The activity type accepted by activity attribute validation.
pub struct ActivityType {
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "activityType")]
    name: Identifier,
}
#[derive(Catalog)]
#[mirror(request = "temporal.api.common.v1.RetryPolicy")]
#[mirror(response_checks = RawRetryPolicy)]
/// Retry parameters under the server's normalization conventions.
pub struct RetryPolicy {
    #[mirror(via = views::initial_interval)]
    #[rule(gte = 0, rejects = "invalid InitialInterval")]
    initial_interval: Duration,
    #[mirror(via = views::backoff_coefficient)]
    #[rule(gte = 1, rejects = "BackoffCoefficient")]
    backoff_coefficient: Counter,
    #[mirror(via = views::maximum_interval)]
    #[rule(gte = 0, rejects = "invalid MaximumInterval")]
    maximum_interval: Duration,
    #[rule(gte = 0, rejects = "MaximumAttempts")]
    maximum_attempts: Counter,
    non_retryable_error_types: Vec<Text>,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.common.v1.RetryPolicy", supplemental)]
/// Response retry intervals checked without request-side normalization.
pub struct RawRetryPolicy {
    initial_interval: RawDuration,
    maximum_interval: RawDuration,
}
#[derive(Catalog)]
#[mirror(request = "temporal.api.failure.v1.ApplicationFailureInfo")]
/// Application failure details supplied by an activity worker.
pub struct ApplicationFailureInfo {
    r#type: Text,
    non_retryable: Flag,
    details: Message<common::Payloads>,
    #[rule(rejects = "NextRetryDelay")]
    next_retry_delay: RawDuration,
    #[rule(accepts)]
    category: Enum<enums::ApplicationErrorCategory>,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.failure.v1.Failure")]
#[allow(
    clippy::struct_field_names,
    reason = "Field names must match the upstream protobuf"
)]
/// A failure and its exhaustive contextual alternatives.
pub struct Failure {
    #[mirror(decided)]
    message: Text,
    stack_trace: Secret,
    #[mirror(decided)]
    encoded_attributes: Message<common::Payload>,
    #[mirror(decided)]
    cause: Message<failure::Failure>,
    #[mirror(decided)]
    #[mirror(
        oneof(
            ApplicationFailureInfo = "application_failure_info",
            TimeoutFailureInfo = "timeout_failure_info",
            CanceledFailureInfo = "canceled_failure_info",
            TerminatedFailureInfo = "terminated_failure_info",
            ServerFailureInfo = "server_failure_info",
            ResetWorkflowFailureInfo = "reset_workflow_failure_info",
            ActivityFailureInfo = "activity_failure_info",
            ChildWorkflowExecutionFailureInfo = "child_workflow_execution_failure_info",
            NexusOperationExecutionFailureInfo = "nexus_operation_execution_failure_info",
            NexusHandlerFailureInfo = "nexus_handler_failure_info",
        )
    )]
    failure_info: OneOf<failure::failure::FailureInfo>,
}
/// A worker failure must carry application failure information.
#[derive(Catalog)]
#[rule(
    required(application_failure_info),
    rejects = "Failure must have ApplicationFailureInfo"
)]
#[mirror(request = "temporal.api.failure.v1.Failure", supplemental)]
pub struct WorkerFailure {
    #[mirror(oneof(ApplicationFailureInfo = "application_failure_info"))]
    failure_info: OneOf<failure::failure::FailureInfo>,
}
#[derive(Catalog)]
#[mirror(request = "temporal.api.taskqueue.v1.TaskQueue")]
/// A task queue name and its declared kind.
pub struct TaskQueue {
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "taskQueue|task queue")]
    name: Identifier,
    #[rule(accepts)]
    kind: Enum<enums::TaskQueueKind>,
    normal_name: Text,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.failure.v1.TimeoutFailureInfo")]
/// A timeout failure with its last heartbeat payloads.
pub struct TimeoutFailureInfo {
    #[rule(accepts)]
    timeout_type: Enum<enums::TimeoutType>,
    #[mirror(decided)]
    last_heartbeat_details: Message<common::Payloads>,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.failure.v1.CanceledFailureInfo")]
/// Cancellation details supplied by the worker.
pub struct CanceledFailureInfo {
    #[mirror(decided)]
    details: Message<common::Payloads>,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.failure.v1.TerminatedFailureInfo")]
/// Termination information outside activity state.
pub struct TerminatedFailureInfo {}
#[derive(Catalog)]
#[mirror(response = "temporal.api.failure.v1.ServerFailureInfo")]
/// Whether a server failure is retryable.
pub struct ServerFailureInfo {
    #[mirror(decided)]
    non_retryable: Flag,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.failure.v1.ResetWorkflowFailureInfo")]
/// Heartbeat details retained across a workflow reset.
pub struct ResetWorkflowFailureInfo {
    last_heartbeat_details: Message<common::Payloads>,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.failure.v1.ActivityFailureInfo")]
/// The scheduled activity and its retry status.
pub struct ActivityFailureInfo {
    scheduled_event_id: Counter,
    started_event_id: Counter,
    identity: Text,
    #[mirror(via = views::activity_type)]
    activity_type: Text,
    activity_id: Text,
    #[rule(accepts)]
    retry_state: Enum<enums::RetryState>,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo")]
/// The child workflow execution and its retry status.
pub struct ChildWorkflowExecutionFailureInfo {
    namespace: Text,
    workflow_execution: Message<common::WorkflowExecution>,
    workflow_type: Message<common::WorkflowType>,
    initiated_event_id: Counter,
    started_event_id: Counter,
    #[rule(accepts)]
    retry_state: Enum<enums::RetryState>,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.failure.v1.NexusOperationFailureInfo")]
/// Operation failure metadata and private correlation tokens.
pub struct NexusOperationFailureInfo {
    scheduled_event_id: Counter,
    endpoint: Text,
    service: Text,
    operation: Text,
    operation_id: Secret,
    operation_token: Secret,
}
#[derive(Catalog)]
#[mirror(response = "temporal.api.failure.v1.NexusHandlerFailureInfo")]
/// A handler failure and its retry disposition.
pub struct NexusHandlerFailureInfo {
    r#type: Text,
    #[rule(accepts)]
    retry_behavior: Enum<enums::NexusHandlerErrorRetryBehavior>,
}
#[derive(Catalog)]
#[mirror(response = "google.protobuf.Duration")]
/// The raw seconds and nanoseconds of a protocol duration.
pub struct ProtoDuration {
    seconds: Counter,
    nanos: Counter,
}
#[derive(Catalog)]
#[mirror(response = "google.protobuf.Timestamp")]
/// The raw seconds and nanoseconds of a protocol timestamp.
pub struct ProtoTimestamp {
    seconds: Counter,
    nanos: Counter,
}

/// Start correlation reads a payload from the actual header map.
#[derive(Catalog)]
#[mirror(response = "temporal.api.common.v1.Header")]
pub struct Header {
    #[mirror(decided)]
    fields: Map<Text, Message<common::Payload>>,
}
