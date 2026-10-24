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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: