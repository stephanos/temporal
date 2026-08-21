import Lean.Data.Json
import Lean.Elab.Term
import Lean.Util.CollectAxioms
import Temporal.Targets.NexusCancellationFencing
import Umpire3.Registration
import Umpire3.TraceReplay

namespace Umpire3

namespace TraceReplay

open Umpire3.Temporal.Targets.NexusCancellationFencing

structure Request where
  traceDigest : String
  target : String
  property : String
  world : String
  variant : String
  semanticHash : String
  actions : List String
  deriving DecidableEq, Repr

def lowerHexDigit (character : Char) : Bool :=
  ('0' ≤ character && character ≤ '9') || ('a' ≤ character && character ≤ 'f')

def hasSha256Prefix (value : String) : Bool :=
  value.toList.take 7 == "sha256:".toList

def Request.validDigest (request : Request) : Bool :=
  hasSha256Prefix request.traceDigest && request.traceDigest.length = 71 &&
    (request.traceDigest.drop 7).toString.toList.all lowerHexDigit

def Request.validSemanticHash (request : Request) : Bool :=
  hasSha256Prefix request.semanticHash && request.semanticHash.length = 71 &&
    (request.semanticHash.drop 7).toString.toList.all lowerHexDigit

def Request.matchesMutatedNexus (request : Request) : Bool :=
  request.validDigest &&
    request.validSemanticHash &&
    request.target == "nexus-cancellation" &&
    request.property == "nexus.cancellation.won-excludes-success" &&
    request.world == "smoke" &&
    request.variant == "stale-completion-guard-removed"

def checkRequest (request : Request) : Bool :=
  request.matchesMutatedNexus &&
    check mutatedFiniteView noStaleCompletion request.actions

theorem checkedRequest (request : Request) (accepted : checkRequest request = true) :
    ∃ state,
      Umpire3.Temporal.System.NexusCancellationFencing.mutatedBehavior.Reachable .smoke state ∧
        noStaleCompletion state = false := by
  simp only [checkRequest, Bool.and_eq_true] at accepted
  exact checked mutatedFiniteView noStaleCompletion request.actions accepted.2

open Lean Elab Term in
syntax (name := resolvedTraceReplayAxioms) "resolved_trace_replay_axioms% " ident : term

open Lean Elab Term in
@[term_elab resolvedTraceReplayAxioms] meta def elabResolvedTraceReplayAxioms : TermElab :=
    fun stx expectedType? => do
  let `(resolved_trace_replay_axioms% $theoremName:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo theoremName
  let axioms ← collectAxioms declarationName
  rejectForbiddenAxioms theoremName axioms
  let axioms := axioms.map toString |>.qsort (· < ·) |>.map Syntax.mkStrLit
  let expanded ← `([$[$axioms],*])
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

def checkedRequestAxioms : List String :=
  resolved_trace_replay_axioms% checkedRequest

def Request.receipt (request : Request) : Lean.Json := Lean.Json.mkObj [
  ("formatVersion", "umpire3/trace-replay-receipt/v1"),
  ("traceDigest", request.traceDigest),
  ("target", request.target),
  ("property", request.property),
  ("world", request.world),
  ("variant", request.variant),
  ("semanticHash", request.semanticHash),
  ("actions", Lean.Json.arr (request.actions.map Lean.toJson).toArray),
  ("status", "accepted"),
  ("trustBadge", "checked-certificate"),
  ("axioms", Lean.Json.arr (checkedRequestAxioms.map Lean.toJson).toArray),
]

end TraceReplay

end Umpire3
