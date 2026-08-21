import Lean.Data.Json
import Lean.Elab.Term
import Lean.Util.CollectAxioms
import Umpire3.FirstOrderView

namespace Umpire3

inductive ActionOutcome where
  | applied
  | suppressed
  | rejected
  | retried
  | faultIntercepted
  deriving DecidableEq, Repr

structure AttemptOutcome where
  outcome : ActionOutcome
  guard : FirstOrderFormula
  transitions : List String
  deriving DecidableEq, Repr

structure AttemptMapping where
  action : String
  outcomes : List AttemptOutcome
  deriving DecidableEq, Repr

structure AttemptArtifact where
  target : String
  property : String
  world : String
  variant : String
  canonicalModel : String
  attempts : List AttemptMapping
  deriving DecidableEq, Repr

private def ActionOutcome.toString : ActionOutcome → String
  | .applied => "applied"
  | .suppressed => "suppressed"
  | .rejected => "rejected"
  | .retried => "retried"
  | .faultIntercepted => "fault-intercepted"

private def stringsUnique : List String → Bool
  | [] => true
  | value :: values => !values.contains value && stringsUnique values

private def AttemptMapping.validFor
    (mapping : AttemptMapping) (firstOrder : FirstOrderArtifact) : Bool :=
  let outcomeNames := mapping.outcomes.map fun outcome => outcome.outcome.toString
  let transitionsKnown := mapping.outcomes.all fun outcome =>
    outcome.transitions.all fun transition => firstOrder.actionIdentifiers.contains transition
  let appliedTransitions := if firstOrder.actionIdentifiers.contains mapping.action then
    [mapping.action]
  else
    []
  let appliedValid := mapping.outcomes.any fun outcome =>
    outcome.outcome = .applied && outcome.transitions = appliedTransitions
  !mapping.action.isEmpty && !mapping.outcomes.isEmpty && stringsUnique outcomeNames &&
    transitionsKnown && appliedValid

def AttemptArtifact.validFor
    (artifact : AttemptArtifact) (firstOrder : FirstOrderArtifact) : Bool :=
  let expectedActions := firstOrder.actionIdentifiers ++ firstOrder.liveOnlyActions
  let mappedActions := artifact.attempts.map AttemptMapping.action
  artifact.target = firstOrder.target && artifact.property = firstOrder.property &&
    artifact.world = firstOrder.world && artifact.variant = firstOrder.variant &&
    artifact.canonicalModel = firstOrder.canonicalModel &&
    stringsUnique expectedActions && mappedActions = expectedActions &&
    artifact.attempts.all fun mapping => mapping.validFor firstOrder

private def applyTransitions (firstOrder : FirstOrderArtifact) :
    FirstOrderState → List String → Option FirstOrderState
  | state, [] => some state
  | state, transition :: transitions =>
      match firstOrder.next state transition with
      | none => none
      | some next => applyTransitions firstOrder next transitions

def AttemptArtifact.apply (artifact : AttemptArtifact) (firstOrder : FirstOrderArtifact)
    (state : FirstOrderState) (action : String) (outcome : ActionOutcome) :
    Option FirstOrderState :=
  match artifact.attempts.find? fun mapping => mapping.action == action with
  | none => none
  | some mapping =>
      match mapping.outcomes.find? fun candidate => candidate.outcome == outcome with
      | none => none
      | some candidate =>
          if candidate.guard.eval state then
            applyTransitions firstOrder state candidate.transitions
          else none

structure AttemptView {World : Type} {model : Behavior World} {world : World}
    {property : model.State world → Bool} (firstOrder : FirstOrderView model world property) where
  artifact : AttemptArtifact
  valid : artifact.validFor firstOrder.artifact = true

private def ActionOutcome.toJson (outcome : ActionOutcome) : Lean.Json :=
  Lean.toJson outcome.toString

private def AttemptOutcome.toJson (outcome : AttemptOutcome) : Lean.Json := Lean.Json.mkObj [
  ("outcome", outcome.outcome.toJson),
  ("guard", outcome.guard.toJson),
  ("transitions", Lean.Json.arr (outcome.transitions.map Lean.toJson).toArray),
]

private def AttemptMapping.toJson (mapping : AttemptMapping) : Lean.Json := Lean.Json.mkObj [
  ("action", mapping.action),
  ("outcomes", Lean.Json.arr (mapping.outcomes.map AttemptOutcome.toJson).toArray),
]

structure ResolvedAttemptView where
  artifact : AttemptArtifact
  declaration : String
  axioms : List String

open Lean Elab Term in
syntax (name := resolvedAttemptView) "resolved_attempt% " ident : term

open Lean Elab Term in
@[term_elab resolvedAttemptView] meta def elabResolvedAttemptView : TermElab :=
    fun stx expectedType? => do
  let `(resolved_attempt% $viewName:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo viewName
  let declaration ← getConstInfo declarationName
  unless declaration.type.getAppFn.constName? == some ``AttemptView do
    throwErrorAt viewName "resolved attempt declaration must have type AttemptView"
  let axioms ← collectAxioms declarationName
  rejectForbiddenAxioms viewName axioms
  let axioms := axioms.qsort Name.lt |>.map (Syntax.mkStrLit ∘ toString)
  let expanded ← `(ResolvedAttemptView.mk
    $(mkIdent declarationName).artifact
    $(Syntax.mkStrLit declarationName.toString)
    [$[$axioms],*])
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

structure AttemptExport where
  view : ResolvedAttemptView

def AttemptExport.toJson (exported : AttemptExport)
    (semanticHash firstOrderSemanticHash : String) : Lean.Json := Lean.Json.mkObj [
  ("formatVersion", "umpire3/attempt-view/v1"),
  ("target", exported.view.artifact.target),
  ("property", exported.view.artifact.property),
  ("world", exported.view.artifact.world),
  ("variant", exported.view.artifact.variant),
  ("semanticHash", semanticHash),
  ("firstOrderSemanticHash", firstOrderSemanticHash),
  ("canonicalModel", exported.view.artifact.canonicalModel),
  ("relation", Lean.Json.mkObj [
    ("declaration", exported.view.declaration),
    ("axioms", Lean.Json.arr (exported.view.axioms.map Lean.toJson).toArray),
    ("trustBadge", if exported.view.axioms.isEmpty then
      "kernel" else "kernel-with-declared-axioms"),
  ]),
  ("attempts", Lean.Json.arr
    (exported.view.artifact.attempts.map AttemptMapping.toJson).toArray),
]

def AttemptExport.json (exported : AttemptExport)
    (semanticHash firstOrderSemanticHash : String) : String :=
  (exported.toJson semanticHash firstOrderSemanticHash).compress

end Umpire3
