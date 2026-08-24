import Temporal.Families.NexusClosure.Refinement
import Temporal.Monitors

namespace Umpire3.Tests.TemporalNexusClosure

open Umpire3.Temporal.Feature.NexusClosure

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
    List Umpire3.Temporal.System.NexusClosure.Action := [
  .schedule,
  .start,
  .settle,
  .close,
]

def closeWhileRunningSystemActions :
    List Umpire3.Temporal.System.NexusClosure.Action := [
  .schedule,
  .start,
  .closeWhileRunning,
]

example : Umpire3.Temporal.System.NexusClosure.executable.follow ()
    [Umpire3.Temporal.System.NexusClosure.initial] ordinarySystemActions =
    [.closed] := by decide

example : Umpire3.Temporal.System.NexusClosure.executable.follow ()
    [Umpire3.Temporal.System.NexusClosure.initial]
    closeWhileRunningSystemActions = [] := by decide

example : Umpire3.Temporal.System.NexusClosure.mutatedExecutable.follow ()
    [Umpire3.Temporal.System.NexusClosure.initial]
    closeWhileRunningSystemActions = [.closedWhileRunning] := by decide

example : Umpire3.Temporal.Refinement.NexusClosure.soundSimulation.Relates
    (world := ())
    Umpire3.Temporal.System.NexusClosure.initial initial := rfl

example :
    ¬StepSimulation
      Umpire3.Temporal.Refinement.NexusClosure.System.mutatedBehavior
      Umpire3.Temporal.Refinement.NexusClosure.Feature.behavior
      Umpire3.Temporal.Refinement.NexusClosure.Projects
      Umpire3.Temporal.Refinement.NexusClosure.actionMap :=
  Umpire3.Temporal.Refinement.NexusClosure.mutationBreaksDeclaredSimulation

example : Umpire3.Temporal.Monitors.nexusOperationClosure.Holds
    (Umpire3.Temporal.Monitors.nexusClosureObservations permittedFinal) :=
  (Umpire3.Temporal.Monitors.nexusOperationClosure_monitor_equivalent permittedFinal).2 (by decide)

example : Umpire3.Temporal.Monitors.nexusOperationClosure.expression.eval
    (Umpire3.Temporal.Monitors.nexusClosureObservations unsafeFinal) = some false := by
  change (MonitorExpression.observation "nexus-operation-closed" true).eval
    [{ identifier := "nexus-operation-closed", value := closureB unsafeFinal }] = some false
  simp [MonitorExpression.eval, lookupObservation, unsafeFinal, initial, closureB,
    allRelatedTerminal, operationIDs, WorkflowState.terminal, OperationState.terminal,
    State.operation, State.caller]

example (state : State) :
    Umpire3.Temporal.Monitors.nexusOperationClosure.Holds
      (Umpire3.Temporal.Monitors.nexusClosureObservations state) ↔ Closure state :=
  Umpire3.Temporal.Monitors.nexusOperationClosure_monitor_equivalent state

end Umpire3.Tests.TemporalNexusClosure
