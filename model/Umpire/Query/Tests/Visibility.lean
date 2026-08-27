import Umpire.Query

/-! Public-facade visibility regression for Umpire Query. -/

namespace Umpire.QueryTests

open Umpire

#check QueryCheckContext.ofTarget

/-! A backend completion signal cannot manufacture proof through the public Query surface. -/
/--
error: Unknown identifier `Umpire.finalizePlanning`
-/
#guard_msgs (error, substring := true) in
#check Umpire.finalizePlanning

end Umpire.QueryTests
