import Umpire.Shared
import Umpire.Id

namespace Temporal.Shared

/-! Internal construction helpers for authored Temporal model definitions. -/

/-- Construct a Definition ID through the lower Umpire-owned seam. -/
def definitionId (value : String) : Umpire.DefinitionId :=
  Umpire.Shared.definitionId value

/-- Fix the Temporal-owned identity root while leaving the semantic family, kind, and suffix
explicit at the call site. Constructing a family performs one string concatenation and one record
assembly; `DefinitionFamily.id` performs four concatenations for each ID. There is no registry,
declaration traversal, normalization, or checker call. Independent equal-sized 1×/10× declaration
sets therefore add exactly 1×/10× construction work before unchanged language-checker work. -/
def definitionFamily (semanticFamily : String) : Umpire.DefinitionFamily := {
  root := definitionId ("temporal." ++ semanticFamily)
}

/-- Construct a source location with the common authored Temporal defaults. -/
def sourceLocation (path : String) : Umpire.SourceLocation :=
  Umpire.Shared.sourceLocation path 1 1 "lean-model"

/-- Construct definition metadata with the common authored Temporal defaults. -/
def definitionMetadata
    (id : Umpire.DefinitionId)
    (kind : Umpire.DefinitionKind)
    (source : Umpire.SourceLocation)
    (behaviorVersion : String) : Umpire.DefinitionMetadata :=
  Umpire.Shared.definitionMetadata id kind source 1 behaviorVersion ""

end Temporal.Shared
