import Temporal.System.Nexus.Tests

namespace Temporal.System.Nexus.ImplementationLinkTests

open Umpire

/-!
This focused System test root stays below Feature. The checked correspondence and its forward
witness are exercised from `Temporal.ImplementationLinkTests.Nexus`, the exact composed-test root.
-/

example : Temporal.System.Nexus.target.kernel.steps
    Temporal.System.Nexus.queuedState
    Temporal.System.Nexus.dispatchAction =
    [Temporal.System.Nexus.dispatchedResult] := by
  native_decide

end Temporal.System.Nexus.ImplementationLinkTests
