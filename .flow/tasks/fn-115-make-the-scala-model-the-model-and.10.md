---
satisfies: [R2, R11, R17, R20]
---
# fn-115-make-the-scala-model-the-model-and.10 Replace model shell orchestration with a Scala gate and lifter tests

## Description
Replace model shell orchestration with a Scala gate and lifter tests. Implements R2, R11, R17, R20 using the reviewed parent contracts.

**Size:** M
**Files:** model/gate Scala program/tests; model/lifter MUnit fixtures/refusals; Make target wiring
**Touches:** [model/gate/**, model/lifter/**, model/*.sh, model/**/project.scala, Makefile, .gitignore]

### Approach
- Move fixture, refusal and source-location checks into the lifter's own MUnit suite. Retain the current explicit root list and supported outputs; fn-114 owns later root declarations and single-run redesign.
- Implement one Scala gate for compile/test/package, proto generation, lifting, deterministic compare/update and Go checks, using established or standard-library process/file facilities. Preserve explicit verify versus update behavior and complete-tree publication.
- Centralize process handling, including scala-cli printing an error while returning zero. Test that case, nonzero exits, missing scala-cli/protoc/go and meaningful path diagnostics through a small process seam.
- Retain independent DSL compilation. Delete model shell scripts only after the new entrypoint runs their complete live obligations; update Make and ignore ownership in the same task.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `model/scalav2/run.sh:119`
- `model/scalav2/scala.sh`
- `model/scalav2/gen.sh`
- `model/scalav2/lifter/testdata`
- `model/scalav2/scala/project.scala`
- `Makefile:1182`

### Quick commands
The new Scala gate in check and --update modes; its focused process/lifter tests; make lint-model; CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/...

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.
## Acceptance
- [ ] No shell script remains inside model, and one Scala gate performs the full live generation/check pipeline.
- [ ] Lifter fixtures/refusals are actual tests; DSL-only compilation, source-location stability and deterministic goldens pass.
- [ ] Printed-error/zero-exit, nonzero process status and missing tools fail with precise diagnostics; ordinary check mode does not rewrite artifacts.
- [ ] Generation updates introduce no untracked/generated drift beyond the reviewed outputs.

## Done summary
Replaced `model/run.sh`, `gen.sh` and `scala.sh` with one Scala gate at `model/gate`: `Tools.scala` (the single process seam: printed-error/zero-exit, nonzero exit and missing-tool handling), `Roots.scala` (the explicit root lists) and `Gate.scala` (compile/test/package, DSL-only compile, IR class generation with content-hash staleness, lifting, check versus `--update` with complete-tree publication, Go checks). The lifter's fixture, refusal and source-position checks are 14 MUnit tests in `model/lifter/test`. `make umpire-check-model`, `umpire-gen-model` and the fmt/lint/fix targets run through the gate, and the gate's own 26 tests run first. No shell script remains inside `model/`.

The new gate and the old `run.sh` produced byte-identical lifts, refusals and IR jar on the same tree. Check mode fails without rewriting on a perturbed IR or expected file, on a real `-Werror` case where scala-cli exits 0, and with the tool named when `protoc` or `go` is missing. `--update` restores perturbed files to their original hashes with no other drift. Full `make umpire-check-model` passes in 406 s (old gate: 336-640 s); `lint-model` and `lint-code-fast` pass; IR, Cases and goldens are unchanged.

Independent review (Claude Fable, fresh context): NEEDS_WORK in round 1 (the update path was unproven), SHIP in round 2. Recorded deviation: the lifter's `project.scala` compiles `gate/Tools.scala` so its tests share the process seam, because scala-cli has no test-scoped file directive. Handover: .flow/tmp/fn115-10-summary.md; evidence: .flow/tmp/fn115-10-evidence.json; reviews: .flow/tmp/fn115-10-review/. No agent commits.
## Evidence
- Commits:
- Tests: mise exec -- scala-cli test model/gate, mise exec -- scala-cli test model/lifter, CC=/usr/bin/clang mise exec -- make umpire-check-model, CC=/usr/bin/clang mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (on perturbed artifacts), Independent review round 2 SHIP (claude-fable-5-1); .flow/tmp/fn115-10-review/round2-review.md
- PRs: