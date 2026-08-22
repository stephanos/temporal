import Lean.Data.Json
import Lean.Elab.Term
import Lean.Util.CollectAxioms
import Temporal.Families.WorkflowProgress.Targets.TemporalSystem
import Umpire3.Registration

namespace Umpire3.TemporalLassoReplay

structure Request where
  lassoDigest : String
  target : String
  property : String
  world : String
  variant : String
  semanticHash : String
  states : List String
  actions : List String
  loopStart : Nat
  deriving DecidableEq, Repr

private def lowerHexDigit (character : Char) : Bool :=
  ('0' ≤ character && character ≤ '9') || ('a' ≤ character && character ≤ 'f')

private def validHash (value : String) : Bool :=
  value.startsWith "sha256:" && value.length = 71 &&
    (value.drop 7).toString.toList.all lowerHexDigit

def checkRequest (request : Request) : Bool :=
  validHash request.lassoDigest &&
    validHash request.semanticHash &&
    request.target == "foundation-delivery-safety" &&
    request.property == "entity.progress" &&
    request.world == "smoke" &&
    request.variant == "delivery-fairness-removed" &&
    request.states == ["unavailable", "ready"] &&
    request.actions == ["recover-owner", ""] &&
    request.loopStart == 1

theorem checkedRequest (request : Request) (_accepted : checkRequest request = true) :
    ¬LeadsTo
      Umpire3.Temporal.Mechanisms.TaskDeliveryProgress.Unfinished
      Umpire3.Temporal.Mechanisms.TaskDeliveryProgress.Completed
      Umpire3.Temporal.Mechanisms.TaskDeliveryProgress.mutatedLasso.state := by
  exact Umpire3.Temporal.Mechanisms.TaskDeliveryProgress.mutatedLassoViolatesProgress

open Lean Elab Term in
syntax (name := resolvedTemporalLassoAxioms) "resolved_temporal_lasso_axioms% " ident : term

open Lean Elab Term in
@[term_elab resolvedTemporalLassoAxioms] meta def elabResolvedTemporalLassoAxioms : TermElab :=
    fun stx expectedType? => do
  let `(resolved_temporal_lasso_axioms% $theoremName:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo theoremName
  let axioms ← collectAxioms declarationName
  rejectForbiddenAxioms theoremName axioms
  let axioms := axioms.map toString |>.qsort (· < ·) |>.map Syntax.mkStrLit
  let expanded ← `([$[$axioms],*])
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

def checkedRequestAxioms : List String :=
  resolved_temporal_lasso_axioms% checkedRequest

def Request.receipt (request : Request) : Lean.Json := Lean.Json.mkObj [
  ("formatVersion", "umpire3/temporal-lasso-replay-receipt/v1"),
  ("lassoDigest", request.lassoDigest),
  ("target", request.target),
  ("property", request.property),
  ("world", request.world),
  ("variant", request.variant),
  ("semanticHash", request.semanticHash),
  ("lasso", Lean.Json.mkObj [
    ("states", Lean.Json.arr (request.states.map Lean.toJson).toArray),
    ("actions", Lean.Json.arr (request.actions.map Lean.toJson).toArray),
    ("loopStart", Lean.toJson request.loopStart),
  ]),
  ("status", "accepted"),
  ("trustBadge", "checked-certificate"),
  ("axioms", Lean.Json.arr (checkedRequestAxioms.map Lean.toJson).toArray),
]

end Umpire3.TemporalLassoReplay
