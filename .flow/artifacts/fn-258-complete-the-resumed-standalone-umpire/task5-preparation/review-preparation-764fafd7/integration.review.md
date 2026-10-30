Review complete. Scope: the task5-preparation scripts and receiving-snapshots evidence against base d1634eb04e, with the integration axis (how the new scripts meet the unchanged exporter, raw-Git verifier, custody capture, ledger and Flow records).

## Summary

The preparation is fit for its stated purpose. The builder writes only to the isolated store and this directory, the verifier reprojects raw Git trees independently of `ls-tree`, the 50 custody commits carry agent identity and no original parents, bundle recovery proves an exact 50-commit graph, the 16 refusal controls exercise real code paths, and the create-only transaction was exercised only in `transaction-test.git` with replay refused (exit 128, "reference already exists"). Canonical standalone 054, source refs and dirty custody fingerprints are checked unchanged before and after. The 049 four-path exception is pinned by commit, mode and blob and refuses drift. Interfaces into unchanged code line up: `require` in the exporter raises `RuntimeError` (caught for 049 only), `Projection.snapshot` takes the rewritten `sets`, `commit_parts` returns `(tree, parents, identity)` with identity tuples as the verifier expects, `Repository.leaves` yields `(bytes, (str, str))`, `capture-custody.py` is import-safe and exposes `fingerprint`, and the filtered allowlist leaves only `LICENSE` plus `.history-support/` entries so the `D_rust_birth` branch of the projector is never reached.

## Findings

- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/receiving-snapshots/snapshot_common.py:100`
- **R-IDs**: []
- **Problem**: The content-level "system-import" check matches only `name = "temporal-…"`, `use umpire_temporal`, `umpire-temporal… =` and `github.com/temporalio/temporal`. The real Temporal Go module path is `go.temporal.io/server` (root `go.mod`), and the framework's own `model/umpire/dev/gate/policy/framework-boundary.txt` forbids `go.temporal.io/`, `temporal.api`, `temporal::api`, `temporal/api`. The prepared targets contain 1,868 `hosts/go/*.go` records; a historical one importing `go.temporal.io/server/...` or `go.temporal.io/api/...` would pass `safe()`, yet `verification.json` records the `system-import` control as proving "no system dependencies". The single control only exercises the `temporal-xtask` package-name case.
- **Suggestion**: Extend the regex with `go\.temporal\.io/` and `temporal(::|\.|/)api` (or load `forbidden` from the policy file), add a Go-import refusal control, and rerun `verify-snapshots.py` so the receipt reflects the stronger check.

- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/verify-current-refresh.py:48`
- **R-IDs**: []
- **Problem**: The committed review copy `receiving-snapshots/prepared-snapshots.bundle` is never hash-verified. The refresh verifier hashes the external `.recovery/...` path from `target-summary.json`; `root-review-inputs.json` and the README only declare the SHA. If the in-repo copy was taken before the final build or is corrupted, a reviewer unbundling it would verify a different graph than the one the receipts describe.
- **Suggestion**: In `verify-current-refresh.py`, assert `sha(snapdir/'prepared-snapshots.bundle') == summary['bundle']['sha256']` and the size, and record it in the result.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/refresh-current-readiness.py:19`
- **R-IDs**: []
- **Problem**: `consumer_worktree_clean_now: true` is derived from `git status` in `.worktrees/fn2583-consumers` after checking that the branch ref `refs/heads/work/fn2583-consumers` resolves to 3f5. The worktree's own HEAD is never checked, so a worktree checked out elsewhere would still be reported as "clean at 3f5".
- **Suggestion**: Add `assert git(worktree,'rev-parse','HEAD').decode().strip()==PAIR['temporal']`.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/task5-preparation/verify-current-refresh.py:38`
- **R-IDs**: []
- **Problem**: The 48 targets were built from the frozen `receiving-ref-creation-work.input.json` (overlay 1abb40…), while the live overlay is now 702d81…. The README asserts the refresh "does not silently rebind these 48 original candidate identities", but no script compares the current plan's `work` entries (candidate ids, heads, receiver refs, provisional mappings) to the frozen input. Today they agree because both derive from the immutable ledger, but the claim is unverified by tooling.
- **Suggestion**: In `verify-current-refresh.py`, load both plans and assert equality of `work` entries after dropping `source_recovery` and `overlay_sha256`.

**FYI (not findings):** `snapshot_common.rules()` generalises the approved projector from ancestry-path-to-SOURCE sets to descendant-of-boundary checks so off-ancestry candidate tips can be projected; this is equivalent for in-ancestry revisions and documented, but "exact-approved-projector-match" is a match against the projector with rewritten `sets` and `H`. Code-smell baseline: `validate()` in `disposition-overlay.py` (~90 lines) and `main()` in `build-snapshots.py` (~130 lines) are Long Methods; `task_record()` re-parses the 2.7k-line tracker JSON on every call (perf only); `(repo, ref, oid)` triples recur as a Data Clump.

Suppressed findings: 1 (a 25-confidence note that `selected()` pins the four excluded 049 paths but does not assert `model/dev/xtask/` holds nothing else at 86f8126; the `temporal-xtask` Cargo.toml has no `src/`, so extra files are unlikely).

Classification counts: 4 introduced, 0 pre_existing.

```json
{"suppressed_count":{"25":1},"classification_counts":{"introduced":4,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>