---
satisfies: [R3, R5, R8, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.9 Add lost-iteration recovery and protected workflow
## Description
Add canary-owned mode-0600 recovery and bounded progress records plus a closed reconcile mode and protected manual workflow. Persist only invocation, lease/fence, active Run identity, dispatch phase, cleanup reserve, and expiry; record a process-lost active Run as lost.

**Size:** L
**Touches:** `tools/canary/recovery/**`, `tools/canary/progress/**`, `.github/workflows/umpire-production-canary.yml`

## Acceptance
- [ ] Reconcile may terminate or verify exact fenced resources but cannot prepare, dispatch, assess, publish, or synthesize Verdict.
- [ ] Missing, tampered, stale, crossed, or uncertain recovery state cannot redispatch or be accepted.
- [ ] Workflow is manual, protected, fixed-ref, least-privilege, bounded, and always attempts reconciliation/evidence retention.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
