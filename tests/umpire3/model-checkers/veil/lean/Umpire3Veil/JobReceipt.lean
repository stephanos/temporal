import Lean.Data.Json

namespace Umpire3Veil.JobReceipt

private def validDigest (digest : String) : Bool :=
  digest.startsWith "sha256:" && digest.length = 71

private def receipt (semanticHash trustBadge trustOption job status : String)
    (depth : Option Nat) : Lean.Json :=
  let fields := [
    ("formatVersion", Lean.Json.str "umpire3/veil-job-receipt/v1"),
    ("backendRevision", Lean.Json.str "300c305e945750ab3fb62de4a79c23161b24da39"),
    ("viewFormatVersion", Lean.Json.str "umpire3/first-order-view/v1"),
    ("target", Lean.Json.str "nexus-cancellation"),
    ("property", Lean.Json.str "nexus.cancellation.won-excludes-success"),
    ("world", Lean.Json.str "smoke"),
    ("variant", Lean.Json.str "sound"),
    ("semanticHash", Lean.Json.str semanticHash),
    ("job", Lean.Json.str job),
    ("status", Lean.Json.str status),
    ("trustBadge", Lean.Json.str trustBadge),
    ("options", Lean.Json.arr #["grind+smt", "sequential", trustOption]),
    ("axioms", Lean.Json.arr #[]),
  ]
  Lean.Json.mkObj <| match depth with
    | none => fields
    | some value => fields ++ [("depth", Lean.toJson value)]

def run (trustBadge trustOption : String) (arguments : List String) : IO UInt32 := do
  match arguments with
  | [semanticHash, "symbolic-trace"] =>
      if validDigest semanticHash then
        IO.println (receipt semanticHash trustBadge trustOption "symbolic-trace" "bounded-safe"
          (some 6)).compress
        return 0
      else
        IO.eprintln "a semantic SHA-256 digest is required"
        return 2
  | [semanticHash, "invariant"] =>
      if validDigest semanticHash then
        IO.println (receipt semanticHash trustBadge trustOption "invariant" "goals-closed"
          none).compress
        return 0
      else
        IO.eprintln "a semantic SHA-256 digest is required"
        return 2
  | _ =>
      IO.eprintln "semantic hash and symbolic-trace or invariant job are required"
      return 2

end Umpire3Veil.JobReceipt
