---
satisfies: [R3]
---
# fn-93-simplify-the-lean-model.3 Build every test module the lakefile roots miss (E3)

## Description
Lane E3. Seven modules are built by no root: `Umpire/CoreImportTests.lean`, `Umpire/Model/CheckImportTests.lean`, `Umpire/Inventory/Tests/{KnownGaps,SemanticStages,PlanningRuntime}.lean`, `Temporal/Tool/InventoryTests.lean`, and the facade `Umpire/Inventory.lean`. Wire each into a root or delete it when another built test pins the same behavior. `SemanticStages` and `PlanningRuntime` import modules lane B deletes; wire them anyway (B2/B3 edit them later).

**Size:** M
**Files:** `model/UmpireTests.lean`, `model/TemporalModelTests.lean`, the seven modules above, `model/Umpire.lean` (if the facade is imported there), a small test that names the two runtime-loaded lint modules (`Temporal/Lint.lean`, `Umpire/Lint.lean`)
**Touches:** [model/Tools/**, Makefile, model/UmpireTests.lean, model/TemporalModelTests.lean, model/Umpire/CoreImportTests.lean, model/Umpire/Model/CheckImportTests.lean, model/Umpire/Inventory.lean, model/Umpire/Inventory/Tests/**, model/Temporal/Tool/InventoryTests.lean, model/Umpire.lean, model/ModelLint/**]
**Depends on other specs:** fn-88.10 edits `Umpire/Inventory/Tests/**` and `Temporal/Tool/**`; re-read at start.

### Approach
- Recompute the unbuilt set first: walk the import closure of every `lean_lib`/`lean_exe` root in `model/lakefile.lean` and list `.lean` files outside it. Expect the seven plus the two lint modules.
- For each: add it to the matching aggregator import list and build it; fix what is red. Deleting instead requires naming the built test that pins the same behavior in the receipt.
- Add a guard (ModelLint test or a small script run by `lint-model`) that fails when a `.lean` file outside `.lake` is unreachable from the roots, allowlisting only the two runtime-loaded lint modules. If a script is simpler, put it under `model/Tools/` and call it from the existing `lint-model` recipe.

### Investigation targets
**Required:**
- `model/lakefile.lean` — library and exe roots (lines 84-158 for exes)
- `model/UmpireTests.lean`, `model/TemporalModelTests.lean` — aggregators
- `model/ModelLint.lean:67` — runtime loading of the lint modules
**Optional:**
- `model/Tools/LeanSourceInventory.lean` — existing source walk to reuse for the guard

### Quick commands
```sh
cd model && lake build UmpireTests TemporalModelTests
LEAN_NUM_THREADS=1 make lint-model
```
## Acceptance
- [ ] Every one of the seven modules is built by a root or deleted with its covering test named
- [ ] A guard fails on a newly added unreachable `.lean` file; only the two lint modules are allowlisted
- [ ] `lake build` of every root green; goldens and regression byte-identical
## Done summary
Wired all seven unbuilt test modules (`Umpire.CoreImportTests`, `Umpire.Model.CheckImportTests`,
`Umpire.Inventory.Tests.{KnownGaps,SemanticStages,PlanningRuntime}`, `Temporal.Tool.InventoryTests`)
and the `Umpire.Inventory` facade into `UmpireTests`/`TemporalModelTests`; the production facade
went into the test tree rather than `Umpire.lean` because a production import trips the existing
`inventoryIsolation` rule. Fixed one latent failure the wiring surfaced: `Temporal.Tool.Inventory`'s
`Inventory` struct duplicates its owning namespace, a builtin-lint warning the file never ran under
until it became reachable; suppressed locally with `set_option linter.extra.dupNamespace false` and
a comment, since the shared name is intentional.

Added `ModelLint.ImportGraph.checkUnbuilt`: BFS from a `buildRoots` policy list over the loaded
import graph, reporting any first-party module no root reaches apart from the two runtime-loaded
lint modules. Wired into `umpire-lint`'s pass/fail. The first fan-out review round (codex) flagged
`buildRoots` as a hand-maintained, unenforced mirror of `model/lakefile.lean`; three further rounds
each reproduced a new false-negative in the lakefile text scanner meant to close that gap (a
block-commented declaration, a multi-line `roots :=`/`root :=`, then a no-space `:=` and a
parenthesized array). The scanner now strips comments first, matches fields against
whitespace-collapsed text, and — since Lean syntax has no bound on how else a field can be
spelled — fails closed on any unrecognized `roots`/`root` field rather than ever silently defaulting
to the target's own name. Round 5 (single re-review) shipped clean.

stage: impl-review - ran [round 1 fanout 3f7ab73c (NEEDS_WORK, buildRoots vs lakefile) -> fix 1aff2737 -> round 2 fanout b616b06d (NEEDS_WORK, comments not stripped) -> fix 2696186d -> round 3 fanout b6a92831 (NEEDS_WORK, multi-line fields) -> fix 788d6314 -> round 4 fanout 0ac94cb0 (NEEDS_WORK, no-space/parenthesized fields) -> fix e4289375 -> round 5 single-dispatch SHIP]

Gate results:
- `lake build UmpireTests TemporalModelTests` and `lake build Umpire umpire-inspect umpire-inventory umpire-inventory-tests`: green
- `umpire-lint` (full ModelLint pipeline, including the new `buildRoots`-drift and unbuilt checks against the real lakefile): green, zero drift, zero unbuilt modules outside the allowlist
- `umpire-lint-tests` (synthetic suite, including all new unit tests for the lakefile scanner): green
- Scoped `lint-model-builtin`, two batches (split to fit this shared machine's memory under heavy concurrent load from other sessions): `ModelLint*` + `UmpireTests` + every fn-93.2 Umpire test module green; `TemporalModelTests` + every fn-93.2 Feature test module green. The fn-93.2 scoped lint passed. A final small-batch rerun of just the three files touched across the review fix rounds (`ModelLint`, `ModelLint.ImportGraph`, `ModelLint.ImportGraphTests`) reconfirmed green after the last fix; the two batches above predate that last commit but their content (UmpireTests/TemporalModelTests closures) does not depend on `ModelLint.*`, so they remain valid evidence.
- `make umpire-check-goldens`: green, byte-identical

Follow-up worth noting (not built, out of this task's scope): the lakefile scanner is still a text
heuristic bounded to the spellings it recognizes, now safely fail-closed rather than silently wrong
on the rest. Reading Lake's own evaluated `Workspace`/`Package` config (`Lake.loadWorkspace`, already
reachable via `ModelLint.lean`'s existing `import Lake.CLI.Main`) would remove that boundary
entirely; every reviewer round suggested it as the alternative to patching one more spelling.

Tier: opus at high (per model routing)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 81f798a88a808785eb9c7a2ac1db2712c0ce9e67, 1aff273752a7f7afa4b1d73e7ecafc2c1fe38b2c, 2696186d9a04bbee613a972ef5ce5a22abfb9853, 788d631429f74a197e147f0bb0fee12c1e88cb2e, e42893757cfe73a78b40b6b66d295c7ee3fc3403
- Tests: cd model && mise exec -- lake build UmpireTests TemporalModelTests, cd model && mise exec -- lake exe umpire-lint, cd model && mise exec -- lake exe umpire-lint-tests, make umpire-check-goldens, LEAN_NUM_THREADS=1 make lint-model-builtin (scoped, split batches, fn-93.2 + fn-93.3 modules green)
- PRs: