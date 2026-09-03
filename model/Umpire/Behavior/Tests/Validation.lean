import Umpire.Behavior.Tests.Fixtures
import Umpire.Shared.DefinitionGraph

/-! Authoring errors, unsatisfiability, schedule contradictions, and occurrence guards. -/

namespace Umpire.BehaviorTests

open Umpire

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

def graphA : DefinitionId := id "test.occurrence.a-tail"
def graphB : DefinitionId := id "test.occurrence.b-cycle"
def graphC : DefinitionId := id "test.occurrence.c-cycle"
def graphD : DefinitionId := id "test.occurrence.d-cycle"

def graphEdge (before after : DefinitionId) : DefinitionGraph.Edge := { before, after }

def acyclicGraphAnalysis : DefinitionGraph.Analysis :=
  DefinitionGraph.analyze [graphD, graphB, graphA, graphC] [graphEdge graphB graphC]

example : (
    (DefinitionGraph.analyze [] []).topologicalOrder,
    acyclicGraphAnalysis.canonicalNodes,
    acyclicGraphAnalysis.canonicalEdges,
    acyclicGraphAnalysis.topologicalOrder,
    acyclicGraphAnalysis.cycleEvidence
  ) = (
    some [],
    [graphA, graphB, graphC, graphD],
    [graphEdge graphB graphC],
    some [graphA, graphB, graphC, graphD],
    none
  ) := by
  native_decide

def graphFaultAnalysis : DefinitionGraph.Analysis := DefinitionGraph.analyze
  [graphB, graphA, graphB]
  [
    graphEdge graphB graphC,
    graphEdge graphA graphA,
    graphEdge graphB graphC
  ]

example : (
    graphFaultAnalysis.nodeFindings.duplicate,
    graphFaultAnalysis.edgeFindings.duplicate,
    graphFaultAnalysis.edgeFindings.self,
    graphFaultAnalysis.edgeFindings.unknownEndpoints
  ) = (
    some graphB,
    some (graphEdge graphB graphC),
    some (graphEdge graphA graphA),
    [{ edge := graphEdge graphB graphC, beforeKnown := true, afterKnown := false }]
  ) := by
  native_decide

def divergentCycleEdges : List DefinitionGraph.Edge := [
  graphEdge graphC graphA,
  graphEdge graphB graphC,
  graphEdge graphC graphD,
  graphEdge graphD graphB
]

def divergentCycleEvidence : Option (DefinitionId × DefinitionId) :=
  (DefinitionGraph.analyze [graphD, graphB, graphA, graphC] divergentCycleEdges).cycleEvidence.map
    fun evidence => (evidence.residualPredecessorWitness, evidence.canonicalWitness)

example : divergentCycleEvidence = some (graphC, graphB) := by
  native_decide

def denseNode (index : Nat) : DefinitionId :=
  id ("test.dense.node-" ++ toString index)

def denseNodes : List DefinitionId :=
  (List.range 22).map denseNode

def denseTailEdges : List DefinitionGraph.Edge :=
  (List.range 22).flatMap fun before =>
    (List.range 22).filterMap fun after =>
      if 1 < before && before < after then
        some (graphEdge (denseNode before) (denseNode after))
      else
        none

def denseCycleEdges : List DefinitionGraph.Edge :=
  [graphEdge (denseNode 0) (denseNode 1), graphEdge (denseNode 1) (denseNode 0)] ++
    denseTailEdges ++
    (List.range 20).map fun index => graphEdge (denseNode 1) (denseNode (index + 2))

example : (DefinitionGraph.analyze denseNodes denseCycleEdges).cycleEvidence.map
    (fun evidence => evidence.canonicalWitness) = some (denseNode 0) := by
  native_decide

def graphOccurrence (occurrenceId : DefinitionId) : NamedOccurrence := {
  id := occurrenceId
  action := requestCancel
}

def divergentCycleDeclaration : BehaviorDeclaration := {
  id := id "test.behavior.divergent-cycle"
  source
  allowedActions := [requestCancel]
  requiredOccurrences := [
    graphOccurrence graphD,
    graphOccurrence graphB,
    graphOccurrence graphA,
    graphOccurrence graphC
  ]
  ordering := divergentCycleEdges.map fun edge => {
    before := edge.before
    after := edge.after
  }
}

def mixedGraphAndBindingFaultDeclaration : BehaviorDeclaration := {
  divergentCycleDeclaration with
  setup := [{
    id := id "test.setup.missing-role"
    relation := .equal
    left := .role (id "test.role.missing")
    right := .value operationA
  }]
}

def multipleGraphFaultDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  ordering := [
    { before := closeOccurrence.id, after := closeOccurrence.id },
    { before := cancelOccurrence.id, after := closeOccurrence.id },
    { before := cancelOccurrence.id, after := closeOccurrence.id },
    { before := closeOccurrence.id, after := cancelOccurrence.id },
    { before := id "test.occurrence.unknown", after := cancelOccurrence.id }
  ]
}

def errorJson (result : Except BehaviorError CheckedBehavior) : Option String :=
  match result with
  | .ok _ => none
  | .error failure => some (canonicalBehaviorErrorJson failure)

example : (
    errorJson (checkBehavior context mixedGraphAndBindingFaultDeclaration),
    errorJson (checkBehavior context multipleGraphFaultDeclaration),
    errorJson (checkBehavior context divergentCycleDeclaration)
  ) = (
    some "{\"kind\":\"invalid-binding\",\"definitionId\":\"test.behavior.divergent-cycle\",\"sourcePath\":\"Umpire/Behavior/Tests.lean\",\"offendingValue\":\"test.role.missing\",\"relatedDefinitionIds\":[\"test.role.missing\"]}",
    some "{\"kind\":\"duplicate-ordering\",\"definitionId\":\"test.behavior.constrained\",\"sourcePath\":\"Umpire/Behavior/Tests.lean\",\"offendingValue\":\"test.occurrence.cancel->test.occurrence.close\",\"relatedDefinitionIds\":[\"test.occurrence.cancel\",\"test.occurrence.close\"]}",
    some "{\"kind\":\"cyclic-ordering\",\"definitionId\":\"test.behavior.divergent-cycle\",\"sourcePath\":\"Umpire/Behavior/Tests.lean\",\"offendingValue\":\"test.occurrence.c-cycle\",\"relatedDefinitionIds\":[\"test.occurrence.c-cycle\"]}"
  ) := by
  native_decide

def invalidBindingDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  setup := [{
    id := id "test.setup.missing-role"
    relation := .equal
    left := .role (id "test.role.missing")
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
    id := id "test.setup.impossible"
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
      id := id "test.setup.resource-not-a"
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
  id := id "test.behavior.exact-sequence-conflict"
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

def manyCancelOccurrences : List NamedOccurrence :=
  (List.range 15).map fun index => {
    id := id ("test.occurrence.cancel-" ++ toString index)
    action := requestCancel
  }

def countDeficitDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with requiredOccurrences := manyCancelOccurrences
}

/-- The checked authoring bound fails closed before occurrence-state exploration can explode. -/
example : actualErrorKind (checkBehavior context countDeficitDeclaration) =
    some .occurrenceLimitExceeded := by
  native_decide

end Umpire.BehaviorTests
