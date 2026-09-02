---
satisfies: [R2]
---
# fn-17-bounded-semantic-exploration-and.3 Implement bounded exhaustive enumeration

## Description
Implement the deterministic exhaustive selector and prove its finite Limit semantics.

**Size:** M
**Files:** `model/Umpire/Exploration/Selection.lean`, `model/Umpire/Exploration/Tests/Selection.lean`
**Touches:** [model/Umpire/Exploration/Selection.lean, model/Umpire/Exploration/Tests/Selection.lean]

### Approach
- Walk non-pinned candidates in semantic-identity order and stop only at universe end or the explicit exploration Limit.
- Return `exhausted` only when every candidate was considered; otherwise return `limit-reached` without an absence claim.
- Prove input reorder stability and every relevant zero/N/N+1 boundary on small finite fixtures.
- Keep planner candidate Limits distinct from the exploration Limit.

### Investigation targets
**Required** (read before coding):
- Task `.2` candidate universe and fixtures.
- `model/Umpire/Search.lean` — typed Limit precedent.
- Parent spec `Selection Algorithms` — exact retained ordering and outcomes.

## Acceptance
- [ ] Canonical finite fixtures enumerate every candidate in identity order within their Limit.
- [ ] Limit exhaustion is never reported as complete universe exhaustion.
- [ ] Reordered inputs produce byte-identical selected identities and outcomes.

## Done summary
Implemented deterministic bounded exhaustive selection over the canonical finite candidate universe, with non-pinned Limit accounting, exact Space fingerprint binding, stable identity order, and truthful exhausted versus limit-reached outcomes. Focused proofs cover zero/N-1/N/N+1 boundaries, pinned overlap, same-ID crossed Spaces, and byte-identical authored-input reorder stability.

Verification: Selection, Candidate, Validation, aggregate UmpireTests/TemporalModelTests, the full model build, and model lint pass. Session and Nexus Exploration targets remain absent future-task baseline failures; `make lint-code` reproduces the inherited 1,385-finding Go baseline unchanged.

Memory capture after the review fix was skipped because Flow memory is not initialized.

stage: impl-review - ran [2026-09-02T19:21:03.376238Z..2026-09-02T19:26:43.637667Z]
## Evidence
- Commits: d82d3b16e860e045360dd74e147820f74e6dcea4, cdf7c633482d81ba661fa2b6fe8830761b839aff
- Tests: TDD_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection (missing selector module), TDD_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection (pinned-overlap and same-ID crossed-Space proofs failed before fix), cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection, cd model && mise exec -- lake build Umpire.Exploration.Tests.Candidate, cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-build-model, make lint-model, INHERITED_BASELINE_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Session (future-task module absent), INHERITED_BASELINE_RED: cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.ExplorationTests (future-task module absent), INHERITED_BASELINE_RED: GOLANGCI_LINT_FIX=false make lint-code (1385 pre-existing Go findings)
- PRs: