import Temporal.Feature.Nexus

/-! Facade-only smoke checks for the Nexus Model. Every name below is reachable through the public
Nexus facade alone, so this module is the ordinary reader path rather than an internal, runtime or
verification surface. -/

namespace Temporal.Feature.NexusTests

open Umpire
open Temporal.Feature.Nexus.Caller

#check Temporal.Feature.Nexus.Caller.nexusProduct
#check Temporal.Feature.Nexus.Caller.nexusProtocol
#check Temporal.Feature.Nexus.Caller.completionSucceeds
#check Temporal.Feature.Nexus.Caller.asyncThenSucceeded
#check Temporal.Feature.Nexus.Caller.asyncCompletion
#check Temporal.Feature.Nexus.Caller.nexusCallerTests
#check Temporal.Feature.Nexus.Caller.nexusCallerCases.asyncCompletion
#check Temporal.Feature.Nexus.Pair.pair
#check Temporal.Feature.Nexus.Pair.bothComplete
#check Temporal.Feature.Nexus.Pair.nexusPairCases.bothComplete

/-- The reader order the facade documents. -/
def readerOrder : List String := [
  "Caller.Model",
  "Pair.Model",
  "System.Nexus.Core",
  "System.Nexus.ImplementationLink"
]

/-- Each functional Query of the caller Model finds its claim on its path through the facade. -/
example : [
    (match asyncCompletion with
      | .ok checked => checked.run.result.outcome.name
      | .error _ => "admission failed"),
    (match syncCompletion with
      | .ok checked => checked.run.result.outcome.name
      | .error _ => "admission failed"),
    (match terminalHolds with
      | .ok checked => checked.run.result.outcome.name
      | .error _ => "admission failed")
  ] = ["found", "found", "verified-within-limits"] := by
  native_decide

end Temporal.Feature.NexusTests
