import Umpire.Target.Tests.Fixtures

/-! Exact typed failure compatibility for the low-level checker seam. -/

namespace Umpire.TargetTests.Compatibility

open Umpire
open Umpire.TargetTests

example : errorOf (composeTarget conflictingTarget) = some {
    kind := .conflictingProviders
    definitionId := DefinitionId.of "test.relation.shared"
    sourcePath := "Test/PrimarySemantic.lean"
    offendingValue := "test.relation.shared"
    relatedDefinitionIds := [
      DefinitionId.of "test.provider.primary",
      DefinitionId.of "test.provider.secondary"
    ]
  } := by
  native_decide

end Umpire.TargetTests.Compatibility
