import Temporal.Feature.NexusTests
import Temporal.Feature.Nexus.Success.RaceSyntaxTests
import Temporal.Feature.Nexus.Success.Tests
import Temporal.Feature.Nexus.Pair.Tests
import Temporal.Feature.Workflow.Start.Tests
import Temporal.Feature.Workflow.Outage.Tests
import Temporal.Feature.System.Info.Tests
import Temporal.Feature.Nexus.Caller.Tests
import Temporal.Feature.Nexus.Control.Tests
import Temporal.Feature.Nexus.Tests.Commands
import Temporal.Feature.Nexus.Tests.Machines
import Temporal.Feature.Nexus.Tests.SecondModel
import Temporal.SharedTests
import Temporal.System
import Temporal.System.Callback.ConfigurationTests
import Temporal.System.Configuration.Tests
import Temporal.System.ConfigurationIntegrationTests
import Temporal.System.Matching.ConfigurationTests
import Temporal.System.Nexus.ImplementationLinkTests
import Temporal.TestpilotTests
import Temporal.Tool.InspectTests
import TemporalModelTests.Nexus.ImplementationLink

/-! The Temporal-side compatibility families (`nexus-lifecycle`, the three `nexus-operations-*`)
retired with the hand-written Nexus models (fn-86 .5); the one family that remains, `switch`, is
pinned by `UmpireTests` until fn-86 .7 re-authors it. -/
