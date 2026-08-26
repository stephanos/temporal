---
satisfies: [R2, R9, R10, R11]
---
# fn-17-bounded-semantic-exploration-and.8 Admit proof-bearing exact artifacts and deterministic protocol batches

## Description
Implement the closed `ExplorationSource/v1` exact-artifact branch and the fully specified bounded pure protocol used by downstream campaign adapters.

**Size:** M
**Files:** `model/Umpire/Exploration/Source.lean`, `model/Umpire/Exploration/Protocol.lean`, `model/Umpire/Exploration/Engine.lean`, `model/Umpire/Exploration/Tests/Source.lean`, `model/Umpire/Exploration/Tests/Protocol.lean`, `model/Umpire/Exploration/Tests/Engine.lean`, `model/Umpire/Exploration.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Exploration/Source.lean, model/Umpire/Exploration/Protocol.lean, model/Umpire/Exploration/Engine.lean, model/Umpire/Exploration/Tests/Source.lean, model/Umpire/Exploration/Tests/Protocol.lean, model/Umpire/Exploration/Tests/Engine.lean, model/Umpire/Exploration.lean, model/UmpireTests.lean]

### Approach
- Add the closed `space | exactCatalogArtifacts` source and generic proof-bearing `ExactCatalogArtifactCertificate/v1` / `CheckedExactCatalogArtifact` interface.
- Require the caller-owned certificate to bind an existing checked catalog subject and stable projection binding, the whole canonical ExperimentSpec, checked Query/model trace/property context, a proof that canonical compilation yields that exact artifact, and a recomputable coverage signature. Treat the projection path as metadata and perform no filesystem read.
- Admit one to 256 identity-sorted certificates, preserve whole ExperimentSpec bytes and semantic identities, and reject catalog/projection/context/compilation/signature drift, duplicate/noncanonical members, or any attempt to manufacture a Space, Query, artifact, or second catalog.
- Restrict exact sources in v1 to exhaustive selection, seed zero, no symmetry/t-wise goals, and a ceiling no larger than the admitted member count.
- Define domain-neutral `ExplorationObservation/v1` fields exactly: protocol and prior-state identities; candidate and ExperimentSpec identities; opaque checked admission identity; and reproduction-tuple digest. Do not name or import Result, evidence, Refinement, conformance, or Property-evaluation types.
- Implement the closed transition equations: initialize fixes selected order/cursor zero; `nextBatch(state)` accepts no size, requires `ready`, and emits/records exactly `min(8, remaining)`; observe requires exactly the outstanding candidates, canonicalizes to their selected order, inserts an identity-sorted observed-admission map, clears outstanding, and never changes selection or coverage.
- Define protocol status `ready | awaiting-observation | drained` separately from CoverageReport termination. Closed v1 contains no mutation language, adaptive corpus, priorities, runtime/evidence vocabulary or admission checking, leases, paths, publication, persistence, or Go coordination state.

### Investigation targets
**Required** (read before coding):
- fn-5 checked catalog and `CatalogProjectionBinding` implementation after dependency lands
- `model/Umpire/Artifact.lean` — canonical ExperimentSpec identity/content rules
- `model/Umpire/Exploration/Engine.lean` — source dispatch and transition integration
- `model/Umpire/Exploration/Tests/Engine.lean` — existing atomicity and determinism fixtures

## Acceptance
- [ ] Exact sources accept one to 256 checked in-memory certificates, preserve exact artifact bytes/identities, and never read a projection fixture path.
- [ ] Unsupported policy features, catalog/projection/context/compilation/signature drift, duplicate/noncanonical members, zero members, and N+1 members reject atomically.
- [ ] `nextBatch(state)` has no size argument, emits exactly `min(8, remaining)` from `ready`, and rejects awaiting/drained states; 1/8/9/16-member fixtures pin deterministic grouping and prove no batch exceeds eight.
- [ ] `observe` requires exactly one fully bound observation per outstanding candidate; missing/extra/duplicate/crossed/state-drift cases reject, while reordered completion yields the same canonical final state/report bytes.
- [ ] Observation only records identity-sorted opaque checked admission bindings and reproduction digests; coverage/selection remain unchanged and final drainage stays distinct from semantic selection termination.
- [ ] Source, protocol, engine integration, public exports, and focused/aggregate Umpire tests cover all positive and negative rows.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
