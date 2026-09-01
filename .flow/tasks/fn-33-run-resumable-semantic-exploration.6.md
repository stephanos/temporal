---
satisfies: [R2, R6]
---
# fn-33-run-resumable-semantic-exploration.6 Enforce the one-active-candidate coordinator boundary

## Description
Make the serial process-local coordinator invariant explicit and testable before runner integration.

**Size:** M
**Files:** `tools/umpire/campaign/session.go`, `tools/umpire/campaign/session_test.go`, `model/Temporal/Tool/ExplorationBridgeTests.lean`
**Touches:** [tools/umpire/campaign/session.go, tools/umpire/campaign/session_test.go, model/Temporal/Tool/ExplorationBridgeTests.lean]

### Approach
- Model only `idle`, `candidate-active`, and `finished` process-local states with one allowed transition at a time.
- Reject another `next` while a candidate is active and reject observation before cleanup plus Result admission are complete.
- Discard the session on interruption and return the corresponding terminal tooling outcome.
- Add API-shape guards for one process-local session and no persisted coordinator format.

### Investigation targets
**Required** (read before coding):
- Task `.1` bridge states and bindings.
- Existing fn-19 cleanup lifecycle.
- Parent spec `Boundaries`.

## Acceptance
- [ ] The coordinator permits only one active candidate and one admitted Result transition.
- [ ] Invalid ordering and interruption fail closed without semantic coverage credit.
- [ ] Focused state-machine and API-shape tests pass with no persisted format.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
