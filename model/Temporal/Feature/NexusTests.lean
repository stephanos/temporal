import Temporal.Feature.Nexus

/-! Facade-only smoke checks for the ordinary Nexus model. -/

namespace Temporal.Feature.NexusTests

#check Temporal.Feature.Nexus.Lifecycle.step
#check Temporal.Feature.Nexus.Lifecycle.target
#check Temporal.Feature.Nexus.Operations.AsyncStart.query
#check Temporal.Feature.Nexus.Operations.Cancellation.query
#check Temporal.Feature.Nexus.Operations.SuccessfulCompletion.query
#check Temporal.Feature.Nexus.Observation.checkedPlan

end Temporal.Feature.NexusTests
