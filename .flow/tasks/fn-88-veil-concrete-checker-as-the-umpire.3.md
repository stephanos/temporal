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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
