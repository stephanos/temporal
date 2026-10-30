**Scope reviewed:** `disposition-overlay.py`, the refresh/tracker evidence (`refresh-current-readiness.py`, `verify-current-refresh.py`, `current-refresh-verification.json`, `current-tracker-observations.json`, `source-readiness-closures.json`, `receiving-readiness.json`) and the listed `receiving-snapshots/` files. I cross-checked the stated counts and identities against the data files rather than re-running anything (no shell in this session).

**What holds up**

- 60 tracker observations are present, fn-254.17 is observed `done`, the 22 definition-vs-observed differences are computed from overlay links and asserted, and tasks 1/2/3 `done` / task 4 `in_progress` come from `flowctl show` with `status_source: flow-state`.
- All 48 planned candidates have `source_original_head == source_r1_observed_head` (two have null original, falling back to observed), so builder and overlay agree on the snapshot commit for every target.
- The 049 exception is pinned by commit, four exact path/mode/blob tuples and keeper ref in three places (policy JSON, `snapshot_common.EXCEPTION_FILES`, verifier), with drift/extra-prefix controls. Target summary shows 1163 files + 4 exception paths = 1167, matching the ownership disposition. The `.history-support` 28 blobs are recorded as not copied.
- The create-only transaction is `start / 48×create / prepare / commit`, pinned to the exact target OIDs; the replay refusal receipt shows `cannot lock ref … already exists`, and `verify-snapshots.py:132` plus `verify-current-refresh.py:57-58` both check the real standalone has no `refs/recovery/fn258/*`.
- Custody claims (075: 45 raw dirty files = 40 untracked + 5 modified; 039: 3 unmerged stages) match the custody manifest. Graph check (`rev-list` of 48 heads == exactly 50 custody commits) and author/parent spoof controls back the "no fabricated history" claim.
- The provenance correction is reconstructible: old manifest hash is recomputed by substituting prior bindings, and the verifier hash is identical before/after, so only `snapshot_common.py` changed.

## Findings

- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/receiving-snapshots/root-review-inputs.json:3` (also `verify-current-refresh.py:48`, `stable-preparation-hashes.json:270`)
- **R-IDs**: []
- **Problem**: `root-review-inputs.json` presents the in-repo `prepared-snapshots.bundle` as an "exact copy" with SHA `efb9a700…`, but every committed check hashes the external `/Users/stephan/…/.recovery/…/prepared-snapshots.bundle` (`verify-current-refresh.py` via `target-summary.json.bundle.path`; `stable-preparation-hashes.json` lists only the external file). A reviewer working from the repository alone, or a truncated/mis-copied committed bundle, passes all committed verifiers unnoticed. The pinned-OID transaction would eventually fail on import, but the preparation evidence claims a verified copy it never verifies.
- **Suggestion**: In `verify-current-refresh.py`, also hash `receiving-snapshots/prepared-snapshots.bundle` and assert it equals `root-review-inputs.json.bundle_sha256` and `target-summary.json.bundle.sha256`; add the committed copy to `stable-preparation-hashes.json`.

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/receiving-snapshots/README.md:31`
- **R-IDs**: []
- **Problem**: README says "The only executed48-line proposed receiving transaction"; the file is 51 lines (48 `create` lines plus `start`/`prepare`/`commit`). The review brief itself calls it 51-line.
- **Suggestion**: Say "48-ref (51-line)" or just "48-ref".

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/refresh-current-readiness.py:57` (and `verify-current-refresh.py:59`)
- **R-IDs**: []
- **Problem**: `unique_task_count: 60` and `current_tracker_observations: 60` are literals, not derived from `current-tracker-observations.json`, and uniqueness of `requested_task` is never asserted. It happens to match today (60 entries), but a regenerated observation file with 59 or 61 entries would still be reported as 60.
- **Suggestion**: Compute `len({o['requested_task'] for o in observations})` and assert it equals `len(observations)`.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/receiving-snapshots/snapshot_common.py:53` (with `verify-snapshots.py:16`)
- **R-IDs**: []
- **Problem**: `inputs()` records absolute `/Users/stephan/…` paths and the verifier asserts `report['input_bindings'] == c.inputs()`. The README's "rerunnable from the Temporal repository root" therefore only holds at that exact checkout path, with the external `.recovery` stores and the live 039/075 worktrees present (`verify-snapshots.py:19-23`). Any other clone fails on the first binding assert rather than on a substantive check.
- **Suggestion**: Bind paths relative to `TEMPORAL` (as `disposition-overlay.py` already does) or document the host-only precondition explicitly in the README.

**FYI (not affecting verdict)**

- `disposition-overlay.py:160` resolves recovery as `original_head or observed_head` while `build-snapshots.py:18/39` and `verify-snapshots.py:91` use `observed or original`. No effect on the 48 (all equal), but a future ledger refresh that diverges them would snapshot a commit the overlay's `required_original_or_observed_object` doesn't name.
- `snapshot_common.safe()` content guard is a regex heuristic (`name = "temporal-…`, `use umpire_temporal`, `github.com/temporalio/temporal`). A `temporal-xtask = { path = … }` dependency line or other consumer coupling would pass it; the "no system import" claim really rests on prefix selection plus this spot check, which the README should say.
- `verify-current-refresh.py:53` trusts `consumer_worktree_clean_now` from the JSON rather than re-observing; fine for frozen-evidence verification, but it is not a live check.
- Code smell: `verify-snapshots.py main()` (~190 lines) and `disposition-overlay.py validate()` are Long Methods; acceptable for one-off evidence scripts.

## Requirements coverage
Skipped by instruction: this is preparation-fitness only; R6 (actual dispositions, receivers, access, publication, cutover) is explicitly deferred to task 4 acceptance and task 5 admission, and R2–R5 were reviewed separately.

Classification counts: 4 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":4,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>