---
satisfies: [R3, R10]
---

# fn-29-bounded-production-canary-execution-and.3 Implement protected authority and exact-scope preflight
## Description
Acquire authority only from the protected manual workflow and preflight trusted ref, workflow context, production-canary target/routing, public capabilities, isolation attestation, fixed Case/Profile/catalog, and run-owned identities before mutation. Keep credentials in canary-owned runtime state.

**Size:** L
**Touches:** `tools/canary/authority/**`, `tools/canary/preflight/**`

## Acceptance
- [ ] Any ref, target, routing, capability, identity, scope, expiry, or disclosure mismatch performs no mutation and creates no Run/receipt.
- [ ] Raw coordinates and credentials never enter Case, Run, Verdict, receipt, progress, or logs.
- [ ] The early proof demonstrates exact canary scope without claiming global production audit.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
