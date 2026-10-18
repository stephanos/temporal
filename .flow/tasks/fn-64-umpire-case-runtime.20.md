---
satisfies: [R5]
---
# fn-64-umpire-case-runtime.20 Record Host-close failure in cleanup outcome

## Description
Resolve the verified completion-review close-only failure: session.Close errors or cooperative timeout must set cleanup failed and link the host_close_failed diagnostic while preserving fixed Run disposition and Verdict.

## Acceptance
Add regression tests for close-only error and cooperative timeout with diagnostic linkage; preserve ordinary disposition and satisfied or violated Verdict. Focused tagged execution tests, relevant race tests, full regression, formatting and task-relative lint pass. Official staged-tree implementation review returns SHIP.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: