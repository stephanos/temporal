---
satisfies: [R1]
---
# fn-160-one-ir-file-per-feature-collected-from.1 Inventory every consumer of today's IR file split

## Description
Audit everything that depends on how IR files are split today, before changing it. Produce a short written inventory, a `## Resolved via Codebase` section appended to the spec, listing every consumer of IR file boundaries or names with a stated migration, or a stated reason one file per feature can't serve it. No code changes.

**Size:** S
**Files:** none changed except the spec; read-only investigation
**Touches:** [.flow/specs/fn-160-one-ir-file-per-feature-collected-from.md]

### Approach
- **Model side:** list every `irFile` and its roots (the standalone activity, nexus standalone and nexus workflow `exports`). For each non-main file, record why it's separate according to its comment: the record designs, the held race, the forged control, the close designs.
- **Checkers given a whole file:** trace whether any consumer reads one IR file and relies on it holding only some Queries. The race file's comment says "so the history record's Queries are the ones its checkers were given". Check the exploration and replay bridge (`umpire-ir-bridge`), `umpire-fuzz`'s `--set` (whether an exploratory set is an IR file or a model-declared `explore(...)`), `umpire-run --model`, `umpire-assess --model`, and `umpire-replay`.
- **Generated artefacts:** how Case file names derive from the IR file name (`model/cases/<file>-<query>-case.json`, `manifest.json`), and the two byte-pinned trees (`tests/testcore/testpilot/testdata/generated`, `tools/canary/casebinding/testdata`).
- **Lint:** the per-file `<name>.lint.json` acceptances and `umpire-lint`'s orphan check.
- **Quint export:** the per-file slices in `tools/umpire/export` and the README tables that report them.
- **Go tests and harnesses:** every Go file naming an IR file (`tools/umpire/ir`, `export`, `cmd/umpire-run`, `tools/canary/preflight`, …).
- **Gate:** `model/check` (lift arguments, `settle`, `acceptWaivers`) and the `IrFiles` test in `model/temporal`.
- **Collection rule:** confirm the lifter can enumerate a package's machine objects, sections and realizations from TASTy, and what counts as a feature folder (one with a feature file). Note any declaration that sits outside its feature package but is only reached through it.
- For each consumer, write: what it keys on, what changes under one file per feature, and the migration. Flag anything that needs a split, and stop for the user's decision if there is any.

### Quick commands
```sh
grep -rn "irFile(" model/temporal
grep -rln "activity-standalone-race\|activity-standalone-record\|nexus-workflow-control\|nexus-workflow-close" tools tests common Makefile
```

## Acceptance
- [ ] The spec has a `## Resolved via Codebase` inventory naming every consumer of IR file boundaries or names, each with what it keys on and its migration.
- [ ] Whether any checker relies on a file holding only some Queries is answered, with the code that shows it.
- [ ] Any consumer that needs a split is flagged for the user's decision before fn-160.2 starts.


## Done summary
Read-only inventory complete; findings are in the spec's `## Resolved via Codebase`. No checker relies on an IR file holding only some Queries (the race file's comment is historical), so no split needs to stay. 15 consumers are inventoried with their migrations (exploration scope and ids, whole-file Go tests, lint acceptances, Case and pin renames, Quint slices). The one owner decision, where foundations' Queries go, was answered with option (a): foundation/actor/kind packages get their own IR file (R10). Exploration is scoped to the asked Query (R11). Both were folded into fn-160.2.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: