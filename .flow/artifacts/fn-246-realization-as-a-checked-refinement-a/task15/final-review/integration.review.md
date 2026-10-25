I’ll read the task contract first, then trace the changed model and checks through their unchanged callers and consumers.
The model preserves timeout fact identity and global-budget priority, and reuses the worker-failure retry effect. I’m checking compatibility projection, generated fixtures, and the promoted guest for assumptions that attempt timeout is always terminal.
Eleven focused controls passed using the cached fixture binary, whose hash and source manifest match this change. The native adapter check also passed. Cargo could not rebuild in this read-only sandbox; I’m finishing the guest-freshness and evidence checks, including the TLC timeout and its recorded complement run.
No surviving findings. The retry transition integrates with unchanged worker handling, compatibility projection, source lifting, and fixture consumers.

Verification: 13 focused cached tests passed, including exact timeout edges, timer resets, selected Worlds, mutants, compatibility controls, snapshots, and native adapter output. Source and fixture-binary hashes match the committed evidence. Cargo rebuilding was blocked by the read-only sandbox. Required TLC evidence records the timeout honestly and supplies a passing complement; the final joined gate remains conductor-owned.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | deferred | Frontier/replay work remains with downstream tasks. |
| R2 | deferred | Remaining capability integration is outside task .15. |
| R3 | deferred | Shared infrastructure migration retains its assigned owners. |
| R4 | deferred | Binding and live-gap work remains downstream. |
| R5 | partial | Task .15’s timeout correction and focused proof renewal are implemented; full protocol/refinement and final validation retain their named owners. |
| R6 | deferred | Worker-policy generation remains downstream. |
| R7 | deferred | Final counts and documentation remain downstream. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge; no blocking introduced findings.