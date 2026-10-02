I’ll read the task spec first, then trace the diff and its callers against the acceptance criteria.
The default selection removes answered seeds, and the plan records the reduced selection and regression mode. I’m checking resume, shards, and empty campaigns next, where this change also affects validation and reporting.
The test suite cannot start because the read-only sandbox blocks Go from creating its build directory. I’m continuing the source review, focusing on frozen guidance across shards and the meaning of the reported counts.
1. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `tools/gomad3/runner/internal/campaign/merge.go:445`  
   **R-IDs:** [R6]  
   **Problem:** Fully answered campaigns produce merged output that cannot be reopened. This change permits empty merges, but `validateMergedCampaign` constructs a `CampaignRecord` without `Guidance`. Its call to `validateCampaign` therefore rejects the zero selection with `campaign record identity is invalid`. Merge succeeds, then inspection fails. The new empty-merge test never reopens its output.  
   **Suggestion:** Preserve the guidance evidence through merged-record validation, and extend the regression test to reopen and inspect the merged campaign.

2. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `tools/gomad3/cmd/gomad/internal/cli/explore_output.go:132`  
   **R-IDs:** [R6]  
   **Problem:** An empty shard incorrectly reports “all requested seeds are answered.” `summary.SelectionCount` counts seeds assigned to that shard, while `Guidance` describes the whole plan. For a guided plan with one unanswered seed and two shards, shard `1/2` prints this claim despite `requested=1 answered=0`. Filtering answered seeds makes empty shards especially likely.  
   **Suggestion:** Check that `guidance.Requested > 0 && guidance.Answered == guidance.Requested`, and test an empty shard of a partially answered plan.

Both required validation commands were attempted, but the read-only sandbox prevented Go from creating its build directory.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R6 | partial | Selection filtering, frozen plans, regression mode, resume checks, and execution counts are implemented; empty-merge reopening and empty-shard reporting remain defective. |
| R1–R5, R7–R12 | deferred | Assigned to other tasks in the epic; outside this task’s review scope. |

Unaddressed R-IDs: []

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>
