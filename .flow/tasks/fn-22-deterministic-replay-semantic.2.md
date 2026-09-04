---
satisfies: [R2, R3, R9]
---

# fn-22-deterministic-replay-semantic.2 Classify fresh Case Runtime reruns
## Description
Implement two fresh isolated reruns of the admitted Case through `PrepareCase` and `PreparedCase.Run`. Classify matching decisive violations as reproduced, satisfied or different violations as not reproduced, and incomplete/inconclusive Runs as indeterminate; keep SDK history replay diagnostic and deferred.

**Size:** M
**Touches:** `tools/umpire/replay/rerun.go`, `tools/umpire/replay/rerun_test.go`

## Acceptance
- [ ] Every attempt uses fresh Run state and exact Profile/catalog preflight.
- [ ] Preparation failure creates no Run; runtime, monitor, cleanup, and Verdict outcomes retain fn-64 precedence.
- [ ] History replay cannot change reproduction or promotion eligibility.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
