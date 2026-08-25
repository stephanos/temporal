import Umpire.Planning

/-! Visibility regression for the `Umpire.Planning` public facade. -/

namespace Umpire.PlanningVisibilityTests

/-! Importing Planning does not expose its private completion finalizer. -/
/--
error: Unknown identifier `Umpire.finalizePlanning`
-/
#guard_msgs (error, substring := true) in
#check Umpire.finalizePlanning

/-! Importing Planning does not expose the private PlanningResult constructor. -/
/--
error: Unknown constant `Umpire.PlanningResult.mk`
-/
#guard_msgs (error, substring := true) in
#check Umpire.PlanningResult.mk

end Umpire.PlanningVisibilityTests
