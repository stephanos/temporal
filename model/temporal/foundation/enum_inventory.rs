//! Reviewed enum admission lists; descriptor regeneration must never replace these lists.

use umpire_temporal_api::temporal::api::enums::v1::{
    ActivityExecutionStatus, ActivityIdConflictPolicy, ActivityIdReusePolicy,
    ActivityTaskFailedCause, ApplicationErrorCategory, NexusHandlerErrorRetryBehavior,
    PendingActivityState, RetryState, TaskQueueKind, TimeoutType,
};

umpire_catalog::annotate! {
    pub catalog enum_inventory;
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
    enum activity_execution_status: ActivityExecutionStatus =
            "temporal.api.enums.v1.ActivityExecutionStatus" {
        ActivityExecutionStatus::Unspecified = "ACTIVITY_EXECUTION_STATUS_UNSPECIFIED",
        ActivityExecutionStatus::Running = "ACTIVITY_EXECUTION_STATUS_RUNNING",
        ActivityExecutionStatus::Completed = "ACTIVITY_EXECUTION_STATUS_COMPLETED",
        ActivityExecutionStatus::Failed = "ACTIVITY_EXECUTION_STATUS_FAILED",
        ActivityExecutionStatus::Canceled = "ACTIVITY_EXECUTION_STATUS_CANCELED",
        ActivityExecutionStatus::Terminated = "ACTIVITY_EXECUTION_STATUS_TERMINATED",
        ActivityExecutionStatus::TimedOut = "ACTIVITY_EXECUTION_STATUS_TIMED_OUT",
        ActivityExecutionStatus::Paused = "ACTIVITY_EXECUTION_STATUS_PAUSED",
    }
    enum pending_activity_state: PendingActivityState =
            "temporal.api.enums.v1.PendingActivityState" {
        PendingActivityState::Unspecified = "PENDING_ACTIVITY_STATE_UNSPECIFIED",
        PendingActivityState::Scheduled = "PENDING_ACTIVITY_STATE_SCHEDULED",
        PendingActivityState::Started = "PENDING_ACTIVITY_STATE_STARTED",
        PendingActivityState::CancelRequested = "PENDING_ACTIVITY_STATE_CANCEL_REQUESTED",
        PendingActivityState::Paused = "PENDING_ACTIVITY_STATE_PAUSED",
        PendingActivityState::PauseRequested = "PENDING_ACTIVITY_STATE_PAUSE_REQUESTED",
    }
    enum activity_id_reuse_policy: ActivityIdReusePolicy =
            "temporal.api.enums.v1.ActivityIdReusePolicy" {
        ActivityIdReusePolicy::Unspecified = "ACTIVITY_ID_REUSE_POLICY_UNSPECIFIED",
        ActivityIdReusePolicy::AllowDuplicate = "ACTIVITY_ID_REUSE_POLICY_ALLOW_DUPLICATE",
        ActivityIdReusePolicy::AllowDuplicateFailedOnly =
            "ACTIVITY_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY",
        ActivityIdReusePolicy::RejectDuplicate = "ACTIVITY_ID_REUSE_POLICY_REJECT_DUPLICATE",
    }
    enum activity_id_conflict_policy: ActivityIdConflictPolicy =
            "temporal.api.enums.v1.ActivityIdConflictPolicy" {
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
    enum application_error_category: ApplicationErrorCategory =
            "temporal.api.enums.v1.ApplicationErrorCategory" {
        ApplicationErrorCategory::Unspecified = "APPLICATION_ERROR_CATEGORY_UNSPECIFIED",
        ApplicationErrorCategory::Benign = "APPLICATION_ERROR_CATEGORY_BENIGN",
    }
    enum activity_task_failed_cause: ActivityTaskFailedCause =
            "temporal.api.enums.v1.ActivityTaskFailedCause" {
        ActivityTaskFailedCause::Unspecified = "ACTIVITY_TASK_FAILED_CAUSE_UNSPECIFIED",
        ActivityTaskFailedCause::PayloadsTooLarge = "ACTIVITY_TASK_FAILED_CAUSE_PAYLOADS_TOO_LARGE",
        ActivityTaskFailedCause::ExternalStorageFailure =
            "ACTIVITY_TASK_FAILED_CAUSE_EXTERNAL_STORAGE_FAILURE",
        ActivityTaskFailedCause::ActivityWorkerUnhandledFailure =
            "ACTIVITY_TASK_FAILED_CAUSE_ACTIVITY_WORKER_UNHANDLED_FAILURE",
    }
    enum timeout_type: TimeoutType = "temporal.api.enums.v1.TimeoutType" {
        TimeoutType::Unspecified = "TIMEOUT_TYPE_UNSPECIFIED",
        TimeoutType::StartToClose = "TIMEOUT_TYPE_START_TO_CLOSE",
        TimeoutType::ScheduleToStart = "TIMEOUT_TYPE_SCHEDULE_TO_START",
        TimeoutType::ScheduleToClose = "TIMEOUT_TYPE_SCHEDULE_TO_CLOSE",
        TimeoutType::Heartbeat = "TIMEOUT_TYPE_HEARTBEAT",
    }
    enum nexus_handler_error_retry_behavior: NexusHandlerErrorRetryBehavior =
            "temporal.api.enums.v1.NexusHandlerErrorRetryBehavior" {
        NexusHandlerErrorRetryBehavior::Unspecified =
            "NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_UNSPECIFIED",
        NexusHandlerErrorRetryBehavior::Retryable = "NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE",
        NexusHandlerErrorRetryBehavior::NonRetryable =
            "NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE",
    }
}
