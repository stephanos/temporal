---
satisfies: [R7]
---
# fn-203-cases-are-produced-in-scala.7 Stop running the Go lowerer, explorer and realization admission in any gate

## Description
Remove every gate invocation of tools/umpire/lower, explore, realization, umpire-gen-cases and umpire-ir-bridge; leave the packages for fn-208 to delete.

**Files:** Makefile, model/framework/check/*.scala, .github or CI config if present.

**Quick commands:** `grep -n 'umpire-gen-cases\|umpire-ir-bridge' Makefile` shows only the Scala-backed targets.

## Acceptance
- No gate runs the Go producers.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: