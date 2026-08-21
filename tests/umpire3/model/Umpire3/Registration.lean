import Lean.Data.Json
import Lean.Elab.Term
import Lean.Util.CollectAxioms

namespace Umpire3

structure ResolvedTheorem where
  name : String
  statement : String
  axioms : List String
  deriving DecidableEq, Repr

structure ResolvedDeclaration where
  name : String
  signature : String
  axioms : List String
  deriving DecidableEq, Repr

def ResolvedTheorem.declaration (resolved : ResolvedTheorem) : ResolvedDeclaration where
  name := resolved.name
  signature := resolved.statement
  axioms := resolved.axioms

def rejectForbiddenAxioms (declarationSyntax : Lean.Syntax) (axioms : Array Lean.Name) :
    Lean.Elab.Term.TermElabM Unit := do
  for declarationAxiom in axioms do
    if declarationAxiom.toString = "sorryAx" || declarationAxiom.toString = "Lean.ofReduceBool" then
      throwErrorAt declarationSyntax "resolved declaration has forbidden dependency {declarationAxiom}"

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
  rejectForbiddenAxioms theoremName axioms
  let axioms := axioms.qsort Name.lt |>.map (Syntax.mkStrLit ∘ toString)
  let expanded ← `(ResolvedTheorem.mk
    $(Syntax.mkStrLit declarationName.toString)
    $(Syntax.mkStrLit (reprStr declaration.type))
    [$[$axioms],*])
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

open Lean Elab Term in
syntax (name := resolvedDeclaration) "resolved_declaration% " ident : term

open Lean Elab Term Meta in
@[term_elab resolvedDeclaration] meta def elabResolvedDeclaration : TermElab := fun stx expectedType? => do
  let `(resolved_declaration% $declarationSyntax:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo declarationSyntax
  let declaration ← getConstInfo declarationName
  let axioms ← collectAxioms declarationName
  rejectForbiddenAxioms declarationSyntax axioms
  let axioms := axioms.qsort Name.lt |>.map (Syntax.mkStrLit ∘ toString)
  let expanded ← `(ResolvedDeclaration.mk
    $(Syntax.mkStrLit declarationName.toString)
    $(Syntax.mkStrLit (reprStr declaration.type))
    [$[$axioms],*])
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

private def stringsJson (values : List String) : Lean.Json :=
  Lean.Json.arr (values.map Lean.toJson).toArray

def ResolvedDeclaration.toJson (declaration : ResolvedDeclaration) : Lean.Json := Lean.Json.mkObj [
  ("declaration", declaration.name),
  ("typeHash", "derived"),
  ("type", declaration.signature),
  ("axioms", stringsJson declaration.axioms),
  ("trustBadge", if declaration.axioms.isEmpty then "kernel" else "kernel-with-declared-axioms"),
]

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
  resolved : ResolvedTheorem

def Guarantee.ofProof (identifier : String) {claim : Prop}
    (proof : claim) (resolved : ResolvedTheorem) : Guarantee where
  identifier
  Claim := claim
  proof
  resolved

open Lean Elab Term in
syntax (name := registeredGuarantee) "registered_guarantee% " str ident : term

open Lean Elab Term in
@[term_elab registeredGuarantee] meta def elabRegisteredGuarantee : TermElab := fun stx expectedType? => do
  let `(registered_guarantee% $identifier:str $theoremName:ident) := stx
    | throwUnsupportedSyntax
  let expanded ← `(Guarantee.ofProof $identifier (@$theoremName) (resolved_theorem% $theoremName))
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

structure Requirement (provider : Guarantee) where
  consumer : String
  proof : provider.Claim

end Umpire3
