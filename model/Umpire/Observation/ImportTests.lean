import Umpire.Observation

/-! Focused import contract for Observation authoring, checking, and evaluation. -/

#check Umpire.ObservationMappingDeclaration
#check Umpire.ObservationFieldSpec
#check Umpire.ObservationFieldSpec.declaration
#check Umpire.ObservationFieldSpec.reference
#check Umpire.ObservationFieldSpec.expression
#check Umpire.ObservationFieldSpec.disposition
#check Umpire.ObservationCheckContext
#check Umpire.ObservationCheckContext.ofTarget
#check Umpire.ObservationErrorKind
#check Umpire.ObservationError
#check Umpire.CheckedObservationExpression
#check Umpire.CheckedObservationPlan
#check Umpire.canonicalObservationPlanJson
#check Umpire.canonicalObservationErrorJson
#check Umpire.checkObservation
#check Umpire.checkedObservation
#check Umpire.EvidenceBound
#check Umpire.EvidenceValue.text
#check Umpire.EvidenceValue.valueType
#check Umpire.EvidenceValue.render
#check Umpire.EvidenceBundle
#check Umpire.EvidenceBundle.profile
#check Umpire.ObservationStatus
#check Umpire.ObservationFailureKind
#check Umpire.ObservationFailureKind.status
#check Umpire.ObservationDiagnostic
#check Umpire.ObservationDiagnostic.status
#check Umpire.ModelCoordinate
#check Umpire.EvidenceLink
#check Umpire.EvidenceLink.coordinate
#check Umpire.UncheckedEvidenceBackedTrace
#synth BEq Umpire.UncheckedEvidenceBackedTrace
#synth DecidableEq Umpire.UncheckedEvidenceBackedTrace
#synth Repr Umpire.UncheckedEvidenceBackedTrace
#synth BEq Umpire.EvidenceBundle
#synth DecidableEq Umpire.EvidenceBundle
#synth Repr Umpire.EvidenceBundle
#synth BEq Umpire.ObservationDiagnostic
#synth DecidableEq Umpire.ObservationDiagnostic
#synth Repr Umpire.ObservationDiagnostic
#check Umpire.EvidenceBackedTrace
#check Umpire.EvidenceBackedTrace.traceId
#check Umpire.EvidenceBackedTrace.checkedPlan
#check Umpire.EvidenceBackedTrace.mappingId
#check Umpire.EvidenceBackedTrace.mappingVersion
#check Umpire.EvidenceBackedTrace.mappingDigest
#check Umpire.EvidenceBackedTrace.source
#check Umpire.EvidenceBackedTrace.profileId
#check Umpire.EvidenceBackedTrace.profileVersion
#check Umpire.EvidenceBackedTrace.sourceClosed
#check Umpire.EvidenceBackedTrace.vocabulary
#check Umpire.EvidenceBackedTrace.dispositions
#check Umpire.EvidenceBackedTrace.appliedBound
#check Umpire.EvidenceBackedTrace.evidenceIdentities
#check Umpire.EvidenceBackedTrace.recordSupport
#check Umpire.EvidenceBackedTrace.trace
#check Umpire.EvidenceBackedTrace.evidenceLinks
#synth BEq Umpire.EvidenceBackedTrace
#synth DecidableEq Umpire.EvidenceBackedTrace
#synth Repr Umpire.EvidenceBackedTrace

private def representativeEvidenceBundle : Umpire.EvidenceBundle := {
  profile := Umpire.DefinitionId.of "test.evidence.profile"
  profileVersion := 1
  records := []
  closures := []
}

example : representativeEvidenceBundle.sourceClosed = true := rfl
example : Umpire.EvidenceValue.render (.boolean true) = "true" := rfl

private def representativeDiagnostic : Umpire.ObservationDiagnostic := {
  kind := .emptyEvidence
  planId := Umpire.DefinitionId.of "test.observation.plan"
}

example : (Umpire.ObservationResult.unknown representativeDiagnostic).status = .unknown := rfl
example : (Umpire.ObservationResult.unknown representativeDiagnostic).diagnostic? =
    some representativeDiagnostic := rfl

/--
error: Unknown constant `Umpire.EvidenceBackedTrace.mk`
-/
#guard_msgs in
#check Umpire.EvidenceBackedTrace.mk

/--
error: invalid {...} notation, constructor for `Umpire.EvidenceBackedTrace` is marked as private
-/
#guard_msgs in
def replaceAcceptedTraceId
    (trace : Umpire.EvidenceBackedTrace) : Umpire.EvidenceBackedTrace := {
  trace with traceId := "forged"
}

#check Umpire.ObservationResult
#check Umpire.ObservationResult.accepted
#check Umpire.ObservationResult.unknown
#check Umpire.ObservationResult.conflict
#check Umpire.ObservationResult.unsupported
#check Umpire.ObservationResult.status
#check Umpire.ObservationResult.diagnostic?
#synth BEq Umpire.ObservationResult
#synth DecidableEq Umpire.ObservationResult
#synth Repr Umpire.ObservationResult
#check Umpire.validateEvidenceBackedTrace
#check Umpire.evaluateEvidence
#check Umpire.SemanticPropertyVerdict
#check Umpire.evaluateObservationProperty
#check Umpire.StrictQuerySummary
#check Umpire.summarizeQueryVerdicts
#check Umpire.RunEvaluation
#check Umpire.checkRunEvaluation
