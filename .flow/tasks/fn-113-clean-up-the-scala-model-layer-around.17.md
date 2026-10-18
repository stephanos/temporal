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
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: