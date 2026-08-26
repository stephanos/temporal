import Umpire.Behavior.Tests.Fixtures

/-! Canonical ordering, symmetric setup, and semantic-digest sensitivity checks. -/

namespace Umpire.BehaviorTests

open Umpire

def peerRole : ResourceRole := {
  id := id "test.role.peer-resource"
  valueKind := .state
}

def tickOccurrence : NamedOccurrence := {
  id := id "test.occurrence.tick"
  action := tick
}

def canonicalDeclaration : BehaviorDeclaration := {
  id := id "test.behavior.canonical"
  source
  requires := [cancellationCapability]
  roles := [operationRole, peerRole]
  setup := [
    setupEqualsA,
    {
      id := id "test.setup.peer-differs"
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

end Umpire.BehaviorTests
