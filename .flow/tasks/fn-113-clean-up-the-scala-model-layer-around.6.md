---
satisfies: [R14]
---
# fn-113-clean-up-the-scala-model-layer-around.6 Audit: map every munit test to a Go test over the IR

## Description
Audit: map every munit test to a Go test over the IR. Implements R14; the committed audit is what allows tasks 7 and 8 to delete tests.

**Size:** M
**Files:** .plans/umpire-scala-evaluator-audit.md (new); new or extended Go tests under tools/umpire/model (fixtures from `model/ir/*.json` and `model/lifter/testdata/lifts/expected/*.json`)
**Touches:** [.plans/umpire-scala-evaluator-audit.md, tools/umpire/model/*_test.go (new or extended tests only), tools/umpire/model/testdata/** (new fixture inputs only if a claim needs one)]

### Approach
- The suites, at planning time: `model/umpire/test/Declarations.test.scala` (16 tests, lines 175-415), `model/temporal/test/NexusCallerPins.test.scala` (9, lines 24-300), `model/temporal/test/StandaloneActivityPins.test.scala` (6, lines 26-164), `model/temporal/test/NexusKernel.test.scala` (2). That is 33 tests; the spec says 32. Audit all 33 and note the count.
- For each test: suite, line, the claim in one sentence, and one outcome: A, an existing Go test over the IR that asserts the same claim (name and package); B, a new Go assertion this task adds (fixtures built from IR through `Load`/`Check`/`Build`, never a typed constructor; `require`); C, a recorded reason the claim no longer applies (for example "the native table refuses a channel, a hole or a monitor, which only the IR interpreter reads"; "coverage targets have no IR form, task 7 records the authored facts"; "the kernel dispatch existed for the proof lemmas, task 13 removes it"; "a native step-function test stays, since step functions remain executable Scala").
- Leads: channels, `interpret_test.go` (`TestChannelCatalogs`, `TestChannelSendAndDelivery`, `TestUnorderedSendKeepsCatalogOrder`, `TestASendToAFullChannelLeavesTheDomain`); stutters and visible projections, `checking_test.go:527 TestRefinementControls`, `interpret_test.go:278`, `internal/checker/refinement_test.go`; "names what a refined machine sees must refine one", `admission_test.go:500`; replacement, `checking_test.go:468`; holes, `interpret_test.go:210`, `bound_test.go:171`; monitors, `checking_test.go:230-242`, `interpret_test.go:300`; table sizes, starts, reachable, stuck, `parity_test.go:24` and the semantics snapshots under `testdata/migration/semantics`; Query answers and explored counts, the golden Query answers; the canary silent-step rule and set checks (`check(...)`), look for a Go admission counterpart in `tools/umpire/lower` and `tools/canary/assessment`, else outcome C with the facts task 7 records.
- The audit file is outside `model/`, but keep it free of the retired front end's vocabulary; name the Scala line of each test so tasks 7 and 8 delete by reference. No Scala file changes in this task.

### Investigation targets
**Required**:
- `model/umpire/test/Declarations.test.scala:1-60,175-415`
- `model/temporal/test/NexusCallerPins.test.scala:24-300`
- `model/temporal/test/StandaloneActivityPins.test.scala:26-164`
- `model/temporal/test/NexusKernel.test.scala:34-54`
- `tools/umpire/model/interpret_test.go:109-300`, `tools/umpire/model/checking_test.go:230-600`, `tools/umpire/model/admission_test.go:479-520`, `tools/umpire/model/parity_test.go:24`, `tools/umpire/model/fixtures_test.go` (how tests load IR fixtures)
- `.plans/umpire-migration-claims.json` (the precedent for a claim inventory)

### Quick commands
CC=/usr/bin/clang mise exec -- go test -tags test_dep -run '<new tests>' ./tools/umpire/model; CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/model/...; GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] `.plans/umpire-scala-evaluator-audit.md` lists all 33 munit tests with suite, line, claim and one of the three outcomes; every A names an existing Go test over the IR that asserts the same claim, every B names the Go test this task added, every C states why the claim no longer applies.
- [ ] The added Go tests build their fixtures from IR, pass, and `lint-code-fast` passes; no Scala file changed.
- [ ] The summary notes the 33 versus 32 count for the conductor.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
