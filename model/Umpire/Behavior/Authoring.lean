import Umpire.Behavior.Language
import Umpire.Target.Authoring
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

structure BehaviorAuthoringDiagnostic where
  error : BehaviorError
  role : BehaviorAuthoringRole
  anchor : BehaviorAuthoringSpan
  deriving BEq, DecidableEq, Repr

private def quoteJson (value : String) : String :=
  Lean.Json.compress (.str value)

def canonicalBehaviorAuthoringDiagnosticJson
    (diagnostic : BehaviorAuthoringDiagnostic) : String :=
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

private structure CapturedBehaviorAuthoringOccurrence where
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

private def captureBehaviorAuthoringOccurrence
    (role : BehaviorAuthoringRole)
    (reference : Lean.TSyntax `term) :
    Lean.Elab.Term.TermElabM CapturedBehaviorAuthoringOccurrence := do
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

private def selectBehaviorAuthoringOccurrence
    (error : BehaviorError)
    (occurrences : List CapturedBehaviorAuthoringOccurrence) :
    Option CapturedBehaviorAuthoringOccurrence :=
  let related := occurrences.filter fun occurrence =>
    error.relatedDefinitionIds.contains occurrence.definitionId
  related.getLast? <|> occurrences.find? (fun occurrence => occurrence.role == .parent)

private def elaborateBehavior
    (specSyntax contextSyntax : Lean.TSyntax `term)
    (occurrences : List CapturedBehaviorAuthoringOccurrence)
    (expectedType : Option Lean.Expr) : Lean.Elab.Term.TermElabM Lean.Expr := do
  let errorExpression ← Lean.Elab.Term.elabTerm
    (← `(ExactSequenceSpec.error? $specSyntax $contextSyntax))
    (some (.app (.const ``Option [.zero]) (.const ``BehaviorError [])))
  let errorExpression ← Lean.instantiateMVars errorExpression
  if errorExpression.hasFVar then
    return ← Lean.Elab.Term.elabTerm
      (← `(ExactSequenceSpec.check $specSyntax $contextSyntax)) expectedType
  match ← evalBehaviorError errorExpression with
  | some error =>
      match selectBehaviorAuthoringOccurrence error occurrences with
      | some occurrence =>
          Lean.throwErrorAt occurrence.reference s!"behavior authoring failed: {
            canonicalBehaviorAuthoringDiagnosticJson {
              error, role := occurrence.role, anchor := occurrence.anchor }}"
      | none =>
          let anchor ← behaviorAuthoringSpan specSyntax.raw
          Lean.throwErrorAt specSyntax.raw s!"behavior authoring failed: {
            canonicalBehaviorAuthoringDiagnosticJson { error, role := .parent, anchor }}"
  | none =>
      Lean.Elab.Term.elabTerm (← `(ExactSequenceSpec.check $specSyntax $contextSyntax)) expectedType

declare_syntax_cat behaviorAuthoringOccurrence
syntax ident term : behaviorAuthoringOccurrence
syntax (name := checkedBehaviorSyntax)
  "behavior%" term "against" term "tracking" "[" behaviorAuthoringOccurrence,* "]" : term

elab_rules : term
  | `(behavior% $spec against $context tracking [$occurrences,*]) => do
      let mut captured := []
      for occurrence in occurrences.getElems do
        let item ← match occurrence with
          | `(behaviorAuthoringOccurrence| $role:ident $reference) =>
              let role ← match role.getId.toString with
                | "behaviorParent" => pure BehaviorAuthoringRole.parent
                | "setupAnchor" => pure BehaviorAuthoringRole.setup
                | "occurrenceAnchor" => pure BehaviorAuthoringRole.occurrence
                | "actionAnchor" => pure BehaviorAuthoringRole.action
                | _ => Lean.throwErrorAt role.raw ("expected behaviorParent, setupAnchor, " ++
                    "occurrenceAnchor, or actionAnchor")
              captureBehaviorAuthoringOccurrence role reference
          | _ => Lean.Elab.throwUnsupportedSyntax
        captured := captured ++ [item]
      elaborateBehavior spec context captured none

end Umpire
