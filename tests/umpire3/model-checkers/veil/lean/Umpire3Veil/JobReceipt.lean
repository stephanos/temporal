import Lean.Data.Json
import Lean.Elab.Term
import Lean.Util.CollectAxioms

namespace Umpire3Veil.JobReceipt

inductive TrustMode where
  | reconstructed
  | trusted
  deriving DecidableEq, Repr

structure Evidence where
  semanticHash : String
  generatedModelHash : String
  invariantAxioms : List String
  trustMode : TrustMode
  deriving DecidableEq, Repr

private def validDigest (digest : String) : Bool :=
  digest.startsWith "sha256:" && digest.length = 71

def Evidence.valid (evidence : Evidence) : Bool :=
  validDigest evidence.semanticHash && validDigest evidence.generatedModelHash &&
    evidence.invariantAxioms.Pairwise (· < ·) &&
    (evidence.trustMode != .reconstructed || !evidence.invariantAxioms.contains "sorryAx")

open Lean Elab Term in
syntax (name := resolvedVeilAxioms) "resolved_veil_axioms% " "[" ident,* "]" : term

open Lean Elab Term in
@[term_elab resolvedVeilAxioms] meta def elabResolvedVeilAxioms : TermElab :=
    fun stx expectedType? => do
  let `(resolved_veil_axioms% [$[$theoremNames:ident],*]) := stx
    | throwUnsupportedSyntax
  let mut allAxioms : Array Name := #[]
  for theoremName in theoremNames do
    let declarationName ← realizeGlobalConstNoOverloadWithInfo theoremName
    allAxioms := (← collectAxioms declarationName) ++ allAxioms
  let axioms := allAxioms.toList.map toString |>.mergeSort (fun a b => a < b) |>.eraseDups
  let axioms := (axioms.map Syntax.mkStrLit).toArray
  let expanded ← `([$[$axioms],*])
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

private def trustOption : TrustMode → String
  | .reconstructed => "smt-trust=false"
  | .trusted => "smt-trust=true"

private def invariantTrustBadge : TrustMode → String
  | .reconstructed => "reconstructed-solver-proof"
  | .trusted => "trusted-solver"

private def receipt (evidence : Evidence) (job status trustBadge : String)
    (axioms : List String) (depth : Option Nat) : Lean.Json :=
  let fields := [
    ("formatVersion", Lean.Json.str "umpire3/veil-job-receipt/v1"),
    ("backendRevision", Lean.Json.str "300c305e945750ab3fb62de4a79c23161b24da39"),
    ("viewFormatVersion", Lean.Json.str "umpire3/first-order-view/v1"),
    ("target", Lean.Json.str "nexus-cancellation"),
    ("property", Lean.Json.str "nexus.cancellation.won-excludes-success"),
    ("world", Lean.Json.str "smoke"),
    ("variant", Lean.Json.str "sound"),
    ("semanticHash", Lean.Json.str evidence.semanticHash),
    ("generatedModelHash", Lean.Json.str evidence.generatedModelHash),
    ("job", Lean.Json.str job),
    ("status", Lean.Json.str status),
    ("trustBadge", Lean.Json.str trustBadge),
    ("options", Lean.Json.arr #["grind+smt", "sequential", trustOption evidence.trustMode]),
    ("axioms", Lean.Json.arr (axioms.map Lean.toJson).toArray),
  ]
  Lean.Json.mkObj <| match depth with
    | none => fields
    | some value => fields ++ [("depth", Lean.toJson value)]

def run (evidence : Evidence) (arguments : List String) : IO UInt32 := do
  if !evidence.valid then
    IO.eprintln "compiled Veil proof evidence is invalid"
    return 2
  match arguments with
  | ["symbolic-trace"] =>
      IO.println (receipt evidence "symbolic-trace" "bounded-safe" "trusted-solver" []
        (some 6)).compress
      return 0
  | ["invariant"] =>
      IO.println (receipt evidence "invariant" "goals-closed"
        (invariantTrustBadge evidence.trustMode) evidence.invariantAxioms none).compress
      return 0
  | _ =>
      IO.eprintln "symbolic-trace or invariant job is required"
      return 2

end Umpire3Veil.JobReceipt
