---
satisfies: [R14]
---
# fn-113-clean-up-the-scala-model-layer-around.11 Port the checker's composition, refinement, monitor and progress tests off the typed fixture layer

## Description
Port the checker's composition, refinement, monitor and progress tests off the typed fixture layer. Same constraint as task 10 (recorded under R14), for the other test group; task 12 deletes the branches.

**Size:** M
**Files:** tools/umpire/model/internal/checker/compose_test.go, composekeys_test.go, refinement_test.go, monitor_test.go, progress_test.go
**Touches:** [tools/umpire/model/internal/checker/compose_test.go, tools/umpire/model/internal/checker/composekeys_test.go, tools/umpire/model/internal/checker/refinement_test.go, tools/umpire/model/internal/checker/monitor_test.go, tools/umpire/model/internal/checker/progress_test.go]

### Approach
- As task 10, for these five files (27 call sites at planning): typed builders become `NewTable`/`TableSpec`/`KeyProperty`/`KeyScenario`/`KeyFind` or IR fixtures; typed refinements (`RefinementSpec` with typed states, `mapValue`/`stepOf`) become key-level refinements (`RefineTables`, `MapState`); typed compositions, monitors and progress claims likewise. Preserve every assertion and comment; typed-only claims become their key-level counterpart or are listed for task 12.
- No deletion of support files or production code here. Count tests before and after.

### Investigation targets
**Required**:
- `tools/umpire/model/internal/checker/compose_test.go`, `composekeys_test.go`, `refinement_test.go`, `monitor_test.go`, `progress_test.go`
- `tools/umpire/model/internal/checker/compose_support_test.go`, `refine_support_test.go`, `monitor_support_test.go`, `progress_support_test.go`, `machine_support_test.go`
- `tools/umpire/model/internal/checker/refine.go:1-60,289` (`Refinement`, `keyLevel`, `checkKeyRefined`, `stepOf`)
- `tools/umpire/model/internal/checker/search.go:339-350,453-470,505-515` (typed `observe` arms, `readState`, `readStep`, `productStep`)
- `tools/umpire/model/internal/checker/compose.go`, `composekeys.go`, `monitor.go`, `progress.go`

### Quick commands
CC=/usr/bin/clang mise exec -- go test -count=1 -tags test_dep ./tools/umpire/model/internal/checker/; CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/model/...; GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] `compose_test.go`, `composekeys_test.go`, `refinement_test.go`, `monitor_test.go` and `progress_test.go` reference no typed support builder; every assertion and comment is preserved or its key-level equivalent is noted; the test count before and after is in the summary.
- [ ] The checker and reader suites pass; `lint-code-fast` passes; no production file and no support file changed.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
