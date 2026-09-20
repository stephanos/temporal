---
satisfies: [R1]
---
# fn-86-retire-hand-written-models-one.1 The hand-written inventory, its lint-model reconciliation check, and the typed-unary Contract baseline

## Description
Confirm the spec's "What is hand-written today" table file by file (R1): a committed inventory of every non-test module under `Temporal.Feature`, `Temporal.Testpilot` and `Umpire.Examples` that builds Umpire records or Testpilot Cases without the commands, each with the Properties, goldens, fixtures, live tests and tools that read it and a destination. The check is a new inventory issue in `lint-model`'s existing reconciliation over the import graph, not a Go tool. Snapshot the typed-unary Contract as the baseline task .2's proof point compares against. Nothing moves in this task.

**Size:** M
**Files:** `model/HANDWRITTEN_INVENTORY.md` (new; follows the ledger shape of `tools/umpire/CLEANUP_INVENTORY.md`), `model/ModelLint/ImportGraph.lean` (an `InventoryIssue` constructor: a production module under the three roots that imports an authoring owner directly and is absent from the inventory), `model/ModelLint.lean` (reads the inventory file's module list), `model/ModelLint/ImportGraphTests.lean`, `tests/testcore/testpilot/baseline/typed-unary-contract.json` (new: the Contract of today's typed-unary fixture, extracted through the generator), `Makefile` (the inventory file is an input of `lint-model`)
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
- Adjusted 2026-09-19 after fn-85 .7 landed (fn-85 .8 to .13 are still open, so re-read at start
  as the F5 line says): `Success/` today holds `Model.lean` (the `lifecycle` machine, the
  `nexusSuccessTests` set and its `case nexusSuccessSet`), `Tests.lean`, `RaceSyntaxTests.lean` (a
  command-authored second lifecycle, not a Race module), `TypedUnary.lean`, `TypedNexus.lean`,
  `Tests/{TypedUnary,TypedNexus}.lean`, `Nexus.md` and `Integration.md`. `register_case` has four
  lines, all in `model/Temporal/Tool/Testpilot.lean:20-27`.
  `Temporal/Feature/Nexus/Tests/{Commands,Machines,SecondModel}.lean` are fn-85's command specimens
  (test modules, not hand-written). `Temporal.Case.Realization.Nexus` is a production module that
  imports `Umpire.Case.Producer` and is outside R7's rule by design, so list it as "kept:
  realization" rather than as hand-written. `Temporal.Case.Template.*` and
  `Temporal.Case.Tests.{Template,ProofPoint}` are fn-85 .11's to delete and are inventory rows only
  if .1 runs before .11 lands. `Umpire/Tests/MigrationCompatibility.lean:121` already reads
  `compatibilityFamilies := ["switch"]`; the Temporal-side families are
  `TemporalModelTests.lean:24-37`. The typed-unary fixture's Contract will have moved by fn-85 .8
  and .9 (typed instructions, observation declarations) before this task runs; the baseline is
  whatever the generator writes on the day, which is why it is a scaffold.

### Investigation targets
**Required:**
- `model/ModelLint/ImportGraph.lean:175-188,208-215,294-322` — `InventoryIssue`, `isProductionModule`, `reconcile`
- `model/ModelLint.lean:28-53` — sources, metadata, the `reconcile` call (the file is 112 lines; there is no `:124`)
- `tools/umpire/CLEANUP_INVENTORY.md:1-8` — the ledger precedent and its "consumer evidence, not absence of an import" standard
- `model/Temporal/System/Nexus/ImplementationLink.lean:1-2,35-350,484,528` — the two Feature imports and what they pin
- `model/Temporal/Tool/{Goldens,Inspect,NexusDiscovery}.lean` — the tools' inputs

**Optional:**
- `model/TemporalModelTests.lean:24-37`, `model/TemporalExperimentalTests.lean:8-20`, `model/UmpireTests.lean:50-52` — the compatibility-family pins

### Key context
- `model/README.md:37` names a `Success.Producer` that does not exist; fn-85 .13 now takes that drift (recorded there on 2026-09-19); confirm it is gone rather than re-recording it for task .9.
- fn-85 .11 replaces `Success/Model.lean` with the Caller Model; confirm what remains under `Success/` before listing (the dated Approach note says what is there as of 2026-09-19).

## Acceptance
- [x] `model/HANDWRITTEN_INVENTORY.md` lists every module in the scout table with every reader and a destination (migrate to a named task, delete with the named spec that records its coverage, keep with the reason, or drop with a reason); the kept exception is listed with its two Feature imports
- [x] `lint-model` raises `handwrittenNotInventoried` for a planted production module that builds records and is missing from the inventory, and is clean for the tree as inventoried; the planted case is pinned in `ImportGraphTests`
- [x] the typed-unary Contract baseline is checked in and equals the generator's current output
- [x] `make lint-model` green (LEAN_NUM_THREADS=1); `make umpire-check-regression` exit 0


## Done summary

Done 2026-09-20; self-review. Commit b7a6509. Nothing moved.

### The tree as fn-85 left it

Re-read at start, as the F5 line asks. `Temporal/Case/Template/**`, the `fixture`-named `case`
form and `Temporal/Case/Tests/{Template,ProofPoint}.lean` are gone (fn-85 .11); the
Case-producing command is `case <name> realizes <set> as <realization>` (`Temporal/Case/Syntax.lean`),
which also admits a canary set (fn-85 .12). `Success/` holds `Model.lean` (the command-authored
`lifecycle` specimen with no set and no Case; it still imports `Race.Terminal`), `Tests.lean`,
`RaceSyntaxTests.lean`, `TypedUnary.lean`, `TypedNexus.lean`, `Tests/{TypedUnary,TypedNexus}.lean`,
`Nexus.md` and `Integration.md`. `register_case` has four lines in `Temporal/Tool/Testpilot.lean`
(`get-system-info`, `worker-outage`, `typed-unary`, `typed-nexus`) and `Registry.register_case`
stays until the last of them goes. The typed Nexus example is still the only emitter of
`StartNexusOperation`, `RespondNexus`, `NexusResponseKind` and the untyped completion result. The
`model/README.md:37` drift the Key context names is gone (fn-85 .13). The file lists of `.2`, `.3`
and `.6` are corrected on their records where the tree differs from what they cite.

### The inventory

`model/HANDWRITTEN_INVENTORY.md` follows `tools/umpire/CLEANUP_INVENTORY.md`'s ledger shape:
task-start commit, the reconciliation rule, four destinations, then four tables. The first is what
the lint reconciles -- the eleven production modules under the three roots that import an
authoring owner directly, found by scanning every `import` line under `Temporal/Feature`,
`Temporal/Testpilot` and `Umpire/Examples`: the two typed examples (migrate, .2 and .3), the
worker-outage and get-system-info Cases (migrate, .6), `Temporal.Testpilot.CaseSupport` (keep:
the realization's and the conformance Cases' helpers), `Lifecycle.Model`, the four `Operations`
modules and `Race.Authoring` (delete, .5, after .4 re-anchors the Implementation Link), each with
every Lean importer, golden, fixture, Go artifact and live test, tool, compatibility family and
document that reads it. The second table lists what the spec's table names without an owner
import of its own (`Observation`: drop, the Umpire evidence tests remain the prover; the three
`Experimental` modules: delete, inputs to fn-33; `Success.Model`: the command specimen, kept until
.5 drops its `Race.Terminal` import; `Umpire.Examples.Switch`: migrate, .7, with its eleven
importers, four goldens, generated view and experiment fixture). The third is what is kept with
the reason: the Implementation Link with its two Feature imports (`Lifecycle`, `Race.Terminal`),
the realization, `Conformance`. The fourth is the readers that are not modules: `Goldens`,
`NexusDiscovery` (delete with Operations), `Inspect` with the `umpire-inspect/list/explain`
targets and `UMPIRE_REGRESSION_INSPECTOR` (delete with Operations unless the user asks to re-point
it at the Caller Model: the open question, recorded), the compatibility-family pins, and the six
documents with their destinations.

### The reconciliation check

`Tools.LeanSourceInventory.InventoryIssue` gains `handwrittenNotInventoried (module imported)`,
keyed for the sorted output like its siblings. `ModelLint.ImportGraph.Policy` gains
`handwrittenRoots` (`Temporal.Feature`, `Temporal.Testpilot`, `Umpire.Examples`) and
`authoringOwners` (the six R7 owners); `Policy.handwrittenImport?` says whether a module record
is hand-written by that definition (classified, under a root, production by the existing
`isProductionModule` predicate, importing an owner directly) and `reconcileHandwritten` reports
each such module the ledger does not list, sorted by name. `inventoriedModules` reads the ledger:
the first cell of each table row that is exactly a backticked qualified name. `ModelLint`'s
`lintImportGraph` reads `HANDWRITTEN_INVENTORY.md` from the model root it runs in and appends the
issues to the reconciliation's, and the Makefile's `lint-model` target tests the file exists before
building, so the ledger is an input of the gate. `ImportGraphTests.testHandwrittenInventory` pins:
the parser over a three-row ledger, the planted `Temporal.Feature.Nexus.Planted` importing
`Umpire.Model.Table` reported with its module and import, a listed module, a `*Tests` module, a
realization under `Temporal.Case` and a module importing only `Umpire.Command` all clean, and the
rendered diagnostic. On the real tree the lint is clean with the ledger as written (and, with the `Race.Authoring` row removed for one run, reports `hand-written module not inventoried: Temporal.Feature.Nexus.Race.Authoring imports Umpire.Property.Elab directly and is missing from HANDWRITTEN_INVENTORY.md`, then passes again with the row restored).

### The baseline

`tests/testcore/testpilot/baseline/typed-unary-contract.json` is the `contract` block of
`typed-unary-case.json` as the generator wrote it, cut out of the fixture and dedented, so it is
the generator's current output byte for byte (checked: the block re-indented equals the fixture's,
and both decode to the same value); `baseline/README.md` says it is a scaffold with no generator
and no reader that task .2 deletes once its comparison has passed. It sits beside `testdata/`
rather than inside it, as the task's file list had it: `umpire-check-case-runtime-conformance`
diffs the fixture directory against the generator's output and rejected the subdirectory ("Only
in testdata: baseline"), so a file the generator does not write cannot live there.

### Gates

`lake exe umpire-lint-tests` passes; `LEAN_NUM_THREADS=1 make lint-model` at the fn-85 baseline: the import-graph and Batteries steps pass with the ledger read, and the `lake lint` step reports the same two generated `Proto.lean` errors and 41 pre-existing warnings, none new;
`go test ./tests/testcore/testpilot/...` ok; `make umpire-check-regression` exit 0 with
29 passing live identities.

## Evidence
- Commits: b7a6509
- Tests: `cd model && lake build umpire-lint-tests umpire-lint && lake exe umpire-lint-tests`; `LEAN_NUM_THREADS=1 make lint-model`; `go test -count=1 -tags test_dep ./tests/testcore/testpilot/...`; `CC=/usr/bin/cc TMPDIR=$(cd /tmp && pwd -P) make umpire-check-regression`
- PRs:
