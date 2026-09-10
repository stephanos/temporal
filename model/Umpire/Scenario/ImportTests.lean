import Umpire.Scenario.Elab

/-! Narrow-import regression for the `Umpire.Scenario` public facade. -/

namespace Umpire.ScenarioImportTests

#check Umpire.Scenario
#check Umpire.Scenario.exactly
#check Umpire.Scenario.checked
#check Umpire.ScenarioLocatedError
#check Umpire.canonicalScenarioLocatedErrorJson
#check Umpire.ScenarioCheckContext.ofTarget
#check Umpire.RoleBinding

#guard_msgs (error, substring := true) in
#check Umpire.Property

#guard_msgs (error, substring := true) in
#check Umpire.QueryDeclaration

end Umpire.ScenarioImportTests
