import Umpire.Query.Language
import Umpire.Id
import Lean.Elab.Term
import Lean.Meta.Eval

/-! Explicit bounded Query construction over the existing checked Query language. -/

namespace Umpire

/-- Named stage values prevent unit omission while retaining the checked Limits verbatim. -/
structure QueryLimitSpec where
  transitions : Nat
  selectedActions : Nat
  candidateEvaluations : Nat
  deriving BEq, DecidableEq, Repr

def QueryLimitSpec.toQueryLimits (spec : QueryLimitSpec) : QueryLimits :=
  QueryLimits.bounded spec.transitions spec.selectedActions spec.candidateEvaluations

structure QuerySpec where
  family : DefinitionFamily
  key : String
  source : SourceLocation
  target : DefinitionId
  form : QueryForm
  behavior : CheckedScenario
  limits : QueryLimitSpec
  policy : PlannerPolicy
  endpoint : QueryEndpoint := .final
  exercise : QueryExercisePolicy := .allowVacuous
  authoredKnownGaps : KnownGapSet := KnownGapSet.empty
  documentation : String := ""

def QuerySpec.declaration (spec : QuerySpec) : QueryDeclaration := {
  id := spec.family.id "query" spec.key
  source := spec.source
  target := spec.target
  form := spec.form
  behavior := spec.behavior
  limits := spec.limits.toQueryLimits
  policy := spec.policy
  endpoint := spec.endpoint
  exercise := spec.exercise
  authoredKnownGaps := spec.authoredKnownGaps
  documentation := spec.documentation
}

/-- Admit a constructor-authored Query only through the existing language checker. -/
def QuerySpec.check
    (spec : QuerySpec)
    (target : QueryModel LawStatement) : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery (.ofTarget target) spec.declaration

/-- Produce the checked value after the kernel verifies that the existing checker succeeds. -/
def QuerySpec.checked
    (spec : QuerySpec)
    (target : QueryModel LawStatement)
    (valid : (spec.check target).toOption.isSome = true) : CheckedQuery LawStatement :=
  checkedQuery target spec.declaration valid

def QuerySpec.error?
    (spec : QuerySpec)
    (target : QueryModel LawStatement) : Option QueryError :=
  match spec.check target with
  | .error error => some error
  | .ok _ => none

/-- A successful-branch Query input keeps Target extraction explicit without trusting it as a proof. -/
structure QueryAuthoringInput (LawStatement : Law → Prop) where
  declaration : QueryDeclaration
  target : QueryModel LawStatement

def QueryAuthoringInput.ofSpec
    (spec : QuerySpec)
    (target : QueryModel LawStatement) : QueryAuthoringInput LawStatement := {
  declaration := spec.declaration
  target
}

def QueryAuthoringInput.check
    (input : QueryAuthoringInput LawStatement) : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery (.ofTarget input.target) input.declaration

def QueryAuthoringInput.error?
    (input : Option (QueryAuthoringInput LawStatement)) : Option QueryError := do
  let input ← input
  match input.check with
  | .error error => some error
  | .ok _ => none

def QueryAuthoringInput.check?
    (input : Option (QueryAuthoringInput LawStatement)) :
    Option (Except QueryError (CheckedQuery LawStatement)) :=
  input.map QueryAuthoringInput.check

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
    (specSyntax targetSyntax : Lean.TSyntax `term)
    (occurrences : List CapturedQuerySourceRef)
    (expectedType : Option Lean.Expr) : Lean.Elab.Term.TermElabM Lean.Expr := do
  let errorExpression ← Lean.Elab.Term.elabTerm
    (← `(QuerySpec.error? $specSyntax $targetSyntax))
    (some (.app (.const ``Option [.zero]) (.const ``QueryError [])))
  let errorExpression ← Lean.instantiateMVars errorExpression
  if errorExpression.hasFVar then
    return ← Lean.Elab.Term.elabTerm
      (← `(QuerySpec.check $specSyntax $targetSyntax)) expectedType
  match ← evalQueryError errorExpression with
  | some error =>
      match selectQuerySourceRef error occurrences with
      | some occurrence =>
          Lean.throwErrorAt occurrence.reference s!"query authoring failed: {
            canonicalQueryLocatedErrorJson {
              error, role := occurrence.role, anchor := occurrence.anchor }}"
      | none =>
          let anchor ← queryAuthoringSpan specSyntax.raw
          Lean.throwErrorAt specSyntax.raw s!"query authoring failed: {
            canonicalQueryLocatedErrorJson { error, role := .parent, anchor }}"
  | none =>
      Lean.Elab.Term.elabTerm (← `(QuerySpec.check $specSyntax $targetSyntax)) expectedType

declare_syntax_cat querySourceRef
syntax ident term : querySourceRef
syntax (name := checkedQuerySyntax)
  "query%" term "against" term "tracking" "[" querySourceRef,* "]" : term

elab_rules : term
  | `(query% $spec against $target tracking [$occurrences,*]) => do
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
      elaborateQuery spec target captured none

private def elaborateQueryInput
    (inputSyntax : Lean.TSyntax `term)
    (occurrences : List CapturedQuerySourceRef)
    (expectedType : Option Lean.Expr) : Lean.Elab.Term.TermElabM Lean.Expr := do
  let errorExpression ← Lean.Elab.Term.elabTerm
    (← `(QueryAuthoringInput.error? $inputSyntax))
    (some (.app (.const ``Option [.zero]) (.const ``QueryError [])))
  let errorExpression ← Lean.instantiateMVars errorExpression
  if errorExpression.hasFVar then
    return ← Lean.Elab.Term.elabTerm
      (← `(QueryAuthoringInput.check? $inputSyntax)) expectedType
  match ← evalQueryError errorExpression with
  | some error =>
      match selectQuerySourceRef error occurrences with
      | some occurrence =>
          Lean.throwErrorAt occurrence.reference s!"query authoring failed: {
            canonicalQueryLocatedErrorJson {
              error, role := occurrence.role, anchor := occurrence.anchor }}"
      | none =>
          let anchor ← queryAuthoringSpan inputSyntax.raw
          Lean.throwErrorAt inputSyntax.raw s!"query authoring failed: {
            canonicalQueryLocatedErrorJson { error, role := .parent, anchor }}"
  | none =>
      Lean.Elab.Term.elabTerm (← `(QueryAuthoringInput.check? $inputSyntax)) expectedType

syntax (name := checkedQueryInputSyntax)
  "query%" term "tracking" "[" querySourceRef,* "]" : term

elab_rules : term
  | `(query% $input tracking [$occurrences,*]) => do
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
      elaborateQueryInput input captured none

end Umpire
