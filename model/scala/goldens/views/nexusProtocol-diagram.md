# nexusProtocol: phases

Generated; do not edit. Self-loops and the fields besides the phase are left out.

```mermaid
stateDiagram-v2
    [*] --> unscheduled
    unscheduled --> scheduled: schedule
    scheduled --> canceled: complete, handlerReply
    scheduled --> failed: complete, handlerReply
    scheduled --> succeeded: complete, handlerReply
    scheduled --> started: handlerReply
    scheduled --> backingOff: handlerReply, transportFault
    scheduled --> timedOut: scheduleToStart, scheduleToClose
    backingOff --> scheduled: backoff
    backingOff --> canceled: complete
    backingOff --> failed: complete
    backingOff --> succeeded: complete
    backingOff --> timedOut: scheduleToStart, scheduleToClose
    started --> canceled: complete
    started --> failed: complete
    started --> succeeded: complete
    started --> timedOut: startToClose, scheduleToClose
    succeeded --> [*]
    failed --> [*]
    canceled --> [*]
    timedOut --> [*]
```
