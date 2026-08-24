import Temporal.API.Generated.Wire
import Temporal.API.Interpretation

namespace Umpire3.Temporal.API.Workflow

open Umpire3.Temporal.API

inductive InterpretationError where
  | missingNamespace
  | missingWorkflow
  | missingWorkflowType
  | missingTaskQueue
  | missingRequestID
  | missingSignal
  | missingQuery
  | missingReason
  | invalidResetPoint
  | missingActivity
  | missingCallback
  deriving DecidableEq, Repr

structure StartCommand where
  namespaceName : RequiredText
  workflowId : RequiredText
  workflowType : RequiredText
  taskQueue : RequiredText
  requestId : RequiredText
  hasSensitiveInput : Bool

structure SignalCommand where
  namespaceName : RequiredText
  workflowId : RequiredText
  runId : String
  signalName : RequiredText
  requestId : RequiredText
  hasSensitiveInput : Bool

structure QueryCommand where
  namespaceName : RequiredText
  workflowId : RequiredText
  runId : String
  queryType : RequiredText
  hasSensitiveArguments : Bool

structure ResetCommand where
  namespaceName : RequiredText
  workflowId : RequiredText
  runId : RequiredText
  reason : RequiredText
  workflowTaskFinishEventId : PositiveInt
  requestId : RequiredText

structure ActivityCallbackCommand where
  namespaceName : RequiredText
  activityId : RequiredText
  activityType : RequiredText
  taskQueue : RequiredText
  requestId : RequiredText
  callbackURLs : NonemptyList RequiredText
  linkCount : Nat
  hasSensitiveInput : Bool

private def parseStart (request : Generated.StartWorkflowExecutionRequest) :
    Except InterpretationError StartCommand := do
  let some namespaceName := RequiredText.fromString request.namespaceName
    | throw .missingNamespace
  let some workflowId := RequiredText.fromString request.workflowId
    | throw .missingWorkflow
  let some workflowType := request.workflowType
    | throw .missingWorkflowType
  let some workflowTypeName := RequiredText.fromString workflowType.name
    | throw .missingWorkflowType
  let some taskQueue := request.taskQueue
    | throw .missingTaskQueue
  let some taskQueueName := RequiredText.fromString taskQueue.name
    | throw .missingTaskQueue
  let some requestId := RequiredText.fromString request.requestId
    | throw .missingRequestID
  return {
    namespaceName
    workflowId
    workflowType := workflowTypeName
    taskQueue := taskQueueName
    requestId
    hasSensitiveInput := request.input.isSome
  }

def interpretStart (request : Generated.StartWorkflowExecutionRequest) :
    Interpretation StartCommand InterpretationError :=
  match parseStart request with
  | .ok command => .accepted command
  | .error failure => .rejected failure

private def parseSignal (request : Generated.SignalWorkflowExecutionRequest) :
    Except InterpretationError SignalCommand := do
  let some namespaceName := RequiredText.fromString request.namespaceName
    | throw .missingNamespace
  let some execution := request.workflowExecution
    | throw .missingWorkflow
  let some workflowId := RequiredText.fromString execution.workflowId
    | throw .missingWorkflow
  let some signalName := RequiredText.fromString request.signalName
    | throw .missingSignal
  let some requestId := RequiredText.fromString request.requestId
    | throw .missingRequestID
  return {
    namespaceName
    workflowId
    runId := execution.runId
    signalName
    requestId
    hasSensitiveInput := request.input.isSome
  }

def interpretSignal (request : Generated.SignalWorkflowExecutionRequest) :
    Interpretation SignalCommand InterpretationError :=
  match parseSignal request with
  | .ok command => .accepted command
  | .error failure => .rejected failure

private def parseQuery (request : Generated.QueryWorkflowRequest) :
    Except InterpretationError QueryCommand := do
  let some namespaceName := RequiredText.fromString request.namespaceName
    | throw .missingNamespace
  let some execution := request.execution
    | throw .missingWorkflow
  let some workflowId := RequiredText.fromString execution.workflowId
    | throw .missingWorkflow
  let some query := request.query
    | throw .missingQuery
  let some queryType := RequiredText.fromString query.queryType
    | throw .missingQuery
  return {
    namespaceName
    workflowId
    runId := execution.runId
    queryType
    hasSensitiveArguments := query.queryArgs.isSome
  }

def interpretQuery (request : Generated.QueryWorkflowRequest) :
    Interpretation QueryCommand InterpretationError :=
  match parseQuery request with
  | .ok command => .accepted command
  | .error failure => .rejected failure

private def parseReset (request : Generated.ResetWorkflowExecutionRequest) :
    Except InterpretationError ResetCommand := do
  let some namespaceName := RequiredText.fromString request.namespaceName
    | throw .missingNamespace
  let some execution := request.workflowExecution
    | throw .missingWorkflow
  let some workflowId := RequiredText.fromString execution.workflowId
    | throw .missingWorkflow
  let some runId := RequiredText.fromString execution.runId
    | throw .missingWorkflow
  let some reason := RequiredText.fromString request.reason
    | throw .missingReason
  let some workflowTaskFinishEventId := PositiveInt.fromInt request.workflowTaskFinishEventId
    | throw .invalidResetPoint
  let some requestId := RequiredText.fromString request.requestId
    | throw .missingRequestID
  return {
    namespaceName
    workflowId
    runId
    reason
    workflowTaskFinishEventId
    requestId
  }

def interpretReset (request : Generated.ResetWorkflowExecutionRequest) :
    Interpretation ResetCommand InterpretationError :=
  match parseReset request with
  | .ok command => .accepted command
  | .error failure => .rejected failure

private def callbackURL (callback : Generated.Callback) : Option RequiredText := do
  let nexus ← callback.nexus
  RequiredText.fromString nexus.url

private def parseActivityCallback (request : Generated.StartActivityExecutionRequest) :
    Except InterpretationError ActivityCallbackCommand := do
  let some namespaceName := RequiredText.fromString request.namespaceName
    | throw .missingNamespace
  let some activityId := RequiredText.fromString request.activityId
    | throw .missingActivity
  let some activityType := request.activityType
    | throw .missingActivity
  let some activityTypeName := RequiredText.fromString activityType.name
    | throw .missingActivity
  let some taskQueue := request.taskQueue
    | throw .missingTaskQueue
  let some taskQueueName := RequiredText.fromString taskQueue.name
    | throw .missingTaskQueue
  let some requestId := RequiredText.fromString request.requestId
    | throw .missingRequestID
  let callbackValues := request.completionCallbacks.filterMap callbackURL
  if callbackValues.length != request.completionCallbacks.length then
    throw .missingCallback
  let some callbackURLs := NonemptyList.fromList callbackValues
    | throw .missingCallback
  return {
    namespaceName
    activityId
    activityType := activityTypeName
    taskQueue := taskQueueName
    requestId
    callbackURLs
    linkCount := request.links.length
    hasSensitiveInput := request.input.isSome
  }

def interpretActivityCallback (request : Generated.StartActivityExecutionRequest) :
    Interpretation ActivityCallbackCommand InterpretationError :=
  match parseActivityCallback request with
  | .ok command => .accepted command
  | .error failure => .rejected failure

theorem accepted_start_has_namespace {request command}
    (_accepted : interpretStart request = .accepted command) :
    command.namespaceName.value ≠ "" := command.namespaceName.present

theorem accepted_signal_has_workflow {request command}
    (_accepted : interpretSignal request = .accepted command) :
    command.workflowId.value ≠ "" := command.workflowId.present

theorem accepted_query_has_query_type {request command}
    (_accepted : interpretQuery request = .accepted command) :
    command.queryType.value ≠ "" := command.queryType.present

theorem accepted_reset_has_positive_event {request command}
    (_accepted : interpretReset request = .accepted command) :
    command.workflowTaskFinishEventId.value > 0 := command.workflowTaskFinishEventId.positive

theorem accepted_activity_has_callback {request command}
    (_accepted : interpretActivityCallback request = .accepted command) :
    command.callbackURLs.values ≠ [] := command.callbackURLs.present

end Umpire3.Temporal.API.Workflow
