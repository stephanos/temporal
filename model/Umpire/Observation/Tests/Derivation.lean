import Umpire.Observation.Tests.Qualification

/-! Stable coordinate identity and exact R3 derivation failures. -/

namespace Umpire.ObservationTests

open Umpire

def completeQualifiedTrace : QualifiedTrace := (qualifiedOf completeQualification).get!

def derivationFailureKinds : List (QualificationStatus × Option QualificationFailureKind) := [
  let result := validateQualifiedTrace {
    completeQualifiedTrace with derivations := completeQualifiedTrace.derivations.tail
  }
  (.unknown, diagnosticKindOf result),
  let result := validateQualifiedTrace {
    completeQualifiedTrace with
    derivations := completeQualifiedTrace.derivations.head! :: completeQualifiedTrace.derivations
  }
  (.conflict, diagnosticKindOf result),
  let result := validateQualifiedTrace {
    completeQualifiedTrace with derivations := completeQualifiedTrace.derivations ++ [{
      completeQualifiedTrace.derivations.head! with coordinate := .observation 1 99
    }]
  }
  (.conflict, diagnosticKindOf result),
  let result := validateQualifiedTrace {
    completeQualifiedTrace with derivations := [{
      completeQualifiedTrace.derivations.head! with mappingVersion := 99
    }] ++ completeQualifiedTrace.derivations.tail
  }
  (.conflict, diagnosticKindOf result),
  let result := validateQualifiedTrace {
    completeQualifiedTrace with evidenceIdentities :=
      completeQualifiedTrace.evidenceIdentities ++ [id "test.evidence.record.unconsumed"]
  }
  (.unknown, diagnosticKindOf result),
  let result := validateQualifiedTrace {
    completeQualifiedTrace with derivations := [{
      completeQualifiedTrace.derivations.head! with closureSupport := []
    }] ++ completeQualifiedTrace.derivations.tail
  }
  (.unknown, diagnosticKindOf result),
  let result := validateQualifiedTrace {
    completeQualifiedTrace with derivations := [{
      completeQualifiedTrace.derivations.head! with orderingSupport := []
    }] ++ completeQualifiedTrace.derivations.tail
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
