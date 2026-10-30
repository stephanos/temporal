## Issue 1
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/verify-current-refresh.py:48
- **R-IDs**: [R6]
- **Title**: Verify the committed recovery bundle copy
- **Problem**: All three draws: current checks hash only the external recovery bundle. Hash/size-bind the committed receiving-snapshots/prepared-snapshots.bundle as well, including corruption refusal and stable inventory coverage.

## Issue 2
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/receiving-snapshots/README.md:19
- **R-IDs**: [R6]
- **Title**: Qualify host-specific verifier dependencies
- **Problem**: Correctness2/contracts4: rerun currently requires original checkout path, live worktrees and external recovery stores. State that exact precondition rather than imply relocation portability; source data need not change.

## Issue 3
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/receiving-snapshots/verify-snapshots.py:178
- **R-IDs**: [R6]
- **Title**: Compute support and deletion recovery claims
- **Problem**: Correctness3: actual scripts do not compute the28 canonical-history support reachability or beforeimage_blobs_not_in_target_bundle claims. Verify actual canonical reachability and bundle reachability, publish computed results and explicit recovery for any missing blobs.

## Issue 4
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/refresh-current-readiness.py:57
- **R-IDs**: [R6]
- **Title**: Derive tracker counts from observations
- **Problem**: Correctness4/contracts3: compute unique requested_task count and assert uniqueness instead of literal60 in refresh and verification receipts.

## Issue 5
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/receiving-snapshots/README.md:31
- **R-IDs**: [R6]
- **Title**: Correct transaction line count
- **Problem**: Correctness5/contracts2: transaction is48refs/51lines, including start prepare commit.

## Issue 6
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/receiving-snapshots/snapshot_common.py:100
- **R-IDs**: [R6]
- **Title**: Cover actual canonical system import spellings
- **Problem**: Integration1: system-import probe omits actual go.temporal.io/ and temporal API spellings prohibited by source policy. Check actual import/dependency forms and add corresponding refusal controls; do not discard legitimate fixture bytes or silently skip impure candidate paths.

## Issue 7
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/refresh-current-readiness.py:19
- **R-IDs**: [R6]
- **Title**: Bind clean worktree claim to its own HEAD
- **Problem**: Integration3: assert consumer worktree HEAD equals3f5 in addition to branch-ref identity before claiming clean at that revision.

## Issue 8
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/verify-current-refresh.py:38
- **R-IDs**: [R6]
- **Title**: Compare current and frozen receiving identity maps
- **Problem**: Integration4: prove actual candidate IDs, source heads, receiver refs and provisional mappings match frozen48plan after excluding observational recovery metadata.

Format-repaired merge of actual three SHIP reviews; correctness used an invalid composite anchor. Duplicates collapsed onto these eight issues. Raw reviews unchanged. Source selection independence is intentional; optional refactors/shared-oracle merging declined because they weaken independent verification or identify no current failing requirement. Verdict is preparation fitness only; no actual receivers/publication/cutover or task5 completion claimed.

```json
{"classification_counts":{"introduced":8,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>
