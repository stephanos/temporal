import Temporal.Feature.NexusTests
import Temporal.Feature.Nexus.LifecycleTests
import Temporal.Feature.Nexus.ObservationTests
import Temporal.Feature.Nexus.OperationsTests
import Temporal.ImplementationLinkTests.Nexus
import Temporal.System
import Temporal.System.Callback.ConfigurationTests
import Temporal.System.Configuration.Tests
import Temporal.System.ConfigurationIntegrationTests
import Temporal.System.Execution.LocalProfileTests
import Temporal.System.Execution.NexusTests
import Temporal.System.Matching.ConfigurationTests
import Temporal.System.Nexus.ImplementationLinkTests

namespace TemporalModelTests

example : Temporal.System.Execution.ephemeralLocalProfile.reference.version = 2 := by
  native_decide

def compatibilityFamilies : List String :=
  Temporal.Feature.Nexus.LifecycleTests.compatibilityTargetAuthors ++
    Temporal.Feature.Nexus.OperationsTests.compatibilityConsumers

example : compatibilityFamilies = [
    "nexus-lifecycle",
    "nexus-operations-async-start",
    "nexus-operations-cancellation",
    "nexus-operations-successful-completion"
  ] := by
  rfl

end TemporalModelTests
