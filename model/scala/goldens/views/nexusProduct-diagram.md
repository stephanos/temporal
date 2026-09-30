# nexusProduct: phases

Generated; do not edit. Self-loops and the fields besides the phase are left out.

```mermaid
stateDiagram-v2
    [*] --> scheduled
    scheduled --> canceled: complete, handlerReply
    scheduled --> failed: complete, handlerReply
    scheduled --> succeeded: complete, handlerReply
    scheduled --> started: handlerReply
    scheduled --> timedOut: timeout
    started --> canceled: complete
    started --> failed: complete
    started --> succeeded: complete
    started --> timedOut: timeout
    succeeded --> [*]
    failed --> [*]
    canceled --> [*]
    timedOut --> [*]
```
