import Umpire.Model.Check

/-! The semantic import exposes authoritative kernels and relation-indexed finite planning. -/

namespace Umpire.Model.CheckImportTests

open Umpire

example (target : CheckedModel LawStatement Setup State Action Outcome Observation)
    (setup : Setup) (state : State) (member : state ∈ target.machine.initialStates setup) :
    target.machine.authoritativeInitial setup state :=
  target.machine.initialSound setup state member

example (target : CheckedModel LawStatement Setup State Action Outcome Observation)
    (capability : FinitePlanningCapability target.machine.authoritativeStep)
    (state : State) (action : Action) (result : Step State Outcome Observation)
    (step : target.machine.authoritativeStep state action result) : action ∈ capability.actions :=
  capability.actionComplete state action result step

#check CheckedModel.withEquivalentMachine
#check checkModel
#check model

/-- error: Unknown constant `Umpire.CheckedModel.mk` -/
#guard_msgs (error) in
#check Umpire.CheckedModel.mk

/-- error: Unknown constant `Umpire.DraftModel.mk` -/
#guard_msgs (error) in
#check Umpire.DraftModel.mk

/-- error: Unknown identifier `Umpire.captureSourceRef` -/
#guard_msgs (error) in
#check Umpire.captureSourceRef

/-- error: Unknown identifier `Umpire.elabModel` -/
#guard_msgs (error) in
#check Umpire.elabModel

end Umpire.Model.CheckImportTests
