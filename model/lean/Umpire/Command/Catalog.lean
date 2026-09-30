import Lean.Elab.Command

/-!
# The catalog check an `evidence:` line asks for, without naming a platform

Evidence names recorded data that confirms a step. Most of those names need no declaration, because
the realization already has a catalog of them -- for Temporal, the generated history event kinds --
and only derived observations, such as a value read back through a call, are declared. Whether a
name is in that catalog is a question about a realization, and `Umpire` names no platform: it asks
whoever owns the catalog to answer.

The arrangement is `Umpire.Command.Schema`'s, for the same reason. The hook is an `IO.Ref` a
platform module fills in at import, so a Model file that imports the platform's catalog module gets
the check and one that does not gets none. What is stored is the name the file wrote; the catalog
itself stays on the platform's side of the seam.
-/

namespace Umpire.Command

/-- What a platform answers about one evidence name: nothing, or the message an author reads. -/
abbrev CatalogCheck := (observed : String) → Except String Unit

/-- The platform's answer, absent until a platform module installs one. -/
initialize catalogCheckRef : IO.Ref (Option CatalogCheck) ← IO.mkRef none

/-- Install the check. A platform module calls this from its own `initialize`, so importing it is
what turns the check on. -/
def installCatalogCheck (check : CatalogCheck) : IO Unit :=
  catalogCheckRef.set (some check)

/-- Run the installed check, if one is installed. No platform module imported means no catalog, so
a name that no `observation` declared is recorded as the file wrote it. -/
def checkCatalog (observed : String) : IO (Except String Unit) := do
  match ← catalogCheckRef.get with
  | some check => pure (check observed)
  | none => pure (.ok ())

end Umpire.Command
