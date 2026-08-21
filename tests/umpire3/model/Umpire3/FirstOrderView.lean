import Lean.Data.Json
import Lean.Elab.Term
import Lean.Util.CollectAxioms
import Umpire3.Explorer
import Umpire3.Registration

namespace Umpire3

inductive FirstOrderSortKind where
  | enumeration
  | uninterpreted
  deriving DecidableEq, Repr

structure FirstOrderSort where
  identifier : String
  kind : FirstOrderSortKind
  values : List String := []
  cardinality : Nat := 0
  deriving DecidableEq, Repr

structure FirstOrderField where
  identifier : String
  sort : String
  deriving DecidableEq, Repr

inductive FirstOrderTerm where
  | field (identifier : String)
  | value (sort : String) (identifier : String)
  deriving DecidableEq, Repr

inductive FirstOrderFormula where
  | truth
  | equal (left right : FirstOrderTerm)
  | not (operand : FirstOrderFormula)
  | all (left right : FirstOrderFormula)
  | any (left right : FirstOrderFormula)
  deriving DecidableEq, Repr

structure FirstOrderUpdate where
  field : String
  value : FirstOrderTerm
  deriving DecidableEq, Repr

structure FirstOrderAction where
  identifier : String
  guard : FirstOrderFormula
  updates : List FirstOrderUpdate
  deriving DecidableEq, Repr

structure FirstOrderBinding where
  field : String
  value : String
  deriving DecidableEq, Repr

structure FirstOrderState where
  fields : List FirstOrderBinding
  deriving DecidableEq, Repr

structure FirstOrderBounds where
  symbolicDepth : Nat
  concreteStateLimit : Nat
  deriving DecidableEq, Repr

structure FirstOrderResource where
  identifier : String
  kind : String
  deriving DecidableEq, Repr

structure FirstOrderArtifact where
  target : String
  property : String
  world : String
  variant : String
  canonicalModel : String
  resources : List FirstOrderResource
  liveOnlyActions : List String
  activatingFaults : List String
  bounds : FirstOrderBounds
  sorts : List FirstOrderSort
  stateFields : List FirstOrderField
  initial : FirstOrderFormula
  actions : List FirstOrderAction
  invariant : FirstOrderFormula
  deriving DecidableEq, Repr

def FirstOrderState.read (state : FirstOrderState) (field : String) : Option String :=
  (state.fields.find? fun binding => binding.field == field).map FirstOrderBinding.value

def FirstOrderState.write (state : FirstOrderState) (field value : String) : FirstOrderState where
  fields := state.fields.map fun binding =>
    if binding.field = field then { binding with value := value } else binding

def FirstOrderTerm.eval (state : FirstOrderState) : FirstOrderTerm → Option String
  | .field identifier => state.read identifier
  | .value _ identifier => some identifier

def FirstOrderFormula.eval (state : FirstOrderState) : FirstOrderFormula → Bool
  | .truth => true
  | .equal left right =>
      match left.eval state, right.eval state with
      | some leftValue, some rightValue => leftValue == rightValue
      | _, _ => false
  | .not operand => !operand.eval state
  | .all left right => left.eval state && right.eval state
  | .any left right => left.eval state || right.eval state

def applyFirstOrderUpdates (before : FirstOrderState) :
    List FirstOrderUpdate → FirstOrderState → Option FirstOrderState
  | [], after => some after
  | update :: updates, after =>
      match update.value.eval before with
      | none => none
      | some value => applyFirstOrderUpdates before updates (after.write update.field value)

def FirstOrderAction.apply (action : FirstOrderAction)
    (state : FirstOrderState) : Option FirstOrderState :=
  if action.guard.eval state then applyFirstOrderUpdates state action.updates state else none

def FirstOrderArtifact.actionIdentifiers (artifact : FirstOrderArtifact) : List String :=
  artifact.actions.map FirstOrderAction.identifier

def FirstOrderArtifact.next (artifact : FirstOrderArtifact)
    (state : FirstOrderState) (action : String) : Option FirstOrderState :=
  match artifact.actions.find? fun candidate => candidate.identifier == action with
  | none => none
  | some candidate => candidate.apply state

structure FirstOrderView {World : Type} (model : Behavior World) (world : World)
    (property : model.State world → Bool) where
  artifact : FirstOrderArtifact
  encodeState : model.State world → FirstOrderState
  encodeAction : model.Action world → String
  admissible : model.State world → Bool
  initial_admissible : ∀ state, model.Initial world state → admissible state = true
  step_admissible : ∀ state action nextState,
    admissible state = true → model.Step world state action nextState → admissible nextState = true
  initial_preserved : ∀ state, model.Initial world state →
    artifact.initial.eval (encodeState state) = true
  step_preserved : ∀ state action nextState,
    admissible state = true → model.Step world state action nextState →
      artifact.next (encodeState state) (encodeAction action) = some (encodeState nextState)
  property_preserved : ∀ state, admissible state = true →
    artifact.invariant.eval (encodeState state) = property state
  action_injective : Function.Injective encodeAction
  action_total : ∀ action, encodeAction action ∈ artifact.actionIdentifiers
  action_complete : ∀ identifier, identifier ∈ artifact.actionIdentifiers →
    ∃ action, encodeAction action = identifier
  action_identifiers_unique : artifact.actionIdentifiers.Pairwise (· ≠ ·)

private def FirstOrderSortKind.toString : FirstOrderSortKind → String
  | .enumeration => "enum"
  | .uninterpreted => "uninterpreted"

private def FirstOrderSort.toJson (sort : FirstOrderSort) : Lean.Json :=
  let fields : List (String × Lean.Json) := [
    ("identifier", sort.identifier),
    ("kind", sort.kind.toString),
    ("values", Lean.Json.arr (sort.values.map Lean.toJson).toArray),
  ]
  let fields := if sort.kind = .uninterpreted then
    fields ++ [("cardinality", Lean.toJson sort.cardinality)]
  else fields
  Lean.Json.mkObj fields

private def FirstOrderField.toJson (field : FirstOrderField) : Lean.Json := Lean.Json.mkObj [
  ("identifier", field.identifier),
  ("sort", field.sort),
]

private def FirstOrderTerm.toJson : FirstOrderTerm → Lean.Json
  | .field identifier => Lean.Json.mkObj [
      ("kind", "field"),
      ("field", identifier),
    ]
  | .value sort identifier => Lean.Json.mkObj [
      ("kind", "value"),
      ("sort", sort),
      ("value", identifier),
    ]

private partial def FirstOrderFormula.toJson : FirstOrderFormula → Lean.Json
  | .truth => Lean.Json.mkObj [("kind", "true")]
  | .equal left right => Lean.Json.mkObj [
      ("kind", "equal"),
      ("left", left.toJson),
      ("right", right.toJson),
    ]
  | .not operand => Lean.Json.mkObj [
      ("kind", "not"),
      ("operand", operand.toJson),
    ]
  | .all left right => Lean.Json.mkObj [
      ("kind", "all"),
      ("operands", Lean.Json.arr #[left.toJson, right.toJson]),
    ]
  | .any left right => Lean.Json.mkObj [
      ("kind", "any"),
      ("operands", Lean.Json.arr #[left.toJson, right.toJson]),
    ]

private def FirstOrderUpdate.toJson (update : FirstOrderUpdate) : Lean.Json := Lean.Json.mkObj [
  ("field", update.field),
  ("value", update.value.toJson),
]

private def FirstOrderAction.toJson (action : FirstOrderAction) : Lean.Json := Lean.Json.mkObj [
  ("identifier", action.identifier),
  ("guard", action.guard.toJson),
  ("updates", Lean.Json.arr (action.updates.map FirstOrderUpdate.toJson).toArray),
]

private def FirstOrderBinding.toJson (binding : FirstOrderBinding) : Lean.Json := Lean.Json.mkObj [
  ("field", binding.field),
  ("value", binding.value),
]

private def FirstOrderState.toJson (state : FirstOrderState) : Lean.Json := Lean.Json.mkObj [
  ("fields", Lean.Json.arr (state.fields.map FirstOrderBinding.toJson).toArray),
]

private def FirstOrderResource.toJson (resource : FirstOrderResource) : Lean.Json := Lean.Json.mkObj [
  ("identifier", resource.identifier),
  ("kind", resource.kind),
]

structure ResolvedFirstOrderView where
  artifact : FirstOrderArtifact
  declaration : String
  axioms : List String

open Lean Elab Term in
syntax (name := resolvedFirstOrderView) "resolved_first_order% " ident : term

open Lean Elab Term in
@[term_elab resolvedFirstOrderView] meta def elabResolvedFirstOrderView : TermElab :=
    fun stx expectedType? => do
  let `(resolved_first_order% $viewName:ident) := stx
    | throwUnsupportedSyntax
  let declarationName ← realizeGlobalConstNoOverloadWithInfo viewName
  let declaration ← getConstInfo declarationName
  unless declaration.type.getAppFn.constName? == some ``FirstOrderView do
    throwErrorAt viewName "resolved first-order declaration must have type FirstOrderView"
  let axioms ← collectAxioms declarationName
  rejectForbiddenAxioms viewName axioms
  let axioms := axioms.qsort Name.lt |>.map (Syntax.mkStrLit ∘ toString)
  let expanded ← `(ResolvedFirstOrderView.mk
    $(mkIdent declarationName).artifact
    $(Syntax.mkStrLit declarationName.toString)
    [$[$axioms],*])
  withMacroExpansion stx expanded <| elabTerm expanded expectedType?

private structure ExhaustiveFirstOrderOracle where
  states : List FirstOrderState

structure FirstOrderExport where
  view : ResolvedFirstOrderView
  private oracle : ExhaustiveFirstOrderOracle

def FirstOrderExport.ofSearch {World : Type} {model : Behavior World} {world : World}
    (resolved : ResolvedFirstOrderView) (finiteView : FiniteView model world)
    (property : model.State world → Bool) (result : Exact.Result model world)
    (encodeState : model.State world → FirstOrderState) : Option FirstOrderExport :=
  match result with
  | .exhausted certificate _ =>
      if certificate.check finiteView property .exhaustive then
        some {
          view := resolved
          oracle := { states := certificate.nodes.map fun node => encodeState node.state }
        }
      else none
  | _ => none

def FirstOrderExport.toJson (exported : FirstOrderExport)
    (semanticHash : String) : Lean.Json := Lean.Json.mkObj [
  ("formatVersion", "umpire3/first-order-view/v2"),
  ("target", exported.view.artifact.target),
  ("property", exported.view.artifact.property),
  ("world", exported.view.artifact.world),
  ("variant", exported.view.artifact.variant),
  ("semanticHash", semanticHash),
  ("canonicalModel", exported.view.artifact.canonicalModel),
  ("resources", Lean.Json.arr
    (exported.view.artifact.resources.map FirstOrderResource.toJson).toArray),
  ("liveOnlyActions", Lean.Json.arr
    (exported.view.artifact.liveOnlyActions.map Lean.toJson).toArray),
  ("activatingFaults", Lean.Json.arr
    (exported.view.artifact.activatingFaults.map Lean.toJson).toArray),
  ("relation", Lean.Json.mkObj [
    ("declaration", exported.view.declaration),
    ("axioms", Lean.Json.arr (exported.view.axioms.map Lean.toJson).toArray),
    ("trustBadge", if exported.view.axioms.isEmpty then
      "kernel" else "kernel-with-declared-axioms"),
  ]),
  ("bounds", Lean.Json.mkObj [
    ("symbolicDepth", Lean.toJson exported.view.artifact.bounds.symbolicDepth),
    ("concreteStateLimit", Lean.toJson exported.view.artifact.bounds.concreteStateLimit),
  ]),
  ("sorts", Lean.Json.arr (exported.view.artifact.sorts.map FirstOrderSort.toJson).toArray),
  ("stateFields", Lean.Json.arr
    (exported.view.artifact.stateFields.map FirstOrderField.toJson).toArray),
  ("initial", exported.view.artifact.initial.toJson),
  ("actions", Lean.Json.arr
    (exported.view.artifact.actions.map FirstOrderAction.toJson).toArray),
  ("invariant", exported.view.artifact.invariant.toJson),
  ("oracle", Lean.Json.mkObj [
    ("resultClass", "finite-exhaustive"),
    ("trustBadge", "checked-certificate"),
    ("states", Lean.Json.arr (exported.oracle.states.map FirstOrderState.toJson).toArray),
  ]),
]

def FirstOrderExport.json (exported : FirstOrderExport) (semanticHash : String) : String :=
  (exported.toJson semanticHash).compress

end Umpire3
