import Umpire.Observation.Evaluation.Types
import Umpire.Observation.Evaluation.Structure
import Umpire.Observation.Evaluation.Raw
import Umpire.Observation.Evaluation.Admission

/-!
Pure Observation Evaluation of bounded synthetic Evidence. The boundary consumes a complete
checked plan and a finite typed bundle, then either returns one fully derived Model Trace or one
typed diagnostic. Raw Evidence is used only while evaluating the bundle and is absent from every
successful result. This layer establishes Model Facts; it does not perform Run Evaluation or Claim
Assessment.
-/

namespace Umpire

private def resultOfDiagnostic (failure : ObservationDiagnostic) : ObservationResult :=
  match failure.status with
  | .unknown => .unknown failure
  | .conflict => .conflict failure
  | .unsupported => .unsupported failure
  | .accepted => .unknown failure

/-- Evaluate Evidence without exposing an intermediate or partially constructed Model Trace. -/
def evaluateEvidence
    (plan : CheckedObservationPlan)
    (bundle : EvidenceBundle) : ObservationResult :=
  match Observation.Internal.evaluateUnchecked plan bundle with
  | .ok unchecked =>
      match validateEvidenceBackedTrace unchecked with
      | .ok trace => .accepted trace
      | .error failure => resultOfDiagnostic failure
  | .error failure => resultOfDiagnostic failure

end Umpire
