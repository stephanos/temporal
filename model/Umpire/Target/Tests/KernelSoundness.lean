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
    declarationId := testTarget.id
    sourcePath := "Umpire/TargetTests.lean"
    offendingValue := testKernel.metadata.id.value
    relatedIdentities := [
      id "umpire.kernel-proof.initial-complete",
      id "umpire.kernel-proof.step-sound"
    ]
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
