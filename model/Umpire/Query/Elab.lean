import Umpire.Query.Check
import Umpire.Id
import Lean.Elab.Term
import Lean.Meta.Eval

/-! Located-diagnostic elaboration of authored Queries. -/

namespace Umpire

inductive QueryAuthoringRole where
  | parent
  | target
  | property
  | behavior
  | limits
  | policy
  deriving BEq, DecidableEq, Repr

def QueryAuthoringRole.name : QueryAuthoringRole → String
  | .parent => "parent"
  | .target => "target"
  | .property => "property"
  | .behavior => "behavior"
  | .limits => "limits"
  | .policy => "policy"

structure QueryAuthoringSpan where
  sourcePath : String
  line : Nat
  column : Nat
  endLine : Nat
  endColumn : Nat
  deriving BEq, DecidableEq, Repr

structure QueryLocatedError where
  error : QueryError
  role : QueryAuthoringRole
  anchor : QueryAuthoringSpan
  deriving BEq, DecidableEq, Repr

private def quoteJson (value : String) : String :=
  Lean.Json.compress (.str value)

def canonicalQueryLocatedErrorJson
    (diagnostic : QueryLocatedError) : String :=
  "{\"error\":" ++ canonicalQueryErrorJson diagnostic.error ++
    ",\"role\":" ++ quoteJson diagnostic.role.name ++
    ",\"anchor\":{\"sourcePath\":" ++ quoteJson diagnostic.anchor.sourcePath ++
    ",\"line\":" ++ toString diagnostic.anchor.line ++
    ",\"column\":" ++ toString diagnostic.anchor.column ++
    ",\"endLine\":" ++ toString diagnostic.anchor.endLine ++
    ",\"endColumn\":" ++ toString diagnostic.anchor.endColumn ++ "}}"

private def queryAuthoringSpan
    (reference : Lean.Syntax) : Lean.Elab.Term.TermElabM QueryAuthoringSpan := do
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

private structure CapturedQuerySourceRef where
  role : QueryAuthoringRole
  definitionId : DefinitionId
  reference : Lean.Syntax
  anchor : QueryAuthoringSpan

private unsafe def evalDefinitionIdUnsafe (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM DefinitionId :=
  Lean.Meta.evalExpr DefinitionId (.const ``DefinitionId []) expression

@[implemented_by evalDefinitionIdUnsafe]
private opaque evalDefinitionId (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM DefinitionId

private unsafe def evalQueryErrorUnsafe (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM (Option QueryError) :=
  Lean.Meta.evalExpr (Option QueryError)
    (.app (.const ``Option [.zero]) (.const ``QueryError [])) expression

@[implemented_by evalQueryErrorUnsafe]
private opaque evalQueryError (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM (Option QueryError)

private def captureQuerySourceRef
    (role : QueryAuthoringRole)
    (reference : Lean.TSyntax `term) :
    Lean.Elab.Term.TermElabM CapturedQuerySourceRef := do
  let expression ← Lean.Elab.Term.elabTerm reference (some (.const ``DefinitionId []))
  let definitionId ← if expression.hasFVar || expression.hasMVar then
    pure (DefinitionId.of "umpire.authoring.local")
  else
    evalDefinitionId expression
  pure {
    role
    definitionId
    reference := reference.raw
    anchor := ← queryAuthoringSpan reference.raw
  }

private def selectRoleFallback
    (error : QueryError) : QueryAuthoringRole :=
  match error.kind with
  | .targetMismatch | .missingFiniteCompleteness | .targetKernelMismatch |
      .duplicateFiniteDomain => .target
  | .duplicateProperty | .missingProperty | .missingCapability |
      .propertyEvaluationFailure => .property
  | .invalidLimit | .unitMismatch => .limits
  | .incompatibleStrategy => .policy
  | .emptyDefinitionId | .invalidDefinitionId => .parent

private def selectQuerySourceRef
    (error : QueryError)
    (occurrences : List CapturedQuerySourceRef) :
    Option CapturedQuerySourceRef :=
  let related := occurrences.filter fun occurrence =>
    error.relatedDefinitionIds.contains occurrence.definitionId
  related.getLast? <|>
    occurrences.find? (fun occurrence => occurrence.role == selectRoleFallback error) <|>
    occurrences.find? (fun occurrence => occurrence.role == .parent)

private def elaborateQuery
    (querySyntax targetSyntax : Lean.TSyntax `term)
    (occurrences : List CapturedQuerySourceRef)
    (expectedType : Option Lean.Expr) : Lean.Elab.Term.TermElabM Lean.Expr := do
  let errorExpression ← Lean.Elab.Term.elabTerm
    (← `(Query.error? $querySyntax $targetSyntax))
    (some (.app (.const ``Option [.zero]) (.const ``QueryError [])))
  let errorExpression ← Lean.instantiateMVars errorExpression
  if errorExpression.hasFVar then
    return ← Lean.Elab.Term.elabTerm
      (← `(Query.check (.ofTarget $targetSyntax) $querySyntax)) expectedType
  match ← evalQueryError errorExpression with
  | some error =>
      match selectQuerySourceRef error occurrences with
      | some occurrence =>
          Lean.throwErrorAt occurrence.reference s!"query authoring failed: {
            canonicalQueryLocatedErrorJson {
              error, role := occurrence.role, anchor := occurrence.anchor }}"
      | none =>
          let anchor ← queryAuthoringSpan querySyntax.raw
          Lean.throwErrorAt querySyntax.raw s!"query authoring failed: {
            canonicalQueryLocatedErrorJson { error, role := .parent, anchor }}"
  | none =>
      Lean.Elab.Term.elabTerm
        (← `(Query.check (.ofTarget $targetSyntax) $querySyntax)) expectedType

declare_syntax_cat querySourceRef
syntax ident term : querySourceRef
syntax (name := checkedQuerySyntax)
  "query%" term "against" term "tracking" "[" querySourceRef,* "]" : term

elab_rules : term
  | `(query% $query against $target tracking [$occurrences,*]) => do
      let mut captured := []
      for occurrence in occurrences.getElems do
        let item ← match occurrence with
          | `(querySourceRef| $role:ident $reference) =>
              let role ← match role.getId.toString with
                | "queryParent" => pure QueryAuthoringRole.parent
                | "targetAnchor" => pure QueryAuthoringRole.target
                | "propertyAnchor" => pure QueryAuthoringRole.property
                | "scenarioAnchor" => pure QueryAuthoringRole.behavior
                | "limitsAnchor" => pure QueryAuthoringRole.limits
                | "policyAnchor" => pure QueryAuthoringRole.policy
                | _ => Lean.throwErrorAt role.raw ("expected queryParent, targetAnchor, " ++
                    "propertyAnchor, scenarioAnchor, limitsAnchor, or policyAnchor")
              captureQuerySourceRef role reference
          | _ => Lean.Elab.throwUnsupportedSyntax
        captured := captured ++ [item]
      elaborateQuery query target captured none

end Umpire
