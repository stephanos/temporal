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

### Carried from fn-88.2 review (2026-09-27)
- Put `veilCommit` on the `veil` backend value rather than on the shared result/receipt shape, so a `reference` run cannot carry a Veil commit (illegal state unrepresentable). Adjust the receipt v2 field .2 introduced accordingly and pin both backends' receipts.

## Acceptance
- [ ] `model/lakefile.lean` requires Veil at the R1 commit; clean `make umpire-build-model` passes; manifest pin check present and tested to fail on a mismatch
- [ ] `Umpire.Search.Backend.Veil` returns each `BackendResult` variant on fixture products in `Tests/BackendVeil.lean`, with `complete` produced when the depth bound is exhausted
- [ ] R7 theorems present with axiom pins limited to `propext`, `Quot.sound`, `Classical.choice`
- [ ] `search-backend-isolation` enforced by `make lint-model` with a pinned controlled-violation diagnostic; the semantic-root rule still passes
- [ ] Cold CI build time recorded in task evidence and within job timeouts, or a `.lake` cache step added with before and after times
- [ ] Defer mode: closed as not applicable citing the R1 receipt identity, nothing added

## Done summary
`model/lakefile.lean` now requires Veil at `517f2bad`. `Umpire.Search.Backend.Veil` runs Veil's concrete checker over the product and returns a `BackendResult`. `search-backend-isolation` is enforced by `make lint-model`. `SearchBackend.veil` carries the Veil commit, and `SearchStats.veilCommit` is removed, as carried from the fn-88.2 review.

- **Veil pin (R2):** `lake update veil` added Veil and its package graph to the manifest; Batteries and protobuf keep their revisions. `make umpire-check-veil-pin` runs under `umpire-check-regression`. It checks that the lakefile's required commit, the manifest's resolved revision and `Veil.commit` agree. It then plants a manifest with another revision and requires the exact diagnostic. The check is a shell function, not a sub-make, so `make -n umpire-check-regression` lists it without running it. `tools/umpire/regression/ci_workflow_test.go` pins both calls. A real mismatched manifest fails the target.
- **Adapter:** `system` presents the product as a Veil `EnumerableTransitionSystem` over `Located` (a product state plus its depth).
  - Order: roots come in product order and successors in `Product.successors` key order. The depth bound (`maximumDepth`) is a Veil state constraint.
  - Driver: the backend stays a pure `Backend`, because Veil's `findReachable` is `IO` with an unfueled loop. It drives Veil's pure `SequentialSearchContext.bfsStep` at most `Limits.search` times over Veil's single-threaded FIFO. First-discovery parents make the first endpoint the shortest, lexicographically least one.
  - One Veil invariant ("not an admitted endpoint") records endpoints. Each one is read as `observeCandidate` reads a candidate: answers from the monitors, fired clauses from the bitset, and the form's stopping rule. Trigger evidence is one witness per clause, from the evaluator on the decoded trace. A trace is rebuilt only when it is reported.
  - Endings: `violationFound` (find, findViolation, pick), `complete` (the frontier emptied, including when the depth bound is exhausted; under verify the first violation is the counterexample), `stateBound` (`Limits.search` states visited with the frontier non-empty), and `invalid` (Property evaluation failed on a witness).
- **Tests:** `Umpire/Search/Tests/BackendVeil.lean` (registered in `Tests.lean`) pins:
  - each ending on fixture products;
  - depth exhaustion as `complete` on the parameterized Model (3, 5, 7, 7 states at depths 1 to 4);
  - outcomes equal to the reference on every form, budget and width of the Search fixture wherever the reference terminates;
  - equal witnesses, and equal validity dimensions on the endpoint cases;
  - one trigger witness per clause;
  - the v2 receipt's `searchBackend`, `searchUnit` and `veilCommit`.

  Mutation checks confirmed that the guards and the lint test catch a wrong ending and a disabled adapter rule.
- **R7:** `transition_equivalence`, `initial_equivalence` and `assumptions_equivalence` prove that Veil's relational reading of `system` is the product's own `relation` in both directions. Axioms are pinned to `propext`, `Classical.choice` and `Quot.sound`.
- **R3 lint:** `SearchBackendBoundary` and `checkSearchBackends` in `ModelLint/ImportGraph.lean` enforce the rule.
  - Allowed: only `Umpire.Search.Backend.Veil` may import `Veil.*`, and only `Umpire.Search.Selection` or a test may import the adapter.
  - Tests: `testSearchBackendIsolation` covers the rule. The controlled violation `Umpire.Search -> Veil.Core.Tools.ModelChecker.Concrete.Checker` is pinned in `lint-model`.
  - Unaffected: Veil stays external metadata with no `ModuleClass`, so the module index and the semantic-root rule need no change. The hand-written inventory needs no row, because no handwritten root is touched.

Deviations:
- **Exact dedup:** the fingerprint is the product `State`, not a 64-bit hash. The R22 hash-merge trust assumption therefore does not apply to this adapter; task .6's `AUTHORING.md` note should say so.
- **Entry point:** the backend drives Veil's pure `bfsStep`, not `findReachable`.
- **`Veil.backend?`:** it returns `Except ProductUnsupported BackendResult`. Selection (task .6) calls it.

R20: not met. No CI run was possible: this worker may not push, and origin's Actions API is blocked by SAML. Local measurements with a warm Batteries:
- `lake update veil`: 17 s.
- Cold checker closure (Aesop plus Veil, 361 jobs, including the npm widget build): 58 s wall, 911 MB peak RSS.
- Complete-mode `umpire-lint`: 36 s warm, 6.8 GB RSS.

Until task .6, no CI-built target imports Veil. CI currently only clones the extra packages. Once `Umpire.Search.Selection` lands, CI compiles Aesop and Veil and runs `npm` for Veil's widget. The ubuntu-24.04 runner ships Node, and `mise.toml` pins none. Carried to .6/.7: measure one cold CI run after .6, and add a `.lake` cache step to `.github/workflows/umpire.yml` if it does not fit the 30- and 40-minute timeouts. `umpire.yml` was left unchanged.

Gates:
- `make umpire-check-regression`: green (1186 s).
- `go test ./tools/umpire/regression/...`: green.
- `lint-model`:
  - `umpire-lint-tests`, the three controlled violations and complete-mode `umpire-lint`: passed.
  - builtin `lake lint` step: INCONCLUSIVE. Concurrent sessions' proto-schema rebuilds deleted `Testpilot/Carried.olean` mid-run.

Follow-ups:
- Duplicate roots, if a `SearchView` ever yields equal roots, would be queued twice by Veil's initial context. The reviewer rated this low confidence.
- Share the direct-import sort (done).
- Pin Node in `mise.toml` for developer machines.

stage: impl-review - ran [2026-09-27..2026-09-27] (claude opus high: SHIP, re-review SHIP after P3 fixes; codex: SHIP on the make -n fix; reviewed from e33c6be052 because another session's commit landed after the pre-edit base)
## Evidence
- Commits: ebccb6940a5849e2bf6a207dbb1c8bee07bb0970, e187d2ccfa4a5ef18a019cd7fc067fc27ebeef24, 231a7ea3131a1b5ccd1f216150c5ce66c152d2d8
- Tests: baseline: green (lake build Umpire.Search Umpire.Search.Product Umpire.Search.Tests Umpire.Search.VisibilityTests; make umpire-check-goldens), cd model && lake build Umpire.Search Umpire.Search.Product Umpire.Search.Backend.Veil Umpire.Search.Tests Umpire.Search.VisibilityTests, cd model && lake exe umpire-lint-tests (+ --controlled-search-backend-violation), cd model && lake exe umpire-lint (complete mode, 35.9 s warm, 6.8 GB RSS), LEAN_NUM_THREADS=1 make lint-model: inventory, umpire-lint-tests, three controlled violations and umpire-lint passed; the builtin-lint build step was INCONCLUSIVE (Testpilot/Carried.olean deleted mid-run by concurrent sessions' proto-schema rebuilds), go test -count=1 -tags test_dep ./tools/umpire/regression/..., make umpire-check-veil-pin (and a real mismatched manifest rejected), make umpire-check-regression (green, 1186 s; includes umpire-check-goldens and live tests)
- PRs: