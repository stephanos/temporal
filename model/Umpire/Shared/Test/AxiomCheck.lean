import Lean.Elab.Command

/-!
# One axiom checker for every trust-bearing declaration

`assert_axioms [d₁, d₂, …] allowing [a₁, a₂, …]` replaces `#print axioms d` followed by a
`#guard_msgs` that pins its printed message: instead of a reader comparing two strings, the
command itself fails elaboration -- so `lake build` fails -- when a checked declaration's axiom
inventory escapes the allowed set. That covers a seeded `sorry` (`sorryAx` is just another axiom,
and no ordinary entry allows it), a widened trust boundary (an axiom outside `allowing`), and a
mistyped or deleted declaration (the name itself fails to resolve, located at its own identifier).

Modelled on the `machine` command's own `sorryAx` guard (`Umpire.Command.Syntax`), which already
builds on `Lean.collectAxioms`.
-/

namespace Umpire.Shared.Test

open Lean Elab Command

private def disallowedAxiomMessage (declName axiomName : Name) (allowed : List Name) : String :=
  s!"'{declName}' depends on axiom '{axiomName}', which is outside the allowed set {allowed}; \
either the checked declaration's trust boundary widened, or '{axiomName}' belongs in `allowing`"

/-- `assert_axioms [d₁, …] allowing [a₁, …]` fails when a checked declaration depends on an axiom
outside the allowed set (`sorryAx` included, so a seeded `sorry` fails unless an entry
deliberately allows it) or names a declaration that does not exist. -/
elab "assert_axioms" "[" decls:ident,+ "]" "allowing" "[" allowed:ident,+ "]" : command => do
  let allowedNames ← allowed.getElems.toList.mapM fun ref =>
    liftTermElabM (realizeGlobalConstNoOverloadWithInfo ref.raw)
  for declRef in decls.getElems do
    let declName ← liftTermElabM (realizeGlobalConstNoOverloadWithInfo declRef)
    let axioms ← liftCoreM (Lean.collectAxioms declName)
    for axiomName in axioms do
      unless allowedNames.contains axiomName do
        throwErrorAt declRef (disallowedAxiomMessage declName axiomName allowedNames)

end Umpire.Shared.Test
