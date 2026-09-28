import Umpire.Search.Tests.Differential
import Temporal.Feature.Nexus.Caller.Model

/-!
# The Caller campaign on both search backends, part 4 of 4

Every candidate Query the Caller's exploratory campaign plans is compared on both backends
(`Umpire.SearchTests.Differential.campaignLine`). Its 889 targets are split over four modules so
Lake builds them in parallel; this one holds targets 675 to 888 in catalog order. Two class-member
candidates are rejected at Property admission, before any search, and are listed.
-/

namespace TemporalModelTests.SearchDifferential.CallerCampaign4

open Temporal.Feature.Nexus.Caller

/--
info: "112 candidates, 112 agree, 110 on veil; nexusCallerExploration.class.temporal.nexus.caller.action.nexusProtocol.handlerReply-handlerError-false.reply.handlerError (retryable .= false): not admitted: admission at Umpire.AdmissionStage.property; nexusCallerExploration.class.temporal.nexus.caller.action.nexusProtocol.handlerReply-handlerError-true.reply.handlerError (retryable .= true): not admitted: admission at Umpire.AdmissionStage.property"
-/
#guard_msgs in
#eval Umpire.SearchTests.Differential.campaignLine nexusProtocol
  { nexusCallerExploration with targets := (nexusCallerExploration.targets.drop 675).take 225 }
  four

end TemporalModelTests.SearchDifferential.CallerCampaign4
