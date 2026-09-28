import Umpire.Search.Tests.Differential
import Temporal.Feature.Nexus.Caller.Model

/-!
# The Caller campaign on both search backends, part 1 of 4

Every candidate Query the Caller's exploratory campaign plans is compared on both backends
(`Umpire.SearchTests.Differential.campaignLine`). Its 889 targets are split over four modules so
Lake builds them in parallel; this one holds targets 0 to 224 in catalog order.
-/

namespace TemporalModelTests.SearchDifferential.CallerCampaign1

open Temporal.Feature.Nexus.Caller

/-- info: "201 candidates, 201 agree, 201 on veil" -/
#guard_msgs in
#eval Umpire.SearchTests.Differential.campaignLine nexusProtocol
  { nexusCallerExploration with targets := (nexusCallerExploration.targets.drop 0).take 225 }
  four

end TemporalModelTests.SearchDifferential.CallerCampaign1
