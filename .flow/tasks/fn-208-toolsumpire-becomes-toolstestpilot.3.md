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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: