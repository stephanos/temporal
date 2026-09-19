---
satisfies: [R5]
---
# fn-17-bounded-semantic-exploration-and.8 Add the in-memory one-candidate session seam

## Description
Provide the minimal pure session interface that fn-33 needs to request and admit one candidate at a time.

**Size:** M
**Files:** `model/Umpire/Exploration/Session.lean`, `model/Umpire/Exploration/Tests/Session.lean`, `model/Umpire/Exploration.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Exploration/Session.lean, model/Umpire/Exploration/Tests/Session.lean, model/Umpire/Exploration.lean, model/UmpireTests.lean]

### Approach
- `beginSession` fixes one checked request and selected order; `next` returns at most one candidate.
- Require one exact opaque admission binding for the outstanding candidate before advancing.
- Reject missing, extra, duplicate, crossed, or stale admission without producing a successor.
- Keep the session process-local with no encoder, decoder, restart token, persisted format, or general reporting API.

### Investigation targets
**Required** (read before coding):
- Task `.5` integrated selection result.
- `model/Umpire/Artifact.lean` — ExperimentSpec identity.
- Parent spec `API Contracts` — exact retained session boundary.

## Acceptance
- [ ] At most one candidate is outstanding and advancing requires its exact checked admission binding.
- [ ] Every crossed/stale/cardinality failure is atomic and selection remains fixed.
- [ ] Focused session tests pass without persistence or runtime vocabulary.

## Done summary
Added the pure process-local `ExplorationSession`: `beginSession` fixes one checked exploratory order, `next` permits at most one outstanding candidate, and `observe` advances only for its exact singleton `ArtifactBinding`. Focused tests cover exact advancement plus atomic missing, extra, duplicate, crossed, and stale rejection without persistence, restart, runtime, or reporting APIs.

The task-owned Lean targets, aggregate model build, full model build, and model lint pass. The future fn-17.6 Nexus target remains absent exactly as in the pre-edit baseline, and no-fix `make lint-code` reproduces the inherited 1,385 Go findings without task-local Go changes. The Codex implementation review shipped on its first pass, so memory capture did not apply.

stage: impl-review - ran [completed 2026-09-02T20:30:54.407970Z]
## Evidence
- Commits: e26835988e51e9f02878219f9b6aa9dacf6919ca
- Tests: baseline: green (cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation), baseline: green (cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection), baseline: red (cd model && mise exec -- lake build Umpire.Exploration.Tests.Session; task-owned module absent pre-edit), INHERITED_BASELINE_RED: cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.ExplorationTests (future fn-17.6 module absent), baseline: green (cd model && mise exec -- lake build UmpireTests TemporalModelTests), GATE_SKIPPED:smoke:green-receipt 92d1d551 - baseline reused from prior post-gate pass, TDD_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Session (beginSession absent), cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation, cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection, cd model && mise exec -- lake build Umpire.Exploration.Tests.Session, INHERITED_RED: cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.ExplorationTests (future fn-17.6 module absent), cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-build-model, make lint-model, GOLANGCI_LINT_FIX=false make lint-code (inherited failure: 1385 pre-existing Go findings; no task-local Go paths)
- PRs: