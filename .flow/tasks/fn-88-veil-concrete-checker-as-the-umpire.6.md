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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
