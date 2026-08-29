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
import Umpire.Artifact.Tests.Codecs
import Umpire.Artifact.Tests.Runtime
import Umpire.Artifact.Tests.Evidence
import Umpire.Artifact.Tests.Result
import Umpire.ExecutionHandoffTests
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

namespace UmpireTests

example : Umpire.Tests.MigrationCompatibility.compatibilityFamilies = ["switch"] := by
  rfl

end UmpireTests
