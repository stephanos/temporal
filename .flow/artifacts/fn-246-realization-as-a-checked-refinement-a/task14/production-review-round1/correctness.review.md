I’ll read the task contract first, then trace the changed code and focused verification evidence for correctness issues.
The continuation path advances history only after a transport completion is correlated with the claimed task. I’m checking SDK retries, terminal acknowledgments, and shutdown to see whether failures reliably reach the driver.
The focused Go tests could not start because this sandbox cannot create Go’s temporary build directory. I’ll continue with source tracing and the committed test receipts, with that runtime verification limit noted.
- **Severity:** P2
- **Confidence:** 75
- **Classification:** introduced
- **File:Line:** `common/testing/testpilot/temporal/act/act.go:433`
- **R-IDs:** [R6]
- **Problem:** Normal raw-worker shutdown can now produce a spurious DriverFault. `close()` cancels the poll hold before canceling the worker context. If the blocked poll returns between those calls, raw `followPolls` sees its parent context still active and returns the cancellation error. `startWorker` latches that error, and this newly added code returns it from `Driver.Close`:
  ```go
  if workerErr := d.executionError(); workerErr != nil {
      errs = append(errs, workerErr)
  }
  ```
  This newly exposes the existing shutdown ordering as a fault during expected empty-poll shutdown, contrary to the task contract.
- **Suggestion:** Cancel the worker context before releasing/canceling the poll hold, then join both paths. Add a synchronized regression that lets the canceled poll return before shutdown proceeds.

Focused Go verification could not start because the sandbox prohibits creating Go’s temporary build directory.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | deferred | Assigned to other parent tasks. |
| R2 | deferred | Assigned to other parent tasks. |
| R3 | deferred | Assigned to other parent tasks. |
| R4 | deferred | Assigned to other parent tasks. |
| R5 | deferred | Protocol/refinement and correction tasks retain ownership. |
| R6 | partial | Continuation codecs, bridge, executors and transport controls implemented; shutdown issue remains. Enumeration and live proofs retain later owners. |
| R7 | deferred | Counting/documentation tasks retain ownership. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict> - Fix the shutdown race.