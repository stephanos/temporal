---
satisfies: [R2]
---
# fn-207-tlc-backend-in-scala.2 Spike: in-process TLC versus a child JVM

## Description
Determine whether repeated in-process `tla2tools` runs are safe given TLC's static state and JVM-exit behaviour, or whether one child JVM per run is required; record findings and the chosen route in a design note.

**Files:** model/design/tlc-backend.md.

**Quick commands:** none (throwaway script in .flow/tmp).

## Acceptance
- Design note with evidence and the chosen route.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: