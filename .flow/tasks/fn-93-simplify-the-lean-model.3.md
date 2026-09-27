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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
