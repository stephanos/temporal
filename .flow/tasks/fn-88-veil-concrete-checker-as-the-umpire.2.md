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
Added the backend seam inside `Umpire.Search`. The new public names are `BackendResult`, `Backend`, `Backend.reference`, `PlanningObservations` (now carrying `ExploredCounts` and `SearchStats`) and `finalizeBackendResult`. `search` is now `finalizeBackendResult` of `Backend.reference`, pinned by an `rfl` example, and `AdmittedQuery.search` calls the seam directly. `SearchStats` renames `backendPulls` to `enumeratorPulls` and gains `searchBackend`, `backendReason`, `searchUnit` and `veilCommit`. `canonicalPlanningReceiptJson` now takes the `PlanResult` and emits `umpire-planning-receipt/v2` with those fields; `veilCommit` appears only when it is set.

Equivalence: a scratch harness ran `search` over 480 fixture Queries (3 widths x 5 forms x 4 strategies x 4 budgets x completeness on/off) before and after the change. It printed the PlanningResult, the artifact and the seven counters, and the two outputs were byte-identical. `make umpire-check-goldens` passes with no golden diff. No receipt goldens exist, so the only receipt change is in `Umpire/Search/Tests/Endpoints.lean`: the existing guard now takes the run, and new guards pin the v2 format string, `searchBackend`, `backendReason`, `searchUnit`, `enumeratorPulls`, the absent and present `veilCommit`, and the `BackendReason` names. The Plan artifact codec, `artifactv2`, `boundWasHit` and `analyzeBranches` are unchanged.

Deviations from the spec:
- `BackendResult` has a fourth constructor, `invalid`. The reference traversal can end in a Property-evaluation failure, and dropping it would change behaviour.
- `Backend` is `CheckedQuery -> SearchView -> BackendResult`, because the Query already carries its Limits and strategy.
- The private enumerator names now say "enumerator" (`PlannerEnumerator`, `pureSearchEnumerator`), so "backend" keeps one meaning.
- `observeCandidate` builds its stop reason from a shared private `stopReason`.
- The kernel-replay slot is the unused `_kernel` argument of `finalizeBackendResult`, which task .6 fills.

Follow-up: the reviewer suggested moving `veilCommit` onto `SearchBackend.veil (commit)` so a reference run with a commit cannot be built. That would change the spec's data shape, so it is left to the maintainer.

Review base: the diff was reviewed as c97c5564ea..HEAD, because the fn-88.8 commits from another session landed between the pre-edit HEAD 7d0997b990 and this task's commit.

Baseline: green (the first `lake build` used the 4.33.1 lake on PATH and failed on Fixtures.lean:316; that was a toolchain mismatch, not a red baseline, and it was green under `mise exec -- lake`).

stage: impl-review - ran [2026-09-27..2026-09-27] (claude backend, SHIP on first round)
## Evidence
- Commits: ff7acfffedfc84957dc8c60d851236e54d9914d4
- Tests: cd model && mise exec -- lake build Umpire.Search Umpire.Search.Tests Umpire.Search.VisibilityTests, make umpire-check-goldens, make umpire-build-model, LEAN_NUM_THREADS=1 make lint-model, equivalence harness: 480 search runs byte-identical before/after (scratch EquivBefore/EquivAfter.lean), GATE_SKIPPED:build:not-applicable - Umpire.Search.Product and Umpire.Search.Selection do not exist until tasks .3 and .6
- PRs: