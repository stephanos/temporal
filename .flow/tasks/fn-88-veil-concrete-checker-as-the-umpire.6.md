---
satisfies: [R10, R12, R16]
---
# fn-88-veil-concrete-checker-as-the-umpire.6 Backend selection module and the kernel replay gate

## Description
Add `Umpire.Search.Selection`, the one function that chooses `veil` or `reference` for an admitted Query and records the reason, wire `AdmittedQuery.search` through it, add the test-visible `AdmittedQuery.searchWith`, and fill the kernel-replay hook in `finalizeBackendResult` with the full admission and verdict decision. Adopt mode only.

**Size:** M
**Files:** `model/Umpire/Search/Selection.lean` (new), `model/Umpire/Search/Admission.lean`, `model/Umpire/Search.lean` (replay inside `finalizeBackendResult`), `model/Umpire/Query.lean` (`QueryErrorKind.unreplayableWitness`), `model/Umpire/Search/Tests/Replay.lean` (new), `model/Umpire/Search/Tests.lean` (register)
**Touches:** [model/Umpire/Search/Selection.lean, model/Umpire/Search/Admission.lean, model/Umpire/Search.lean, model/Umpire/Query.lean, model/Umpire/Search/Tests/Replay.lean, model/Umpire/Search/Tests.lean]

### Approach
- Selection: `veil` when every clause lowers (task .4), the Scenario lowers (task .3), the strategy is not `seeded`, and the form is one of the four; otherwise `reference` with `backendReason` set to the matching `unsupported-*` value. `AdmittedQuery.search` (`model/Umpire/Search/Admission.lean:137-145`) calls it; `AdmittedQuery.searchWith (backend)` runs a named backend for tests.
- Kernel replay inside `finalizeBackendResult`: accept a decoded `Scenario.Trace` only when every step is a `SearchView` member from a proven initial state and the same decision `observeCandidate` makes today holds: `behavior.admits`, the `isTerminal` condition for `ending = terminal` (`model/Umpire/Search.lean:1080-1083`), the endpoint answer with the partial flag, and the `find`/`pick` coverage condition (`:965`). Expose that decision as one public function used by both the reference path and replay so they cannot drift.
- `QueryErrorKind.unreplayableWitness` (`model/Umpire/Query.lean:246`); render the trace and the diagnostic into `QueryError.offendingValue` (`:281`).
- Negative control: a faulty backend fixture that returns a trace failing admission yields `invalid` with that kind; pin it.
- Set the R12 line-count `#guard` on `Search.lean` after this task's edits, since this is the last task that edits the file.
- Keep `finalizePlanning`'s completeness-evidence gate and the admitted-endpoint rule for `unsatisfiable` unchanged for both backends (R16).

### Investigation targets
**Required:**
- `model/Umpire/Search/Admission.lean:126-186`
- `model/Umpire/Search.lean:738-757, 948-1000, 1080-1083, 1110-1151`
- `model/Umpire/Query.lean:9-20, 76-79, 246-283`

**Optional:**
- `model/Umpire/Search/Tests/Fixtures.lean`

### Key context
- `Umpire.Search.Selection` is the only importer of `Umpire.Search.Backend.Veil`; the lint rule from task .5 says so.
- This module plus the adapter are the closed set the rollback drill (task .7) reduces.

## Acceptance
- [ ] `Umpire.Search.Selection` exists; `AdmittedQuery.search` selects through it; `AdmittedQuery.searchWith` runs a named backend
- [ ] Every ineligible Query routes to `reference` with the matching reason, covered by tests for each reason value
- [ ] Kernel replay implemented as the shared admission-and-verdict decision; the faulty-backend negative control yields `invalid` with `QueryErrorKind.unreplayableWitness`
- [ ] Reference results unchanged: search goldens byte-identical to task .2's state
- [ ] `Search.lean` line-count pin set and passing
- [ ] `make lint-model` passes with the isolation rule
- [ ] Defer mode: closed as not applicable citing the R1 receipt identity, nothing added

## Done summary
`Umpire.Search.Selection` now chooses `veil` or `reference` for each Query and records the reason. Kernel replay inside `finalizeBackendResult` accepts a backend's witness only when it is a trace the Query's own search could report.

- **Selection:** `select` returns `Choice.veil` with the Query's product, or `Choice.reference` with a reason. It checks, in order, the strategy (`unsupported-strategy:seeded`), the form, and the product build (`unsupported-clause:<kind>`, then `unsupported-scenario:<construct>`).
  - `AdmittedQuery.search` and `searchWithIntent` both select through it. `searchWithIntent` goes through the new `projectPlanRequest`, which `searchWithPlanRequest` now also uses.
  - `AdmittedQuery.searchWith (backend)` and `Selection.searchWith` run a named backend. On an ineligible Query, `.veil` falls back to `reference` and records the reason.
  - `unsupported-form` has no Query that reaches it: `Query.Form` has exactly the four supported forms. `formReason` is an exhaustive match, so adding a form forces a decision.
- **Cutover hold:** `Selection.cutover := false`. While it is false, a Query `select` sends to `veil` runs on `reference` with reason `default`, so the goldens stay byte-identical. Only the receipt's `backendReason` changes, and only for ineligible Queries; receipts are not goldens. `SwitchCompiledArtifact.json` would flip under `veil` (its `explored` counts). **Task .10 must flip `cutover` to `true` in the same commit as the R18 re-pin. `Umpire/Search/Selection.lean` is not in .10's Touches, so plan-sync should add it.** The .7 rollback drill reduces the file to always choosing `reference`.
- **Kernel replay (R10):** replay checks that:
  - the setup is one of the search's candidate setups;
  - the initial state and every step are search-view members (∃ index below the limit);
  - the trace is within `maximumDepth`;
  - the shared endpoint decision holds.

  `isAdmittedEndpoint` (used by `traverseLoop`) and `endpointDecision`/`EndpointDecision` (whose evaluation half `observeCandidate` uses) are the one decision that the reference and replay share. A failing witness, whether the stopping trace or a `verify` counterexample, yields `invalid` with `QueryErrorKind.unreplayableWitness`, and `offendingValue` is `<diagnostic>: <rendered trace>`. The rejection wins over `finalizePlanning`'s unsatisfiable-Scenario priority; that was the review's finding, fixed in df1a7c927f. `finalizePlanning`'s completeness and admitted-endpoint gates are unchanged (R16).
- **Tests:** `Umpire/Search/Tests/Replay.lean`, registered in `Tests.lean`, pins:
  - each selection reason, `searchWith` on both backends, the cutover hold, and admitted selection on the Switch;
  - seven faulty-backend negative controls (bare root, non-member step result, foreign action, past the depth bound, wrong initial state, wrong setup, the full offending value), plus the find-violation and verify decision failures, a real counterexample that replays, and the unsatisfiable-Scenario case;
  - the new public surface, and that `replayFailure` stays private;
  - the R12 line count of `Search.lean`, 1428, read via `IO.FS.lines` beside the test file.

  Mutation checks: disabling replay fails 5 guards, and dropping the rejection override fails the unsatisfiable guard.

Deviations:
- `model/Umpire/Query/Elab.lean` is outside Touches. It got a one-token edit: `.unreplayableWitness` joins the `.parent` arm of `selectRoleFallback`'s exhaustive match, without which the new constructor does not compile.
- The R12 `#check` pins for the new names live in `Tests/Replay.lean`, not `VisibilityTests.lean`, which is outside Touches.
- The `Selection` inductive is named `Choice` (c49af502d5). `Selection.Selection.run` tripped `linter.extra.dupNamespace`.

R20 for .7: `Umpire.Search.Admission` now imports Selection, and so the adapter. Every CI Lean job that builds the model (`umpire-build-model`, `lint-model`, `umpire-check-regression`) now compiles Aesop, Veil and Veil's npm widget from a cold `.lake`. Measure one cold CI run of `.github/workflows/umpire.yml` against its 30- and 40-minute timeouts. Locally, fn-88.5 measured 58 s wall and 911 MB peak RSS for the checker closure (warm Batteries). If CI does not fit, add a `.lake` cache step, and pin Node if the runner's default fails the widget build.

Follow-ups:
- .10: flip `Selection.cutover` in the re-pin commit (see above).
- .9: `searchWith .veil` is the differential test's `veil` arm. It falls back to `reference` on ineligible Queries, and `backendReason` shows the fallback.
- .7 `AUTHORING.md`: a `veil` absence answer rests on the adapter theorems and the differential test, as fn-88.5 recorded. Every `veil` witness now passes replay.

Defer mode: not applicable (R22 adopt).

stage: impl-review - ran [2026-09-27..2026-09-27] (codex fan-out: 3 draws NEEDS_WORK on one shared P2, the unsatisfiable Scenario masking a replay rejection; fixed in df1a7c927f, re-review SHIP; re-review SHIP again after the c49af502d5 lint rename)
## Evidence
- Commits: 9e5dc6c54c9b2be456643277eeb4efe30bc6e38a, df1a7c927f5dd97649a58dc0bcfad0573676ef93, c49af502d507660e4eb67fe49dd1217dcde3485c
- Tests: baseline: green (fn-88.5's make umpire-check-regression; no model/, Makefile or tools/ change between 231a7ea313 and base 46cb9b1320b1549b02416b53845b70a37cd1cfeb), cd model && lake build Umpire.Search Umpire.Search.Product Umpire.Search.Selection Umpire.Search.Tests Umpire.Search.VisibilityTests (green; --wfail at c49af502d5), make umpire-check-goldens (green at 9e5dc6c54c and c49af502d5; byte-identical), make umpire-check-regression (green at df1a7c927f, receipt .flow/tmp/green-receipts/df1a7c92-regression.json; c49af502d5 is a rename only), LEAN_NUM_THREADS=1 make lint-model: umpire-lint-tests, the three controlled violations and complete-mode umpire-lint passed; the builtin lake lint step INCONCLUSIVE (concurrent sessions' Testpilot/Temporal rebuilds deleted .olean files mid-run); its one real finding (dupNamespace on Selection.Selection.run) fixed in c49af502d5; lake --wfail lint --builtin-only over Umpire.Search, Selection, Admission, Query, Query.Elab, Search.Tests, Tests.Replay: clean, mutation: disabling replay in finalizeBackendResult fails 5 Replay guards; dropping the rejection override fails the unsatisfiable-Scenario guard
- PRs: