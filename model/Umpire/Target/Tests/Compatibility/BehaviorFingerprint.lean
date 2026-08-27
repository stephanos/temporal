import Umpire.Target.Tests.Fixtures

/-! The exact Behavior Fingerprint is independent from canonical metadata and authored layout. -/

namespace Umpire.TargetTests.Compatibility

open Umpire
open Umpire.TargetTests

private def expectedBehaviorFingerprint : String :=
  include_str "Fixtures/TestTargetBehaviorFingerprint.txt"

example : (composeTarget testTarget).toOption.map
    (fun target => target.behaviorFingerprint.render ++ "\n") =
    some expectedBehaviorFingerprint := by
  native_decide

end Umpire.TargetTests.Compatibility
