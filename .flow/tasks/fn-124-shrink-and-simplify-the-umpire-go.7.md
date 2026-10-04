---
satisfies: [R7]
---
# fn-124-shrink-and-simplify-the-umpire-go.7 Retire the migration harness and frozen snapshots

## Description
Implements R7. Cross-spec entry gate: fn-114, fn-120 and fn-122 are all closed. First rewrite behaviour tests that only the harness exercises against live IR, then remove internal/golden, the original-baseline and migration tests, pin and parity tests that re-check frozen values, and the frozen snapshots listed in R7. Report test lines and testdata bytes before and after.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
