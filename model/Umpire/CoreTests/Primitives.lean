import Umpire.Core

/-! Focused checks for shared Definition identity and semantic construction primitives. -/

namespace Umpire.CoreTests

open Umpire

private def definitionId (value : String) : DefinitionId := DefinitionId.of value

private def validationErrorOf : Except DefinitionId.ValidationError Unit →
    Option DefinitionId.ValidationError
  | .error failure => some failure
  | .ok () => none

example : DefinitionId.canonicalSet [
    definitionId "zeta.state.ready",
    definitionId "alpha.action.start",
    definitionId "zeta.state.ready",
    definitionId "alpha.action.start",
    definitionId "beta.outcome.started"
  ] = [
    definitionId "alpha.action.start",
    definitionId "beta.outcome.started",
    definitionId "zeta.state.ready"
  ] := by
  native_decide

example : (
    DefinitionId.firstDuplicate [
      definitionId "zeta.state.ready",
      definitionId "beta.outcome.started",
      definitionId "zeta.state.ready",
      definitionId "alpha.action.start",
      definitionId "beta.outcome.started"
    ],
    DefinitionId.firstDuplicate [
      definitionId "beta.outcome.started",
      definitionId "alpha.action.start"
    ]
  ) = (some (definitionId "beta.outcome.started"), none) := by
  native_decide

example : [
    validationErrorOf (DefinitionId.validate (definitionId "")),
    validationErrorOf (DefinitionId.validate (definitionId "state")),
    validationErrorOf (DefinitionId.validate (definitionId "gamma.state.ready"))
  ] = [
    some .empty,
    some .malformed,
    none
  ] := by
  native_decide

example : [
    SourceLocation.displayPath { path := "Umpire/CoreTests/Primitives.lean" },
    SourceLocation.displayPath { path := "" }
  ] = ["Umpire/CoreTests/Primitives.lean", "<unknown>"] := by
  decide

private def step : Step Bool Nat String := {
  outcome := 4
  state := true
  facts := ["accepted", "persisted"]
}

example : ModelTraceStep.result false step = ({
    selectedAction := false
    outcome := 4
    state := true
    facts := ["accepted", "persisted"]
  } : ModelTraceStep Bool Bool Nat String) := by
  rfl

example : step.map not (fun outcome => outcome + 1) String.length = ({
    outcome := 5
    state := false
    facts := [8, 9]
  } : Step Bool Nat Nat) := by
  rfl

end Umpire.CoreTests
