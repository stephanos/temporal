//! Generated Go validator identities; checked by `TestValidatorCatalogFreshness`.

#[rustfmt::skip]
#[allow(missing_docs, non_snake_case, non_upper_case_globals)]
pub mod activity {
    pub const Activity_validateActivityTaskToken: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "Activity.validateActivityTaskToken",
    );
    pub const ValidateAndNormalizeEmbeddedActivity: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "ValidateAndNormalizeEmbeddedActivity",
    );
    pub const ValidateAndNormalizeStandaloneActivity: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "ValidateAndNormalizeStandaloneActivity",
    );
    pub const activityDispatchTaskHandler_Validate: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "activityDispatchTaskHandler.Validate",
    );
    pub const frontendHandler_validateAndPopulateStartRequest: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "frontendHandler.validateAndPopulateStartRequest",
    );
    pub const handler_StartActivityExecution: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "handler.StartActivityExecution",
    );
    pub const heartbeatTimeoutTaskHandler_Validate: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "heartbeatTimeoutTaskHandler.Validate",
    );
    pub const linkValidatorProvider: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "linkValidatorProvider",
    );
    pub const newLinkValidator: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "newLinkValidator",
    );
    pub const scheduleToCloseTimeoutTaskHandler_Validate: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "scheduleToCloseTimeoutTaskHandler.Validate",
    );
    pub const scheduleToStartTimeoutTaskHandler_Validate: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "scheduleToStartTimeoutTaskHandler.Validate",
    );
    pub const startToCloseTimeoutTaskHandler_Validate: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "startToCloseTimeoutTaskHandler.Validate",
    );
    pub const validateActivityRetryPolicy: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateActivityRetryPolicy",
    );
    pub const validateAndNormalizeActivityAttributes: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeActivityAttributes",
    );
    pub const validateAndNormalizeDeleteActivityExecutionRequest: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeDeleteActivityExecutionRequest",
    );
    pub const validateAndNormalizeDescribeActivityExecutionRequest: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeDescribeActivityExecutionRequest",
    );
    pub const validateAndNormalizeIDPolicy: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeIDPolicy",
    );
    pub const validateAndNormalizePauseActivityExecutionRequest: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizePauseActivityExecutionRequest",
    );
    pub const validateAndNormalizePollActivityExecutionRequest: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizePollActivityExecutionRequest",
    );
    pub const validateAndNormalizeRequestCancelActivityExecutionRequest: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeRequestCancelActivityExecutionRequest",
    );
    pub const validateAndNormalizeRequestID: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeRequestID",
    );
    pub const validateAndNormalizeResetActivityExecutionRequest: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeResetActivityExecutionRequest",
    );
    pub const validateAndNormalizeSearchAttributes: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeSearchAttributes",
    );
    pub const validateAndNormalizeStartRequest: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeStartRequest",
    );
    pub const validateAndNormalizeTerminateActivityExecutionRequest: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeTerminateActivityExecutionRequest",
    );
    pub const validateAndNormalizeTimeouts: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeTimeouts",
    );
    pub const validateAndNormalizeUnpauseActivityExecutionRequest: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeUnpauseActivityExecutionRequest",
    );
    pub const validateAndNormalizeUpdateActivityExecutionOptionsRequest: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateAndNormalizeUpdateActivityExecutionOptionsRequest",
    );
    pub const validateBlobSize: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateBlobSize",
    );
    pub const validateJitter: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateJitter",
    );
    pub const validateOnConflictOptions: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateOnConflictOptions",
    );
    pub const validateOriginalOptionsRestorable: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateOriginalOptionsRestorable",
    );
    pub const validateStartDelay: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateStartDelay",
    );
    pub const validateUserMetadata: &str = concat!(
        "go.temporal.io/server/chasm/lib/activity.",
        "validateUserMetadata",
    );
}
#[rustfmt::skip]
#[allow(missing_docs, non_snake_case, non_upper_case_globals)]
pub mod retrypolicy {
    pub const Validate: &str = concat!(
        "go.temporal.io/server/common/retrypolicy.",
        "Validate",
    );
}
#[rustfmt::skip]
#[allow(missing_docs, non_snake_case, non_upper_case_globals)]
pub mod tqid {
    pub const NormalizeAndValidate: &str = concat!(
        "go.temporal.io/server/common/tqid.",
        "NormalizeAndValidate",
    );
    pub const NormalizeAndValidatePartition: &str = concat!(
        "go.temporal.io/server/common/tqid.",
        "NormalizeAndValidatePartition",
    );
    pub const NormalizeAndValidateUserDefined: &str = concat!(
        "go.temporal.io/server/common/tqid.",
        "NormalizeAndValidateUserDefined",
    );
    pub const Validate: &str = concat!(
        "go.temporal.io/server/common/tqid.",
        "Validate",
    );
    pub const normalizeAndValidate: &str = concat!(
        "go.temporal.io/server/common/tqid.",
        "normalizeAndValidate",
    );
    pub const validate: &str = concat!(
        "go.temporal.io/server/common/tqid.",
        "validate",
    );
}
#[rustfmt::skip]
#[allow(missing_docs, non_snake_case, non_upper_case_globals)]
pub mod frontend {
    pub const AdminHandler_unaliasAndValidateSearchAttributes: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "AdminHandler.unaliasAndValidateSearchAttributes",
    );
    pub const AdminHandler_validateAndResolveArchetypeID: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "AdminHandler.validateAndResolveArchetypeID",
    );
    pub const AdminHandler_validateGetWorkflowExecutionRawHistoryV2Request: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "AdminHandler.validateGetWorkflowExecutionRawHistoryV2Request",
    );
    pub const AdminHandler_validateRemoteClusterMetadata: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "AdminHandler.validateRemoteClusterMetadata",
    );
    pub const Config_IsScheduleValidationDisabled: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "Config.IsScheduleValidationDisabled",
    );
    pub const NamespaceValidatorInterceptorProvider: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "NamespaceValidatorInterceptorProvider",
    );
    pub const NexusEndpointClient_validatePageSize: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "NexusEndpointClient.validatePageSize",
    );
    pub const NexusEndpointClient_validateUpsertSpec: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "NexusEndpointClient.validateUpsertSpec",
    );
    pub const OperatorHandlerImpl_validateRemoteClusterMetadata: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "OperatorHandlerImpl.validateRemoteClusterMetadata",
    );
    pub const WorkflowHandler_PollActivityTaskQueue: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.PollActivityTaskQueue",
    );
    pub const WorkflowHandler_RespondActivityTaskCanceled: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.RespondActivityTaskCanceled",
    );
    pub const WorkflowHandler_RespondActivityTaskCompleted: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.RespondActivityTaskCompleted",
    );
    pub const WorkflowHandler_RespondActivityTaskFailed: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.RespondActivityTaskFailed",
    );
    pub const WorkflowHandler_ValidateWorkerDeploymentVersionComputeConfig: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.ValidateWorkerDeploymentVersionComputeConfig",
    );
    pub const WorkflowHandler_checkBadBinary: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.checkBadBinary",
    );
    pub const WorkflowHandler_checkWorkerDeploymentReadRateLimit: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.checkWorkerDeploymentReadRateLimit",
    );
    pub const WorkflowHandler_getWorkerTaskReachabilityValidated: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.getWorkerTaskReachabilityValidated",
    );
    pub const WorkflowHandler_handleScheduleValidationError: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.handleScheduleValidationError",
    );
    pub const WorkflowHandler_validateAndPopulateTimeSkippingConfig: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateAndPopulateTimeSkippingConfig",
    );
    pub const WorkflowHandler_validateBuildIdCompatibilityUpdate: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateBuildIdCompatibilityUpdate",
    );
    pub const WorkflowHandler_validateDeploymentOptions: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateDeploymentOptions",
    );
    pub const WorkflowHandler_validateNamespace: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateNamespace",
    );
    pub const WorkflowHandler_validateOnConflictOptions: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateOnConflictOptions",
    );
    pub const WorkflowHandler_validateScheduleMatchingTimesTimestamps: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateScheduleMatchingTimesTimestamps",
    );
    pub const WorkflowHandler_validateScheduleOverlapPolicies: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateScheduleOverlapPolicies",
    );
    pub const WorkflowHandler_validateSchedulePatchTimestamps: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateSchedulePatchTimestamps",
    );
    pub const WorkflowHandler_validateSchedulePayloadSize: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateSchedulePayloadSize",
    );
    pub const WorkflowHandler_validateStartWorkflowArgsForSchedule: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateStartWorkflowArgsForSchedule",
    );
    pub const WorkflowHandler_validateVersionRuleBuildId: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateVersionRuleBuildId",
    );
    pub const WorkflowHandler_validateVersioningInfo: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateVersioningInfo",
    );
    pub const WorkflowHandler_validateWorkflowDeletionCluster: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateWorkflowDeletionCluster",
    );
    pub const WorkflowHandler_validateWorkflowID: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "WorkflowHandler.validateWorkflowID",
    );
    pub const callbackValidatorProvider: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "callbackValidatorProvider",
    );
    pub const healthCheckerImpl_Check: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "healthCheckerImpl.Check",
    );
    pub const healthCheckerImpl_checkHost: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "healthCheckerImpl.checkHost",
    );
    pub const namespaceHandler_validateHistoryArchivalURI: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "namespaceHandler.validateHistoryArchivalURI",
    );
    pub const namespaceHandler_validateRetentionDuration: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "namespaceHandler.validateRetentionDuration",
    );
    pub const namespaceHandler_validateVisibilityArchivalURI: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "namespaceHandler.validateVisibilityArchivalURI",
    );
    pub const validateActivityFailureNextRetryDelays: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateActivityFailureNextRetryDelays",
    );
    pub const validateAdminBatchOperation: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateAdminBatchOperation",
    );
    pub const validateClusterNotInUseByNamespaces: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateClusterNotInUseByNamespaces",
    );
    pub const validateDeleteRequest: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateDeleteRequest",
    );
    pub const validateExecution: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateExecution",
    );
    pub const validateFairnessWeightUpdate: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateFairnessWeightUpdate",
    );
    pub const validateGetRequest: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateGetRequest",
    );
    pub const validateHistoryDLQKey: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateHistoryDLQKey",
    );
    pub const validateRateLimit: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateRateLimit",
    );
    pub const validateReplicationStateUpdate: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateReplicationStateUpdate",
    );
    pub const validateRequestId: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateRequestId",
    );
    pub const validateScheduleIntervalDurations: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateScheduleIntervalDurations",
    );
    pub const validateScheduleOverlapPolicy: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateScheduleOverlapPolicy",
    );
    pub const validateScheduleRemainingActions: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateScheduleRemainingActions",
    );
    pub const validateScheduleTimestamps: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateScheduleTimestamps",
    );
    pub const validateStateUpdate: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateStateUpdate",
    );
    pub const validateStringField: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateStringField",
    );
    pub const validateTimestamp: &str = concat!(
        "go.temporal.io/server/service/frontend.",
        "validateTimestamp",
    );
}
