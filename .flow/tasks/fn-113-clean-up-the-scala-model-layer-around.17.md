---
satisfies: [R10, R13]
---
# fn-113-clean-up-the-scala-model-layer-around.17 Reuse interpreted Models in migration verification

## Description
Apply the performance instruction in MILESTONES.md to the measured repeated Model interpretation in migration verification. The fresh full Go timing baseline is `.flow/tmp/fn113-8-review/test-timings.json`: `TestMigrationProjectionPreservesSemantics/ir/nexus-close.json` takes 86.83 s and `TestMigrationGoldens` 30.19 s. `migrationBinding` interprets a Model, then `migrationMeaningOf` calls `Check` and interprets it again, despite the existing comment that each comparison interprets once.

**Size:** S
**Touches:** [tools/umpire/model/checking.go, tools/umpire/model/claims.go, tools/umpire/model/migration_golden_test.go, tools/umpire/model/nexus_close_test.go, tools/umpire/model/activity_system_test.go]

Use the smallest reuse of existing binding/checking code that removes duplicated construction within one comparison. A checked interpretation and the reader snapshots may share their immutable interpreted tables, but keep separate binding caches where realization fields require them. Each original, mapped and current Model comparison remains independently interpreted; witness replay still uses its separate fresh interpretation. Prefer a private seam; keep every public interface and Check receipt behavior unchanged. Reuse existing tests and strict migration goldens as independent oracles. Avoid introducing a caching or profiling framework or shared mutable cross-test fixtures. Preserve comments. No new libraries, schema, Model/IR/Case/expected changes, staging, commits, pushes or worktrees.

Before changing code, inspect task .13's final full model package JSON and wall-time evidence against the post-task-15 inputs. Record source and fixture digests and reuse that measured baseline when its command, source scope, fixtures and environment still apply; a task or agent transition alone is not a reason to rerun it. If an affected input invalidates it, measure the focused projection/golden tests first and repeat that same command afterward. After the change, run affected model tests and the model package/full goldens once with `CC=/usr/bin/clang GOMEMLIMIT=4500MiB`, `-tags test_dep -json -count=1 -p 1 -parallel 1`; compare equivalent-input test events and separately measured wall times using the same command. Preserve every assertion. Run `make lint-code-fast` against `origin/main` with `GOLANGCI_LINT_FIX=false`. Keep logs and timing evidence under `.flow/tmp/fn113-17/`, handover `.flow/tmp/fn113-17-summary.md`, evidence `.flow/tmp/fn113-17-evidence.json`. The closing task reuses this passing package baseline rather than repeating it. Report the measured outcome candidly; if the safe change does not reduce the bottleneck, keep the no-semantics-change constraint and diagnose the result instead of widening the framework.

## Acceptance
- [ ] Duplicated Model construction within migration comparisons is removed, with public Check behavior and fresh witness replay unchanged.
- [ ] Original, mapped and current comparisons and their semantic/declaration/refinement/Case assertions remain independent and strict; no shared mutable fixture state is introduced.
- [ ] Equivalent-input before/after test timings and wall time are recorded; affected full model tests, migration goldens and lint pass.
- [ ] Model sources, IR, Cases, expected fixtures and dependencies are unchanged; handover identifies baseline reuse for the closing task.


## Done summary
# fn-113.17 handover

Migration receipts now check a separate binding over the same interpreted machines used for that comparison's reader snapshots. Its claim caches are reset; original, mapped, and current comparisons each still bind independently, and witness replay still calls `checker.again()` for a fresh interpretation. Public `Check` and the strict migration assertions are unchanged. Task-only diff: `.flow/tmp/fn113-17/task-only.diff` (SHA-256 `a13350c4bb5be68cc194e786aa2ca4d18c472bf0fdee80eab746b3d9d005a99a`). The task 13 `TypesRenamed` additions already present in `migration_golden_test.go` are excluded from that diff.

Baseline reuse: task 13's full model command, CC, memory limit, fixtures and environment apply (`.flow/tmp/fn113-13/go-model-full-result.json`, exit 0, 151s wall, 802/802 test events passed). Task 15 changed documentation and comments only. Because task 13's later golden helper changed the Golden test's inputs after that full run, a current-source focused baseline was recorded before editing. All 32 task 13 artifact hashes still match; `.flow/tmp/fn113-17/before-source-fixture-hashes.json` and `after-source-fixture-hashes.json` cover 123 source/fixture/dependency files, with only the three task source files changed during this work. Model/Scala sources, IR, Cases, expected fixtures, and dependencies were not edited.

Equivalent focused command, pre/post: exit 0 both; wall 89.203s → 79.528s; `TestMigrationGoldens` 20.85s → 17.83s; `TestMigrationProjectionPreservesSemantics` 65.62s → 57.03s, including `ir/nexus-close.json` 60.16s → 52.30s. These are single runs, so the measured reduction is not a statistical speedup claim. The required full model package run passed in 104.219s wall with 802/802 identical test names and no failures; `TestCheckedOnceIsCheck` and the migration goldens passed. `mise exec -- make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast` passed with zero issues; `git diff --check` passed. Commands, JSON event logs, results, hashes, and timing comparison: `.flow/tmp/fn113-17/`. The closing task can reuse the passing full package result while these inputs remain unchanged. No lower/export/Scala gate was repeated because their inputs were unchanged.

No staging, commit, push, worktree, review, or Flow completion was performed. Task remains `in_progress` for conductor review.

stage: impl-review - ran (model: gpt-6-sol; receipt: .flow/tmp/fn113-17-review/receipt.json; verdict: SHIP; task-only source pinned by snapshot.diff/source-hashes.json)

stage: wave-dispatch - ran (model: gpt-6-sol; sequential worker in current checkout)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive).
## Evidence
- Commits:
- Tests: baseline: green via handoff (.flow/tmp/fn113-13/go-model-full-result.json; 151s; 802/802 test events passed), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -count=1 -tags test_dep -p 1 -parallel 1 -run "^(TestMigrationGoldens|TestMigrationProjectionPreservesSemantics)$" ./tools/umpire/model (pre-edit: exit 0, 89.203s wall), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -count=1 -tags test_dep -p 1 -parallel 1 -run "^(TestMigrationGoldens|TestMigrationProjectionPreservesSemantics)$" ./tools/umpire/model (post-edit: exit 0, 79.528s wall), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -count=1 -tags test_dep -p 1 -parallel 1 ./tools/umpire/model (exit 0, 104.219s wall; 802/802 test events passed), mise exec -- make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast (exit 0, zero issues), git diff --check (exit 0), Independent codex:gpt-6-sol:high implementation review SHIP; reviewed source identity verified
- PRs: