I’ll read the task specification first, then review the diff and validation evidence for agreement with its contract.
The new decoder preserves the explicit-null rejection and guards against serde’s positional-array decoding. I’m checking the existing controls and recorded runs for metadata, budgets, and typed replay.
I haven’t found a contract mismatch. All 28 tests passed when rerun from the existing focused test binaries; I did not rebuild in this read-only session. The evidence hashes match the reviewed source, and the joined log records a successful run with Apalache required.
No surviving findings. The derived envelope preserves strict field handling, explicit-null refusal, object-only input, budgets, and typed validation.

All 28 tests passed using existing focused test binaries; no rebuild was performed. Recorded source hashes match the reviewed files, and joined evidence records successful required-Apalache execution.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Task .6 scope: serde-derived envelope and strict refusal controls; dependency already locked; feature inventories retained. Other replacements belong to sibling tasks. |
| R2 | deferred | Assigned to task .2. |
| R3 | deferred | Assigned to task .3. |
| R4 | deferred | Assigned to task .1. |
| R5 | deferred | Full joined gate and guest publication belong to task .7. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>