import Umpire.Target.Language

/-! Stable, source-independent identity construction shared by the authoring owners. -/

namespace Umpire

/-- An explicit semantic identity root. Language kind and local key remain visible at each use. -/
structure DefinitionFamily where
  root : DefinitionId
  deriving BEq, DecidableEq, Repr

namespace DefinitionFamily

def id (family : DefinitionFamily) (kind key : String) : DefinitionId :=
  DefinitionId.of (family.root.value ++ "." ++ kind ++ "." ++ key)

end DefinitionFamily

end Umpire
