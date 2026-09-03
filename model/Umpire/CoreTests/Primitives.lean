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
    validationErrorOf (DefinitionId.validate (definitionId "workflow.state.ready"))
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

private def transitionResult : TransitionResult Bool Nat String := {
  modelOutcome := 4
  resultingState := true
  observations := ["accepted", "persisted"]
}

example : ModelTraceStep.result false transitionResult = ({
    selectedAction := false
    modelOutcome := 4
    resultingState := true
    observations := ["accepted", "persisted"]
  } : ModelTraceStep Bool Bool Nat String) := by
  rfl

example : transitionResult.map not (fun outcome => outcome + 1) String.length = ({
    modelOutcome := 5
    resultingState := false
    observations := [8, 9]
  } : TransitionResult Bool Nat Nat) := by
  rfl

end Umpire.CoreTests
