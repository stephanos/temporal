import Lean.Elab.Term
import Lean.Util.CollectAxioms

namespace Umpire3

structure ResolvedTheorem where
  name : String
  statement : String
  axioms : List String
  deriving DecidableEq, Repr

open Lean Elab Term in
syntax (name := resolvedTheorem) "resolved_theorem% " ident : term

open Lean Elab Term Meta in
@[term_elab resolvedTheorem] meta def elabResolvedTheorem : TermElab := fun stx expectedType? => do
  let `(resolved_theorem% $theoremName:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo theoremName
  let declaration ← getConstInfo declarationName
  unless ← isProp declaration.type do
    throwErrorAt theoremName "registered declaration must be a theorem"
  let axioms ← collectAxioms declarationName
  let axioms := axioms.qsort Name.lt |>.map (Syntax.mkStrLit ∘ toString)
  let expanded ← `(ResolvedTheorem.mk
    $(Syntax.mkStrLit declarationName.toString)
    $(Syntax.mkStrLit (reprStr declaration.type))
    [$[$axioms],*])
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

structure RegisteredProperty where
  identifier : String
  description : String
  evidence : List String
  proof : ResolvedTheorem
  deriving DecidableEq, Repr

structure RegisteredAction (Action : Type) where
  identifier : String
  value : Action

structure Guarantee where
  identifier : String
  Claim : Prop
  proof : Claim

structure Requirement (provider : Guarantee) where
  consumer : String
  proof : provider.Claim

end Umpire3
