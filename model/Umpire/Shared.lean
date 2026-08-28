import Umpire.Core

namespace Umpire.Shared

/-! Internal construction helpers for values owned by the Umpire core model. -/

/-- Construct a Definition ID while leaving its caller-owned value explicit. -/
def definitionId (value : String) : DefinitionId :=
  DefinitionId.of value

/-- Construct a source location with every source-sensitive field supplied by the caller. -/
def sourceLocation
    (path : String)
    (line column : Nat)
    (provenance : String) : SourceLocation := {
  path
  line
  column
  provenance
}

/-- Construct definition metadata without introducing shared semantic defaults. -/
def definitionMetadata
    (id : DefinitionId)
    (kind : DefinitionKind)
    (source : SourceLocation)
    (version : Nat)
    (canonicalBehavior documentation : String) : DefinitionMetadata := {
  id
  kind
  source
  version
  canonicalBehavior
  documentation
}

end Umpire.Shared
