import Umpire3.Explorer
import Umpire3.Manifest

namespace Umpire3

def ExactMutationDetected (classification : Exact.Classification) : Prop :=
  classification = .traceWitness

open Lean Elab Term in
syntax (name := resolvedMutationRejection) "resolved_mutation_rejection% " ident : term

open Lean Elab Term in
@[term_elab resolvedMutationRejection] meta def elabResolvedMutationRejection : TermElab :=
    fun stx expectedType? => do
  let `(resolved_mutation_rejection% $theoremName:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo theoremName
  let declaration ← getConstInfo declarationName
  unless declaration.type.isAppOfArity ``Not 1 &&
      (declaration.type.getArg! 0).getAppFn.constName? == some ``StepSimulation do
    throwErrorAt theoremName
      "resolved mutation rejection must prove the negation of a StepSimulation"
  let axioms ← collectAxioms declarationName
  rejectForbiddenAxioms theoremName axioms
  let axioms := axioms.qsort Name.lt |>.map (Syntax.mkStrLit ∘ toString)
  let statement := reprStr declaration.type
  let expanded ← `(ResolvedProof.mk
    ProofResultClass.refinementProved
    $(Syntax.mkStrLit declarationName.toString)
    $(Syntax.mkStrLit statement)
    [$[$axioms],*])
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

open Lean Elab Term in
syntax (name := resolvedExactMutation) "resolved_exact_mutation% " ident : term

open Lean Elab Term in
@[term_elab resolvedExactMutation] meta def elabResolvedExactMutation : TermElab :=
    fun stx expectedType? => do
  let `(resolved_exact_mutation% $theoremName:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo theoremName
  let declaration ← getConstInfo declarationName
  unless declaration.type.getAppFn.constName? == some ``ExactMutationDetected do
    throwErrorAt theoremName
      "resolved exact mutation must prove ExactMutationDetected"
  let axioms ← collectAxioms declarationName
  rejectForbiddenAxioms theoremName axioms
  let axioms := axioms.qsort Name.lt |>.map (Syntax.mkStrLit ∘ toString)
  let statement := reprStr declaration.type
  let expanded ← `(ResolvedProof.mk
    ProofResultClass.traceWitness
    $(Syntax.mkStrLit declarationName.toString)
    $(Syntax.mkStrLit statement)
    [$[$axioms],*])
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

end Umpire3
