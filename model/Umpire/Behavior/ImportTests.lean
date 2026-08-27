import Umpire.Behavior

/-! Narrow-import regression for the `Umpire.Behavior` public facade. -/

namespace Umpire.BehaviorImportTests

#check Umpire.BehaviorDeclaration
#check Umpire.BehaviorCheckContext.ofTarget

#guard_msgs (error, substring := true) in
#check Umpire.PropertyDeclaration

#guard_msgs (error, substring := true) in
#check Umpire.QueryDeclaration

end Umpire.BehaviorImportTests
