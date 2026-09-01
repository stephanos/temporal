---
satisfies: [R1]
---
# fn-17-bounded-semantic-exploration-and.1 Define the bounded exploration request and coordinate vocabulary

## Description
Define the closed checked inputs shared by bounded exhaustive and uncovered-coordinate selection.

**Size:** M
**Files:** `model/Umpire/Exploration.lean`, `model/Umpire/Exploration/Core.lean`, `model/Umpire/Exploration/Language.lean`, `model/Umpire/Exploration/Tests/Validation.lean`
**Touches:** [model/Umpire/Exploration.lean, model/Umpire/Exploration/**, model/UmpireTests.lean]

### Approach
- Define one checked Space source, the `exhaustive | uncoveredCoordinate` policy, an explicit `experiment-specs` Limit, canonical Model Coordinates, pinned input, and structured failures.
- Reject empty/oversized bounds, an unknown guided coordinate, and incompatible pinned contracts before compilation or selection.
- Keep the types pure and free of runtime, persistence, campaign, promotion, or alternate-source fields.
- Preserve existing comments in every touched file.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Search.lean` — established Limit and strategy patterns.
- `model/Umpire/Space/Language.lean` — checked Space and coordinate vocabulary.
- `model/Umpire/Artifact.lean` — canonical ExperimentSpec identities.

## Acceptance
- [ ] The checked request exposes exactly the retained two policies and one finite Space source.
- [ ] Invalid Limits, coordinates, and pinned bindings fail with canonical typed errors.
- [ ] Focused validation tests pass and existing comments remain intact.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
