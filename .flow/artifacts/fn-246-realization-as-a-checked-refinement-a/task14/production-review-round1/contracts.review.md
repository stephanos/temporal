I’ll read the task specification first, then trace the changed contracts through the implementation and focused test evidence.
The task adds a bridge that correlates actual RPC completions with exact heartbeat histories, while retaining terminal-only behavior. I’m checking admission, SDK cancellation, and fault capture against the task’s refusal and shutdown promises.
The focused Go test command could not start because the read-only sandbox blocks Go’s temporary build directory. I’m continuing with source tracing and the committed test logs, keeping recorded results separate from tests run in this session.
I found a possible shutdown gap: terminal acknowledgments are validated asynchronously, and Driver.Close cancels that validation path first. I’m checking whether a malformed acknowledgment that has already arrived can be discarded as ordinary shutdown.
- **Severity:** P2
- **Confidence:** 75
- **Classification:** introduced
- **File:Line:** `model/temporal/hosts/go/reference/worker/sdk/sdk.go:198`
- **R-IDs:** [R6]
- **Problem:** Shutdown can silently discard validation of an already captured terminal acknowledgment. `Driver.Close` cancels the worker context before joining. If cancellation precedes coordinator processing, `Assignment.wait` returns cancellation despite an available completion, and `coordinateTerminals` exits without checking `TerminalResult`. Even a validation error already computed is suppressed when `run.Err() != nil`. Consequently, a mutated terminal payload or nonempty Failed acknowledgment failures can close cleanly without fault evidence. The existing mismatch tests wait for executor failure before stopping, so they do not exercise this ordering.
- **Suggestion:** During shutdown, validate completed terminal expectations and retain substantive validation errors while allowing unfinished canceled operations to close normally. Also preserve these errors through `act.go:336`, which currently drops late Follow errors unless another worker fault already exists. Add a deterministic control that captures an invalid acknowledgment, delays coordinator processing, then closes the Driver.

The focused Go tests could not start because the read-only sandbox prevented creation of Go’s temporary build directory.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Other parent tasks own Reach/frontier integration. |
| R2 | deferred | Capability integration remains with other parent tasks. |
| R3 | deferred | Shared catalog/lens completion remains separately assigned. |
| R4 | deferred | Activity migration and live parity remain separately assigned. |
| R5 | deferred | Tasks .15/.13/.4 retain correction and refinement proofs. |
| R6 | partial | History codec, raw/SDK continuation and focused controls implemented; shutdown validation gap remains. Enumeration and integrated coverage belong to .5/.6. |
| R7 | deferred | Mapping counts and author documentation remain separately assigned. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict> - Preserve terminal validation failures across shutdown.