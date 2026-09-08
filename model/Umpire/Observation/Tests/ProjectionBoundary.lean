import Umpire.Observation

/-! Projection seals raw evidence and authority; its kernel path adds no compiler-trust axiom. -/

#check Umpire.Observation.Projection.check
#check Umpire.Observation.Projection.Checked.start
#check Umpire.Observation.Projection.Run.admit
#check Umpire.Observation.Projection.Run.close

/-- error: Unknown constant `Umpire.Observation.Projection.Checked.mk` -/
#guard_msgs in
#check Umpire.Observation.Projection.Checked.mk

/-- error: Unknown constant `Umpire.Observation.Projection.Run.mk` -/
#guard_msgs in
#check Umpire.Observation.Projection.Run.mk

/-- error: Unknown constant `Umpire.Observation.Projection.Step.mk` -/
#guard_msgs in
#check Umpire.Observation.Projection.Step.mk

/-- error: Unknown constant `Umpire.Observation.Projection.Run.payload` -/
#guard_msgs in
#check Umpire.Observation.Projection.Run.payload

/--
error: Type mismatch
  event
has type
  Umpire.Observation.Projection.Event
but is expected to have type
  Umpire.ModelTrace Bool Bool Bool Bool
-/
#guard_msgs in
private def rawPropertyTrace (event : Umpire.Observation.Projection.Event) :
    Umpire.ModelTrace Bool Bool Bool Bool := event

/-- info: 'Umpire.Observation.Projection.check' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Observation.Projection.check
/-- info: 'Umpire.Observation.Projection.Run.admit' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Observation.Projection.Run.admit
/-- info: 'Umpire.Observation.Projection.Run.close' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Observation.Projection.Run.close
/-- info: 'Umpire.Observation.Projection.Step.semantic' does not depend on any axioms -/
#guard_msgs in
#print axioms Umpire.Observation.Projection.Step.semantic
/-- info: 'Umpire.validateEvidenceBackedTrace' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.validateEvidenceBackedTrace
/-- info: 'Umpire.evaluateProperty' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.evaluateProperty
