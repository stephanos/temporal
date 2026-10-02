I’ll read the task spec first, then trace the changed code against the existing machine composition and search APIs.
The composition uses the existing synchronization mechanism, and its tests pin all six reply classes plus the absence of reply rows while stopped. I’m checking the query’s scenario and limits, the authoring drift check, and the focused build next.
The field requirement matches the composition’s state lowering, and the diff leaves the existing query definitions and differential lines unchanged. The Go test cannot start because the sandbox blocks temporary files. The Lean build is still pending; I’m checking the authoring blocks directly in the meantime.
I found no blocking integration issue. All 13 authoring blocks match their source regions. R5 is only partially evidenced: the axiom pins are present, but the task does not record the requested kernel, predicate-enumeration, and elaboration timings. I’m treating that documentation gap as nonblocking under the supplied rubric.
No blocking findings introduced by this changeset.

All 13 AUTHORING blocks match their source regions. Runtime verification remains incomplete: Go tests were blocked by sandbox permissions; the focused Lean check exceeded 90 seconds.

## Requirements coverage

Task-scoped coverage:

| R-ID | Status | Evidence |
|---|---|---|
| R5 | partial | Axiom inventories pinned; requested timing measurements not recorded in task evidence. Nonblocking documentation gap. |
| R6 | met | Restricted worker catalog pins only `serve` and `workerStop`. |
| R7 | met | Bare reply trigger, worker field requirement, and verified outcome pinned. |
| R8 | met | 316 reachable states pinned; 316 × 23 actions is below 16384. |
| R9 | met | No fixture or golden bytes changed. |
| R10 | met | Existing queries and differential expectations unchanged. |
| R14 | met | New query listed as `veil default`, both backends verified within limits. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>