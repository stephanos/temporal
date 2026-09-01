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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
