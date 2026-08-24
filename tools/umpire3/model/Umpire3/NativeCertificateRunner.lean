import Lean.Data.Json
import Lean.Elab.Term
import Lean.Util.CollectAxioms
import Temporal.Families.NexusCancellation.Targets.Finite
import Umpire3.Certificate
import Umpire3.Generated.NexusCertificateBinding
import Umpire3.ReplicatedSafety
import Umpire3.Registration

namespace Umpire3.NativeCertificate

open Umpire3.Temporal.System.NexusCancellationFencing
open Umpire3.Temporal.Targets.NexusCancellationFencing

structure WireNode where
  parent : Option Nat
  action : Option Action
  state : State
  deriving DecidableEq, Repr

structure Request where
  certificateDigest : String
  viewDigest : String
  target : String
  property : String
  world : String
  variant : String
  semanticHash : String
  replicas : Nat
  expandedStates : Nat
  representativeStates : Nat
  nodes : List WireNode
  deriving DecidableEq, Repr

private def lowerHexDigit (character : Char) : Bool :=
  ('0' ≤ character && character ≤ '9') || ('a' ≤ character && character ≤ 'f')

private def validHash (value : String) : Bool :=
  value.startsWith "sha256:" && value.length = 71 &&
    (value.drop 7).toString.toList.all lowerHexDigit

def Request.identityMatches (request : Request) : Bool :=
  validHash request.certificateDigest &&
    request.viewDigest == Umpire3.Generated.NexusCertificateBinding.viewDigest &&
    request.target == "nexus-cancellation" &&
    request.property == "nexus.cancellation.won-excludes-success" &&
    request.world == "smoke" &&
    request.variant == "sound" &&
    request.semanticHash == Umpire3.Generated.NexusCertificateBinding.semanticHash &&
    0 < request.replicas && request.replicas ≤ 10 &&
    request.representativeStates == request.nodes.length &&
    request.expandedStates == request.representativeStates * request.replicas

private def expandNodes (nodes : List WireNode) : Option (List (Exact.Node behavior .smoke)) :=
  let rec visit (remaining : List WireNode) (expanded : List (Exact.Node behavior .smoke)) :=
    match remaining with
    | [] => some expanded
    | wire :: tail => do
        let node ← match wire.parent, wire.action with
          | none, none =>
              if wire.state = initial then
                some ({ initial := initial, actions := [], state := wire.state } :
                  Exact.Node behavior .smoke)
              else none
          | some parentIndex, some action => do
              let parent ← expanded[parentIndex]?
              let actions := parent.actions ++ [action]
              some ({ initial := parent.initial, actions := actions, state := wire.state } :
                Exact.Node behavior .smoke)
          | _, _ => none
        visit tail (expanded ++ [node])
  visit nodes []

def Request.certificate? (request : Request) : Option (Exact.Certificate behavior .smoke) := do
  let nodes ← expandNodes request.nodes
  some { nodes }

def checkRequest (request : Request) : Bool :=
  request.identityMatches && match request.certificate? with
    | none => false
    | some certificate => certificate.check soundFiniteView noStaleCompletion .exhaustive

theorem checkedRequest (request : Request) (accepted : checkRequest request = true) :
    Safety
      ((replicatedBehavior behavior).at { core := .smoke, replicas := request.replicas })
      (fun state => noStaleCompletion state.2 = true) := by
  simp only [checkRequest, Bool.and_eq_true] at accepted
  rcases accepted with ⟨_, checked⟩
  generalize certificateEquality : request.certificate? = certificate at checked
  cases certificate with
  | none => simp at checked
  | some certificate =>
      have valid : certificate.Valid soundFiniteView noStaleCompletion .exhaustive := by
        exact checked
      exact replicatedSafety
        (model := behavior)
        (world := { core := .smoke, replicas := request.replicas })
        (property := fun state => noStaleCompletion state = true)
        (certificate.valid_exhaustive_safety soundFiniteView noStaleCompletion valid)

open Lean Elab Term in
syntax (name := resolvedNativeCertificateAxioms)
  "resolved_native_certificate_axioms% " ident : term

open Lean Elab Term in
@[term_elab resolvedNativeCertificateAxioms] meta def elabResolvedNativeCertificateAxioms :
    TermElab := fun stx expectedType? => do
  let `(resolved_native_certificate_axioms% $theoremName:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo theoremName
  let axioms ← collectAxioms declarationName
  rejectForbiddenAxioms theoremName axioms
  let axioms := axioms.map toString |>.qsort (· < ·) |>.map Syntax.mkStrLit
  let expanded ← `([$[$axioms],*])
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

def checkedRequestAxioms : List String :=
  resolved_native_certificate_axioms% checkedRequest

private def encodeLifecycle : Lifecycle → String
  | .open => "open"
  | .cancellationAccepted => "cancellation-accepted"
  | .cancelled => "cancelled"
  | .succeeded => "succeeded"

private def encodeTask : TaskStage → String
  | .idle => "idle"
  | .dispatched => "dispatched"
  | .returned => "returned"

private def encodeEpoch (value : Nat) : String := s!"epoch-{value}"

private def encodeOptionalEpoch : Option Nat → String
  | none => "none"
  | some value => encodeEpoch value

private def stateJson (state : State) : Lean.Json := Lean.Json.mkObj [
  ("fields", Lean.Json.arr #[
    Lean.Json.mkObj [("field", "lifecycle"), ("value", encodeLifecycle state.lifecycle)],
    Lean.Json.mkObj [("field", "task"), ("value", encodeTask state.task)],
    Lean.Json.mkObj [("field", "owner-epoch"), ("value", encodeEpoch state.ownerEpoch)],
    Lean.Json.mkObj [("field", "worker-epoch"), ("value", encodeOptionalEpoch state.workerEpoch)],
    Lean.Json.mkObj [
      ("field", "completion-epoch"), ("value", encodeOptionalEpoch state.completionEpoch)],
  ]),
]

private def checkedNodeJson (wire : WireNode) (node : Exact.Node behavior .smoke) : Lean.Json :=
  Lean.Json.mkObj [
    ("state", stateJson wire.state),
    ("parent", Lean.toJson (match wire.parent with | none => (-1 : Int) | some value => value)),
    ("action", match wire.action with | none => "" | some action => actionName action),
    ("depth", Lean.toJson node.depth),
  ]

private def Request.checkedNodesJson (request : Request) : Lean.Json :=
  match request.certificate? with
  | none => Lean.Json.arr #[]
  | some certificate => Lean.Json.arr
      ((request.nodes.zip certificate.nodes).map fun pair =>
        checkedNodeJson pair.1 pair.2).toArray

def Request.receipt (request : Request) : Lean.Json := Lean.Json.mkObj [
  ("formatVersion", "umpire3/native-certificate-receipt/v1"),
  ("certificateDigest", request.certificateDigest),
  ("viewDigest", request.viewDigest),
  ("target", request.target),
  ("property", request.property),
  ("world", request.world),
  ("variant", request.variant),
  ("semanticHash", request.semanticHash),
  ("resultClass", "finite-exhaustive"),
  ("trustBadge", "checked-certificate"),
  ("expandedStates", Lean.toJson request.expandedStates),
  ("representativeStates", Lean.toJson request.representativeStates),
  ("replicas", Lean.toJson request.replicas),
  ("nodes", request.checkedNodesJson),
  ("axioms", Lean.Json.arr (checkedRequestAxioms.map Lean.toJson).toArray),
]

def parseAction : String → Option Action
  | "dispatch-task" => some .dispatchTask
  | "request-cancellation" => some .acceptCancellation
  | "acquire-ownership" => some .acquireOwnership
  | "commit-cancellation" => some .commitCancellation
  | "worker-returns-success" => some .returnSuccess
  | "persist-success" => some .persistSuccess
  | _ => none

def parseLifecycle : String → Option Lifecycle
  | "open" => some .open
  | "cancellation-accepted" => some .cancellationAccepted
  | "cancelled" => some .cancelled
  | "succeeded" => some .succeeded
  | _ => none

def parseTask : String → Option TaskStage
  | "idle" => some .idle
  | "dispatched" => some .dispatched
  | "returned" => some .returned
  | _ => none

def parseOptionalEpoch : String → Option (Option Nat)
  | "none" => some none
  | value => value.toNat?.map some

def parseWireNode (parentText actionText lifecycleText taskText ownerText workerText
    completionText : String) : Option WireNode := do
  let parent ← if parentText == "root" then some none else parentText.toNat?.map some
  let action ← if actionText == "root" then some none else (parseAction actionText).map some
  let lifecycle ← parseLifecycle lifecycleText
  let task ← parseTask taskText
  let ownerEpoch ← ownerText.toNat?
  let workerEpoch ← parseOptionalEpoch workerText
  let completionEpoch ← parseOptionalEpoch completionText
  some { parent, action, state := { lifecycle, task, ownerEpoch, workerEpoch, completionEpoch } }

def parseWireNodes : Nat → List String → Option (List WireNode)
  | 0, [] => some []
  | count + 1,
      parent :: action :: lifecycle :: task :: owner :: worker :: completion :: remaining => do
      let node ← parseWireNode parent action lifecycle task owner worker completion
      let tail ← parseWireNodes count remaining
      some (node :: tail)
  | _, _ => none

end Umpire3.NativeCertificate
