import Umpire.Behavior.Tests.Canonicalization
import Umpire.Planning.Tests.Fixtures

/-! Typed constraint lowering preserves the canonical checker and exact regression forms. -/

namespace Umpire.BehaviorTests

private def authored : BehaviorSpec := {
  family := { root := id "test" }
  key := "constrained"
  source
  requires := [cancellationCapability]
  roles := [operationRole]
  setup := [setupEqualsA]
  constraints := [
    .allow [requestCancel, callerClose, tick],
    .require "cancel" requestCancel,
    .require "close" callerClose,
    .bound (.exactly requestCancel 1),
    .bound (.exactly callerClose 1),
    .bound (.atMost tick 1),
    .before "cancel" "close",
    .inOrder [requestCancel, callerClose]
  ]
}

#guard (authored.check context).isOk
#guard canonicalOf authored.declaration == canonicalOf constrainedDeclaration
#guard fingerprintOf authored.declaration == fingerprintOf constrainedDeclaration

private def surface : Except BehaviorError CheckedBehavior :=
  behavior% authored against context tracking []

#guard surface.toOption.map canonicalBehaviorJson == canonicalOf constrainedDeclaration
#guard surface.toOption.map CheckedBehavior.behaviorFingerprint ==
  fingerprintOf constrainedDeclaration

private def withConstraint (constraint : BehaviorConstraint) : BehaviorSpec :=
  { authored with constraints := authored.constraints ++ [constraint] }

#guard checkedAdmits authored.declaration interleavedTrace
#guard !checkedAdmits authored.declaration reversedTrace
#guard checkedAdmits (withConstraint (.adjacent [requestCancel, callerClose])).declaration acceptedTrace
#guard !checkedAdmits (withConstraint (.adjacent [requestCancel, callerClose])).declaration interleavedTrace
#guard canonicalOf (withConstraint (.forbid [abort])).declaration ==
  canonicalOf { constrainedDeclaration with forbiddenActions := [abort] }
#guard fingerprintOf (withConstraint (.adjacent [requestCancel, callerClose])).declaration ==
  fingerprintOf { constrainedDeclaration with adjacencies := [[requestCancel, callerClose]] }

private def exactActions : BehaviorSpec :=
  { authored with actionsExactly := some [requestCancel, callerClose] }

private def exactTrace : BehaviorSpec :=
  { authored with traceExactly := some exactWitness }

#guard checkedAdmits exactActions.declaration acceptedTrace
#guard checkedAdmits exactActions.declaration rejectedTrace
#guard !checkedAdmits exactActions.declaration interleavedTrace
#guard checkedAdmits exactTrace.declaration acceptedTrace
#guard !checkedAdmits exactTrace.declaration rejectedTrace
#guard canonicalOf exactTrace.declaration ==
  canonicalOf { constrainedDeclaration with traceExactly := some exactWitness }

private def exactSequence : ExactSequenceSpec := {
  family := { root := id "test" }
  key := "pinned"
  source
  occurrences := [⟨"cancel", requestCancel⟩, ⟨"close", callerClose⟩]
}

#guard (behavior% {
  family := { root := id "test" }
  key := "inline"
  source
  occurrences := [⟨"cancel", requestCancel⟩]
} against context tracking []).isOk
#guard (behavior% {
  family := { root := id "test" }
  key := "inline"
  source
  constraints := [.allow [requestCancel]]
} against context tracking []).isOk

private def pinnedConstructor : BehaviorDeclaration := {
  id := id "test.behavior.pinned"
  source
  allowedActions := [callerClose, requestCancel]
  requiredOccurrences := [cancelOccurrence, closeOccurrence]
  occurrenceBounds := [.exactly callerClose 1, .exactly requestCancel 1]
  ordering := [{ before := cancelOccurrence.id, after := closeOccurrence.id }]
  actionsExactly := some [requestCancel, callerClose]
}

#guard canonicalOf exactSequence.declaration == canonicalOf pinnedConstructor
#guard (behavior% exactSequence against context tracking []).toOption.map
  CheckedBehavior.behaviorFingerprint == fingerprintOf pinnedConstructor

#guard (exactSequence.check context).toOption.map (·.behaviorFingerprint.render) ==
  some "sha256:d1c4c2027d55f942a1a01ed69f9c901b076812c22aa3f77a14cef2027c613a97"

private def malformedBound : BehaviorSpec := {
  authored with constraints := [.bound { action := tick, minimum := 2, maximum := some 1 }]
}

/-- error: behavior authoring failed: {"error":{"kind":"contradictory-occurrence-bounds" -/
#guard_msgs (error, substring := true) in
#check behavior% malformedBound against context tracking [actionAnchor tick]

private def unknownAction : BehaviorSpec := {
  authored with constraints := [.allow [id "test.action.missing"]]
}

/-- error: behavior authoring failed: {"error":{"kind":"unknown-reference" -/
#guard_msgs (error, substring := true) in
#check behavior% unknownAction against context tracking []

private def wrongKind : BehaviorSpec := {
  authored with constraints := [.allow [accepted]]
}

/-- error: behavior authoring failed: {"error":{"kind":"wrong-reference-kind" -/
#guard_msgs (error, substring := true) in
#check behavior% wrongKind against context tracking [actionAnchor accepted]

/-- error: behavior authoring failed: {"error":{"kind":"unknown-reference" -/
#guard_msgs (error, substring := true) in
#check behavior% (withConstraint (.before "cancel" "missing")) against context tracking []

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
#check behavior% authored against (0 : Nat) tracking []

private def impossible : BehaviorSpec := {
  family := { root := id "planner" }
  key := "twice"
  source
  roles := [{ id := PlanningTests.role, valueKind := .state }]
  constraints := [.require "first" PlanningTests.request, .require "second" PlanningTests.request]
}

private def planningContext : BehaviorCheckContext := {
  definitions := (PlanningTests.target 0).definitions ++ [
    PlanningTests.metadata PlanningTests.request .action "request/v1"
  ]
}

#guard (impossible.check planningContext).isOk
#guard ((impossible.check planningContext).toOption.bind fun behavior =>
  (plan { PlanningTests.checkedQuery 0 (.witness PlanningTests.property) .exhaustive
      (selectedBehavior := behavior) with limits := QueryLimits.bounded 2 2 20 }
    (PlanningTests.incrementalKernel 0)).toOption.map
      (·.result.metadata.validity.satisfiability)) == some .impossible

#print axioms BehaviorSpec.declaration
#print axioms BehaviorSpec.checked

end Umpire.BehaviorTests
