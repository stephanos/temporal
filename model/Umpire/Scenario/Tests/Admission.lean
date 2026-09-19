import Umpire.Scenario.Tests.Fixtures

/-! Allowed, forbidden, ordered, adjacent, action-exact, and trace-exact admission checks. -/

namespace Umpire.ScenarioTests

open Umpire

example : checkedAdmits constrainedDeclaration acceptedTrace := by native_decide
example : checkedAdmits constrainedDeclaration interleavedTrace := by native_decide
example : !checkedAdmits constrainedDeclaration reversedTrace := by native_decide
example : !checkedAdmits constrainedDeclaration repeatedCancelTrace := by native_decide
example : !checkedAdmits constrainedDeclaration otherSetupTrace := by native_decide

example : (Scenario.check context constrainedDeclaration).toOption.bind (fun behavior =>
    behavior.assignOccurrences [requestCancel, callerClose]) =
    some [some cancelOccurrence, some closeOccurrence] := by
  native_decide

example : (Scenario.check context constrainedDeclaration).toOption.bind (fun behavior =>
    behavior.assignOccurrences [callerClose, requestCancel]) = none := by
  native_decide

def adjacentDeclaration : Scenario := {
  constrainedDeclaration with adjacencies := [[requestCancel, callerClose]]
}

example : checkedAdmits adjacentDeclaration acceptedTrace := by native_decide
example : !checkedAdmits adjacentDeclaration interleavedTrace := by native_decide

def forbiddenDeclaration : Scenario := {
  constrainedDeclaration with
  allowedActions := [requestCancel, callerClose]
  forbiddenActions := [tick]
}

example : checkedAdmits forbiddenDeclaration acceptedTrace := by native_decide
example : !checkedAdmits forbiddenDeclaration interleavedTrace := by native_decide

def exactActionsDeclaration : Scenario := {
  constrainedDeclaration with actionsExactly := some [requestCancel, callerClose]
}

/-- Target-owned outcomes remain variable when only controllable actions are exact. -/
example :
    checkedAdmits exactActionsDeclaration acceptedTrace &&
    checkedAdmits exactActionsDeclaration rejectedTrace := by
  native_decide

example : !checkedAdmits exactActionsDeclaration interleavedTrace := by native_decide

def exactTraceDeclaration : Scenario := {
  constrainedDeclaration with traceExactly := some exactWitness
}

/-- A complete exact trace is a singleton, including outcomes and observations. -/
example :
    checkedAdmits exactTraceDeclaration acceptedTrace &&
    !checkedAdmits exactTraceDeclaration rejectedTrace := by
  native_decide

end Umpire.ScenarioTests
