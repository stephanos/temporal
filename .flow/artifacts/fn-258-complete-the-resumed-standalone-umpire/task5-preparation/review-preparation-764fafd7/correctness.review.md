**Outcome:** Preparation fitness is sound. I found no introduced P0/P1 in the in-scope scripts or evidence: selection, collision refusal, deletion before-images, custody authorship, the create-only transaction and the tracker/prerequisite provenance all hold up on trace. The findings below are evidence-chain gaps (P2) and small inconsistencies (P3).

## Findings

**Finding 1**
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/verify-current-refresh.py:48` (also `receiving-snapshots/verify-snapshots.py:178`, `receiving-snapshots/root-review-inputs.json:3-5`)
- **R-IDs**: `[R6]`
- **Problem**: The committed bundle copy `receiving-snapshots/prepared-snapshots.bundle` is the only recovery artifact inside the repo, but no checked-in verifier hashes it. Both verifiers hash the uncommitted `/Users/stephan/.../.recovery/.../prepared-snapshots.bundle` path from `target-summary.json` / `prepared-targets.json`. Scenario: the committed copy is truncated or stale while the `.recovery` store is intact. Every verifier still reports pass, and the "exact copy" claim in `root-review-inputs.json` is unbacked. If the `.recovery` store is later lost, the only recovery path is unverified.
- **Suggestion**: In `verify-current-refresh.py`, also assert `sha(snapdir/'prepared-snapshots.bundle') == summary['bundle']['sha256']` and the recorded size, so the committed copy is bound by the same evidence chain.

**Finding 2**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `receiving-snapshots/snapshot_common.py:53` and `receiving-snapshots/verify-snapshots.py:16,19-20`
- **R-IDs**: `[]`
- **Problem**: `inputs()` records absolute `/Users/stephan/...` paths and the verifier asserts byte-equality of `report['input_bindings']` against a fresh `inputs()`; it also opens the `.recovery` bare stores by absolute path. The README's "the verifier is rerunnable" holds only at this checkout path on this machine with the uncommitted `.recovery` store present. Elsewhere it fails at line 16 before verifying anything.
- **Suggestion**: Record paths relative to `TEMPORAL.parent` in `inputs()` (as `disposition-overlay.py` already does), and state the `.recovery` dependency in the README reproduction section.

**Finding 3**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `receiving-snapshots/README.md:19,33` and `receiving-snapshots/recovery-primitives.json:13`, `receiving-snapshots/candidate-049-ownership-disposition.json:72`
- **R-IDs**: `[R6]`
- **Problem**: Two claims the review is asked to assess have no in-scope script producing them: "all 28 qualified mixed support blobs are reachable in canonical export history" and "beforeimage_blobs_not_in_target_bundle: []". Neither `build-snapshots.py` nor `verify-snapshots.py` computes them. They are hand-asserted JSON. No data-loss exposure today because originals stay under retained Temporal refs, but the provenance of the claim is not reproducible.
- **Suggestion**: Add both checks to `verify-snapshots.py` (rev-list the recovery bundle for beforeimage blobs; `cat-file -e` the 28 blobs against the canonical export repo) so the receipt reflects computed facts.

**Finding 4**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `task5-preparation/refresh-current-readiness.py:57` and `task5-preparation/verify-current-refresh.py:59`
- **R-IDs**: `[]`
- **Problem**: `unique_task_count: 60` and `'current_tracker_observations': 60` are literals, not counts of `current-tracker-observations.json`. They match today (60 `requested_task` entries), but a re-capture with a different observation set would be reported as 60 regardless.
- **Suggestion**: Compute `len({o['requested_task'] for o in observations})` in both scripts.

**Finding 5**
- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `receiving-snapshots/README.md:31`
- **R-IDs**: `[]`
- **Problem**: Says "the only executed 48-line proposed receiving transaction". The file is 51 lines (`start`, 48 `create`, `prepare`, `commit`), and the root prompt itself describes it as 51 lines.
- **Suggestion**: Say "48-ref (51-line)" or drop the line count.

## FYI (not findings)

- **049 prefix guard is evidence-only.** `snapshot_common.selected()` and `verify-snapshots.project()` assert the four pinned files are excluded, but neither asserts that no *other* `model/dev/xtask/**` path is admitted at `86f8126`. The data shows exactly four files exist there (test-only package, `autotests=false`, no `src/`; base-only `model/dev/xtask/src/*` appear as net deletions), so there is no failing input today. An explicit assertion would encode the "consumer-owned prefix" policy rather than rely on the tree shape.
- **Verifier side effects.** `verify-snapshots.py` writes `proposed-create-only-transaction.txt` and creates `transaction-test.git` on first run. Isolated and deterministic, so harmless, but it is a verifier that mutates state.
- **Transaction semantics confirmed.** `start`/`create`/`prepare`/`commit` via `update-ref --stdin` is create-only and atomic; the replay refusal (`cannot lock ref ... reference already exists`, exit 128) is the correct signal. It requires the bundle to be unbundled first, which the README states.
- **Negative-control probe shares caches.** `negative_controls.run` shallow-copies `Facts`, sharing `object_cache`/`containment`. This works only because `missing_objects` and `refs` are consulted before cache lookups; a future control that mutates something else would silently read stale cache.
- **Frozen-evidence coupling.** `receiving-readiness.json` embeds `observed_utc`, so rerunning the refresh changes its hash, which propagates into `current-refresh-verification.json` and would stale `stable-preparation-hashes.json`. Acceptable for a frozen handoff; just not rerunnable without re-pinning.
- **Code smells**: Long Method in `build-snapshots.main` (~130 lines) and `verify-snapshots.main` (~190 lines); Duplicated Code between `snapshot_common.rules()` and the `allowed` expression in `verify-snapshots.py:32,73` (intentional independence, but the second copy could be a shared pure function taking the three ancestry booleans).

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | partial | Candidate ledger bound by SHA in overlay and prepared-targets; newer checkpoint frozen explicitly under `historical-preparation-3658582/` rather than rewritten; dirty/unintegrated work kept source-only, `transferred=false` for all 88 |
| R2 | deferred | Runtime source R2/R3/R4 reviewed separately per review scope |
| R3 | deferred | Out of scope per review instructions |
| R4 | deferred | Task 3 done source-readiness; task 4 in progress outside this repo |
| R5 | deferred | Task 4 gates execute outside this repo; not to be relied on here |
| R6 | partial | Preparation only: 48 targets, create-only transaction, tracker/keeper/prerequisite links per candidate; publication, destination inspection (SAML 403 recorded), cutover, post-archive identity explicitly open |

Unaddressed R-IDs: []

Classification counts: 5 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":5,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>