---
satisfies: [R2]
---
# fn-206-apalache-backend-in-scala.2 Spike: Apalache as a Maven artifact and in-process run

## Description
Determine whether the full Apalache checker is published to Maven, whether `Tool.run` under `OutputManager.withScope` is usable in-process, and the JSON-RPC server alternative; record the findings and the chosen route in a design note.

**Files:** model/design/apalache-backend.md.

**Quick commands:** none (research and a throwaway scala-cli script in .flow/tmp).

## Acceptance
- Design note with evidence and the chosen route.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: