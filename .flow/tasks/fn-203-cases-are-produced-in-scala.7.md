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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
