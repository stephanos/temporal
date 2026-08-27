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
import Umpire.Tests.MigrationCompatibility
import Umpire.Observation.Tests
import Umpire.Observation.Tests.Mutations
import Umpire.Observation.ImportTests
import Umpire.ImplementationLink.Tests

namespace UmpireTests

example : Umpire.Tests.MigrationCompatibility.compatibilityFamilies = ["switch"] := by
  rfl

end UmpireTests
