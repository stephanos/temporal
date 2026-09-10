import Umpire.Property.Check
import Umpire.Id
import Lean.Elab.Term
import Lean.Meta.Eval

/-! Narrow ordinary-Lean constructors over the existing checked Property language. -/

namespace Umpire

namespace PropertyPattern

def selectedAction (value : ModelValue) : PropertyPattern :=
  .exact .selectedAction value.definitionId value.value

def resultingState (value : ModelValue) : PropertyPattern :=
  .exact .resultingState value.definitionId value.value

def modelOutcome (value : ModelValue) : PropertyPattern :=
  .exact .modelOutcome value.definitionId value.value

def fact (value : ModelValue) : PropertyPattern :=
  .exact .observation value.definitionId value.value

end PropertyPattern

namespace PropertyPredicate

/-- Author an independent field relationship in the existing closed Boolean language. -/
def compareFields (operator : PropertyFieldOperator) (left right : PropertyFieldOperand)
    (source : SourceLocation) : PropertyPredicate := .atom {
  field := .selectedAction
  reference := .of "umpire.property.fields"
  constraint := .fields ⟨operator, left, right, source⟩ }

/-- Surface spelling elaborates to exactly the ordinary typed field comparison constructor. -/
syntax "field_compare%" term:max "with" term:max term:max "at" term:max : term

macro_rules
  | `(field_compare% $left with $operator $right at $source) =>
    `(PropertyPredicate.compareFields $operator $left $right $source)

def priorStateIs (value : ModelValue) : PropertyPredicate :=
  .atom {
    field := .priorState
    reference := value.definitionId
    constraint := .equals (.text value.value)
  }

def selectedActionIs (value : ModelValue) : PropertyPredicate :=
  .atom {
    field := .selectedAction
    reference := value.definitionId
    constraint := .equals (.text value.value)
  }

def resultingStateIs (value : ModelValue) : PropertyPredicate :=
  .atom {
    field := .resultingState
    reference := value.definitionId
    constraint := .equals (.text value.value)
  }

def modelOutcomeIs (value : ModelValue) : PropertyPredicate :=
  .atom {
    field := .modelOutcome
    reference := value.definitionId
    constraint := .equals (.text value.value)
  }

def factIs (value : ModelValue) : PropertyPredicate :=
  .atom {
    field := .expectationFact
    reference := value.definitionId
    constraint := .equals (.text value.value)
  }

end PropertyPredicate

/-- Build the three independent baseline obligations for one Target-owned transition result. -/
def stepClauses
    (family : DefinitionFamily)
    (propertyKey : String)
    (action state outcome fact : ModelValue) : List PropertyClause := [
  .transitionContract (family.id "property" (propertyKey ++ ".state"))
    (.selectedAction action) (.resultingState state),
  .transitionContract (family.id "property" (propertyKey ++ ".outcome"))
    (.selectedAction action) (.modelOutcome outcome),
  .inputOutput (family.id "property" (propertyKey ++ ".fact"))
    (.selectedAction action) (.fact fact)
]

/-- Readable bounded response clauses elaborate directly to the typed declaration. Admission,
reference resolution, and canonicalization remain owned by `property%` and `checkProperty`. -/
syntax (name := boundedResponseSyntax)
  "bounded_response%" term:max "at" term:max &"whenever" term:max &"eventually" term:max
  &"within" term:max &"on" term:max "scoped" term:max "by" term:max &"closing" term:max : term

macro_rules
  | `(bounded_response% $id at $source whenever $trigger eventually $response
      within $bound on $clock scoped $scope by $key closing $endpoint) =>
      `(({ id := $id, source := $source, trigger := $trigger, response := $response,
           bound := $bound, clock := $clock, scope := $scope, key := $key,
           endpoint := $endpoint } : PropertyScopedClause))

structure PropertySpec where
  family : DefinitionFamily
  key : String
  source : SourceLocation
  version : Nat := 1
  requires : List DefinitionId
  clauses : List PropertyClause
  scopedClauses : List PropertyScopedClause := []
  logicalTimeSource : Option DefinitionId := none
  documentation : String := ""

def PropertySpec.declaration (spec : PropertySpec) : PropertyDeclaration := {
  id := spec.family.id "property" spec.key
  source := spec.source
  version := spec.version
  requires := spec.requires
  clauses := spec.clauses
  scopedClauses := spec.scopedClauses
  logicalTimeSource := spec.logicalTimeSource
  documentation := spec.documentation
}

/-- Admit a constructor-authored Property only through the existing language checker. -/
def PropertySpec.check
    (spec : PropertySpec)
    (context : PropertyCheckContext) : Except PropertyError CheckedProperty :=
  checkProperty context (.portable spec.declaration)

/-- Produce the checked value after the kernel verifies that the existing checker succeeds. -/
def PropertySpec.checked
    (spec : PropertySpec)
    (context : PropertyCheckContext)
    (valid : (spec.check context).toOption.isSome = true) : CheckedProperty :=
  checkedProperty context (.portable spec.declaration) valid

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

def canonicalPropertyAuthoringDiagnosticJson (diagnostic : PropertyLocatedError) : String :=
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

private unsafe def evalPropertySpecUnsafe (expression : Lean.Expr) :
    Lean.Elab.Term.TermElabM PropertySpec :=
  Lean.Meta.evalExpr PropertySpec (.const ``PropertySpec []) expression

@[implemented_by evalPropertySpecUnsafe]
private opaque evalPropertySpec (expression : Lean.Expr) : Lean.Elab.Term.TermElabM PropertySpec

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
  let specExpression ← Lean.Elab.Term.elabTerm specSyntax (some (.const ``PropertySpec []))
  let contextExpression ←
    Lean.Elab.Term.elabTerm contextSyntax (some (.const ``PropertyCheckContext []))
  let spec ← evalPropertySpec specExpression
  let context ← evalPropertyCheckContext contextExpression
  match spec.check context with
  | .error error =>
      match selectPropertySourceRef error occurrences with
      | some occurrence =>
          Lean.throwErrorAt occurrence.reference s!"property authoring failed: {
            canonicalPropertyAuthoringDiagnosticJson {
              error, role := occurrence.role, anchor := occurrence.anchor }}"
      | none =>
          let anchor ← propertyAuthoringSpan specSyntax.raw
          Lean.throwErrorAt specSyntax.raw s!"property authoring failed: {
            canonicalPropertyAuthoringDiagnosticJson { error, role := .parent, anchor }}"
  | .ok _ =>
      Lean.Elab.Term.elabTerm (← `(PropertySpec.check $specSyntax $contextSyntax)) expectedType

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
