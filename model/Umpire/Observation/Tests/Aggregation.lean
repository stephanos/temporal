import Umpire.Observation.Tests.Verdict

/-! Strict query aggregation preserves every result and fails closed. -/

namespace Umpire.ObservationTests

open Umpire

def aggregationQuery : CheckedQuery Umpire.Examples.Switch.LawStatement :=
  verdictQuery [satisfiedProperty, violatedProperty]

def verdictAs
    (propertyId : DefinitionId)
    (status : SemanticVerdictStatus)
    (traceId : Option String := some completeEvidenceBackedTrace.traceId) : SemanticPropertyVerdict := {
  satisfiedVerdict with
  queryId := aggregationQuery.id
  propertyId
  propertyDigest := (aggregationQuery.form.properties.find? fun property =>
    property.id == propertyId).map (fun property => property.behaviorFingerprint.render)
      |>.getD "unexpected"
  traceId
  status
}

def aggregateStatus (verdicts : List SemanticPropertyVerdict) : StrictQueryStatus :=
  (summarizeQueryVerdicts aggregationQuery verdicts).status

/-- Complete resolved inputs distinguish all-satisfied from at-least-one-violation. -/
example : [
    aggregateStatus [
      verdictAs satisfiedProperty.id .satisfied,
      verdictAs violatedProperty.id .satisfied
    ],
    aggregateStatus [
      verdictAs satisfiedProperty.id .satisfied,
      verdictAs violatedProperty.id .violated
    ]
  ] = [.satisfied, .violated] := by
  native_decide

/-- Every unresolved result remains inspectable and makes the summary incomplete. -/
example : [
    aggregateStatus [
      verdictAs satisfiedProperty.id .unknown,
      verdictAs violatedProperty.id .satisfied
    ],
    aggregateStatus [
      verdictAs satisfiedProperty.id .conflict,
      verdictAs violatedProperty.id .satisfied
    ],
    aggregateStatus [
      verdictAs satisfiedProperty.id .unsupported,
      verdictAs violatedProperty.id .satisfied
    ],
    aggregateStatus [
      verdictAs satisfiedProperty.id .violated,
      verdictAs violatedProperty.id .unknown
    ]
  ] = [.incomplete, .incomplete, .incomplete, .incomplete] := by
  native_decide

/-- Missing, duplicate, unexpected, and divergent inputs can never aggregate to success. -/
example : [
    aggregateStatus [verdictAs satisfiedProperty.id .satisfied],
    aggregateStatus [
      verdictAs satisfiedProperty.id .satisfied,
      verdictAs satisfiedProperty.id .satisfied,
      verdictAs violatedProperty.id .satisfied
    ],
    aggregateStatus [
      verdictAs satisfiedProperty.id .satisfied,
      verdictAs violatedProperty.id .satisfied,
      verdictAs (id "test.property.observation.unexpected") .satisfied
    ],
    aggregateStatus [
      verdictAs satisfiedProperty.id .satisfied (some "trace-a"),
      verdictAs violatedProperty.id .satisfied (some "trace-b")
    ]
  ] = [.incomplete, .incomplete, .incomplete, .incomplete] := by
  native_decide

/-- Strict summaries retain all inputs and identify each structural defect deterministically. -/
example :
    let unexpected := id "test.property.observation.unexpected"
    let verdicts := [
      verdictAs satisfiedProperty.id .satisfied (some "trace-b"),
      verdictAs satisfiedProperty.id .satisfied (some "trace-a"),
      verdictAs unexpected .satisfied (some "trace-a")
    ]
    let summary := summarizeQueryVerdicts aggregationQuery verdicts
    (summary.verdicts.length, summary.missingProperties, summary.duplicateProperties,
      summary.unexpectedProperties, summary.traceIds) =
      (3, [violatedProperty.id], [satisfiedProperty.id], [unexpected], ["trace-a", "trace-b"]) := by
  native_decide

/-- Canonical aggregation is independent of result source order. -/
example :
    let verdicts := [
      verdictAs violatedProperty.id .violated,
      verdictAs satisfiedProperty.id .satisfied
    ]
    summarizeQueryVerdicts aggregationQuery verdicts =
      summarizeQueryVerdicts aggregationQuery verdicts.reverse := by
  native_decide

/-- Canonical ordering breaks ties across every retained verdict field. -/
example :
    let first := verdictAs satisfiedProperty.id .satisfied
    let second := { first with queryId := id "test.query.other" }
    summarizeQueryVerdicts aggregationQuery [first, second] =
      summarizeQueryVerdicts aggregationQuery [second, first] := by
  native_decide

/-- A result with divergent Property semantics cannot satisfy the checked Query. -/
example :
    let forged := {
      verdictAs violatedProperty.id .satisfied with propertyDigest := "property/other"
    }
    let summary := summarizeQueryVerdicts aggregationQuery [
      verdictAs satisfiedProperty.id .satisfied,
      forged
    ]
    (summary.status, summary.divergentProperties) =
      (.incomplete, [violatedProperty.id]) := by
  native_decide

end Umpire.ObservationTests
