import SharedModel.Transition

namespace Gomad.VirtualTime

inductive TimerId where
  | timerA
  | timerB
  | timerNow
deriving BEq, DecidableEq, Repr

inductive WorkId where
  | worker
deriving BEq, DecidableEq, Repr

inductive TimerStatus where
  | pending
  | ready
  | fired
  | cancelled
deriving BEq, DecidableEq, Repr

structure Timer where
  id : TimerId
  deadline : Nat
  status : TimerStatus
deriving BEq, DecidableEq, Repr

structure State where
  now : Nat
  runnable : Bool
  timers : List Timer
deriving BEq, DecidableEq, Repr

def State.empty (now : Nat) : State :=
  { now, runnable := false, timers := [] }

inductive Action where
  | scheduleTimer (id : TimerId) (deadline : Nat)
  | cancelTimer (id : TimerId)
  | setRunnable (id : WorkId) (runnable : Bool)
  | advanceTime
  | fireTimer (id : TimerId)
deriving BEq, DecidableEq, Repr

inductive ActionKind where
  | scheduleTimer
  | cancelTimer
  | setRunnable
  | advanceTime
  | fireTimer
deriving BEq, DecidableEq, Repr

structure ObservableDelta where
  kind : ActionKind
  timerId : Option TimerId := none
  timerBefore : Option TimerStatus := none
  timerAfter : Option TimerStatus := none
  workId : Option WorkId := none
  runnable : Option Bool := none
  previousTime : Option Nat := none
  currentTime : Option Nat := none
  readyTimers : List TimerId := []
deriving BEq, DecidableEq, Repr

inductive RejectionCode where
  | timerExists
  | unknownTimer
  | deadlineBeforeNow
  | timerTerminal
  | timerNotReady
  | runnableUnchanged
  | runnableWork
  | readyTimer
  | noPendingTimer
deriving BEq, DecidableEq, Repr

inductive StepResult where
  | accepted (state : State) (delta : ObservableDelta)
  | rejected (code : RejectionCode)
deriving BEq, DecidableEq, Repr

def State.timer? (state : State) (id : TimerId) : Option Timer :=
  state.timers.find? fun timer => timer.id == id

private def replaceTimer (timers : List Timer) (updated : Timer) : List Timer :=
  timers.map fun timer => if timer.id == updated.id then updated else timer

private def minimum? : List Nat → Option Nat
  | [] => none
  | first :: rest => some (rest.foldl Nat.min first)

private def pendingDeadlines (state : State) : List Nat :=
  state.timers.filterMap fun timer =>
    if timer.status == .pending then some timer.deadline else none

private def advanceTimers (timers : List Timer) (deadline : Nat) : List Timer :=
  timers.map fun timer =>
    if timer.status == .pending && timer.deadline == deadline then
      { timer with status := .ready }
    else
      timer

private def timersReadyAt (timers : List Timer) (deadline : Nat) : List TimerId :=
  timers.filterMap fun timer =>
    if timer.status == .pending && timer.deadline == deadline then some timer.id else none

def step (state : State) (action : Action) : StepResult :=
  match action with
  | .scheduleTimer id deadline =>
      if (state.timer? id).isSome then
        .rejected .timerExists
      else if deadline < state.now then
        .rejected .deadlineBeforeNow
      else
        let status := if deadline == state.now then TimerStatus.ready else TimerStatus.pending
        let timer := { id, deadline, status }
        .accepted { state with timers := state.timers ++ [timer] }
          { kind := .scheduleTimer, timerId := some id, timerAfter := some status }
  | .cancelTimer id =>
      match state.timer? id with
      | none => .rejected .unknownTimer
      | some timer =>
          if timer.status == .fired || timer.status == .cancelled then
            .rejected .timerTerminal
          else
            let updated := { timer with status := .cancelled }
            .accepted { state with timers := replaceTimer state.timers updated }
              { kind := .cancelTimer, timerId := some id, timerBefore := some timer.status,
                timerAfter := some .cancelled }
  | .setRunnable id runnable =>
      if state.runnable == runnable then
        .rejected .runnableUnchanged
      else
        .accepted { state with runnable }
          { kind := .setRunnable, workId := some id, runnable := some runnable }
  | .advanceTime =>
      if state.runnable then
        .rejected .runnableWork
      else if state.timers.any fun timer => timer.status == .ready then
        .rejected .readyTimer
      else
        match minimum? (pendingDeadlines state) with
        | none => .rejected .noPendingTimer
        | some deadline =>
            .accepted { state with now := deadline, timers := advanceTimers state.timers deadline }
              { kind := .advanceTime, previousTime := some state.now,
                currentTime := some deadline, readyTimers := timersReadyAt state.timers deadline }
  | .fireTimer id =>
      match state.timer? id with
      | none => .rejected .unknownTimer
      | some timer =>
          if timer.status != .ready then
            .rejected .timerNotReady
          else
            let updated := { timer with status := .fired }
            .accepted { state with timers := replaceTimer state.timers updated }
              { kind := .fireTimer, timerId := some id, timerBefore := some .ready,
                timerAfter := some .fired }

def transitionSystem : SharedModel.TransitionSystem where
  State := State
  Action := Action
  Initial := fun state => state = State.empty 0
  Step := fun state action nextState =>
    ∃ delta, step state action = .accepted nextState delta

def run : State → List Action → Option State
  | state, [] => some state
  | state, action :: actions =>
      match step state action with
      | .accepted nextState _ => run nextState actions
      | .rejected _ => none

def boundedActions : List Action := [
  .scheduleTimer .timerA 2,
  .scheduleTimer .timerB 2,
  .scheduleTimer .timerNow 5,
  .cancelTimer .timerA,
  .cancelTimer .timerB,
  .cancelTimer .timerNow,
  .setRunnable .worker true,
  .setRunnable .worker false,
  .advanceTime,
  .fireTimer .timerA,
  .fireTimer .timerB,
  .fireTimer .timerNow,
]

def actionName : Action → String
  | .scheduleTimer .timerA 2 => "schedule-timer-a"
  | .scheduleTimer .timerB 2 => "schedule-timer-b"
  | .scheduleTimer .timerNow 5 => "schedule-timer-now"
  | .scheduleTimer _ _ => "schedule-timer-out-of-bounds"
  | .cancelTimer .timerA => "cancel-timer-a"
  | .cancelTimer .timerB => "cancel-timer-b"
  | .cancelTimer .timerNow => "cancel-timer-now"
  | .setRunnable .worker true => "set-worker-runnable"
  | .setRunnable .worker false => "clear-worker-runnable"
  | .advanceTime => "advance-time"
  | .fireTimer .timerA => "fire-timer-a"
  | .fireTimer .timerB => "fire-timer-b"
  | .fireTimer .timerNow => "fire-timer-now"

def successors (state : State) : List (Action × State) :=
  boundedActions.filterMap fun action =>
    match step state action with
    | .accepted nextState _ => some (action, nextState)
    | .rejected _ => none

end Gomad.VirtualTime
