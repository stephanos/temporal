import Temporal.Refinement.MigratedFamilies
import Temporal.Monitors

namespace Umpire3.Tests.TemporalNexusClosure

open Umpire3.Temporal.Product.NexusClosure

def permittedActions : List Action := [
  .schedule .primary,
  .start .primary,
  .settle .primary .succeeded,
  .closeWorkflow .completed,
]

def forbiddenActions : List Action := [
  .schedule .primary,
  .start .primary,
  .closeWorkflow .completed,
]

example : executable.follow [initial] permittedActions = [permittedFinal] := by decide

example : Closure permittedFinal := by decide

example : (permittedFinal.workflow.terminal = true ∧
    permittedFinal.caller .primary = true ∧
    (permittedFinal.operation .primary).terminal = true) := by decide

example : executable.follow [initial] forbiddenActions = [] := by decide

example : unsafeExecutable.follow [initial] forbiddenActions = [unsafeFinal] := by decide

example : ¬Closure unsafeFinal := by decide

set_option maxRecDepth 100000 in
example :
    (bounded.explore { maxDepth := 4, maxResults := 1000 }).all
      (fun execution => closureB execution.2) = true := by decide

def ordinarySystemActions :
    List Umpire3.Temporal.System.MigratedFamilies.NexusClosure.Action := [
  .schedule,
  .start,
  .settle,
  .close,
]

def closeWhileRunningSystemActions :
    List Umpire3.Temporal.System.MigratedFamilies.NexusClosure.Action := [
  .schedule,
  .start,
  .closeWhileRunning,
]

example : Umpire3.Temporal.System.MigratedFamilies.NexusClosure.executable.follow ()
    [Umpire3.Temporal.System.MigratedFamilies.NexusClosure.initial] ordinarySystemActions =
    [.closed] := by decide

example : Umpire3.Temporal.System.MigratedFamilies.NexusClosure.executable.follow ()
    [Umpire3.Temporal.System.MigratedFamilies.NexusClosure.initial]
    closeWhileRunningSystemActions = [] := by decide

example : Umpire3.Temporal.System.MigratedFamilies.NexusClosure.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.MigratedFamilies.NexusClosure.initial]
    closeWhileRunningSystemActions = [.closedWhileRunning] := by decide

example : Umpire3.Temporal.Refinement.MigratedFamilies.NexusClosure.soundSimulation.Relates
    (world := ())
    Umpire3.Temporal.System.MigratedFamilies.NexusClosure.initial initial := rfl

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.MigratedFamilies.NexusClosure.System.mutatedBehavior
      Umpire3.Temporal.Refinement.MigratedFamilies.NexusClosure.Feature.behavior
      Umpire3.Temporal.Refinement.MigratedFamilies.NexusClosure.Projects
      Umpire3.Temporal.Refinement.MigratedFamilies.NexusClosure.actionMap :=
  Umpire3.Temporal.Refinement.MigratedFamilies.NexusClosure.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.nexusOperationClosure.Holds
    (Umpire3.Temporal.Monitors.nexusClosureObservations permittedFinal) :=
  (Umpire3.Temporal.Monitors.nexusOperationClosure_monitor_equivalent permittedFinal).2 (by decide)

example : Umpire3.Temporal.Monitors.nexusOperationClosure.expression.eval
    (Umpire3.Temporal.Monitors.nexusClosureObservations unsafeFinal) = some false := by decide

example (state : State) :
    Umpire3.Temporal.Monitors.nexusOperationClosure.Holds
      (Umpire3.Temporal.Monitors.nexusClosureObservations state) ↔ Closure state :=
  Umpire3.Temporal.Monitors.nexusOperationClosure_monitor_equivalent state

end Umpire3.Tests.TemporalNexusClosure
