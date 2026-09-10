import Umpire.Behavior

/-! Narrow-import regression for the `Umpire.Behavior` public facade. -/

namespace Umpire.BehaviorImportTests

#check Umpire.BehaviorDeclaration
#check Umpire.ExactSequenceSpec
#check Umpire.ExactSequenceSpec.checked
#check Umpire.BehaviorLocatedError
#check Umpire.canonicalBehaviorAuthoringDiagnosticJson
#check Umpire.BehaviorCheckContext.ofTarget
#check Umpire.RoleBinding

#guard_msgs (error, substring := true) in
#check Umpire.PropertyDeclaration

#guard_msgs (error, substring := true) in
#check Umpire.QueryDeclaration

end Umpire.BehaviorImportTests
