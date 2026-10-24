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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: