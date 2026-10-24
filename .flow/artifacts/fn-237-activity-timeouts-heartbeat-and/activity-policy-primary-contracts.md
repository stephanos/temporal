I’ll read the task contract first, then trace the changed code and evidence for consistency with its promises.
The deadline priority, cancellation races, and live constructor changes match the task contract so far. I’m checking whether the fairness, replay, and retained-graph controls substantiate the documented guarantees.
The committed controls cover heartbeat and stale-Poll starvation, exact expiry facts, and the retained lifecycle projection. The focused Cargo run is blocked by the read-only build cache; I’ll check the archived execution receipts and retained artifact hashes for supporting evidence.
No blocking findings.

The implementation, documentation, and controls agree on deadline priority, cancellation races, exact expiry facts, fairness, and compatibility projection.

Verification: all 15 original artifacts and 863 protected hashes matched; `git diff --check` passed. Both focused Cargo suites were blocked by the read-only build cache. Archived execution receipts record passing focused suites and required TLC controls.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | met | Existing F/O support consumed; configuration validation and exact-fact controls retained. |
| R2 | met | Deadline/cancellation matrices, concrete fairness controls, replay and spanning-budget laws. |
| R3 | met | Exact retained-graph comparison, changed-edge control and unchanged original hashes. |
| R4 | deferred | Live admission and evidence belong to task .4. |
| R5 | deferred | Final design write-up belongs to task .5. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>