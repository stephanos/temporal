import Umpire.Behavior
import Umpire.Shared.Test

/-! Shared semantic vocabulary, traces, and helpers for the Behavior concern tests. -/

namespace Umpire.BehaviorTests

open Umpire

def id (value : String) : DefinitionId := Shared.Test.definitionId value

def source : SourceLocation := Shared.Test.sourceLocation "Umpire/Behavior/Tests.lean"

def metadata (value : String) (kind : DefinitionKind) : DefinitionMetadata :=
  Shared.Test.definitionMetadata value kind source (value ++ "/v1")

def cancellationCapability : DefinitionId := id "test.capability.cancellation"
def operationState : DefinitionId := id "test.state.resource-id"
def phaseState : DefinitionId := id "test.state.phase"
def requestCancel : DefinitionId := id "test.action.request-cancel"
def callerClose : DefinitionId := id "test.action.close"
def tick : DefinitionId := id "test.action.tick"
def retry : DefinitionId := id "test.action.retry"
def abort : DefinitionId := id "test.action.abort"
def noop : DefinitionId := id "test.action.noop"
def accepted : DefinitionId := id "test.outcome.accepted"
def rejected : DefinitionId := id "test.outcome.rejected"
def cancelRequested : DefinitionId := id "test.observation.cancel-requested"
def callerClosed : DefinitionId := id "test.observation.closed"

def context : BehaviorCheckContext := {
  definitions := [
    metadata cancellationCapability.value .capability,
    metadata operationState.value .state,
    metadata phaseState.value .state,
    metadata requestCancel.value .action,
    metadata callerClose.value .action,
    metadata tick.value .action,
    metadata retry.value .action,
    metadata abort.value .action,
    metadata noop.value .action,
    metadata accepted.value .outcome,
    metadata rejected.value .outcome,
    metadata cancelRequested.value .observation,
    metadata callerClosed.value .observation
  ]
}

def operationRole : ResourceRole := {
  id := id "test.role.resource"
  valueKind := .state
}

def operationA : ModelValue := ModelValue.named operationState "operation-a"
def operationB : ModelValue := ModelValue.named operationState "operation-b"
def initial : ModelValue := ModelValue.named phaseState "started"
def cancelling : ModelValue := ModelValue.named phaseState "cancelling"
def closed : ModelValue := ModelValue.named phaseState "closed"

def actionValue (action : DefinitionId) : ModelValue := ModelValue.named action action.value
def outcomeValue (outcome : DefinitionId) : ModelValue := ModelValue.named outcome outcome.value
def observationValue (observation : DefinitionId) : ModelValue :=
  ModelValue.named observation observation.value

def cancelStep (outcome : DefinitionId) :
    ModelTraceStep ModelValue ModelValue ModelValue ModelValue := {
  selectedAction := actionValue requestCancel
  modelOutcome := outcomeValue outcome
  resultingState := cancelling
  observations := [observationValue cancelRequested]
}

def closeStep : ModelTraceStep ModelValue ModelValue ModelValue ModelValue := {
  selectedAction := actionValue callerClose
  modelOutcome := outcomeValue accepted
  resultingState := closed
  observations := [observationValue callerClosed]
}

def tickStep : ModelTraceStep ModelValue ModelValue ModelValue ModelValue := {
  selectedAction := actionValue tick
  modelOutcome := outcomeValue accepted
  resultingState := cancelling
  observations := []
}

def traceWith
    (setupValue : ModelValue)
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue)) :
    BehaviorTrace := {
  setup := [{ role := operationRole.id, value := setupValue }]
  trace := { initialState := initial, steps }
}

def acceptedTrace : BehaviorTrace := traceWith operationA [cancelStep accepted, closeStep]
def rejectedTrace : BehaviorTrace := traceWith operationA [cancelStep rejected, closeStep]
def interleavedTrace : BehaviorTrace := traceWith operationA [cancelStep accepted, tickStep, closeStep]
def reversedTrace : BehaviorTrace := traceWith operationA [closeStep, cancelStep accepted]
def repeatedCancelTrace : BehaviorTrace :=
  traceWith operationA [cancelStep accepted, cancelStep rejected, closeStep]
def otherSetupTrace : BehaviorTrace := traceWith operationB [cancelStep accepted, closeStep]

def cancelOccurrence : NamedOccurrence := {
  id := id "test.occurrence.cancel"
  action := requestCancel
}

def closeOccurrence : NamedOccurrence := {
  id := id "test.occurrence.close"
  action := callerClose
}

def setupEqualsA : SetupConstraint := {
  id := id "test.setup.resource-a"
  relation := .equal
  left := .role operationRole.id
  right := .value operationA
}

def exactWitness : AuthoredExactTrace := {
  setup := acceptedTrace.setup
  initialState := some acceptedTrace.trace.initialState
  steps := acceptedTrace.trace.steps.map fun step => {
    selectedAction := some step.selectedAction
    modelOutcome := some step.modelOutcome
    resultingState := some step.resultingState
    observations := some step.observations
  }
}

def constrainedDeclaration : BehaviorDeclaration := {
  id := id "test.behavior.constrained"
  source
  requires := [cancellationCapability]
  roles := [operationRole]
  setup := [setupEqualsA]
  allowedActions := [requestCancel, callerClose, tick]
  requiredOccurrences := [cancelOccurrence, closeOccurrence]
  occurrenceBounds := [
    OccurrenceBound.exactly requestCancel 1,
    OccurrenceBound.exactly callerClose 1,
    OccurrenceBound.atMost tick 1
  ]
  ordering := [{ before := cancelOccurrence.id, after := closeOccurrence.id }]
  sequences := [[requestCancel, callerClose]]
}

def checkedAdmits (declaration : BehaviorDeclaration) (trace : BehaviorTrace) : Bool :=
  (checkBehavior context declaration).toOption.any fun checked => checked.admits trace

end Umpire.BehaviorTests
