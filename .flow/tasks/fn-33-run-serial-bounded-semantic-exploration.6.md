---
satisfies: [R2, R6]
---

# fn-33-run-serial-bounded-semantic-exploration.6 Enforce the serial process-local coordinator boundary
## Description
Model process-local idle, preparing, running, observing, and finished states with one transition at a time. Bound candidate count, aggregate Case bytes/static work, Run time/work, event references, and report bytes; define stop/crash handling without durable recovery.

**Size:** M
**Touches:** `tools/umpire/campaign/session.go`, `tools/umpire/campaign/session_test.go`

## Acceptance
- [ ] Invalid ordering, concurrent work, duplicate result, and N+1 state fail closed.
- [ ] SIGINT performs bounded cleanup when possible; process loss records no fabricated Verdict or coverage.
- [ ] API-shape tests prove no lease, checkpoint, resume, adaptive selection, or persisted campaign format.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
