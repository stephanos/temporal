import Temporal.Testpilot

/-! Facade smoke checks for `Temporal.Testpilot`: since fn-86 .6 it carries the six public-facade
conformance Cases and the support the realizations call, and no hand-written Case. -/

namespace Temporal.TestpilotTests

open temporal.server.api.testpilot.v1

#check Temporal.Testpilot.CaseSupport.source
#check Temporal.Testpilot.CaseSupport.historyEvents

end Temporal.TestpilotTests
