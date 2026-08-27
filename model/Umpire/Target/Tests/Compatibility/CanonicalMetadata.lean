import Umpire.Target.Tests.Fixtures

/-! Exact canonical metadata keeps stable `SemanticSource` provenance separate from elaboration. -/

namespace Umpire.TargetTests.Compatibility

open Umpire
open Umpire.TargetTests

private def expectedCanonicalMetadata : String :=
  include_str "Fixtures/TestTargetCanonicalMetadata.json"

example : (composeTarget testTarget).toOption.map
    (fun target => target.canonicalMetadata ++ "\n") = some expectedCanonicalMetadata := by
  native_decide

end Umpire.TargetTests.Compatibility
