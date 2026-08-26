---
satisfies: [R2, R3, R8]
---
# fn-17-bounded-semantic-exploration-and.2 Build the atomic candidate universe and semantic signatures

## Description
Implement R2/R3's one authoritative bridge from a checked fn-16 space to a canonical coverage-scored candidate universe.

**Size:** M
**Files:** `model/Umpire/Exploration/Candidate.lean`, `model/Umpire/Exploration/Coverage.lean`, `model/Umpire/Exploration/Tests/Coverage.lean`, `model/Umpire/Exploration/Tests/Fixtures.lean`
**Touches:** [model/Umpire/Exploration/Candidate.lean, model/Umpire/Exploration/Coverage.lean, model/Umpire/Exploration/Tests/**]

### Approach
- Call fn-16 `compileBatch` with the caller's dependent base-target kernel and validate the full at-most-256 output before constructing `CandidateUniverse`.
- Recompute ExperimentSpec identities, reject duplicates, and digest canonical identity order independently of incoming list order.
- Project each artifact into exact intent/model/property coordinates, preserving the request-only fault label and evaluating Property through the existing pure semantics.
- Credit a goal at most once per distinct spec, retain property truth polarity, and expose goal-independent semantic coordinate coverage.
- Test raw-case-count divergence, repeated trace subjects, selected-fault intent versus target outcome, relation/observation distinction, and reordered candidates.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact.lean:36-80,228-382` — canonical artifact fields and identities
- `model/Umpire/Planning/Engine.lean:367-470` — selected model trace/artifact seam
- `model/Umpire/Property/Language.lean:1162-1228` — pure Property evaluation
- `model/Umpire/Space/Compilation.lean` — fn-16 batch contract after dependency lands
- `model/Umpire/Planning/Tests/Artifacts.lean:22-68` — inspectability and identity tests

### Acceptance
- [ ] Every candidate came through fn-16 atomic compilation with the exact supplied kernel; a batch failure yields no universe.
- [ ] Signature tests pin all coordinate kinds and one-credit-per-spec behavior.
- [ ] Duplicate/invalid artifacts fail instead of deduplicating, and source order cannot change universe bytes.
- [ ] Model coverage never claims runtime realization or conformance.

## Acceptance
- [ ] Candidate construction and coverage extraction implement R2/R3 exactly.
- [ ] The 256-point bound and separate per-Query/exploration units remain visible.
- [ ] Focused coverage tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
