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
- Run once, serially, after every other task of the spec is done: `CC=/usr/bin/clang mise exec -- make umpire-check-model` (full: gate tests, vocabulary, generation, DSL alone, Models and their tests, lifter tests, lifts compared to `model/ir`, Cases, go vet and go test over `./tools/umpire/...`); `mise exec -- make lint-model`; `GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast` (R10) and plain `GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code`; `CC=/usr/bin/clang mise exec -- go test -count=1 -json -tags test_dep ./tools/umpire/... > .flow/tmp/fn113-16/full-go.jsonl` (goldens inside; keep exit status and wall time, per MILESTONES.md); `make umpire-check-cases umpire-check-fixtures canary-check-case`; `git diff --check`; `git status --short model/ir model/cases model/lifter/testdata` equals the regenerated state. Not run: `make umpire-check-backends` (deferred by the owner). `./common/testing/testpilot/...`, `./tools/canary/...` and the functional compile only if a file they import changed (none should have).
- R3: `model/project.scala` has no new `using dep`; files under `model/temporal` import only `umpire.*`, sibling Models and the standard library (grep imports).
- R24: count and record before and after. Planning-time figures (2026-10-02): `model/umpire` non-test 2,415, `model/temporal` non-test 5,496, Scala tests 961 (the spec's 2,873 / 5,062 / 982 predate fn-115.7; state both). Target: `model/umpire` non-test at most 1,300; if above, name what stayed and why. The lifter's count from task 3 (R11).
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
- [ ] The full model gate, `lint-model`, `lint-code-fast`, plain `lint-code`, the complete `./tools/umpire/...` Go suite with goldens, `umpire-check-cases`, `umpire-check-fixtures` and `canary-check-case` pass once, serially, with logs and exit statuses under `.flow/tmp/fn113-16/`; `make umpire-check-backends` is recorded as deferred.
- [ ] No library a file under `model/temporal` can import was added (`model/project.scala` unchanged in dependencies).
- [ ] The summary states `model/umpire` (non-test, at most 1,300 or what stayed and why), `model/temporal` and Scala test line counts before and after against both the spec's and the planning-time baselines, the lifter's R11 figures, and the R25 roll-up.
- [ ] The handover gives the conductor the MILESTONES facts, the module map rows, the manifest deltas and the spec facts the tree contradicted.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
