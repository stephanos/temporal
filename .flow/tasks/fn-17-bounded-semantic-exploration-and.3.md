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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
