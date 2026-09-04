---
satisfies: [R1, R4, R5, R7, R8]
---

# fn-26-local-qualification-receipts-and-staged.6 Prove assessment isolation and document the contract
## Description
Complete Profile, subject, receipt, status, reason, multiplicity, Limit, Known Gap, protocol, cancellation, and publication mutation matrices. Document the exact local claim, non-self-authentication, and separation between fn-64 verification and offline Claim Assessment.

**Size:** M
**Touches:** `model/Umpire/EvaluationTests.lean`, `tools/umpire/evaluation/**`, `docs/**`, `.plans/UMPIRE4_COMPONENTS.md`

## Acceptance
- [ ] Crossed, N/N+1, multiple-Profile, idempotency, cleanup, evidence, identity, and output cases fail at the intended boundary.
- [ ] Focused and aggregate Lean/Go/regression, formatting, and lint gates pass with `-tags test_dep` for Go tests.
- [ ] Existing comments remain accurate and docs exclude CI, remote, canary, production, release, and implicit authority.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
