---
satisfies: [R14]
---
# fn-88-veil-concrete-checker-as-the-umpire.3 Scenario progress automaton and the product state space

## Description
Build `Umpire.Search.Product`: the product of model state, Scenario progress, monitor states, and a fired-clause bitset, with transitions labeled by model `Step`, plus the version-one Scenario progress automaton whose final state decides `CheckedScenario.admits`. Monitor states are a parameter here; task .4 supplies them. Adopt mode only.

**Size:** M
**Files:** `model/Umpire/Search/Product.lean` (new), `model/Umpire/Search/Product/Scenario.lean` (new), `model/Umpire/Search/Tests/Product.lean` (new), `model/Umpire/Search/Tests.lean` (register the new test module)
**Touches:** [model/Umpire/Search/Product.lean, model/Umpire/Search/Product/Scenario.lean, model/Umpire/Search/Tests/Product.lean, model/Umpire/Search/Tests.lean]

### Approach
- Version-one progress state: per-action occurrence counts saturating one above the declared maximum, subsequence (`sequences`) progress indices, the exact-trace index for `traceExactly` and `actionsExactly`, and the fixed setup. `ordering` constraints (slot assignment depends on the set of remaining occurrences, `model/Umpire/Scenario/Check.lean:311-352`) and `adjacencies` (substring containment) return a typed `Unsupported` naming the construct.
- Successors come from `SearchView` (`model/Umpire/Search.lean:35-68`, `SearchView.ofCheckedQuery :392`) in table order so witness order is preserved downstream.
- Product state is `BEq` and `Hashable` and fully determines the future verdict; document that invariant in the module header.
- Decoding: a product path yields a `Scenario.Trace` with its setup and initial state.
- R14 evidence: a theorem that acceptance equals `admits` on completed traces for the supported constructs, or an exhaustive differential test enumerating every trace of every checked-in Scenario within its Limits; record which in the module header. Keep `admitsPrefix` (`:733`) as a pruning oracle in tests.
- Register `Tests/Product.lean` in `model/Umpire/Search/Tests.lean` so `lake build` compiles it and ModelLint sees it covered.

### Investigation targets
**Required:**
- `model/Umpire/Scenario/Check.lean:311-352, 710, 733` — slot assignment, `admits`, `admitsPrefix`
- `model/Umpire/Scenario.lean:212` — `CheckedScenario` fields
- `model/Umpire/Search.lean:792, 820, 849` — `maximumDepth`, `nextRoot?`, `pullCandidate` key order

**Optional:**
- `model/Temporal/Feature/Nexus/Pair/Model.lean:134-145` — a two-instance Scenario to test against

### Key context
- `steps` and `actions` both bound depth today (`maximumDepth = min`); keep that.
- This module imports `Umpire.Search`; `Umpire.Search` does not import it.

## Acceptance
- [ ] `Umpire.Search.Product` builds products from `SearchView`, a `CheckedScenario`, and a supplied monitor family, with `BEq`/`Hashable` product states and lossless trace decoding
- [ ] Progress automaton implemented for the version-one constructs; `ordering` and `adjacencies` return typed `Unsupported`
- [ ] R14 evidence present and its kind recorded
- [ ] New test module registered in `model/Umpire/Search/Tests.lean`; `make lint-model` passes; `Umpire.Search` closure unchanged
- [ ] Defer mode: closed as not applicable citing the R1 receipt identity, nothing added

## Done summary
Added `Umpire.Search.Product.Scenario`, which lowers a `CheckedScenario` to a progress automaton, and `Umpire.Search.Product`, the product of Model state, Scenario progress, monitor states (a `MonitorFamily` parameter; `MonitorFamily.empty` for none) and a fired-clause bitset, built from a `SearchView` in reference key order, with `BEq`/`Hashable` states, lossless `decode`, and kernel theorems (`run_progress`, `accepts_iff_admits`, `mem_successors`, `decode_lossless`; axioms propext, Quot.sound, Classical.choice, pinned). `Umpire.Search` does not import it.

R14 evidence is testing, recorded in the module header: `Umpire/Search/Tests/Product.lean` (registered in `Tests.lean`) compares `ScenarioAutomaton.admits` with `CheckedScenario.admits` exhaustively over 1023 synthetic traces for 20 Scenarios covering every construct (admitted counts pinned so the check is not vacuous), and over every trace of the Switch, Search-fixture and parameterized Models within and past their Limits, with `admitsPrefix` as a pruning oracle. A deliberately wrong acceptance was caught by it (then reverted).

Deviations:
- `ordering` and `adjacencies` lower when `actionsExactly`/`traceExactly` pins the schedule: each is then one Boolean computed at lowering by `admits`. Only a free schedule returns `Unsupported`. Every `scenario`-command Scenario carries an `ordering` (`Scenario.exactly`), so the literal reading would keep Caller and Pair off veil and contradict R9. The spec's R14/Boundaries text should be amended to say so (not in this task's Touches); carried into fn-88.9.
- Counters never store a count above the maximum: a step past it is a dead state (same prefixes `admitsPrefix` rejects), so they saturate at the maximum, or at the minimum when unbounded.
- The Temporal feature Scenarios are outside the Umpire test roots; `productAgrees` is public and fn-88.9 now carries applying it to Caller and Pair.

Follow-ups: `model/HANDWRITTEN_INVENTORY.md`'s Switch row lists "eleven importers"; `Umpire/Search/Tests/Product.lean` is a twelfth. `make lint-model` was red only on `Testpilot.Tests.Authoring`, broken by another session's uncommitted Testpilot/proto edits in the shared checkout.

stage: impl-review - ran [2026-09-27..2026-09-27] (claude backend, opus high; NEEDS_WORK then SHIP; reviewed 66edf3144b..HEAD because other sessions' commits landed after the pre-edit base)
## Evidence
- Commits: dd8ff632e9b460db2b6a4b18fa13e9da9dcd2e87, f6916d876416278443798ef9557425f25f5edd09
- Tests: baseline: green (mise exec -- lake build Umpire.Search Umpire.Search.Tests Umpire.Search.VisibilityTests), cd model && mise exec -- lake build Umpire.Search Umpire.Search.Product Umpire.Search.Tests Umpire.Search.VisibilityTests (green), make umpire-check-goldens (green), LEAN_NUM_THREADS=1 make lint-model: INCONCLUSIVE - red on Testpilot.Tests.Authoring from another session's uncommitted Testpilot/proto edits in the shared checkout; Batteries lint passed for Umpire.Lint (which imports UmpireTests, including the new modules), make umpire-check-regression: not run (spec final gate; shared tree carries foreign uncommitted Testpilot edits)
- PRs: