---
satisfies: [R7, R10]
---

# fn-29-bounded-production-canary-execution-and.6 Add secret-free canary provenance and receipt bindings
## Description
Define bounded canary provenance over authority class, workflow context, target/routing digests, lease/fence, invocation, limits, public evidence closure, isolation, cleanup/reconciliation, trust, and Known Gaps. Bind it to the fn-26 receipt without placing credentials or canary policy in reusable Umpire.

**Size:** L
**Touches:** `tools/canary/assessment/provenance.go`, `tools/canary/assessment/provenance_test.go`, `api/umpire/**`

## Acceptance
- [ ] Canonical identity and cross-language fields are exact, bounded, and secret-free.
- [ ] Every version, identity, status, closure, relation, and N+1 mutation fails closed.
- [ ] `releaseEligibility:false` is structural and cannot be overridden.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
