import Umpire.Behavior.Language
import Umpire.Id
import Lean.Elab.Term
import Lean.Meta.Eval

/-! Narrow exact-sequence construction over the existing checked Behavior language. -/

namespace Umpire

structure SequenceOccurrence where
  key : String
  action : DefinitionId
  deriving BEq, DecidableEq, Repr

structure ExactSequenceSpec where
  family : DefinitionFamily
  key : String
  source : SourceLocation
  requires : List DefinitionId := []
  roles : List ResourceRole := []
  setup : List SetupConstraint := []
  occurrences : List SequenceOccurrence
  documentation : String := ""

private def ExactSequenceSpec.namedOccurrences
    (spec : ExactSequenceSpec) : List NamedOccurrence :=
  spec.occurrences.map fun occurrence => {
    id := spec.family.id "occurrence" occurrence.key
    action := occurrence.action
  }

private def adjacentOrders : List NamedOccurrence → List OccurrenceOrder
  | first :: second :: rest =>
      { before := first.id, after := second.id } :: adjacentOrders (second :: rest)
  | _ => []

def ExactSequenceSpec.declaration (spec : ExactSequenceSpec) : BehaviorDeclaration :=
  let occurrences := spec.namedOccurrences
  let actions := occurrences.map NamedOccurrence.action
  {
    id := spec.family.id "behavior" spec.key
    source := spec.source
    requires := spec.requires
    roles := spec.roles
    setup := spec.setup
    allowedActions := DefinitionId.canonicalSet actions
    requiredOccurrences := occurrences
    occurrenceBounds := (DefinitionId.canonicalSet actions).map fun action =>
      OccurrenceBound.exactly action (actions.count action)
    ordering := adjacentOrders occurrences
    actionsExactly := some actions
    documentation := spec.documentation
  }

/-- Admit a constructor-authored Behavior only through the existing language checker. -/
def ExactSequenceSpec.check
    (spec : ExactSequenceSpec)
    (context : BehaviorCheckContext) : Except BehaviorError CheckedBehavior :=
  checkBehavior context spec.declaration

/-- Produce the checked value after the kernel verifies that the existing checker succeeds. -/
def ExactSequenceSpec.checked
    (spec : ExactSequenceSpec)
    (context : BehaviorCheckContext)
    (valid : (spec.check context).toOption.isSome = true) : CheckedBehavior :=
  checkedBehavior context spec.declaration valid

def ExactSequenceSpec.error?
    (spec : ExactSequenceSpec)
    (context : BehaviorCheckContext) : Option BehaviorError :=
  match spec.check context with
  | .error error => some error
  | .ok _ => none

/-- Existing trace constraints. Occurrence keys are local to the spec's definition family;
`before` and `inOrder` permit intervening allowed actions, while `adjacent` does not. -/
inductive BehaviorConstraint where
  | allow (actions : List DefinitionId)
  | forbid (actions : List DefinitionId)
  | require (key : String) (action : DefinitionId)
  | bound (bound : OccurrenceBound)
  | before (first second : String)
  | inOrder (actions : List DefinitionId)
  | adjacent (actions : List DefinitionId)
  deriving BEq, DecidableEq, Repr

/-- Typed authoring of existing Behavior constraints. Exact schedules and traces remain explicit;
checking these constraints never establishes that a Target can execute them. -/
structure BehaviorSpec where
  family : DefinitionFamily
  key : String
  source : SourceLocation
  requires : List DefinitionId := []
  roles : List ResourceRole := []
  setup : List SetupConstraint := []
  constraints : List BehaviorConstraint := []
  actionsExactly : Option (List DefinitionId) := none
  traceExactly : Option AuthoredExactTrace := none
  documentation : String := ""

/-- Lower into the existing declaration before validation, canonicalization, or evaluation. -/
def BehaviorSpec.declaration (spec : BehaviorSpec) : BehaviorDeclaration :=
  spec.constraints.foldl (fun declaration constraint =>
    match constraint with
    | .allow actions =>
        { declaration with allowedActions := declaration.allowedActions ++ actions }
    | .forbid actions =>
        { declaration with forbiddenActions := declaration.forbiddenActions ++ actions }
    | .require key action =>
        { declaration with requiredOccurrences := declaration.requiredOccurrences ++ [{
          id := spec.family.id "occurrence" key
          action
        }] }
    | .bound bound =>
        { declaration with occurrenceBounds := declaration.occurrenceBounds ++ [bound] }
    | .before first second =>
        { declaration with ordering := declaration.ordering ++ [{
          before := spec.family.id "occurrence" first
          after := spec.family.id "occurrence" second
        }] }
    | .inOrder actions =>
        { declaration with sequences := declaration.sequences ++ [actions] }
    | .adjacent actions =>
        { declaration with adjacencies := declaration.adjacencies ++ [actions] }) {
    id := spec.family.id "behavior" spec.key
    source := spec.source
    requires := spec.requires
    roles := spec.roles
    setup := spec.setup
    actionsExactly := spec.actionsExactly
    traceExactly := spec.traceExactly
    documentation := spec.documentation
  }

/-- Admit typed constraints through the canonical Behavior checker. -/
def BehaviorSpec.check
    (spec : BehaviorSpec)
    (context : BehaviorCheckContext) : Except BehaviorError CheckedBehavior :=
  checkBehavior context spec.declaration

/-- Construct a checked Behavior using kernel-checked admission evidence. -/
def BehaviorSpec.checked
    (spec : BehaviorSpec)
    (context : BehaviorCheckContext)
    (valid : (spec.check context).toOption.isSome = true) : CheckedBehavior :=
  checkedBehavior context spec.declaration valid

def BehaviorSpec.error?
    (spec : BehaviorSpec)
    (context : BehaviorCheckContext) : Option BehaviorError :=
  match spec.check context with
  | .error error => some error
  | .ok _ => none

inductive BehaviorAuthoringRole where
  | parent
  | setup
  | occurrence
  | action
  deriving BEq, DecidableEq, Repr

def BehaviorAuthoringRole.name : BehaviorAuthoringRole → String
  | .parent => "parent"
  | .setup => "setup"
  | .occurrence => "occurrence"
  | .action => "action"

structure BehaviorAuthoringSpan where
  sourcePath : String
  line : Nat
  column : Nat
  endLine : Nat
  endColumn : Nat
  deriving BEq, DecidableEq, Repr

structure BehaviorLocatedError where
  error : BehaviorError
  role : BehaviorAuthoringRole
  anchor : BehaviorAuthoringSpan
  deriving BEq, DecidableEq, Repr

private def quoteJson (value : String) : String :=
  Lean.Json.compress (.str value)

def canonicalBehaviorLocatedErrorJson
    (diagnostic : BehaviorLocatedError) : String :=
  "{\"error\":" ++ canonicalBehaviorErrorJson diagnostic.error ++
    ",\"role\":" ++ quoteJson diagnostic.role.name ++
    ",\"anchor\":{\"sourcePath\":" ++ quoteJson diagnostic.anchor.sourcePath ++
    ",\"line\":" ++ toString diagnostic.anchor.line ++
    ",\"column\":" ++ toString diagnostic.anchor.column ++
    ",\"endLine\":" ++ toString diagnostic.anchor.endLine ++
    ",\"endColumn\":" ++ toString diagnostic.anchor.endColumn ++ "}}"

private def behaviorAuthoringSpan
    (reference : Lean.Syntax) : Lean.Elab.Term.TermElabM BehaviorAuthoringSpan := do
  let fileMap ← Lean.getFileMap
  let fileName ← Lean.getFileName
  let sourcePath := (fileName.splitOn "/model/").getLast?.getD fileName
  let startOffset := reference.getPos?.getD 0
  let endOffset := reference.getTailPos?.getD startOffset
  let startPosition := fileMap.toPosition startOffset
  let endPosition := fileMap.toPosition endOffset
  pure {
    sourcePath
    line := startPosition.line
    column := startPosition.column
    endLine := endPosition.line
    endColumn := endPosition.column
  }

private structure CapturedBehaviorSourceRef where
  role : BehaviorAuthoringRole
  definitionId : DefinitionId
  reference : Lean.Syntax
  anchor : BehaviorAuthoringSpan

private unsafe def evalDefinitionIdUnsafe (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM DefinitionId :=
  Lean.Meta.evalExpr DefinitionId (.const ``DefinitionId []) expression

@[implemented_by evalDefinitionIdUnsafe]
private opaque evalDefinitionId (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM DefinitionId

private unsafe def evalBehaviorErrorUnsafe (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM (Option BehaviorError) :=
  Lean.Meta.evalExpr (Option BehaviorError)
    (.app (.const ``Option [.zero]) (.const ``BehaviorError [])) expression

@[implemented_by evalBehaviorErrorUnsafe]
private opaque evalBehaviorError (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM (Option BehaviorError)

private def captureBehaviorSourceRef
    (role : BehaviorAuthoringRole)
    (reference : Lean.TSyntax `term) :
    Lean.Elab.Term.TermElabM CapturedBehaviorSourceRef := do
  let expression ← Lean.Elab.Term.elabTerm reference (some (.const ``DefinitionId []))
  let definitionId ← if expression.hasFVar || expression.hasMVar then
    pure (DefinitionId.of "umpire.authoring.local")
  else
    evalDefinitionId expression
  pure {
    role
    definitionId
    reference := reference.raw
    anchor := ← behaviorAuthoringSpan reference.raw
  }

private def selectBehaviorSourceRef
    (error : BehaviorError)
    (occurrences : List CapturedBehaviorSourceRef) :
    Option CapturedBehaviorSourceRef :=
  let related := occurrences.filter fun occurrence =>
    error.relatedDefinitionIds.contains occurrence.definitionId
  related.getLast? <|> occurrences.find? (fun occurrence => occurrence.role == .parent)

private def elaborateBehavior
    (specSyntax contextSyntax : Lean.TSyntax `term)
    (occurrences : List CapturedBehaviorSourceRef)
    (expectedType : Option Lean.Expr) : Lean.Elab.Term.TermElabM Lean.Expr := do
  let exactSpec ← Lean.Elab.Term.observing <| Lean.Elab.Term.withoutErrToSorry <|
    Lean.Elab.Term.elabTermEnsuringType specSyntax (some (.const ``ExactSequenceSpec []))
  let (errorSyntax, checkSyntax) ← match exactSpec with
    | .ok _ _ => pure (
        ← `(ExactSequenceSpec.error? $specSyntax $contextSyntax),
        ← `(ExactSequenceSpec.check $specSyntax $contextSyntax))
    | .error _ _ => pure (
        ← `(BehaviorSpec.error? $specSyntax $contextSyntax),
        ← `(BehaviorSpec.check $specSyntax $contextSyntax))
  let errorExpression ← Lean.Elab.Term.elabTerm
    errorSyntax
    (some (.app (.const ``Option [.zero]) (.const ``BehaviorError [])))
  let errorExpression ← Lean.instantiateMVars errorExpression
  if errorExpression.hasFVar then
    return ← Lean.Elab.Term.elabTerm
      checkSyntax expectedType
  match ← evalBehaviorError errorExpression with
  | some error =>
      match selectBehaviorSourceRef error occurrences with
      | some occurrence =>
          Lean.throwErrorAt occurrence.reference s!"behavior authoring failed: {
            canonicalBehaviorLocatedErrorJson {
              error, role := occurrence.role, anchor := occurrence.anchor }}"
      | none =>
          let anchor ← behaviorAuthoringSpan specSyntax.raw
          Lean.throwErrorAt specSyntax.raw s!"behavior authoring failed: {
            canonicalBehaviorLocatedErrorJson { error, role := .parent, anchor }}"
  | none =>
      Lean.Elab.Term.elabTerm checkSyntax expectedType

declare_syntax_cat behaviorSourceRef
syntax ident term : behaviorSourceRef
syntax (name := checkedBehaviorSyntax)
  "behavior%" term "against" term "tracking" "[" behaviorSourceRef,* "]" : term

elab_rules : term
  | `(behavior% $spec against $context tracking [$occurrences,*]) => do
      let mut captured := []
      for occurrence in occurrences.getElems do
        let item ← match occurrence with
          | `(behaviorSourceRef| $role:ident $reference) =>
              let role ← match role.getId.toString with
                | "behaviorParent" => pure BehaviorAuthoringRole.parent
                | "setupAnchor" => pure BehaviorAuthoringRole.setup
                | "occurrenceAnchor" => pure BehaviorAuthoringRole.occurrence
                | "actionAnchor" => pure BehaviorAuthoringRole.action
                | _ => Lean.throwErrorAt role.raw ("expected behaviorParent, setupAnchor, " ++
                    "occurrenceAnchor, or actionAnchor")
              captureBehaviorSourceRef role reference
          | _ => Lean.Elab.throwUnsupportedSyntax
        captured := captured ++ [item]
      elaborateBehavior spec context captured none

end Umpire
