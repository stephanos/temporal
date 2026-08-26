---
satisfies: [R1, R3, R8]
---
# fn-17-bounded-semantic-exploration-and.1 Define checked exploration requests and semantic coverage vocabulary

## Description
Build the closed reusable request/configuration layer for R1/R3/R8 and remove the false per-Query coverage-guided name without adding compatibility aliases.

**Size:** M
**Files:** `model/Umpire/Search.lean`, `model/Umpire/Planning/Engine.lean`, `model/Umpire/Exploration.lean`, `model/Umpire/Exploration/Core.lean`, `model/Umpire/Exploration/Language.lean`, `model/Umpire/Exploration/Tests/Fixtures.lean`, `model/Umpire/Exploration/Tests/Validation.lean`
**Touches:** [model/Umpire/Search.lean, model/Umpire/Planning/Engine.lean, model/Umpire/Exploration.lean, model/Umpire/Exploration/**, model/UmpireTests.lean]

### Approach
- Add the separate five-variant Exploration strategy/policy, exact selection/t-strength bounds, typed units, canonical names, and structured errors.
- Rename `SearchStrategy.coverageGuided` to `seeded`, update exhaustive matches/tests/comments, and reject the old text spelling at command boundaries. Explicitly prove both seeded setup rotation and seeded action/outcome indexing remain stable.
- Define closed semantic coordinate, coverage signature, goal-credit, pinned input, termination, state, report, and result types behind a small facade.
- Implement canonical checking/digests for requests, policies, pinned inputs, and optional prior-state headers; do not implement selectors yet.
- Preserve existing comments in every touched file.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Search.lean:5-76` — current per-Query policy and bounds
- `model/Umpire/Planning/Engine.lean:210-230` — seed-rotation behavior being renamed
- `model/Umpire/Core.lean:28-109` — semantic declaration/trace vocabulary
- `model/Umpire/Space/Language.lean` — fn-16 checked-space and coverage-goal API after dependency lands
- `model/Umpire/Property/Language.lean:1162-1228` — pure evaluation result

### Acceptance
- [ ] Invalid strategy parameters, budgets, t strengths, duplicate/case-colliding pinned IDs, and malformed state headers return canonical typed failures.
- [ ] Query-level `seeded` retains the prior deterministic behavior and no API calls it coverage-guided.
- [ ] Exploration types contain no Temporal, runtime, evidence, IO, persisted-reader, or promotion field.
- [ ] Existing comments remain intact.

## Acceptance
- [ ] The checked vocabulary implements the parent bounds, identities, and closed strategy contract.
- [ ] Existing Query planning stays deterministic under the accurately named seeded policy.
- [ ] Focused validation and exhaustive-match tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
