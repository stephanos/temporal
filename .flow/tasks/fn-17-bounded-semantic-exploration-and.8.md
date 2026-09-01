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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
