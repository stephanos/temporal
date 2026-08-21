import Lean.Data.Json
import Umpire3.AttemptView
import Umpire3.Explorer
import Umpire3.Registration

namespace Umpire3

structure FiniteReplayAttempt where
  action : String
  outcomes : List ActionOutcome
  appliedPaths : List (List String)
  deriving DecidableEq, Repr

structure FiniteReplayArtifact where
  target : String
  property : String
  world : String
  variant : String
  canonicalModel : String
  attempts : List FiniteReplayAttempt
  deriving DecidableEq, Repr

private def stringsUnique : List String → Bool
  | [] => true
  | value :: values => !values.contains value && stringsUnique values

private def ActionOutcome.toString : ActionOutcome → String
  | .applied => "applied"
  | .suppressed => "suppressed"
  | .rejected => "rejected"
  | .retried => "retried"
  | .faultIntercepted => "fault-intercepted"

private def FiniteReplayAttempt.valid (attempt : FiniteReplayAttempt) : Bool :=
  let outcomes := attempt.outcomes.map ActionOutcome.toString
  !attempt.action.isEmpty && !attempt.outcomes.isEmpty &&
    attempt.outcomes.contains .applied && stringsUnique outcomes && !attempt.appliedPaths.isEmpty

def FiniteReplayArtifact.valid (artifact : FiniteReplayArtifact) : Bool :=
  let actions := artifact.attempts.map FiniteReplayAttempt.action
  !artifact.target.isEmpty && !artifact.property.isEmpty && !artifact.world.isEmpty &&
    !artifact.variant.isEmpty && !artifact.canonicalModel.isEmpty &&
    !artifact.attempts.isEmpty && stringsUnique actions &&
    artifact.attempts.all FiniteReplayAttempt.valid

structure FiniteReplayView {World : Type} (model : Behavior World) (world : World) where
  artifact : FiniteReplayArtifact
  finite : FiniteView model world
  property : model.State world → Bool
  valid : artifact.valid = true

def FiniteView.ofExecutable {World : Type} {model : Behavior World} {world : World}
    [DecidableEq (model.State world)]
    (executable : ExecutableView model) (actionDecidableEq : DecidableEq (model.Action world))
    (actionName : model.Action world → String) (actionNameInjective : Function.Injective actionName)
    (encodedSize : Nat := 8) : FiniteView model world where
  executable := executable
  identity := {
    Code := model.State world
    codeDecidableEq := inferInstance
    encode := id
    encode_injective := fun _ _ equality => equality
    fingerprint := fun _ => 0
    encodedSize := fun _ => encodedSize
  }
  actionDecidableEq := actionDecidableEq
  actionName := actionName
  actionName_injective := actionNameInjective

private def nodeIndex? {World : Type} {model : Behavior World} {world : World}
    (view : FiniteView model world) (nodes : List (Exact.Node model world))
    (state : model.State world) : Option Nat :=
  letI : DecidableEq (model.State world) := view.identity.decidableEq
  nodes.findIdx? fun node => node.state = state

private def edgeToJson? {World : Type} {model : Behavior World} {world : World}
    (view : FiniteView model world) (nodes : List (Exact.Node model world))
    (sourceIndex : Nat) (successor : model.Action world × model.State world) : Option Lean.Json := do
  let some target := nodeIndex? view nodes successor.2 | none
  pure (Lean.Json.mkObj [
    ("from", Lean.toJson sourceIndex),
    ("action", view.actionName successor.1),
    ("to", Lean.toJson target),
  ])

private def edgesToJson? {World : Type} {model : Behavior World} {world : World}
    (view : FiniteView model world) (nodes : List (Exact.Node model world)) : Option (List Lean.Json) :=
  let rec visit : List (Exact.Node model world) → Nat → Option (List Lean.Json)
    | [], _ => some []
    | node :: remaining, index => do
        let current ← (view.successors node.state).mapM (edgeToJson? view nodes index)
        let rest ← visit remaining (index + 1)
        pure (current ++ rest)
  visit nodes 0

private def initialStateIDs {World : Type} {model : Behavior World} {world : World} :
    List (Exact.Node model world) → Nat → List Nat
  | [], _ => []
  | node :: nodes, index =>
      (if node.actions.isEmpty then [index] else []) ++ initialStateIDs nodes (index + 1)

private def FiniteReplayAttempt.toJson (attempt : FiniteReplayAttempt) : Lean.Json :=
  Lean.Json.mkObj [
    ("action", attempt.action),
    ("outcomes", Lean.Json.arr
      (attempt.outcomes.map (Lean.toJson ∘ ActionOutcome.toString)).toArray),
    ("appliedPaths", Lean.Json.arr (attempt.appliedPaths.map fun path =>
      Lean.Json.arr (path.map Lean.toJson).toArray).toArray),
  ]

def FiniteReplayView.toJson? {World : Type} {model : Behavior World} {world : World}
    (view : FiniteReplayView model world) (relation : ResolvedDeclaration)
    (limits : Exact.Limits) (semanticHash : String) : Option Lean.Json :=
  let result := Exact.explore view.finite view.property limits
  match result with
  | .exhausted certificate statistics =>
      if certificate.check view.finite view.property .exhaustive then
        let mappedTransitions := view.artifact.attempts.flatMap fun attempt =>
          attempt.appliedPaths.flatten
        if statistics.coverage.all mappedTransitions.contains &&
            mappedTransitions.all statistics.coverage.contains then
          match edgesToJson? view.finite certificate.nodes with
          | none => none
          | some edges => some (Lean.Json.mkObj [
              ("target", view.artifact.target),
              ("property", view.artifact.property),
              ("world", view.artifact.world),
              ("variant", view.artifact.variant),
              ("semanticHash", semanticHash),
              ("canonicalModel", view.artifact.canonicalModel),
              ("relation", relation.toJson),
              ("resultClass", "finite-exhaustive"),
              ("trustBadge", "checked-certificate"),
              ("bounds", Lean.Json.mkObj [
                ("maxDepth", Lean.toJson limits.maxDepth),
                ("maxStates", Lean.toJson limits.maxStates),
                ("maxTransitions", Lean.toJson limits.maxTransitions),
                ("maxStateBytes", Lean.toJson limits.maxStateBytes),
                ("maxWork", Lean.toJson limits.maxWork),
              ]),
              ("statistics", Lean.Json.mkObj [
                ("states", Lean.toJson statistics.visited),
                ("transitions", Lean.toJson statistics.transitions),
                ("stateBytes", Lean.toJson statistics.stateBytes),
              ]),
              ("initialStates", Lean.Json.arr
                ((initialStateIDs certificate.nodes 0).map Lean.toJson).toArray),
              ("stateCount", Lean.toJson certificate.nodes.length),
              ("transitions", Lean.Json.arr edges.toArray),
              ("attempts", Lean.Json.arr
                (view.artifact.attempts.map FiniteReplayAttempt.toJson).toArray),
            ])
        else none
      else none
  | _ => none

def finiteReplayCatalogJson (semanticHash catalogHash : String) (targets : List Lean.Json) : String :=
  (Lean.Json.mkObj [
    ("formatVersion", "umpire3/finite-replay-catalog/v1"),
    ("semanticHash", semanticHash),
    ("catalogHash", catalogHash),
    ("targets", Lean.Json.arr targets.toArray),
  ]).compress

end Umpire3
