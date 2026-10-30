**Review scope:** fn-258 task 1 (R1/R2): renewed source manifest, candidate ledger, isolated history export at ac60daaf47, independent verifier, negative/isolation/preservation controls, bundle. Product/standalone/consumer work and tasks 2–5 are out of scope.

**What checks out**

- **Source manifest, ledger and delta are internally consistent.** `candidate-ledger.py` verifies all 1,189 manifest records against blob bytes/sizes and tree modes, reproduces the c88..ac60 delta and its patch digest, and reports 0 errors. The markdown counts (88 entries, 81 originals, 80 found, 36/15/28/9 ownership split, 26 originals without a corroborated task) agree with the JSON (33 gap strings minus 7 non-original entries). The one missing branch keeps its object under `refs/recovery/fn257-candidates/...`. No entry is labelled transferred; only four are `ongoing-preparation-only`.
- **History export covers the full available ancestry.** Commit map has 7,261 rows (627 retained, 6,634 zero rows), includes c88fdc5506→cf8bec4d62 and ac60→zero; the original 7,149 rows are unchanged and the old export is an ancestor. `verification.json` and `bundle-recovery-verification.json` pass with 1,189 active files, 30 `.history-support` files, source state equal before/after. Reproduction's commit map is byte-identical.
- **Source mutation safety.** Export guards reject shared/redirected git dirs, alternates, hard-links, registered worktrees; verifier fingerprints refs/index/registry/worktree bytes before and after. Negative controls mutate only the isolated clone and a bare synthetic clone, restore exactly, and each is rejected for the expected reason. Nothing under fn-257 artifacts is touched.

**Findings**

- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/history/verify-export.py:251` (and `export-history.py:372`)
- **R-IDs**: [R2]
- **Problem**: The relaxed tip guard accepts *any* checkpoint whenever the source tip's map row is zero. The only ancestry check that follows is one-directional (every mapped commit must be reachable from the checkpoint); nothing requires the checkpoint itself, or every commit in `export_graph`, to be a mapped value. Failing scenario: an export clone with one extra commit on top of a92db06 carrying the identical tree but a foreign author/message. Tree equality, worktree checks, metadata checks for the 627 mapped commits and reachability all pass, so the verifier reports `pass` for a history that contains an unattributed commit. The fn-257 verifier rejected this via `mapping[SOURCE] == checkpoint`; the README's "independent neutrality/reachability checks" claim therefore overstates what replaces that guard. The committed evidence is unaffected (map row 736 maps 192dd9ce→a92db06, which is the bundle head).
- **Suggestion**: After the loop, require `ordered_frontiers[SOURCE] == [checkpoint]` (the pruned tip's nearest retained frontier must be exactly the supplied checkpoint) and `set(export_graph) == {m for m in mapping.values() if m}`. Add a fourth control in `run-controls.py` that commits an empty-diff commit on a bare clone and expects rejection.

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/history/isolation-controls.json:5`
- **R-IDs**: [R2]
- **Problem**: The `source-refused` control's diagnostic is "Reports must remain outside the filtered clone", i.e. it tripped the report-dir guard at `export-history.py:313`, not the "original repository cannot be filtered" guard at line 80. README states this control proves source filtering is refused; it proves a refusal, but the identity guard was never reached. The invocation arguments are not recorded.
- **Suggestion**: Re-run with the report dir outside the source root so line 80 is the guard that fires, and record the arguments in the control entry.

**FYI (not affecting verdict)**

- `history/export/` has no `export-result.json` because the first run refused at the old tip guard after rewriting, and the script hash for that run is not retained. Byte-identical reproduction plus independent verification closes the gap; README discloses it.
- `refs_sha256` differs between `verification.json` (05:20) and `bundle-recovery-verification.json` (05:21): a source ref moved between runs (not by the scripts; each run's before/after is equal). Likely `origin/umpire`, as the ledger drift shows. Worth one line in the README.
- `candidate-ledger.py:204` executes a git argument vector read from committed JSON. Acceptable for evidence replay, but it trusts the artifact.
- Code smell (heuristic): `candidate-ledger.py:141-163` Long Method with a 20-key dict literal built inline; `run-controls.py` is dense one-liners that make the three controls hard to audit.

## Requirements coverage

| R-ID | Status | Evidence |
|------|--------|----------|
| R1 | met | `source-manifest.json` (1,189 records, trees, modes), `candidate-ledger.json`/`.md` with explicit ownership, recovery refs and `not-transferred` status, `candidate-ledger-verification.json` pass; prior-commit `destination-access.json` records the blocked probe |
| R2 | met (task 1 portion) | Isolated `--no-local` clone export, 7,261-row map, `verification.json`/`bundle-recovery-verification.json` pass, negative controls; join into the standalone is assigned to task 2 by the spec |
| R3 | deferred | task 2/4 |
| R4 | deferred | task 3/5 |
| R5 | deferred | task 4 |
| R6 | deferred | task 5 |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 2 introduced, 0 pre_existing.
Protected-path filter: 0

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>