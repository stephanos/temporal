import Lean.Data.Json
import Lean.Elab.Term
import Lean.Util.CollectAxioms
import Umpire3.Veil.Binding

namespace Umpire3.Veil.JobReceipt

structure Evidence where
  binding : BindingExport
  invariantAxioms : List String

private def validDigest (digest : String) : Bool :=
  digest.startsWith "sha256:" && digest.length = 71

def Evidence.valid (evidence : Evidence) : Bool :=
  (evidence.binding.binding.trustMode = "reconstructed" ||
      evidence.binding.binding.trustMode = "trusted") &&
    evidence.invariantAxioms.Pairwise (· < ·) &&
    (evidence.binding.binding.trustMode != "reconstructed" ||
      !evidence.invariantAxioms.contains "sorryAx")

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

private def trustOption (trustMode : String) : String :=
  if trustMode = "trusted" then "smt-trust=true" else "smt-trust=false"

private def invariantTrustBadge (trustMode : String) : String :=
  if trustMode = "trusted" then "trusted-solver" else "reconstructed-solver-proof"

private def receipt (evidence : Evidence) (semanticHash job status trustBadge : String)
    (axioms : List String) (depth : Option Nat) : Lean.Json :=
  let fields := [
    ("formatVersion", Lean.Json.str "umpire3/veil-job-receipt/v1"),
    ("backendRevision", Lean.Json.str "300c305e945750ab3fb62de4a79c23161b24da39"),
    ("binding", evidence.binding.compiledJson semanticHash),
    ("job", Lean.Json.str job),
    ("status", Lean.Json.str status),
    ("trustBadge", Lean.Json.str trustBadge),
    ("options", Lean.Json.arr #["grind+smt", "sequential",
      trustOption evidence.binding.binding.trustMode]),
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
  | [semanticHash, "symbolic-trace"] =>
      if !validDigest semanticHash then
        IO.eprintln "first-order semantic hash is invalid"
        return 2
      IO.println (receipt evidence semanticHash "symbolic-trace" "bounded-safe" "trusted-solver" []
        (some evidence.binding.binding.view.artifact.bounds.symbolicDepth)).compress
      return 0
  | [semanticHash, "invariant"] =>
      if !validDigest semanticHash then
        IO.eprintln "first-order semantic hash is invalid"
        return 2
      IO.println (receipt evidence semanticHash "invariant" "goals-closed"
        (invariantTrustBadge evidence.binding.binding.trustMode) evidence.invariantAxioms none).compress
      return 0
  | _ =>
      IO.eprintln "semantic hash and symbolic-trace or invariant job are required"
      return 2

end Umpire3.Veil.JobReceipt
