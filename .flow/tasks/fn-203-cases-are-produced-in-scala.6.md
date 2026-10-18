---
satisfies: [R6]
---
# fn-203-cases-are-produced-in-scala.6 Candidate enumeration, drop-prefix reduction and the NDJSON bridge server

## Description
Implement `Explore.candidates`, `Explore.reduce` and `Bridge.serve` over the typed `Variation`s and `Space`; honour the coordinator's frame limits and the outstanding-candidate rule; pass the exploration and replay bridge checks with the Go coordinator unchanged.

**Files:** model/framework/export/testpilot/Explore.scala, Bridge.scala; Makefile umpire-check-exploration-bridge / umpire-check-replay-bridge pointing at the Scala server.

**Quick commands:** `make umpire-check-exploration-bridge umpire-check-replay-bridge`

## Acceptance
- Both bridge checks pass against the Scala server.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
