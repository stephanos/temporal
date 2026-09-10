import Umpire.Value.Tests
import Umpire.Value.FieldTests
import Umpire.ImportTests
import Umpire.FingerprintTests
import Umpire.Model.ImportTests
import Umpire.CoreTests
import Umpire.Operation.Tests
import Umpire.ModelTests
import Umpire.Property.Tests
import Umpire.Scenario.Tests
import Umpire.Property.ImportTests
import Umpire.Scenario.ImportTests
import Umpire.Query.Tests
import Umpire.Search.Tests
import Umpire.Search.VisibilityTests
import Umpire.PromotionTests
import Umpire.Artifact.Tests.Codecs
import Umpire.Artifact.Tests.RunRecord
import Umpire.Artifact.Tests.Evidence
import Umpire.Artifact.Tests.Result
import Umpire.Artifact.Tests.Goldens
import Umpire.Artifact.Tests.Set
import Umpire.Tests.MigrationCompatibility
import Umpire.Evidence.Tests
import Umpire.Evidence.Tests.Mutations
import Umpire.Evidence.ImportTests
import Umpire.ImplementationLink.Tests
import Umpire.Variations.Tests.Compilation
import Umpire.Variations.Tests.Determinism
import Umpire.Variations.Tests.Intent
import Umpire.Variations.Tests.Lowering
import Umpire.Variations.Tests.Metadata
import Umpire.Variations.Tests.Validation
import Umpire.Exploration.Tests.Validation
import Umpire.Exploration.Tests.Candidate
import Umpire.Exploration.Tests.Selection
import Umpire.Exploration.Tests.Guided
import Umpire.Exploration.Tests.Engine
import Umpire.Exploration.Tests.Pinned
import Umpire.Exploration.Tests.Session
import Umpire.Case.CompilerTests
import Umpire.Case.Tests.FieldLowering
import Umpire.Case.Tests.Producer
import Umpire.Case.Tests.Projection
import Umpire.Case.Tests.ProjectionBoundary
import Umpire.Case.Tests.ObservedPath

namespace UmpireTests

example : Umpire.Tests.MigrationCompatibility.compatibilityFamilies = ["switch"] := by
  rfl

end UmpireTests
