---
satisfies: [R4, R12, R17]
---
# fn-88-veil-concrete-checker-as-the-umpire.2 Backend seam inside Umpire.Search, frozen reference, receipt v2

## Description
Add the backend seam inside `Search.lean`, where the traversal, `finalizePlanning`, `observeCandidate`, and the `PlanningResult` constructor are private: a public `BackendResult`, a public `Backend.reference` returning it, a public `PlanningObservations` carrying the observations plus `ExploredCounts` and the `SearchStats` counters, and a public `finalizeBackendResult` that owns the observation aggregate, the finalization, and the post-pass. Rename `backendPulls` to `enumeratorPulls`, add the four backend fields to `SearchStats` and the receipt at v2, and pin the reference. Adopt mode only.

**Size:** M
**Files:** `model/Umpire/Search.lean`, `model/Umpire/Search/Admission.lean` (call `finalizeBackendResult` with `Backend.reference`), `model/Umpire/Search/VisibilityTests.lean`, `model/Umpire/Search/Tests/*.lean`, receipt goldens under `model/Umpire/Search/Tests/Fixtures/` and the `umpire-goldens` directories
**Touches:** [model/Umpire/Search.lean, model/Umpire/Search/Admission.lean, model/Umpire/Search/VisibilityTests.lean, model/Umpire/Search/Tests/**, model/Umpire/Search/Tests/Fixtures/**]

### Approach
- `BackendResult` per the spec's Data shapes: `violationFound (trace : Scenario.Trace) (observations)`, `complete (observations)`, `stateBound (visited) (observations)`; no `depthBound`, no `cancelled`. Make `PlanningObservations` (`model/Umpire/Search.lean:935`) public so an adapter in another module can build it.
- `Backend.reference`: today's `pullCandidate`/`traverseLoop`/`observeCandidate` path (`:849`, `:1056`, `:948`) unchanged, returning the observations it already accumulates; depth exhaustion stays `complete`.
- `finalizeBackendResult : CheckedQuery → SearchView → BackendResult → Except KnownGapError PlanResult`: the kernel-replay hook (a no-op for reference in this task, filled by task .6), then `finalizePlanning` (`:738`) and the `stillPending`/`neverTriggered` post-pass (`:1119-1148`). `search` becomes `finalizeBackendResult` of `Backend.reference`; `searchWithPlanRequest` follows.
- `SearchStats` (`:493`): rename `backendPulls` to `enumeratorPulls`; add `searchBackend`, `backendReason`, `searchUnit`, `veilCommit`. Receipt (`:628`) format string to `umpire-planning-receipt/v2`. Plan artifact codec (`model/Umpire/Artifact/Codecs.lean:77,157`) untouched.
- Equivalence harness: `make umpire-check-goldens` and the Search suite before and after; the only diffs are the receipt format string, the renamed key, and the new fields. List them in task evidence.
- Pins: extend the `#check` surface pin in `VisibilityTests.lean` with the new public names. The `Search.lean` line-count pin is set by task .6, the last task that edits the file.
- `analyzeBranches` (`model/Umpire/Search/Branches.lean:627`) keeps using `traverseBoundedCandidates`; do not touch it.

### Investigation targets
**Required:**
- `model/Umpire/Search.lean:35-68, 493-501, 595-673, 738-757, 935-1000, 1110-1155`
- `model/Umpire/Search/Admission.lean:137-186`
- `model/Umpire/Command/Authoring.lean:594,810` — `checkAdmitted` and `boundWasHit` read `explored.traces`

**Optional:**
- `model/Umpire/Search/Tests/Fixtures.lean` — `incrementalKernel` fixtures

### Key context
- `Umpire.Search` is a ModelLint semantic root whose closure may not reach `Lean.Elab.Term`; add no import to it.
- Do not name anything `Engine` (retired by fn-82).

## Acceptance
- [ ] `BackendResult`, `Backend.reference`, `PlanningObservations`, and `finalizeBackendResult` are public in `Umpire.Search`; `search` equals `finalizeBackendResult` of `Backend.reference`
- [ ] Every golden and fixture byte-identical except the receipt format string, `enumeratorPulls`, and the four new fields; the changed list is in task evidence
- [ ] Plan artifact codec and `tools/umpire/internal/artifactv2` unchanged; `boundWasHit` unchanged
- [ ] Visibility `#check` pin added and passing
- [ ] `LEAN_NUM_THREADS=1 make lint-model` passes with no new semantic-root finding
- [ ] Defer mode: closed as not applicable citing the R1 receipt identity, nothing added

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
