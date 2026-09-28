import Umpire.Search.Tests.Differential
import Temporal.Feature.Nexus.Caller.Model

/-!
# The Caller campaign on both search backends, part 2 of 4

Every candidate Query the Caller's exploratory campaign plans is compared on both backends
(`Umpire.SearchTests.Differential.campaignLine`). Its 889 targets are split over four modules so
Lake builds them in parallel; this one holds targets 225 to 449 in catalog order.
-/

namespace TemporalModelTests.SearchDifferential.CallerCampaign2

open Temporal.Feature.Nexus.Caller

/-- info: "88 candidates, 88 agree, 88 on veil" -/
#guard_msgs in
#eval Umpire.SearchTests.Differential.campaignLine nexusProtocol
  { nexusCallerExploration with targets := (nexusCallerExploration.targets.drop 225).take 225 }
  four

end TemporalModelTests.SearchDifferential.CallerCampaign2
