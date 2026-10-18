---
satisfies: [R3]
---
# fn-208-toolsumpire-becomes-toolstestpilot.3 Boundary tests for tools/testpilot and common/testing/testpilot

## Description
Add tests holding tools/testpilot to no `model` imports and no Model-reading package, and common/testing/testpilot to no `tools` imports; each fails naming the violating path.

**Files:** tools/testpilot/boundary_test.go, common/testing/testpilot/dependency_boundary_test.go.

**Quick commands:** `go test -tags test_dep ./tools/testpilot/... ./common/testing/testpilot/...`

## Acceptance
- Tests pass and fail on a planted violation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
