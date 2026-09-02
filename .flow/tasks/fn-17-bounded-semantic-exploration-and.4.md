---
satisfies: [R3]
---
# fn-17-bounded-semantic-exploration-and.4 Prioritize one uncovered Model Coordinate

## Description
Implement the single retained semantic guidance policy over the immutable candidate universe.

**Size:** M
**Files:** `model/Umpire/Exploration/Guided.lean`, `model/Umpire/Exploration/Tests/Guided.lean`
**Touches:** [model/Umpire/Exploration/Guided.lean, model/Umpire/Exploration/Tests/Guided.lean]

### Approach
- Accept exactly one checked uncovered Model Coordinate and rank matching candidates before nonmatching candidates.
- Break every tie with ExperimentSpec semantic identity and apply the explicit exploration Limit.
- Return `coordinate-selected` or `coordinate-uncovered`; claim unreachable only when a separate exhaustive run completed.
- Prove observations cannot change the universe, coordinate, ordering, or scoring rule.

### Investigation targets
**Required** (read before coding):
- Task `.2` coordinate extraction and canonical universe.
- Task `.3` Limit and ordering behavior.
- `.plans/UMPIRE4_SPEC.md` — EXP-02 and PLN-04.

## Acceptance
- [ ] A matching uncovered coordinate deterministically changes the first eligible selection.
- [ ] Unknown coordinates reject and absent matches remain truthful without an unreachability claim.
- [ ] Focused guidance and reorder tests pass with no adaptive state.

## Done summary
Implemented deterministic bounded guidance for one checked uncovered Model Coordinate, with matching-first selection, ExperimentSpec semantic-identity ties, explicit Limit application, exact Space binding, and only coordinate-selected or coordinate-uncovered outcomes. Focused proofs cover changed first eligibility, tie order, absent and unknown coordinates, exhaustive-policy rejection, and the immutable no-observation selector boundary.

Verification: Guided, Selection, Validation, aggregate UmpireTests/TemporalModelTests, the full model build, and model lint pass. Session and Nexus Exploration targets remain absent future-task baseline failures; `make lint-code` reproduces the inherited 1,385-finding Go baseline unchanged.

stage: impl-review - ran [completed 2026-09-02T19:44:07.391542Z]
## Evidence
- Commits: fe7f99a97559199f462ef41fdb121c3c19509469
- Tests: baseline: green for implemented prerequisite targets; inherited red only for future Session and Nexus integration targets, TDD_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Guided (missing Guided module), TDD_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Guided (missing coordinate prioritization seam), TDD_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Guided (missing truthful guided outcome seam), cd model && mise exec -- lake build Umpire.Exploration.Tests.Guided, cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection, cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation, cd model && mise exec -- lake lint Umpire.Exploration.Guided Umpire.Exploration.Tests.Guided, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-build-model, make lint-model, INHERITED_BASELINE_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Session (future-task module absent), INHERITED_BASELINE_RED: cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.ExplorationTests (future-task module absent), INHERITED_BASELINE_RED: GOLANGCI_LINT_FIX=false make lint-code (1385 pre-existing Go findings)
- PRs: