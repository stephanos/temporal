import Umpire.Property.Evaluate
import Umpire.Property.Scoped
import Umpire.Scenario.Check
import Umpire.Query.Language
import Umpire.Search.Branches

/-! Semantic consumers retain their checked types without exposing Target elaboration. -/

#check Umpire.CheckedModel
#check Umpire.FinitePlanningCapability
#check Umpire.CheckedProperty
#check Umpire.CheckedScenario
#check Umpire.CheckedQuery
#check Umpire.PlanningMetadata
#check Umpire.evaluatePropertyEndpoint

/-- error: Unknown identifier -/
#guard_msgs (error, substring := true) in
#check Umpire.elabModel

/-- error: Unknown identifier -/
#guard_msgs (error, substring := true) in
#check Lean.Elab.TermElabM
