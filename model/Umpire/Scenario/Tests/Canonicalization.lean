import Umpire.Scenario.Tests.Fixtures

/-! Canonical ordering, symmetric setup, and Behavior Fingerprint sensitivity checks. -/

namespace Umpire.ScenarioTests

open Umpire

def peerRole : Scenario.Role := {
  id := id "test.role.peer-resource"
  valueKind := .state
}

def tickOccurrence : Scenario.Step := {
  id := id "test.occurrence.tick"
  action := tick
}

def canonicalDeclaration : Scenario := {
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
    Scenario.Count.exactly requestCancel 1,
    Scenario.Count.atLeast callerClose 1,
    Scenario.Count.atMost tick 2
  ]
  ordering := [
    { before := cancelOccurrence.id, after := closeOccurrence.id },
    { before := cancelOccurrence.id, after := tickOccurrence.id }
  ]
  sequences := [[requestCancel, callerClose], [requestCancel, tick]]
  adjacencies := [[requestCancel, callerClose], [tick, callerClose]]
}

def reorderedCanonicalDeclaration : Scenario := {
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

def canonicalOf (declaration : Scenario) : Option String :=
  (Scenario.check context declaration).toOption.map canonicalScenarioJson

def fingerprintOf (declaration : Scenario) : Option BehaviorFingerprint :=
  (Scenario.check context declaration).toOption.map CheckedScenario.behaviorFingerprint

example : canonicalOf canonicalDeclaration = canonicalOf reorderedCanonicalDeclaration := by
  native_decide

def reversedSetupOperands : Scenario := {
  constrainedDeclaration with
  setup := [{ setupEqualsA with left := setupEqualsA.right, right := setupEqualsA.left }]
}

example : canonicalOf constrainedDeclaration = canonicalOf reversedSetupOperands := by
  native_decide

def setupMutation : Scenario := {
  constrainedDeclaration with
  setup := [{ setupEqualsA with relation := .different }]
}

def actionMutation : Scenario := {
  constrainedDeclaration with allowedActions := [requestCancel, callerClose, tick, retry]
}

def occurrenceMutation : Scenario := {
  constrainedDeclaration with requiredOccurrences := [cancelOccurrence, closeOccurrence, tickOccurrence]
}

def orderMutation : Scenario := {
  constrainedDeclaration with
  ordering := [{ before := closeOccurrence.id, after := cancelOccurrence.id }]
}

def boundMutation : Scenario := {
  constrainedDeclaration with
  occurrenceBounds := [
    Scenario.Count.atMost requestCancel 2,
    Scenario.Count.exactly callerClose 1,
    Scenario.Count.atMost tick 1
  ]
}

def traceMutation : Scenario := {
  constrainedDeclaration with traceExactly := some exactWitness
}

example : [
    fingerprintOf setupMutation,
    fingerprintOf actionMutation,
    fingerprintOf occurrenceMutation,
    fingerprintOf orderMutation,
    fingerprintOf boundMutation,
    fingerprintOf traceMutation
  ].all (fun fingerprint =>
    fingerprint.isSome && fingerprint != fingerprintOf constrainedDeclaration) := by
  native_decide

end Umpire.ScenarioTests
