import Umpire.Case.Projection
import Umpire.Evidence
import Umpire.Shared.Test

/-! Projection seals raw evidence and authority; its kernel path adds no compiler-trust axiom. -/

#check Umpire.Case.Projection.check
#check Umpire.Case.Projection.Checked.start
#check Umpire.Case.Projection.Run.admit
#check Umpire.Case.Projection.Run.close

/-- error: Unknown constant `Umpire.Case.Projection.Checked.mk` -/
#guard_msgs in
#check Umpire.Case.Projection.Checked.mk

/-- error: Unknown constant `Umpire.Case.Projection.Run.mk` -/
#guard_msgs in
#check Umpire.Case.Projection.Run.mk

/-- error: Unknown constant `Umpire.Case.Projection.Step.mk` -/
#guard_msgs in
#check Umpire.Case.Projection.Step.mk

/-- error: Unknown constant `Umpire.Case.Projection.Run.payload` -/
#guard_msgs in
#check Umpire.Case.Projection.Run.payload

/--
error: Type mismatch
  event
has type
  Umpire.Case.Projection.Event
but is expected to have type
  Umpire.ModelTrace Bool Bool Bool Bool
-/
#guard_msgs in
private def rawPropertyTrace (event : Umpire.Case.Projection.Event) :
    Umpire.ModelTrace Bool Bool Bool Bool := event

-- `evaluateProperty` is checked once, in `Umpire.Property.Tests.Endpoints`.
assert_axioms [Umpire.Case.Projection.check, Umpire.Case.Projection.Run.admit,
  Umpire.Case.Projection.Run.close, Umpire.validateEvidenceBackedTrace]
  allowing [propext, Classical.choice, Quot.sound]
assert_axioms [Umpire.Case.Projection.Step.semantic] allowing []
