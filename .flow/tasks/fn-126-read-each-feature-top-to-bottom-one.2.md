---
satisfies: [R1, R2, R3, R5, R9, R10, R13]
---
# fn-126-read-each-feature-top-to-bottom-one.2 Write the Nexus folders and shared Models as feature files, share the bounds, and retire the per-kind file names

## Description
Convert the remaining folders to feature files with today's declaration forms: `features/nexuscaller` (with the control Model), `nexuscaller/closepolicy`, `features/nexusoperation`, `shared/taskqueue` and `shared/worker` (R1-R3, R5). Add the shared bounds source (R13, bounds half). Make the layout test refuse the retired file names (R10). Bring the docs to the new layout (R9, layout half).

**Cross-spec entry gate:**
- Task 1 is done.
- Never alongside fn-124.8.
- Before fn-124.7.

**Size:** L
**Files:**
- `model/temporal/features/{nexuscaller,nexuscaller/closepolicy,nexusoperation}/**`, `model/temporal/shared/{taskqueue,worker}/**`;
- `model/temporal/shared/Bounds.scala` (new);
- `tools/umpire/model/layout_test.go` (or its fn-124.8 home);
- Go tests that assert Scala positions (`tools/umpire/lower/activity_test.go:742` and others found by grep);
- `tools/umpire/internal/golden/config.json`;
- `model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`, `.plans/UMPIRE4_VISION.md`, `AGENTS.md`.

**Touches:** [model/temporal/features/**, model/temporal/shared/**, tools/umpire/model/layout_test.go, tools/umpire/lower/**, tools/umpire/internal/golden/**, model/ir/**, model/cases/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, .plans/UMPIRE4_VISION.md, AGENTS.md]

### Approach
- Follow task 1's pattern folder by folder.
- `shared/taskqueue` keeps borrowing the record's pin (`…System$package$`) and family.
- The close policy gets one module object for `rejectAfterClose` and its nine derivations here. Task 5 splits them into `Derived` objects.
- Bounds: move every `Limits` that two folders declare (today `three` ×3, `five`, `twelve`) to `shared/Bounds.scala`. Names stay, and the IR changes only in positions.
- R10: extend the layout test with the five retired file names (one case each proves detection) and with live files that name such a path. Fix the Go tests that assert old Scala paths.
- Docs (R9, layout): describe the feature-file layout and R2's order, with the activity as the example. Remove every reference to the per-kind files and to "its own `Capabilities.scala`". Task 5 updates the declaration shape, and task 6 the names.

### Investigation targets
**Required:**
- the per-kind files of each folder above
- `tools/umpire/model/layout_test.go`
- `model/README.md` "Writing a Model" and "Where things are"
**Optional:**
- `.plans/UMPIRE_MODULES.md:30,321,484`

### Quick commands
```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks && make lint-model
go test -count=1 -tags test_dep -p 2 ./tools/umpire/...
grep -rn "Model.scala\|Properties.scala\|Queries.scala\|IrFiles.scala" model/README.md model/SEMANTICS.md .plans/UMPIRE_MODULES.md .plans/UMPIRE4_VISION.md AGENTS.md
```

### Execution constraints
- R5 holds as in task 1. The R4 lint passes on every converted folder.

## Acceptance
- [ ] Every Model folder holds one feature file in R2's order. 30 per-kind files have become 8 feature files.
- [ ] Shared bounds come from `model/temporal/shared/Bounds.scala`. No bound is declared twice, and Limits names are unchanged.
- [ ] The layout test fails on each of the five retired file names (one case each) and on a live file that names one.
- [ ] The R9 docs describe the feature-file layout and name no retired file.
- [ ] R5 holds, with the deltas recorded. All gates of the spec's Verification pass.


## Done summary
# fn-126.2 done summary

Branch `umpire-fn126-2` in wt/lane-a. It is merged with `umpire` 02e0e99e8f (fn-124.6) in ee95b2ebbe. Commits:
- cf29b2f8eb: lint, misnamed feature file;
- 579acdbe32: layout;
- ee95b2ebbe: merge;
- 8d215df628: Go tests;
- fb8899ac00: pinned Runs re-recorded;
- 1d60f14527: lint acceptance prose;
- 0406b89c73: R10 layout test;
- 3bf2fede94: docs.

After review (NEEDS_WORK, one P2):
- ce38db9cda: lint covers companions outside feature files;
- 9bafe2aeab: R10 bare-name prose pattern and two stale references;
- ff18c8608d and cbe02abe84: golden check that each function-name substitution names a current Function.

### What landed
- **R1, R2, R3.** Five feature files replace 17 per-kind files, so all 30 per-kind files are now 8 feature files. Each file reads in this order: header, types, signature, one module object per machine, then `Files`.
  - `nexuscaller/NexusCaller.scala`: `Product`, `Protocol`, `HandlerWorker`, `NexusCaller`, `Control` (keeps its own pin and family).
  - `nexuscaller/closepolicy/ClosePolicy.scala`: one `RejectAfterClose` holding all ten designs. It pins `…closepolicy.Model$package$` for its monitors and assumptions.
  - `nexusoperation/NexusOperation.scala`: `Operation`.
  - `shared/taskqueue/TaskQueue.scala`: `DispatchQueue` (pins `…System$package$` for its two assumptions) and `MatchingQueue`.
  - `shared/worker/Worker.scala`: `Polling`.
  - Step functions sit in `effects`; Properties and claim bundles in `properties`; capabilities in `laws`; Scenarios, then Queries, in `queries`.
  - `Realization.scala` files changed in qualified references only.
- **Lines per folder** (files/lines, from `.flow/tmp/fn-126/baseline.py`, before → after):

  | Folder | Files | Lines | Cross-file refs |
  | --- | --- | --- | --- |
  | nexuscaller | 5 → 2 | 1337 → 1353 | 65 → 15 |
  | closepolicy | 4 → 1 | 1095 → 1120 | 68 → 0 |
  | nexusoperation | 6 → 2 | 335 → 343 | 25 → 9 |
  | taskqueue | 3 → 1 | 412 → 437 | 24 → 0 |
  | worker | 1 → 1 | 85 → 94 | 0 → 0 |
  | shared/Bounds.scala (new) | 1 | 14 | — |

  - `disabled` is 74 and inverted guards 45, both unchanged.
  - Standalone activity tree, unchanged apart from the bounds imports: 1076 / 449 / 291 lines.
  - Full counts: `fn126-2/after-counts.txt`.
- **Shared bounds (R13).** `shared/Bounds.scala` declares `object Bounds` with `three` (used by activity, nexuscaller and nexusoperation) and `four` (activity and nexuscaller). Users import it as `shared.Bounds.{four, three}`.
  - The study's `five` and `twelve` are NOT identical. The close policy's `four`/`five`/`twelve` search 1<<20 / 1<<20 / 1<<22, against the activity's `five` (65536) and the task queue's `twelve` (262144).
  - Merging them would change `limits.search` in the IR, and renaming is forbidden. So they stay in their own feature files: each bound is declared once, but some names are shared.
- **Lint (carry-forward).** In a features/ or shared/ folder that has no feature file, Order.scala (d) now refuses a Model declaration in a file not named after its folder, at its line.
  - Fixture: `model/irgen/testdata/misnamed/Lamp.scala`.
  - Test: "the declaration-order lint refuses a Model in a folder with no feature file". It expects refusals at :18 (machine object), :29 (top-level Property) and :34 (a type's companion).
  - After review, a type's companion is read in both non-feature-file branches. Beside a feature file, `initOrder/Forward.scala:29` is refused (a Scenario in a companion).
  - The message names the folder ("the file named after the folder, in <folder>/") instead of guessing a capitalized file name.

### R5 deltas (all recorded)
- **Projection.** `.flow/tmp/fn-126/fn126-2/project2.py` is a generalized task-2 projection. It is strict on paths: task 1's collapse of the standalone activity tree was dropped after review, so every path outside the retired-file map must stay the same file. It compares against a fresh `before/` = `umpire` 02e0e99e8f after merge (REV recorded there); the previous base is kept in `before-fdf9c66bff/`.
  - `ir-deltas.json`: OK. 25 files are equal under the deltas and 13 are identical.
  - `lifts-deltas.json`: OK. hints, hintsRefused and taskqueue are equal under the deltas; the rest are identical.
- **Positions.** File paths and lines in the IR and Cases.
- **Source root strings.** 30 roots, listed in `root_moves`.
- **Function symbols.** 91 moved, listed in `function_renames`.
  - The rule: a function moves only from its former owner to its assigned module object, into the object itself or one of its sections. Each moved symbol must match exactly one symbol in the regenerated file.
  - One rename: `Operation$.ends` → `Operation$.end`, as task 1 did.
- **Law-sidecar binding text.** `…Model$package.` → `…NexusOperation$package.`, the same kind of delta as task 1's.
- **Lint acceptance `because` prose** in five `*.lint.json` (`LINT_PROSE`). Kinds, owners and subjects are unchanged.
- **Golden `config.json`:**
  - 29 `source_root_moves` retargeted and 4 added (`nexusProduct`, `nexusProtocol`, `handlerWorker`, `nexusCaller`);
  - 25 `function_name_substitutions` retargeted (`new` side only; old frozen names unchanged), and two dead ones dropped (`Product$.productStep`, `Protocol$.moves`, gone from the IR since fn-114.2), which the new check found;
  - `Config.FunctionsCurrent` (`tools/umpire/internal/golden`) now refuses a substitution whose new name names no current Function. It runs over the real configuration in `TestMigrationGoldensAdmitOnlyTheProjection` and has a failing case in `TestProjectionIsClosed`;
  - a `source_path_merges` entry for `nexuscaller/NexusCaller.scala`;
  - a `source_path_splits` entry for `taskqueue/TaskQueue.scala`.
  - The original-baseline tests pass.
- **Unchanged:** IDs, type names, machine, Property, Scenario, Query, Limits and law names, tables, answers, lint findings and coverage.
- **fn-124.6's own deltas.** forgedCompletion gains `conformanceReason = Some(Reason.incomplete)`, and `nexus-control.json` and the manifest change with it. These arrive through the merge and are inside the projection's base.

### R10 layout test
- `TestRetiredModelPathsStayRetired` now also fails on:
  - any of the five retired names under `model/temporal/{features,shared}`;
  - a live file naming one: by a folder path, a Model-folder-relative path such as `worker/Model.scala`, or joined Go path parts.
- `TestRetiredFeatureFilesAreFound` has one subtest per name, each proving it is found in a nested Model folder and not under capabilities/ or irgen testdata.
- `TestRetiredModelMentionsAreFound` gains 11 path cases and, after review, 8 bare-name prose cases:
  - the prose pattern matches a per-kind name after "folder's", "feature's" or "Model's", or after the prepositions off, in, from, beside or see;
  - it allowlists `Capabilities.scala` in the framework's, the kit's and the IR generator's files, and on lines naming their folders.
- The bare-name pattern found two stale references, now reworded: the kit's `capabilities/Capabilities.scala` header and `conformance/played_test.go`.
- Live mentions fixed: README, UMPIRE_MODULES, the lint prose, Scala comments, Go tests, the `SyntaxRule.test.scala` synthetic paths and the `isolation_test` legacy path.

### Docs (R9 layout half)
- `model/README.md`:
  - the example and its line references (NexusCaller.scala 473/540/585);
  - the `object Files` irFile text;
  - the full layout tree with all folders;
  - a retired-names and shared-bounds paragraph;
  - lint table row (d) and the misnamed fixture;
  - the capabilities steps;
  - the control machine path.
- `.plans/UMPIRE_MODULES.md`: the Models, Nexus operation, capabilities and task queue rows, the step-function note and the IR-roots note.
- `.plans/QUINT_MODULE_LAYOUT.md` and `DSL_SIMPLIFICATION.md`: marked as landed, including the bounds finding.
- `SEMANTICS.md`, `UMPIRE4_VISION.md` and `AGENTS.md` name no per-kind file: no change needed.

### Gates (logs in .flow/tmp/fn-126/fn126-2/)
- `make umpire-gen-model --skip-go-checks`: `gen1.log` (pre-merge), `gen2.log` (post-merge). Both exit 0.
- `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`: `model-gate.log`, exit 0. This includes the irgen munit fixtures (misnamed among them) and the Models' munit tests.
- `make lint-model`: `lint-model.log`, exit 0 (the scalafix warnings are pre-existing).
- Original baseline: `baseline2.log` (golden, lower), `baseline-model1.log` (model; its failure fixed in the test).
- Full Go suite, `-json -p 2`: `go-suite2.json`, 4m01s wall, 0 failures.
  - `tools/umpire/model` and `export` were not OOM-killed this time.
  - Slowest: TestOriginalBaselineCases 48.9s, TestOriginalBaselineModel 47.2s, TestMigrationProjectionPreservesSemantics 44.1s.
  - The first run (`go-suite.json`) had failures, all fixed: position tests, close-policy source regexes, and assess/canary pinned identities.
- `umpire-gen-cases`, `umpire-gen-fixtures` and `canary-gen-case` regenerate path-only changes. The `check-*` variants are logged by target name, each 0.
- `make umpire-rerecord-pinned-runs`: `rerecord.log`, exit 0. Both pinned Runs were re-recorded live.
- `lint-code-fast`: `lint-code-fast.log`, 0 issues.

### Review reruns
- Model gate: `model-gate3.log`, exit 0.
- `tools/umpire/model`, `internal/golden` and `conformance` at -p 1: `review-go.json`. All pass, apart from one golden test-setup error, fixed in cbe02abe84; the rerun is in `review-golden.log`.
- Original baseline, `lower`: `review-baseline-lower.log`, ok. The `model` and `golden` original-baseline tests ran in the package runs above.
- `lint-code-fast`: `lint-code-fast2.log`, 0 issues.
- `lint-model`: `lint-model2.log`, 0.
- `tools/umpire/lint` was not rerun: no lifter expected output shifted (only refusal fixtures changed).

### Decisions
1. **Module object names.** They keep today's vocabulary names (`Product`, `Protocol`, `Operation`), following task 1. New objects take R1's pre-R18 names (`HandlerWorker`, `NexusCaller`, `RejectAfterClose`, `DispatchQueue`, `MatchingQueue`, `Polling`). `NexusOperation`/`ActivityProduct`-style renames are left to task 4.
2. **Wildcard imports.** `ProtocolFact.*` and `ProductFact.*` are imported inside `effects` only. At object level, `ProtocolFact.pendingAttempts` would clash with the top-level Observation `pendingAttempts` the machine's evidence reads.
3. **`unscheduled`** becomes `Protocol.unscheduled` (vocabulary), like the activity's `Protocol.unstarted`.
4. **Close policy.** The Boolean promise predicates (`closedHistoryIsFrozen`, `awaitingOwner`, …) sit in `properties`; the predicates the monitors read stay vocabulary.
5. **Bounds.** Only identical bounds are shared (see above); `object Bounds` is used rather than package-level vals, so every use shows an import.
6. **Naming enforcement.** Done in the lint, not in the layout test. The lint catches a misnamed file the moment it holds Models; the layout test keeps the five specific names retired.
7. **Pinned Runs.** The control and canary Case identities moved because their source path changed. Following fn-114.9's precedent, I re-recorded live through `make umpire-rerecord-pinned-runs`, re-rendered the receipt goldens and repinned the canary policy. A first attempt to patch the run's case hash by hand was reverted.

### For the owner
- The study miscounted `five` and `twelve` as shared bounds; `four` is the real second one. R13's "No bound is declared twice" holds per bound, not per name.
- The pinned Runs were re-recorded live (new Run identities in `replay/testdata` and `canary/assessment/testdata`, and new receipt goldens). This follows fn-114.9's precedent, but it is a change to recorded artifacts.
- Commit cf29b2f8eb does not build on its own; that cannot be fixed without rewriting history.
- Two dead golden function substitutions were removed (see R5).
- The merge needed `touch api/umpire/v1/*.go` and `make proto/api.binpb`, both local build state. The gate refused a stale mtime after the merge brought in `ir.proto`.

Subagents used: 0.

Review: claude-opus-5-5 at high, fresh context (host-dispatched subagent). Writer and reviewer are the same family (Opus).
- Round 1: NEEDS_WORK, one P2 and five P3s.
  - P2: two live comments still named per-kind files, and R10's folder-prefixed pattern missed them.
  - P3s: lint (d) skipped companions outside feature files; the projection collapsed paths; substitution `new` sides went unchecked; the refusal message guessed a capitalization; cf29b2f8eb does not build alone.
  - The reviewer verified `before/` byte for byte, reran the projection strictly, and compared 423 ID and type values raw.
- All fixed in ce38db9cda..cbe02abe84, except cf29b2f8eb, which cannot change without rewriting history and is accepted as is.
- Round 2: SHIP. The reviewer reran `TestRetired*`, `TestProjectionIsClosed` and `TestMigrationGoldensAdmitOnly*` at HEAD (pass).
- The one P3 left, the R10 prose check missing "its own `Capabilities.scala`", was applied by the host in 1cee51c789 with a test case. `TestRetired*` passes over the live tree.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cf29b2f8eb, 579acdbe32, ee95b2ebbe, 8d215df628, fb8899ac00, 1d60f14527, 0406b89c73, 3bf2fede94, ce38db9cda, 9bafe2aeab, ff18c8608d, cbe02abe84, 1cee51c789
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks, make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks, make lint-model, go test -count=1 -json -tags test_dep -p 2 -timeout 30m ./tools/umpire/..., go test -tags test_dep -count=1 -p 1 -run 'OriginalBaseline|Migration' ./tools/umpire/internal/golden ./tools/umpire/lower ./tools/umpire/model, make umpire-check-cases, make umpire-check-fixtures, make canary-check-case, make umpire-rerecord-pinned-runs, make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast, python3 .flow/tmp/fn-126/fn126-2/project2.py .flow/tmp/fn-126/fn126-2/before/ir model/ir .flow/tmp/fn-126/fn126-2/before/cases model/cases, go test -count=1 -json -tags test_dep -p 1 ./tools/umpire/model ./tools/umpire/internal/golden ./tools/umpire/conformance, go test -count=1 -tags test_dep -p 1 -run TestRetired ./tools/umpire/model/ (after 1cee51c789, ok)
- PRs: