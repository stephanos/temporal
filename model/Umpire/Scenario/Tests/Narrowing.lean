import Umpire.Scenario.Tests.Fixtures

/-! Constraint-refinement checks over the bounded Behavior trace universe. -/

namespace Umpire.ScenarioTests

open Umpire

def broadDeclaration : Scenario := {
  id := id "test.behavior.broad"
  source
  roles := [operationRole]
}

def candidates : List Scenario.Trace := [
  acceptedTrace,
  rejectedTrace,
  interleavedTrace,
  reversedTrace,
  repeatedCancelTrace,
  otherSetupTrace
]

def declarationNarrows
    (base narrowed : Scenario) : Bool :=
  candidates.all fun candidate =>
    !checkedAdmits narrowed candidate || checkedAdmits base candidate

def requiredDeclaration : Scenario := {
  broadDeclaration with requiredOccurrences := [cancelOccurrence, closeOccurrence]
}

def narrowedDeclarations : List (Scenario × Scenario) := [
  (broadDeclaration, { broadDeclaration with setup := [setupEqualsA] }),
  (broadDeclaration, { broadDeclaration with allowedActions := [requestCancel, callerClose] }),
  (broadDeclaration, requiredDeclaration),
  (broadDeclaration, { broadDeclaration with forbiddenActions := [tick] }),
  (broadDeclaration, {
    broadDeclaration with occurrenceBounds := [Scenario.Count.atMost requestCancel 1]
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

end Umpire.ScenarioTests
