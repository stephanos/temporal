import Umpire.Model.Tests.Fixtures
import Umpire.Json

/-! Exact canonical metadata keeps stable `SourceLocation` provenance separate from elaboration. -/

namespace Umpire.ModelTests.Compatibility

open Umpire
open Umpire.ModelTests

private def expectedCanonicalMetadata : String :=
  include_str "Fixtures/TestTargetCanonicalMetadata.json"

example : ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map
    (Json.prettyBytes ∘ CheckedModel.canonicalMetadata) = some expectedCanonicalMetadata := by
  native_decide

end Umpire.ModelTests.Compatibility
