import Umpire.Model.Tests.Fixtures

/-! Exact typed failure compatibility for the low-level checker seam. -/

namespace Umpire.ModelTests.Compatibility

open Umpire
open Umpire.ModelTests

example : errorOf ((checkModel (DraftModel.make conflictingTarget) |>.mapError LocatedError.error)) = some {
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

end Umpire.ModelTests.Compatibility
