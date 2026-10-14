---
satisfies: [R3, R10, R13, R24, R25]
---
# fn-113-clean-up-the-scala-model-layer-around.16 Close the spec: full validation, line counts and the facts for MILESTONES

## Description
Close the spec: full validation, line counts and the facts for MILESTONES. Implements R3, R10, R24, the R25 roll-up and R13's final run.

**Size:** M (long gates, little editing)
**Files:** fixes only for what the gates find; .flow/tmp/fn113-16/** (logs), .flow/tmp/fn113-16-summary.md (the handover with the MILESTONES and module map facts)
**Touches:** [.flow/tmp/fn113-16/**, any file a gate failure requires (name it in the summary)]

### Approach
- Apply the owner's verification instructions in `MILESTONES.md` to the commands below. Inspect task .13's serial model/lower/export JSON results and task .17's post-optimization model results; reuse each passing package only when its command, source scope, fixtures and environment remain applicable. Complete the remaining `./tools/umpire/...` packages and `go vet` serially with JSON output, exit statuses and wall times. Run the Scala gate in check mode with `--skip-go-checks` when these separate Go results cover its complete Go step; this is the documented gate option, not omitted coverage. Do not rerun the complete suite a second time solely to collect JSON. Record the exact evidence and inputs reused alongside newly run checks.
- Run once, serially, after every other task of the spec is done: `CC=/usr/bin/clang mise exec -- make umpire-check-model` (full: gate tests, vocabulary, generation, DSL alone, Models and their tests, lifter tests, lifts compared to `model/ir`, Cases, go vet and go test over `./tools/umpire/...`); `mise exec -- make lint-model`; `GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast` (R10) and plain `GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code`; `CC=/usr/bin/clang mise exec -- go test -count=1 -json -tags test_dep ./tools/umpire/... > .flow/tmp/fn113-16/full-go.jsonl` (goldens inside; keep exit status and wall time, per MILESTONES.md); `make umpire-check-cases umpire-check-fixtures canary-check-case`; `git diff --check`; `git status --short model/ir model/cases model/lifter/testdata` equals the regenerated state. Not run: `make umpire-check-backends` (deferred by the owner). `./common/testing/testpilot/...`, `./tools/canary/...` and the functional compile only if a file they import changed (none should have).
- R3: `model/project.scala` has no new `using dep`; files under `model/temporal` import only `umpire.*`, sibling Models and the standard library (grep imports).
- R24: count and record before and after. Planning-time figures (2026-10-02): `model/umpire` non-test 2,415, `model/temporal` non-test 5,496, Scala tests 961 (the spec's 2,873 / 5,062 / 982 predate fn-115.7; state both). Target: `model/umpire` non-test at most 1,300; if above, name what stayed and why. The lifter's count from task 3 (R11).
- Establish the test-count scope from the original tree and task .8's evidence. Report authoring tests (`model/umpire`, `model/temporal`) and all Model tests (including lifter and gate) separately when the historical totals use different scopes; never compare unlike scopes as a reduction.
- R25 roll-up: each task's library weighing (tasks 3, 8, 13) in one table; which was adopted (ScalaPB) and which not.
- R13: the goldens pass; state the projection the owner agreed to (task 5) or the re-capture that was done instead.
- Fix only what a gate finds; a fix reruns the affected gate, broader gates only when the change invalidates them.
- Handover for the conductor: the fn-113 section of `MILESTONES.md` (parts done, counts, deferred items, owner decisions taken), the module map rows (DSL, Lifter, Gate, Checker, goldens' projection), manifest deltas (files deleted and added across the spec), and the spec facts the tree contradicted (test count 33, line-count baselines, the `_$1` count).

### Investigation targets
**Required**:
- `MILESTONES.md:16-40,108-122`
- `.plans/UMPIRE_MODULES.md:22-30,58-85,265-281`
- `.flow/tmp/fn113-*-summary.md` (every task's handover)
- `Makefile:664-700`

### Quick commands
CC=/usr/bin/clang mise exec -- make umpire-check-model; mise exec -- make lint-model; GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast; GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code; CC=/usr/bin/clang mise exec -- go test -count=1 -json -tags test_dep ./tools/umpire/...; make umpire-check-cases umpire-check-fixtures canary-check-case; git diff --check; find model/umpire -name '*.scala' -not -path '*/test/*' | xargs wc -l; find model/temporal -name '*.scala' -not -path '*/test/*' | xargs wc -l; find model -path '*/test/*' -name '*.scala' | xargs wc -l

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] The complete model gate coverage, `lint-model`, `lint-code-fast`, plain `lint-code`, the complete `./tools/umpire/...` Go suite with goldens, `umpire-check-cases`, `umpire-check-fixtures` and `canary-check-case` pass, serially. Logs and exit statuses under `.flow/tmp/fn113-16/` identify both newly run checks and applicable passing evidence reused under the owner's `MILESTONES.md` instructions; separate Go verification permits the documented `--skip-go-checks` gate mode. `make umpire-check-backends` is recorded as deferred.
- [ ] No library a file under `model/temporal` can import was added (`model/project.scala` unchanged in dependencies).
- [ ] The summary states `model/umpire` (non-test, at most 1,300 or what stayed and why), `model/temporal` and Scala test line counts before and after against both the spec's and the planning-time baselines, the lifter's R11 figures, and the R25 roll-up.
- [ ] The handover gives the conductor the MILESTONES facts, the module map rows, the manifest deltas and the spec facts the tree contradicted.


## Done summary
# fn-113.16 final validation handover

All required closure coverage passed. No source fix was needed, and this task made no code or shared-document edit. Baseline: green via applicable task 13/15/17 handoffs; the owner’s MILESTONES rule avoids repeating their full suites. `make umpire-check-backends` remains deferred by the owner. No staging, commit, push, worktree, tracker action, review verdict, or `flowctl done` was performed.

### Gate evidence

| Check | Result and wall time | Evidence |
| --- | --- | --- |
| Remaining 13 `./tools/umpire/...` packages; `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -count=1 -tags test_dep -p 1 -parallel 1 <go-packages-remaining.txt>` | pass, 38s; 795 passing test events | `fn113-16/go-remaining.jsonl`, `go-remaining-result.json` |
| Reused `./tools/umpire/model` full package, same serial flags | pass, 104.219s; 802/802 test events | `../fn113-17/go-model-full.jsonl`, `go-model-full-result.json` |
| Reused `./tools/umpire/lower` and `./tools/umpire/export` full packages, same serial flags | pass, 235s/69s; 402/83 passing test events | `../fn113-13/go-{lower,export}-full.jsonl`, corresponding result JSON |
| `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go vet -tags test_dep -p 1 ./tools/umpire/...` | pass, 12s | `fn113-16/go-vet.log`, `go-vet-result.json` |
| `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` | pass, 64s; check-mode regeneration and fixture comparisons | `fn113-16/check-model.log`, `check-model-result.json` |
| `mise exec -- make lint-model` | exit 0, 8s | `fn113-16/lint-model.log`, `lint-model-result.json` |
| Reused `mise exec -- make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast` | exit 0, zero issues, 20.158s | `../fn113-17/lint-code-fast-result.json`, log in same directory |
| `GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main CC=/usr/bin/clang mise exec -- make lint-code` | exit 0, zero issues, 121s | `fn113-16/lint-code.log`, `lint-code-result.json` |
| `CC=/usr/bin/clang GOMEMLIMIT=4500MiB make umpire-check-cases umpire-check-fixtures canary-check-case` | pass, 36s | `fn113-16/cases-fixtures-canary.log`, result JSON |
| `git diff --check` | pass | `fn113-16/diff-check.log` |

The package inventory is `fn113-16/coverage.json` and `go-packages-all.txt`: all 16 packages have pass completion events and no failures. The split runs total 2,082 passing test events; **do not sum package durations as wall time**. Task 17 changed only the model package’s private checking/binding seam after task 13; its full current-source package run covers that change, while public `Check` and replay behavior stay pinned by its strict goldens and tests. Task 13’s lower/export source, their focused golden helper tests, the golden configuration and 32 artifact hashes remain applicable; task 15 changed prose/comments, and task 17’s 123 source/fixture/dependency hashes still match current bytes. The exact current hash map is `fn113-16/current-hashes.json`. The generated IR, Cases and expected fixtures match the task 13/17 recorded hashes after the fresh check-mode gate. `common/testing/testpilot/...`, `tools/canary/...` and functional compile were not run because no file those packages import changed. Model lint has the inherited Scalafix `NoSuchFieldException: path` trace even at exit 0, as task 15 recorded; scalafmt completed, and the project’s separate syntax probe is in `fn113-8-review/lint-probe-forbidden.log`.

### Counts and library weighing

Measured current tree (`fn113-16/counts.json`): non-test `model/umpire` **1,191** (target ≤1,300); non-test `model/temporal` **5,299**; authoring tests under `model/{umpire,temporal}/test` **12** lines; all tests under `model/`, including lifter and gate, **900** lines. The 2026-10-02 planning tree (`1f16ef5272`) measured **2,415 / 5,496 / 961** for the first three matching scopes and **1,719** all-test lines. The older spec figures **2,873 / 5,062 / 982** predate fn-115.7; they are recorded as historical estimates, not a matching-scope reduction. Task 8’s own measured before/after was **2,224→1,206** DSL, **5,414→5,414** Temporal, **908→12** authoring tests. Later Part D and prelude edits yield the final 1,191/5,299. The checker’s production code is **4,161→4,064** lines (current 4,064); nine typed test-support files totaling **1,342** lines left, with **29** lines of key helpers retained.

| Task | Existing/generic path vs library | Decision |
| --- | --- | --- |
| 3, lifter | Ten source files **2,107→1,911**; plus 16-line project file, **2,123→1,927** (−196). About 227 builder lines removed for about 31 ScalaPB wiring/printing lines; 113 actual `newBuilder` calls removed (plan estimated 108); one exhaustive `TypeRef.Ref` oneof match. | Adopted ScalaPB runtime/ProtoJSON printer. Subsequent tasks added code; current ten files are 1,981 lines plus project 16 = **1,997**. |
| 8, `Finite` | Existing Mirror derivation **38** lines; Magnolia join/split about **20** plus dependency and field-name error code; shapeless-3 K0 about **15** plus dependency and same error-code issue. `Finite.upTo` is one line with its bounded given; Iron adds refined field types, Model/lifter changes and dependency. | Kept Mirror and existing bound; R3 bars adding a Model-visible dependency here. |
| 13, Nexus/projection | Ordinary Scala and existing protobuf reflection; old Nexus 336 + Actions 72 + Prelude 15 = **423**, new Nexus **328** (−95). Golden helper grew from 580 to about 640 lines for exact substitutions, without a new library. | Reused standard Scala and existing reflection; no new dependency. |

`model/project.scala` is unchanged from the planning tree and has no new `using dep`; its only dependency directive is the existing munit `test.dep`. Imports under `model/temporal` are `umpire.*`, sibling Model packages and local names. See `fn113-16/project-dependencies.diff` and `temporal-imports.txt`.

### Completion-review follow-up

Task 3's R9 refusal fixtures add 24 lifter-test lines, so the current all-Model-test count is
**924**; the table above records task 16's original 900-line measurement. DSL and Temporal counts
remain 1,191 and 5,299. Its full 17-test lifter run, check-mode model gate and model lint passed,
with no change to production sources or 30 checked-in IR/Case/expected artifacts; evidence is
under `.flow/tmp/fn113-3-completion-fix/`. Task 16's Go, vet, Go lint and Case results remain applicable.

Task 14's R25 comparison now weighs Scalameta traversal against its 64 net compiler-reflection
lines. Direct integration removes no `quotes.reflect` symbol/owner/rename logic and adds at least
two dependency/import lines (64 versus at least 66, an explicit lower bound rather than a measured
replacement). The maintained compiler `TreeAccumulator` remains; no library is added. The exact
comparison and primary-source references are in task 14's handover.

## Shared-document facts for conductor

- **MILESTONES.md:** mark task 16 complete after host review, record all 16 package passes, whole-tools vet, Scala check-mode gate plus separate Go verification, model/fast/full lint and Cases/fixtures/canary. Keep backend comparison deferred. Parts A–D are already accurately summarized there, including task 17’s one-run timing (89.203→79.528s focused; no statistical speedup claim). Current counts above replace the still-pending closure entry.
- **Module map:** existing DSL/Lifter/Gate rows already describe declarations only, ScalaPB IR/ProtoJSON and generation. Checker stays private with 4,064 production lines and no typed fixture layer; reader owns public `Check`. The existing immutable-goldens paragraph already states the closed projection. No additional architecture row is required unless the conductor wants these final counts in prose.
- **R13:** no new recapture for Part D or parameter renames. Independent frozen inputs remain strict for tables, IDs, refinement rows, fingerprints, answers, and Query Case bytes. The closed configuration admits positions by file, exact 11 type and 22 function substitutions, one Nexus source-path rename, parameter alpha-normalization, the resulting mapped-original type sort, and exploration Case IDs only. Current type/enum/record ordering remains strict; task 13’s negative regression tests and model/lower full packages passed.
- **Manifest/source delta:** exact added/deleted paths relative to planning commit `1f16ef5272` are in `fn113-16/manifest-deltas.txt`. Grouped: removed five DSL evaluator/sets files, Prelude, three Scala test files, two Nexus kernel files, nine checker support files, and the stale `model/specimens` prose; added `model/temporal/nexuscaller/Nexus.scala`, six compiler/refusal fixture files, four lower golden records for the two new verify Queries across original/mapped snapshots, three reader pin test files, and one checker key-helper test file. Task 8’s original/mapped Nexus golden additions are historical captured evidence; preserve their original fingerprint/index rather than recapturing or rewriting that baseline now. `model/cases` and all task-13 snapshotted generated artifacts remain byte-equal to their recorded post-regeneration hashes.
- **Spec corrections:** the audit found **35**, not the older 33/32, munit tests (two multiline `test(` calls were missed); it mapped all 35. The line baselines differ by date as above. R26’s 47 `_$1` estimate became **49 `_$N` name/var lines plus 24 `x$1` lines**; task 14 renamed both compiler forms and proved the structural changes without changing Case bytes.

stage: impl-review - ran(model: gpt-6-sol high; verdict: SHIP; receipt: .flow/tmp/fn113-16-review/receipt.json; current source/evidence hashes verified)
stage: wave-dispatch - ran(model: gpt-6-sol high; sequential worker)
stage: plan-sync - skipped(policy: planSync.enabled=false)
stage: tracker-sync - skipped(policy: sync inactive)
## Evidence
- Commits:
- Tests: baseline: green via handoff (task13 lower/export JSON and task17 model JSON; current hash checks in .flow/tmp/fn113-16/current-hashes.json), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -count=1 -tags test_dep -p 1 -parallel 1 <.flow/tmp/fn113-16/go-packages-remaining.txt> (exit 0; 13 packages; .flow/tmp/fn113-16/go-remaining-result.json), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -count=1 -tags test_dep -p 1 -parallel 1 ./tools/umpire/model (reused; exit 0; .flow/tmp/fn113-17/go-model-full-result.json), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -count=1 -tags test_dep -p 1 -parallel 1 ./tools/umpire/lower (reused; exit 0; .flow/tmp/fn113-13/go-lower-full-result.json), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -json -count=1 -tags test_dep -p 1 -parallel 1 ./tools/umpire/export (reused; exit 0; .flow/tmp/fn113-13/go-export-full-result.json), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go vet -tags test_dep -p 1 ./tools/umpire/... (exit 0; .flow/tmp/fn113-16/go-vet-result.json), CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; separate Go coverage complete; .flow/tmp/fn113-16/check-model-result.json), mise exec -- make lint-model (exit 0; inherited Scalafix path exception; .flow/tmp/fn113-16/lint-model-result.json), mise exec -- make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast (reused; exit 0; zero issues; .flow/tmp/fn113-17/lint-code-fast-result.json), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main CC=/usr/bin/clang mise exec -- make lint-code (exit 0; zero issues; .flow/tmp/fn113-16/lint-code-result.json), CC=/usr/bin/clang GOMEMLIMIT=4500MiB make umpire-check-cases umpire-check-fixtures canary-check-case (exit 0; .flow/tmp/fn113-16/cases-fixtures-canary-result.json), git diff --check (exit 0; .flow/tmp/fn113-16/diff-check.log), make umpire-check-backends (deferred by owner; not run)
- PRs: