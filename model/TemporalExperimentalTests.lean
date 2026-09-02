import Umpire.Tests.MigrationCompatibility
import Temporal.NexusExecutionIntegrationTests
import TemporalModelTests
import Temporal.Feature.Nexus.Experimental.CallerClosureTests
import Temporal.Feature.Nexus.Experimental.CallerClosureFaultTests
import Temporal.Feature.Nexus.Experimental.VariationSpaceTests
import Temporal.Tool.NexusDiscoveryTests
import Temporal.Tool.InspectTests
import Temporal.Tool.GenerateTestsTests
import Temporal.Tool.RunEvaluationTests

namespace TemporalExperimentalTests

def compatibilityFamilies : List String :=
  Umpire.Tests.MigrationCompatibility.compatibilityFamilies ++
    TemporalModelTests.compatibilityFamilies ++
    Temporal.Feature.Nexus.Experimental.CallerClosureTests.compatibilityTargetAuthors

example : compatibilityFamilies = [
    "switch",
    "nexus-lifecycle",
    "nexus-operations-async-start",
    "nexus-operations-cancellation",
    "nexus-operations-successful-completion",
    "nexus-experimental-caller-closure"
  ] := by
  rfl

end TemporalExperimentalTests
