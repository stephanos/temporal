---
satisfies: [R2, R3, R7, R20]
---
# fn-88-veil-concrete-checker-as-the-umpire.5 Veil dependency, adapter with equivalence theorems, and the isolation lint rule

## Description
Add the pinned `require` to `model/lakefile.lean`, write `Umpire.Search.Backend.Veil` over the product with the R7 theorems, add the `search-backend-isolation` direct-import rule to ModelLint, and measure the cold CI build. Depends on the GOV-02 amendment drafts (task .8) so no rule text contradicts the dependency when it lands. Adopt mode only.

**Size:** M
**Files:** `model/lakefile.lean`, `model/lake-manifest.json`, `model/Umpire/Search/Backend/Veil.lean` (new), `model/Umpire/Search/Tests/BackendVeil.lean` (new), `model/Umpire/Search/Tests.lean` (register), `model/ModelLint/ImportGraph.lean`, `model/ModelLint/ImportGraphTests.lean`, `model/ModelLint/ModuleIndex.lean`, `model/HANDWRITTEN_INVENTORY.md`, `Makefile` (manifest pin check under `umpire-check-regression`; controlled violation under `lint-model`), `.github/workflows/umpire.yml` only if R20 needs a `.lake` cache
**Touches:** [model/lakefile.lean, model/lake-manifest.json, model/Umpire/Search/Backend/Veil.lean, model/Umpire/Search/Tests/BackendVeil.lean, model/Umpire/Search/Tests.lean, model/ModelLint/**, model/HANDWRITTEN_INVENTORY.md, Makefile, .github/workflows/umpire.yml]

### Approach
- `require` at exactly the R1 commit; add a `make umpire-check-regression` step that fails when `lake-manifest.json` resolves another Veil revision (no such check exists today).
- Adapter: wrap the product as `Veil.EnumerableTransitionSystem`, supply successors in table order, translate `Limits` to depth and state bounds, run the concrete checker single-threaded with a deterministic frontier, and decode into `BackendResult`: frontier exhausted within `maximumDepth` → `complete` with observations; a violation → `violationFound` with the decoded `Scenario.Trace`; state budget hit → `stateBound`. For `verify`, continue to the frontier's end after a violation so the reported witness is the shortest. Build `PlanningObservations` from the product's fired-clause bitset and endpoint answers.
- Restate the probe's theorems (`experiments/umpire-dsl/veil/Main.lean:9-41`) over the product; pin axioms.
- Lint: follow `checkAuthoringPath` (`model/ModelLint/ImportGraph.lean:323`) and `testAuthoringPathIsolation` (`ImportGraphTests.lean:629`); allowed importers are exactly `Umpire.Search.Backend.Veil` for `Veil.*` and `Umpire.Search.Selection` for the adapter; add the `ModuleClass`/external decision for `Veil.*` in the module index policy; pin a controlled violation in `lint-model` (`Makefile:997-1020`); add the new modules to `HANDWRITTEN_INVENTORY.md` if the inventory check requires it.
- R20: run the umpire workflow once from a cold `.lake` and record the cold build time against the 30 and 40 minute job timeouts; if it does not fit, add a `.lake` cache step in this task and record both numbers.

### Investigation targets
**Required:**
- The R1 receipt in `experiments/umpire-dsl/VEIL_RESULTS.md` — commit, module names, exposure, threading facts
- `experiments/umpire-dsl/veil/Main.lean`
- `model/ModelLint/ImportGraph.lean:68-110, 182-202, 323-357` — rule shape, labels, direct-import pattern, semantic-root check
- `model/lakefile.lean`, `model/lake-manifest.json`, `Makefile:478, 829, 997-1020`

**Optional:**
- `.github/workflows/umpire.yml` — job timeouts and `cache: false`

### Key context
- `Umpire.Search.Selection` does not exist until task .6; until then the adapter is reachable only from its own tests, which the lint rule must allow as test modules.
- Veil-only tests live in their own module so the rollback drill (task .7) can delete them as a unit.

## Acceptance
- [ ] `model/lakefile.lean` requires Veil at the R1 commit; clean `make umpire-build-model` passes; manifest pin check present and tested to fail on a mismatch
- [ ] `Umpire.Search.Backend.Veil` returns each `BackendResult` variant on fixture products in `Tests/BackendVeil.lean`, with `complete` produced when the depth bound is exhausted
- [ ] R7 theorems present with axiom pins limited to `propext`, `Quot.sound`, `Classical.choice`
- [ ] `search-backend-isolation` enforced by `make lint-model` with a pinned controlled-violation diagnostic; the semantic-root rule still passes
- [ ] Cold CI build time recorded in task evidence and within job timeouts, or a `.lake` cache step added with before and after times
- [ ] Defer mode: closed as not applicable citing the R1 receipt identity, nothing added

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
