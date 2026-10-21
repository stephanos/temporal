# fn-217-annotation-catalog-over-the-temporal.7 Rename model/temporal/features/ to model/temporal/feature/

## Description
Owner (2026-10-20): rename `model/temporal/features/` to `model/temporal/feature/` (singular, matching the planned `model/temporal/foundation/`). Do it before fn-217.6 so the foundation split and the `api.rs` rewrite land in the final layout.

Touch points found on 2026-10-20 (outside the directory itself):
- `model/temporal/api/build.rs:47` catalog input root.
- `model/temporal/realization/src/lib.rs` `#[path]` attributes for `api.rs` and `temporal.rs`.
- `model/temporal/realization/tests/realization.rs` `include_str!` of the expected snapshots; `realization/tests/ui/supplemental_*.rs` `#[path]` attributes and their `.stderr` files (regenerate with trybuild).
- `model/temporal/realization/README.md`, `model/README.md` (fixture roots sentence), `model/DESIGN.md` (activity model reference).
- `Makefile`: `MODEL_FEATURE_ROOT` and the `temporal.features.` package prefix in `MODEL_UNLISTED_PACKAGES`; check whether that prefix is a path or a Scala package name from the archive before changing it.
- `model/temporal/features/activity/expected/lift.snap.txt` carries the relative path in diagnostics; regenerate.

Verify with `make model-check` and the Makefile lint/lift targets that read `MODEL_FEATURE_ROOT`.

**Files:** the existing feature directory, API build root, Realization module paths and fixtures, Model documentation, Makefile.
**Touches:** `model/temporal/features/**`, `model/temporal/feature/**`, `model/temporal/api/build.rs`, `model/temporal/realization/src/lib.rs`, `model/temporal/realization/tests/**`, `model/temporal/realization/README.md`, `model/README.md`, `model/DESIGN.md`, `Makefile`, `model/temporal/testing/build.rs`, `model/temporal/testing/fixtures.rs`, `model/temporal/api/tests/build_inputs.rs`, `model/design/second-opinion.md`, `tests/testpilot_model_gaps_test.go`
**Quick:** scoped old-root reference check, workspace format check, Realization trybuild and fixture snapshots, applicable Makefile fixture-root targets; `make umpire-check-model-gaps` (actual live probes and logger canary), including separated Go path components in the operational reference audit.

Keep historical Flow specs, task descriptions, proof evidence and archived Scala package names intact. The old-path check covers active source, build configuration and documentation; exclude `.flow/`, `.worktrees/`, generated build/cache trees, and `model/archived/`. Regenerate path-bearing goldens with their owning tools. Do not change panic behavior or any annotation, Model, protocol or judgment semantics.

## Acceptance
- [ ] `model/temporal/feature/activity/` exists and no `model/temporal/features/` remains; every operational reference listed in the description is updated and the scoped old-root search over active source, build configuration and documentation is empty.
- [ ] `make model-check` passes; snapshot and trybuild `.stderr` files are regenerated, not hand-edited.


## Done summary
The active Temporal authoring root is now `model/temporal/feature/activity/`. All 20 files moved; 19 retain their bytes and the lift diagnostic changes only its source path. Owning tools regenerated three trybuild diagnostics. Model, catalog, report, rows, states and gap semantics remain unchanged.

Review found a missed Go `filepath.Join` consumer. The actual live gap gate reproduced its missing-file failure, then passed after the single path-component fix. All 62 probes and the failure-preserving privacy canary pass with the unchanged golden. Scoped Go lint reports zero issues.

Both final full Model gates passed on `91c5dfd56f`, including 281 Rust tests and both Go hosts. Required TLC checks the admitted shared corpus; the activity Model remains source-refused and has no TLC agreement claim. Archived Scala paths and its Makefile lint/lift dry run remain unchanged.

Review passed after one fix. Three first-round draws found one deduplicated integration issue; the single re-review returned SHIP with zero surviving findings at source `91c5dfd56f3dff0fea45c3d778a930888f425713`. Reviewer and implementer are same-family GPT contexts. Reviewer ran `gpt-6.1-sol` at high effort. Receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-217-annotation-catalog-over-the-temporal.7.json`.

Conductor verification checked all 20 moved files, the fragmented Go path and archive dry-run equivalence. The helper uses the task-unique saved base. Source range: `5bf8bc014cfe8345936f4fdba57b2c15ce78f2a3..91c5dfd56f3dff0fea45c3d778a930888f425713`.

Gates: baseline reused (green receipt 91c5dfd5, unittest).
Gates: baseline reused (green receipt 91c5dfd5, required TLC).
Gates: baseline reused (green receipt 91c5dfd5, live model gaps).
Tier: session (jev-unavailable(no_key)); project implementer selection honored.
stage: impl-review - ran (Codex fan-out, one fix, single re-review; model: gpt-6.1-sol)
stage: memory - skipped(policy: mechanical path correction)
Tracker sync: n/a (bridge inactive).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cb739765e88dd5cbd754bf8b46b3bb7653446d88, 284c28d410a64d1f910b5035d233fef52af4b5ad, 91c5dfd56f3dff0fea45c3d778a930888f425713
- Tests: baseline: green (make model-check and focused tests; no handoff), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2177-rename GOCACHE=/home/agent/.cache/codex-build/fn2177-go make model-check, cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2177-rename mise exec -- cargo test -p umpire-temporal -p umpire-temporal-tests -p umpire-temporal-api --quiet, cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2177-rename INSTA_UPDATE=always TRYBUILD=overwrite mise exec -- cargo test -p umpire-temporal -p umpire-temporal-tests -p umpire-temporal-api --quiet, cd model && mise exec -- cargo fmt --all, FAILED: first post-change make model-check exited 2 at fmt; .flow/tmp/fn2177-verify-model-check.log; corrected in 284c28d410, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2177-rename GOCACHE=/home/agent/.cache/codex-build/fn2177-go UMPIRE_BACKENDS=require make model-check, bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2177-verify-layout.sh, make -n lift-model lint-model (dry run only; baseline and final output identical), INCONCLUSIVE: relative layout-script invocation exited 127; corrected absolute invocation passed, fix-round baseline: RED at 284c28d410; make umpire-check-model-gaps exited 2 because old golden path was missing (.flow/tmp/fn2177-fix-red-gaps.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2177-rename GOCACHE=/home/agent/.cache/codex-build/fn2177-go make umpire-check-model-gaps (GREEN; 62 live probe subtests and logger failure-preservation canary; .flow/tmp/fn2177-fix-green-gaps.log), GOCACHE=/home/agent/.cache/codex-build/fn2177-go make lint-code-fast GOLANGCI_LINT_BASE_REV=284c28d410a64d1f910b5035d233fef52af4b5ad GOLANGCI_LINT_FIX=false TEST_TAG=integration (GREEN; 0 issues), gofmt -l tests/testpilot_model_gaps_test.go (no files), fix-round make model-check GREEN at 91c5dfd56f (.flow/tmp/fn2177-fix-verify-model-check.log), fix-round UMPIRE_BACKENDS=require make model-check GREEN at 91c5dfd56f (.flow/tmp/fn2177-fix-verify-required-tlc.log), fragmented Go reference audit: 27 quoted features components before fix, 26 after; preserved Scala/negative fixtures; zero old activity gaps join references, conductor: task-unique-base layout verifier passed all 20 moved files, zero active old-root references, zero fragmented old gaps path, archive dry-run equivalence; git diff --check passed, conductor: current-head unittest, required-TLC and model-gaps green receipts honored at 91c5dfd5; no duplicate full gates run, formal Codex review: three first-round draws merged one P2; fixed in 91c5dfd56f; single re-review SHIP at 2026-10-21T01:50:03.289659Z, gpt-6.1-sol high, receipt /tmp/impl-review-receipt-8f37faba39e2-fn-217-annotation-catalog-over-the-temporal.7.json
- PRs: