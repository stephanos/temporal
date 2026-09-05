import Umpire.ImportTests
import Umpire.FingerprintTests
import Umpire.Target.ImportTests
import Umpire.CoreTests
import Umpire.TargetTests
import Umpire.Property.Tests
import Umpire.Behavior.Tests
import Umpire.Property.ImportTests
import Umpire.Behavior.ImportTests
import Umpire.Query.Tests
import Umpire.Planning.Tests
import Umpire.Planning.VisibilityTests
import Umpire.PromotionTests
import Umpire.Artifact.Tests.Codecs
import Umpire.Artifact.Tests.Runtime
import Umpire.Artifact.Tests.Evidence
import Umpire.Artifact.Tests.Result
import Umpire.Artifact.Tests.Goldens
import Umpire.Artifact.Tests.Set
import Umpire.ExecutionHandoffTests
import Umpire.CaseTests
import Umpire.Case.CompilerTests
import Umpire.Tests.MigrationCompatibility
import Umpire.Observation.Tests
import Umpire.Observation.Tests.Mutations
import Umpire.Observation.ImportTests
import Umpire.ImplementationLink.Tests
import Umpire.Space.Tests.Compilation
import Umpire.Space.Tests.Determinism
import Umpire.Space.Tests.Intent
import Umpire.Space.Tests.Metadata
import Umpire.Space.Tests.Validation
import Umpire.Exploration.Tests.Validation
import Umpire.Exploration.Tests.Candidate
import Umpire.Exploration.Tests.Selection
import Umpire.Exploration.Tests.Guided
import Umpire.Exploration.Tests.Engine
import Umpire.Exploration.Tests.Pinned
import Umpire.Exploration.Tests.Session

namespace UmpireTests

example : Umpire.Tests.MigrationCompatibility.compatibilityFamilies = ["switch"] := by
  rfl

end UmpireTests
