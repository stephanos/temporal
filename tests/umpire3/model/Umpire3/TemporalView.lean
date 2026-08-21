import Lean.Data.Json
import Lean.Elab.Term
import Lean.Util.CollectAxioms
import Umpire3.Registration
import Umpire3.TemporalLogic

namespace Umpire3

structure TemporalTransition where
  action : String
  fromState : String
  toState : String
  deriving DecidableEq, Repr

structure TemporalFairness where
  identifier : String
  kind : String
  action : String
  enabledStates : List String
  deriving DecidableEq, Repr

structure TemporalProgress where
  identifier : String
  triggerStates : List String
  goalStates : List String
  deriving DecidableEq, Repr

structure TemporalBounds where
  maxTraceLength : Nat
  deriving DecidableEq, Repr

structure TemporalResource where
  identifier : String
  kind : String
  deriving DecidableEq, Repr

structure TemporalArtifact where
  target : String
  property : String
  world : String
  variant : String
  claimScope : String
  canonicalModel : String
  resources : List TemporalResource
  liveOnlyActions : List String
  states : List String
  initial : String
  actions : List String
  transitions : List TemporalTransition
  fairness : List TemporalFairness
  progress : TemporalProgress
  bounds : TemporalBounds
  deriving DecidableEq, Repr

structure TemporalView {World : Type} (model : Behavior World) (world : World) where
  artifact : TemporalArtifact
  encodeState : model.State world → String
  encodeAction : model.Action world → String
  states_complete : ∀ identifier, identifier ∈ artifact.states →
    ∃ state, encodeState state = identifier
  actions_complete : ∀ identifier, identifier ∈ artifact.actions →
    ∃ action, encodeAction action = identifier
  initial_exact : ∀ state,
    model.Initial world state ↔ encodeState state = artifact.initial
  step_exact : ∀ state action nextState,
    model.Step world state action nextState ↔
      { action := encodeAction action, fromState := encodeState state,
        toState := encodeState nextState } ∈ artifact.transitions

structure ResolvedTemporalView where
  artifact : TemporalArtifact
  declaration : String
  axioms : List String

open Lean Elab Term in
syntax (name := resolvedTemporalView) "resolved_temporal% " ident : term

open Lean Elab Term in
@[term_elab resolvedTemporalView] meta def elabResolvedTemporalView : TermElab :=
    fun stx expectedType? => do
  let `(resolved_temporal% $viewName:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo viewName
  let declaration ← getConstInfo declarationName
  unless declaration.type.getAppFn.constName? == some ``TemporalView do
    throwErrorAt viewName "resolved temporal declaration must have type TemporalView"
  let axioms ← collectAxioms declarationName
  rejectForbiddenAxioms viewName axioms
  let axioms := axioms.qsort Name.lt |>.map (Syntax.mkStrLit ∘ toString)
  let expanded ← `(ResolvedTemporalView.mk
    $(mkIdent declarationName).artifact
    $(Syntax.mkStrLit declarationName.toString)
    [$[$axioms],*])
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

structure TemporalExport where
  view : ResolvedTemporalView
  proof : Option ResolvedTheorem

private def stringsJson (values : List String) : Lean.Json :=
  Lean.Json.arr (values.map Lean.toJson).toArray

private def TemporalTransition.toJson (transition : TemporalTransition) : Lean.Json :=
  Lean.Json.mkObj [
    ("action", transition.action),
    ("fromState", transition.fromState),
    ("toState", transition.toState),
  ]

private def TemporalFairness.toJson (fairness : TemporalFairness) : Lean.Json :=
  Lean.Json.mkObj [
    ("identifier", fairness.identifier),
    ("kind", fairness.kind),
    ("action", fairness.action),
    ("enabledStates", stringsJson fairness.enabledStates),
  ]

private def TemporalProgress.toJson (progress : TemporalProgress) : Lean.Json :=
  Lean.Json.mkObj [
    ("identifier", progress.identifier),
    ("triggerStates", stringsJson progress.triggerStates),
    ("goalStates", stringsJson progress.goalStates),
  ]

private def TemporalResource.toJson (resource : TemporalResource) : Lean.Json :=
  Lean.Json.mkObj [
    ("identifier", resource.identifier),
    ("kind", resource.kind),
  ]

private def ResolvedTheorem.temporalJson (proof : ResolvedTheorem)
    (fairness : List TemporalFairness) : Lean.Json :=
  Lean.Json.mkObj [
    ("theorem", proof.name),
    ("statement", proof.statement),
    ("resultClass", "temporal-proved"),
    ("trustBadge", if proof.axioms.isEmpty then "kernel" else "kernel-with-declared-axioms"),
    ("axioms", stringsJson proof.axioms),
    ("fairnessAssumptions", stringsJson (fairness.map TemporalFairness.identifier)),
  ]

def TemporalExport.toJson (exported : TemporalExport) (semanticHash : String) : Lean.Json :=
  Lean.Json.mkObj [
    ("formatVersion", "umpire3/temporal-view/v1"),
    ("target", exported.view.artifact.target),
    ("property", exported.view.artifact.property),
    ("world", exported.view.artifact.world),
    ("variant", exported.view.artifact.variant),
    ("claimScope", exported.view.artifact.claimScope),
    ("semanticHash", semanticHash),
    ("canonicalModel", exported.view.artifact.canonicalModel),
    ("resources", Lean.Json.arr
      (exported.view.artifact.resources.map TemporalResource.toJson).toArray),
    ("liveOnlyActions", stringsJson exported.view.artifact.liveOnlyActions),
    ("states", stringsJson exported.view.artifact.states),
    ("initial", exported.view.artifact.initial),
    ("actions", stringsJson exported.view.artifact.actions),
    ("transitions", Lean.Json.arr
      (exported.view.artifact.transitions.map TemporalTransition.toJson).toArray),
    ("fairness", Lean.Json.arr
      (exported.view.artifact.fairness.map TemporalFairness.toJson).toArray),
    ("progress", exported.view.artifact.progress.toJson),
    ("bounds", Lean.Json.mkObj [
      ("maxTraceLength", Lean.toJson exported.view.artifact.bounds.maxTraceLength),
    ]),
    ("relation", Lean.Json.mkObj [
      ("declaration", exported.view.declaration),
      ("trustBadge", if exported.view.axioms.isEmpty then
        "kernel" else "kernel-with-declared-axioms"),
      ("axioms", stringsJson exported.view.axioms),
    ]),
    ("proof", match exported.proof with
      | none => Lean.Json.null
      | some proof => proof.temporalJson exported.view.artifact.fairness),
  ]

def TemporalExport.json (exported : TemporalExport) (semanticHash : String) : String :=
  (exported.toJson semanticHash).compress

end Umpire3
