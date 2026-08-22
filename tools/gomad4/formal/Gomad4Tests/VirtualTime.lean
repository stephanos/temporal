import Gomad4.VirtualTime
import SharedModel.TraceReplay

namespace Gomad4.Tests.VirtualTime

open Gomad4.VirtualTime

def equalDeadlineActions : List Action := [
  .scheduleTimer .timerA 2,
  .scheduleTimer .timerB 2,
  .setRunnable .worker true,
  .setRunnable .worker false,
  .advanceTime,
  .fireTimer .timerA,
  .cancelTimer .timerB,
]

def equalDeadlineFinal : State := {
  now := 2
  runnable := false
  timers := [
    { id := .timerA, deadline := 2, status := .fired },
    { id := .timerB, deadline := 2, status := .cancelled },
  ]
}

example : run (State.empty 0) equalDeadlineActions = some equalDeadlineFinal := by decide

example : step (State.empty 0) .advanceTime = .rejected .noPendingTimer := by decide

example :
    step (State.empty 5) (.scheduleTimer .timerNow 5) =
      .accepted
        { now := 5, runnable := false,
          timers := [{ id := .timerNow, deadline := 5, status := .ready }] }
        { kind := .scheduleTimer, timerId := some .timerNow, timerAfter := some .ready } := by
  decide

example :
    SharedModel.TraceReplay.followNamed successors actionName [State.empty 0]
      ["schedule-timer-a", "schedule-timer-b", "set-worker-runnable",
        "clear-worker-runnable", "advance-time", "fire-timer-a", "cancel-timer-b"] =
      [equalDeadlineFinal] := by
  decide

end Gomad4.Tests.VirtualTime
