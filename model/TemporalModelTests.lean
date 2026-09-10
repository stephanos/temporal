import Temporal.Feature.NexusTests
import Temporal.Feature.Nexus.Experimental.ExplorationTests
import Temporal.Feature.Nexus.LifecycleTests
import Temporal.Feature.Nexus.ObservationTests
import Temporal.Feature.Nexus.OperationsTests
import Temporal.Feature.Nexus.Race.Tests
import Temporal.Feature.Nexus.Success.RaceSyntaxTests
import Temporal.Feature.Nexus.Success.Tests
import Temporal.Feature.Nexus.Success.Tests.TypedNexus
import Temporal.Feature.Nexus.Success.Tests.TypedUnary
import Temporal.Feature.Nexus.Race.AuthoringTests
import Temporal.SharedTests
import Temporal.System
import Temporal.System.Callback.ConfigurationTests
import Temporal.System.Configuration.Tests
import Temporal.System.ConfigurationIntegrationTests
import Temporal.System.Matching.ConfigurationTests
import Temporal.System.Nexus.ImplementationLinkTests
import Temporal.TestpilotTests
import TemporalModelTests.Nexus.ImplementationLink

namespace TemporalModelTests

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
