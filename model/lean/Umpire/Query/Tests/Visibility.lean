import Umpire.Query.Elab

/-! Public-facade visibility regression for Umpire Query. -/

namespace Umpire.QueryTests

open Umpire

#check QueryCheckContext.ofTarget
#check Query
#check Query.check
#check Query.checked
#check Query.error?
#check Limits
#check Limits.bounded
#check QueryLocatedError
#check canonicalQueryLocatedErrorJson
#check KnownGap
#check KnownGapSet
#check KnownGapSet.empty
#check KnownGapSet.checkCanonical
#check KnownGapError

/-! Case analysis remains owned by Planning and does not create a Query-to-Planning cycle. -/
/--
error: Unknown identifier `Umpire.analyzeBranches`
-/
#guard_msgs (error, substring := true) in
#check Umpire.analyzeBranches

/-! A backend completion signal cannot manufacture proof through the public Query surface. -/
/--
error: Unknown identifier `Umpire.finalizePlanning`
-/
#guard_msgs (error, substring := true) in
#check Umpire.finalizePlanning

end Umpire.QueryTests
