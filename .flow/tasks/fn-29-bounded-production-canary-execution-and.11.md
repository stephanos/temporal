---
satisfies: [R3, R4, R5, R7, R8, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.11 Run adversarial authority and containment matrices
## Description
Exercise target/routing drift, credential disclosure, lease/fence collision, duplicate dispatch, Run crossing, scope escape, crash, stale recovery, cleanup uncertainty, N+1 load, publication conflict, and report failure against the external controller.

**Size:** L
**Touches:** `tools/canary/**/*_test.go`, `tests/umpire_canary_test.go`

## Acceptance
- [ ] Every mutation fails at its owning boundary and cannot affect unrelated resources.
- [ ] Proven violation remains violated; otherwise incomplete execution/evaluation stays inconclusive.
- [ ] Race-enabled tests prove no cross-Run or cross-lease state leakage.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
