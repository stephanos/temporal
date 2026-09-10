import Umpire.Query

/-! Public-facade visibility regression for Umpire Query. -/

namespace Umpire.QueryTests

open Umpire

#check QueryCheckContext.ofTarget
#check QuerySpec
#check QuerySpec.checked
#check QueryLimitSpec
#check QueryLocatedError
#check canonicalQueryLocatedErrorJson
#check QueryAuthoringInput
#check QueryAuthoringInput.ofSpec
#check QueryAuthoringInput.check
#check QueryAuthoringInput.check?
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
