# fn-114-state-every-scala-model-declaration-once.9 Rename model/ tool folders and group model/temporal into features/ and shared/

## Description
Owner request (2026-10-04): several `model/` folder names do not say what the folder holds. Rename them so a newcomer can tell the role from the path, and keep the behavior byte-identical.

**Cross-spec entry gate:** after fn-114.7, before fn-114.8. Every Model rewrite (fn-112, fn-114.1–.7) has landed, so the renames do not collide with in-flight Model edits.

**Size:** M
**Files:** `model/{gate,lifter,metrics,gen}/**` (moved), `model/.gitignore`, `model/project.scala`, the scala-cli `project.scala` files, `Makefile` (model targets and paths), `.github/workflows/**` that name the targets or paths, `tools/umpire/**` tests and golden configs that name the paths, `model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`, and the path references in **open** Flow task/spec files.
**Touches:** [model/**, Makefile, .github/workflows/**, tools/umpire/**, .plans/UMPIRE_MODULES.md, .flow/tasks/**, .flow/specs/**]

### Renames
| Now | New | Why |
| --- | --- | --- |
| `model/lifter/` (package `umpire.lift`) | `model/irgen/` (package `umpire.irgen`) | It generates Umpire IR from the compiled Scala Models; "lifter" is jargon. In live prose, call it "the IR generator". |
| `model/gate/` (package `umpire.gate`) | `model/check/` (package `umpire.check`) | It is the model check that `make umpire-check-model` runs. |
| `model/metrics/` (one file in package `umpire.gate`, a separate project only to keep one main class) | folded into `model/check/` as a second entry point | It reuses the check's sources; the folder owned no code. |
| `model/gen/` (gitignored build cache: scalapb jars, classpath, protoc plugin) | `model/build/` | `ir/` and `cases/` are the generated artifacts; the cache should not be called `gen`. |

`model/ir/`, `model/cases/`, `model/temporal/` keep their names. `model/umpire/` (the DSL framework) is out of scope unless the owner asks.

### Approach
- `git mv` the folders so history follows; rename packages and every import; update `using file`/`mainClass` directives and Makefile variables (`MODEL_ROOT`, jar paths) and targets (`lint-model-lifter`, `lint-model-lifts`, `lint-model-gate`, `lint-model-metrics` → names matching the new folders). Update CI workflows that call renamed targets; no aliases.
- Update live documentation (`model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`) and the paths in open Flow tasks/specs. Leave closed specs, done-task records and dated `.plans/` research as history.
- Add or adjust a check (e.g. the existing ownership/architecture test) that fails if the old paths reappear.

### Investigation targets
**Required:** `Makefile:690-760`, `model/.gitignore`, `model/metrics/Metrics.scala`, `model/gate/project.scala`, `model/lifter/project.scala`, `tools/umpire/internal/golden/config.json`, `tools/umpire/model/ownership_test.go`.

### Grouping under model/temporal (owner request, 2026-10-04)
`model/temporal/` mixes feature Models with shared Temporal parts. Group them in the same rename pass so paths change once:

| Now | New | Package |
| --- | --- | --- |
| `model/temporal/{standaloneactivity,nexuscaller,nexusoperation}` | `model/temporal/features/<same>` | `temporal.features.<same>` |
| `model/temporal/{taskqueue,worker}` (parts the features compose) | `model/temporal/shared/<same>` | `temporal.shared.<same>` |
| `model/temporal/{capabilities,realize}` | unchanged | unchanged |

Package renames change owner symbols, so keep every Definition ID and IR type name with the existing mechanisms (DefinitionScope pins, the lifter's pinned-file type-name rule) and record the root and path moves in the golden config (append-only). Update `IrFiles.scala` roots, imports, the guard tests, `model/README.md`, `.plans/UMPIRE_MODULES.md` and paths in open Flow tasks/specs.

Name choice for the shared folder: `shared/`. Rejected: `entities/` (features declare entities too), `system/` (taken by the system contract, `SystemFamily` and `activity-system.json`), `components/` (a CHASM term in this repository), `infra/` (reads as deployment infrastructure; the worker is a client process).
## Acceptance
- [ ] `model/lifter`, `model/gate`, `model/metrics` and `model/gen` no longer exist; `model/irgen`, `model/check` and `model/build` replace them with packages `umpire.irgen` and `umpire.check`; source metrics run from `model/check`.
- [ ] `make umpire-gen-model` reproduces `model/ir/**` and `model/cases/**` byte-identically, and the fn-112.1 original-baseline check passes unchanged.
- [ ] The model gate, `lint-model` (renamed sub-targets), the Go tooling suite and `make lint-code-fast` pass; CI workflows reference only the new targets and paths.
- [ ] Live docs and open Flow tasks/specs name only the new paths and "IR generator"; a test fails if an old path is reintroduced.
- [ ] Feature Models live under `model/temporal/features/` and the shared task queue and worker under `model/temporal/shared/`, with matching packages; Definition IDs, IR type names, `model/ir/**` and `model/cases/**` are unchanged apart from positions and recorded path/root moves.
## Done summary
Renamed the model/ tool folders and grouped model/temporal. Behavior and identity are unchanged. Commits: e1fe271011 (rename), 142d57296c (diagnostic name and golden helper), b314b4bf69 (review round 1).

**Renames** (all with git mv)
- model/lifter (umpire.lift) is now model/irgen (umpire.irgen). Live prose calls it "the IR generator".
- model/gate (umpire.gate) is now model/check (umpire.check), with main class `run`.
- model/metrics is folded into model/check as a second entry point: `scala-cli run model/check --main-class umpire.check.metrics -- <dirs>`.
- The build cache model/gen is now model/build. Updated: .gitignore, Makefile `MODEL_BUILD`, the check, the IR generator's tests, .scalafmt.conf, and the vocabulary walk.
- model/temporal/{standaloneactivity,nexuscaller,nexusoperation} moved to model/temporal/features. Their packages are `temporal.features.*`, declared as `package temporal` followed by `package features.<x>`, which keeps line counts.
- model/temporal/{taskqueue,worker} moved to model/temporal/shared (`temporal.shared.*`).
- Make targets: `lint-model-irgen`, `lint-model-irgen-lifts` and `lint-model-check`. `lint-model-metrics` is gone. There are no aliases, and no CI workflow names these targets.

**Identity**
- New `DefinitionScope` pins:
  - file pins: nexuscaller, closepolicy, nexusoperation and standaloneactivity `Model.scala`;
  - object pins: `Control`, `NexusRealization`, `OperationRealization`, `ActivityRealization`.
- The existing pins and the pinned-file type-name rule keep every Definition ID and IR type name.
- After mapping the moves back (`.flow/tmp/fn114-9/compare.py`, `ir-equivalence.txt`), these are identical apart from position lines:
  - model/ir and the law sidecars;
  - the expected IR in model/irgen/testdata/lifts/expected;
  - model/cases;
  - rejects.txt.
- Two kinds of compiler name follow the package:
  - source roots, which are recorded;
  - Function names, sidecar binding strings and two refusals that name the moved symbol `Inputs.scheduleToStart`.
- No golden reads them: Functions are compared by reference, and the original baseline is sourceless and functionless. Because the IR lists Functions by name, its Function list is reordered.
- original.json is untouched.

**Golden config.** Two keys were appended: `source_path_moves` and `source_package_moves`. nexusoperation is not listed because no compared input names it. The moves are applied in:
- `Unmoved` (MatchAt, before Unsplit) and `UnmoveSources`;
- the location projection, through `UnmovedText`;
- inventory lookup, through `MovedPath`;
- `RootsApply`, where the list is closed.
Unit tests cover each of these.

**Canary.** The Case paths moved, so the policy's caseIdentity is now 58fb49bc…. Both pinned Runs were recorded again live with `make umpire-rerecord-pinned-runs`, and the receipt goldens were rendered from the new record.

**Guard.** `tools/umpire/model/layout_test.go` adds two tests:
- `TestRetiredModelPathsStayRetired` fails if a moved folder holds sources again, or if a live file names an old path, Scala package, package clause or import, joined path, or target. It covers model/, the tools, Makefile, workflows, AGENTS.md, UMPIRE_MODULES and UMPIRE4_VISION.
- `TestModelFilesLeaveOutTheBuildCaches` checks that the model walk skips model/build and a stale model/gen.

**Docs.** Updated model/README.md, SEMANTICS.md, .plans/UMPIRE_MODULES.md, AGENTS.md, tools/umpire/README.md and the testpilot README, plus the paths in the open fn-114, 118, 119, 122, 123 and 125 specs and tasks. Closed specs and dated research are left as history.

**Decisions taken autonomously**
- Function names follow the compiler, not the pins. A pin rule for Function names cannot reproduce today's names: the worker and taskqueue files already pin to another file, and their Functions keep their own file's name. Changing that would be a new mechanism.
- Recorded the moves as prefix moves rather than one entry per root. The existing root moves then apply unchanged, and the config stays append-only.
- Parked the old cache and the empty folders in `.flow/tmp/fn114-9-old-model-gen` with mv, not rm. model/.gitignore keeps `/gen/` for other checkouts.
- Hints.scala's longer import was reformatted to 6 lines, so its reader pins moved by +5.
- Out of scope: model/umpire/Capabilities.scala marks `cited`'s citations `@unused`. A clean build of lint-model-models failed under -Wunused:all, and scalafix still exited 0, so the models were not being linted.

**Review.** Reviewer: claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Base 42548a1da4.
- Round 1: NEEDS_WORK. All four findings were fixed in b314b4bf69:
  - P2: a stale model/gen was neither ignored nor skipped;
  - P3: the "gen" skip;
  - P3: four duplicated prefix helpers;
  - P3: comment alignment.
- Round 2: SHIP, no findings.

**Gates.** Logs and the list are in `.flow/tmp/fn114-9/gates.txt`. All exit 0:
- model gate;
- OriginalBaseline;
- migration goldens;
- full Go suite (6057 tests). The only failure was the diagnostic name, fixed in 142d57296c; the affected packages were rerun;
- lint-model, lint-code-fast, umpire-check-lint, check-cases, check-fixtures, canary-check-case and rerecord.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e1fe271011, 142d57296c, b314b4bf69
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; model/ir, model/cases, lifts/expected identical to BASE after mapping recorded moves, ir-equivalence.txt), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -tags test_dep -count=1 -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), go test -tags test_dep -count=1 -p 2 -run 'Migration|Golden' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (6057 tests; one failure fixed in 142d57296c, affected packages rerun exit 0), make lint-model (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), make umpire-check-lint (exit 0), make umpire-check-cases (exit 0), make umpire-check-fixtures (exit 0), make canary-check-case (exit 0), make umpire-rerecord-pinned-runs (exit 0, both pinned Runs re-recorded live)
- PRs: