---
satisfies: [R3, R4]
---
# fn-217-annotation-catalog-over-the-temporal.4 Gap probe and reverse coverage: gaps.txt golden

## Description
Gap probe: a Go test beside fn-216.3's that, for each `Expectation` of the mini activity catalog, builds one minimal violating request from a valid filled one, sends it to the in-process server and classifies `Confirmed`/`Mismatch`/`Accepted`/`Unverifiable`; writes/compares the golden `features/activity_mini/expected/gaps.txt`; `Mismatch` and `Accepted` fail unless listed in the catalog's dated `accepted_gaps` with a reason. Reverse coverage: a Go test parses the server packages of the touched RPCs, fails on a listed `covers:` function that no longer exists, and reports validators without a catalog entry. After .3. **Touches:** `common/testing/testpilot/temporal/**` (tests), `model/temporal/features/activity_mini/expected/gaps.txt`, Makefile Testpilot section (exact minimal replacement).

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
