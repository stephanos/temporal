# activityProtocol: phases

Generated; do not edit. Self-loops and the fields besides the phase are left out.

```mermaid
stateDiagram-v2
    [*] --> unstarted
    unstarted --> scheduled: start
    scheduled --> started: attemptStart
    scheduled --> paused: control
    scheduled --> cancelRequested: control
    scheduled --> terminated: control
    scheduled --> timedOut: scheduleToStart, scheduleToClose
    backingOff --> scheduled: backoff
    backingOff --> paused: control
    backingOff --> cancelRequested: control
    backingOff --> terminated: control
    backingOff --> timedOut: scheduleToStart, scheduleToClose
    started --> completed: attemptResult
    started --> failed: attemptResult
    started --> backingOff: attemptResult
    started --> pauseRequested: control
    started --> cancelRequested: control
    started --> terminated: control
    started --> timedOut: startToClose, scheduleToClose
    paused --> cancelRequested: control
    paused --> terminated: control
    paused --> scheduled: control
    paused --> timedOut: scheduleToClose
    pauseRequested --> completed: attemptResult
    pauseRequested --> failed: attemptResult
    pauseRequested --> paused: attemptResult
    pauseRequested --> cancelRequested: control
    pauseRequested --> terminated: control
    pauseRequested --> started: control
    pauseRequested --> timedOut: startToClose, scheduleToClose
    cancelRequested --> canceled: attemptResult
    cancelRequested --> completed: attemptResult
    cancelRequested --> failed: attemptResult
    cancelRequested --> terminated: control
    cancelRequested --> timedOut: startToClose, scheduleToClose
    completed --> [*]
    failed --> [*]
    canceled --> [*]
    terminated --> [*]
    timedOut --> [*]
```
