import Umpire.Target.Tests.Fixtures
import Umpire.Json

/-! Exact canonical metadata keeps stable `SourceLocation` provenance separate from elaboration. -/

namespace Umpire.TargetTests.Compatibility

open Umpire
open Umpire.TargetTests

private def expectedCanonicalMetadata : String :=
  include_str "Fixtures/TestTargetCanonicalMetadata.json"

example : (composeTarget testTarget).toOption.map
    (Json.prettyBytes ∘ CheckedTarget.canonicalMetadata) = some expectedCanonicalMetadata := by
  native_decide

end Umpire.TargetTests.Compatibility
