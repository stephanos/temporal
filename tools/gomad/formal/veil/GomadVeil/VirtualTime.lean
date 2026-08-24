import Veil

veil module GomadVirtualTime

enum Time = {T0, T2, T5}
enum TimerStatus = {Absent, Pending, Ready, Fired, Cancelled}
enum Flag = {No, Yes}

individual now : Time
individual runnable : Flag
individual timerA : TimerStatus
individual timerB : TimerStatus
individual timerNow : TimerStatus

#gen_state

after_init {
  now := *
  runnable := No
  timerA := Absent
  timerB := Absent
  timerNow := Absent
  assume ((now = T0) ∨ (now = T5))
}

action ScheduleTimerAAtTwo {
  require ((now = T0) ∧ (timerA = Absent))
  timerA := Pending
}

action ScheduleTimerBAtTwo {
  require ((now = T0) ∧ (timerB = Absent))
  timerB := Pending
}

action ScheduleImmediateTimer {
  require ((now = T5) ∧ (timerNow = Absent))
  timerNow := Ready
}

action SetWorkerRunnable {
  require (runnable = No)
  runnable := Yes
}

action ClearWorkerRunnable {
  require (runnable = Yes)
  runnable := No
}

action AdvanceToTwo {
  require ((now = T0) ∧ (runnable = No) ∧
    (¬ (timerA = Ready)) ∧ (¬ (timerB = Ready)) ∧
    ((timerA = Pending) ∨ (timerB = Pending)))
  now := T2
  if timerA = Pending then timerA := Ready
  if timerB = Pending then timerB := Ready
}

action FireTimerA {
  require (timerA = Ready)
  timerA := Fired
}

action FireImmediateTimer {
  require (timerNow = Ready)
  timerNow := Fired
}

action CancelTimerB {
  require ((timerB = Pending) ∨ (timerB = Ready))
  timerB := Cancelled
}

safety [FiredTimerAHasReachedDeadline] ((¬ (timerA = Fired)) ∨ (now = T2))

#gen_spec

end GomadVirtualTime
