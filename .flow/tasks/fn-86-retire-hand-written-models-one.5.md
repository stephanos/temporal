---
satisfies: [R6]
---
# fn-86-retire-hand-written-models-one.5 Delete Race, Lifecycle, Operations, Observation and Experimental with their goldens, tools and pins; record behavior in fn-79 and fn-33

## Description
Remove every module the inventory marks for deletion (R6) with the goldens and fixtures only it produced, the tools that served only it, and the compatibility-family pins that named it; write the cancellation-race behavior into fn-79's spec and the exploration inputs into fn-33's. Nothing here needs fn-79 or fn-33 to be resumed.

**Size:** M (mostly deletion; the two spec-text records and the tool decision are the work)
**Files:** `model/Temporal/Feature/Nexus/Race/**` (8 modules and 3 docs), `Nexus/Lifecycle/**` and `Nexus/Lifecycle.lean`, `Nexus/LifecycleTests.lean`, `Nexus/Operations/**` and `Nexus/Operations.lean`, `Nexus/OperationsTests.lean`, `Nexus/Observation.lean`, `Nexus/ObservationTests.lean`, `Nexus/Experimental/**`, `Nexus/Fixtures/Operations*.json` (6 goldens), `Nexus/COVERAGE.md`, `model/Temporal/Feature/Nexus.lean` and `NexusTests.lean` (facade rewritten around the Caller Model), `model/Temporal/Tool/Goldens.lean:41-58` (the Operations loop), `model/Temporal/Tool/NexusDiscovery.lean`, `NexusDiscoveryTests.lean`, `Inspect.lean`, `InspectTests.lean` (deleted, with `Makefile:518-527` `umpire-inspect/list/explain`, unless re-pointed; see Approach), `model/lakefile.lean` (exe entries), `model/TemporalModelTests.lean:24-37`, `model/TemporalExperimentalTests.lean` (deleted with its lakefile root, or emptied), `model/Umpire/Tests/MigrationCompatibility.lean:121`, `model/UmpireTests.lean:50-52`, `Makefile` (`UMPIRE_GOLDEN_DIRECTORIES`, `UMPIRE_REGRESSION_FIXTURES`), `.flow/specs/fn-79-deferred-nexus-operation-cancellation.md` (the race behavior under `## Scope`, dated), `.flow/specs/fn-33-run-serial-bounded-semantic-exploration.md` (exploration inputs beside the campaign Space paragraph), `model/HANDWRITTEN_INVENTORY.md` (rows marked deleted)
**Touches:** [model/Temporal/Feature/Nexus/**, model/Temporal/Tool/**, model/lakefile.lean, model/TemporalModelTests.lean, model/TemporalExperimentalTests.lean, model/Umpire/Tests/MigrationCompatibility.lean, model/UmpireTests.lean, Makefile, .flow/specs/fn-79-deferred-nexus-operation-cancellation.md, .flow/specs/fn-33-run-serial-bounded-semantic-exploration.md, model/HANDWRITTEN_INVENTORY.md]

### Approach
- Plan review round 1 (F2), decided: **re-point the inspector, do not delete it.**
  `make umpire-inspect`, `umpire-list` and `umpire-explain` are documented developer entry points
  (`model/README.md:95`, `.plans/UMPIRE4_COMPONENTS.md:358-359`) and fn-85's
  `umpire-case --list/--render` renders Cases, not Plans, so it replaces neither `inspect` nor
  `explain`. `Temporal.Tool.Inspect`'s scenario registry loses its `Nexus.Operations` and
  `NexusDiscovery` entries and gains the Caller Model's Queries, keeping `Umpire.Examples.Switch`;
  the three Makefile targets stay and `umpire-list` prints the new registry.
- Order: Experimental first (`AutoClose` has no importer), then Observation, then Race (its `Terminal` importers are gone after fn-85 and task .4), then Operations, then Lifecycle; build after each.
- Tools: `NexusDiscovery` and `Inspect` exist only to serve Operations (their `productionRegistry` is Operations plus Switch). Default is deletion with the three Make targets and their tests, recorded in the inventory as a deliberate drop; if the user wants `umpire-inspect` kept, re-point `productionRegistry` at the Caller Model's Queries instead. Ask before deleting only if the receipt would otherwise remove a documented product surface the user named.
- Compatibility families: `TemporalModelTests.compatibilityFamilies` derives from `LifecycleTests` and `OperationsTests`; collapse to the families that remain (`switch` until task .7 re-authors it) and update all three `rfl` pins in one commit; delete `TemporalExperimentalTests` and its `lean_lib` root (it aggregates nothing) and note it for fn-46's root list (already annotated on fn-46 .2).
- fn-79 record: the cancel rows of the old Race model (request, delivery, delivered and rejected replies, the race with completion) as a dated paragraph under `## Scope` beside the existing 2026-09-10 note. fn-33 record: the `VariationSpace` and `Exploration` inputs (which Queries, which variation axes, budgets) beside the campaign Space paragraph under `## Contracts`.
- Observation drop: name `Umpire/Evidence/Tests/Evaluation.lean` as the remaining prover of the offline Observation evaluation in the inventory row.
- Adjusted 2026-09-19 after fn-85 .7 landed (fn-85 .10 and .11 still open): the Race sweep must
  not take `model/Temporal/Feature/Nexus/Success/RaceSyntaxTests.lean` by name -- it is a
  command-authored specimen that imports the Success Model, and fn-85 .11 moves or deletes it with
  that Model; nor `Temporal/Feature/Nexus/Tests/{Commands,Machines,SecondModel}.lean`, fn-85's
  command specimens, which .10 re-points at the Caller Model. `Umpire/Tests/MigrationCompatibility.lean:121`
  already reads `["switch"]`; the pins to collapse are the Temporal-side ones
  (`TemporalModelTests.lean:24-37`, `TemporalExperimentalTests.lean:8-20`). The inspector's new
  registry entries are the Caller Model's Queries under `Temporal.Feature.Nexus.<Caller>` as
  fn-85 .10 names them, and `umpire-case --list` (which also lists set-derived Cases since fn-85 .7)
  stays a Case renderer, not a Plan one.

### Investigation targets
**Required:**
- the scout inventory in the planning record (section 1b to 1d) — every importer of each deleted module
- `model/Temporal/Tool/Goldens.lean:41-74`, `Inspect.lean:83-90`, `NexusDiscovery.lean:2-4`
- `model/TemporalModelTests.lean:24-37`, `model/TemporalExperimentalTests.lean:8-20`, `model/UmpireTests.lean:50-52`, `model/Umpire/Tests/MigrationCompatibility.lean:121`
- `.flow/specs/fn-79-deferred-nexus-operation-cancellation.md:5-16` and `.flow/specs/fn-33-run-serial-bounded-semantic-exploration.md:28-40`
- `Makefile:85-89,94,106-110,520-527,588-600` — golden directories, `UMPIRE_REGRESSION_INSPECTOR`, the inventory and regression fixture lists, the three inspector targets, `umpire-check-goldens`

**Optional:**
- `model/Temporal/Feature/Nexus/Race/{README,DESIGN,COVERAGE}.md` — the behavior to summarize for fn-79

### Key context
- Memory: full integration gates must select the complete migrated suite; after deletion, `make umpire-check-goldens` and `umpire-check-regression-views` must still check a non-empty set, and the live-test gate still needs at least one passing identity.

## Acceptance
- [x] every deletion-row module, golden, fixture, doc and tool is gone; no import of a deleted module remains (the build proves it); the facade `Temporal.Feature.Nexus` re-exports the Caller Model
- [x] fn-79's spec carries the dated race-behavior record and fn-33's the exploration inputs; the inventory rows name them
- [x] the compatibility-family pins compile with the remaining families; `TemporalExperimentalTests` is removed from the lakefile
- [x] `make umpire-check-goldens`, `make umpire-check-regression-views`, `make lint-model`, `make umpire-check-regression` exit 0; `make umpire-inspect` either removed from the Makefile or shows the Caller Model


## Done summary

Done 2026-09-20; self-review. Commit 7766383.

### Deleted

`Temporal/Feature/Nexus/Race/**` (eight modules, three docs), `Nexus/Lifecycle/**` with
`Lifecycle.lean` and `LifecycleTests.lean`, `Nexus/Operations/**` with `Operations.lean` and
`OperationsTests.lean`, `Nexus/Observation.lean` and `ObservationTests.lean`, `Nexus/Experimental/**`
(`VariationSpace`, `Exploration`, `AutoClose` and the two tests), the six `Nexus/Fixtures/Operations*`
goldens (the directory with them, and its entry in `UMPIRE_GOLDEN_DIRECTORIES`),
`Nexus/COVERAGE.md`, `Temporal/Tool/NexusDiscovery.lean` and its tests, and
`TemporalExperimentalTests.lean` with its `lean_lib` and its place in the regression build line.
Order: Experimental, Observation, Race, Operations, Lifecycle; the build after the sweep is the
proof that no import of any of them remains (`lake build`, 586 targets).

### Rewritten around the Caller Model

- `Temporal.Feature.Nexus` imports the Caller and Pair Models and documents the reader order
  (Caller, Pair, `System.Nexus.Core`, the Implementation Link); `Temporal.Feature.NexusTests`
  checks the facade names and that the functional Queries find their claims through the facade.
- `Temporal.Tool.Inspect`: the registry is the Caller Model's eight Queries in declaration order
  (`temporal.nexus.caller.query.{syncCompletion,asyncCompletion,asyncFailure,handlerError,retry,
  scheduleToStartTimeout,startToCloseTimeout,terminalHolds}`) beside `switch.query.exact-action`;
  `inspect <id>` prints a found Query's planned Artifact and reports `planning-failure` for the
  verify Query, which selects none; `list` prints the registry as canonical JSON; `explain <id>`
  prints one Query's checked lineage (source, form, fingerprints of Query, Property, Scenario and
  Target, the outcome, the witness's actions, the Artifact checksum). The three Make targets stay;
  `umpire-check-regression` explains `asyncCompletion`. `InspectTests` pins the registry, both
  commands' determinism and their diagnostics.
- `Temporal.Tool.Goldens` renders four families (the Operations loop is gone); no golden changed.
- `Temporal.SharedTests` keeps the Temporal identity, source and metadata pins over
  `Temporal.Shared` alone; the Query-checker pins it carried over the Operations declarations
  duplicated `Umpire.Query`'s own tests and went with them.
- `TemporalModelTests` imports the roots that remain plus `Temporal.Tool.InspectTests` (which
  `TemporalExperimentalTests` used to carry) and pins no compatibility family: the four Nexus
  families are retired, `switch` stays pinned by `UmpireTests` until .7.
- `Temporal.Feature.Nexus.Success.Model` no longer imports `Race.Terminal`.
- `tools/umpire/internal/artifactv2/artifact_test.go` decodes the three Switch Plans only.

### Records

fn-79 (`## Scope`, 2026-09-20): the race Model's rows, outcomes and facts, the terminal slice, the
three Properties, the four Scenarios and their planner statuses, the baseline's match against the
first-generation lifecycle, and what each becomes on the Caller Model. fn-33 (`## Contracts`,
2026-09-20): the Space's base Query, Scenario and Limits, the two fault axes with their choices,
faults and coverage goals, the four pinned points and the reordering invariance, the Exploration's
policies, limits, pinned-candidate and session behavior, and the successor inputs
(`nexusCallerExploration` and its coverage golden). `HANDWRITTEN_INVENTORY.md` marks each row
deleted, dropped, re-pointed or kept with the date and the record that carries its behavior.
`model/README.md` and `.plans/UMPIRE4_COMPONENTS.md` no longer link the deleted docs or name the
old inspector scenarios.

### Gates

`lake build` green (586 targets, 633 before); `make umpire-gen-goldens && make umpire-check-goldens`
exit 0 with no golden changed; `make umpire-gen-regression-views && make umpire-check-regression-views`
exit 0 (the inspector's Switch view unchanged); inventory, retired vocabulary (its required-file
list no longer names the deleted root), conformance, protocol and authoring checks exit 0;
`make umpire-list` prints the nine-entry registry and `make umpire-explain
QUERY=temporal.nexus.caller.query.asyncCompletion` the Query's lineage; `LEAN_NUM_THREADS=1 make
lint-model` at the `.1` baseline (40 warnings, one fewer with the deleted modules, none new; the
same two generated `Proto.lean` errors; the MOD-10 exception unchanged);
`GOLANGCI_LINT_BASE_REV=8555116 make lint-code-fast` clean; `go test` over `tools/umpire` ok;
`make umpire-check-regression` exit 0 with 29 passing live identities. One finding: the inspector
now imports the caller Model, whose closure carries pre-existing warnings that `lake exe` replays
on stderr at every run, and the regression-view generator reads the inspector's stderr as its
diagnostic channel; the generator and the Makefile's inspector invocations run lake with
`--log-level=error` so only the inspector's own output reaches that channel.

## Evidence
- Commits: 7766383
- Tests: `lake build`; `make umpire-gen-goldens && make umpire-check-goldens`; `make umpire-gen-regression-views && make umpire-check-regression-views`; `make umpire-gen-inventory && make umpire-check-inventory`; `make umpire-check-retired-vocabulary`; `make umpire-gen-case-runtime-conformance && make umpire-check-case-runtime-conformance`; `make umpire-check-testpilot-protocol`; `make umpire-check-testpilot-authoring`; `LEAN_NUM_THREADS=1 make lint-model`; `make lint-code-fast`; `go test -count=1 -tags test_dep ./tools/umpire/...`; `make umpire-list`; `make umpire-explain QUERY=temporal.nexus.caller.query.asyncCompletion`; `CC=/usr/bin/cc TMPDIR=$(cd /tmp && pwd -P) make umpire-check-regression`
- PRs:
