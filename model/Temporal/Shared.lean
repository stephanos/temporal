import Umpire.Shared

namespace Temporal.Shared

/-! Internal construction helpers for authored Temporal model definitions. -/

/-- Construct a Definition ID through the lower Umpire-owned seam. -/
def definitionId (value : String) : Umpire.DefinitionId :=
  Umpire.Shared.definitionId value

/-- Construct a source location with the common authored Temporal defaults. -/
def sourceLocation (path : String) : Umpire.SourceLocation :=
  Umpire.Shared.sourceLocation path 1 1 "lean-model"

/-- Construct definition metadata with the common authored Temporal defaults. -/
def definitionMetadata
    (id : Umpire.DefinitionId)
    (kind : Umpire.DefinitionKind)
    (source : Umpire.SourceLocation)
    (canonicalBehavior : String) : Umpire.DefinitionMetadata :=
  Umpire.Shared.definitionMetadata id kind source 1 canonicalBehavior ""

end Temporal.Shared
