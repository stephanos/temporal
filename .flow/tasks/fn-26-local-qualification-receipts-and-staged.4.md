---
satisfies: [R2, R3, R4, R6]
---

# fn-26-local-qualification-receipts-and-staged.4 Implement offline local Claim Assessment
## Description
Implement the deep offline assessor over one admitted subject and one compiled Profile. Apply the complete reason table, preserve absent evidence and Known Gaps, and construct a receipt in memory without Host construction, target I/O, Contract evaluation, or caller-defined policy.

**Size:** M
**Touches:** `tools/umpire/evaluation/assess.go`, `tools/umpire/evaluation/assess_test.go`

## Acceptance
- [ ] Accepted, rejected, and incomplete decisions accumulate all reasons deterministically.
- [ ] Cancellation, protocol failure, missing evidence, status drift, and N+1 never produce accepted.
- [ ] Repeated or different-Profile assessments create no Run and preserve the source subject.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
