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

/-- Admit a constructor-authored mapping only through the existing Observation checker. -/
def ObservationMappingSpec.check
    (spec : ObservationMappingSpec)
    (context : ObservationCheckContext) : Except ObservationError CheckedObservationPlan :=
  checkObservation context spec.declaration

/-- Produce the checked mapping after the kernel verifies that the existing checker succeeds. -/
def ObservationMappingSpec.checked
    (spec : ObservationMappingSpec)
    (context : ObservationCheckContext)
    (valid : (spec.check context).toOption.isSome = true) : CheckedObservationPlan :=
  checkedObservation context spec.declaration valid

end Umpire
