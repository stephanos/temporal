import Umpire.Query

/-! Public-facade visibility regression for Umpire Query. -/

namespace Umpire.QueryTests

open Umpire

#check QueryCheckContext.ofTarget
#check QuerySpec
#check QuerySpec.checked
#check QueryLimitSpec
#check QueryAuthoringDiagnostic
#check canonicalQueryAuthoringDiagnosticJson
#check QueryAuthoringInput
#check QueryAuthoringInput.ofSpec
#check QueryAuthoringInput.check
#check QueryAuthoringInput.check?

/-! Case analysis remains owned by Planning and does not create a Query-to-Planning cycle. -/
/--
error: Unknown identifier `Umpire.analyzeCases`
-/
#guard_msgs (error, substring := true) in
#check Umpire.analyzeCases

/-! A backend completion signal cannot manufacture proof through the public Query surface. -/
/--
error: Unknown identifier `Umpire.finalizePlanning`
-/
#guard_msgs (error, substring := true) in
#check Umpire.finalizePlanning

end Umpire.QueryTests
