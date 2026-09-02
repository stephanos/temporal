---
satisfies: [R1]
---
# fn-17-bounded-semantic-exploration-and.2 Build the canonical finite candidate universe

## Description
Compile one checked fn-16 Space atomically into the canonical at-most-256 candidate universe.

**Size:** M
**Files:** `model/Umpire/Exploration/Candidate.lean`, `model/Umpire/Exploration/Coverage.lean`, `model/Umpire/Exploration/Tests/Candidate.lean`
**Touches:** [model/Umpire/Exploration/Candidate.lean, model/Umpire/Exploration/Coverage.lean, model/Umpire/Exploration/Tests/**]

### Approach
- Delegate to fn-16 `compileBatch` with the caller's exact target kernel and reject the whole build on any point failure.
- Preserve canonical ExperimentSpec bytes, recompute identities, reject duplicates, and order independently of source order.
- Extract only Model Coordinates already present in the checked trace, keeping requested faults labeled as intent.
- Test N/N+1, invalid artifact, duplicate identity, and reordered-input cases.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Space/Compilation.lean` — atomic finite compilation.
- `model/Umpire/Artifact.lean` — canonical artifact fields and identities.
- `model/Umpire/Planning/Engine.lean` — selected trace and exact kernel seam.

## Acceptance
- [ ] Every candidate comes from one atomic checked Space compilation with the exact kernel.
- [ ] Invalid, duplicate, empty, or oversized universes produce no partial value.
- [ ] Coordinate extraction and canonical-order tests pass without runtime claims.

## Done summary
Built the canonical finite CandidateUniverse from one checked Experiment Space through the caller's exact planner kernel, retaining recomputed Artifact identities, canonical bytes, Model Trace coordinates, and explicitly labeled fault intent. The universe now rejects compilation failures, malformed/incomplete artifacts, duplicates, empty/oversized results, and count drift atomically, with its alternate constructor paths sealed.

Verification: focused Candidate and Validation targets, aggregate UmpireTests/TemporalModelTests, full model build, and model lint pass. The Selection, Session, and Nexus Exploration targets remain absent future-task baseline failures; `make lint-code` reproduces the inherited 1,385-finding Go baseline unchanged.

Memory capture after the review fix was skipped because Flow memory is not initialized.

stage: impl-review - ran [2026-09-02T18:50:29Z..2026-09-02T19:02:36.681286Z]
## Evidence
- Commits: bc1df2ad44e31c5986089fe8424b3fcf89d627c9, b7921cdc6982cdf7a33c028b90ed8485289b5831
- Tests: cd model && mise exec -- lake build Umpire.Exploration.Tests.Candidate, cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-build-model, make lint-model, INHERITED_BASELINE_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection (future-task module absent), INHERITED_BASELINE_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Session (future-task module absent), INHERITED_BASELINE_RED: cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.ExplorationTests (future-task module absent), INHERITED_BASELINE_RED: GOLANGCI_LINT_FIX=false make lint-code (1385 pre-existing Go findings)
- PRs: