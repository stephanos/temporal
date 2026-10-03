---
satisfies: [R14]
---
# fn-113-clean-up-the-scala-model-layer-around.10 Port the checker's search, lowering and claim tests off the typed fixture layer

## Description
Port the checker's search, lowering and claim tests off the typed fixture layer. Advances the spec's "Go checker's typed fixture layer" constraint (recorded under R14: the checker's tests then run over tables built from keys or IR); task 12 deletes the branches.

**Size:** M
**Files:** tools/umpire/model/internal/checker/umpire_test.go, keyclaims_test.go, keylower_test.go, keyunknown_test.go
**Touches:** [tools/umpire/model/internal/checker/umpire_test.go, tools/umpire/model/internal/checker/keyclaims_test.go, tools/umpire/model/internal/checker/keylower_test.go, tools/umpire/model/internal/checker/keyunknown_test.go]

### Approach
- Replace every use of the typed support layer (`NewMachine`, `Step0`/`Step1`, the typed Property and Scenario builders, reflection domains and keys from `*_support_test.go`; 47 call sites across these four files at planning) with the key-level constructors the module map names as private-checker operations (`NewTable`, `TableSpec`, `KeyProperty`, `KeyScenario`, `KeyFind`) or with IR fixtures the reader's tests already load. Preserve every assertion and comment; a test that asserts typed-only behavior (for example the typed `alterer` or a typed `holds`) becomes the key-level assertion of the same claim, or is listed for task 12 to decide.
- Do not delete the support files or any production branch here (task 12 does, once task 11 is also done). `require` over `assert`; `Equal`-style whole-value comparisons.
- Count the tests before and after; none disappears without a note.

### Investigation targets
**Required**:
- `tools/umpire/model/internal/checker/umpire_test.go`, `keyclaims_test.go`, `keylower_test.go`, `keyunknown_test.go`
- `tools/umpire/model/internal/checker/machine_support_test.go`, `domain_support_test.go`, `action_support_test.go`, `claims_support_test.go`, `keyclaims_support_test.go` (what each builder produces)
- `tools/umpire/model/internal/checker/table.go:60-80` (`Table` fields; `NewTable`, `TableSpec`)
- `tools/umpire/model/internal/checker/keyclaims.go:75-121` (`keyLevel`, `KeyProperty`, `observeKeys`)
- `tools/umpire/model/internal/checker/lower.go:94-200` (`asking.accepts`, `alterer`, `keyAlterer`)
- `.flow/tmp/fn115-7-summary.md` (decision 1: the branch list)

### Quick commands
CC=/usr/bin/clang mise exec -- go test -count=1 -tags test_dep ./tools/umpire/model/internal/checker/; CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/model/...; GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] `umpire_test.go`, `keyclaims_test.go`, `keylower_test.go` and `keyunknown_test.go` reference no typed support builder; every assertion and comment is preserved or its key-level equivalent is noted; the test count before and after is in the summary.
- [ ] The checker and reader suites pass; `lint-code-fast` passes; no production file and no support file changed.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
