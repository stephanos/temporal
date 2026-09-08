import Umpire.Property.Evaluation
import Umpire.Property.Scoped
import Umpire.Behavior.Language
import Umpire.Query.Language
import Umpire.Planning.CaseAnalysis

/-! Semantic consumers retain their checked types without exposing Target elaboration. -/

#check Umpire.CheckedTarget
#check Umpire.FinitePlanningCapability
#check Umpire.CheckedProperty
#check Umpire.CheckedBehavior
#check Umpire.CheckedQuery
#check Umpire.PlanningMetadata
#check Umpire.evaluatePropertyEndpoint

/-- error: Unknown identifier -/
#guard_msgs (error, substring := true) in
#check Umpire.elaborateTarget

/-- error: Unknown identifier -/
#guard_msgs (error, substring := true) in
#check Lean.Elab.TermElabM
