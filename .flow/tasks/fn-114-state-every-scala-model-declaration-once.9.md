# fn-114-state-every-scala-model-declaration-once.9 Rename model/ tool folders so each says what it holds

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

## Acceptance
- [ ] `model/lifter`, `model/gate`, `model/metrics` and `model/gen` no longer exist; `model/irgen`, `model/check` and `model/build` replace them with packages `umpire.irgen` and `umpire.check`; source metrics run from `model/check`.
- [ ] `make umpire-gen-model` reproduces `model/ir/**` and `model/cases/**` byte-identically, and the fn-112.1 original-baseline check passes unchanged.
- [ ] The model gate, `lint-model` (renamed sub-targets), the Go tooling suite and `make lint-code-fast` pass; CI workflows reference only the new targets and paths.
- [ ] Live docs and open Flow tasks/specs name only the new paths and "IR generator"; a test fails if an old path is reintroduced.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
