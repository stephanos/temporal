---
satisfies: [R1]
---
# fn-86-retire-hand-written-models-one.1 The hand-written inventory, its lint-model reconciliation check, and the typed-unary Contract baseline

## Description
Confirm the spec's "What is hand-written today" table file by file (R1): a committed inventory of every non-test module under `Temporal.Feature`, `Temporal.Testpilot` and `Umpire.Examples` that builds Umpire records or Testpilot Cases without the commands, each with the Properties, goldens, fixtures, live tests and tools that read it and a destination. The check is a new inventory issue in `lint-model`'s existing reconciliation over the import graph, not a Go tool. Snapshot the typed-unary Contract as the baseline task .2's proof point compares against. Nothing moves in this task.

**Size:** M
**Files:** `model/HANDWRITTEN_INVENTORY.md` (new; follows the ledger shape of `tools/umpire/CLEANUP_INVENTORY.md`), `model/ModelLint/ImportGraph.lean` (an `InventoryIssue` constructor: a production module under the three roots that imports an authoring owner directly and is absent from the inventory), `model/ModelLint.lean` (reads the inventory file's module list), `model/ModelLint/ImportGraphTests.lean`, `tests/testcore/testpilot/testdata/baseline/typed-unary-contract.json` (new: the Contract of today's typed-unary fixture, extracted through the generator), `Makefile` (the inventory file is an input of `lint-model`)
**Touches:** [model/HANDWRITTEN_INVENTORY.md, model/ModelLint.lean, model/ModelLint/**, tests/testcore/testpilot/testdata/baseline/**, Makefile]

### Approach
- Plan review round 1 (F5): this task touches nothing, so it is where the tree is re-read as fn-85
  left it. fn-85 deletes `Temporal/Case/Template/**`, the `case` command in
  `Temporal/Case/Syntax.lean` and `Nexus/Success/Model.lean`, and reshapes
  `Umpire/Case/Producer.lean` and `Temporal/Case/Registry.lean`. Check in particular whether
  `register_case` still exists and for which Cases, and correct the file lists of `.2`, `.3` and
  `.6` before `.2` starts.
- Plan review round 1 (F1): `testdata/baseline/typed-unary-contract.json` is a scaffold for `.2`'s
  comparison, not a checked-in artifact. No generator writes it and no gate regenerates it, so `.2`
  deletes it once the comparison has passed; say so in the file itself.
- Plan review round 1 (F4): "kept, with the reason" is a fourth destination. `Temporal.Testpilot`'s
  `Conformance.lean` and `CaseSupport.lean` build Cases by hand and stay, because they test the
  runtime, and `Temporal.System.Nexus` is the kept SEM-08 exception.
- Rows from the planning record's inventory (the scout table): the typed examples and their tests, fixtures, artifact and live tests; `WorkerOutage`, `GetSystemInfo`; `Umpire.Examples.Switch` and its 11 importers, 2 goldens, regression view and experiment fixture; Race (8 modules, `Terminal` imported by the kept Implementation Link); Lifecycle (imported by the Implementation Link and `Race/Lifecycle`); Operations (6 goldens, `Tool/Goldens`, `Tool/NexusDiscovery`, `Tool/Inspect`, `TemporalModelTests/Nexus/ImplementationLink`); `Observation` + `ObservationTests`; Experimental (`AutoClose` has no importer; `VariationSpace`, `Exploration`, `TemporalExperimentalTests`); the docs `Success/Nexus.md`, `Success/Integration.md`, `Race/{README,DESIGN,COVERAGE}.md`, `Nexus/COVERAGE.md`; and `Temporal.System.Nexus` as the kept exception with its two Feature imports.
- Destinations per the spec table plus the planner's decisions: `Observation.lean`'s offline evaluation is a recorded drop with `Umpire/Evidence/Tests/Evaluation.lean` named as the remaining prover; `NexusDiscovery`/`Inspect` and the `umpire-inspect/list/explain` targets are deleted with Operations unless the user asks to re-point them at the Caller Model (record the open question in the row); Race docs fold into fn-79's text; `Nexus/COVERAGE.md` is deleted (its citations are deleted modules); the two design sketches are deleted with the typed examples.
- Check: `ModelLint` already reconciles sources against loaded metadata (`reconcile`, `InventoryIssue.uncoveredSource`); add `handwrittenNotInventoried` raised for a production module under the three roots that imports `Umpire.Model`, `Umpire.Property`, `Umpire.Scenario`, `Umpire.Query`, `Umpire.Operation` or `Umpire.Case` directly and is not listed; a planted case in `ImportGraphTests` proves it fires.
- Baseline: extract `contract` from `typed-unary-case.json` through the generator's persisted form so task .2 can diff Contract reads structurally.

### Investigation targets
**Required:**
- `model/ModelLint/ImportGraph.lean:175-188,208-215,294-322` — `InventoryIssue`, `isProductionModule`, `reconcile`
- `model/ModelLint.lean:43-53,124` — sources, metadata, `main`
- `tools/umpire/CLEANUP_INVENTORY.md:1-8` — the ledger precedent and its "consumer evidence, not absence of an import" standard
- `model/Temporal/System/Nexus/ImplementationLink.lean:1-2,35-350,484,528` — the two Feature imports and what they pin
- `model/Temporal/Tool/{Goldens,Inspect,NexusDiscovery}.lean` — the tools' inputs

**Optional:**
- `model/TemporalModelTests.lean:24-37`, `model/TemporalExperimentalTests.lean:8-20`, `model/UmpireTests.lean:50-52` — the compatibility-family pins

### Key context
- `model/README.md:36` names a `Success.Producer` that does not exist; record it as a doc drift for task .9.
- fn-85 replaced `Success/Model.lean` with the Caller Model; confirm what remains under `Success/` before listing.

## Acceptance
- [ ] `model/HANDWRITTEN_INVENTORY.md` lists every module in the scout table with every reader and a destination (migrate to a named task, delete with the named spec that records its coverage, keep with the reason, or drop with a reason); the kept exception is listed with its two Feature imports
- [ ] `lint-model` raises `handwrittenNotInventoried` for a planted production module that builds records and is missing from the inventory, and is clean for the tree as inventoried; the planted case is pinned in `ImportGraphTests`
- [ ] the typed-unary Contract baseline is checked in and equals the generator's current output
- [ ] `make lint-model` green (LEAN_NUM_THREADS=1); `make umpire-check-regression` exit 0


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
