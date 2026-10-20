---
satisfies: [R3, R4]
---
# fn-217-annotation-catalog-over-the-temporal.4 Gap probe and reverse coverage: gaps.txt golden

## Description
For each expectation from the Rust activity catalog, build one minimal violating request
from a valid filled one, send it to the existing in-process activity server, and classify
Confirmed/Mismatch/Accepted/Unverifiable. Compare the committed golden under
model/temporal/features/activity/expected; ordinary tests never update goldens.
Mismatch and Accepted fail unless the catalog contains a dated accepted-gap reason.
Parse the actual server packages to verify listed validator functions still exist and
report unlisted validators for touched RPCs. The start/describe/cancel validators are in
chasm/lib/activity/validator.go; preserve the frontend validators for other RPCs.
Use .3's actual catalog-driven guest route. Do not duplicate annotations in a Go table.

**Touches:** `tests/testpilot_model_gaps_test.go`, `tests/testpilot_model_slice_test.go`, `model/temporal/features/activity/api.rs`, `model/temporal/features/activity/expected/**`, `model/temporal/testing/**`, `Makefile`
Makefile edits are limited to the Testpilot section.
**Quick:** `go test -tags test_dep,integration ./tests -run '^TestTestpilotModelGaps$'` with the guest built by the existing slice target and its environment supplied.
**Full gates:** `make model-check`; dedicated gap and model-slice targets.

## Acceptance
- Every catalog request rule/constraint produces an actual live probe result and a stable golden.
- Expected code, message pattern and details type are checked; mismatches and accepted violations require dated catalog reasons.
- Listed validators are verified to exist; missing reverse coverage is reported.
- Probe reads the Rust catalog rather than a duplicate Go rule list; all commands execute actual tests.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
