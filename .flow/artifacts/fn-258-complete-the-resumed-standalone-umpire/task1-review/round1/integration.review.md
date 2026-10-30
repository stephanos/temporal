Evidence gathering is complete. Writing up the verdict now.

## Findings

**Finding 1**
- **Severity**: P1
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/history/verify-export.py:251`
- **R-IDs**: [R2]
- **Problem**: The one semantic change to the independent verifier is the checkpoint guard, relaxed from `mapping[SOURCE] == checkpoint` to `mapping[SOURCE] in {checkpoint, None}`. When the source tip is pruned (the actual case here: commit map row 4948 maps ac60 to zeros), `None` is always in that set, so the supplied `--checkpoint` is bound to nothing. The later walk computes `frontiers[SOURCE]` and checks its tree equals the projection, but never requires `checkpoint` to be that frontier, and nothing requires every commit in `export_graph` to be a mapped commit. Concrete failing scenario: take the bundle, add commit X on top of a92db06 with the same tree but a bogus author, or with a second parent pointing at a foreign history, and run the verifier with `--checkpoint X`. Every retained commit is still reachable, every mapped identity still matches, the active tree still equals the manifest, so the report says `pass`. The README's claim at `history/README.md:34-37` that zero-row acceptance is "backed by ... reachability checks" is not what the code enforces. The negative controls do not cover this: `run-controls.py` mutates a mapped commit's parents, which the rewritten-parent-order check catches, but an unmapped tip is invisible to every check. This matters most for `bundle-recovery-verification.json`, which is exactly the fresh-clone consumer run where a tampered tip would be accepted.
- **Suggestion**: After the ordered walk, when `mapping[SOURCE] is None` require `frontiers[SOURCE] == {checkpoint}` (the unique nonredundant retained frontier), and in all cases require `set(export_graph) == {new for new in mapping.values() if new}` so foreign or synthetic commits fail. Add a negative control that appends an unmapped tip commit to a synthetic clone and asserts refusal. Optionally mirror the frontier binding at `export-history.py:372`, where the pruned-tip branch likewise leaves `export_head` unbound (the complete-tree check at line 380 covers tree identity but not parents or metadata).

**Finding 2**
- **Severity**: P2
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/history/prepare-history-map.py:31`
- **R-IDs**: [R2]
- **Problem**: The rebinding script recomputes `count`, `sha256_sorted_lf_terminated` and `members` for every ancestry set but leaves each `definition_command` untouched, so the committed `history-map.json` now records `git rev-list c88fdc55...` at line 159 next to a count of 7261, which is ac60's ancestry (c88's is 7149). The same stale revision appears in all thirteen `D_*` definitions (lines 164-224) and in `representative_history[].command` (line 2063), whose `history_commit_count` and hashes were never recomputed at ac60. A reader reproducing the map by its own recorded commands gets a mismatch on every set.
- **Suggestion**: In `prepare-history-map.py`, rewrite each `definition_command` with `SOURCE` and either recompute the representative histories at ac60 or record explicitly that they are c88-bound provenance. Regenerate `history-map.json` and `history-map-reproduced.json`; the verifier reads neither field, so hashes bound into `preflight.json` and `verification.json` will change but the proof logic will not.

**Finding 3**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/source/candidate-ledger.py:146`
- **R-IDs**: [R1]
- **Problem**: `capture()` computes `original_head_object_present` with `check=False`, then unconditionally runs `for-each-ref --contains <original_head>` with `check=True`. If an original candidate object is ever absent, that call raises `CalledProcessError`, which nothing catches, so no ledger is written at all. R1 says an unknown stays unknown; here it aborts the inventory instead. Not hit today, since all 81 original heads are present.
- **Suggestion**: Guard the `--contains` call on `original_head_object_present` and record an empty list plus an explicit gap entry when the object is missing.

**FYI (not findings)**
- `candidate-ledger.py:13` hardcodes the standalone sibling as `TEMPORAL.parent / 'umpire'`; the verifier fails outright without that checkout. Fine for this environment, but the README's reproduction command silently assumes it.
- `original-history-continuity.json`, `checkpoint.json` and `bundle-proof.json` have no generating command recorded in `tool-provenance.json`. Process-level only.
- Task 2 through 5 markdown edits are in the range but out of this review's scope per the focus instruction.

## Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R1 | met | `source/source-manifest.json` (1189 records with mode, blob, size, sha256) verified by `candidate-ledger.py` and `verify-export.py`; `destination-access.json` records the probe as unknown; ledger has 88 entries, 81 original candidates, 0 transferred, only `not-transferred` / `ongoing-preparation-only` statuses; the removed `fn2551-prep-generator` keeps its original OID and recovery ref; `preservation-comparison.json` before equals after; fn-257 artifacts untouched in the range |
| R2 | partial | Isolated clone guards, 627 retained identities, 6634 pruned neutrality checks, active tree equals manifest, c88 row unchanged and cf8bec4 an ancestor, both archive-move lineages in R, isolation controls refuse source and registered worktrees. Checkpoint binding gap is Finding 1. The join into the standalone is owned by task 2. |
| R3 | deferred | Task 2 |
| R4 | deferred | Tasks 3 and 5 |
| R5 | deferred | Task 4 |
| R6 | deferred | Task 5 |

Unaddressed R-IDs: []

Classification counts: 3 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>