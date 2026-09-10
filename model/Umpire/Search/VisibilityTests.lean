import Umpire.Search
import Umpire.Search.Branches

/-! Visibility regression for the `Umpire.Search` public facade. -/

namespace Umpire.PlanningVisibilityTests

#check SearchView.ofCheckedQuery?
#check SearchView.ofCheckedQuery
#check FiniteSearchAdmissionError
#check FinitePlannerAdmissionErrorKind
#check traverseBoundedCandidates
#check analyzeBranches
#check BranchAnalysisResult
#check OverlapStatus
#check OverlapConflictEvidence
#check OverlapModelIncompatibility
#check OverlapUnsupportedFormulaClass
#check composeSearchKnownGaps
#check artifactOfSelection
#check search

/-! Importing Search does not expose its private completion finalizer. -/
/--
error: Unknown identifier `Umpire.finalizePlanning`
-/
#guard_msgs (error, substring := true) in
#check Umpire.finalizePlanning

/-! Importing Search does not expose the private PlanningResult constructor. -/
/--
error: Unknown constant `Umpire.PlanningResult.mk`
-/
#guard_msgs (error, substring := true) in
#check Umpire.PlanningResult.mk

end Umpire.PlanningVisibilityTests
