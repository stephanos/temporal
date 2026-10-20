//! Exhaustive activity transport annotations, independent of the Model's runtime state.
#![allow(deprecated, reason = "Exhaustive catalogs must name deprecated upstream protobuf fields")]

use crate::activity::{self, Client, Deadline, Worker};
use prost::Message;
use umpire_annotations::{Catalog, Expectation};

#[macro_use]
#[path = "api_requests.rs"]
mod requests;
use umpire_temporal_api::temporal::api::{
    activity::v1::{ActivityExecutionInfo, ActivityExecutionOutcome, activity_execution_outcome},
    common::v1::{ActivityType, Payload, Payloads, RetryPolicy},
    enums::v1::{
        ActivityExecutionStatus, ActivityIdConflictPolicy, ActivityIdReusePolicy,
        ActivityTaskFailedCause, ApplicationErrorCategory, PendingActivityState, RetryState,
        TaskQueueKind, TimeoutType,
    },
    failure::v1::{
        ActivityFailureInfo, ApplicationFailureInfo, CanceledFailureInfo,
        ChildWorkflowExecutionFailureInfo, Failure, NexusHandlerFailureInfo,
        NexusOperationFailureInfo, ResetWorkflowFailureInfo, ServerFailureInfo,
        TerminatedFailureInfo, TimeoutFailureInfo, failure,
    },
    taskqueue::v1::TaskQueue,
    workflowservice::v1::{
        DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        PollActivityExecutionRequest, PollActivityExecutionResponse, PollActivityTaskQueueRequest,
        PollActivityTaskQueueResponse, RequestCancelActivityExecutionRequest,
        RequestCancelActivityExecutionResponse, RespondActivityTaskCanceledRequest,
        RespondActivityTaskCanceledResponse, RespondActivityTaskCompletedRequest,
        RespondActivityTaskCompletedResponse, RespondActivityTaskFailedRequest,
        RespondActivityTaskFailedResponse, StartActivityExecutionRequest,
        StartActivityExecutionResponse,
    },
};
use umpire_wasm::google::protobuf::{Duration as ProtoDuration, Timestamp as ProtoTimestamp};
type ModelAction = <activity::World as umpire_dsl::Machine>::Action;

#[path = "api_transport.rs"]
mod transport;

pub use transport::{AnnotationResult, annotate};
use transport::{duration_view, raw_duration_view, uuid_view};

/// Typed adapter for the protobuf map entry represented by prost as a map.
#[derive(Clone, Default, PartialEq)]
pub struct MetadataEntry {
    key: String,
    value: Vec<u8>,
}

umpire_temporal_annotate::annotate! {
    pub catalog catalog;
    enum retry_state: RetryState = "temporal.api.enums.v1.RetryState" {
        RetryState::Unspecified = "RETRY_STATE_UNSPECIFIED",
        RetryState::InProgress = "RETRY_STATE_IN_PROGRESS",
        RetryState::NonRetryableFailure = "RETRY_STATE_NON_RETRYABLE_FAILURE",
        RetryState::Timeout = "RETRY_STATE_TIMEOUT",
        RetryState::MaximumAttemptsReached = "RETRY_STATE_MAXIMUM_ATTEMPTS_REACHED",
        RetryState::RetryPolicyNotSet = "RETRY_STATE_RETRY_POLICY_NOT_SET",
        RetryState::InternalServerError = "RETRY_STATE_INTERNAL_SERVER_ERROR",
        RetryState::CancelRequested = "RETRY_STATE_CANCEL_REQUESTED",
    }
    enum activity_execution_status: ActivityExecutionStatus = "temporal.api.enums.v1.ActivityExecutionStatus" {
        ActivityExecutionStatus::Unspecified = "ACTIVITY_EXECUTION_STATUS_UNSPECIFIED",
        ActivityExecutionStatus::Running = "ACTIVITY_EXECUTION_STATUS_RUNNING",
        ActivityExecutionStatus::Completed = "ACTIVITY_EXECUTION_STATUS_COMPLETED",
        ActivityExecutionStatus::Failed = "ACTIVITY_EXECUTION_STATUS_FAILED",
        ActivityExecutionStatus::Canceled = "ACTIVITY_EXECUTION_STATUS_CANCELED",
        ActivityExecutionStatus::Terminated = "ACTIVITY_EXECUTION_STATUS_TERMINATED",
        ActivityExecutionStatus::TimedOut = "ACTIVITY_EXECUTION_STATUS_TIMED_OUT",
        ActivityExecutionStatus::Paused = "ACTIVITY_EXECUTION_STATUS_PAUSED",
    }
    enum pending_activity_state: PendingActivityState = "temporal.api.enums.v1.PendingActivityState" {
        PendingActivityState::Unspecified = "PENDING_ACTIVITY_STATE_UNSPECIFIED",
        PendingActivityState::Scheduled = "PENDING_ACTIVITY_STATE_SCHEDULED",
        PendingActivityState::Started = "PENDING_ACTIVITY_STATE_STARTED",
        PendingActivityState::CancelRequested = "PENDING_ACTIVITY_STATE_CANCEL_REQUESTED",
        PendingActivityState::Paused = "PENDING_ACTIVITY_STATE_PAUSED",
        PendingActivityState::PauseRequested = "PENDING_ACTIVITY_STATE_PAUSE_REQUESTED",
    }
    enum activity_id_reuse_policy: ActivityIdReusePolicy = "temporal.api.enums.v1.ActivityIdReusePolicy" {
        ActivityIdReusePolicy::Unspecified = "ACTIVITY_ID_REUSE_POLICY_UNSPECIFIED",
        ActivityIdReusePolicy::AllowDuplicate = "ACTIVITY_ID_REUSE_POLICY_ALLOW_DUPLICATE",
        ActivityIdReusePolicy::AllowDuplicateFailedOnly = "ACTIVITY_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY",
        ActivityIdReusePolicy::RejectDuplicate = "ACTIVITY_ID_REUSE_POLICY_REJECT_DUPLICATE",
    }
    enum activity_id_conflict_policy: ActivityIdConflictPolicy = "temporal.api.enums.v1.ActivityIdConflictPolicy" {
        ActivityIdConflictPolicy::Unspecified = "ACTIVITY_ID_CONFLICT_POLICY_UNSPECIFIED",
        ActivityIdConflictPolicy::Fail = "ACTIVITY_ID_CONFLICT_POLICY_FAIL",
        ActivityIdConflictPolicy::UseExisting = "ACTIVITY_ID_CONFLICT_POLICY_USE_EXISTING",
    }
    enum task_queue_kind: TaskQueueKind = "temporal.api.enums.v1.TaskQueueKind" {
        TaskQueueKind::Unspecified = "TASK_QUEUE_KIND_UNSPECIFIED",
        TaskQueueKind::Normal = "TASK_QUEUE_KIND_NORMAL",
        TaskQueueKind::Sticky = "TASK_QUEUE_KIND_STICKY",
        TaskQueueKind::WorkerCommands = "TASK_QUEUE_KIND_WORKER_COMMANDS",
    }
    enum application_error_category: ApplicationErrorCategory = "temporal.api.enums.v1.ApplicationErrorCategory" {
        ApplicationErrorCategory::Unspecified = "APPLICATION_ERROR_CATEGORY_UNSPECIFIED",
        ApplicationErrorCategory::Benign = "APPLICATION_ERROR_CATEGORY_BENIGN",
    }
    enum activity_task_failed_cause: ActivityTaskFailedCause = "temporal.api.enums.v1.ActivityTaskFailedCause" {
        ActivityTaskFailedCause::Unspecified = "ACTIVITY_TASK_FAILED_CAUSE_UNSPECIFIED",
        ActivityTaskFailedCause::PayloadsTooLarge = "ACTIVITY_TASK_FAILED_CAUSE_PAYLOADS_TOO_LARGE",
        ActivityTaskFailedCause::ExternalStorageFailure = "ACTIVITY_TASK_FAILED_CAUSE_EXTERNAL_STORAGE_FAILURE",
        ActivityTaskFailedCause::ActivityWorkerUnhandledFailure = "ACTIVITY_TASK_FAILED_CAUSE_ACTIVITY_WORKER_UNHANDLED_FAILURE",
    }
    enum timeout_type: TimeoutType = "temporal.api.enums.v1.TimeoutType" {
        TimeoutType::Unspecified = "TIMEOUT_TYPE_UNSPECIFIED",
        TimeoutType::StartToClose = "TIMEOUT_TYPE_START_TO_CLOSE",
        TimeoutType::ScheduleToStart = "TIMEOUT_TYPE_SCHEDULE_TO_START",
        TimeoutType::ScheduleToClose = "TIMEOUT_TYPE_SCHEDULE_TO_CLOSE",
        TimeoutType::Heartbeat = "TIMEOUT_TYPE_HEARTBEAT",
    }
    message response activity_execution_outcome: ActivityExecutionOutcome = "temporal.api.activity.v1.ActivityExecutionOutcome" {
        retry_state: Enum(retry_state) + Observed;
        value: Bytes + Decided via Vec<u8> (|m: &ActivityExecutionOutcome| match &m.value { Some(activity_execution_outcome::Value::Result(v)) => v.encode_to_vec(), Some(activity_execution_outcome::Value::Failure(v)) => v.encode_to_vec(), None => Vec::new() }, |_: &mut ActivityExecutionOutcome, _: Vec<u8>| {}) variants { activity_execution_outcome::Value::Result = "result", activity_execution_outcome::Value::Failure = "failure", };
        rules { one_of(result, failure); }
    }
    message response activity_execution_info: ActivityExecutionInfo = "temporal.api.activity.v1.ActivityExecutionInfo" {
        activity_id: Text + Observed;
        run_id: Uuid + Observed via Option<String> (|m: &ActivityExecutionInfo| uuid_view(&m.run_id), |_: &mut ActivityExecutionInfo, _: Option<String>| {});
        activity_type: Bytes + Decided via Vec<u8> (|m: &ActivityExecutionInfo| m.activity_type.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut ActivityExecutionInfo, _: Vec<u8>| {});
        status: Enum(activity_execution_status) + Observed;
        run_state: Enum(pending_activity_state) + Observed;
        task_queue: Text + Observed;
        schedule_to_close_timeout: Duration + Observed via Option<umpire_annotations::Duration> (|m: &ActivityExecutionInfo| raw_duration_view(m.schedule_to_close_timeout.as_ref()), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Duration>| {});
        schedule_to_start_timeout: Duration + Observed via Option<umpire_annotations::Duration> (|m: &ActivityExecutionInfo| raw_duration_view(m.schedule_to_start_timeout.as_ref()), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Duration>| {});
        start_to_close_timeout: Duration + Observed via Option<umpire_annotations::Duration> (|m: &ActivityExecutionInfo| raw_duration_view(m.start_to_close_timeout.as_ref()), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Duration>| {});
        heartbeat_timeout: Duration + Observed via Option<umpire_annotations::Duration> (|m: &ActivityExecutionInfo| raw_duration_view(m.heartbeat_timeout.as_ref()), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Duration>| {});
        retry_policy: Bytes + Decided via Vec<u8> (|m: &ActivityExecutionInfo| m.retry_policy.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut ActivityExecutionInfo, _: Vec<u8>| {});
        heartbeat_details: Bytes + Decided via Vec<u8> (|m: &ActivityExecutionInfo| m.heartbeat_details.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut ActivityExecutionInfo, _: Vec<u8>| {});
        last_heartbeat_time: Timestamp + Observed via Option<umpire_annotations::Timestamp> (|m: &ActivityExecutionInfo| m.last_heartbeat_time.as_ref().map(|v| umpire_annotations::Timestamp { seconds: v.seconds, nanos: v.nanos }), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Timestamp>| {});
        last_started_time: Timestamp + Observed via Option<umpire_annotations::Timestamp> (|m: &ActivityExecutionInfo| m.last_started_time.as_ref().map(|v| umpire_annotations::Timestamp { seconds: v.seconds, nanos: v.nanos }), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Timestamp>| {});
        attempt: Counter[gte = 0] + Observed;
        execution_duration: Duration + Observed via Option<umpire_annotations::Duration> (|m: &ActivityExecutionInfo| raw_duration_view(m.execution_duration.as_ref()), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Duration>| {});
        schedule_time: Timestamp + Observed via Option<umpire_annotations::Timestamp> (|m: &ActivityExecutionInfo| m.schedule_time.as_ref().map(|v| umpire_annotations::Timestamp { seconds: v.seconds, nanos: v.nanos }), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Timestamp>| {});
        expiration_time: Timestamp + Observed via Option<umpire_annotations::Timestamp> (|m: &ActivityExecutionInfo| m.expiration_time.as_ref().map(|v| umpire_annotations::Timestamp { seconds: v.seconds, nanos: v.nanos }), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Timestamp>| {});
        close_time: Timestamp + Observed via Option<umpire_annotations::Timestamp> (|m: &ActivityExecutionInfo| m.close_time.as_ref().map(|v| umpire_annotations::Timestamp { seconds: v.seconds, nanos: v.nanos }), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Timestamp>| {});
        last_failure: Bytes + Decided via Vec<u8> (|m: &ActivityExecutionInfo| m.last_failure.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut ActivityExecutionInfo, _: Vec<u8>| {});
        last_worker_identity: Text + Observed;
        current_retry_interval: Duration + Observed via Option<umpire_annotations::Duration> (|m: &ActivityExecutionInfo| raw_duration_view(m.current_retry_interval.as_ref()), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Duration>| {});
        last_attempt_complete_time: Timestamp + Observed via Option<umpire_annotations::Timestamp> (|m: &ActivityExecutionInfo| m.last_attempt_complete_time.as_ref().map(|v| umpire_annotations::Timestamp { seconds: v.seconds, nanos: v.nanos }), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Timestamp>| {});
        next_attempt_schedule_time: Timestamp + Observed via Option<umpire_annotations::Timestamp> (|m: &ActivityExecutionInfo| m.next_attempt_schedule_time.as_ref().map(|v| umpire_annotations::Timestamp { seconds: v.seconds, nanos: v.nanos }), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Timestamp>| {});
        last_deployment_version: Bytes + Ignored("This activity Model does not project ActivityExecutionInfo.last_deployment_version; no semantic validation is claimed") opaque;
        priority: Bytes + Ignored("This activity Model does not project ActivityExecutionInfo.priority; no semantic validation is claimed") opaque;
        state_transition_count: Counter + Observed;
        state_size_bytes: Counter + Observed;
        search_attributes: Bytes + Ignored("This activity Model does not project ActivityExecutionInfo.search_attributes; no semantic validation is claimed") opaque;
        header: Bytes + Ignored("This activity Model does not project ActivityExecutionInfo.header; no semantic validation is claimed") opaque;
        user_metadata: Bytes + Ignored("This activity Model does not project ActivityExecutionInfo.user_metadata; no semantic validation is claimed") opaque;
        canceled_reason: Text + Observed;
        links: Bytes + Ignored("This activity Model does not project ActivityExecutionInfo.links; no semantic validation is claimed") opaque;
        total_heartbeat_count: Counter + Observed;
        sdk_name: Text + Observed;
        sdk_version: Text + Observed;
        start_delay: Duration + Observed via Option<umpire_annotations::Duration> (|m: &ActivityExecutionInfo| raw_duration_view(m.start_delay.as_ref()), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Duration>| {});
        execution_time: Timestamp + Observed via Option<umpire_annotations::Timestamp> (|m: &ActivityExecutionInfo| m.execution_time.as_ref().map(|v| umpire_annotations::Timestamp { seconds: v.seconds, nanos: v.nanos }), |_: &mut ActivityExecutionInfo, _: Option<umpire_annotations::Timestamp>| {});
    }
    message response payloads: Payloads = "temporal.api.common.v1.Payloads" {
        payloads: Bytes + Decided via Vec<Vec<u8>> (|m: &Payloads| m.payloads.iter().map(Message::encode_to_vec).collect::<Vec<_>>(), |_: &mut Payloads, _: Vec<Vec<u8>>| {});
    }
    message response payload: Payload = "temporal.api.common.v1.Payload" {
        metadata: Secret + Decided via Vec<Vec<u8>> (|m: &Payload| m.metadata.values().cloned().collect::<Vec<_>>(), |m: &mut Payload, values: Vec<Vec<u8>>| { if values.is_empty() { m.metadata.clear(); } });
        data: Secret + Decided;
        external_payloads: Bytes + Ignored("External payload storage handles are unsupported by the mini activity callback and remain opaque on transport") opaque;
    }
    message request activity_type: ActivityType = "temporal.api.common.v1.ActivityType" {
        name: Identifier[max_bytes = 1000] + Decided => Rejects(3, "activityType") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeActivityAttributes";
    }
    message request retry_policy: RetryPolicy = "temporal.api.common.v1.RetryPolicy" {
        initial_interval: Duration[gte = 0] + Decided via Option<umpire_annotations::Duration> (|m: &RetryPolicy| if m.maximum_attempts == 1 { None } else { duration_view(m.initial_interval.as_ref()) }, |_: &mut RetryPolicy, _: Option<umpire_annotations::Duration>| {}) => Rejects(3, "invalid InitialInterval") covers "go.temporal.io/server/common/retrypolicy.Validate";
        backoff_coefficient: Counter[gte = 1] + Decided via i32 (|m: &RetryPolicy| i32::from(m.maximum_attempts == 1 || m.backoff_coefficient == 0.0 || m.backoff_coefficient.partial_cmp(&1.0) != Some(std::cmp::Ordering::Less)), |_: &mut RetryPolicy, _: i32| {}) => Rejects(3, "BackoffCoefficient") covers "go.temporal.io/server/common/retrypolicy.Validate";
        maximum_interval: Duration[gte = 0] + Decided via Option<umpire_annotations::Duration> (|m: &RetryPolicy| if m.maximum_attempts == 1 { None } else { duration_view(m.maximum_interval.as_ref()) }, |_: &mut RetryPolicy, _: Option<umpire_annotations::Duration>| {}) => Rejects(3, "invalid MaximumInterval") covers "go.temporal.io/server/common/retrypolicy.Validate";
        maximum_attempts: Counter[gte = 0] + Decided => Rejects(3, "MaximumAttempts") covers "go.temporal.io/server/common/retrypolicy.Validate";
        non_retryable_error_types: Text + Decided;
    }
    message request application_failure_info: ApplicationFailureInfo = "temporal.api.failure.v1.ApplicationFailureInfo" {
        r#type: Text + Decided;
        non_retryable: Counter + Decided via i32 (|m: &ApplicationFailureInfo| i32::from(m.non_retryable), |_: &mut ApplicationFailureInfo, _: i32| {});
        details: Bytes + Decided via Vec<u8> (|m: &ApplicationFailureInfo| m.details.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut ApplicationFailureInfo, _: Vec<u8>| {});
        next_retry_delay: Duration + Decided via Option<umpire_annotations::Duration> (|m: &ApplicationFailureInfo| m.next_retry_delay.as_ref().map(|v| umpire_annotations::Duration { seconds: v.seconds, nanos: v.nanos }), |_: &mut ApplicationFailureInfo, _: Option<umpire_annotations::Duration>| {}) => Rejects(3, "NextRetryDelay") covers "go.temporal.io/server/service/frontend.validateActivityFailureNextRetryDelays";
        category: Enum(application_error_category) + Decided => Accepts covers "go.temporal.io/server/service/frontend.WorkflowHandler.RespondActivityTaskFailed";
    }
    message response failure: Failure = "temporal.api.failure.v1.Failure" {
        message: Text + Decided;
        source: Bytes + Ignored("This activity Model does not project Failure.source; no semantic validation is claimed") opaque;
        stack_trace: Bytes + Ignored("This activity Model does not project Failure.stack_trace; no semantic validation is claimed") opaque;
        encoded_attributes: Bytes + Decided via Vec<u8> (|m: &Failure| m.encoded_attributes.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut Failure, _: Vec<u8>| {});
        cause: Bytes + Decided via Vec<u8> (|m: &Failure| m.cause.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut Failure, _: Vec<u8>| {});
        failure_info: Bytes + Decided via Vec<u8> (|m: &Failure| match &m.failure_info { Some(failure::FailureInfo::ApplicationFailureInfo(v)) => v.encode_to_vec(), Some(failure::FailureInfo::TimeoutFailureInfo(v)) => v.encode_to_vec(), Some(failure::FailureInfo::CanceledFailureInfo(v)) => v.encode_to_vec(), Some(failure::FailureInfo::TerminatedFailureInfo(v)) => v.encode_to_vec(), Some(failure::FailureInfo::ServerFailureInfo(v)) => v.encode_to_vec(), Some(failure::FailureInfo::ResetWorkflowFailureInfo(v)) => v.encode_to_vec(), Some(failure::FailureInfo::ActivityFailureInfo(v)) => v.encode_to_vec(), Some(failure::FailureInfo::ChildWorkflowExecutionFailureInfo(v)) => v.encode_to_vec(), Some(failure::FailureInfo::NexusOperationExecutionFailureInfo(v)) => v.encode_to_vec(), Some(failure::FailureInfo::NexusHandlerFailureInfo(v)) => v.encode_to_vec(), None => Vec::new() }, |_: &mut Failure, _: Vec<u8>| {}) variants {
            failure::FailureInfo::ApplicationFailureInfo = "application_failure_info",
            failure::FailureInfo::TimeoutFailureInfo = "timeout_failure_info",
            failure::FailureInfo::CanceledFailureInfo = "canceled_failure_info",
            failure::FailureInfo::TerminatedFailureInfo = "terminated_failure_info",
            failure::FailureInfo::ServerFailureInfo = "server_failure_info",
            failure::FailureInfo::ResetWorkflowFailureInfo = "reset_workflow_failure_info",
            failure::FailureInfo::ActivityFailureInfo = "activity_failure_info",
            failure::FailureInfo::ChildWorkflowExecutionFailureInfo = "child_workflow_execution_failure_info",
            failure::FailureInfo::NexusOperationExecutionFailureInfo = "nexus_operation_execution_failure_info",
            failure::FailureInfo::NexusHandlerFailureInfo = "nexus_handler_failure_info",
        };
    }
    message request task_queue: TaskQueue = "temporal.api.taskqueue.v1.TaskQueue" {
        name: Identifier[max_bytes = 1000] + Decided => Rejects(3, "taskQueue|task queue") covers "go.temporal.io/server/common/tqid.validate";
        kind: Enum(task_queue_kind) + Decided => Accepts covers "go.temporal.io/server/common/tqid.normalizeAndValidate";
        normal_name: Text + Decided;
    }
    message request poll_activity_task_queue_request: PollActivityTaskQueueRequest = "temporal.api.workflowservice.v1.PollActivityTaskQueueRequest" {
        namespace: Text + Decided;
        task_queue: Bytes + Decided via Option<Vec<u8>> (|m: &PollActivityTaskQueueRequest| m.task_queue.as_ref().map(Message::encode_to_vec), |_: &mut PollActivityTaskQueueRequest, _: Option<Vec<u8>>| {});
        poller_group_id: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueRequest.poller_group_id; no semantic validation is claimed") opaque;
        identity: Text[max_bytes = 1000] + Decided => Rejects(3, "identity|Identity") covers "go.temporal.io/server/service/frontend.WorkflowHandler.PollActivityTaskQueue";
        worker_instance_key: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueRequest.worker_instance_key; no semantic validation is claimed") opaque;
        worker_control_task_queue: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueRequest.worker_control_task_queue; no semantic validation is claimed") opaque;
        task_queue_metadata: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueRequest.task_queue_metadata; no semantic validation is claimed") opaque;
        worker_version_capabilities: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueRequest.worker_version_capabilities; no semantic validation is claimed") opaque;
        deployment_options: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueRequest.deployment_options; no semantic validation is claimed") opaque;
        rules { required(task_queue) => Rejects(3, "taskQueue|task queue") covers "go.temporal.io/server/common/tqid.normalizeAndValidate"; }
    }
    message response poll_activity_task_queue_response: PollActivityTaskQueueResponse = "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse" {
        task_token: Secret + Decided;
        workflow_namespace: Text + Observed;
        workflow_type: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueResponse.workflow_type; no semantic validation is claimed") opaque;
        workflow_execution: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueResponse.workflow_execution; no semantic validation is claimed") opaque;
        activity_type: Bytes + Decided via Vec<u8> (|m: &PollActivityTaskQueueResponse| m.activity_type.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut PollActivityTaskQueueResponse, _: Vec<u8>| {});
        activity_id: Text + Observed;
        header: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueResponse.header; no semantic validation is claimed") opaque;
        input: Bytes + Decided via Vec<u8> (|m: &PollActivityTaskQueueResponse| m.input.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut PollActivityTaskQueueResponse, _: Vec<u8>| {});
        heartbeat_details: Bytes + Decided via Vec<u8> (|m: &PollActivityTaskQueueResponse| m.heartbeat_details.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut PollActivityTaskQueueResponse, _: Vec<u8>| {});
        scheduled_time: Timestamp + Observed via Option<umpire_annotations::Timestamp> (|m: &PollActivityTaskQueueResponse| m.scheduled_time.as_ref().map(|v| umpire_annotations::Timestamp { seconds: v.seconds, nanos: v.nanos }), |_: &mut PollActivityTaskQueueResponse, _: Option<umpire_annotations::Timestamp>| {});
        current_attempt_scheduled_time: Timestamp + Observed via Option<umpire_annotations::Timestamp> (|m: &PollActivityTaskQueueResponse| m.current_attempt_scheduled_time.as_ref().map(|v| umpire_annotations::Timestamp { seconds: v.seconds, nanos: v.nanos }), |_: &mut PollActivityTaskQueueResponse, _: Option<umpire_annotations::Timestamp>| {});
        started_time: Timestamp + Observed via Option<umpire_annotations::Timestamp> (|m: &PollActivityTaskQueueResponse| m.started_time.as_ref().map(|v| umpire_annotations::Timestamp { seconds: v.seconds, nanos: v.nanos }), |_: &mut PollActivityTaskQueueResponse, _: Option<umpire_annotations::Timestamp>| {});
        attempt: Counter[gte = 0] + Observed;
        schedule_to_close_timeout: Duration + Observed via Option<umpire_annotations::Duration> (|m: &PollActivityTaskQueueResponse| raw_duration_view(m.schedule_to_close_timeout.as_ref()), |_: &mut PollActivityTaskQueueResponse, _: Option<umpire_annotations::Duration>| {});
        start_to_close_timeout: Duration + Observed via Option<umpire_annotations::Duration> (|m: &PollActivityTaskQueueResponse| raw_duration_view(m.start_to_close_timeout.as_ref()), |_: &mut PollActivityTaskQueueResponse, _: Option<umpire_annotations::Duration>| {});
        heartbeat_timeout: Duration + Observed via Option<umpire_annotations::Duration> (|m: &PollActivityTaskQueueResponse| raw_duration_view(m.heartbeat_timeout.as_ref()), |_: &mut PollActivityTaskQueueResponse, _: Option<umpire_annotations::Duration>| {});
        retry_policy: Bytes + Decided via Vec<u8> (|m: &PollActivityTaskQueueResponse| m.retry_policy.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut PollActivityTaskQueueResponse, _: Vec<u8>| {});
        poller_scaling_decision: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueResponse.poller_scaling_decision; no semantic validation is claimed") opaque;
        priority: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueResponse.priority; no semantic validation is claimed") opaque;
        activity_run_id: Uuid + Observed via Option<String> (|m: &PollActivityTaskQueueResponse| uuid_view(&m.activity_run_id), |_: &mut PollActivityTaskQueueResponse, _: Option<String>| {});
        poller_group_infos: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueResponse.poller_group_infos; no semantic validation is claimed") opaque;
        poller_groups_info: Bytes + Ignored("This activity Model does not project PollActivityTaskQueueResponse.poller_groups_info; no semantic validation is claimed") opaque;
        rules { requires(activity_id, task_token); requires(input, task_token); }
    }
    message request respond_activity_task_completed_request: RespondActivityTaskCompletedRequest = "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest" {
        task_token: Secret + Decided;
        result: Bytes + Decided via Vec<u8> (|m: &RespondActivityTaskCompletedRequest| m.result.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut RespondActivityTaskCompletedRequest, _: Vec<u8>| {});
        identity: Text[max_bytes = 1000] + Decided => Rejects(3, "identity|Identity") covers "go.temporal.io/server/service/frontend.WorkflowHandler.RespondActivityTaskCompleted";
        namespace: Text + Decided;
        resource_id: Text + Decided;
        worker_version: Bytes + Ignored("This activity Model does not project RespondActivityTaskCompletedRequest.worker_version; no semantic validation is claimed") opaque;
        deployment: Bytes + Ignored("This activity Model does not project RespondActivityTaskCompletedRequest.deployment; no semantic validation is claimed") opaque;
        deployment_options: Bytes + Ignored("This activity Model does not project RespondActivityTaskCompletedRequest.deployment_options; no semantic validation is claimed") opaque;
    }
    message response respond_activity_task_completed_response: RespondActivityTaskCompletedResponse = "temporal.api.workflowservice.v1.RespondActivityTaskCompletedResponse" {

    }
    message request respond_activity_task_failed_request: RespondActivityTaskFailedRequest = "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest" {
        task_token: Secret + Decided;
        failure: Bytes + Decided via Vec<u8> (|m: &RespondActivityTaskFailedRequest| m.failure.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut RespondActivityTaskFailedRequest, _: Vec<u8>| {});
        identity: Text[max_bytes = 1000] + Decided => Rejects(3, "identity|Identity") covers "go.temporal.io/server/service/frontend.WorkflowHandler.RespondActivityTaskFailed";
        namespace: Text + Decided;
        resource_id: Text + Decided;
        last_heartbeat_details: Bytes + Decided via Vec<u8> (|m: &RespondActivityTaskFailedRequest| m.last_heartbeat_details.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut RespondActivityTaskFailedRequest, _: Vec<u8>| {});
        worker_version: Bytes + Ignored("This activity Model does not project RespondActivityTaskFailedRequest.worker_version; no semantic validation is claimed") opaque;
        deployment: Bytes + Ignored("This activity Model does not project RespondActivityTaskFailedRequest.deployment; no semantic validation is claimed") opaque;
        deployment_options: Bytes + Ignored("This activity Model does not project RespondActivityTaskFailedRequest.deployment_options; no semantic validation is claimed") opaque;
        cause: Enum(activity_task_failed_cause) + Decided => Accepts covers "go.temporal.io/server/service/frontend.WorkflowHandler.RespondActivityTaskFailed";
    }
    message response respond_activity_task_failed_response: RespondActivityTaskFailedResponse = "temporal.api.workflowservice.v1.RespondActivityTaskFailedResponse" {
        failures: Bytes + Decided via Vec<Vec<u8>> (|m: &RespondActivityTaskFailedResponse| m.failures.iter().map(Message::encode_to_vec).collect::<Vec<_>>(), |_: &mut RespondActivityTaskFailedResponse, _: Vec<Vec<u8>>| {});
    }
    message request respond_activity_task_canceled_request: RespondActivityTaskCanceledRequest = "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest" {
        task_token: Secret + Decided;
        details: Bytes + Decided via Vec<u8> (|m: &RespondActivityTaskCanceledRequest| m.details.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut RespondActivityTaskCanceledRequest, _: Vec<u8>| {});
        identity: Text[max_bytes = 1000] + Decided => Rejects(3, "identity|Identity") covers "go.temporal.io/server/service/frontend.WorkflowHandler.RespondActivityTaskCanceled";
        namespace: Text + Decided;
        resource_id: Text + Decided;
        worker_version: Bytes + Ignored("This activity Model does not project RespondActivityTaskCanceledRequest.worker_version; no semantic validation is claimed") opaque;
        deployment: Bytes + Ignored("This activity Model does not project RespondActivityTaskCanceledRequest.deployment; no semantic validation is claimed") opaque;
        deployment_options: Bytes + Ignored("This activity Model does not project RespondActivityTaskCanceledRequest.deployment_options; no semantic validation is claimed") opaque;
    }
    message response respond_activity_task_canceled_response: RespondActivityTaskCanceledResponse = "temporal.api.workflowservice.v1.RespondActivityTaskCanceledResponse" {

    }
    message request start_activity_execution_request: StartActivityExecutionRequest = "temporal.api.workflowservice.v1.StartActivityExecutionRequest" {
        namespace: Text + Decided;
        identity: Text[max_bytes = 1000] + Decided => Rejects(3, "identity|Identity") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeStartRequest";
        request_id: Text[max_bytes = 1000] + Decided + IdempotencyKey => Rejects(3, "request ID exceeds length limit") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeStartRequest";
        activity_id: Identifier[max_bytes = 1000] + Decided => Rejects(3, "activity[Ii][Dd]|activity ID") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeActivityAttributes";
        activity_type: Bytes + Decided via Option<Vec<u8>> (|m: &StartActivityExecutionRequest| m.activity_type.as_ref().map(Message::encode_to_vec), |_: &mut StartActivityExecutionRequest, _: Option<Vec<u8>>| {});
        task_queue: Bytes + Decided via Option<Vec<u8>> (|m: &StartActivityExecutionRequest| m.task_queue.as_ref().map(Message::encode_to_vec), |_: &mut StartActivityExecutionRequest, _: Option<Vec<u8>>| {});
        schedule_to_close_timeout: Duration[gte = 0] + Decided via Option<umpire_annotations::Duration> (|m: &StartActivityExecutionRequest| duration_view(m.schedule_to_close_timeout.as_ref()), |_: &mut StartActivityExecutionRequest, _: Option<umpire_annotations::Duration>| {}) => Rejects(3, "invalid ScheduleToCloseTimeout") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeTimeouts";
        schedule_to_start_timeout: Duration[gte = 0] + Decided via Option<umpire_annotations::Duration> (|m: &StartActivityExecutionRequest| duration_view(m.schedule_to_start_timeout.as_ref()), |_: &mut StartActivityExecutionRequest, _: Option<umpire_annotations::Duration>| {}) => Rejects(3, "invalid ScheduleToStartTimeout") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeTimeouts";
        start_to_close_timeout: Duration[gte = 0] + Decided via Option<umpire_annotations::Duration> (|m: &StartActivityExecutionRequest| duration_view(m.start_to_close_timeout.as_ref()), |_: &mut StartActivityExecutionRequest, _: Option<umpire_annotations::Duration>| {}) => Rejects(3, "invalid StartToCloseTimeout") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeTimeouts";
        heartbeat_timeout: Duration[gte = 0] + Decided via Option<umpire_annotations::Duration> (|m: &StartActivityExecutionRequest| duration_view(m.heartbeat_timeout.as_ref()), |_: &mut StartActivityExecutionRequest, _: Option<umpire_annotations::Duration>| {}) => Rejects(3, "invalid HeartbeatTimeout") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeTimeouts";
        retry_policy: Bytes + Decided via Vec<u8> (|m: &StartActivityExecutionRequest| m.retry_policy.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut StartActivityExecutionRequest, _: Vec<u8>| {});
        input: Bytes[max_len = 2_097_152] + Decided via Vec<u8> (|m: &StartActivityExecutionRequest| m.input.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut StartActivityExecutionRequest, _: Vec<u8>| {}) => Rejects(3, "input exceeds length limit") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeStartRequest";
        id_reuse_policy: Enum(activity_id_reuse_policy) + Decided => Accepts covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeIDPolicy";
        id_conflict_policy: Enum(activity_id_conflict_policy) + Decided => Accepts covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeIDPolicy";
        search_attributes: Bytes + Ignored("This activity Model does not project StartActivityExecutionRequest.search_attributes; no semantic validation is claimed") opaque;
        header: Bytes + Ignored("This activity Model does not project StartActivityExecutionRequest.header; no semantic validation is claimed") opaque;
        user_metadata: Bytes + Ignored("This activity Model does not project StartActivityExecutionRequest.user_metadata; no semantic validation is claimed") opaque;
        priority: Bytes + Ignored("This activity Model does not project StartActivityExecutionRequest.priority; no semantic validation is claimed") opaque;
        completion_callbacks: Bytes + Ignored("This activity Model does not project StartActivityExecutionRequest.completion_callbacks; no semantic validation is claimed") opaque;
        links: Bytes + Ignored("This activity Model does not project StartActivityExecutionRequest.links; no semantic validation is claimed") opaque;
        on_conflict_options: Bytes + Ignored("This activity Model does not project StartActivityExecutionRequest.on_conflict_options; no semantic validation is claimed") opaque;
        start_delay: Duration[gte = 0] + Decided via Option<umpire_annotations::Duration> (|m: &StartActivityExecutionRequest| duration_view(m.start_delay.as_ref()), |_: &mut StartActivityExecutionRequest, _: Option<umpire_annotations::Duration>| {}) => Rejects(3, "invalid StartDelay") covers "go.temporal.io/server/chasm/lib/activity.validateStartDelay";
        rules {
            required(activity_type, task_queue) => Rejects(3, "activityType|taskQueue|task queue") covers "go.temporal.io/server/chasm/lib/activity.ValidateAndNormalizeStandaloneActivity";
            any_of(schedule_to_close_timeout, start_to_close_timeout) => Rejects(3, "a valid StartToCloseTimeout or ScheduleToCloseTimeout") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeTimeouts";
        }
    }
    message response start_activity_execution_response: StartActivityExecutionResponse = "temporal.api.workflowservice.v1.StartActivityExecutionResponse" {
        run_id: Uuid + Observed via Option<String> (|m: &StartActivityExecutionResponse| uuid_view(&m.run_id), |_: &mut StartActivityExecutionResponse, _: Option<String>| {});
        started: Counter + Observed via i32 (|m: &StartActivityExecutionResponse| i32::from(m.started), |_: &mut StartActivityExecutionResponse, _: i32| {});
        link: Bytes + Ignored("This activity Model does not project StartActivityExecutionResponse.link; no semantic validation is claimed") opaque;
        rules { required(run_id); }
    }
    message request describe_activity_execution_request: DescribeActivityExecutionRequest = "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest" {
        namespace: Text + Decided;
        activity_id: Identifier[max_bytes = 1000] + Decided => Rejects(3, "activity[Ii][Dd]|activity ID") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeDescribeActivityExecutionRequest";
        run_id: Uuid + Decided via Option<String> (|m: &DescribeActivityExecutionRequest| uuid_view(&m.run_id), |_: &mut DescribeActivityExecutionRequest, _: Option<String>| {}) => Rejects(3, "invalid run id") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeDescribeActivityExecutionRequest";
        include_input: Bytes + Ignored("This activity Model does not project DescribeActivityExecutionRequest.include_input; no semantic validation is claimed") opaque;
        include_outcome: Bytes + Ignored("This activity Model does not project DescribeActivityExecutionRequest.include_outcome; no semantic validation is claimed") opaque;
        long_poll_token: Secret + Decided;
        include_heartbeat_details: Bytes + Ignored("This activity Model does not project DescribeActivityExecutionRequest.include_heartbeat_details; no semantic validation is claimed") opaque;
        include_last_failure: Bytes + Ignored("This activity Model does not project DescribeActivityExecutionRequest.include_last_failure; no semantic validation is claimed") opaque;
        rules { requires(long_poll_token, run_id) => Rejects(3, "run id is required") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeDescribeActivityExecutionRequest"; }
    }
    message response describe_activity_execution_response: DescribeActivityExecutionResponse = "temporal.api.workflowservice.v1.DescribeActivityExecutionResponse" {
        run_id: Uuid + Observed via Option<String> (|m: &DescribeActivityExecutionResponse| uuid_view(&m.run_id), |_: &mut DescribeActivityExecutionResponse, _: Option<String>| {});
        info: Bytes + Decided via Option<Vec<u8>> (|m: &DescribeActivityExecutionResponse| m.info.as_ref().map(Message::encode_to_vec), |_: &mut DescribeActivityExecutionResponse, _: Option<Vec<u8>>| {});
        input: Bytes + Decided via Vec<u8> (|m: &DescribeActivityExecutionResponse| m.input.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut DescribeActivityExecutionResponse, _: Vec<u8>| {});
        outcome: Bytes + Decided via Vec<u8> (|m: &DescribeActivityExecutionResponse| m.outcome.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut DescribeActivityExecutionResponse, _: Vec<u8>| {});
        long_poll_token: Secret + Decided;
        callbacks: Bytes + Ignored("This activity Model does not project DescribeActivityExecutionResponse.callbacks; no semantic validation is claimed") opaque;
        rules { required(info); }
    }
    message request poll_activity_execution_request: PollActivityExecutionRequest = "temporal.api.workflowservice.v1.PollActivityExecutionRequest" {
        namespace: Text + Decided;
        activity_id: Identifier[max_bytes = 1000] + Decided => Rejects(3, "activity[Ii][Dd]|activity ID") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizePollActivityExecutionRequest";
        run_id: Uuid + Decided via Option<String> (|m: &PollActivityExecutionRequest| uuid_view(&m.run_id), |_: &mut PollActivityExecutionRequest, _: Option<String>| {}) => Rejects(3, "invalid run id") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizePollActivityExecutionRequest";
    }
    message response poll_activity_execution_response: PollActivityExecutionResponse = "temporal.api.workflowservice.v1.PollActivityExecutionResponse" {
        run_id: Uuid + Observed via Option<String> (|m: &PollActivityExecutionResponse| uuid_view(&m.run_id), |_: &mut PollActivityExecutionResponse, _: Option<String>| {});
        outcome: Bytes + Decided via Vec<u8> (|m: &PollActivityExecutionResponse| m.outcome.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut PollActivityExecutionResponse, _: Vec<u8>| {});
    }
    message request request_cancel_activity_execution_request: RequestCancelActivityExecutionRequest = "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest" {
        namespace: Text + Decided;
        activity_id: Identifier[max_bytes = 1000] + Decided => Rejects(3, "activity[Ii][Dd]|activity ID") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeRequestCancelActivityExecutionRequest";
        run_id: Uuid + Decided via Option<String> (|m: &RequestCancelActivityExecutionRequest| uuid_view(&m.run_id), |_: &mut RequestCancelActivityExecutionRequest, _: Option<String>| {}) => Rejects(3, "invalid run id") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeRequestCancelActivityExecutionRequest";
        identity: Text[max_bytes = 1000] + Decided => Rejects(3, "identity|Identity") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeRequestCancelActivityExecutionRequest";
        request_id: Text[max_bytes = 1000] + Decided + IdempotencyKey => Rejects(3, "request ID exceeds length limit") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeRequestID";
        reason: Text[max_bytes = 2_097_152] + Decided => Rejects(3, "reason exceeds length limit") covers "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeRequestCancelActivityExecutionRequest";
    }
    message response request_cancel_activity_execution_response: RequestCancelActivityExecutionResponse = "temporal.api.workflowservice.v1.RequestCancelActivityExecutionResponse" {

    }
    message response timeout_failure_info: TimeoutFailureInfo = "temporal.api.failure.v1.TimeoutFailureInfo" {
        timeout_type: Enum(timeout_type) + Observed;
        last_heartbeat_details: Bytes + Decided via Vec<u8> (|m: &TimeoutFailureInfo| m.last_heartbeat_details.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut TimeoutFailureInfo, _: Vec<u8>| {});
    }
    message response canceled_failure_info: CanceledFailureInfo = "temporal.api.failure.v1.CanceledFailureInfo" {
        details: Bytes + Decided via Vec<u8> (|m: &CanceledFailureInfo| m.details.as_ref().map(Message::encode_to_vec).unwrap_or_default(), |_: &mut CanceledFailureInfo, _: Vec<u8>| {});
        identity: Bytes + Ignored("This activity Model does not project CanceledFailureInfo.identity; no semantic validation is claimed") opaque;
    }
    message response terminated_failure_info: TerminatedFailureInfo = "temporal.api.failure.v1.TerminatedFailureInfo" {
        identity: Bytes + Ignored("This activity Model does not project TerminatedFailureInfo.identity; no semantic validation is claimed") opaque;
    }
    message response server_failure_info: ServerFailureInfo = "temporal.api.failure.v1.ServerFailureInfo" {
        non_retryable: Counter + Decided via i32 (|m: &ServerFailureInfo| i32::from(m.non_retryable), |_: &mut ServerFailureInfo, _: i32| {});
    }
    message response reset_workflow_failure_info: ResetWorkflowFailureInfo = "temporal.api.failure.v1.ResetWorkflowFailureInfo" {
        last_heartbeat_details: Bytes + Ignored("This activity Model does not project ResetWorkflowFailureInfo.last_heartbeat_details; no semantic validation is claimed") opaque;
    }
    message response activity_failure_info: ActivityFailureInfo = "temporal.api.failure.v1.ActivityFailureInfo" {
        scheduled_event_id: Bytes + Ignored("This activity Model does not project ActivityFailureInfo.scheduled_event_id; no semantic validation is claimed") opaque;
        started_event_id: Bytes + Ignored("This activity Model does not project ActivityFailureInfo.started_event_id; no semantic validation is claimed") opaque;
        identity: Bytes + Ignored("This activity Model does not project ActivityFailureInfo.identity; no semantic validation is claimed") opaque;
        activity_type: Bytes + Ignored("This activity Model does not project ActivityFailureInfo.activity_type; no semantic validation is claimed") opaque;
        activity_id: Bytes + Ignored("This activity Model does not project ActivityFailureInfo.activity_id; no semantic validation is claimed") opaque;
        retry_state: Bytes + Ignored("This activity Model does not project ActivityFailureInfo.retry_state; no semantic validation is claimed") opaque;
    }
    message response child_workflow_execution_failure_info: ChildWorkflowExecutionFailureInfo = "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo" {
        namespace: Bytes + Ignored("This activity Model does not project ChildWorkflowExecutionFailureInfo.namespace; no semantic validation is claimed") opaque;
        workflow_execution: Bytes + Ignored("This activity Model does not project ChildWorkflowExecutionFailureInfo.workflow_execution; no semantic validation is claimed") opaque;
        workflow_type: Bytes + Ignored("This activity Model does not project ChildWorkflowExecutionFailureInfo.workflow_type; no semantic validation is claimed") opaque;
        initiated_event_id: Bytes + Ignored("This activity Model does not project ChildWorkflowExecutionFailureInfo.initiated_event_id; no semantic validation is claimed") opaque;
        started_event_id: Bytes + Ignored("This activity Model does not project ChildWorkflowExecutionFailureInfo.started_event_id; no semantic validation is claimed") opaque;
        retry_state: Bytes + Ignored("This activity Model does not project ChildWorkflowExecutionFailureInfo.retry_state; no semantic validation is claimed") opaque;
    }
    message response nexus_operation_failure_info: NexusOperationFailureInfo = "temporal.api.failure.v1.NexusOperationFailureInfo" {
        scheduled_event_id: Bytes + Ignored("This activity Model does not project NexusOperationFailureInfo.scheduled_event_id; no semantic validation is claimed") opaque;
        endpoint: Bytes + Ignored("This activity Model does not project NexusOperationFailureInfo.endpoint; no semantic validation is claimed") opaque;
        service: Bytes + Ignored("This activity Model does not project NexusOperationFailureInfo.service; no semantic validation is claimed") opaque;
        operation: Bytes + Ignored("This activity Model does not project NexusOperationFailureInfo.operation; no semantic validation is claimed") opaque;
        operation_id: Bytes + Ignored("This activity Model does not project NexusOperationFailureInfo.operation_id; no semantic validation is claimed") opaque;
        operation_token: Secret + Ignored("Nexus operation credentials are outside the mini activity lifecycle") opaque;
    }
    message response nexus_handler_failure_info: NexusHandlerFailureInfo = "temporal.api.failure.v1.NexusHandlerFailureInfo" {
        r#type: Bytes + Ignored("This activity Model does not project NexusHandlerFailureInfo.r#type; no semantic validation is claimed") opaque;
        retry_behavior: Bytes + Ignored("This activity Model does not project NexusHandlerFailureInfo.retry_behavior; no semantic validation is claimed") opaque;
    }
    message response proto_duration: ProtoDuration = "google.protobuf.Duration" {
        seconds: Counter + Observed;
        nanos: Counter + Observed;
    }
    message response proto_timestamp: ProtoTimestamp = "google.protobuf.Timestamp" {
        seconds: Counter + Observed;
        nanos: Counter + Observed;
    }
    message response metadata_entry: MetadataEntry = "temporal.api.common.v1.Payload.MetadataEntry" {
        key: Text + Decided;
        value: Secret + Decided;
    }
    service "temporal.api.workflowservice.v1.WorkflowService" {
        RegisterNamespace: OutOfScope("Namespace administration is outside this standalone activity Model");
        DescribeNamespace: OutOfScope("SDK namespace discovery is infrastructure outside activity lifecycle validation");
        ListNamespaces: OutOfScope("Namespace administration is outside this standalone activity Model");
        UpdateNamespace: OutOfScope("Namespace administration is outside this standalone activity Model");
        DeprecateNamespace: OutOfScope("Namespace administration is outside this standalone activity Model");
        StartWorkflowExecution: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        ExecuteMultiOperation: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        GetWorkflowExecutionHistory: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        GetWorkflowExecutionHistoryReverse: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        PollWorkflowTaskQueue: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        RespondWorkflowTaskCompleted: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        RespondWorkflowTaskFailed: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        PollActivityTaskQueue: Action("Worker::Poll");
        RecordActivityTaskHeartbeat: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        RecordActivityTaskHeartbeatById: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        RespondActivityTaskCompleted: Action("Worker::RespondCompleted");
        RespondActivityTaskCompletedById: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        RespondActivityTaskFailed: Action("Worker::RespondFailed");
        RespondActivityTaskFailedById: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        RespondActivityTaskCanceled: Observed("worker acknowledges client cancellation");
        RespondActivityTaskCanceledById: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        RequestCancelWorkflowExecution: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        SignalWorkflowExecution: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        SignalWithStartWorkflowExecution: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        ResetWorkflowExecution: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        TerminateWorkflowExecution: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        DeleteWorkflowExecution: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        ListOpenWorkflowExecutions: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        ListClosedWorkflowExecutions: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        ListWorkflowExecutions: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        ListArchivedWorkflowExecutions: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        ScanWorkflowExecutions: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        CountWorkflowExecutions: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        GetSearchAttributes: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        RespondQueryTaskCompleted: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        ResetStickyTaskQueue: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        ShutdownWorker: OutOfScope("SDK worker shutdown bookkeeping does not change this activity Model");
        QueryWorkflow: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        DescribeWorkflowExecution: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        DescribeTaskQueue: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        GetClusterInfo: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        GetSystemInfo: OutOfScope("SDK server capability negotiation does not observe an activity lifecycle");
        ListTaskQueuePartitions: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        CreateSchedule: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        DescribeSchedule: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        UpdateSchedule: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        PatchSchedule: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        ListScheduleMatchingTimes: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        DeleteSchedule: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        ListSchedules: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        CountSchedules: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        UpdateWorkerBuildIdCompatibility: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        GetWorkerBuildIdCompatibility: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        UpdateWorkerVersioningRules: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        GetWorkerVersioningRules: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        GetWorkerTaskReachability: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        DescribeDeployment: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        DescribeWorkerDeploymentVersion: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        ListDeployments: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        GetDeploymentReachability: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        GetCurrentDeployment: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        SetCurrentDeployment: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        SetWorkerDeploymentCurrentVersion: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        DescribeWorkerDeployment: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        DeleteWorkerDeployment: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        DeleteWorkerDeploymentVersion: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        SetWorkerDeploymentRampingVersion: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        ListWorkerDeployments: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        CreateWorkerDeployment: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        CreateWorkerDeploymentVersion: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        UpdateWorkerDeploymentVersionComputeConfig: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        ValidateWorkerDeploymentVersionComputeConfig: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        UpdateWorkerDeploymentVersionMetadata: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        SetWorkerDeploymentManager: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        UpdateWorkflowExecution: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        PollWorkflowExecutionUpdate: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        StartBatchOperation: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        StopBatchOperation: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        DescribeBatchOperation: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        ListBatchOperations: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        PollNexusTaskQueue: OutOfScope("SDK worker control commands use Nexus polling; their contents are outside this activity Model");
        RespondNexusTaskCompleted: OutOfScope("SDK Nexus worker control completion is outside this activity Model");
        RespondNexusTaskFailed: OutOfScope("SDK Nexus worker control failure is outside this activity Model");
        UpdateActivityOptions: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        UpdateWorkflowExecutionOptions: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        PauseActivity: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        UnpauseActivity: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        ResetActivity: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        CreateWorkflowRule: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        DescribeWorkflowRule: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        DeleteWorkflowRule: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        ListWorkflowRules: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        TriggerWorkflowRule: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        RecordWorkerHeartbeat: OutOfScope("SDK worker health and configuration heartbeat is infrastructure outside activity lifecycle validation");
        ListWorkers: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        CountWorkers: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        UpdateTaskQueueConfig: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        FetchWorkerConfig: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        UpdateWorkerConfig: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        DescribeWorker: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        PauseWorkflowExecution: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        UnpauseWorkflowExecution: OutOfScope("Workflow execution operations are outside this standalone activity Model");
        StartActivityExecution: Observed("successful start establishes the Model initial state");
        StartNexusOperationExecution: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        DescribeActivityExecution: Observed("activity lifecycle phase");
        DescribeNexusOperationExecution: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        PollActivityExecution: Observed("SDK activity handle outcome");
        PollNexusOperationExecution: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        ListActivityExecutions: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        ListNexusOperationExecutions: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        CountActivityExecutions: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        CountNexusOperationExecutions: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        RequestCancelActivityExecution: Action("Client::Cancel");
        RequestCancelNexusOperationExecution: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        TerminateActivityExecution: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        DeleteActivityExecution: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        PauseActivityExecution: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        ResetActivityExecution: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        UnpauseActivityExecution: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        UpdateActivityExecutionOptions: OutOfScope("This activity operation is not represented by the mini activity lifecycle");
        TerminateNexusOperationExecution: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        DeleteNexusOperationExecution: OutOfScope("Service administration and other resource lifecycles are outside this activity Model");
        PollWorkflowExecutionTimeSkipping: OutOfScope("Workflow execution operations are outside this standalone activity Model");
    }
    actions bind: ModelAction {
        ModelAction::Client => { Client::Cancel => Rpc("temporal.api.workflowservice.v1.WorkflowService.RequestCancelActivityExecution"), },
        ModelAction::Worker => {
            Worker::Poll => Rpc("temporal.api.workflowservice.v1.WorkflowService.PollActivityTaskQueue"),
            Worker::RespondCompleted => Rpc("temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskCompleted"),
            Worker::RespondFailed => Rpc("temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskFailed"),
        },
        ModelAction::Deadline => {
            Deadline::ScheduleToStart => Environmental("abstract schedule-to-start deadline"),
            Deadline::StartToClose => Environmental("abstract start-to-close deadline"),
        },
    }
}

/// UUID intent for start request IDs; the server accepts non-UUID IDs.
///
/// # Errors
/// Returns catalog admission errors.
pub fn start_request_id_intent()
-> Result<Catalog<StartActivityExecutionRequest>, umpire_annotations::AdmissionErrors> {
    supplemental_request!(
        uuid StartActivityExecutionRequest,
        "temporal.api.workflowservice.v1.StartActivityExecutionRequest",
        "temporal.api.workflowservice.v1.StartActivityExecutionRequest.request_id",
        get = |message| &message.request_id,
        set = |message| &mut message.request_id,
        expected = Expectation::Accepts,
        covers = "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeStartRequest",
    )
}

/// UUID intent for cancel request IDs; the server accepts non-UUID IDs.
///
/// # Errors
/// Returns catalog admission errors.
pub fn cancel_request_id_intent()
-> Result<Catalog<RequestCancelActivityExecutionRequest>, umpire_annotations::AdmissionErrors> {
    supplemental_request!(
        uuid RequestCancelActivityExecutionRequest,
        "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest",
        "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest.request_id",
        get = |message| &message.request_id,
        set = |message| &mut message.request_id,
        expected = Expectation::Accepts,
        covers = "go.temporal.io/server/chasm/lib/activity.validateAndNormalizeRequestID",
    )
}

/// Worker-supplied failures require application info; server failures do not.
///
/// # Errors
/// Returns catalog admission errors.
pub fn worker_failure_presence() -> Result<Catalog<Failure>, umpire_annotations::AdmissionErrors> {
    supplemental_request!(
        required Failure,
        "temporal.api.failure.v1.Failure",
        "temporal.api.failure.v1.Failure.application_failure_info",
        "application_failure_info",
        get = |message: &Failure| match &message.failure_info {
            Some(failure::FailureInfo::ApplicationFailureInfo(value)) => Some(value.encode_to_vec()),
            _ => None,
        },
        set = |_: &mut Failure, _: Option<Vec<u8>>| {},
        expected = Expectation::Rejects {
            code: 3,
            message_pattern: "Failure must have ApplicationFailureInfo".into(),
            details_type: None,
        },
        covers = "go.temporal.io/server/service/frontend.WorkflowHandler.RespondActivityTaskFailed",
    )
}
