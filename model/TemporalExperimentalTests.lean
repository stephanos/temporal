import Umpire.Tests.MigrationCompatibility
import TemporalModelTests
import Temporal.Feature.Nexus.Experimental.VariationSpaceTests
import Temporal.Tool.NexusDiscoveryTests
import Temporal.Tool.InspectTests

namespace TemporalExperimentalTests

def compatibilityFamilies : List String :=
  Umpire.Tests.MigrationCompatibility.compatibilityFamilies ++
    TemporalModelTests.compatibilityFamilies

example : compatibilityFamilies = [
    "switch",
    "nexus-lifecycle",
    "nexus-operations-async-start",
    "nexus-operations-cancellation",
    "nexus-operations-successful-completion"
  ] := by
  rfl

end TemporalExperimentalTests
