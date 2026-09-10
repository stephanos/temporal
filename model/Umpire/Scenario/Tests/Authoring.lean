import Umpire.Scenario.Tests.Canonicalization
import Umpire.Search.Tests.Fixtures

/-! Typed constraint lowering preserves the canonical checker and exact regression forms. -/

namespace Umpire.ScenarioTests

private def testFamily : DefinitionFamily := { root := id "test" }

private def withConstraints (constraints : List Scenario.Constraint) : Scenario :=
  Scenario.constrained
    (family := testFamily)
    (key := "constrained")
    (source := source)
    (requires := [cancellationCapability])
    (roles := [operationRole])
    (setup := [setupEqualsA])
    (constraints := constraints)

private def baseConstraints : List Scenario.Constraint := [
  .allow [requestCancel, callerClose, tick],
  .require "cancel" requestCancel,
  .require "close" callerClose,
  .bound (.exactly requestCancel 1),
  .bound (.exactly callerClose 1),
  .bound (.atMost tick 1),
  .before "cancel" "close",
  .inOrder [requestCancel, callerClose]
]

private def authored : Scenario := withConstraints baseConstraints

#guard (authored.check context).isOk
#guard canonicalOf authored == canonicalOf constrainedDeclaration
#guard fingerprintOf authored == fingerprintOf constrainedDeclaration

private def surface : Except ScenarioError CheckedScenario :=
  scenario% authored against context tracking []

#guard surface.toOption.map canonicalScenarioJson == canonicalOf constrainedDeclaration
#guard surface.toOption.map CheckedScenario.behaviorFingerprint ==
  fingerprintOf constrainedDeclaration

private def withConstraint (constraint : Scenario.Constraint) : Scenario :=
  withConstraints (baseConstraints ++ [constraint])

#guard checkedAdmits authored interleavedTrace
#guard !checkedAdmits authored reversedTrace
#guard checkedAdmits (withConstraint (.adjacent [requestCancel, callerClose])) acceptedTrace
#guard !checkedAdmits (withConstraint (.adjacent [requestCancel, callerClose])) interleavedTrace
#guard canonicalOf (withConstraint (.forbid [abort])) ==
  canonicalOf { constrainedDeclaration with forbiddenActions := [abort] }
#guard fingerprintOf (withConstraint (.adjacent [requestCancel, callerClose])) ==
  fingerprintOf { constrainedDeclaration with adjacencies := [[requestCancel, callerClose]] }

private def exactActions : Scenario :=
  { authored with actionsExactly := some [requestCancel, callerClose] }

private def exactTrace : Scenario :=
  { authored with traceExactly := some exactWitness }

#guard checkedAdmits exactActions acceptedTrace
#guard checkedAdmits exactActions rejectedTrace
#guard !checkedAdmits exactActions interleavedTrace
#guard checkedAdmits exactTrace acceptedTrace
#guard !checkedAdmits exactTrace rejectedTrace
#guard canonicalOf exactTrace ==
  canonicalOf { constrainedDeclaration with traceExactly := some exactWitness }

private def exactSequence : Scenario :=
  Scenario.exactly
    (family := testFamily)
    (key := "pinned")
    (source := source)
    (occurrences := [⟨"cancel", requestCancel⟩, ⟨"close", callerClose⟩])

#guard (scenario% (Scenario.exactly
  (family := testFamily) (key := "inline") (source := source)
  (occurrences := [⟨"cancel", requestCancel⟩])) against context tracking []).isOk
#guard (scenario% (Scenario.constrained
  (family := testFamily) (key := "inline") (source := source)
  (constraints := [.allow [requestCancel]])) against context tracking []).isOk

private def pinnedConstructor : Scenario := {
  id := id "test.behavior.pinned"
  source
  allowedActions := [callerClose, requestCancel]
  requiredOccurrences := [cancelOccurrence, closeOccurrence]
  occurrenceBounds := [.exactly callerClose 1, .exactly requestCancel 1]
  ordering := [{ before := cancelOccurrence.id, after := closeOccurrence.id }]
  actionsExactly := some [requestCancel, callerClose]
}

#guard canonicalOf exactSequence == canonicalOf pinnedConstructor
#guard (scenario% exactSequence against context tracking []).toOption.map
  CheckedScenario.behaviorFingerprint == fingerprintOf pinnedConstructor

#guard (exactSequence.check context).toOption.map (·.behaviorFingerprint.render) ==
  some "sha256:d1c4c2027d55f942a1a01ed69f9c901b076812c22aa3f77a14cef2027c613a97"

private def malformedBound : Scenario :=
  withConstraints [.bound { action := tick, minimum := 2, maximum := some 1 }]

/-- error: scenario authoring failed: {"error":{"kind":"contradictory-occurrence-bounds" -/
#guard_msgs (error, substring := true) in
#check scenario% malformedBound against context tracking [actionAnchor tick]

private def unknownAction : Scenario :=
  withConstraints [.allow [id "test.action.missing"]]

/-- error: scenario authoring failed: {"error":{"kind":"unknown-reference" -/
#guard_msgs (error, substring := true) in
#check scenario% unknownAction against context tracking []

private def wrongKind : Scenario := withConstraints [.allow [accepted]]

/-- error: scenario authoring failed: {"error":{"kind":"wrong-reference-kind" -/
#guard_msgs (error, substring := true) in
#check scenario% wrongKind against context tracking [actionAnchor accepted]

/-- error: scenario authoring failed: {"error":{"kind":"unknown-reference" -/
#guard_msgs (error, substring := true) in
#check scenario% (withConstraint (.before "cancel" "missing")) against context tracking []

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
#check scenario% authored against (0 : Nat) tracking []

private def impossible : Scenario :=
  Scenario.constrained
    (family := { root := id "planner" : DefinitionFamily })
    (key := "twice")
    (source := source)
    (roles := [{ id := SearchTests.role, valueKind := .state }])
    (constraints := [.require "first" SearchTests.request, .require "second" SearchTests.request])

private def planningContext : ScenarioCheckContext := {
  definitions := (SearchTests.target 0).definitions ++ [
    SearchTests.metadata SearchTests.request .action "request/v1"
  ]
}

#guard (impossible.check planningContext).isOk
#guard ((impossible.check planningContext).toOption.bind fun behavior =>
  (search { SearchTests.checkedQuery 0 (.witness SearchTests.property) .exhaustive
      (selectedBehavior := behavior) with limits := QueryLimits.bounded 2 2 20 }
    (SearchTests.incrementalKernel 0)).toOption.map
      (·.result.metadata.validity.satisfiability)) == some .impossible

#print axioms Scenario.constrained
#print axioms Scenario.exactly
#print axioms Scenario.checked

end Umpire.ScenarioTests
