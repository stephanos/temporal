//! Shared upstream message catalogs, independent of activity state.
#![allow(dead_code, reason = "Mirror fields describe the API and are never instantiated")]

use super::{dc, validators, views};
use umpire_temporal_annotate::{
    Catalog,
    kinds::{
        Counter, Duration, Enum, Flag, Identifier, Ignored, Map, Message, OneOf, RawDuration,
        Secret, Text,
    },
};
use umpire_temporal_api::temporal::api::{
    common::v1 as common, enums::v1 as enums, failure::v1 as failure,
};

const RETRY: &str = validators::retrypolicy::Validate;
const FAILURE: &str = validators::frontend::WorkflowHandler_RespondActivityTaskFailed;

#[derive(Catalog)]
#[api(response = "temporal.api.common.v1.WorkflowExecution")]
/// A workflow execution's identifiers.
pub struct WorkflowExecution {
    workflow_id: Text,
    run_id: Text,
}

#[derive(Catalog)]
#[api(response = "temporal.api.common.v1.WorkflowType")]
/// A workflow type name.
pub struct WorkflowType {
    name: Text,
}

#[derive(Catalog)]
#[api(response = "temporal.api.common.v1.Payloads")]
/// An ordered collection of encoded payloads.
pub struct Payloads {
    #[api(decided)]
    payloads: Vec<Message<common::Payload>>,
}

#[derive(Catalog)]
#[api(response = "temporal.api.common.v1.Payload")]
/// Payload metadata and data remain external secrets.
pub struct Payload {
    #[api(decided)]
    metadata: Map<Text, Secret>,
    data: Secret,
    #[api(ignored = "External payload storage handles are unsupported by the mini activity \
        callback and remain opaque on transport")]
    external_payloads: Ignored,
}

#[derive(Catalog)]
#[api(request = "temporal.api.common.v1.ActivityType")]
#[api(covers = validators::activity::validateAndNormalizeActivityAttributes)]
/// The activity type accepted by activity attribute validation.
pub struct ActivityType {
    #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "activityType")]
    name: Identifier,
}

#[derive(Catalog)]
#[api(request = "temporal.api.common.v1.RetryPolicy", covers = RETRY)]
#[api(response_checks = RawRetryPolicy)]
/// Retry parameters under the server's normalization conventions.
pub struct RetryPolicy {
    #[api(gte = 0, via = views::initial_interval, rejects = "invalid InitialInterval")]
    initial_interval: Duration,
    #[api(gte = 1, via = views::backoff_coefficient, rejects = "BackoffCoefficient")]
    backoff_coefficient: Counter,
    #[api(gte = 0, via = views::maximum_interval, rejects = "invalid MaximumInterval")]
    maximum_interval: Duration,
    #[api(gte = 0, rejects = "MaximumAttempts")]
    maximum_attempts: Counter,
    non_retryable_error_types: Vec<Text>,
}

#[derive(Catalog)]
#[api(response = "temporal.api.common.v1.RetryPolicy", supplemental)]
/// Response retry intervals checked without request-side normalization.
pub struct RawRetryPolicy {
    initial_interval: RawDuration,
    maximum_interval: RawDuration,
}

#[derive(Catalog)]
#[api(request = "temporal.api.failure.v1.ApplicationFailureInfo", covers = FAILURE)]
/// Application failure details supplied by an activity worker.
pub struct ApplicationFailureInfo {
    r#type: Text,
    non_retryable: Flag,
    details: Message<common::Payloads>,
    #[api(covers = validators::frontend::validateActivityFailureNextRetryDelays,
        rejects = "NextRetryDelay")]
    next_retry_delay: RawDuration,
    #[api(accepts)]
    category: Enum<enums::ApplicationErrorCategory>,
}

#[derive(Catalog)]
#[api(response = "temporal.api.failure.v1.Failure")]
#[allow(clippy::struct_field_names, reason = "Field names must match the upstream protobuf")]
/// A failure and its exhaustive contextual alternatives.
pub struct Failure {
    #[api(decided)]
    message: Text,
    #[api(ignored = "This activity Model does not project Failure.source; \
        no semantic validation is claimed")]
    source: Ignored,
    stack_trace: Secret,
    #[api(decided)]
    encoded_attributes: Message<common::Payload>,
    #[api(decided)]
    cause: Message<failure::Failure>,
    #[api(decided)]
    #[api(oneof(
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
    ))]
    failure_info: OneOf<failure::failure::FailureInfo>,
}

/// A worker failure must carry application failure information.
#[derive(Catalog)]
#[api(request = "temporal.api.failure.v1.Failure", supplemental, covers = FAILURE)]
pub struct WorkerFailure {
    #[api(oneof(ApplicationFailureInfo = "application_failure_info"))]
    #[api(required, rejects = "Failure must have ApplicationFailureInfo")]
    failure_info: OneOf<failure::failure::FailureInfo>,
}

#[derive(Catalog)]
#[api(request = "temporal.api.taskqueue.v1.TaskQueue")]
/// A task queue name and its declared kind.
pub struct TaskQueue {
    #[api(max_bytes = dc::MAX_ID_LENGTH, covers = validators::tqid::validate,
        rejects = "taskQueue|task queue")]
    name: Identifier,
    #[api(accepts, covers = validators::tqid::normalizeAndValidate)]
    kind: Enum<enums::TaskQueueKind>,
    normal_name: Text,
}

#[derive(Catalog)]
#[api(response = "temporal.api.failure.v1.TimeoutFailureInfo", covers = FAILURE)]
/// A timeout failure with its last heartbeat payloads.
pub struct TimeoutFailureInfo {
    #[api(accepts)]
    timeout_type: Enum<enums::TimeoutType>,
    #[api(decided)]
    last_heartbeat_details: Message<common::Payloads>,
}

#[derive(Catalog)]
#[api(response = "temporal.api.failure.v1.CanceledFailureInfo")]
/// Cancellation details supplied by the worker.
pub struct CanceledFailureInfo {
    #[api(decided)]
    details: Message<common::Payloads>,
    #[api(ignored = "This activity Model does not project CanceledFailureInfo.identity; \
        no semantic validation is claimed")]
    identity: Ignored,
}

#[derive(Catalog)]
#[api(response = "temporal.api.failure.v1.TerminatedFailureInfo")]
/// Termination information outside activity state.
pub struct TerminatedFailureInfo {
    #[api(ignored = "This activity Model does not project TerminatedFailureInfo.identity; \
        no semantic validation is claimed")]
    identity: Ignored,
}

#[derive(Catalog)]
#[api(response = "temporal.api.failure.v1.ServerFailureInfo")]
/// Whether a server failure is retryable.
pub struct ServerFailureInfo {
    #[api(decided)]
    non_retryable: Flag,
}

#[derive(Catalog)]
#[api(response = "temporal.api.failure.v1.ResetWorkflowFailureInfo")]
/// Heartbeat details retained across a workflow reset.
pub struct ResetWorkflowFailureInfo {
    last_heartbeat_details: Message<common::Payloads>,
}

#[derive(Catalog)]
#[api(response = "temporal.api.failure.v1.ActivityFailureInfo", covers = FAILURE)]
/// The scheduled activity and its retry status.
pub struct ActivityFailureInfo {
    scheduled_event_id: Counter,
    started_event_id: Counter,
    identity: Text,
    #[api(via = views::activity_type)]
    activity_type: Text,
    activity_id: Text,
    #[api(accepts)]
    retry_state: Enum<enums::RetryState>,
}

#[derive(Catalog)]
#[api(response = "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo", covers = FAILURE)]
/// The child workflow execution and its retry status.
pub struct ChildWorkflowExecutionFailureInfo {
    namespace: Text,
    workflow_execution: Message<common::WorkflowExecution>,
    workflow_type: Message<common::WorkflowType>,
    initiated_event_id: Counter,
    started_event_id: Counter,
    #[api(accepts)]
    retry_state: Enum<enums::RetryState>,
}

#[derive(Catalog)]
#[api(response = "temporal.api.failure.v1.NexusOperationFailureInfo")]
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
#[api(response = "temporal.api.failure.v1.NexusHandlerFailureInfo", covers = FAILURE)]
/// A handler failure and its retry disposition.
pub struct NexusHandlerFailureInfo {
    r#type: Text,
    #[api(accepts)]
    retry_behavior: Enum<enums::NexusHandlerErrorRetryBehavior>,
}

#[derive(Catalog)]
#[api(response = "google.protobuf.Duration")]
/// The raw seconds and nanoseconds of a protocol duration.
pub struct ProtoDuration {
    seconds: Counter,
    nanos: Counter,
}

#[derive(Catalog)]
#[api(response = "google.protobuf.Timestamp")]
/// The raw seconds and nanoseconds of a protocol timestamp.
pub struct ProtoTimestamp {
    seconds: Counter,
    nanos: Counter,
}
