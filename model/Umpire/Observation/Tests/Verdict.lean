import Umpire.Observation.Verdict
import Umpire.Observation.Tests.Derivation

/-! Property verdict preflight, semantic preservation, and coordinate provenance. -/

namespace Umpire.ObservationTests

open Umpire

def satisfiedVerdict : SemanticPropertyVerdict :=
  evaluateQualifiedProperty (verdictQuery [satisfiedProperty]) satisfiedProperty completeQualification

def violatedVerdict : SemanticPropertyVerdict :=
  evaluateQualifiedProperty (verdictQuery [violatedProperty]) violatedProperty completeQualification

/-- Supported qualification preserves the existing Property evaluator's Boolean result. -/
example : (satisfiedVerdict.status, violatedVerdict.status) =
    (.satisfied, .violated) := by
  native_decide

/-- Qualification failures remain unresolved and expose no clause evaluations. -/
example : [
    evaluateQualifiedProperty (verdictQuery [satisfiedProperty]) satisfiedProperty
      (.unknown (qualificationDiagnostic .compatibleAlternatives)),
    evaluateQualifiedProperty (verdictQuery [satisfiedProperty]) satisfiedProperty
      (.conflict (qualificationDiagnostic .duplicateEvidenceIdentity)),
    evaluateQualifiedProperty (verdictQuery [satisfiedProperty]) satisfiedProperty
      (.unsupported (qualificationDiagnostic .profileMismatch))
  ].map (fun verdict => (verdict.status, verdict.clauses.isEmpty)) = [
    (.unknown, true),
    (.conflict, true),
    (.unsupported, true)
  ] := by
  native_decide

/-- Exhausted bounds and non-bijective wrappers fail before Property evaluation. -/
example : [
    evaluateQualifiedProperty (verdictQuery [satisfiedProperty]) satisfiedProperty
      (.unknown {
        kind := .evidenceBoundExhausted
        planId := qualificationDeclaration.id
        limit := some { value := 1, unit := .evidenceRecords }
        observedCount := some 2
      }),
    evaluateQualifiedProperty (verdictQuery [satisfiedProperty]) satisfiedProperty
      (.qualified {
        completeQualifiedTrace with derivations := completeQualifiedTrace.derivations.tail
      })
  ].map (fun verdict => (verdict.status, verdict.clauses.isEmpty)) = [
    (.unknown, true),
    (.unknown, true)
  ] := by
  native_decide

/-- A property that requires unavailable logical time is unknown rather than violated. -/
example :
    let verdict := evaluateQualifiedProperty (verdictQuery [logicalTimeProperty])
      logicalTimeProperty completeQualification
    (verdict.status, verdict.clauses.isEmpty) = (.unknown, true) := by
  native_decide

/-- A zero-transition trace still lacks the logical-time coordinate required by the Property. -/
example :
    let qualification := qualifyFixture {
      completeEvidence with
      records := [initialEvidence]
      closures := [{ kind := eventKind, lastSequence := 1 }]
    }
    let verdict := evaluateQualifiedProperty (verdictQuery [logicalTimeProperty])
      logicalTimeProperty qualification
    (qualification.status, verdict.status,
      verdict.diagnostic.map SemanticVerdictDiagnostic.kind) =
      (.qualified, .unknown, some .missingLogicalTime) := by
  native_decide

/-- Evaluation accepts only the exact checked Property embedded in the checked Query. -/
example :
    let substituted := {
      satisfiedProperty with clauses := violatedProperty.clauses
    }
    let verdict := evaluateQualifiedProperty (verdictQuery [satisfiedProperty])
      substituted completeQualification
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
          semanticDigest := "test-observation-missing/v1"
        }]
      }
    }
    let mismatchedDigest := {
      satisfiedProperty with
      access := {
        satisfiedProperty.access with
        meanings := satisfiedProperty.access.meanings.map fun meaning =>
          if meaning.definitionId == operationState then
            { meaning with semanticDigest := "test-operation-state/mismatched" }
          else
            meaning
      }
    }
    ([missingCapability, missingVocabulary, mismatchedDigest].map fun property =>
      let verdict := evaluateQualifiedProperty (verdictQuery [property])
        property completeQualification
      (verdict.status, verdict.clauses.isEmpty)) = [
      (.unsupported, true),
      (.unsupported, true),
      (.unsupported, true)
    ] := by
  native_decide

/-- Clause evidence carries the exact query/evidence bounds and coordinate-keyed derivation. -/
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
      (clause.queryBounds, clause.evidenceBound,
        clause.derivations.map SemanticDerivation.coordinate)) = [(
      (verdictQuery [satisfiedProperty]).bounds,
      completeQualifiedTrace.appliedBound,
      [.initialState]
    )] := by
  native_decide

example : satisfiedVerdict.clauses.all fun clause => !clause.provenance.isEmpty := by
  native_decide

/-- Violated constraints retain the coordinate and derivation that explain the failure. -/
example :
    violatedVerdict.clauses.map (fun clause =>
      (clause.status, clause.coordinates,
        clause.derivations.map SemanticDerivation.coordinate)) = [(
      .violated,
      [.initialState],
      [.initialState]
    )] := by
  native_decide

/-- Conflicting duplicate vocabulary is unsupported independent of source order. -/
example :
    let original := completeQualifiedTrace.vocabulary.head?.get (by native_decide)
    let conflicting := { original with semanticDigest := original.semanticDigest ++ "/other" }
    [
      { completeQualifiedTrace with
        vocabulary := conflicting :: completeQualifiedTrace.vocabulary },
      { completeQualifiedTrace with
        vocabulary := completeQualifiedTrace.vocabulary ++ [conflicting] }
    ].map (fun trace =>
      let verdict := evaluateQualifiedProperty (verdictQuery [satisfiedProperty])
        satisfiedProperty (.qualified trace)
      (verdict.status, verdict.diagnostic.map SemanticVerdictDiagnostic.kind)) = [
      (.unsupported, some .ambiguousVocabulary),
      (.unsupported, some .ambiguousVocabulary)
    ] := by
  native_decide

/-- Repeated equal values retain distinct coordinate-linked clause provenance. -/
example :
    let qualification := qualifyFixture repeatedValueEvidence
    let verdict := evaluateQualifiedProperty (verdictQuery [repeatedProperty])
      repeatedProperty qualification
    verdict.clauses.map (fun clause => clause.coordinates) = [[
      .selectedAction 1,
      .modelOutcome 1,
      .selectedAction 2,
      .modelOutcome 2
    ]] := by
  native_decide

end Umpire.ObservationTests
