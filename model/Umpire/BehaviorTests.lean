import Umpire.Behavior

namespace Umpire.BehaviorTests

open Umpire

def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Umpire/BehaviorTests.lean"
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

def cancellationCapability := id "nexus.capability.cancellation"
def operationState := id "nexus.state.operation-id"
def phaseState := id "nexus.state.phase"
def requestCancel := id "nexus.action.request-cancel"
def callerClose := id "workflow.action.caller-close"
def tick := id "nexus.action.tick"
def retry := id "nexus.action.retry"
def abort := id "nexus.action.abort"
def noop := id "nexus.action.noop"
def accepted := id "nexus.outcome.accepted"
def rejected := id "nexus.outcome.rejected"
def cancelRequested := id "nexus.observation.cancel-requested"
def callerClosed := id "workflow.observation.caller-closed"

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
  id := id "scenario.role.operation"
  valueKind := .state
}

def peerRole : ResourceRole := {
  id := id "scenario.role.peer-operation"
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
  id := id "scenario.occurrence.cancel"
  action := requestCancel
}

def closeOccurrence : NamedOccurrence := {
  id := id "scenario.occurrence.close"
  action := callerClose
}

def tickOccurrence : NamedOccurrence := {
  id := id "scenario.occurrence.tick"
  action := tick
}

def setupEqualsA : SetupConstraint := {
  id := id "scenario.setup.operation-a"
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
  id := id "scenario.behavior.caller-closure"
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

example : checkedAdmits constrainedDeclaration acceptedTrace := by native_decide
example : checkedAdmits constrainedDeclaration interleavedTrace := by native_decide
example : !checkedAdmits constrainedDeclaration reversedTrace := by native_decide
example : !checkedAdmits constrainedDeclaration repeatedCancelTrace := by native_decide
example : !checkedAdmits constrainedDeclaration otherSetupTrace := by native_decide

def adjacentDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with adjacencies := [[requestCancel, callerClose]]
}

example : checkedAdmits adjacentDeclaration acceptedTrace := by native_decide
example : !checkedAdmits adjacentDeclaration interleavedTrace := by native_decide

def forbiddenDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  allowedActions := [requestCancel, callerClose]
  forbiddenActions := [tick]
}

example : checkedAdmits forbiddenDeclaration acceptedTrace := by native_decide
example : !checkedAdmits forbiddenDeclaration interleavedTrace := by native_decide

def exactActionsDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with actionsExactly := some [requestCancel, callerClose]
}

/-- Target-owned outcomes remain variable when only controllable actions are exact. -/
example :
    checkedAdmits exactActionsDeclaration acceptedTrace &&
    checkedAdmits exactActionsDeclaration rejectedTrace := by
  native_decide

example : !checkedAdmits exactActionsDeclaration interleavedTrace := by native_decide

def exactTraceDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with traceExactly := some exactWitness
}

/-- A complete exact trace is a singleton, including outcomes and observations. -/
example :
    checkedAdmits exactTraceDeclaration acceptedTrace &&
    !checkedAdmits exactTraceDeclaration rejectedTrace := by
  native_decide

def actualErrorKind : Except BehaviorError CheckedBehavior → Option BehaviorErrorKind
  | .ok _ => none
  | .error error => some error.kind

def cyclicDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  ordering := [
    { before := closeOccurrence.id, after := cancelOccurrence.id },
    { before := cancelOccurrence.id, after := closeOccurrence.id }
  ]
}

def invalidBindingDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  setup := [{
    id := id "scenario.setup.missing-role"
    relation := .equal
    left := .role (id "scenario.role.missing")
    right := .value operationA
  }]
}

def contradictoryCountDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  occurrenceBounds := [{ action := requestCancel, minimum := 2, maximum := some 1 }]
}

def forbiddenRequiredDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  allowedActions := [callerClose, tick]
  forbiddenActions := [requestCancel]
}

def incompleteExactDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  traceExactly := some {
    exactWitness with
    steps := exactWitness.steps.modifyHead fun step => { step with observations := none }
  }
}

example : [
    actualErrorKind (checkBehavior context cyclicDeclaration),
    actualErrorKind (checkBehavior context invalidBindingDeclaration),
    actualErrorKind (checkBehavior context contradictoryCountDeclaration),
    actualErrorKind (checkBehavior context forbiddenRequiredDeclaration),
    actualErrorKind (checkBehavior context incompleteExactDeclaration)
  ] = [
    some .cyclicOrdering,
    some .invalidBinding,
    some .contradictoryOccurrenceBounds,
    some .forbiddenRequired,
    some .incompleteExactTrace
  ] := by
  native_decide

def canonicalError (declaration : BehaviorDeclaration) : Option String :=
  match checkBehavior context declaration with
  | .ok _ => none
  | .error error => some (canonicalBehaviorErrorJson error)

example : canonicalError cyclicDeclaration = canonicalError {
    cyclicDeclaration with ordering := cyclicDeclaration.ordering.reverse
  } := by
  native_decide

/-- An empty semantic space is a checked result, distinct from invalid authoring. -/
def unsatisfiableDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  setup := [{
    id := id "scenario.setup.impossible"
    relation := .different
    left := .role operationRole.id
    right := .role operationRole.id
  }]
}

example : (checkBehavior context unsatisfiableDeclaration).toOption.map
    CheckedBehavior.isUnsatisfiable = some true := by
  native_decide

example : !checkedAdmits unsatisfiableDeclaration acceptedTrace := by native_decide

def pairedSetupConflict : BehaviorDeclaration := {
  constrainedDeclaration with
  setup := [
    setupEqualsA,
    {
      id := id "scenario.setup.operation-not-a"
      relation := .different
      left := .role operationRole.id
      right := .value operationA
    }
  ]
}

example : (checkBehavior context pairedSetupConflict).toOption.map
    CheckedBehavior.isUnsatisfiable = some true := by
  native_decide

def exactSequenceConflict : BehaviorDeclaration := {
  id := id "scenario.behavior.exact-sequence-conflict"
  source
  roles := [operationRole]
  actionsExactly := some [requestCancel]
  sequences := [[callerClose]]
}

def exactAdjacencyConflict : BehaviorDeclaration := {
  exactSequenceConflict with
  sequences := []
  adjacencies := [[requestCancel, callerClose]]
}

def exactOrderingConflict : BehaviorDeclaration := {
  exactSequenceConflict with
  requiredOccurrences := [cancelOccurrence, closeOccurrence]
  ordering := [{ before := cancelOccurrence.id, after := closeOccurrence.id }]
  actionsExactly := some [callerClose, requestCancel]
  sequences := []
}

def exactTraceSequenceConflict : BehaviorDeclaration := {
  constrainedDeclaration with
  traceExactly := some exactWitness
  sequences := [[callerClose, requestCancel]]
}

/-! Mechanically contradictory exact schedules and traces fail during Behavior checking. -/
example : [
    actualErrorKind (checkBehavior context exactSequenceConflict),
    actualErrorKind (checkBehavior context exactAdjacencyConflict),
    actualErrorKind (checkBehavior context exactOrderingConflict),
    actualErrorKind (checkBehavior context exactTraceSequenceConflict)
  ] = List.replicate 4 (some .contradictoryConstraint) := by
  native_decide

def canonicalDeclaration : BehaviorDeclaration := {
  id := id "scenario.behavior.canonical"
  source
  requires := [cancellationCapability]
  roles := [operationRole, peerRole]
  setup := [
    setupEqualsA,
    {
      id := id "scenario.setup.peer-differs"
      relation := .different
      left := .role peerRole.id
      right := .value operationA
    }
  ]
  allowedActions := [requestCancel, callerClose, tick, retry]
  requiredOccurrences := [cancelOccurrence, closeOccurrence, tickOccurrence]
  forbiddenActions := [abort, noop]
  occurrenceBounds := [
    OccurrenceBound.exactly requestCancel 1,
    OccurrenceBound.atLeast callerClose 1,
    OccurrenceBound.atMost tick 2
  ]
  ordering := [
    { before := cancelOccurrence.id, after := closeOccurrence.id },
    { before := cancelOccurrence.id, after := tickOccurrence.id }
  ]
  sequences := [[requestCancel, callerClose], [requestCancel, tick]]
  adjacencies := [[requestCancel, callerClose], [tick, callerClose]]
}

def reorderedCanonicalDeclaration : BehaviorDeclaration := {
  canonicalDeclaration with
  roles := canonicalDeclaration.roles.reverse
  setup := canonicalDeclaration.setup.reverse
  allowedActions := canonicalDeclaration.allowedActions.reverse
  requiredOccurrences := canonicalDeclaration.requiredOccurrences.reverse
  forbiddenActions := canonicalDeclaration.forbiddenActions.reverse
  occurrenceBounds := canonicalDeclaration.occurrenceBounds.reverse
  ordering := canonicalDeclaration.ordering.reverse
  sequences := canonicalDeclaration.sequences.reverse
  adjacencies := canonicalDeclaration.adjacencies.reverse
}

def canonicalOf (declaration : BehaviorDeclaration) : Option String :=
  (checkBehavior context declaration).toOption.map canonicalBehaviorJson

def digestOf (declaration : BehaviorDeclaration) : Option String :=
  (checkBehavior context declaration).toOption.map CheckedBehavior.semanticDigest

example : canonicalOf canonicalDeclaration = canonicalOf canonicalDeclaration := by native_decide
example : canonicalOf canonicalDeclaration = canonicalOf reorderedCanonicalDeclaration := by
  native_decide

def reversedSetupOperands : BehaviorDeclaration := {
  constrainedDeclaration with
  setup := [{ setupEqualsA with left := setupEqualsA.right, right := setupEqualsA.left }]
}

example : canonicalOf constrainedDeclaration = canonicalOf reversedSetupOperands := by
  native_decide

def setupMutation : BehaviorDeclaration := {
  constrainedDeclaration with
  setup := [{ setupEqualsA with relation := .different }]
}

def actionMutation : BehaviorDeclaration := {
  constrainedDeclaration with allowedActions := [requestCancel, callerClose, tick, retry]
}

def occurrenceMutation : BehaviorDeclaration := {
  constrainedDeclaration with requiredOccurrences := [cancelOccurrence, closeOccurrence, tickOccurrence]
}

def orderMutation : BehaviorDeclaration := {
  constrainedDeclaration with
  ordering := [{ before := closeOccurrence.id, after := cancelOccurrence.id }]
}

def boundMutation : BehaviorDeclaration := {
  constrainedDeclaration with
  occurrenceBounds := [
    OccurrenceBound.atMost requestCancel 2,
    OccurrenceBound.exactly callerClose 1,
    OccurrenceBound.atMost tick 1
  ]
}

def traceMutation : BehaviorDeclaration := {
  constrainedDeclaration with traceExactly := some exactWitness
}

example : [
    digestOf setupMutation,
    digestOf actionMutation,
    digestOf occurrenceMutation,
    digestOf orderMutation,
    digestOf boundMutation,
    digestOf traceMutation
  ].all (fun digest => digest.isSome && digest != digestOf constrainedDeclaration) := by
  native_decide

def broadDeclaration : BehaviorDeclaration := {
  id := id "scenario.behavior.broad"
  source
  roles := [operationRole]
}

def candidates : List BehaviorTrace := [
  acceptedTrace,
  rejectedTrace,
  interleavedTrace,
  reversedTrace,
  repeatedCancelTrace,
  otherSetupTrace
]

def declarationNarrows
    (base narrowed : BehaviorDeclaration) : Bool :=
  candidates.all fun candidate =>
    !checkedAdmits narrowed candidate || checkedAdmits base candidate

def requiredDeclaration : BehaviorDeclaration := {
  broadDeclaration with requiredOccurrences := [cancelOccurrence, closeOccurrence]
}

def narrowedDeclarations : List (BehaviorDeclaration × BehaviorDeclaration) := [
  (broadDeclaration, { broadDeclaration with setup := [setupEqualsA] }),
  (broadDeclaration, { broadDeclaration with allowedActions := [requestCancel, callerClose] }),
  (broadDeclaration, requiredDeclaration),
  (broadDeclaration, { broadDeclaration with forbiddenActions := [tick] }),
  (broadDeclaration, {
    broadDeclaration with occurrenceBounds := [OccurrenceBound.atMost requestCancel 1]
  }),
  (requiredDeclaration, {
    requiredDeclaration with
    ordering := [{ before := cancelOccurrence.id, after := closeOccurrence.id }]
  }),
  (broadDeclaration, { broadDeclaration with sequences := [[requestCancel, callerClose]] }),
  (broadDeclaration, { broadDeclaration with adjacencies := [[requestCancel, callerClose]] }),
  (broadDeclaration, {
    broadDeclaration with actionsExactly := some [requestCancel, callerClose]
  }),
  (broadDeclaration, { broadDeclaration with traceExactly := some exactWitness })
]

/-- Every supported constraint preserves or narrows membership over the bounded fixture universe. -/
example : narrowedDeclarations.all fun pair => declarationNarrows pair.1 pair.2 := by
  native_decide

def manyCancelOccurrences : List NamedOccurrence :=
  (List.range 15).map fun index => {
    id := id ("scenario.occurrence.cancel-" ++ toString index)
    action := requestCancel
  }

def countDeficitDeclaration : BehaviorDeclaration := {
  broadDeclaration with requiredOccurrences := manyCancelOccurrences
}

/-- The checked authoring bound fails closed before occurrence-state exploration can explode. -/
example : actualErrorKind (checkBehavior context countDeficitDeclaration) =
    some .occurrenceLimitExceeded := by
  native_decide

end Umpire.BehaviorTests
