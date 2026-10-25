I’ll read the task contract first, then trace the changed code and its focused validation evidence for correctness issues.
The timeout path matches the approved behavior: it decrements budgets, gives global expiry priority, and retries eligible local expiries while retaining the selected fact. I’m checking the renewed laws, exact-edge controls, and generated transitions for gaps.
All 12 focused checks passed using the retained executable, whose hash and source manifest match the reviewed files. These cover the timeout matrix, exact retry edges, source mutants, selected-World agreement, global bounds, fairness, and the original lifecycle checkpoint. The recorded TLC run timed out; I’m checking its completed tests against the successful complement.
No surviving correctness findings.

Verification: 12 focused checks passed using the hash-matched retained executable, covering retry semantics, exact facts, timer resets, selected-World agreement, mutants, fairness, global bounds, and the original lifecycle checkpoint. All 43 artifact hashes matched. Recorded required-TLC logs contain 39 unique passing tests across the timed-out run and successful complement; the whole-command timeout remains inconclusive.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | deferred | Other runtime/replay task owners. |
| R2 | deferred | Other capability task owners. |
| R3 | deferred | Other catalog/boundary task owners. |
| R4 | deferred | Other binding/probe task owners. |
| R5 | partial | Task .15’s approved correction and focused proofs are covered. Full protocol/refinement work remains deferred to .13/.4; conductor validation remains pending. |
| R6 | deferred | Worker-policy task owners. |
| R7 | deferred | Measurement/documentation task owners. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge; no blocking introduced findings.