I’ll read the task specification first, then trace the changed activity-policy code through its callers, replay behavior, and focused verification evidence.
The live constructors preserve the timeoutless atomic policy required by this task. I’m checking that the new actions integrate with the unchanged realization and replay paths, and that the compatibility projection still detects changes to old behavior.
The two Quick Cargo commands could not run because the read-only sandbox blocks Cargo’s build lock. I’ll continue the source review and use cached test binaries where possible, while keeping those results distinct from a fresh build.
The cached native activity checks, realization tests, and admission tests passed; backend-only tests were skipped in that run. All 863 preserved artifact hashes also match. I found no blocking integration defect. Required-TLC verification remains based on the committed execution receipts because the sandbox prevents a fresh run.
No surviving findings. The changed policy integrates correctly with the existing realization, action catalog, replay, and retained compatibility checks.

Cached activity, realization, and admission checks passed. All 863 preserved artifact hashes matched. Fresh Cargo verification was blocked by the read-only build lock; required-TLC evidence was checked in the committed execution archive.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Delivered by prerequisite tasks .1/.2; existing support consumed here. |
| R2 | met | Deadline/cancellation matrices, exact facts, fairness and countdown laws; native checks and retained TLC controls. |
| R3 | met | Exact retained lifecycle comparison, changed-edge control, and immutable-original verification. |
| R4 | deferred | Live admission and worker evidence belong to task .4. |
| R5 | deferred | Final design write-up belongs to task .5. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge (no blocking `introduced` findings, all R-IDs met or deferred)