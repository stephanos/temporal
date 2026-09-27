---
satisfies: [R1]
---
# fn-93-simplify-the-lean-model.1 Install the schema and catalog checks for production Models (E1)

## Description
Lane E1. Production Models import only `Temporal.Case.Syntax`, so the `schema:` and `evidence:` checks installed by `Temporal.Case.Schema` and `Temporal.Case.Catalog` never run for them. Pin the defect with a negative fixture first, then fix the import.

**Size:** M
**Files:** `model/Temporal/Case/Syntax.lean` (imports), `model/Temporal/Case/Tests/ProductionImports.lean` (new negative fixture; name may differ), `model/TemporalModelTests.lean` (wire the fixture), any production Model under `model/Temporal/Feature/**/Model.lean` that turns out to carry a latent schema or catalog error
**Touches:** [model/Temporal/Case/Syntax.lean, model/Temporal/Case/Tests/**, model/TemporalModelTests.lean, model/Temporal/Feature/**/Model.lean]
**Depends on other specs:** fn-92.1 adds `Temporal/Feature/Worker/Model.lean`; it must elaborate with the checks on.

### Approach
1. Write the fixture: a module whose only import is `Temporal.Case.Syntax`, declaring a small action with an unknown `schema:` message and a set whose `evidence:` names an uncatalogued observation, each under `#guard_msgs`. Confirm it is red today (no message), which proves the defect.
2. Add `import Temporal.Case.Schema` and `import Temporal.Case.Catalog` to `Temporal/Case/Syntax.lean`. The hooks are `IO.Ref`s installed by `initialize` in those two modules; importing them is the whole fix.
3. `lake build` every production Model; fix each latent error in its own Model and list it in the receipt.
4. Record the cold-build cost of the new closure (`Temporal.API` and `Testpilot.Protocol` now reach every Model) in the receipt.

### Investigation targets
**Required:**
- `model/Temporal/Case/Syntax.lean:1-7` — current imports
- `model/Umpire/Command/Schema.lean:26-39`, `model/Temporal/Case/Schema.lean:82` — schema hook and its installer
- `model/Umpire/Command/Catalog.lean:24-36`, `model/Temporal/Case/Catalog.lean:81` — catalog hook and installer
- `model/Temporal/Feature/Nexus/Tests/Commands.lean` — the only module that imports both today
**Optional:**
- `model/ModelLint/ImportGraph.lean:155-170` — authoring-path policy, which exempts `Temporal.Case`

### Key context
- `Umpire.Command.checkSchema` accepts every name when no check is installed, so the fixture's `#guard_msgs` must expect the rejection text, not silence.

### Quick commands
```sh
cd model && lake build Temporal.Case.Syntax TemporalModelTests
make umpire-check-goldens
LEAN_NUM_THREADS=1 make lint-model
```

## Acceptance
- [ ] Fixture committed; it failed before the import change and passes after (receipt shows both runs)
- [ ] Every production Model, the Worker Model included, elaborates; latent errors fixed and listed
- [ ] `make umpire-check-goldens` and `make umpire-check-regression` byte-identical (R10)
- [ ] `LEAN_NUM_THREADS=1 make lint-model` green; receipt records build cost and, as the first fn-93 task, the start baseline split (generated / test / production) that R12's floors use


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
