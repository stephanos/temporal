import Umpire.Search
import Umpire.Search.Branches
import Umpire.Search.Admission

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
#check SearchView.retarget
#check Search.admit
#check Query.Shape
#check AdmissionDiagnostic
#check AdmissionDiagnostic.located
#check AdmittedQuery
#check AdmittedQuery.search
#check AdmittedQuery.searchWithIntent
#check AdmittedQuery.analyzeBranches
#check AdmittedQuery.withQuery
#check AdmittedQuery.retarget

/-! An admitted Query's search view is reachable only through its search operations: neither the
view nor the constructor that would pair a view with an unrelated Query is exported. -/
/--
error: Unknown constant `Umpire.AdmittedQuery.view`
-/
#guard_msgs (error, substring := true) in
#check Umpire.AdmittedQuery.view

/--
error: Unknown constant `Umpire.AdmittedQuery.mk`
-/
#guard_msgs (error, substring := true) in
#check Umpire.AdmittedQuery.mk

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
