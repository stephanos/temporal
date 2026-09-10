import Umpire.Model.Tests.Fixtures

/-! Incomplete-kernel rejection and checked-kernel proof obligations. -/

namespace Umpire.ModelTests

open Umpire

def incompleteKernelTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  machine := .incomplete testKernel.metadata [
    id "umpire.kernel-proof.initial-complete",
    id "umpire.kernel-proof.step-sound"
  ]
}

example : (errorOf ((checkModel (DraftModel.make incompleteKernelTarget) |>.mapError LocatedError.error))) = some {
    kind := .incompleteMachine
    definitionId := testTarget.id
    sourcePath := "Umpire/ModelTests.lean"
    offendingValue := testKernel.metadata.id.value
    relatedDefinitionIds := [
      id "umpire.kernel-proof.initial-complete",
      id "umpire.kernel-proof.step-sound"
    ]
  } := by
  native_decide

def missingBehaviorDomainTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with machine := .checked { testKernel with vocabulary := .missing }
}

def incompleteBehaviorDomainTarget :
    ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with machine := .checked {
    testKernel with
    vocabulary := .incomplete [id "umpire.target-domain.action-coverage"]
  }
}

example : (errorOf ((checkModel (DraftModel.make missingBehaviorDomainTarget) |>.mapError LocatedError.error))).map DefinitionError.kind =
    some .missingVocabulary := by
  native_decide

example : (errorOf ((checkModel (DraftModel.make incompleteBehaviorDomainTarget) |>.mapError LocatedError.error))) = some {
    kind := .incompleteVocabulary
    definitionId := testTarget.id
    sourcePath := "Umpire/ModelTests.lean"
    offendingValue := testKernel.metadata.id.value
    relatedDefinitionIds := [id "umpire.target-domain.action-coverage"]
  } := by
  native_decide

-- An emitted step outside the authoritative relation cannot inhabit a checked kernel proof.
def outsideRelation : Step Bool Bool Bool := {
  outcome := false
  state := false
  facts := [true]
}

example : ¬testKernel.authoritativeStep false true outsideRelation := by
  simp [testKernel, outsideRelation, transition]

example (result : Step Bool Bool Bool)
    (member : result ∈ testKernel.steps false true) :
    testKernel.authoritativeStep false true result :=
  testKernel.stepSound false true result member

/-- Exhaustive domains cannot omit an admitted setup or an action merely because it has no steps. -/
example
    (domain : Vocabulary
      (fun _ : Unit => True)
      (fun _ : Bool => True)
      (fun _ : Bool => True)
      (fun _ : Bool => True)
      (fun _ : Bool => True)
      testKernel.initialStates
      (fun _ _ => ([] : List (Step Bool Bool Bool)))) :
    () ∈ domain.setups ∧ true ∈ domain.actions :=
  ⟨domain.setupComplete () trivial, domain.actionComplete true trivial⟩

end Umpire.ModelTests
