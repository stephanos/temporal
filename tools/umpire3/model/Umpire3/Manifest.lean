import Lean.Data.Json
import Lean.Elab.Term
import Lean.Util.CollectAxioms
import Umpire3.Refinement
import Umpire3.Registration

namespace Umpire3

def formatVersion : String := "umpire3/v2"

def leanVersion : String := Lean.versionString

def proofManifestFormatVersion : String := "umpire3/proof-manifest/v3"

structure SemanticProofDependency where
  identifier : String
  statementHash : String

inductive ProofResultClass where
  | traceWitness
  | invariantProved
  | temporalProved
  | refinementProved
  deriving DecidableEq, Repr

private def ProofResultClass.toString : ProofResultClass → String
  | .traceWitness => "trace-witness"
  | .invariantProved => "invariant-proved"
  | .temporalProved => "temporal-proved"
  | .refinementProved => "refinement-proved"

structure ResolvedProof where
  resultClass : ProofResultClass
  theoremName : String
  statement : String
  axioms : List String
  deriving DecidableEq, Repr

structure SemanticProofManifest where
  identifier : String
  proof : ResolvedProof
  assumptions : List SemanticProofDependency := []

open Lean Elab Term in
syntax (name := resolvedRefinement) "resolved_refinement% " ident : term

open Lean Elab Term in
@[term_elab resolvedRefinement] meta def elabResolvedRefinement : TermElab := fun stx expectedType? => do
  let `(resolved_refinement% $theoremName:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo theoremName
  let declaration ← getConstInfo declarationName
  unless declaration.type.getAppFn.constName? == some ``Refinement do
    throwErrorAt theoremName "resolved refinement manifest declaration must have type Refinement"
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
syntax (name := resolvedSimulation) "resolved_simulation% " ident : term

open Lean Elab Term in
@[term_elab resolvedSimulation] meta def elabResolvedSimulation : TermElab := fun stx expectedType? => do
  let `(resolved_simulation% $theoremName:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo theoremName
  let declaration ← getConstInfo declarationName
  unless declaration.type.getAppFn.constName? == some ``SafetySimulation do
    throwErrorAt theoremName "resolved simulation manifest declaration must have type SafetySimulation"
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

private def SemanticProofDependency.toJson (dependency : SemanticProofDependency) : Lean.Json :=
  Lean.Json.mkObj [
    ("identifier", dependency.identifier),
    ("statementHash", dependency.statementHash),
  ]

def SemanticProofManifest.toJson (manifest : SemanticProofManifest) : Lean.Json := Lean.Json.mkObj [
  ("formatVersion", proofManifestFormatVersion),
  ("identifier", manifest.identifier),
  ("theorem", manifest.proof.theoremName),
  ("statement", manifest.proof.statement),
  ("resultClass", manifest.proof.resultClass.toString),
  ("axioms", Lean.Json.arr (manifest.proof.axioms.map Lean.toJson).toArray),
  ("leanVersion", leanVersion),
  ("assumptions", Lean.Json.arr (manifest.assumptions.map SemanticProofDependency.toJson).toArray),
]

def SemanticProofManifest.json (manifest : SemanticProofManifest) : String :=
  manifest.toJson.compress

end Umpire3
