---
satisfies: [R16]
---
# fn-113-clean-up-the-scala-model-layer-around.9 Prove the compile-time guarantees with must-not-compile fixtures

## Description
Prove the compile-time guarantees with must-not-compile fixtures. Implements R16 after task 8 removed the runtime code.

**Size:** S
**Files:** model/lifter/testdata/crossed/Crossed.scala.fixture (and a new fixture directory if a case needs its own build), model/lifter/test/Fixtures.test.scala (the expected refusal positions)
**Touches:** [model/lifter/testdata/crossed/**, model/lifter/testdata/<new fixture dir>/**, model/lifter/test/Fixtures.test.scala, tools/umpire/model/diagnostics_test.go]

### Approach
- Four guarantees, each a fixture that must not compile, refused at its line by the lifter's build step: (1) a step function bound to an action with other inputs (the typed `~>` of `Machine.scala:23-50`); (2) a Query pairing a `Property[S]` with a `Scenario[S]` of an unrelated machine; (3) a missing evidence case where evidence is total (already `werror/Evidence.scala.fixture:29:16`; keep); (4) a state type with a non-finite field (`Finite.derived`'s compile-time error, `Domain.scala:67-79`).
- `crossed` already proves a monitor of another state type (35:14) and a delivery of another message type (45:28); extend it or add fixtures, and add the new positions to the exact set the suite asserts ("the build refuses crossed types, at their lines"). Keep every existing position unchanged (R8).
- Two unrelated machines sharing the same state type must also be covered: `Reads.identity[S]` permits this at compile time, while the lifter and Go admission reject a Query crossing them without refinement. Keep the different-state compiler fixture and prove the same-state fallback with the authored Scala Query position in Go's diagnostic. `tools/umpire/model/diagnostics_test.go` is permitted for this required fallback.
- If a guarantee cannot be kept without runtime code, list it with the Go check that now reports it and a fixture proving the Go report carries the Scala line (`tools/umpire/model` diagnostics, `diagnostics_test.go`).
- The lifter suite needs `model/gen/model-scala.jar` and `.classpath` from a gate run; run the suite after a check-mode gate.

### Investigation targets
**Required**:
- `model/lifter/testdata/crossed/Crossed.scala.fixture:28-48`, `model/lifter/testdata/crossed/project.scala.fixture`
- `model/lifter/testdata/werror/Evidence.scala.fixture:24-32`
- `model/lifter/test/Fixtures.test.scala:202-230`
- `model/umpire/Machine.scala:23-50`, `model/umpire/Claims.scala:47-110`, `model/umpire/Domain.scala:57-79`

### Quick commands
CC=/usr/bin/clang mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (packages the jars and runs the lifter suite); mise exec -- scala-cli test model/lifter; mise exec -- make lint-model

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] Fixtures that must not compile cover all four guarantees (step bound to an action with other inputs; Property paired with a Scenario of an unrelated machine; missing evidence case where evidence is total; state type with a non-finite field), each refused at its line, and the lifter suite asserts the exact positions.
- [ ] Every previously asserted refusal position is unchanged; any guarantee that could not be kept at compile time is listed with the Go check that reports it and the fixture proving the report carries the Scala line.
- [ ] The check-mode gate, the lifter suite and `lint-model` pass with no change under `model/ir` or `expected/`.


## Done summary
Added exact-position compile refusals for wrong action input, a Property/Scenario with different state types, and a non-finite state field. The existing evidence, crossed monitor and crossed delivery refusals remain at Evidence.scala:29:16, Crossed.scala:35:14 and Crossed.scala:45:28.

R16 same-state fallback: distinct machines over the same `State` compile through `Reads.identity[State]`, so this pairing cannot be a must-not-compile guarantee without changing the DSL. New `samestate/SameState.scala` compiles and its `wrongPair` root is refused by `model/lifter/Claims.scala` at authored SameState.scala:24. `TestValidateReportsUnrelatedSameStateMachinesAtQueryPosition` mutates an actually lifted nexus-close Query between two unrelated machines sharing `temporal.nexuscaller.closepolicy.CloseResetState`; `tools/umpire/model/validate.go` reports exactly `model/temporal/nexuscaller/closepolicy/Claims.scala:228: query ackByOriginal.ackedThenReset pairs a Property of ackByOriginal with a Scenario of rejectAfterClose`. This covers the Go report if invalid IR reaches it, with the lifter-recorded Scala Query position.

Changed paths: model/lifter/testdata/crossed/ActionInput.scala (new, refusal 15:28), model/lifter/testdata/crossed/QueryPair.scala (new, refusal 26:69), model/lifter/testdata/nonfinite/NonFinite.scala (new, refusal 5:47), model/lifter/testdata/nonfinite/project.scala (new), model/lifter/testdata/samestate/SameState.scala (new, Query line 24), model/lifter/testdata/samestate/project.scala (new), model/lifter/test/Fixtures.test.scala, tools/umpire/model/diagnostics_test.go. No DSL, lifter, or checker production code changed.

Baseline: green via task-8 handoff at HEAD 2093a63f2da7c6c66f3509414dd78476eb4d5c04 (.flow/tmp/fn113-8-review/check-model-after-protoc.log, lint-model.log, go-results.json). Its full Go model/lower/export suites and goldens remain applicable except for the new focused diagnostic test. Final results: `mise exec -- scala-cli compile model/lifter/testdata/samestate` pass; crossed and nonfinite compile probes fail at exactly their asserted positions; focused Go diagnostic test pass (.flow/tmp/fn113-9/go-samestate-focused-2.log); `mise exec -- scala-cli test model/lifter` pass (lifter-samestate-focused.log); `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` pass (check-model-review-fix.log); `mise exec -- make lint-model` pass (lint-model-review-fix.log); `CC=/usr/bin/clang GOMEMLIMIT=4500MiB GOLANGCI_LINT_FIX=false mise exec -- make lint-code-fast` pass (lint-code-fast-review-fix.log); `git diff --check` pass. model/ir, model/cases and model/lifter/testdata/lifts/expected have no tracked changes.

R25: no generic framework or lifter code was added (0 lines), so no library candidate was needed. Shared-document follow-up for the conductor: MILESTONES.md should mark task 9 done after review; .plans/UMPIRE_MODULES.md should list the nonfinite and samestate refusal fixtures and update its stale native-evaluator sentence. The initial gate receipt was declined because unrelated MILESTONES.md was dirty; the final gate itself passed. All unrelated rebase changes were untouched.

stage: impl-review - ran (model: gpt-6-sol)
stage: wave-dispatch - ran (model: gpt-6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)

Independent resumed review: SHIP; prior R16 finding fixed, no open findings (`.flow/tmp/fn113-9-review/receipt.json`). Uncommitted source identity is recorded by `source-hashes.json` and `snapshot.diff` beside the receipt; the standalone review explicitly reviewed those working files because the user owns commits.
## Evidence
- Commits:
- Tests: baseline: green via handoff (.flow/tmp/fn113-8-review/check-model-after-protoc.log; lint-model.log; go-results.json), mise exec -- scala-cli compile model/lifter/testdata/samestate, compile probes: crossed and nonfinite expected failures at exact positions, CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- go test -tags test_dep -p 1 -parallel 1 -run '^TestValidateReportsUnrelatedSameStateMachinesAtQueryPosition$' ./tools/umpire/model, mise exec -- scala-cli test model/lifter, CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks, mise exec -- make lint-model, CC=/usr/bin/clang GOMEMLIMIT=4500MiB GOLANGCI_LINT_FIX=false mise exec -- make lint-code-fast, git diff --check, flowctl codex impl-review --base HEAD --spec codex:gpt-6-sol:high (uncommitted task-9 snapshot, resumed same receipt) -> SHIP (.flow/tmp/fn113-9-review/receipt.json; source-hashes.json)
- PRs: