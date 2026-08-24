import Temporal.API.Generated.Wire

namespace Umpire3.Temporal.API.Failure

inductive Kind where
  | application
  | timeout
  | canceled
  | terminated
  | server
  | resetWorkflow
  | activity
  | childWorkflow
  | nexusOperation
  | nexusHandler
  | generic
  deriving DecidableEq, Repr

structure Normalized where
  kind : Kind
  hasCause : Bool
  hasEncodedAttributes : Bool
  deriving DecidableEq, Repr

def interpret (failure : Generated.Failure) : Normalized := {
  kind :=
    if failure.applicationFailureInfo.isSome then .application
    else if failure.timeoutFailureInfo.isSome then .timeout
    else if failure.canceledFailureInfo.isSome then .canceled
    else if failure.terminatedFailureInfo.isSome then .terminated
    else if failure.serverFailureInfo.isSome then .server
    else if failure.resetWorkflowFailureInfo.isSome then .resetWorkflow
    else if failure.activityFailureInfo.isSome then .activity
    else if failure.childWorkflowExecutionFailureInfo.isSome then .childWorkflow
    else if failure.nexusOperationExecutionFailureInfo.isSome then .nexusOperation
    else if failure.nexusHandlerFailureInfo.isSome then .nexusHandler
    else .generic
  hasCause := failure.cause.isSome
  hasEncodedAttributes := failure.encodedAttributes.isSome
}

theorem interpretation_does_not_expose_message (failure : Generated.Failure) :
    interpret failure = {
      kind := (interpret failure).kind
      hasCause := failure.cause.isSome
      hasEncodedAttributes := failure.encodedAttributes.isSome
    } := by
  simp [interpret]

end Umpire3.Temporal.API.Failure
