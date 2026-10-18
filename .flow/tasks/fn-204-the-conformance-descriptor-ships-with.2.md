---
satisfies: [R2]
---
# fn-204-the-conformance-descriptor-ships-with.2 Go conformance interprets the descriptor; no Model-reading imports

## Description
Replace `conformance.Prepare(model, claimKey, case, limits)` with `Prepare(descriptor, case, limits)`; the bitset search reads rows and bits; evidence roles map through the Temporal driver's mapping; add the 'outside descriptor bound' reason; the package imports nothing from ir, interp, check or realization.

**Files:** tools/umpire/conformance/*.go -> common/testing/testpilot/conformance/, tests.

**Quick commands:** `go test -tags test_dep ./common/testing/testpilot/...`

## Acceptance
- Package imports no Model-reading package (boundary test).
- Evidence outside the slice is Inconclusive with the new reason.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
