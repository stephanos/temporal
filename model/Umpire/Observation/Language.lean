import Umpire.Observation.Declaration
import Umpire.Observation.Compiler

/-!
Stable authoring facade for Observation declarations and deterministic checking. The declaration
and compiler modules own the implementation, while `checkedObservation` remains the explicit-proof
convenience for authors.
-/

namespace Umpire

/-- Produce a checked Observation plan directly from an explicit proof that the typed checker
succeeds. Use `checkObservation` when an invalid mapping's typed diagnostic is needed. -/
def checkedObservation
    (context : ObservationCheckContext)
    (declaration : ObservationMappingDeclaration)
    (valid : (checkObservation context declaration).toOption.isSome = true) :
    CheckedObservationPlan :=
  (checkObservation context declaration).toOption.get valid

end Umpire
