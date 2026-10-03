---
satisfies: [R1, R2, R15, R17, R19]
---
# fn-113-clean-up-the-scala-model-layer-around.7 Remove the sets, Coverage and the rules with no Go consumer, and verify Part A

## Description
Remove the sets, Coverage and the rules with no Go consumer, and verify Part A. Implements R1 and R2 (verification), R17, the no-consumer clause of R15 and part of R19. First task that edits `model/umpire` and the Models.

**Size:** M
**Files:** model/umpire/Sets.scala and Coverage.scala (deleted), model/umpire/Domain.scala (`Keys.spelling`), model/temporal/nexuscaller/Claims.scala, model/temporal/standaloneactivity/Claims.scala, model/temporal/test/NexusCallerPins.test.scala, model/temporal/test/StandaloneActivityPins.test.scala, model/ir/*.json (regenerated), model/README.md (only if it names a set)
**Touches:** [model/umpire/Sets.scala, model/umpire/Coverage.scala, model/umpire/Domain.scala, model/temporal/nexuscaller/Claims.scala, model/temporal/standaloneactivity/Claims.scala, model/temporal/test/NexusCallerPins.test.scala, model/temporal/test/StandaloneActivityPins.test.scala, model/ir/**, model/README.md (set mentions only)]

### Approach
- Part A first (R1, R2): confirm `Canonical.scala`, `Lower.scala`, `Alterer`, `Table.alter` and `Machine.alterer` are absent (`grep -rn 'Alterer\|alterer\|Canonical' model/`), restate fn-115.7's borderline-kept list, and cite its passing check-mode gate with unchanged IR (`.flow/tmp/fn115-7-summary.md` section 1 and `.flow/tmp/fn115-7/check-model*.log`; `model/umpire` non-test is 2,415 lines). Rerun the check-mode gate only if that evidence is invalidated (MILESTONES.md: reuse a passing baseline). Record the outcome in the summary.
- Record before deleting (R17): each set with name, purpose, bindings, repeat, Queries, machine, coverage goals and budget: `nexusCallerTests`, `nexusCallerCanary`, `nexusCallerExploration` (`nexuscaller/Claims.scala:273-310`), `standaloneActivityTests`, `standaloneActivityCanary`, `standaloneActivityExploration` (`standaloneactivity/Claims.scala:242-260`).
- R17's stop clause: the set names appear in Go and testdata only as Case and pin labels (`tools/canary/assessment/testdata/nexusCallerCanary-*`, `tools/umpire/lower/{lower,activity,activity_cases}_test.go`, `tools/umpire/conformance/{nexus,activity}_test.go`, `tools/umpire/cmd/umpire-fuzz/run_test.go`). Confirm they are strings fixed in Go or in the Case manifest and that no consumer reads a set from Scala or the IR (the IR has no set message). If one does, stop and report for the owner's decision.
- Delete `UmpireSet`, `Purpose`, `Binding`, the set member of `Declaration`, `checkSet`, `CoverageGoal`, `CoverageTarget`, `Coverage.targets` and `Coverage.within`, and `Keys.spelling` (its only caller was Coverage); these three rules were the ones fn-115.12 marked for removal here (R19). `check`, `checkQuery` and `checkModel` stay for task 8.
- Tests, by the audit's reference: delete "the sets" (`NexusCallerPins.test.scala:212-255`) and the set arguments of the `check(...)` calls (`NexusCallerPins.test.scala:300-315`, `StandaloneActivityPins.test.scala:146-162`); the rest of those suites stays for task 8.
- Declarations follow the sets in both `Claims.scala` files, so their positions shift: regenerate with `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks`; the diff of `model/ir` must show position changes only. The goldens pass under task 5's projection; Case bytes, the functional fixtures and the canary Case do not change.
- The DSL must still compile alone (gate step "compile the framework alone"). R25: nothing generic is written here.

### Investigation targets
**Required**:
- `model/umpire/Sets.scala:1-128`, `model/umpire/Coverage.scala:1-85`, `model/umpire/Domain.scala:117-126`
- `model/temporal/nexuscaller/Claims.scala:273-310`, `model/temporal/standaloneactivity/Claims.scala:242-260`
- `model/temporal/test/NexusCallerPins.test.scala:212-255,300-315`, `model/temporal/test/StandaloneActivityPins.test.scala:146-162`
- `tools/umpire/lower/lower_test.go` and `tools/canary/assessment/testdata/` (set names as labels)
- `.flow/tmp/fn115-7-summary.md` (section 1), `.flow/tmp/fn115-12-summary.md` (section 6 table), `.plans/umpire-scala-evaluator-audit.md`

### Quick commands
mise exec -- scala-cli compile model/project.scala model/umpire; mise exec -- scala-cli test model/project.scala model/umpire model/temporal; CC=/usr/bin/clang mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks; git diff --stat model/ir (positions only); CC=/usr/bin/clang mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks; CC=/usr/bin/clang mise exec -- go test -tags test_dep -run '^TestMigrationGoldens$' ./tools/umpire/model ./tools/umpire/lower; make umpire-check-cases umpire-check-fixtures canary-check-case; mise exec -- make lint-model

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] Part A verified: the named files and declarations are absent, the borderline list is restated, and the summary cites the evidence (or a fresh check-mode gate) that Part A changed no lifted output.
- [ ] `UmpireSet`, `Purpose`, `Binding`, `Coverage`, `CoverageGoal`, `CoverageTarget` and `Keys.spelling` are gone and nothing under `model/` names them; the summary lists each of the six sets with purpose, bindings, repeat, Queries, machine, goals and budget.
- [ ] No Go consumer needed a set (checked and recorded); the regenerated IR differs in positions only; the goldens pass under the projection; `umpire-check-cases`, `umpire-check-fixtures` and `canary-check-case` report no change; the DSL compiles alone and `lint-model` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
