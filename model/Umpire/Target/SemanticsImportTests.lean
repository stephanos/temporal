import Umpire.Target.Semantics

/-! The semantic import exposes authoritative kernels and relation-indexed finite planning. -/

namespace Umpire.Target.SemanticsImportTests

open Umpire

example (target : CheckedTarget LawStatement Setup State Action Outcome Observation)
    (setup : Setup) (state : State) (member : state ∈ target.kernel.initialStates setup) :
    target.kernel.authoritativeInitial setup state :=
  target.kernel.initialSound setup state member

example (target : CheckedTarget LawStatement Setup State Action Outcome Observation)
    (capability : FinitePlanningCapability target.kernel.authoritativeStep)
    (state : State) (action : Action) (result : TransitionResult State Outcome Observation)
    (step : target.kernel.authoritativeStep state action result) : action ∈ capability.actions :=
  capability.actionComplete state action result step

#check CheckedTarget.withEquivalentKernel
#check composeTarget
#check checkTarget
#check checkedTarget

/-- error: Unknown constant `Umpire.CheckedTarget.mk` -/
#guard_msgs (error) in
#check Umpire.CheckedTarget.mk

/-- error: Unknown constant `Umpire.AuthoredTarget.mk` -/
#guard_msgs (error) in
#check Umpire.AuthoredTarget.mk

/-- error: Unknown identifier `Umpire.captureAuthoringOccurrence` -/
#guard_msgs (error) in
#check Umpire.captureAuthoringOccurrence

/-- error: Unknown identifier `Umpire.elaborateTarget` -/
#guard_msgs (error) in
#check Umpire.elaborateTarget

end Umpire.Target.SemanticsImportTests
