import Umpire.Target.Tests.Fixtures

/-! Exact semantic identity is independent from canonical metadata and authored layout. -/

namespace Umpire.TargetTests.Compatibility

open Umpire
open Umpire.TargetTests

private def expectedSemanticDigest : String :=
  include_str "Fixtures/TestTargetSemanticDigest.txt"

example : (composeTarget testTarget).toOption.map
    (fun target => target.semanticDigest ++ "\n") = some expectedSemanticDigest := by
  native_decide

end Umpire.TargetTests.Compatibility
