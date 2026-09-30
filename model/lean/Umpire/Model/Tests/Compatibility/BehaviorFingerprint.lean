import Umpire.Model.Tests.Fixtures

/-! The exact Behavior Fingerprint is independent from canonical metadata and authored layout. -/

namespace Umpire.ModelTests.Compatibility

open Umpire
open Umpire.ModelTests

private def expectedBehaviorFingerprint : String :=
  include_str "Fixtures/TestTargetBehaviorFingerprint.txt"

example : ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map
    (fun target => target.behaviorFingerprint.render ++ "\n") =
    some expectedBehaviorFingerprint := by
  native_decide

end Umpire.ModelTests.Compatibility
