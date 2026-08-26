import Umpire.Behavior

/-! Shared semantic vocabulary, traces, and helpers for the Behavior concern tests. -/

namespace Umpire.BehaviorTests

open Umpire

def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Umpire/Behavior/Tests.lean"
  line := 1
  column := 1
  provenance := "lean-test"
}

def metadata (value : String) (kind : DeclarationKind) : DeclarationMetadata := {
  id := id value
  kind
  source
  contractDigest := value ++ "/v1"
}

def cancellationCapability := id "test.capability.cancellation"
def operationState := id "test.state.resource-id"
def phaseState := id "test.state.phase"
def requestCancel := id "test.action.request-cancel"
def callerClose := id "test.action.close"
def tick := id "test.action.tick"
def retry := id "test.action.retry"
def abort := id "test.action.abort"
def noop := id "test.action.noop"
def accepted := id "test.outcome.accepted"
def rejected := id "test.outcome.rejected"
def cancelRequested := id "test.observation.cancel-requested"
def callerClosed := id "test.observation.closed"

def context : BehaviorCheckContext := {
  declarations := [
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

def operationA : SemanticValue := { identity := operationState, value := "operation-a" }
def operationB : SemanticValue := { identity := operationState, value := "operation-b" }
def initial : SemanticValue := { identity := phaseState, value := "started" }
def cancelling : SemanticValue := { identity := phaseState, value := "cancelling" }
def closed : SemanticValue := { identity := phaseState, value := "closed" }

def actionValue (action : DeclarationId) : SemanticValue := { identity := action, value := action.value }
def outcomeValue (outcome : DeclarationId) : SemanticValue := { identity := outcome, value := outcome.value }
def observationValue (observation : DeclarationId) : SemanticValue := {
  identity := observation
  value := observation.value
}

def cancelStep (outcome : DeclarationId) :
    SemanticTraceStep SemanticValue SemanticValue SemanticValue SemanticValue := {
  selectedAction := actionValue requestCancel
  modelOutcome := outcomeValue outcome
  resultingState := cancelling
  observations := [observationValue cancelRequested]
}

def closeStep : SemanticTraceStep SemanticValue SemanticValue SemanticValue SemanticValue := {
  selectedAction := actionValue callerClose
  modelOutcome := outcomeValue accepted
  resultingState := closed
  observations := [observationValue callerClosed]
}

def tickStep : SemanticTraceStep SemanticValue SemanticValue SemanticValue SemanticValue := {
  selectedAction := actionValue tick
  modelOutcome := outcomeValue accepted
  resultingState := cancelling
  observations := []
}

def traceWith
    (setupValue : SemanticValue)
    (steps : List (SemanticTraceStep SemanticValue SemanticValue SemanticValue SemanticValue)) :
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
