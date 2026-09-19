import Umpire.Property.Check
import Umpire.Id
import Lean.Elab.Term
import Lean.Meta.Eval

/-! Located-diagnostic elaboration of authored Properties. -/

namespace Umpire

inductive PropertyAuthoringRole where
  | parent
  | case
  | exception
  | clause
  deriving BEq, DecidableEq, Repr

def PropertyAuthoringRole.name : PropertyAuthoringRole → String
  | .parent => "parent"
  | .case => "case"
  | .exception => "exception"
  | .clause => "clause"

structure PropertyAuthoringSpan where
  sourcePath : String
  line : Nat
  column : Nat
  endLine : Nat
  endColumn : Nat
  deriving BEq, DecidableEq, Repr

structure PropertyLocatedError where
  error : PropertyError
  role : PropertyAuthoringRole
  anchor : PropertyAuthoringSpan
  deriving BEq, DecidableEq, Repr

private def quoteJson (value : String) : String :=
  Lean.Json.compress (.str value)

def canonicalPropertyLocatedErrorJson (diagnostic : PropertyLocatedError) : String :=
  "{\"error\":" ++ canonicalPropertyErrorJson diagnostic.error ++
    ",\"role\":" ++ quoteJson diagnostic.role.name ++
    ",\"anchor\":{\"sourcePath\":" ++ quoteJson diagnostic.anchor.sourcePath ++
    ",\"line\":" ++ toString diagnostic.anchor.line ++
    ",\"column\":" ++ toString diagnostic.anchor.column ++
    ",\"endLine\":" ++ toString diagnostic.anchor.endLine ++
    ",\"endColumn\":" ++ toString diagnostic.anchor.endColumn ++ "}}"

private def propertyAuthoringSpan
    (reference : Lean.Syntax) : Lean.Elab.Term.TermElabM PropertyAuthoringSpan := do
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

private structure CapturedPropertySourceRef where
  role : PropertyAuthoringRole
  definitionId : DefinitionId
  reference : Lean.Syntax
  anchor : PropertyAuthoringSpan

private unsafe def evalDefinitionIdUnsafe (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM DefinitionId :=
  Lean.Meta.evalExpr DefinitionId (.const ``DefinitionId []) expression

@[implemented_by evalDefinitionIdUnsafe]
private opaque evalDefinitionId (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM DefinitionId

private unsafe def evalPropertyUnsafe (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM Property :=
  Lean.Meta.evalExpr Property (.const ``Property []) expression

@[implemented_by evalPropertyUnsafe]
private opaque evalProperty (expression : Lean.Expr) : Lean.Elab.Term.TermElabM Property

private unsafe def evalPropertyCheckContextUnsafe (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM PropertyCheckContext :=
  Lean.Meta.evalExpr PropertyCheckContext (.const ``PropertyCheckContext []) expression

@[implemented_by evalPropertyCheckContextUnsafe]
private opaque evalPropertyCheckContext (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM PropertyCheckContext

private def capturePropertySourceRef
    (role : PropertyAuthoringRole)
    (reference : Lean.TSyntax `term) :
    Lean.Elab.Term.TermElabM CapturedPropertySourceRef := do
  let expression ← Lean.Elab.Term.elabTerm reference (some (.const ``DefinitionId []))
  let definitionId ← evalDefinitionId expression
  pure {
    role := role
    definitionId := definitionId
    reference := reference.raw
    anchor := ← propertyAuthoringSpan reference.raw }

private def selectPropertySourceRef
    (error : PropertyError)
    (occurrences : List CapturedPropertySourceRef) :
    Option CapturedPropertySourceRef :=
  let related := occurrences.filter fun occurrence =>
    error.relatedDefinitionIds.contains occurrence.definitionId
  related.getLast? <|>
    occurrences.find? (fun occurrence => occurrence.definitionId == error.definitionId) <|>
    occurrences.find? (fun occurrence => occurrence.role == .parent)

private def elaborateProperty
    (specSyntax contextSyntax : Lean.TSyntax `term)
    (occurrences : List CapturedPropertySourceRef)
    (expectedType : Option Lean.Expr) : Lean.Elab.Term.TermElabM Lean.Expr := do
  let specExpression ← Lean.Elab.Term.elabTerm specSyntax (some (.const ``Property []))
  let contextExpression ←
    Lean.Elab.Term.elabTerm contextSyntax (some (.const ``PropertyCheckContext []))
  let spec ← evalProperty specExpression
  let context ← evalPropertyCheckContext contextExpression
  match spec.check context with
  | .error error =>
      match selectPropertySourceRef error occurrences with
      | some occurrence =>
          Lean.throwErrorAt occurrence.reference s!"property authoring failed: {
            canonicalPropertyLocatedErrorJson {
              error, role := occurrence.role, anchor := occurrence.anchor }}"
      | none =>
          let anchor ← propertyAuthoringSpan specSyntax.raw
          Lean.throwErrorAt specSyntax.raw s!"property authoring failed: {
            canonicalPropertyLocatedErrorJson { error, role := .parent, anchor }}"
  | .ok _ =>
      Lean.Elab.Term.elabTerm (← `(Property.check $contextSyntax $specSyntax)) expectedType

declare_syntax_cat propertySourceRef
syntax ident term : propertySourceRef
syntax (name := checkedPropertySyntax)
  "property%" term "against" term "tracking" "[" propertySourceRef,* "]" : term

elab_rules : term
  | `(property% $spec against $context tracking [$occurrences,*]) => do
      let mut captured := []
      for occurrence in occurrences.getElems do
        let item ← match occurrence with
          | `(propertySourceRef| $role:ident $reference) =>
              let role ← match role.getId.toString with
                | "parentAnchor" => pure PropertyAuthoringRole.parent
                | "caseAnchor" => pure PropertyAuthoringRole.case
                | "exceptionAnchor" => pure PropertyAuthoringRole.exception
                | "clauseAnchor" => pure PropertyAuthoringRole.clause
                | _ => Lean.throwErrorAt role.raw "expected parentAnchor, caseAnchor, exceptionAnchor, or clauseAnchor"
              capturePropertySourceRef role reference
          | _ => Lean.Elab.throwUnsupportedSyntax
        captured := captured ++ [item]
      elaborateProperty spec context captured none

end Umpire
