import Umpire.Observation

/-! Focused import contract for Observation authoring, checking, and evaluation. -/

#check Umpire.ObservationMappingDeclaration
#check Umpire.CheckedObservationPlan
#check Umpire.EvidenceBound
#check Umpire.ObservationStatus
#check Umpire.ObservationFailureKind
#check Umpire.ObservationDiagnostic
#check Umpire.ModelCoordinate
#check Umpire.EvidenceLink
#check Umpire.EvidenceBackedTrace
#synth BEq Umpire.EvidenceBackedTrace
#synth DecidableEq Umpire.EvidenceBackedTrace
#synth Repr Umpire.EvidenceBackedTrace

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
#check Umpire.evaluateEvidence
#check Umpire.SemanticPropertyVerdict
#check Umpire.evaluateObservationProperty
#check Umpire.StrictQuerySummary
#check Umpire.summarizeQueryVerdicts
#check Umpire.RunEvaluation
#check Umpire.checkRunEvaluation
