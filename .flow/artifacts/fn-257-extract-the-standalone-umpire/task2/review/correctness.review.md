I’ll read the task specification first, then review the export helpers and bounded evidence, focusing on Git isolation and history preservation.

The task contract is R2: preserve the selected history in an isolated clone and prove the exported checkpoint matches the frozen source. I’m checking the merge callback, pruning checks, and preservation evidence next; build and runtime gates remain outside this review.

The read-only verifier passed: all 7,149 available commits were checked, the 1,178 active files matched, and source state stayed unchanged. A separate parent-frontier comparison found no ancestry mismatches across the 581 retained commits. I’m checking whether the verifier itself would reject a changed merge graph.

- **Severity:** P2
- **Confidence:** 75
- **Classification:** introduced
- **File:Line:** `.flow/artifacts/fn-257-extract-the-standalone-umpire/task2/verify-export.py:344`
- **R-IDs:** [R2]
- **Problem:** The verifier discards rewritten parent IDs and checks only reachability, metadata and snapshots. An in-memory negative control added the sibling archive move as a parent of the canonical move, recomputed commit IDs and updated the map; verification still reported zero failures. It therefore accepts incorrect ancestry. The committed export itself passed a separate parent-frontier comparison.
- **Suggestion:** Compare each retained commit’s rewritten parents against the retained frontiers of its original parents, accounting for permitted pruning.

Verification passed for all 7,149 available source commits and 1,178 active files. The 581 retained commits had no parent-frontier mismatches. Preservation evidence matched, both commit maps were identical, and strict export integrity checks passed.

FYI, heuristic only: Long Method — `def verify(args, report, failures):` combines history, inventory and preservation verification.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | deferred | Owned by tasks 1 and 6; outside this review. |
| R2 | met | Exact tree, metadata, ancestry, isolation and preservation checks passed for the committed export. |
| R3 | deferred | Portable checkout work belongs to task 3. |
| R4 | deferred | Consumer wiring belongs to task 4. |
| R5 | deferred | Execution gates remain under the owner hold. |
| R6 | deferred | Publication and cutover belong to task 6. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge (no blocking introduced findings, all R-IDs met or deferred)