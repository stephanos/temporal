namespace Umpire3.Temporal.API.Generated

def descriptorHash : String := "sha256:082b6b66bbd5faf7ccd88deb28e49df237e5fa9202fa8a233c6db4a608edf081"

structure RedactedBytes where
  digest : String
  size : Nat
  deriving DecidableEq, Repr

structure BoundedMessage where
  descriptor : String
  remainingDepth : Nat
  deriving DecidableEq, Repr

structure FieldMetadata where
  path : String
  kind : String
  presence : Bool
  oneofName : String
  repeated : Bool
  mapField : Bool
  disposition : String
  deriving DecidableEq, Repr

structure FieldDomain where
  path : String
  cases : List String
  deriving DecidableEq, Repr

structure ActivityIdConflictPolicy where
  number : Int
  deriving DecidableEq, Repr

namespace ActivityIdConflictPolicy
def activityIdConflictPolicyUnspecified : ActivityIdConflictPolicy := { number := 0 }
def activityIdConflictPolicyFail : ActivityIdConflictPolicy := { number := 1 }
def activityIdConflictPolicyUseExisting : ActivityIdConflictPolicy := { number := 2 }
end ActivityIdConflictPolicy

structure ActivityIdReusePolicy where
  number : Int
  deriving DecidableEq, Repr

namespace ActivityIdReusePolicy
def activityIdReusePolicyUnspecified : ActivityIdReusePolicy := { number := 0 }
def activityIdReusePolicyAllowDuplicate : ActivityIdReusePolicy := { number := 1 }
def activityIdReusePolicyAllowDuplicateFailedOnly : ActivityIdReusePolicy := { number := 2 }
def activityIdReusePolicyRejectDuplicate : ActivityIdReusePolicy := { number := 3 }
end ActivityIdReusePolicy

structure ApplicationErrorCategory where
  number : Int
  deriving DecidableEq, Repr

namespace ApplicationErrorCategory
def applicationErrorCategoryUnspecified : ApplicationErrorCategory := { number := 0 }
def applicationErrorCategoryBenign : ApplicationErrorCategory := { number := 1 }
end ApplicationErrorCategory

structure ContinueAsNewInitiator where
  number : Int
  deriving DecidableEq, Repr

namespace ContinueAsNewInitiator
def continueAsNewInitiatorUnspecified : ContinueAsNewInitiator := { number := 0 }
def continueAsNewInitiatorWorkflow : ContinueAsNewInitiator := { number := 1 }
def continueAsNewInitiatorRetry : ContinueAsNewInitiator := { number := 2 }
def continueAsNewInitiatorCronSchedule : ContinueAsNewInitiator := { number := 3 }
end ContinueAsNewInitiator

structure ContinueAsNewVersioningBehavior where
  number : Int
  deriving DecidableEq, Repr

namespace ContinueAsNewVersioningBehavior
def continueAsNewVersioningBehaviorUnspecified : ContinueAsNewVersioningBehavior := { number := 0 }
def continueAsNewVersioningBehaviorAutoUpgrade : ContinueAsNewVersioningBehavior := { number := 1 }
def continueAsNewVersioningBehaviorUseRampingVersion : ContinueAsNewVersioningBehavior := { number := 2 }
end ContinueAsNewVersioningBehavior

structure EventType where
  number : Int
  deriving DecidableEq, Repr

namespace EventType
def eventTypeUnspecified : EventType := { number := 0 }
def eventTypeWorkflowExecutionStarted : EventType := { number := 1 }
def eventTypeWorkflowExecutionCompleted : EventType := { number := 2 }
def eventTypeWorkflowExecutionFailed : EventType := { number := 3 }
def eventTypeWorkflowExecutionTimedOut : EventType := { number := 4 }
def eventTypeWorkflowTaskScheduled : EventType := { number := 5 }
def eventTypeWorkflowTaskStarted : EventType := { number := 6 }
def eventTypeWorkflowTaskCompleted : EventType := { number := 7 }
def eventTypeWorkflowTaskTimedOut : EventType := { number := 8 }
def eventTypeWorkflowTaskFailed : EventType := { number := 9 }
def eventTypeActivityTaskScheduled : EventType := { number := 10 }
def eventTypeActivityTaskStarted : EventType := { number := 11 }
def eventTypeActivityTaskCompleted : EventType := { number := 12 }
def eventTypeActivityTaskFailed : EventType := { number := 13 }
def eventTypeActivityTaskTimedOut : EventType := { number := 14 }
def eventTypeActivityTaskCancelRequested : EventType := { number := 15 }
def eventTypeActivityTaskCanceled : EventType := { number := 16 }
def eventTypeTimerStarted : EventType := { number := 17 }
def eventTypeTimerFired : EventType := { number := 18 }
def eventTypeTimerCanceled : EventType := { number := 19 }
def eventTypeWorkflowExecutionCancelRequested : EventType := { number := 20 }
def eventTypeWorkflowExecutionCanceled : EventType := { number := 21 }
def eventTypeRequestCancelExternalWorkflowExecutionInitiated : EventType := { number := 22 }
def eventTypeRequestCancelExternalWorkflowExecutionFailed : EventType := { number := 23 }
def eventTypeExternalWorkflowExecutionCancelRequested : EventType := { number := 24 }
def eventTypeMarkerRecorded : EventType := { number := 25 }
def eventTypeWorkflowExecutionSignaled : EventType := { number := 26 }
def eventTypeWorkflowExecutionTerminated : EventType := { number := 27 }
def eventTypeWorkflowExecutionContinuedAsNew : EventType := { number := 28 }
def eventTypeStartChildWorkflowExecutionInitiated : EventType := { number := 29 }
def eventTypeStartChildWorkflowExecutionFailed : EventType := { number := 30 }
def eventTypeChildWorkflowExecutionStarted : EventType := { number := 31 }
def eventTypeChildWorkflowExecutionCompleted : EventType := { number := 32 }
def eventTypeChildWorkflowExecutionFailed : EventType := { number := 33 }
def eventTypeChildWorkflowExecutionCanceled : EventType := { number := 34 }
def eventTypeChildWorkflowExecutionTimedOut : EventType := { number := 35 }
def eventTypeChildWorkflowExecutionTerminated : EventType := { number := 36 }
def eventTypeSignalExternalWorkflowExecutionInitiated : EventType := { number := 37 }
def eventTypeSignalExternalWorkflowExecutionFailed : EventType := { number := 38 }
def eventTypeExternalWorkflowExecutionSignaled : EventType := { number := 39 }
def eventTypeUpsertWorkflowSearchAttributes : EventType := { number := 40 }
def eventTypeWorkflowExecutionUpdateAdmitted : EventType := { number := 47 }
def eventTypeWorkflowExecutionUpdateAccepted : EventType := { number := 41 }
def eventTypeWorkflowExecutionUpdateRejected : EventType := { number := 42 }
def eventTypeWorkflowExecutionUpdateCompleted : EventType := { number := 43 }
def eventTypeWorkflowPropertiesModifiedExternally : EventType := { number := 44 }
def eventTypeActivityPropertiesModifiedExternally : EventType := { number := 45 }
def eventTypeWorkflowPropertiesModified : EventType := { number := 46 }
def eventTypeNexusOperationScheduled : EventType := { number := 48 }
def eventTypeNexusOperationStarted : EventType := { number := 49 }
def eventTypeNexusOperationCompleted : EventType := { number := 50 }
def eventTypeNexusOperationFailed : EventType := { number := 51 }
def eventTypeNexusOperationCanceled : EventType := { number := 52 }
def eventTypeNexusOperationTimedOut : EventType := { number := 53 }
def eventTypeNexusOperationCancelRequested : EventType := { number := 54 }
def eventTypeWorkflowExecutionOptionsUpdated : EventType := { number := 55 }
def eventTypeNexusOperationCancelRequestCompleted : EventType := { number := 56 }
def eventTypeNexusOperationCancelRequestFailed : EventType := { number := 57 }
def eventTypeWorkflowExecutionPaused : EventType := { number := 58 }
def eventTypeWorkflowExecutionUnpaused : EventType := { number := 59 }
def eventTypeWorkflowExecutionTimeSkippingTransitioned : EventType := { number := 60 }
end EventType

structure NexusHandlerErrorRetryBehavior where
  number : Int
  deriving DecidableEq, Repr

namespace NexusHandlerErrorRetryBehavior
def nexusHandlerErrorRetryBehaviorUnspecified : NexusHandlerErrorRetryBehavior := { number := 0 }
def nexusHandlerErrorRetryBehaviorRetryable : NexusHandlerErrorRetryBehavior := { number := 1 }
def nexusHandlerErrorRetryBehaviorNonRetryable : NexusHandlerErrorRetryBehavior := { number := 2 }
end NexusHandlerErrorRetryBehavior

structure NexusOperationIdConflictPolicy where
  number : Int
  deriving DecidableEq, Repr

namespace NexusOperationIdConflictPolicy
def nexusOperationIdConflictPolicyUnspecified : NexusOperationIdConflictPolicy := { number := 0 }
def nexusOperationIdConflictPolicyFail : NexusOperationIdConflictPolicy := { number := 1 }
def nexusOperationIdConflictPolicyUseExisting : NexusOperationIdConflictPolicy := { number := 2 }
end NexusOperationIdConflictPolicy

structure NexusOperationIdReusePolicy where
  number : Int
  deriving DecidableEq, Repr

namespace NexusOperationIdReusePolicy
def nexusOperationIdReusePolicyUnspecified : NexusOperationIdReusePolicy := { number := 0 }
def nexusOperationIdReusePolicyAllowDuplicate : NexusOperationIdReusePolicy := { number := 1 }
def nexusOperationIdReusePolicyAllowDuplicateFailedOnly : NexusOperationIdReusePolicy := { number := 2 }
def nexusOperationIdReusePolicyRejectDuplicate : NexusOperationIdReusePolicy := { number := 3 }
end NexusOperationIdReusePolicy

structure QueryRejectCondition where
  number : Int
  deriving DecidableEq, Repr

namespace QueryRejectCondition
def queryRejectConditionUnspecified : QueryRejectCondition := { number := 0 }
def queryRejectConditionNone : QueryRejectCondition := { number := 1 }
def queryRejectConditionNotOpen : QueryRejectCondition := { number := 2 }
def queryRejectConditionNotCompletedCleanly : QueryRejectCondition := { number := 3 }
end QueryRejectCondition

structure ResetReapplyExcludeType where
  number : Int
  deriving DecidableEq, Repr

namespace ResetReapplyExcludeType
def resetReapplyExcludeTypeUnspecified : ResetReapplyExcludeType := { number := 0 }
def resetReapplyExcludeTypeSignal : ResetReapplyExcludeType := { number := 1 }
def resetReapplyExcludeTypeUpdate : ResetReapplyExcludeType := { number := 2 }
def resetReapplyExcludeTypeNexus : ResetReapplyExcludeType := { number := 3 }
def resetReapplyExcludeTypeCancelRequest : ResetReapplyExcludeType := { number := 4 }
end ResetReapplyExcludeType

structure ResetReapplyType where
  number : Int
  deriving DecidableEq, Repr

namespace ResetReapplyType
def resetReapplyTypeUnspecified : ResetReapplyType := { number := 0 }
def resetReapplyTypeSignal : ResetReapplyType := { number := 1 }
def resetReapplyTypeNone : ResetReapplyType := { number := 2 }
def resetReapplyTypeAllEligible : ResetReapplyType := { number := 3 }
end ResetReapplyType

structure RetryState where
  number : Int
  deriving DecidableEq, Repr

namespace RetryState
def retryStateUnspecified : RetryState := { number := 0 }
def retryStateInProgress : RetryState := { number := 1 }
def retryStateNonRetryableFailure : RetryState := { number := 2 }
def retryStateTimeout : RetryState := { number := 3 }
def retryStateMaximumAttemptsReached : RetryState := { number := 4 }
def retryStateRetryPolicyNotSet : RetryState := { number := 5 }
def retryStateInternalServerError : RetryState := { number := 6 }
def retryStateCancelRequested : RetryState := { number := 7 }
end RetryState

structure TaskQueueKind where
  number : Int
  deriving DecidableEq, Repr

namespace TaskQueueKind
def taskQueueKindUnspecified : TaskQueueKind := { number := 0 }
def taskQueueKindNormal : TaskQueueKind := { number := 1 }
def taskQueueKindSticky : TaskQueueKind := { number := 2 }
def taskQueueKindWorkerCommands : TaskQueueKind := { number := 3 }
end TaskQueueKind

structure TimeoutType where
  number : Int
  deriving DecidableEq, Repr

namespace TimeoutType
def timeoutTypeUnspecified : TimeoutType := { number := 0 }
def timeoutTypeStartToClose : TimeoutType := { number := 1 }
def timeoutTypeScheduleToStart : TimeoutType := { number := 2 }
def timeoutTypeScheduleToClose : TimeoutType := { number := 3 }
def timeoutTypeHeartbeat : TimeoutType := { number := 4 }
end TimeoutType

structure UpdateWorkflowExecutionLifecycleStage where
  number : Int
  deriving DecidableEq, Repr

namespace UpdateWorkflowExecutionLifecycleStage
def updateWorkflowExecutionLifecycleStageUnspecified : UpdateWorkflowExecutionLifecycleStage := { number := 0 }
def updateWorkflowExecutionLifecycleStageAdmitted : UpdateWorkflowExecutionLifecycleStage := { number := 1 }
def updateWorkflowExecutionLifecycleStageAccepted : UpdateWorkflowExecutionLifecycleStage := { number := 2 }
def updateWorkflowExecutionLifecycleStageCompleted : UpdateWorkflowExecutionLifecycleStage := { number := 3 }
end UpdateWorkflowExecutionLifecycleStage

structure VersioningBehavior where
  number : Int
  deriving DecidableEq, Repr

namespace VersioningBehavior
def versioningBehaviorUnspecified : VersioningBehavior := { number := 0 }
def versioningBehaviorPinned : VersioningBehavior := { number := 1 }
def versioningBehaviorAutoUpgrade : VersioningBehavior := { number := 2 }
end VersioningBehavior

structure WorkerVersioningMode where
  number : Int
  deriving DecidableEq, Repr

namespace WorkerVersioningMode
def workerVersioningModeUnspecified : WorkerVersioningMode := { number := 0 }
def workerVersioningModeUnversioned : WorkerVersioningMode := { number := 1 }
def workerVersioningModeVersioned : WorkerVersioningMode := { number := 2 }
end WorkerVersioningMode

structure WorkflowIdConflictPolicy where
  number : Int
  deriving DecidableEq, Repr

namespace WorkflowIdConflictPolicy
def workflowIdConflictPolicyUnspecified : WorkflowIdConflictPolicy := { number := 0 }
def workflowIdConflictPolicyFail : WorkflowIdConflictPolicy := { number := 1 }
def workflowIdConflictPolicyUseExisting : WorkflowIdConflictPolicy := { number := 2 }
def workflowIdConflictPolicyTerminateExisting : WorkflowIdConflictPolicy := { number := 3 }
end WorkflowIdConflictPolicy

structure WorkflowIdReusePolicy where
  number : Int
  deriving DecidableEq, Repr

namespace WorkflowIdReusePolicy
def workflowIdReusePolicyUnspecified : WorkflowIdReusePolicy := { number := 0 }
def workflowIdReusePolicyAllowDuplicate : WorkflowIdReusePolicy := { number := 1 }
def workflowIdReusePolicyAllowDuplicateFailedOnly : WorkflowIdReusePolicy := { number := 2 }
def workflowIdReusePolicyRejectDuplicate : WorkflowIdReusePolicy := { number := 3 }
def workflowIdReusePolicyTerminateIfRunning : WorkflowIdReusePolicy := { number := 4 }
end WorkflowIdReusePolicy

structure PinnedOverrideBehavior where
  number : Int
  deriving DecidableEq, Repr

namespace PinnedOverrideBehavior
def pinnedOverrideBehaviorUnspecified : PinnedOverrideBehavior := { number := 0 }
def pinnedOverrideBehaviorPinned : PinnedOverrideBehavior := { number := 1 }
end PinnedOverrideBehavior

structure Duration where
  seconds : Int
  nanos : Int
  deriving Repr

structure FieldMask where
  paths : List String
  deriving Repr

structure Timestamp where
  seconds : Int
  nanos : Int
  deriving Repr

structure ActivityType where
  name : String
  deriving Repr

structure Internal where
  data : RedactedBytes
  deriving Repr

structure Nexus where
  url : String
  header : List (String × String)
  deriving Repr

structure Activity where
  namespaceName : String
  activityId : String
  runId : String
  deriving Repr

structure BatchJob where
  jobId : String
  deriving Repr

structure NexusOperation where
  namespaceName : String
  operationId : String
  runId : String
  deriving Repr

structure Workflow where
  namespaceName : String
  workflowId : String
  runId : String
  reason : String
  deriving Repr

structure EventReference where
  eventId : Int
  eventType : EventType
  deriving Repr

structure RequestIdReference where
  requestId : String
  eventType : EventType
  deriving Repr

structure WorkflowEvent where
  namespaceName : String
  workflowId : String
  runId : String
  eventRef : Option EventReference
  requestIdRef : Option RequestIdReference
  deriving Repr

structure TemporalApiCommonV1Link where
  workflowEvent : Option WorkflowEvent
  batchJob : Option BatchJob
  activity : Option Activity
  nexusOperation : Option NexusOperation
  workflow : Option Workflow
  deriving Repr

structure Callback where
  nexus : Option Nexus
  internal : Option Internal
  links : List TemporalApiCommonV1Link
  deriving Repr

structure ExternalPayloadDetails where
  sizeBytes : Int
  deriving Repr

structure Payload where
  metadata : List (String × RedactedBytes)
  data : RedactedBytes
  externalPayloads : List ExternalPayloadDetails
  deriving Repr

structure Header where
  fields : List (String × Payload)
  deriving Repr

structure Memo where
  fields : List (String × Payload)
  deriving Repr

structure TemporalApiCommonV1OnConflictOptions where
  attachRequestId : Bool
  attachCompletionCallbacks : Bool
  attachLinks : Bool
  deriving Repr

structure Payloads where
  payloads : List Payload
  deriving Repr

structure Priority where
  priorityKey : Int
  fairnessKey : String
  fairnessWeight : Float
  deriving Repr

structure RetryPolicy where
  initialInterval : Option Duration
  backoffCoefficient : Float
  maximumInterval : Option Duration
  maximumAttempts : Int
  nonRetryableErrorTypes : List String
  deriving Repr

structure SearchAttributes where
  indexedFields : List (String × Payload)
  deriving Repr

structure TimeSkippingConfig where
  enabled : Bool
  fastForward : Option Duration
  disablePropagation : Bool
  deriving Repr

structure TimeSkippingStatePropagation where
  initialSkippedDuration : Option Duration
  fastForwardTargetTime : Option Timestamp
  deriving Repr

structure WorkerVersionStamp where
  buildId : String
  useVersioning : Bool
  deriving Repr

structure WorkflowExecution where
  workflowId : String
  runId : String
  deriving Repr

structure WorkflowType where
  name : String
  deriving Repr

structure Deployment where
  seriesName : String
  buildId : String
  deriving Repr

structure WorkerDeploymentVersion where
  buildId : String
  deploymentName : String
  deriving Repr

structure InheritedAutoUpgradeInfo where
  sourceDeploymentVersion : Option WorkerDeploymentVersion
  sourceDeploymentRevisionNumber : Int
  continueAsNewInitialVersioningBehavior : ContinueAsNewVersioningBehavior
  deriving Repr

structure WorkerDeploymentOptions where
  deploymentName : String
  buildId : String
  workerVersioningMode : WorkerVersioningMode
  deriving Repr

structure ActivityFailureInfo where
  scheduledEventId : Int
  startedEventId : Int
  identity : String
  activityType : Option ActivityType
  activityId : String
  retryState : RetryState
  deriving Repr

structure ApplicationFailureInfo where
  typeName : String
  nonRetryable : Bool
  details : Option Payloads
  nextRetryDelay : Option Duration
  category : ApplicationErrorCategory
  deriving Repr

structure CanceledFailureInfo where
  details : Option Payloads
  identity : String
  deriving Repr

structure ChildWorkflowExecutionFailureInfo where
  namespaceName : String
  workflowExecution : Option WorkflowExecution
  workflowType : Option WorkflowType
  initiatedEventId : Int
  startedEventId : Int
  retryState : RetryState
  deriving Repr

structure NexusHandlerFailureInfo where
  typeName : String
  retryBehavior : NexusHandlerErrorRetryBehavior
  deriving Repr

structure NexusOperationFailureInfo where
  scheduledEventId : Int
  endpoint : String
  service : String
  operation : String
  operationId : String
  operationToken : String
  deriving Repr

structure ResetWorkflowFailureInfo where
  lastHeartbeatDetails : Option Payloads
  deriving Repr

structure ServerFailureInfo where
  nonRetryable : Bool
  deriving Repr

structure TerminatedFailureInfo where
  identity : String
  deriving Repr

structure TimeoutFailureInfo where
  timeoutType : TimeoutType
  lastHeartbeatDetails : Option Payloads
  deriving Repr

structure Failure where
  message : String
  source : String
  stackTrace : String
  encodedAttributes : Option Payload
  cause : Option BoundedMessage
  applicationFailureInfo : Option ApplicationFailureInfo
  timeoutFailureInfo : Option TimeoutFailureInfo
  canceledFailureInfo : Option CanceledFailureInfo
  terminatedFailureInfo : Option TerminatedFailureInfo
  serverFailureInfo : Option ServerFailureInfo
  resetWorkflowFailureInfo : Option ResetWorkflowFailureInfo
  activityFailureInfo : Option ActivityFailureInfo
  childWorkflowExecutionFailureInfo : Option ChildWorkflowExecutionFailureInfo
  nexusOperationExecutionFailureInfo : Option NexusOperationFailureInfo
  nexusHandlerFailureInfo : Option NexusHandlerFailureInfo
  deriving Repr

structure ActivityTaskCompletedEventAttributes where
  result : Option Payloads
  scheduledEventId : Int
  startedEventId : Int
  identity : String
  workerVersion : Option WorkerVersionStamp
  deriving Repr

structure TaskQueue where
  name : String
  kind : TaskQueueKind
  normalName : String
  deriving Repr

structure ActivityTaskScheduledEventAttributes where
  activityId : String
  activityType : Option ActivityType
  taskQueue : Option TaskQueue
  header : Option Header
  input : Option Payloads
  scheduleToCloseTimeout : Option Duration
  scheduleToStartTimeout : Option Duration
  startToCloseTimeout : Option Duration
  heartbeatTimeout : Option Duration
  workflowTaskCompletedEventId : Int
  retryPolicy : Option RetryPolicy
  useWorkflowBuildId : Bool
  priority : Option Priority
  deriving Repr

structure DeclinedTargetVersionUpgrade where
  deploymentVersion : Option WorkerDeploymentVersion
  revisionNumber : Int
  deriving Repr

structure NexusOperationCanceledEventAttributes where
  scheduledEventId : Int
  failure : Option Failure
  requestId : String
  deriving Repr

structure NexusOperationCompletedEventAttributes where
  scheduledEventId : Int
  result : Option Payload
  requestId : String
  deriving Repr

structure NexusOperationFailedEventAttributes where
  scheduledEventId : Int
  failure : Option Failure
  requestId : String
  deriving Repr

structure NexusOperationScheduledEventAttributes where
  endpoint : String
  service : String
  operation : String
  input : Option Payload
  scheduleToCloseTimeout : Option Duration
  nexusHeader : List (String × String)
  workflowTaskCompletedEventId : Int
  requestId : String
  endpointId : String
  scheduleToStartTimeout : Option Duration
  startToCloseTimeout : Option Duration
  deriving Repr

structure NexusOperationStartedEventAttributes where
  scheduledEventId : Int
  operationId : String
  requestId : String
  operationToken : String
  deriving Repr

structure NexusOperationTimedOutEventAttributes where
  scheduledEventId : Int
  failure : Option Failure
  requestId : String
  deriving Repr

structure WorkflowExecutionContinuedAsNewEventAttributes where
  newExecutionRunId : String
  workflowType : Option WorkflowType
  taskQueue : Option TaskQueue
  input : Option Payloads
  workflowRunTimeout : Option Duration
  workflowTaskTimeout : Option Duration
  workflowTaskCompletedEventId : Int
  backoffStartInterval : Option Duration
  initiator : ContinueAsNewInitiator
  failure : Option Failure
  lastCompletionResult : Option Payloads
  header : Option Header
  memo : Option Memo
  searchAttributes : Option SearchAttributes
  inheritBuildId : Bool
  initialVersioningBehavior : ContinueAsNewVersioningBehavior
  deriving Repr

structure WorkflowExecutionSignaledEventAttributes where
  signalName : String
  input : Option Payloads
  identity : String
  header : Option Header
  skipGenerateWorkflowTask : Bool
  externalWorkflowExecution : Option WorkflowExecution
  requestId : String
  deriving Repr

structure ResetPointInfo where
  buildId : String
  binaryChecksum : String
  runId : String
  firstWorkflowTaskCompletedId : Int
  createTime : Option Timestamp
  expireTime : Option Timestamp
  resettable : Bool
  deriving Repr

structure ResetPoints where
  points : List ResetPointInfo
  deriving Repr

structure OneTimeOverride where
  targetDeploymentVersion : Option WorkerDeploymentVersion
  deriving Repr

structure PinnedOverride where
  behavior : PinnedOverrideBehavior
  version : Option WorkerDeploymentVersion
  deriving Repr

structure VersioningOverride where
  pinned : Option PinnedOverride
  autoUpgrade : Option Bool
  oneTime : Option OneTimeOverride
  behavior : VersioningBehavior
  deployment : Option Deployment
  pinnedVersion : String
  deriving Repr

structure WorkflowExecutionStartedEventAttributes where
  workflowType : Option WorkflowType
  parentWorkflowNamespace : String
  parentWorkflowNamespaceId : String
  parentWorkflowExecution : Option WorkflowExecution
  parentInitiatedEventId : Int
  taskQueue : Option TaskQueue
  input : Option Payloads
  workflowExecutionTimeout : Option Duration
  workflowRunTimeout : Option Duration
  workflowTaskTimeout : Option Duration
  continuedExecutionRunId : String
  initiator : ContinueAsNewInitiator
  continuedFailure : Option Failure
  lastCompletionResult : Option Payloads
  originalExecutionRunId : String
  identity : String
  firstExecutionRunId : String
  retryPolicy : Option RetryPolicy
  attempt : Int
  workflowExecutionExpirationTime : Option Timestamp
  cronSchedule : String
  firstWorkflowTaskBackoff : Option Duration
  memo : Option Memo
  searchAttributes : Option SearchAttributes
  prevAutoResetPoints : Option ResetPoints
  header : Option Header
  parentInitiatedEventVersion : Int
  workflowId : String
  sourceVersionStamp : Option WorkerVersionStamp
  completionCallbacks : List Callback
  rootWorkflowExecution : Option WorkflowExecution
  inheritedBuildId : String
  versioningOverride : Option VersioningOverride
  parentPinnedWorkerDeploymentVersion : String
  priority : Option Priority
  inheritedPinnedVersion : Option WorkerDeploymentVersion
  inheritedAutoUpgradeInfo : Option InheritedAutoUpgradeInfo
  eagerExecutionAccepted : Bool
  declinedTargetVersionUpgrade : Option DeclinedTargetVersionUpgrade
  timeSkippingConfig : Option TimeSkippingConfig
  timeSkippingStatePropagation : Option TimeSkippingStatePropagation
  deriving Repr

structure Input where
  header : Option Header
  name : String
  args : Option Payloads
  deriving Repr

structure Meta where
  updateId : String
  identity : String
  deriving Repr

structure Request where
  metadata : Option Meta
  input : Option Input
  requestId : String
  completionCallbacks : List Callback
  links : List TemporalApiCommonV1Link
  deriving Repr

structure WorkflowExecutionUpdateAcceptedEventAttributes where
  protocolInstanceId : String
  acceptedRequestMessageId : String
  acceptedRequestSequencingEventId : Int
  acceptedRequest : Option Request
  deriving Repr

structure TemporalApiNexusV1Link where
  url : String
  typeName : String
  deriving Repr

structure StartOperationRequest where
  service : String
  operation : String
  requestId : String
  callback : String
  payload : Option Payload
  callbackHeader : List (String × String)
  links : List TemporalApiNexusV1Link
  deriving Repr

structure WorkflowQuery where
  queryType : String
  queryArgs : Option Payloads
  header : Option Header
  deriving Repr

structure UserMetadata where
  summary : Option Payload
  details : Option Payload
  deriving Repr

structure WaitPolicy where
  lifecycleStage : UpdateWorkflowExecutionLifecycleStage
  deriving Repr

structure TemporalApiWorkflowV1OnConflictOptions where
  attachRequestId : Bool
  attachCompletionCallbacks : Bool
  attachLinks : Bool
  deriving Repr

structure SignalWorkflow where
  signalName : String
  input : Option Payloads
  header : Option Header
  links : List TemporalApiCommonV1Link
  deriving Repr

structure WorkflowExecutionOptions where
  versioningOverride : Option VersioningOverride
  priority : Option Priority
  timeSkippingConfig : Option TimeSkippingConfig
  deriving Repr

structure UpdateWorkflowOptions where
  workflowExecutionOptions : Option WorkflowExecutionOptions
  updateMask : Option FieldMask
  deriving Repr

structure PostResetOperation where
  signalWorkflow : Option SignalWorkflow
  updateWorkflowOptions : Option UpdateWorkflowOptions
  deriving Repr

structure QueryWorkflowRequest where
  namespaceName : String
  execution : Option WorkflowExecution
  query : Option WorkflowQuery
  queryRejectCondition : QueryRejectCondition
  deriving Repr

structure RequestCancelNexusOperationExecutionRequest where
  namespaceName : String
  operationId : String
  runId : String
  identity : String
  requestId : String
  reason : String
  deriving Repr

structure ResetWorkflowExecutionRequest where
  namespaceName : String
  workflowExecution : Option WorkflowExecution
  reason : String
  workflowTaskFinishEventId : Int
  requestId : String
  resetReapplyType : ResetReapplyType
  resetReapplyExcludeTypes : List ResetReapplyExcludeType
  postResetOperations : List PostResetOperation
  identity : String
  deriving Repr

structure SignalWorkflowExecutionRequest where
  namespaceName : String
  workflowExecution : Option WorkflowExecution
  signalName : String
  input : Option Payloads
  identity : String
  requestId : String
  control : String
  header : Option Header
  links : List TemporalApiCommonV1Link
  deriving Repr

structure StartActivityExecutionRequest where
  namespaceName : String
  identity : String
  requestId : String
  activityId : String
  activityType : Option ActivityType
  taskQueue : Option TaskQueue
  scheduleToCloseTimeout : Option Duration
  scheduleToStartTimeout : Option Duration
  startToCloseTimeout : Option Duration
  heartbeatTimeout : Option Duration
  retryPolicy : Option RetryPolicy
  input : Option Payloads
  idReusePolicy : ActivityIdReusePolicy
  idConflictPolicy : ActivityIdConflictPolicy
  searchAttributes : Option SearchAttributes
  header : Option Header
  userMetadata : Option UserMetadata
  priority : Option Priority
  completionCallbacks : List Callback
  links : List TemporalApiCommonV1Link
  onConflictOptions : Option TemporalApiCommonV1OnConflictOptions
  startDelay : Option Duration
  deriving Repr

structure StartNexusOperationExecutionRequest where
  namespaceName : String
  identity : String
  requestId : String
  operationId : String
  endpoint : String
  service : String
  operation : String
  scheduleToCloseTimeout : Option Duration
  scheduleToStartTimeout : Option Duration
  startToCloseTimeout : Option Duration
  input : Option Payload
  idReusePolicy : NexusOperationIdReusePolicy
  idConflictPolicy : NexusOperationIdConflictPolicy
  searchAttributes : Option SearchAttributes
  nexusHeader : List (String × String)
  userMetadata : Option UserMetadata
  deriving Repr

structure StartWorkflowExecutionRequest where
  namespaceName : String
  workflowId : String
  workflowType : Option WorkflowType
  taskQueue : Option TaskQueue
  input : Option Payloads
  workflowExecutionTimeout : Option Duration
  workflowRunTimeout : Option Duration
  workflowTaskTimeout : Option Duration
  identity : String
  requestId : String
  workflowIdReusePolicy : WorkflowIdReusePolicy
  workflowIdConflictPolicy : WorkflowIdConflictPolicy
  retryPolicy : Option RetryPolicy
  cronSchedule : String
  memo : Option Memo
  searchAttributes : Option SearchAttributes
  header : Option Header
  requestEagerExecution : Bool
  continuedFailure : Option Failure
  lastCompletionResult : Option Payloads
  workflowStartDelay : Option Duration
  completionCallbacks : List Callback
  userMetadata : Option UserMetadata
  links : List TemporalApiCommonV1Link
  versioningOverride : Option VersioningOverride
  onConflictOptions : Option TemporalApiWorkflowV1OnConflictOptions
  priority : Option Priority
  eagerWorkerDeploymentOptions : Option WorkerDeploymentOptions
  timeSkippingConfig : Option TimeSkippingConfig
  deriving Repr

structure UpdateWorkflowExecutionRequest where
  namespaceName : String
  workflowExecution : Option WorkflowExecution
  firstExecutionRunId : String
  waitPolicy : Option WaitPolicy
  request : Option Request
  deriving Repr

def fieldMetadata : List FieldMetadata := [
  { path := "google.protobuf.Duration.seconds", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "google.protobuf.Duration.nanos", kind := "int32", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "google.protobuf.FieldMask.paths", kind := "string", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "google.protobuf.Timestamp.seconds", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "google.protobuf.Timestamp.nanos", kind := "int32", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.ActivityType.name", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Callback.Internal.data", kind := "bytes", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Callback.Nexus.url", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Callback.Nexus.header", kind := "message", presence := false, oneofName := "", repeated := false, mapField := true, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.Activity.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.Activity.activity_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.Activity.run_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.BatchJob.job_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.NexusOperation.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.NexusOperation.operation_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.NexusOperation.run_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.Workflow.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.Workflow.workflow_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.Workflow.run_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.Workflow.reason", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.EventReference.event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.EventReference.event_type", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.RequestIdReference.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.RequestIdReference.event_type", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.workflow_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.run_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.event_ref", kind := "message", presence := true, oneofName := "reference", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.request_id_ref", kind := "message", presence := true, oneofName := "reference", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Link.workflow_event", kind := "message", presence := true, oneofName := "variant", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.common.v1.Link.batch_job", kind := "message", presence := true, oneofName := "variant", repeated := false, mapField := false, disposition := "intentionally-unmodeled" },
  { path := "temporal.api.common.v1.Link.activity", kind := "message", presence := true, oneofName := "variant", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.common.v1.Link.nexus_operation", kind := "message", presence := true, oneofName := "variant", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.common.v1.Link.workflow", kind := "message", presence := true, oneofName := "variant", repeated := false, mapField := false, disposition := "intentionally-unmodeled" },
  { path := "temporal.api.common.v1.Callback.nexus", kind := "message", presence := true, oneofName := "variant", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.common.v1.Callback.internal", kind := "message", presence := true, oneofName := "variant", repeated := false, mapField := false, disposition := "intentionally-unmodeled" },
  { path := "temporal.api.common.v1.Callback.links", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.common.v1.Payload.ExternalPayloadDetails.size_bytes", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Payload.metadata", kind := "message", presence := false, oneofName := "", repeated := false, mapField := true, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Payload.data", kind := "bytes", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Payload.external_payloads", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Header.fields", kind := "message", presence := false, oneofName := "", repeated := false, mapField := true, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Memo.fields", kind := "message", presence := false, oneofName := "", repeated := false, mapField := true, disposition := "transport-only" },
  { path := "temporal.api.common.v1.OnConflictOptions.attach_request_id", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.OnConflictOptions.attach_completion_callbacks", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.OnConflictOptions.attach_links", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Payloads.payloads", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Priority.priority_key", kind := "int32", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Priority.fairness_key", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.Priority.fairness_weight", kind := "float", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.RetryPolicy.initial_interval", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.RetryPolicy.backoff_coefficient", kind := "double", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.RetryPolicy.maximum_interval", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.RetryPolicy.maximum_attempts", kind := "int32", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.RetryPolicy.non_retryable_error_types", kind := "string", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.SearchAttributes.indexed_fields", kind := "message", presence := false, oneofName := "", repeated := false, mapField := true, disposition := "transport-only" },
  { path := "temporal.api.common.v1.TimeSkippingConfig.enabled", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.TimeSkippingConfig.fast_forward", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.TimeSkippingConfig.disable_propagation", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.TimeSkippingStatePropagation.initial_skipped_duration", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.TimeSkippingStatePropagation.fast_forward_target_time", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.WorkerVersionStamp.build_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.WorkerVersionStamp.use_versioning", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.WorkflowExecution.workflow_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.WorkflowExecution.run_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.common.v1.WorkflowType.name", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.deployment.v1.Deployment.series_name", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.deployment.v1.Deployment.build_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.deployment.v1.WorkerDeploymentVersion.build_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.deployment.v1.WorkerDeploymentVersion.deployment_name", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.deployment.v1.InheritedAutoUpgradeInfo.source_deployment_version", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.deployment.v1.InheritedAutoUpgradeInfo.source_deployment_revision_number", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.deployment.v1.InheritedAutoUpgradeInfo.continue_as_new_initial_versioning_behavior", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.deployment.v1.WorkerDeploymentOptions.deployment_name", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.deployment.v1.WorkerDeploymentOptions.build_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.deployment.v1.WorkerDeploymentOptions.worker_versioning_mode", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ActivityFailureInfo.scheduled_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ActivityFailureInfo.started_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ActivityFailureInfo.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ActivityFailureInfo.activity_type", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ActivityFailureInfo.activity_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ActivityFailureInfo.retry_state", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ApplicationFailureInfo.type", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ApplicationFailureInfo.non_retryable", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ApplicationFailureInfo.details", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ApplicationFailureInfo.next_retry_delay", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ApplicationFailureInfo.category", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.CanceledFailureInfo.details", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.CanceledFailureInfo.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.workflow_execution", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.workflow_type", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.initiated_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.started_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.retry_state", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.NexusHandlerFailureInfo.type", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.NexusHandlerFailureInfo.retry_behavior", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.NexusOperationFailureInfo.scheduled_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.NexusOperationFailureInfo.endpoint", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.NexusOperationFailureInfo.service", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.NexusOperationFailureInfo.operation", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.NexusOperationFailureInfo.operation_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.NexusOperationFailureInfo.operation_token", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ResetWorkflowFailureInfo.last_heartbeat_details", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.ServerFailureInfo.non_retryable", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.TerminatedFailureInfo.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.TimeoutFailureInfo.timeout_type", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.TimeoutFailureInfo.last_heartbeat_details", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.Failure.message", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.failure.v1.Failure.source", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.Failure.stack_trace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.failure.v1.Failure.encoded_attributes", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.Failure.cause", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.failure.v1.Failure.application_failure_info", kind := "message", presence := true, oneofName := "failure_info", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.Failure.timeout_failure_info", kind := "message", presence := true, oneofName := "failure_info", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.Failure.canceled_failure_info", kind := "message", presence := true, oneofName := "failure_info", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.Failure.terminated_failure_info", kind := "message", presence := true, oneofName := "failure_info", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.Failure.server_failure_info", kind := "message", presence := true, oneofName := "failure_info", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.Failure.reset_workflow_failure_info", kind := "message", presence := true, oneofName := "failure_info", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.Failure.activity_failure_info", kind := "message", presence := true, oneofName := "failure_info", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.Failure.child_workflow_execution_failure_info", kind := "message", presence := true, oneofName := "failure_info", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.Failure.nexus_operation_execution_failure_info", kind := "message", presence := true, oneofName := "failure_info", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.failure.v1.Failure.nexus_handler_failure_info", kind := "message", presence := true, oneofName := "failure_info", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.ActivityTaskCompletedEventAttributes.result", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.history.v1.ActivityTaskCompletedEventAttributes.scheduled_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.ActivityTaskCompletedEventAttributes.started_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.ActivityTaskCompletedEventAttributes.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.history.v1.ActivityTaskCompletedEventAttributes.worker_version", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.taskqueue.v1.TaskQueue.name", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.taskqueue.v1.TaskQueue.kind", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.taskqueue.v1.TaskQueue.normal_name", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.activity_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.activity_type", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.task_queue", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.header", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.input", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.schedule_to_close_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.schedule_to_start_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.start_to_close_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.heartbeat_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.workflow_task_completed_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.retry_policy", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.use_workflow_build_id", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.priority", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.DeclinedTargetVersionUpgrade.deployment_version", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.DeclinedTargetVersionUpgrade.revision_number", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationCanceledEventAttributes.scheduled_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.NexusOperationCanceledEventAttributes.failure", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.history.v1.NexusOperationCanceledEventAttributes.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationCompletedEventAttributes.scheduled_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.NexusOperationCompletedEventAttributes.result", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.history.v1.NexusOperationCompletedEventAttributes.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationFailedEventAttributes.scheduled_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.NexusOperationFailedEventAttributes.failure", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.history.v1.NexusOperationFailedEventAttributes.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.endpoint", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.service", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.operation", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.input", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.schedule_to_close_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.nexus_header", kind := "message", presence := false, oneofName := "", repeated := false, mapField := true, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.workflow_task_completed_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.endpoint_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.schedule_to_start_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.start_to_close_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationStartedEventAttributes.scheduled_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationStartedEventAttributes.operation_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.NexusOperationStartedEventAttributes.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationStartedEventAttributes.operation_token", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.NexusOperationTimedOutEventAttributes.scheduled_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.NexusOperationTimedOutEventAttributes.failure", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.history.v1.NexusOperationTimedOutEventAttributes.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.new_execution_run_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.workflow_type", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.task_queue", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.input", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.workflow_run_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.workflow_task_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.workflow_task_completed_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.backoff_start_interval", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.initiator", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.failure", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.last_completion_result", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.header", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.memo", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.search_attributes", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.inherit_build_id", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.initial_versioning_behavior", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.signal_name", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.input", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.header", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.skip_generate_workflow_task", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.external_workflow_execution", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.ResetPointInfo.build_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.ResetPointInfo.binary_checksum", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.ResetPointInfo.run_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.ResetPointInfo.first_workflow_task_completed_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.ResetPointInfo.create_time", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.ResetPointInfo.expire_time", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.ResetPointInfo.resettable", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.ResetPoints.points", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.VersioningOverride.OneTimeOverride.target_deployment_version", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.VersioningOverride.PinnedOverride.behavior", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.VersioningOverride.PinnedOverride.version", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.VersioningOverride.pinned", kind := "message", presence := true, oneofName := "override", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.VersioningOverride.auto_upgrade", kind := "bool", presence := true, oneofName := "override", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.VersioningOverride.one_time", kind := "message", presence := true, oneofName := "override", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.VersioningOverride.behavior", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.VersioningOverride.deployment", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.VersioningOverride.pinned_version", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.workflow_type", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.parent_workflow_namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.parent_workflow_namespace_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.parent_workflow_execution", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.parent_initiated_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.task_queue", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.input", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.workflow_execution_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.workflow_run_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.workflow_task_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.continued_execution_run_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.initiator", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.continued_failure", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.last_completion_result", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.original_execution_run_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.first_execution_run_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.retry_policy", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.attempt", kind := "int32", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.workflow_execution_expiration_time", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.cron_schedule", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.first_workflow_task_backoff", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.memo", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.search_attributes", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.prev_auto_reset_points", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.header", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.parent_initiated_event_version", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.workflow_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.source_version_stamp", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.completion_callbacks", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.root_workflow_execution", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.inherited_build_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.versioning_override", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.parent_pinned_worker_deployment_version", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.priority", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.inherited_pinned_version", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.inherited_auto_upgrade_info", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.eager_execution_accepted", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.declined_target_version_upgrade", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.time_skipping_config", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.time_skipping_state_propagation", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.update.v1.Input.header", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.update.v1.Input.name", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.update.v1.Input.args", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.update.v1.Meta.update_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.update.v1.Meta.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.update.v1.Request.meta", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.update.v1.Request.input", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.update.v1.Request.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.update.v1.Request.completion_callbacks", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.update.v1.Request.links", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionUpdateAcceptedEventAttributes.protocol_instance_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.history.v1.WorkflowExecutionUpdateAcceptedEventAttributes.accepted_request_message_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionUpdateAcceptedEventAttributes.accepted_request_sequencing_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.history.v1.WorkflowExecutionUpdateAcceptedEventAttributes.accepted_request", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.nexus.v1.Link.url", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.nexus.v1.Link.type", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.nexus.v1.StartOperationRequest.service", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.nexus.v1.StartOperationRequest.operation", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.nexus.v1.StartOperationRequest.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.nexus.v1.StartOperationRequest.callback", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.nexus.v1.StartOperationRequest.payload", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.nexus.v1.StartOperationRequest.callback_header", kind := "message", presence := false, oneofName := "", repeated := false, mapField := true, disposition := "transport-only" },
  { path := "temporal.api.nexus.v1.StartOperationRequest.links", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.query.v1.WorkflowQuery.query_type", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.query.v1.WorkflowQuery.query_args", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.query.v1.WorkflowQuery.header", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.sdk.v1.UserMetadata.summary", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.sdk.v1.UserMetadata.details", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.update.v1.WaitPolicy.lifecycle_stage", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.OnConflictOptions.attach_request_id", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.OnConflictOptions.attach_completion_callbacks", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.OnConflictOptions.attach_links", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.PostResetOperation.SignalWorkflow.signal_name", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.PostResetOperation.SignalWorkflow.input", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.PostResetOperation.SignalWorkflow.header", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.PostResetOperation.SignalWorkflow.links", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.WorkflowExecutionOptions.versioning_override", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.WorkflowExecutionOptions.priority", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.WorkflowExecutionOptions.time_skipping_config", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.PostResetOperation.UpdateWorkflowOptions.workflow_execution_options", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.PostResetOperation.UpdateWorkflowOptions.update_mask", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.PostResetOperation.signal_workflow", kind := "message", presence := true, oneofName := "variant", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflow.v1.PostResetOperation.update_workflow_options", kind := "message", presence := true, oneofName := "variant", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.QueryWorkflowRequest.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.QueryWorkflowRequest.execution", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.QueryWorkflowRequest.query", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.workflowservice.v1.QueryWorkflowRequest.query_reject_condition", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest.operation_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest.run_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest.reason", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.workflow_execution", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.reason", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.workflow_task_finish_event_id", kind := "int64", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.reset_reapply_type", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.reset_reapply_exclude_types", kind := "enum", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.post_reset_operations", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.workflow_execution", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.signal_name", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.input", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.control", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.header", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.links", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.activity_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.activity_type", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.task_queue", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.schedule_to_close_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.schedule_to_start_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.start_to_close_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.heartbeat_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.retry_policy", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.input", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.id_reuse_policy", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.id_conflict_policy", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.search_attributes", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.header", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.user_metadata", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.priority", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.completion_callbacks", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.links", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.on_conflict_options", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.start_delay", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.operation_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.endpoint", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.service", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.operation", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.schedule_to_close_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.schedule_to_start_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.start_to_close_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.input", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.id_reuse_policy", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.id_conflict_policy", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.search_attributes", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.nexus_header", kind := "message", presence := false, oneofName := "", repeated := false, mapField := true, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.user_metadata", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_type", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.task_queue", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.input", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_execution_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_run_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_task_timeout", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.identity", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.request_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_id_reuse_policy", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_id_conflict_policy", kind := "enum", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.retry_policy", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.cron_schedule", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.memo", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.search_attributes", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.header", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.request_eager_execution", kind := "bool", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.continued_failure", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.last_completion_result", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_start_delay", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.completion_callbacks", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.user_metadata", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.links", kind := "message", presence := false, oneofName := "", repeated := true, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.versioning_override", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.on_conflict_options", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.priority", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.eager_worker_deployment_options", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.time_skipping_config", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.UpdateWorkflowExecutionRequest.namespace", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.UpdateWorkflowExecutionRequest.workflow_execution", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "interpreted" },
  { path := "temporal.api.workflowservice.v1.UpdateWorkflowExecutionRequest.first_execution_run_id", kind := "string", presence := false, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.UpdateWorkflowExecutionRequest.wait_policy", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "transport-only" },
  { path := "temporal.api.workflowservice.v1.UpdateWorkflowExecutionRequest.request", kind := "message", presence := true, oneofName := "", repeated := false, mapField := false, disposition := "sensitive" },
]

end Umpire3.Temporal.API.Generated

namespace Umpire3.Temporal.API.Generated

def fieldDomains : List FieldDomain := [
  { path := "google.protobuf.Duration.seconds", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "google.protobuf.Duration.nanos", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "google.protobuf.FieldMask.paths", cases := ["empty", "non-empty", "empty-list", "ordered-list"] },
  { path := "google.protobuf.Timestamp.seconds", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "google.protobuf.Timestamp.nanos", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.common.v1.ActivityType.name", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Callback.Internal.data", cases := ["empty-digest", "non-empty-digest"] },
  { path := "temporal.api.common.v1.Callback.Nexus.url", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Callback.Nexus.header", cases := ["default", "non-default", "map-permuted-keys"] },
  { path := "temporal.api.common.v1.Link.Activity.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.Activity.activity_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.Activity.run_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.BatchJob.job_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.NexusOperation.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.NexusOperation.operation_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.NexusOperation.run_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.Workflow.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.Workflow.workflow_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.Workflow.run_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.Workflow.reason", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.EventReference.event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.EventReference.event_type", cases := ["known", "unknown-number"] },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.RequestIdReference.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.RequestIdReference.event_type", cases := ["known", "unknown-number"] },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.workflow_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.run_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.event_ref", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.common.v1.Link.WorkflowEvent.request_id_ref", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.common.v1.Link.workflow_event", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.common.v1.Link.batch_job", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.common.v1.Link.activity", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.common.v1.Link.nexus_operation", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.common.v1.Link.workflow", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.common.v1.Callback.nexus", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.common.v1.Callback.internal", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.common.v1.Callback.links", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.common.v1.Payload.ExternalPayloadDetails.size_bytes", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.common.v1.Payload.metadata", cases := ["default", "non-default", "map-permuted-keys"] },
  { path := "temporal.api.common.v1.Payload.data", cases := ["empty-digest", "non-empty-digest"] },
  { path := "temporal.api.common.v1.Payload.external_payloads", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.common.v1.Header.fields", cases := ["default", "non-default", "map-permuted-keys"] },
  { path := "temporal.api.common.v1.Memo.fields", cases := ["default", "non-default", "map-permuted-keys"] },
  { path := "temporal.api.common.v1.OnConflictOptions.attach_request_id", cases := ["false", "true"] },
  { path := "temporal.api.common.v1.OnConflictOptions.attach_completion_callbacks", cases := ["false", "true"] },
  { path := "temporal.api.common.v1.OnConflictOptions.attach_links", cases := ["false", "true"] },
  { path := "temporal.api.common.v1.Payloads.payloads", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.common.v1.Priority.priority_key", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.common.v1.Priority.fairness_key", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.Priority.fairness_weight", cases := ["default", "non-default"] },
  { path := "temporal.api.common.v1.RetryPolicy.initial_interval", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.common.v1.RetryPolicy.backoff_coefficient", cases := ["default", "non-default"] },
  { path := "temporal.api.common.v1.RetryPolicy.maximum_interval", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.common.v1.RetryPolicy.maximum_attempts", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.common.v1.RetryPolicy.non_retryable_error_types", cases := ["empty", "non-empty", "empty-list", "ordered-list"] },
  { path := "temporal.api.common.v1.SearchAttributes.indexed_fields", cases := ["default", "non-default", "map-permuted-keys"] },
  { path := "temporal.api.common.v1.TimeSkippingConfig.enabled", cases := ["false", "true"] },
  { path := "temporal.api.common.v1.TimeSkippingConfig.fast_forward", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.common.v1.TimeSkippingConfig.disable_propagation", cases := ["false", "true"] },
  { path := "temporal.api.common.v1.TimeSkippingStatePropagation.initial_skipped_duration", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.common.v1.TimeSkippingStatePropagation.fast_forward_target_time", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.common.v1.WorkerVersionStamp.build_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.WorkerVersionStamp.use_versioning", cases := ["false", "true"] },
  { path := "temporal.api.common.v1.WorkflowExecution.workflow_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.WorkflowExecution.run_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.common.v1.WorkflowType.name", cases := ["empty", "non-empty"] },
  { path := "temporal.api.deployment.v1.Deployment.series_name", cases := ["empty", "non-empty"] },
  { path := "temporal.api.deployment.v1.Deployment.build_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.deployment.v1.WorkerDeploymentVersion.build_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.deployment.v1.WorkerDeploymentVersion.deployment_name", cases := ["empty", "non-empty"] },
  { path := "temporal.api.deployment.v1.InheritedAutoUpgradeInfo.source_deployment_version", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.deployment.v1.InheritedAutoUpgradeInfo.source_deployment_revision_number", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.deployment.v1.InheritedAutoUpgradeInfo.continue_as_new_initial_versioning_behavior", cases := ["known", "unknown-number"] },
  { path := "temporal.api.deployment.v1.WorkerDeploymentOptions.deployment_name", cases := ["empty", "non-empty"] },
  { path := "temporal.api.deployment.v1.WorkerDeploymentOptions.build_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.deployment.v1.WorkerDeploymentOptions.worker_versioning_mode", cases := ["known", "unknown-number"] },
  { path := "temporal.api.failure.v1.ActivityFailureInfo.scheduled_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.failure.v1.ActivityFailureInfo.started_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.failure.v1.ActivityFailureInfo.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.ActivityFailureInfo.activity_type", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.failure.v1.ActivityFailureInfo.activity_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.ActivityFailureInfo.retry_state", cases := ["known", "unknown-number"] },
  { path := "temporal.api.failure.v1.ApplicationFailureInfo.type", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.ApplicationFailureInfo.non_retryable", cases := ["false", "true"] },
  { path := "temporal.api.failure.v1.ApplicationFailureInfo.details", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.failure.v1.ApplicationFailureInfo.next_retry_delay", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.failure.v1.ApplicationFailureInfo.category", cases := ["known", "unknown-number"] },
  { path := "temporal.api.failure.v1.CanceledFailureInfo.details", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.failure.v1.CanceledFailureInfo.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.workflow_execution", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.workflow_type", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.initiated_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.started_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.failure.v1.ChildWorkflowExecutionFailureInfo.retry_state", cases := ["known", "unknown-number"] },
  { path := "temporal.api.failure.v1.NexusHandlerFailureInfo.type", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.NexusHandlerFailureInfo.retry_behavior", cases := ["known", "unknown-number"] },
  { path := "temporal.api.failure.v1.NexusOperationFailureInfo.scheduled_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.failure.v1.NexusOperationFailureInfo.endpoint", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.NexusOperationFailureInfo.service", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.NexusOperationFailureInfo.operation", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.NexusOperationFailureInfo.operation_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.NexusOperationFailureInfo.operation_token", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.ResetWorkflowFailureInfo.last_heartbeat_details", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.failure.v1.ServerFailureInfo.non_retryable", cases := ["false", "true"] },
  { path := "temporal.api.failure.v1.TerminatedFailureInfo.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.TimeoutFailureInfo.timeout_type", cases := ["known", "unknown-number"] },
  { path := "temporal.api.failure.v1.TimeoutFailureInfo.last_heartbeat_details", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.failure.v1.Failure.message", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.Failure.source", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.Failure.stack_trace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.failure.v1.Failure.encoded_attributes", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.failure.v1.Failure.cause", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.failure.v1.Failure.application_failure_info", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.failure.v1.Failure.timeout_failure_info", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.failure.v1.Failure.canceled_failure_info", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.failure.v1.Failure.terminated_failure_info", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.failure.v1.Failure.server_failure_info", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.failure.v1.Failure.reset_workflow_failure_info", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.failure.v1.Failure.activity_failure_info", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.failure.v1.Failure.child_workflow_execution_failure_info", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.failure.v1.Failure.nexus_operation_execution_failure_info", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.failure.v1.Failure.nexus_handler_failure_info", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.history.v1.ActivityTaskCompletedEventAttributes.result", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.ActivityTaskCompletedEventAttributes.scheduled_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.ActivityTaskCompletedEventAttributes.started_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.ActivityTaskCompletedEventAttributes.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.ActivityTaskCompletedEventAttributes.worker_version", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.taskqueue.v1.TaskQueue.name", cases := ["empty", "non-empty"] },
  { path := "temporal.api.taskqueue.v1.TaskQueue.kind", cases := ["known", "unknown-number"] },
  { path := "temporal.api.taskqueue.v1.TaskQueue.normal_name", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.activity_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.activity_type", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.task_queue", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.header", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.input", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.schedule_to_close_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.schedule_to_start_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.start_to_close_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.heartbeat_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.workflow_task_completed_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.retry_policy", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.use_workflow_build_id", cases := ["false", "true"] },
  { path := "temporal.api.history.v1.ActivityTaskScheduledEventAttributes.priority", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.DeclinedTargetVersionUpgrade.deployment_version", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.DeclinedTargetVersionUpgrade.revision_number", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.NexusOperationCanceledEventAttributes.scheduled_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.NexusOperationCanceledEventAttributes.failure", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.NexusOperationCanceledEventAttributes.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.NexusOperationCompletedEventAttributes.scheduled_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.NexusOperationCompletedEventAttributes.result", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.NexusOperationCompletedEventAttributes.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.NexusOperationFailedEventAttributes.scheduled_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.NexusOperationFailedEventAttributes.failure", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.NexusOperationFailedEventAttributes.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.endpoint", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.service", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.operation", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.input", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.schedule_to_close_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.nexus_header", cases := ["default", "non-default", "map-permuted-keys"] },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.workflow_task_completed_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.endpoint_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.schedule_to_start_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.NexusOperationScheduledEventAttributes.start_to_close_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.NexusOperationStartedEventAttributes.scheduled_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.NexusOperationStartedEventAttributes.operation_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.NexusOperationStartedEventAttributes.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.NexusOperationStartedEventAttributes.operation_token", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.NexusOperationTimedOutEventAttributes.scheduled_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.NexusOperationTimedOutEventAttributes.failure", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.NexusOperationTimedOutEventAttributes.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.new_execution_run_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.workflow_type", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.task_queue", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.input", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.workflow_run_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.workflow_task_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.workflow_task_completed_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.backoff_start_interval", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.initiator", cases := ["known", "unknown-number"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.failure", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.last_completion_result", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.header", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.memo", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.search_attributes", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.inherit_build_id", cases := ["false", "true"] },
  { path := "temporal.api.history.v1.WorkflowExecutionContinuedAsNewEventAttributes.initial_versioning_behavior", cases := ["known", "unknown-number"] },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.signal_name", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.input", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.header", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.skip_generate_workflow_task", cases := ["false", "true"] },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.external_workflow_execution", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionSignaledEventAttributes.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflow.v1.ResetPointInfo.build_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflow.v1.ResetPointInfo.binary_checksum", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflow.v1.ResetPointInfo.run_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflow.v1.ResetPointInfo.first_workflow_task_completed_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.workflow.v1.ResetPointInfo.create_time", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflow.v1.ResetPointInfo.expire_time", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflow.v1.ResetPointInfo.resettable", cases := ["false", "true"] },
  { path := "temporal.api.workflow.v1.ResetPoints.points", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.workflow.v1.VersioningOverride.OneTimeOverride.target_deployment_version", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflow.v1.VersioningOverride.PinnedOverride.behavior", cases := ["known", "unknown-number"] },
  { path := "temporal.api.workflow.v1.VersioningOverride.PinnedOverride.version", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflow.v1.VersioningOverride.pinned", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.workflow.v1.VersioningOverride.auto_upgrade", cases := ["absent", "present-default", "false", "true", "oneof-replacement"] },
  { path := "temporal.api.workflow.v1.VersioningOverride.one_time", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.workflow.v1.VersioningOverride.behavior", cases := ["known", "unknown-number"] },
  { path := "temporal.api.workflow.v1.VersioningOverride.deployment", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflow.v1.VersioningOverride.pinned_version", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.workflow_type", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.parent_workflow_namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.parent_workflow_namespace_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.parent_workflow_execution", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.parent_initiated_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.task_queue", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.input", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.workflow_execution_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.workflow_run_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.workflow_task_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.continued_execution_run_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.initiator", cases := ["known", "unknown-number"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.continued_failure", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.last_completion_result", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.original_execution_run_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.first_execution_run_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.retry_policy", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.attempt", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.workflow_execution_expiration_time", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.cron_schedule", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.first_workflow_task_backoff", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.memo", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.search_attributes", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.prev_auto_reset_points", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.header", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.parent_initiated_event_version", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.workflow_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.source_version_stamp", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.completion_callbacks", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.root_workflow_execution", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.inherited_build_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.versioning_override", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.parent_pinned_worker_deployment_version", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.priority", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.inherited_pinned_version", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.inherited_auto_upgrade_info", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.eager_execution_accepted", cases := ["false", "true"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.declined_target_version_upgrade", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.time_skipping_config", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.history.v1.WorkflowExecutionStartedEventAttributes.time_skipping_state_propagation", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.update.v1.Input.header", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.update.v1.Input.name", cases := ["empty", "non-empty"] },
  { path := "temporal.api.update.v1.Input.args", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.update.v1.Meta.update_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.update.v1.Meta.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.update.v1.Request.meta", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.update.v1.Request.input", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.update.v1.Request.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.update.v1.Request.completion_callbacks", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.update.v1.Request.links", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.history.v1.WorkflowExecutionUpdateAcceptedEventAttributes.protocol_instance_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionUpdateAcceptedEventAttributes.accepted_request_message_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.history.v1.WorkflowExecutionUpdateAcceptedEventAttributes.accepted_request_sequencing_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.history.v1.WorkflowExecutionUpdateAcceptedEventAttributes.accepted_request", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.nexus.v1.Link.url", cases := ["empty", "non-empty"] },
  { path := "temporal.api.nexus.v1.Link.type", cases := ["empty", "non-empty"] },
  { path := "temporal.api.nexus.v1.StartOperationRequest.service", cases := ["empty", "non-empty"] },
  { path := "temporal.api.nexus.v1.StartOperationRequest.operation", cases := ["empty", "non-empty"] },
  { path := "temporal.api.nexus.v1.StartOperationRequest.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.nexus.v1.StartOperationRequest.callback", cases := ["empty", "non-empty"] },
  { path := "temporal.api.nexus.v1.StartOperationRequest.payload", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.nexus.v1.StartOperationRequest.callback_header", cases := ["default", "non-default", "map-permuted-keys"] },
  { path := "temporal.api.nexus.v1.StartOperationRequest.links", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.query.v1.WorkflowQuery.query_type", cases := ["empty", "non-empty"] },
  { path := "temporal.api.query.v1.WorkflowQuery.query_args", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.query.v1.WorkflowQuery.header", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.sdk.v1.UserMetadata.summary", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.sdk.v1.UserMetadata.details", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.update.v1.WaitPolicy.lifecycle_stage", cases := ["known", "unknown-number"] },
  { path := "temporal.api.workflow.v1.OnConflictOptions.attach_request_id", cases := ["false", "true"] },
  { path := "temporal.api.workflow.v1.OnConflictOptions.attach_completion_callbacks", cases := ["false", "true"] },
  { path := "temporal.api.workflow.v1.OnConflictOptions.attach_links", cases := ["false", "true"] },
  { path := "temporal.api.workflow.v1.PostResetOperation.SignalWorkflow.signal_name", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflow.v1.PostResetOperation.SignalWorkflow.input", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflow.v1.PostResetOperation.SignalWorkflow.header", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflow.v1.PostResetOperation.SignalWorkflow.links", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.workflow.v1.WorkflowExecutionOptions.versioning_override", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflow.v1.WorkflowExecutionOptions.priority", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflow.v1.WorkflowExecutionOptions.time_skipping_config", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflow.v1.PostResetOperation.UpdateWorkflowOptions.workflow_execution_options", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflow.v1.PostResetOperation.UpdateWorkflowOptions.update_mask", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflow.v1.PostResetOperation.signal_workflow", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.workflow.v1.PostResetOperation.update_workflow_options", cases := ["absent", "present-default", "default", "non-default", "oneof-replacement"] },
  { path := "temporal.api.workflowservice.v1.QueryWorkflowRequest.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.QueryWorkflowRequest.execution", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.QueryWorkflowRequest.query", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.QueryWorkflowRequest.query_reject_condition", cases := ["known", "unknown-number"] },
  { path := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest.operation_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest.run_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.RequestCancelNexusOperationExecutionRequest.reason", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.workflow_execution", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.reason", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.workflow_task_finish_event_id", cases := ["negative", "zero", "positive", "boundary"] },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.reset_reapply_type", cases := ["known", "unknown-number"] },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.reset_reapply_exclude_types", cases := ["known", "unknown-number", "empty-list", "ordered-list"] },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.post_reset_operations", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.workflow_execution", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.signal_name", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.input", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.control", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.header", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.SignalWorkflowExecutionRequest.links", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.activity_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.activity_type", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.task_queue", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.schedule_to_close_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.schedule_to_start_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.start_to_close_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.heartbeat_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.retry_policy", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.input", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.id_reuse_policy", cases := ["known", "unknown-number"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.id_conflict_policy", cases := ["known", "unknown-number"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.search_attributes", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.header", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.user_metadata", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.priority", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.completion_callbacks", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.links", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.on_conflict_options", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartActivityExecutionRequest.start_delay", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.operation_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.endpoint", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.service", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.operation", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.schedule_to_close_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.schedule_to_start_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.start_to_close_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.input", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.id_reuse_policy", cases := ["known", "unknown-number"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.id_conflict_policy", cases := ["known", "unknown-number"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.search_attributes", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.nexus_header", cases := ["default", "non-default", "map-permuted-keys"] },
  { path := "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest.user_metadata", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_type", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.task_queue", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.input", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_execution_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_run_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_task_timeout", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.identity", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.request_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_id_reuse_policy", cases := ["known", "unknown-number"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_id_conflict_policy", cases := ["known", "unknown-number"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.retry_policy", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.cron_schedule", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.memo", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.search_attributes", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.header", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.request_eager_execution", cases := ["false", "true"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.continued_failure", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.last_completion_result", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.workflow_start_delay", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.completion_callbacks", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.user_metadata", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.links", cases := ["default", "non-default", "empty-list", "ordered-list"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.versioning_override", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.on_conflict_options", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.priority", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.eager_worker_deployment_options", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.StartWorkflowExecutionRequest.time_skipping_config", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.UpdateWorkflowExecutionRequest.namespace", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.UpdateWorkflowExecutionRequest.workflow_execution", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.UpdateWorkflowExecutionRequest.first_execution_run_id", cases := ["empty", "non-empty"] },
  { path := "temporal.api.workflowservice.v1.UpdateWorkflowExecutionRequest.wait_policy", cases := ["absent", "present-default", "default", "non-default"] },
  { path := "temporal.api.workflowservice.v1.UpdateWorkflowExecutionRequest.request", cases := ["absent", "present-default", "default", "non-default"] },
]

end Umpire3.Temporal.API.Generated
