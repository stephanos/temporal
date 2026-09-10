import Umpire.Scenario.Tests.Fixtures
import Umpire.Shared.DefinitionGraph

/-! Authoring errors, unsatisfiability, schedule contradictions, and occurrence guards. -/

namespace Umpire.ScenarioTests

open Umpire

def actualErrorKind : Except ScenarioError CheckedScenario → Option ScenarioErrorKind
  | .ok _ => none
  | .error error => some error.kind

def cyclicDeclaration : Scenario := {
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

def graphOccurrence (occurrenceId : DefinitionId) : Scenario.Step := {
  id := occurrenceId
  action := requestCancel
}

def divergentCycleDeclaration : Scenario := {
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

def mixedGraphAndBindingFaultDeclaration : Scenario := {
  divergentCycleDeclaration with
  setup := [{
    id := id "test.setup.missing-role"
    relation := .equal
    left := .role (id "test.role.missing")
    right := .value operationA
  }]
}

def multipleGraphFaultDeclaration : Scenario := {
  constrainedDeclaration with
  ordering := [
    { before := closeOccurrence.id, after := closeOccurrence.id },
    { before := cancelOccurrence.id, after := closeOccurrence.id },
    { before := cancelOccurrence.id, after := closeOccurrence.id },
    { before := closeOccurrence.id, after := cancelOccurrence.id },
    { before := id "test.occurrence.unknown", after := cancelOccurrence.id }
  ]
}

def errorJson (result : Except ScenarioError CheckedScenario) : Option String :=
  match result with
  | .ok _ => none
  | .error failure => some (canonicalScenarioErrorJson failure)

example : (
    errorJson (Scenario.check context mixedGraphAndBindingFaultDeclaration),
    errorJson (Scenario.check context multipleGraphFaultDeclaration),
    errorJson (Scenario.check context divergentCycleDeclaration)
  ) = (
    some "{\"kind\":\"invalid-binding\",\"definitionId\":\"test.behavior.divergent-cycle\",\"sourcePath\":\"Umpire/Scenario/Tests.lean\",\"offendingValue\":\"test.role.missing\",\"relatedDefinitionIds\":[\"test.role.missing\"]}",
    some "{\"kind\":\"duplicate-ordering\",\"definitionId\":\"test.behavior.constrained\",\"sourcePath\":\"Umpire/Scenario/Tests.lean\",\"offendingValue\":\"test.occurrence.cancel->test.occurrence.close\",\"relatedDefinitionIds\":[\"test.occurrence.cancel\",\"test.occurrence.close\"]}",
    some "{\"kind\":\"cyclic-ordering\",\"definitionId\":\"test.behavior.divergent-cycle\",\"sourcePath\":\"Umpire/Scenario/Tests.lean\",\"offendingValue\":\"test.occurrence.c-cycle\",\"relatedDefinitionIds\":[\"test.occurrence.c-cycle\"]}"
  ) := by
  native_decide

def invalidBindingDeclaration : Scenario := {
  constrainedDeclaration with
  setup := [{
    id := id "test.setup.missing-role"
    relation := .equal
    left := .role (id "test.role.missing")
    right := .value operationA
  }]
}

def contradictoryCountDeclaration : Scenario := {
  constrainedDeclaration with
  occurrenceBounds := [{ action := requestCancel, minimum := 2, maximum := some 1 }]
}

def forbiddenRequiredDeclaration : Scenario := {
  constrainedDeclaration with
  allowedActions := [callerClose, tick]
  forbiddenActions := [requestCancel]
}

def incompleteExactDeclaration : Scenario := {
  constrainedDeclaration with
  traceExactly := some {
    exactWitness with
    steps := exactWitness.steps.modifyHead fun step => { step with observations := none }
  }
}

example : [
    actualErrorKind (Scenario.check context cyclicDeclaration),
    actualErrorKind (Scenario.check context invalidBindingDeclaration),
    actualErrorKind (Scenario.check context contradictoryCountDeclaration),
    actualErrorKind (Scenario.check context forbiddenRequiredDeclaration),
    actualErrorKind (Scenario.check context incompleteExactDeclaration)
  ] = [
    some .cyclicOrdering,
    some .invalidBinding,
    some .contradictoryOccurrenceBounds,
    some .forbiddenRequired,
    some .incompleteExactTrace
  ] := by
  native_decide

def canonicalError (declaration : Scenario) : Option String :=
  match Scenario.check context declaration with
  | .ok _ => none
  | .error error => some (canonicalScenarioErrorJson error)

example : canonicalError cyclicDeclaration = canonicalError {
    cyclicDeclaration with ordering := cyclicDeclaration.ordering.reverse
  } := by
  native_decide

/-- An empty semantic space is a checked result, distinct from invalid authoring. -/
def unsatisfiableDeclaration : Scenario := {
  constrainedDeclaration with
  setup := [{
    id := id "test.setup.impossible"
    relation := .different
    left := .role operationRole.id
    right := .role operationRole.id
  }]
}

example : (Scenario.check context unsatisfiableDeclaration).toOption.map
    CheckedScenario.isUnsatisfiable = some true := by
  native_decide

example : !checkedAdmits unsatisfiableDeclaration acceptedTrace := by native_decide

def pairedSetupConflict : Scenario := {
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

example : (Scenario.check context pairedSetupConflict).toOption.map
    CheckedScenario.isUnsatisfiable = some true := by
  native_decide

def exactSequenceConflict : Scenario := {
  id := id "test.behavior.exact-sequence-conflict"
  source
  roles := [operationRole]
  actionsExactly := some [requestCancel]
  sequences := [[callerClose]]
}

def exactAdjacencyConflict : Scenario := {
  exactSequenceConflict with
  sequences := []
  adjacencies := [[requestCancel, callerClose]]
}

def exactOrderingConflict : Scenario := {
  exactSequenceConflict with
  requiredOccurrences := [cancelOccurrence, closeOccurrence]
  ordering := [{ before := cancelOccurrence.id, after := closeOccurrence.id }]
  actionsExactly := some [callerClose, requestCancel]
  sequences := []
}

def exactTraceSequenceConflict : Scenario := {
  constrainedDeclaration with
  traceExactly := some exactWitness
  sequences := [[callerClose, requestCancel]]
}

/-! Mechanically contradictory exact schedules and traces fail during Behavior checking. -/
example : [
    actualErrorKind (Scenario.check context exactSequenceConflict),
    actualErrorKind (Scenario.check context exactAdjacencyConflict),
    actualErrorKind (Scenario.check context exactOrderingConflict),
    actualErrorKind (Scenario.check context exactTraceSequenceConflict)
  ] = List.replicate 4 (some .contradictoryConstraint) := by
  native_decide

def manyCancelOccurrences : List Scenario.Step :=
  (List.range 15).map fun index => {
    id := id ("test.occurrence.cancel-" ++ toString index)
    action := requestCancel
  }

def countDeficitDeclaration : Scenario := {
  constrainedDeclaration with requiredOccurrences := manyCancelOccurrences
}

/-- The checked authoring bound fails closed before occurrence-state exploration can explode. -/
example : actualErrorKind (Scenario.check context countDeficitDeclaration) =
    some .occurrenceLimitExceeded := by
  native_decide

end Umpire.ScenarioTests
