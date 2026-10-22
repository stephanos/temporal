I’ll read the task specification first, then trace the changed code through its callers and consumers to check for integration regressions.
The one-attempt projection retains both deadline edges and all four rejected terminal Cancel loops. I’m now checking how payload actions pass through the shared RPC generator and guest runtime, where unchanged consumers could expose regressions.
Cargo cannot run either focused suite because the sandbox blocks access to its build lock. The retained-artifact hash check passed. I also found a possible callback issue: the runtime treats a rejected delivery step as replayable, so I’m tracing whether a mismatched Poll attempt can still produce a worker reply.
I reproduced the callback issue against the committed Wasm guest. After a valid attempt-one callback, callbacks for attempts two and three incorrectly return an Answer; repeating attempt one is correctly refused. The new rejected Poll self-loop is being accepted as delivery evidence by the unchanged runtime.
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/temporal/feature/activity/temporal.rs:198`
- **R-IDs**: [R4]
- **Problem**: A mismatched callback attempt can produce a successful worker reply. After an attempt-one callback leaves the runtime in `Started(1)`, an attempt-two callback proposes `Poll(2)`. The new handler emits a rejected self-loop, which unchanged `Runtime::replay` accepts as delivery evidence. `RespondCompleted` remains enabled, so Answer succeeds despite the rejected Poll. Reproduced against the committed Wasm: callbacks for attempts two and three both return Answer after attempt one; repeating attempt one correctly returns Refusal.
- **Suggestion**: Give `respond` the supported state parameter and reject callbacks whose attempt differs from the current attempt before constructing the reply. Add a guest-level regression test that delivers attempt one, then mismatched callbacks, and verifies refusal without state changes.

The retained-artifact verifier passed, and all 15 copies match their pinned Git blobs. Cargo verification was blocked by the read-only build-lock path.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Owned by other tasks in the epic. |
| R2 | deferred | Owned by other tasks in the epic. |
| R3 | met | Bound source lifting, TLC agreement goldens, and payload/outcome replay controls. |
| R4 | partial | Retry and retained-graph controls are present; callback integration bypasses mismatched-attempt rejection. |
| R5 | deferred | Retired requirement moved to fn-237. |
| R6 | deferred | Request admission and live retry proof belong to task .5. |
| R7 | deferred | Final documentation belongs to later epic work. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>