import Umpire.Property

/-! Narrow-import regression for the `Umpire.Property` public facade. -/

namespace Umpire.PropertyImportTests

#check Umpire.PropertyDeclaration

#guard_msgs (error, substring := true) in
#check Umpire.BehaviorDeclaration

#guard_msgs (error, substring := true) in
#check Umpire.QueryDeclaration

end Umpire.PropertyImportTests
