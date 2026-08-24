import Temporal.Families.UpdateLifecycle.Refinement
import Temporal.Families.UpdateLifecycle.Targets.Behavior

namespace Umpire3.Tests.Families.UpdateLifecycle

example : Umpire3.Temporal.Targets.UpdateLifecycleBehavior.featureExecutable.initials () =
    [Umpire3.Temporal.Feature.UpdateLifecycle.initial] := rfl

example : Umpire3.Temporal.Targets.UpdateLifecycleBehavior.systemExecutable.initials () =
    [Umpire3.Temporal.System.UpdateLifecycle.initial] := rfl

example : Umpire3.SafetySimulation
    Umpire3.Temporal.System.UpdateLifecycle.behavior
    Umpire3.Temporal.Feature.UpdateLifecycle.behavior :=
  Umpire3.Temporal.Refinement.UpdateLifecycle.soundSimulation

end Umpire3.Tests.Families.UpdateLifecycle
