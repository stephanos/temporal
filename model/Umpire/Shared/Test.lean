import Umpire.Shared

namespace Umpire.Shared.Test

/-! Internal construction helpers shared by Umpire concern test fixtures. -/

/-- Construct a Definition ID for a test fixture. -/
def definitionId (value : String) : DefinitionId :=
  Shared.definitionId value

/-- Construct a test source location while leaving its caller-owned path explicit. -/
def sourceLocation (path : String) : SourceLocation :=
  Shared.sourceLocation path 1 1 "lean-test"

/-- Construct test definition metadata while leaving its semantic behavior explicit. -/
def definitionMetadata
    (value : String)
    (kind : DefinitionKind)
    (source : SourceLocation)
    (canonicalBehavior : String) : DefinitionMetadata :=
  Shared.definitionMetadata (definitionId value) kind source 1 canonicalBehavior ""

end Umpire.Shared.Test
