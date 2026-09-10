import Umpire.Property.Elab
import Umpire.Property.Evaluate
import Umpire.Property.Scoped

/-! Narrow-import regression for the `Umpire.Property` public facade. -/

namespace Umpire.PropertyImportTests

#check (Umpire.Property : Type)
#check Umpire.stepClauses
#check Umpire.Property.checked
#check Umpire.canonicalPropertyLocatedErrorJson

private def escapedDiagnosticJson : String :=
  Umpire.canonicalPropertyLocatedErrorJson {
    error := {
      kind := .invalidDefinitionId
      definitionId := Umpire.DefinitionId.of "test.property"
      sourcePath := "source\npath\twith\rcontrols"
      offendingValue := "bad"
      relatedDefinitionIds := []
    }
    role := .parent
    anchor := {
      sourcePath := "anchor\npath\twith\rcontrols"
      line := 1
      column := 2
      endLine := 3
      endColumn := 4
    }
  }

#guard match Lean.Json.parse escapedDiagnosticJson with
  | .ok _ => true
  | .error _ => false

#check (Umpire.Property.check :
  Umpire.PropertyCheckContext → Umpire.Property →
    Except Umpire.PropertyError Umpire.CheckedProperty)
#check (Umpire.CheckedProperty.traceView :
  Umpire.CheckedProperty →
    Umpire.ModelTrace Umpire.ModelValue Umpire.ModelValue Umpire.ModelValue Umpire.ModelValue →
      Umpire.PropertyTraceView)
#check (Umpire.evaluateProperty :
  (property : Umpire.CheckedProperty) →
    Umpire.CheckedPropertyEvaluationInput property → Umpire.PropertyEvaluation)
#check (Umpire.evaluatePropertyOnTrace :
  (property : Umpire.CheckedProperty) →
    Umpire.ModelTrace Umpire.ModelValue Umpire.ModelValue Umpire.ModelValue Umpire.ModelValue →
      Except Umpire.PropertyError Umpire.PropertyEvaluation)
#check (Umpire.checkPropertyEvaluationInput :
  (property : Umpire.CheckedProperty) →
    Umpire.ModelTrace Umpire.ModelValue Umpire.ModelValue Umpire.ModelValue Umpire.ModelValue →
      Except Umpire.PropertyError (Umpire.CheckedPropertyEvaluationInput property))
#check Umpire.evaluatePropertyClause_agrees
#check (Umpire.evaluatePropertyClause :
  (property : Umpire.CheckedProperty) →
    (input : Umpire.CheckedPropertyEvaluationInput property) →
    { clause // clause ∈ property.clauses } → Bool)
#check (Umpire.evaluateProperty_agrees :
  ∀ (property : Umpire.CheckedProperty) (input : Umpire.CheckedPropertyEvaluationInput property),
    (Umpire.evaluateProperty property input).satisfied = true ↔ property.denote input)
#check (Umpire.checkPropertyPredicate :
  (context : Umpire.PropertyCheckContext) → (owner : Umpire.Property) →
    (contextKind : Umpire.PropertyPredicateContext) → Umpire.PropertyPredicate →
      Except Umpire.PropertyError (Umpire.CheckedPropertyPredicate contextKind))
#check (Umpire.checkPropertyPredicateInput :
  ∀ {contextKind : Umpire.PropertyPredicateContext},
    (predicate : Umpire.CheckedPropertyPredicate contextKind) → Umpire.PropertyPredicateInput →
      Except Umpire.PropertyError (Umpire.CheckedPropertyPredicateInput predicate))
#check (Umpire.evaluatePropertyPredicate :
  ∀ {contextKind : Umpire.PropertyPredicateContext},
    (predicate : Umpire.CheckedPropertyPredicate contextKind) →
      Umpire.CheckedPropertyPredicateInput predicate → Bool)
#check (Umpire.evaluatePropertyPredicate_agrees :
  ∀ {contextKind : Umpire.PropertyPredicateContext}
    (predicate : Umpire.CheckedPropertyPredicate contextKind)
    (input : Umpire.CheckedPropertyPredicateInput predicate),
      Umpire.evaluatePropertyPredicate predicate input = true ↔ predicate.denote input)
#check (Umpire.CheckedPropertyTemporalClause : Type)
#check (Umpire.PropertyTemporalClause : Type)
#check (Umpire.CheckedProperty.hasGuardedTemporalClauses : Umpire.CheckedProperty → Bool)
#check (Umpire.CheckedProperty.guardedTemporalClauseIds :
  Umpire.CheckedProperty → List Umpire.DefinitionId)
#check (Umpire.CheckedProperty.guardedClauseIds :
  Umpire.CheckedProperty → List Umpire.DefinitionId)
#check (Umpire.JointObligationObservation : Type)
#check (Umpire.JointTriggerOccurrence : Type)
#check (Umpire.analyzeOverlapObligations :
  (property : Umpire.CheckedProperty) →
    Umpire.CheckedPropertyEvaluationInput property →
      List Umpire.JointObligationObservation)

#guard_msgs (error, substring := true) in
#check Umpire.PropertyPredicate.evaluate

#guard_msgs (error, substring := true) in
#check Umpire.PropertyPredicate.denote

#guard_msgs (error, substring := true) in
#check Umpire.PropertyAtom.evaluate

#guard_msgs (error, substring := true) in
#check Umpire.PropertyAtom.denote

#guard_msgs (error, substring := true) in
#check Umpire.evaluateResolvedPropertyClause

#guard_msgs (error, substring := true) in
#check Umpire.resolvedPropertyClauseDenotes

#guard_msgs (error, substring := true) in
def crossPropertyEvaluationInput
    (first second : Umpire.CheckedProperty)
    (input : Umpire.CheckedPropertyEvaluationInput first) : Umpire.PropertyEvaluation :=
  Umpire.evaluateProperty second input

#guard_msgs (error, substring := true) in
def forgedPropertyEvaluationInput
    (property : Umpire.CheckedProperty)
    (view : Umpire.PropertyTraceView) : Umpire.CheckedPropertyEvaluationInput property := {
  view
}

#guard_msgs (error, substring := true) in
#check Umpire.Scenario

#guard_msgs (error, substring := true) in
#check Umpire.QueryDeclaration

end Umpire.PropertyImportTests
