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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: