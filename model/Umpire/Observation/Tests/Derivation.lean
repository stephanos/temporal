import Umpire.Observation.Tests.Qualification

/-! Stable coordinate identity and exact R3 derivation failures. -/

namespace Umpire.ObservationTests

open Umpire

/-- The qualified trace produced by the complete synthetic evidence fixture. -/
def completeQualifiedTrace : QualifiedTrace :=
  (qualifiedOf completeQualification).get (by native_decide)

/-- The first derivation in the complete qualified trace. -/
def completeFirstDerivation : SemanticDerivation :=
  completeQualifiedTrace.derivations.head?.get (by native_decide)

private def rehashQualifiedTrace (trace : QualifiedTrace) : QualifiedTrace := {
  trace with
  traceId := semanticDigestOf <|
    trace.mappingDigest ++ ":" ++ reprStr trace.evidenceIdentities ++ ":" ++ reprStr trace.trace ++
      ":" ++ reprStr trace.derivations
}

/-- Rehashed wrappers still fail when a rule's required disposition evidence is incomplete. -/
example :
    let derivations := completeQualifiedTrace.derivations.mapIdx fun index derivation =>
      if index == 0 then { derivation with appliedDispositions := derivation.appliedDispositions.tail }
      else derivation
    let mutated := rehashQualifiedTrace { completeQualifiedTrace with derivations }
    diagnosticKindOf (validateQualifiedTrace mutated) != none := by
  native_decide

def transitiveName : ObservationBinding := {
  id := id "test.binding.transitive-name"
  valueType := .text
  expression := .portable (.binding normalizedName.id)
}

def transitiveDeclaration : ObservationMappingDeclaration := {
  qualificationDeclaration with
  bindings := qualificationDeclaration.bindings ++ [transitiveName]
  rules := qualificationDeclaration.rules.map fun rule =>
    if rule.id == initialRule.id then
      { rule with value := .portable (.binding transitiveName.id) }
    else rule
}

/-- Derivations name both direct and transitive checked-binding dependencies. -/
example :
    let result := match checkObservation qualificationContext transitiveDeclaration with
      | .ok plan => qualifyEvidence plan completeEvidence
      | .error _ => .unknown {
          kind := .zeroUsableInterpretations
          planId := transitiveDeclaration.id
        }
    (qualifiedOf result).map (fun trace => trace.derivations.head?.map SemanticDerivation.bindingIds) =
      some (some [normalizedName.id, transitiveName.id]) := by
  native_decide

/-- Exact statuses and diagnostics for invalid semantic derivation fixtures. -/
def derivationFailureKinds : List (QualificationStatus × Option QualificationFailureKind) := [
  let result := validateQualifiedTrace {
    completeQualifiedTrace with derivations := completeQualifiedTrace.derivations.tail
  }
  (.unknown, diagnosticKindOf result),
  let result := validateQualifiedTrace {
    completeQualifiedTrace with
    derivations := completeFirstDerivation :: completeQualifiedTrace.derivations
  }
  (.conflict, diagnosticKindOf result),
  let result := validateQualifiedTrace {
    completeQualifiedTrace with derivations := completeQualifiedTrace.derivations ++ [{
      completeFirstDerivation with coordinate := .observation 1 99
    }]
  }
  (.conflict, diagnosticKindOf result),
  let result := validateQualifiedTrace {
    completeQualifiedTrace with trace := {
      completeQualifiedTrace.trace with initialState := {
        completeQualifiedTrace.trace.initialState with value := "tampered"
      }
    }
  }
  (.conflict, diagnosticKindOf result),
  let result := validateQualifiedTrace {
    completeQualifiedTrace with evidenceIdentities :=
      completeQualifiedTrace.evidenceIdentities ++ [id "test.evidence.record.unconsumed"]
  }
  (.unknown, diagnosticKindOf result),
  let derivations := completeQualifiedTrace.derivations.map fun derivation => {
    derivation with closureSupport := [{
        kind := eventKind
        lastSequence := 99
      }]
  }
  let result := validateQualifiedTrace {
    completeQualifiedTrace with derivations
  }
  (.unknown, diagnosticKindOf result),
  let derivations := completeQualifiedTrace.derivations.map fun derivation =>
    let recordId := derivation.evidenceIdentities.head?.getD (id "test.evidence.record.missing")
    { derivation with orderingSupport := [{
        recordId
        kind := eventKind
        sequence := 1
        causalParents := [recordId]
      }]
    }
  let result := validateQualifiedTrace {
    completeQualifiedTrace with derivations
  }
  (.unknown, diagnosticKindOf result)
]

/-- Missing, duplicate, extra, inconsistent, and unsupported derivations fail exactly. -/
example : derivationFailureKinds = [
  (.unknown, some .absentCoordinate),
  (.conflict, some .duplicateCoordinate),
  (.conflict, some .extraCoordinate),
  (.conflict, some .inconsistentDerivation),
  (.unknown, some .unconsumedReference),
  (.unknown, some .missingClosureSupport),
  (.unknown, some .missingOrderSupport)
] := by
  native_decide

/-- Closed evidence with a second step that repeats the first step's values. -/
def repeatedValueEvidence : EvidenceBundle := {
  completeEvidence with
  records := completeEvidence.records ++ [{
    stepEvidence with
    id := secondStepEvidenceId
    sequence := 3
    causalParents := [stepEvidenceId]
  }]
  closures := [{ kind := eventKind, lastSequence := 3 }]
}

/-- Equal semantic values at different slots retain distinct one-based coordinates. -/
example :
    let qualified := qualifiedOf (qualifyFixture repeatedValueEvidence)
    qualified.map (fun trace => trace.derivations.map SemanticDerivation.coordinate) = some [
      .initialState,
      .selectedAction 1,
      .modelOutcome 1,
      .resultingState 1,
      .observation 1 1,
      .observation 1 2,
      .selectedAction 2,
      .modelOutcome 2,
      .resultingState 2,
      .observation 2 1,
      .observation 2 2
    ] := by
  native_decide

end Umpire.ObservationTests
