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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
