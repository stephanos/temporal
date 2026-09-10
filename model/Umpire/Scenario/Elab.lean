import Umpire.Scenario.Check
import Umpire.Id
import Lean.Elab.Term
import Lean.Meta.Eval

/-! Located-diagnostic elaboration of authored Scenarios. -/

namespace Umpire

inductive ScenarioAuthoringRole where
  | parent
  | setup
  | occurrence
  | action
  deriving BEq, DecidableEq, Repr

def ScenarioAuthoringRole.name : ScenarioAuthoringRole → String
  | .parent => "parent"
  | .setup => "setup"
  | .occurrence => "occurrence"
  | .action => "action"

structure ScenarioAuthoringSpan where
  sourcePath : String
  line : Nat
  column : Nat
  endLine : Nat
  endColumn : Nat
  deriving BEq, DecidableEq, Repr

structure ScenarioLocatedError where
  error : ScenarioError
  role : ScenarioAuthoringRole
  anchor : ScenarioAuthoringSpan
  deriving BEq, DecidableEq, Repr

private def quoteJson (value : String) : String :=
  Lean.Json.compress (.str value)

def canonicalScenarioLocatedErrorJson
    (diagnostic : ScenarioLocatedError) : String :=
  "{\"error\":" ++ canonicalScenarioErrorJson diagnostic.error ++
    ",\"role\":" ++ quoteJson diagnostic.role.name ++
    ",\"anchor\":{\"sourcePath\":" ++ quoteJson diagnostic.anchor.sourcePath ++
    ",\"line\":" ++ toString diagnostic.anchor.line ++
    ",\"column\":" ++ toString diagnostic.anchor.column ++
    ",\"endLine\":" ++ toString diagnostic.anchor.endLine ++
    ",\"endColumn\":" ++ toString diagnostic.anchor.endColumn ++ "}}"

private def scenarioAuthoringSpan
    (reference : Lean.Syntax) : Lean.Elab.Term.TermElabM ScenarioAuthoringSpan := do
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

private structure CapturedScenarioSourceRef where
  role : ScenarioAuthoringRole
  definitionId : DefinitionId
  reference : Lean.Syntax
  anchor : ScenarioAuthoringSpan

private unsafe def evalDefinitionIdUnsafe (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM DefinitionId :=
  Lean.Meta.evalExpr DefinitionId (.const ``DefinitionId []) expression

@[implemented_by evalDefinitionIdUnsafe]
private opaque evalDefinitionId (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM DefinitionId

private unsafe def evalScenarioErrorUnsafe (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM (Option ScenarioError) :=
  Lean.Meta.evalExpr (Option ScenarioError)
    (.app (.const ``Option [.zero]) (.const ``ScenarioError [])) expression

@[implemented_by evalScenarioErrorUnsafe]
private opaque evalScenarioError (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM (Option ScenarioError)

private def captureScenarioSourceRef
    (role : ScenarioAuthoringRole)
    (reference : Lean.TSyntax `term) :
    Lean.Elab.Term.TermElabM CapturedScenarioSourceRef := do
  let expression ← Lean.Elab.Term.elabTerm reference (some (.const ``DefinitionId []))
  let definitionId ← if expression.hasFVar || expression.hasMVar then
    pure (DefinitionId.of "umpire.authoring.local")
  else
    evalDefinitionId expression
  pure {
    role
    definitionId
    reference := reference.raw
    anchor := ← scenarioAuthoringSpan reference.raw
  }

private def selectScenarioSourceRef
    (error : ScenarioError)
    (occurrences : List CapturedScenarioSourceRef) :
    Option CapturedScenarioSourceRef :=
  let related := occurrences.filter fun occurrence =>
    error.relatedDefinitionIds.contains occurrence.definitionId
  related.getLast? <|> occurrences.find? (fun occurrence => occurrence.role == .parent)

private def elaborateScenario
    (specSyntax contextSyntax : Lean.TSyntax `term)
    (occurrences : List CapturedScenarioSourceRef)
    (expectedType : Option Lean.Expr) : Lean.Elab.Term.TermElabM Lean.Expr := do
  let errorSyntax ← `(Scenario.error? $contextSyntax $specSyntax)
  let checkSyntax ← `(Scenario.check $contextSyntax $specSyntax)
  let errorExpression ← Lean.Elab.Term.elabTerm
    errorSyntax
    (some (.app (.const ``Option [.zero]) (.const ``ScenarioError [])))
  let errorExpression ← Lean.instantiateMVars errorExpression
  if errorExpression.hasFVar then
    return ← Lean.Elab.Term.elabTerm
      checkSyntax expectedType
  match ← evalScenarioError errorExpression with
  | some error =>
      match selectScenarioSourceRef error occurrences with
      | some occurrence =>
          Lean.throwErrorAt occurrence.reference s!"scenario authoring failed: {
            canonicalScenarioLocatedErrorJson {
              error, role := occurrence.role, anchor := occurrence.anchor }}"
      | none =>
          let anchor ← scenarioAuthoringSpan specSyntax.raw
          Lean.throwErrorAt specSyntax.raw s!"scenario authoring failed: {
            canonicalScenarioLocatedErrorJson { error, role := .parent, anchor }}"
  | none =>
      Lean.Elab.Term.elabTerm checkSyntax expectedType

declare_syntax_cat scenarioSourceRef
syntax ident term : scenarioSourceRef
syntax (name := checkedScenarioSyntax)
  "scenario%" term "against" term "tracking" "[" scenarioSourceRef,* "]" : term

elab_rules : term
  | `(scenario% $spec against $context tracking [$occurrences,*]) => do
      let mut captured := []
      for occurrence in occurrences.getElems do
        let item ← match occurrence with
          | `(scenarioSourceRef| $role:ident $reference) =>
              let role ← match role.getId.toString with
                | "scenarioParent" => pure ScenarioAuthoringRole.parent
                | "setupAnchor" => pure ScenarioAuthoringRole.setup
                | "occurrenceAnchor" => pure ScenarioAuthoringRole.occurrence
                | "actionAnchor" => pure ScenarioAuthoringRole.action
                | _ => Lean.throwErrorAt role.raw ("expected scenarioParent, setupAnchor, " ++
                    "occurrenceAnchor, or actionAnchor")
              captureScenarioSourceRef role reference
          | _ => Lean.Elab.throwUnsupportedSyntax
        captured := captured ++ [item]
      elaborateScenario spec context captured none

end Umpire
