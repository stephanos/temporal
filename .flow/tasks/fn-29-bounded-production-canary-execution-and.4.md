---
satisfies: [R4, R5, R10]
---

# fn-29-bounded-production-canary-execution-and.4 Implement fenced serial Run and cleanup lifecycle
## Description
Implement one exclusive lease/fence, one active serial `PreparedCase.Run`, hard iteration/RPC/worker/evidence/time limits, and unsuppressible cleanup under a fresh context. Scope every effect and cleanup action to exact canary-owned identities.

**Size:** L
**Touches:** `tools/canary/controller/run.go`, `tools/canary/controller/run_test.go`, `tools/canary/control/**`

## Acceptance
- [ ] Collision, concurrent Run, stale fence, ambiguous dispatch, duplicate command, scope escape, and N+1 work fail closed.
- [ ] A 10x request increase is capped without adding concurrency or unbounded retained state.
- [ ] Cleanup never mutates unrelated resources and preserves failure or uncertainty independently from Verdict.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
