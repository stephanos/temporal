**Finding 1**
- **Severity**: P1
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/history/verify-export.py:251`
- **R-IDs**: [R2]
- **Problem**: The fn-257 verifier pinned the export tip with `mapping[SOURCE] == checkpoint`. The fn-258 adaptation relaxes this to `mapping[SOURCE] in {checkpoint, None}`. When the source tip is projection-neutral (the actual case here: the ac60 row is zero), nothing bounds the checkpoint to the commit map. The remaining checks are one-directional: every retained mapped commit must be reachable from the checkpoint, and the checkpoint tree must equal the manifest. Concrete failing scenario: in an export clone run `git commit-tree a92db06^{tree} -p a92db06 -m injected` and pass that commit as `--checkpoint`. All 627 retained commits are reachable, the pruned ac60 frontier is still `{a92db06}` with an equal tree, the final tree equals the 1,189-file manifest, and the worktree checks pass. The verifier reports `pass` for a history containing an unmapped commit with arbitrary author, date and message. The README claims the zero row is accepted only "alongside ... independent neutrality/reachability checks", but reachability alone does not establish that the exported history consists of exactly the mapped commits. The negative controls (file corruption, missing map row, injected sibling parent) do not exercise this path. The committed evidence itself is sound because a92db06 is the mapped image of 192dd9ce (commit-map row 736), so this is a verifier gap rather than a bad artifact.
- **Suggestion**: After building `export_graph`, add `require(set(export_graph) == {new for new in mapping.values() if new}, "export-graph-bounded", ...)`, and when `mapping[SOURCE]` is None additionally require that the checkpoint is a member of the pruned source's computed retained frontier. Re-run verification and add a negative control for an unmapped tip commit.

**Finding 2**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/source/candidate-ledger.py:125`
- **R-IDs**: [R1]
- **Problem**: Dirty paths observed in the standalone `umpire` repository are assigned ownership `framework` without being inserted into `path_mappings`. The verifier at line 229 then rejects every such path with "candidate path lacks ownership mapping". Any future `--capture` taken while the standalone worktree is dirty produces a ledger that fails its own verification. The current run passed only because the standalone checkout was clean at observation time.
- **Suggestion**: Route standalone dirty paths through `ownership()` with a framework rule, or skip the mapping check for non-temporal entries.

**Finding 3**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/source/candidate-ledger.py:146`
- **R-IDs**: [R1]
- **Problem**: `for-each-ref --contains <original_head>` runs with `check=True`. If an original candidate's head object has been pruned, this raises `CalledProcessError` and aborts the capture instead of recording the candidate as missing with its recovery gap. Lines 115 and 223 are similarly inconsistent: capture tolerates a missing merge-base, verification does not, so an orphan branch would crash verification without a report.
- **Suggestion**: Guard the `--contains` call on `original_head_object_present` and make the verify-side merge-base call `check=False` with an explicit error entry.

FYI (not blocking): `verify()` in verify-export.py is a 245-line function and `capture()` builds a 25-key dict literal inline, both Long Method. The fn-258 scripts are near-verbatim copies of fn-257's with constants changed, which is Duplicated Code, but adaptation.json records this as the deliberate "new-only tooling, original unmodified" decision.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Manifest frozen at ac60 (1,189 records, blob/mode/size/sha256, framework and archive tree ids) and reverified by candidate-ledger.py. Ledger covers 88 entries including all 81 original candidates with recovery refs, task links, gaps and `not-transferred`/`ongoing-preparation-only` status only. fn-257 artifacts untouched. Destination probe failure recorded in execution-window.md. |
| R2 | met (with Finding 1 caveat) | Export produced in a detached `--no-local` clone with guard_repositories isolation; 7,261-row map covers all available source commits, original 7,149 rows identical, cf8bec4d old export is an ancestor; verification.json passes with 627 retained metadata/tree/parent comparisons and 6,634 neutral pruned commits; active projection equals manifest; source preservation before/after equal. |
| R3 | deferred | Task 2 |
| R4 | deferred | Tasks 3 and 5 |
| R5 | deferred | Task 4 |
| R6 | deferred | Task 5 |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 3 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>