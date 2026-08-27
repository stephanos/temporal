import Umpire.Target.Tests.Fixtures

/-! Incomplete-kernel rejection and checked-kernel proof obligations. -/

namespace Umpire.TargetTests

open Umpire

def incompleteKernelTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  kernel := .incomplete testKernel.metadata [
    id "umpire.kernel-proof.initial-complete",
    id "umpire.kernel-proof.step-sound"
  ]
}

example : (errorOf (composeTarget incompleteKernelTarget)) = some {
    kind := .incompleteKernel
    definitionId := testTarget.id
    sourcePath := "Umpire/TargetTests.lean"
    offendingValue := testKernel.metadata.id.value
    relatedDefinitionIds := [
      id "umpire.kernel-proof.initial-complete",
      id "umpire.kernel-proof.step-sound"
    ]
  } := by
  native_decide

def missingBehaviorDomainTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with kernel := .checked { testKernel with behaviorDomain := .missing }
}

def incompleteBehaviorDomainTarget :
    TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with kernel := .checked {
    testKernel with
    behaviorDomain := .incomplete [id "umpire.target-domain.action-coverage"]
  }
}

example : (errorOf (composeTarget missingBehaviorDomainTarget)).map DefinitionError.kind =
    some .missingBehaviorDomain := by
  native_decide

example : (errorOf (composeTarget incompleteBehaviorDomainTarget)) = some {
    kind := .incompleteBehaviorDomain
    definitionId := testTarget.id
    sourcePath := "Umpire/TargetTests.lean"
    offendingValue := testKernel.metadata.id.value
    relatedDefinitionIds := [id "umpire.target-domain.action-coverage"]
  } := by
  native_decide

-- An emitted step outside the authoritative relation cannot inhabit a checked kernel proof.
def outsideRelation : TransitionResult Bool Bool Bool := {
  modelOutcome := false
  resultingState := false
  observations := [true]
}

example : ¬testKernel.authoritativeStep false true outsideRelation := by
  simp [testKernel, outsideRelation, transition]

example (result : TransitionResult Bool Bool Bool)
    (member : result ∈ testKernel.steps false true) :
    testKernel.authoritativeStep false true result :=
  testKernel.stepSound false true result member

end Umpire.TargetTests
