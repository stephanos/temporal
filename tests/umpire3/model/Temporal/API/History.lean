import Temporal.API.Generated.Wire
import Temporal.API.Interpretation

namespace Umpire3.Temporal.API.History

open Umpire3.Temporal.API

inductive InterpretationError where
  | missingIdentity
  | invalidEventReference
  | missingFailure
  | invalidLinkVariant
  | missingCallback
  deriving DecidableEq, Repr

inductive NexusOutcome where
  | scheduled (service operation requestId : RequiredText)
  | started (scheduledEventId : Int) (operationId requestId : RequiredText)
  | completed (scheduledEventId : Int) (requestId : RequiredText)
  | failed (scheduledEventId : Int) (requestId : RequiredText)
  | canceled (scheduledEventId : Int) (requestId : RequiredText)
  | timedOut (scheduledEventId : Int) (requestId : RequiredText)

inductive LinkedIdentity where
  | workflowEvent (namespaceName workflowId runId : RequiredText)
  | activity (namespaceName activityId runId : RequiredText)
  | nexusOperation (namespaceName operationId runId : RequiredText)
  deriving Repr

structure WorkflowLineage where
  continuedExecutionRunId : String
  originalExecutionRunId : RequiredText
  firstExecutionRunId : RequiredText
  completionCallbackCount : Nat

structure ContinuedAsNewLineage where
  newExecutionRunId : RequiredText
  workflowTaskCompletedEventId : Int

structure ActivityScheduled where
  activityId : RequiredText
  workflowTaskCompletedEventId : Int

structure ActivityCompleted where
  scheduledEventId : Int
  startedEventId : Int

private def parseScheduled (event : Generated.NexusOperationScheduledEventAttributes) :
    Except InterpretationError NexusOutcome := do
  let some service := RequiredText.fromString event.service
    | throw .missingIdentity
  let some operation := RequiredText.fromString event.operation
    | throw .missingIdentity
  let some requestId := RequiredText.fromString event.requestId
    | throw .missingIdentity
  return .scheduled service operation requestId

def interpretScheduled (event : Generated.NexusOperationScheduledEventAttributes) :
    Interpretation NexusOutcome InterpretationError :=
  match parseScheduled event with
  | .ok outcome => .accepted outcome
  | .error failure => .rejected failure

private def parseStarted (event : Generated.NexusOperationStartedEventAttributes) :
    Except InterpretationError NexusOutcome := do
  if event.scheduledEventId ≤ 0 then throw .invalidEventReference
  else
    let some operationId := RequiredText.fromString event.operationId
      | throw .missingIdentity
    let some requestId := RequiredText.fromString event.requestId
      | throw .missingIdentity
    return .started event.scheduledEventId operationId requestId

def interpretStarted (event : Generated.NexusOperationStartedEventAttributes) :
    Interpretation NexusOutcome InterpretationError :=
  match parseStarted event with
  | .ok outcome => .accepted outcome
  | .error failure => .rejected failure

private def parseCompleted (event : Generated.NexusOperationCompletedEventAttributes) :
    Except InterpretationError NexusOutcome := do
  if event.scheduledEventId ≤ 0 then throw .invalidEventReference
  else
    let some requestId := RequiredText.fromString event.requestId
      | throw .missingIdentity
    return .completed event.scheduledEventId requestId

def interpretCompleted (event : Generated.NexusOperationCompletedEventAttributes) :
    Interpretation NexusOutcome InterpretationError :=
  match parseCompleted event with
  | .ok outcome => .accepted outcome
  | .error failure => .rejected failure

private def parseFailed (event : Generated.NexusOperationFailedEventAttributes) :
    Except InterpretationError NexusOutcome := do
  if event.failure.isNone then throw .missingFailure
  else if event.scheduledEventId ≤ 0 then throw .invalidEventReference
  else
    let some requestId := RequiredText.fromString event.requestId
      | throw .missingIdentity
    return .failed event.scheduledEventId requestId

def interpretFailed (event : Generated.NexusOperationFailedEventAttributes) :
    Interpretation NexusOutcome InterpretationError :=
  match parseFailed event with
  | .ok outcome => .accepted outcome
  | .error failure => .rejected failure

private def parseCanceled (event : Generated.NexusOperationCanceledEventAttributes) :
    Except InterpretationError NexusOutcome := do
  if event.failure.isNone then throw .missingFailure
  else if event.scheduledEventId ≤ 0 then throw .invalidEventReference
  else
    let some requestId := RequiredText.fromString event.requestId
      | throw .missingIdentity
    return .canceled event.scheduledEventId requestId

def interpretCanceled (event : Generated.NexusOperationCanceledEventAttributes) :
    Interpretation NexusOutcome InterpretationError :=
  match parseCanceled event with
  | .ok outcome => .accepted outcome
  | .error failure => .rejected failure

private def parseTimedOut (event : Generated.NexusOperationTimedOutEventAttributes) :
    Except InterpretationError NexusOutcome := do
  if event.failure.isNone then throw .missingFailure
  else if event.scheduledEventId ≤ 0 then throw .invalidEventReference
  else
    let some requestId := RequiredText.fromString event.requestId
      | throw .missingIdentity
    return .timedOut event.scheduledEventId requestId

def interpretTimedOut (event : Generated.NexusOperationTimedOutEventAttributes) :
    Interpretation NexusOutcome InterpretationError :=
  match parseTimedOut event with
  | .ok outcome => .accepted outcome
  | .error failure => .rejected failure

private def parseLink (link : Generated.TemporalApiCommonV1Link) :
    Except InterpretationError LinkedIdentity := do
  match link.workflowEvent, link.activity, link.nexusOperation with
  | some workflow, none, none =>
      let some namespaceName := RequiredText.fromString workflow.namespaceName
        | throw .missingIdentity
      let some workflowId := RequiredText.fromString workflow.workflowId
        | throw .missingIdentity
      let some runId := RequiredText.fromString workflow.runId
        | throw .missingIdentity
      return .workflowEvent namespaceName workflowId runId
  | none, some activity, none =>
      let some namespaceName := RequiredText.fromString activity.namespaceName
        | throw .missingIdentity
      let some activityId := RequiredText.fromString activity.activityId
        | throw .missingIdentity
      let some runId := RequiredText.fromString activity.runId
        | throw .missingIdentity
      return .activity namespaceName activityId runId
  | none, none, some operation =>
      let some namespaceName := RequiredText.fromString operation.namespaceName
        | throw .missingIdentity
      let some operationId := RequiredText.fromString operation.operationId
        | throw .missingIdentity
      let some runId := RequiredText.fromString operation.runId
        | throw .missingIdentity
      return .nexusOperation namespaceName operationId runId
  | _, _, _ => throw .invalidLinkVariant

def interpretLink (link : Generated.TemporalApiCommonV1Link) :
    Interpretation LinkedIdentity InterpretationError :=
  match parseLink link with
  | .ok identity => .accepted identity
  | .error failure => .rejected failure

private def parseWorkflowLineage (event : Generated.WorkflowExecutionStartedEventAttributes) :
    Except InterpretationError WorkflowLineage := do
  let some originalExecutionRunId := RequiredText.fromString event.originalExecutionRunId
    | throw .missingIdentity
  let some firstExecutionRunId := RequiredText.fromString event.firstExecutionRunId
    | throw .missingIdentity
  return {
    continuedExecutionRunId := event.continuedExecutionRunId
    originalExecutionRunId
    firstExecutionRunId
    completionCallbackCount := event.completionCallbacks.length
  }

def interpretWorkflowLineage (event : Generated.WorkflowExecutionStartedEventAttributes) :
    Interpretation WorkflowLineage InterpretationError :=
  match parseWorkflowLineage event with
  | .ok lineage => .accepted lineage
  | .error failure => .rejected failure

private def parseContinuedAsNew (event : Generated.WorkflowExecutionContinuedAsNewEventAttributes) :
    Except InterpretationError ContinuedAsNewLineage := do
  let some newExecutionRunId := RequiredText.fromString event.newExecutionRunId
    | throw .missingIdentity
  if event.workflowTaskCompletedEventId ≤ 0 then throw .invalidEventReference
  return { newExecutionRunId, workflowTaskCompletedEventId := event.workflowTaskCompletedEventId }

def interpretContinuedAsNew (event : Generated.WorkflowExecutionContinuedAsNewEventAttributes) :
    Interpretation ContinuedAsNewLineage InterpretationError :=
  match parseContinuedAsNew event with
  | .ok lineage => .accepted lineage
  | .error failure => .rejected failure

private def parseActivityScheduled (event : Generated.ActivityTaskScheduledEventAttributes) :
    Except InterpretationError ActivityScheduled := do
  let some activityId := RequiredText.fromString event.activityId
    | throw .missingIdentity
  if event.workflowTaskCompletedEventId ≤ 0 then throw .invalidEventReference
  return { activityId, workflowTaskCompletedEventId := event.workflowTaskCompletedEventId }

def interpretActivityScheduled (event : Generated.ActivityTaskScheduledEventAttributes) :
    Interpretation ActivityScheduled InterpretationError :=
  match parseActivityScheduled event with
  | .ok identity => .accepted identity
  | .error failure => .rejected failure

private def parseActivityCompleted (event : Generated.ActivityTaskCompletedEventAttributes) :
    Except InterpretationError ActivityCompleted := do
  if event.scheduledEventId ≤ 0 || event.startedEventId ≤ 0 then throw .invalidEventReference
  return { scheduledEventId := event.scheduledEventId, startedEventId := event.startedEventId }

def interpretActivityCompleted (event : Generated.ActivityTaskCompletedEventAttributes) :
    Interpretation ActivityCompleted InterpretationError :=
  match parseActivityCompleted event with
  | .ok identity => .accepted identity
  | .error failure => .rejected failure

end Umpire3.Temporal.API.History
