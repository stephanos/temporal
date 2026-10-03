---
satisfies: [R13]
---
# fn-113-clean-up-the-scala-model-layer-around.5 Teach the fn-115 golden comparison the projections fn-113's IR changes need

## Description
Teach the fn-115 golden comparison the projections fn-113's IR changes need. Implements the machinery R13 needs for Parts C and D and R26; Go test support only, no reader or lowering semantics change.

**Size:** M
**Files:** tools/umpire/internal/golden/golden.go, tools/umpire/internal/golden/config.json, its tests; tools/umpire/model/migration_golden_test.go; tools/umpire/lower/migration_golden_test.go
**Touches:** [tools/umpire/internal/golden/**, tools/umpire/model/migration_golden_test.go, tools/umpire/lower/migration_golden_test.go]

### Approach
- Measured at planning: `golden.Config.Match` (`golden.go:163`) requires the current IR to be `proto.Equal` to the frozen original or to its image under the closed path and label substitutions; the reader's semantics snapshots store receipt positions as `file:line[:col]` (for example `Claims.scala:192` in `testdata/migration/semantics/ir/nexus-caller.json.gz`); Behavior Fingerprints (`checker/canonical.go`) hash no positions, expressions or names; lowered Case bytes carry no positions, kernel names or `_$1`. Task 7 shifts lines (declarations follow the removed sets in both `Claims.scala`), task 13 renames functions out of `temporal.nexuscaller.kernel` and moves `kernel/Nexus.scala`, task 14 renames `_$1`. The spec's API contract allows exactly these text changes, and R13 wants the derived tables, IDs, rows, fingerprints, answers and Case bytes unchanged.
- Add a projection, declared as closed data in `config.json`: (1) positions compare by file only (line and column dropped) in the IR input match and in the semantics snapshot receipts (where `replace` already rewrites `Position` strings); (2) `function_name_substitutions`, exact-match like `source_path_substitutions`, applied to `functions[].name` and every reference; (3) lambda parameters alpha-normalized (binder and `var` references renamed consistently) in both models before `proto.Equal`, and a closed `parameter_name_substitutions` list for snapshot strings only if a receipt explanation prints a parameter name (measure first; say so either way).
- Everything else stays strict. Keep `TestMigrationGoldenDetectsSemanticMutations` and add negative cases: a changed table row, an unlisted function rename, a position in an unlisted file, a renamed parameter with a different body all still fail.
- The golden helper keeps its import rule (IR, protobuf, standard library; testify in its tests) enforced by `tools/umpire/model/ownership_test.go`.
- Owner decision: this amends how R13's baseline is compared, not the baseline itself. The summary lists the three projections in one paragraph for the conductor to put to the owner (R13's amendment clause) and gives the alternative, re-capturing both goldens per IR-changing task with two independent captures, with its cost, so the owner can choose. Tasks 7, 13 and 14 wait on this task.

### Investigation targets
**Required**:
- `tools/umpire/internal/golden/golden.go:25-45,62-135,163-175` (`Config`, `Inputs`, `Migrate`, `substitute`, `Match`)
- `tools/umpire/internal/golden/config.json`
- `tools/umpire/model/migration_golden_test.go:187-260,367-415` (`migrationInputs`, `TestMigrationGoldens`, the `replace` of positions)
- `tools/umpire/lower/migration_golden_test.go:192-260`
- `tools/umpire/model/checking.go:115,135,304-305` (receipt positions)
- `tools/umpire/model/internal/checker/canonical.go:19-25` (what fingerprints hash)
- `tools/umpire/model/ownership_test.go` (the golden helper's import rule)
- `.plans/UMPIRE_MODULES.md:203-263` (the goldens' contract: capture, two independent captures)

### Quick commands
CC=/usr/bin/clang mise exec -- go test -tags test_dep -run 'Migration|Golden' ./tools/umpire/internal/golden/... ./tools/umpire/model ./tools/umpire/lower; CC=/usr/bin/clang mise exec -- go test -tags test_dep -run 'Ownership|Dependency' ./tools/umpire/model; GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

### Follow-up from fn-115.14
- Once the positions-by-file projection exists, format the five lifter fixtures that fn-115.14 left out of scalafmt (`model/lifter/testdata/lifts/{Admission,Channels,CloseReset,Realizations,Rejects}.scala`), remove their exclusion from `model/.scalafmt.conf`, regenerate the expected fixture IR and `rejects.txt` positions through the gate, and show the goldens pass under the projection. The owner asked for scalafmt to apply to all fixtures.

## Acceptance
- [ ] All lifter fixtures are scalafmt-formatted; `model/.scalafmt.conf` excludes none of them.
- [ ] `TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower` pass unchanged on the current tree, and still fail on a changed table row, an unlisted function rename, a position in an unlisted file and a renamed parameter with a changed body (negative tests added).
- [ ] The projection is closed data in `config.json` (positions by file only; exact function-name substitutions; parameter alpha-normalization, plus snapshot substitutions only if measured necessary), and the helper's import rule still holds.
- [ ] The summary states the three projections for the owner's agreement under R13 and the re-capture alternative with its cost; `lint-code-fast` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
