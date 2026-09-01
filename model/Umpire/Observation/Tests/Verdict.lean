import Umpire.Observation.Verdict
import Umpire.Observation.Tests.EvidenceLink

/-! Property verdict preflight, semantic preservation, and coordinate provenance. -/

namespace Umpire.ObservationTests

open Umpire

def satisfiedVerdict : SemanticPropertyVerdict :=
  evaluateObservationProperty (verdictQuery [satisfiedProperty]) satisfiedProperty completeEvaluation

def violatedVerdict : SemanticPropertyVerdict :=
  evaluateObservationProperty (verdictQuery [violatedProperty]) violatedProperty completeEvaluation

/-- Supported evaluation preserves the existing Property evaluator's Boolean result. -/
example : (satisfiedVerdict.status, violatedVerdict.status) =
    (.satisfied, .violated) := by
  native_decide

/-- Evaluation failures remain unresolved and expose no clause evaluations. -/
example : [
    evaluateObservationProperty (verdictQuery [satisfiedProperty]) satisfiedProperty
      (.unknown (evaluationDiagnostic .compatibleAlternatives)),
    evaluateObservationProperty (verdictQuery [satisfiedProperty]) satisfiedProperty
      (.conflict (evaluationDiagnostic .duplicateEvidenceIdentity)),
    evaluateObservationProperty (verdictQuery [satisfiedProperty]) satisfiedProperty
      (.unsupported (evaluationDiagnostic .profileMismatch))
  ].map (fun verdict => (verdict.status, verdict.clauses.isEmpty)) = [
    (.unknown, true),
    (.conflict, true),
    (.unsupported, true)
  ] := by
  native_decide

/-- Exhausted limits and non-bijective wrappers fail before Property evaluation. -/
example : [
    evaluateObservationProperty (verdictQuery [satisfiedProperty]) satisfiedProperty
      (.unknown {
        kind := .evidenceBoundExhausted
        planId := evaluationDeclaration.id
        limit := some { value := 1, unit := .evidenceRecords }
        observedCount := some 2
      }),
    evaluateObservationProperty (verdictQuery [satisfiedProperty]) satisfiedProperty
      (observationResultOfAdmission <| validateEvidenceBackedTrace {
        completeUncheckedEvidenceBackedTrace with
        evidenceLinks := completeUncheckedEvidenceBackedTrace.evidenceLinks.tail
      })
  ].map (fun verdict => (verdict.status, verdict.clauses.isEmpty)) = [
    (.unknown, true),
    (.unknown, true)
  ] := by
  native_decide

/-- A property that requires unavailable logical time is unknown rather than violated. -/
example :
    let verdict := evaluateObservationProperty (verdictQuery [logicalTimeProperty])
      logicalTimeProperty completeEvaluation
    (verdict.status, verdict.clauses.isEmpty) = (.unknown, true) := by
  native_decide

/-- A zero-transition trace still lacks the logical-time coordinate required by the Property. -/
example :
    let evaluation := evaluateFixture {
      completeEvidence with
      records := [initialEvidence]
      closures := [{ kind := eventKind, lastSequence := 1 }]
    }
    let verdict := evaluateObservationProperty (verdictQuery [logicalTimeProperty])
      logicalTimeProperty evaluation
    (evaluation.status, verdict.status,
      verdict.diagnostic.map SemanticVerdictDiagnostic.kind) =
      (.accepted, .unknown, some .missingLogicalTime) := by
  native_decide

/-- Evaluation accepts only the exact checked Property embedded in the checked Query. -/
example :
    let substituted := {
      satisfiedProperty with clauses := violatedProperty.clauses
    }
    let verdict := evaluateObservationProperty (verdictQuery [satisfiedProperty])
      substituted completeEvaluation
    (verdict.status, verdict.clauses.isEmpty,
      verdict.diagnostic.map SemanticVerdictDiagnostic.kind) =
      (.unsupported, true, some .queryPropertyMismatch) := by
  native_decide

/-- Capability, vocabulary, and meaning-digest mismatches are unsupported before evaluation. -/
example :
    let missingCapability := {
      satisfiedProperty with
      requires := satisfiedProperty.requires ++ [id "test.capability.observation.missing"]
    }
    let missingVocabulary := {
      satisfiedProperty with
      access := {
        satisfiedProperty.access with
        meanings := satisfiedProperty.access.meanings ++ [{
          definitionId := id "test.observation.missing"
          kind := .observation
          canonicalBehavior := "test-observation-missing/v1"
        }]
      }
    }
    let mismatchedDigest := {
      satisfiedProperty with
      access := {
        satisfiedProperty.access with
        meanings := satisfiedProperty.access.meanings.map fun meaning =>
          if meaning.definitionId == operationState then
            { meaning with canonicalBehavior := "test-operation-state/mismatched" }
          else
            meaning
      }
    }
    ([missingCapability, missingVocabulary, mismatchedDigest].map fun property =>
      let verdict := evaluateObservationProperty (verdictQuery [property])
        property completeEvaluation
      (verdict.status, verdict.clauses.isEmpty)) = [
      (.unsupported, true),
      (.unsupported, true),
      (.unsupported, true)
    ] := by
  native_decide

/-- Clause evidence carries the exact query/evidence limits and coordinate-keyed Evidence Link. -/
example :
    (satisfiedVerdict.clauses.map fun clause =>
      (clause.clauseId, clause.status, clause.coordinates)) = [(
      id "test.property.observation.satisfied.initial",
      .satisfied,
      [.initialState]
    )] := by
  native_decide

example :
    satisfiedVerdict.clauses.map (fun clause =>
      (clause.queryLimits, clause.evidenceBound,
        clause.evidenceLinks.map EvidenceLink.coordinate)) = [(
      (verdictQuery [satisfiedProperty]).limits,
      completeEvidenceBackedTrace.appliedBound,
      [.initialState]
    )] := by
  native_decide

example : satisfiedVerdict.clauses.all fun clause => !clause.provenance.isEmpty := by
  native_decide

/-- Violated constraints retain the Model Coordinate and Evidence Link that explain the failure. -/
example :
    violatedVerdict.clauses.map (fun clause =>
      (clause.status, clause.coordinates,
        clause.evidenceLinks.map EvidenceLink.coordinate)) = [(
      .violated,
      [.initialState],
      [.initialState]
    )] := by
  native_decide

/-- Conflicting duplicate vocabulary fails at admission independent of source order. -/
example :
    let original := completeUncheckedEvidenceBackedTrace.vocabulary.head?.get (by native_decide)
    let conflicting := { original with canonicalBehavior := original.canonicalBehavior ++ "/other" }
    [
      { completeUncheckedEvidenceBackedTrace with
        vocabulary := conflicting :: completeUncheckedEvidenceBackedTrace.vocabulary },
      { completeUncheckedEvidenceBackedTrace with
        vocabulary := completeUncheckedEvidenceBackedTrace.vocabulary ++ [conflicting] }
    ].map (fun trace =>
      let observationResult := observationResultOfAdmission (validateEvidenceBackedTrace trace)
      let verdict := evaluateObservationProperty (verdictQuery [satisfiedProperty])
        satisfiedProperty observationResult
      (verdict.status, verdict.diagnostic.map SemanticVerdictDiagnostic.kind)) = [
      (.conflict, some (.observationEvaluationFailure .inconsistentEvidenceLink)),
      (.conflict, some (.observationEvaluationFailure .inconsistentEvidenceLink))
    ] := by
  native_decide

/-- Repeated equal values retain distinct coordinate-linked clause provenance. -/
example :
    let evaluation := evaluateFixture repeatedValueEvidence
    let verdict := evaluateObservationProperty (verdictQuery [repeatedProperty])
      repeatedProperty evaluation
    verdict.clauses.map (fun clause => clause.coordinates) = [[
      .selectedAction 1,
      .modelOutcome 1,
      .selectedAction 2,
      .modelOutcome 2
    ]] := by
  native_decide

/-- Property field compatibility rejects invalid coordinates and preserves prior-state edges. -/
example :
    let oneStepTrace := completeEvidenceBackedTrace.trace
    let twoStepTrace := (acceptedOf (evaluateFixture repeatedValueEvidence)).get (by native_decide)
      |>.trace
    let emptyTrace := { oneStepTrace with steps := [] }
    [
      (PropertyTraceField.valueAt? .state emptyTrace .initialState).isSome,
      (PropertyTraceField.valueAt? .priorState emptyTrace .initialState).isSome,
      (PropertyTraceField.valueAt? .resultingState emptyTrace .initialState).isSome,
      (PropertyTraceField.valueAt? .priorState oneStepTrace .initialState).isSome,
      (PropertyTraceField.valueAt? .priorState oneStepTrace (.resultingState 1)).isSome,
      (PropertyTraceField.valueAt? .resultingState oneStepTrace (.resultingState 1)).isSome,
      (PropertyTraceField.valueAt? .priorState twoStepTrace (.resultingState 1)).isSome,
      (PropertyTraceField.valueAt? .priorState twoStepTrace (.resultingState 2)).isSome,
      (PropertyTraceField.valueAt? .observation twoStepTrace (.observation 1 1)).isSome,
      (PropertyTraceField.valueAt? .relation twoStepTrace (.observation 1 1)).isSome,
      (PropertyTraceField.valueAt? .selectedAction twoStepTrace (.selectedAction 0)).isSome,
      (PropertyTraceField.valueAt? .modelOutcome twoStepTrace (.modelOutcome 0)).isSome,
      (PropertyTraceField.valueAt? .state twoStepTrace (.resultingState 0)).isSome,
      (PropertyTraceField.valueAt? .observation twoStepTrace (.observation 0 1)).isSome,
      (PropertyTraceField.valueAt? .relation twoStepTrace (.observation 1 0)).isSome,
      (PropertyTraceField.valueAt? .selectedAction twoStepTrace (.selectedAction 3)).isSome,
      (PropertyTraceField.valueAt? .observation twoStepTrace (.observation 1 99)).isSome
    ] = [
      true, false, false,
      true, false, true,
      true, false,
      true, true,
      false, false, false, false, false, false, false
    ] := by
  native_decide

end Umpire.ObservationTests
