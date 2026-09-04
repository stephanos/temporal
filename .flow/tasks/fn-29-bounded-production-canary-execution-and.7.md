---
satisfies: [R7, R10]
---

# fn-29-bounded-production-canary-execution-and.7 Implement immutable canary receipt publication
## Description
Publish each valid fn-26-derived canary receipt with exact source closure and secret-free provenance through a canary-owned retained-artifact channel. Preserve source Case Runtime values byte-for-byte and make same-content retry idempotent.

**Size:** M
**Touches:** `tools/canary/publication/**`, `tools/canary/assessment/receipt_test.go`

## Acceptance
- [ ] Accepted, rejected, and incomplete receipts publish honestly; lost or unconstructible iterations do not fabricate receipts.
- [ ] Conflicting content, alias/symlink drift, partial output, or crossed subject fails closed.
- [ ] Reporting ambiguity after successful publication forbids automatic rerun.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
