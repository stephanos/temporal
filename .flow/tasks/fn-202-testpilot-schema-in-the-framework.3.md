---
satisfies: [R3]
---
# fn-202-testpilot-schema-in-the-framework.3 Split the Go runtime into generic packages and the Temporal driver; boundary test

## Description
Ensure internal, evaluation, contract, recordedrun, campaign and replay import no Temporal API type; keep Temporal reads in the temporal package and the Driver; add a boundary test.

**Files:** common/testing/testpilot/**, new common/testing/testpilot/boundary_test.go.

**Quick commands:** `go test -tags test_dep ./common/testing/testpilot/...`

## Acceptance
- Boundary test passes and fails on a planted violation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
