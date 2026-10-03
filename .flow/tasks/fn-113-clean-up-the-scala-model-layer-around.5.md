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
# fn-113.5 handover: the projection the fn-115 golden comparison needs

### Conductor decisions applied (second pass)

1. **Exploration Case IDs projected.** `config.json` `projection.projected_case_ids: ["exploration"]`.
   Both `Configuration` and `Projection.Case` reject any unknown kind. `golden.Projection.Case(kind,
   bytes, id)` replaces the varying part of an exploration Case's IDs (the candidate digest, which
   appears in caseId, programId and contractId) with a fixed token. Every other byte stays strict, and
   Query Cases are returned unchanged, so they stay fully strict.
   `TestMigrationProjectionKeepsLoweredCases` (lower) replaces `TestMigrationProjectionKeepsQueryCases`.
   It lowers the projected-changed nexus-caller IR (lines +3, `_$1` renamed, a kernel Function renamed
   under a listed substitution) and compares it against the **mapped goldens** in testdata. Every Query
   Case's case/program/contract/identity must be byte-equal. Every exploration Case's
   case/program/contract, candidates and reductions, must be equal except its ID; the identity.json of
   an exploration Case is a digest over its ID-bearing bytes and is not compared. Every golden Case of
   that Model must be lowered. Negative tests: an exploration Case with a changed non-ID byte
   (`"major":1` to `2`) fails, a Query Case with a changed ID fails, and a changed step fails Match.
   The golden unit test `TestCaseProjectionDropsOnlyAnExplorationID` covers the same cases plus an ID
   the Case does not carry and unknown kinds.
   **R13 wording amended (agreed by the conductor under the owner's delegation):** "lowered Case
   bytes" now reads "Query Case bytes, and exploration Cases except their IDs".
   **For task 13 (no change now):** moving `kernel/Nexus.scala` needs a `source_path_renames` entry from
   `model/temporal/nexuscaller/kernel/Nexus.scala` to its new path. Query Case `sources[].path` carries
   file paths, and the projection does not cover that.
2. **Fixture formatting (acceptance item 1): done.** `model/.scalafmt.conf` no longer excludes any
   fixture. The comment that explained the exclusion went with it. None of the five is a refusal
   fixture: the `var`/`while`, werror and crossed fixtures live in `model/lifter/testdata/{unsupported,
   werror,crossed}` and were not touched. Rejects' changes only rewrap two `(using ...)` calls, so what
   it tests is unchanged and no file stays excluded. I formatted with `mise exec -- scala-cli fmt
   --scalafmt-conf model/.scalafmt.conf <the five files>`. The result was identical to the earlier
   scratch measurement: whitespace, sorted import selectors, docstring asterisks and trailing commas.
   Diff at `.flow/tmp/fn113-5/fixtures-formatted.diff`. I then ran `make umpire-gen-model
   MODEL_GATE_ARGS=--skip-go-checks`; no touch of the generated Go was needed.
   - `model/ir/*.json`: unchanged (`diff -r` against the pre-run copy in `.flow/tmp/fn113-5/before/`).
     `model/cases`: unchanged.
   - `expected/{admission,channels,closereset,realizations}.json` differ only in `line`: equal with
     `line` removed. Lines changed: 355/558, 41/131, 526/528 and 608/1274. `declarations.json` and
     `presence.json` are byte-identical (`.flow/tmp/fn113-5/expected-projection-check.log`).
     `TestMigrationGoldens` (`Match`, protojson plus `proto.Equal` under positions-by-file) passes on
     them. So does `TestMigrationProjectionPreservesSemantics`, which now evaluates these line-shifted
     current fixtures and finds every reader snapshot equal to the frozen one under the projection.
   - `expected/rejects.txt`: only three Rejects line numbers moved (194 to 196, 227 to 230, 211 to 214).
   - No lifter test asserts a refusal position outside `rejects.txt`. `scala-cli test model/lifter`
     passes.


### What changed (Touches only; nothing staged or committed)

- `tools/umpire/internal/golden/golden.go` (+220 lines): `Config.Projection` (closed data, decoded with
  `DisallowUnknownFields`). `Match` still accepts the exact original, then the exact
  `Rename(Migrate(original))`, and only after that compares `project(mapped, original=true)` with
  `project(current)`. `Migrate` and `Rename` are unchanged, so the captured `mapped` variant and every
  derived golden are still produced from frozen inputs exactly as before. New: `FunctionsRenamed`
  (every listed function substitution must name a Function of some frozen input), `messages` (the
  generic walker; `positions` now uses it), `parameters`/`alpha`/`bound` (alpha-normalization).
- `tools/umpire/internal/golden/config.json`: `"projection": {"positions_by_file": true,
  "alpha_normalized_parameters": true, "function_name_substitutions": []}`. Tasks 13 and 14 add their
  exact renames here. The list is empty today because nothing has been renamed yet.
- `tools/umpire/internal/golden/golden_test.go`: three tests over the `JobModel` fixture (see "Negative
  tests").
- `tools/umpire/model/migration_golden_test.go`: `TestMigrationGoldens` also calls
  `cfg.FunctionsRenamed`. `TestMigrationProjectionPreservesSemantics` now also reads the **current**
  IR and requires its semantics, declarations and refined-Property snapshots to equal the frozen
  original's, with located strings projected (`locationProjection`: every frozen, mapped or renamed
  path goes to its current spelling, and the `:line[:col]` after it is dropped). Before this change,
  "the current IR reads as the original does" was only inferred from `Match`; now it is checked for
  the reader. New: `TestMigrationGoldensAdmitOnlyTheProjection`.
- `tools/umpire/lower/migration_golden_test.go`: new `TestMigrationProjectionKeepsQueryCases`.
  `TestMigrationGoldens` in lower is unchanged. It picks up the projection through `cfg.Match`, and a
  projected current selects the `mapped` variant.
- R25: no library was weighed. The projection uses protoreflect, which golden.go already used, plus the
  standard library: 220 added lines in golden.go. The import rule still holds (IR, protobuf and the
  standard library; testify only in tests). `TestLiveModelDependencyGraph` passes.

### The projections

1. **Positions by file.** `Position.line` is cleared on both sides, and `file` stays strict. A position
   in an unlisted file fails in `Migrate` (frozen side) or in `proto.Equal` (current side).
2. **Function-name substitutions.** These are exact `{old,new}` pairs applied to the mapped original
   only. Each pair renames the Function and every non-map string field equal to the old name (refs
   seen in the IR: `Call.function`, `StepBinding.function`, `Property.holds`, `Machine.evidence`,
   refinement `map`, and others). It is generic, so no reference field can be missed. A pair that
   collides with an existing name fails, and so does a pair whose rename is not made in the current IR
   (when anything else differs). `FunctionsRenamed` keeps the list closed.
3. **Alpha-normalized parameters.** The parameters of every `Function` and every `Lambda` become
   `#<depth>.<index>`, and `var`s referring to them follow. `let` and pattern binders hide a parameter
   in their scope, free variables keep their names, and the generated names cannot be written in
   Scala. **Function params are included, not only lambdas.** Measured: `_$1` (and `_$2` in
   declarations.json) is a `Function.params` name in 11 lifted Functions (properties/monitors lifted
   from placeholder lambdas, e.g. `nexusProtocol.property.completionSucceeds`) as well as in Lambda
   params, so R26 changes both. Calls pass arguments by position, so a parameter's name is never read.

**Parameter-name substitution for snapshot strings: measured unnecessary, not added.** I decompressed
all 48 reader snapshots and 1,363 artifact snapshots (`.flow/tmp/fn113-5/unz/`). Outside `inputs/`,
the reader snapshots contain no `_$1` and no `temporal.nexuscaller.kernel` names. Positions occur only
in semantics `Position` (309) and `Rejected` (3), which `locationProjection` covers. In the lower
artifacts, `_$1` and kernel names occur only in exploration `model.json` and `proposal.json`, and
`.scala:N` occurs in `lowering.json`, `error.json`, `plan.json`, `candidate.json`, `trace.html` and
`manifest.json`. The goldens derive all of these from frozen inputs.

### Findings for the owner (important)

- **Exploration Case bytes are NOT independent of positions or names.** `explore.Plan.lower` sets each
  candidate's Case ID to the sha256 of the whole candidate Model, positions, Function names and
  parameter names included. I measured it with a probe test, since removed to
  `.flow/tmp/trash/fn113-5/`; log at `.flow/tmp/fn113-5/lower-exploration-measure.log`. Shifting lines
  by 3, renaming `_$1`, or renaming one kernel Function changes **3/3** exploration Cases each for
  `nexusControl` (nexus-control) and `nexusDeadlines` (nexus-caller). realizations.json has
  `nexusDeadlines` too. The spec's statement that no "lowered Case byte depends on" these is wrong for
  exploration Cases. Query Cases are invariant, and `TestMigrationProjectionKeepsQueryCases` proves it.
  The conformance `assessment.json` binding identity (`goir.model/v1`) also hashes Function and
  parameter names, though not positions.
- **Query Case bytes already carry source paths.** Case `sources[].path` is a file path. The fn-115
  fixture renames (`.scala.fixture` to `.scala`) already make the current tree's realizations.json
  query Cases differ from the goldens' bytes. The goldens accept this by design: the read-only check
  matches the current IR structurally and derives every artifact from frozen inputs. Task 13's move of
  `kernel/Nexus.scala` is a path rename. It needs a `source_path_renames` entry
  (`model/temporal/nexuscaller/kernel/Nexus.scala` to its new path), which already exists as a
  mechanism. It is not a projection.
- (Superseded by conductor decision 2 above; kept for the record.) **Acceptance item 1, first pass:** `model/.scalafmt.conf` still excludes Admission, Channels, CloseReset, Realizations and
  Rejects. I measured with scratch copies and a conf without the exclusions. scalafmt would change
  175, 4, 203, 111 and 7 diff lines in those files, starting at lines 8, 65, 8, 14 and 179. The diff is
  at `.flow/tmp/fn113-5/scalafmt-fixtures.diff`. Doing it needs edits to `model/.scalafmt.conf` and the
  fixtures, plus a gate run (`--update`) to regenerate expected IR and `rejects.txt`. I did not edit or
  run the gate. With this projection, the resulting line shifts in expected IR are admitted, provided
  scalafmt's rewrites (`convertToNewSyntax`, `RedundantBraces`) do not change lifted structure. Run
  the goldens after the gate to confirm. `rejects.txt` is not a golden input.

### Paragraph for the owner (R13 amendment)

R13's baseline (the fn-115 goldens) is unchanged and is still derived from the frozen inputs. Only how
the current IR is matched to those inputs changes. Three closed projections are declared in
`tools/umpire/internal/golden/config.json`. (1) Positions are compared by file only, with line numbers
dropped, in the IR match and in the reader's located strings. (2) Function renames are listed as exact
old-to-new pairs and applied to every reference. (3) Function and lambda parameters are compared by
position, not by name (alpha-normalization). Everything else stays strict. The reader now also checks
that the current IR's tables, Definition IDs, refinement rows, fingerprints and Query answers equal the
frozen ones under this projection. Query Case bytes are shown invariant under it. Exploration Case IDs
are not invariant: they hash the whole candidate Model. The spec's "no lowered Case byte depends on
positions or names" must therefore read "no Query Case byte". Exploration Cases of
nexus-caller/nexus-control/realizations get new IDs when tasks 7, 13 and 14 land. The goldens still
pass because they evaluate frozen inputs.

**Alternative: re-capture both goldens per IR-changing task.** Each of tasks 7, 13 and 14 would run
two independent captures per package (`TestCaptureMigrationGoldens` in tools/umpire/model and
tools/umpire/lower, into new absolute directories). It would compare them byte-for-byte, review the
diff against the old baseline, and install the new one. Measured cost of one read-only verification:
about 2 min (reader, 108 s test time) and about 2 min (lowerer, 120 s). Capture costs about the same.
So that is roughly 8 to 10 min of machine time per task for two captures each, three times over. The
reviewer also has to judge a diff across 48 + 1,363 gzip entries each time (positions shift in
hundreds of receipts and every exploration artifact). Its weakness is the reason not to choose it:
every re-capture replaces the evidence that meaning did not change with a reviewer's reading of a
large diff, and R13's "unchanged" stops being machine-checked across tasks.

### Negative tests

- golden unit (`TestProjectionAdmitsOnlyLiftedTextChanges`, JobModel). Admitted: the listed function
  rename, lines shifted +7, parameters renamed (a Function and the machine `ends` Lambda), and all of
  these together. Rejected: a changed table row (a step's result state), an unlisted function rename,
  a listed rename not made (with lines shifted), a function renamed to a listed old name, a position in
  an unlisted file, a renamed parameter whose body still uses the old name, and a renamed parameter
  with a changed body (OR to AND).
- `TestProjectionIsClosed`: an unused function substitution fails `FunctionsRenamed`; a colliding
  substitution fails ("two Functions"); with no projection declared, line and parameter changes fail.
- `TestAlphaNormalizationRespectsShadowing`: `let` and pattern shadowing, nested lambdas and free
  variables.
- model `TestMigrationGoldensAdmitOnlyTheProjection` (real frozen and current nexus-caller). Admitted:
  lines +3, `_$1` renamed to `placeholder`, and a kernel Function renamed under a listed substitution.
  It also checks that the admitted IR's projected reader snapshots equal the frozen original's.
  Rejected: a changed table row (If branches swapped in a step function; Match fails and the projected
  snapshots differ), an unlisted function rename, a listed rename not made, a position in an unlisted
  file, and a renamed parameter with a changed body.
- lower `TestMigrationProjectionKeepsQueryCases`: the same admitted change. Match selects `mapped`, and
  every nexus-caller Query lowers to byte-identical case/program/contract/identity. A changed step body
  fails Match.
- `TestMigrationGoldenDetectsSemanticMutations` is kept unchanged and passes.

### Checks (logs under .flow/tmp/fn113-5/)

Final runs after both conductor decisions:
- `mise exec -- scala-cli test model/lifter`: passed (lifter-test.log).
- `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks`: ok (gen-model.log).
  `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`: ok (check-model.log).
- `make lint-model`: rc 0, "All files are formatted with scalafmt" (lint-model.log). It prints scalafix
  `NoSuchFieldException: path` traces, but the exit code is 0.
- `go test -run 'Migration|Golden' ./tools/umpire/model`: ok, 247 s (model-migration.log).
- `go test -run 'Migration|Golden' ./tools/umpire/lower`: ok, 132 s (lower-migration.log).
- `go test ./tools/umpire/internal/golden/...`: ok, 9 tests (golden-unit.log). Ownership/import
  tests: ok (model-ownership.log).
- `make lint-code` on the three packages: 0 issues (lint.log).

First pass:

All were run through `mise exec --` with `GOFLAGS=-p=1 -tags test_dep -count=1 -timeout 30m`.
- Baseline before the edits: `go test -run 'Migration|Golden' ./tools/umpire/model` passed (baseline-model.log).
- `go test ./tools/umpire/internal/golden/...` passed: 8 tests (golden-unit.log).
- `go test -run 'Migration|Golden' ./tools/umpire/model` passed. `TestMigrationGoldens` took 25.8 s and
  `TestMigrationProjectionPreservesSemantics` took 80.7 s; package total 108 s (model-migration.log).
- `go test -run 'Migration|Golden' ./tools/umpire/lower` passed. `TestMigrationGoldens` took 112 s;
  package total 120 s (lower-migration.log).
- `go test -run 'Ownership|Dependency|Import|Caller' ./tools/umpire/model` passed (model-ownership.log).
- Lint: `make lint-code-fast` is unusable on this branch (its base rev is far behind, about 718
  pre-existing findings), so I ran `GOLANGCI_LINT_FIX=false make lint-code
  LINT_CODE_TARGETS="./tools/umpire/internal/golden/... ./tools/umpire/model ./tools/umpire/lower"`.
  Result: 0 issues (lint.log). One revive finding (`confusing-results`) was fixed first.
- Not run: the model gate, `make umpire-check-backends`, and the whole tools/umpire tree (per the
  constraints).

### Shared-doc changes for the conductor

- `.plans/UMPIRE_MODULES.md` "Immutable migration goldens": add that the read-only check matches the
  current IR "strictly, modulo the closed projection in config.json (positions by file, exact function
  renames, alpha-normalized Function/Lambda parameters)". Add that the reader also compares the
  current IR's projected snapshots with the frozen ones.
- The spec's API contract and its "R13 allowed list" decision: "lowered Case bytes" should read
  "lowered Query Case bytes". Exploration Case IDs hash the whole candidate Model (see Findings).
  Also, alpha-normalization covers Function parameters as well as Lambda parameters.
- MILESTONES.md: tasks 7, 13 and 14 are unblocked. Task 13 adds its function pairs to
  `function_name_substitutions` and a `source_path_renames` entry for the moved `kernel/Nexus.scala`.
  Task 14 needs no config entry.
- Record the R13 amendment ("Query Case bytes, and exploration Cases except their IDs") in the spec's
  API contract and decisions section.
- `model/README.md` or MILESTONES.md, if either mentions the fixtures' scalafmt exclusion: it is gone.

### Review (independent review: SHIP with fixes; all applied)

1. `TestAlphaNormalizationRespectsShadowing` has three new checks:
   - A two-parameter lambda `(x, y) => x - y` differs from `(y, x) => x - y`, and equals
     `(a, b) => a - b`.
   - `x => x => x` equals `a => b => b`, which pins same-name shadowing.
   - `x => x => x` differs from `a => b => a`.
2. The model and lower admission tests no longer append the kernel pair unconditionally. If
   `config.json` already lists `kernel.Protocol$.completeStep`, they use its listed new name, so task
   13 can list the pair without breaking these tests.
   - The model test's "unlisted function rename" and "listed rename unmade" now rewrite the moved name
     (`moved` to `moved+"Unlisted"`, and `moved` back to `kernel`). Both stay meaningful after task 13.
3. `TestMigrationProjectionKeepsLoweredCases` now lowers the **real current IR of every Model** in the
   inventory (12 subtests, including nexus-control, nexus-close and realizations). It compares every
   Query and exploration Case, candidates and reductions, with the mapped goldens. The synthetic
   nexus-caller edit stays as an extra case.
   - **Finding:** realizations.json's Query Cases name `Realizations.scala` where the goldens have
     `Realizations.scala.fixture`. That rename is fn-115's existing `source_path_renames`.
   - So the comparison applies the listed renames to the golden Case, Program and Contract through a
     new `golden.Config.RenameSources`. It replaces only a JSON string exactly equal to a renamed path,
     and its unit test is `TestRenameSourcesRenamesOnlyWholePaths`.
   - For such a Case, the golden identity is derived again from the renamed golden bytes
     (`recordedrun.CaseIdentity` and `runtime.CaseFingerprint`) and compared strictly.
   - **Query Case bytes are therefore strict modulo the already-listed source path renames.** This is
     not a new projection, but it is worth stating to the owner next to the R13 amendment.
4. For `ExplorationCase`, `Projection.Case` now requires the ID to match `^[0-9a-f]{64}$` and to occur
   in the bytes. New negatives cover a short ID and an uppercase ID.
5. Negative subtests that failed for unrelated reasons:
   - Golden: renamed to "renamed parameter with a changed operator".
   - Model: replaced by "parameters swapped over an unchanged body". It picks a Function whose body
     reads both of its first two parameters and swaps their names, so the subtest fails only by
     binding. It does not depend on `_$1`, so it survives task 14.
6. For exploration Cases, `identity.json` must exist on both sides. Neither of its fields is compared,
   because both hash the whole Case, IDs included: `Canonical` is `recordedrun.CaseIdentity` over the Case bytes, and `Fingerprint` is `runtime.CaseFingerprint`, a sha256 of the deterministic proto encoding of the whole Case. No field is free of the ID.
   The function comment says so.
7. `golden.go` has a comment that the function substitution renames every equal string on purpose:
   the names are fully qualified, and no reference field can be missed.
8. **Lint:** `make lint-code-fast` was replaced by a scoped `make lint-code`. On this branch,
   lint-code-fast diffs against a base rev far behind and reports about 718 pre-existing findings in
   unrelated packages, so its result says nothing about this change. I ran `GOLANGCI_LINT_FIX=false
   mise exec -- make lint-code LINT_CODE_TARGETS="./tools/umpire/internal/golden/...
   ./tools/umpire/model ./tools/umpire/lower"`. It reported 0 issues after one testifylint fix
   (`require.JSONEq`).

Re-runs after the fixes (logs in `.flow/tmp/fn113-5/`):
- golden unit tests: ok, 10 tests (golden-unit.log).
- `go test -run 'Migration|Golden' ./tools/umpire/model`: ok, 233 s (model-migration.log).
- `go test -run 'Migration|Golden' ./tools/umpire/lower`: ok, 184 s. `TestMigrationProjectionKeepsLoweredCases`
  took 16.8 s (lower-migration.log).
- Ownership/import tests: ok (model-ownership.log).
- Scoped `make lint-code`: 0 issues (lint.log).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ff23e8ddb106f0acdf9899a029a6f47b39095f45
- Tests: GOFLAGS=-p=1 mise exec -- go test -tags test_dep -count=1 -timeout 30m ./tools/umpire/internal/golden/... -> ok (10 tests, after review fixes; .flow/tmp/fn113-5/golden-unit.log), GOFLAGS=-p=1 mise exec -- go test -tags test_dep -count=1 -timeout 30m -run 'Migration|Golden' ./tools/umpire/model -> ok 233s (after review fixes; .flow/tmp/fn113-5/model-migration.log), GOFLAGS=-p=1 mise exec -- go test -tags test_dep -count=1 -timeout 30m -run 'Migration|Golden' ./tools/umpire/lower -> ok 184s, incl. TestMigrationProjectionKeepsLoweredCases over all 12 current Models (after review fixes; .flow/tmp/fn113-5/lower-migration.log), GOFLAGS=-p=1 mise exec -- go test -tags test_dep -count=1 -timeout 30m -run 'Ownership|Dependency|Import|Caller' ./tools/umpire/model -> ok (.flow/tmp/fn113-5/model-ownership.log), GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS="./tools/umpire/internal/golden/... ./tools/umpire/model ./tools/umpire/lower" -> 0 issues (replaces lint-code-fast, which reports ~718 pre-existing findings on this branch; .flow/tmp/fn113-5/lint.log), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf <5 fixtures> -> formatted (.flow/tmp/fn113-5/fixtures-formatted.diff), mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks -> ok; model/ir and model/cases unchanged; expected/*.json differ only in line (.flow/tmp/fn113-5/gen-model.log, expected-projection-check.log), mise exec -- scala-cli test model/lifter -> passed (.flow/tmp/fn113-5/lifter-test.log), mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks -> ok (.flow/tmp/fn113-5/check-model.log), mise exec -- make lint-model -> rc 0, all files formatted (.flow/tmp/fn113-5/lint-model.log), exploration Case measurement probe (removed to .flow/tmp/trash/fn113-5/) -> exploration Case bytes differ 3/3 under each projected change before the Case-ID projection (.flow/tmp/fn113-5/lower-exploration-measure.log), independent review (claude-opus-5-5, fresh context): SHIP; three should-fix items and five nits applied by the implementer, all re-runs pass
- PRs: