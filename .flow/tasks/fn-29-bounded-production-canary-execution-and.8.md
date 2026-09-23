---
satisfies: [R3, R4, R5, R6, R7, R8, R10]
---

# fn-29-bounded-production-canary-execution-and.8 Compose the canary controller and its run mode

## Description
Add `tools/canary/controller/controller.go` composing, in order: preflight, the lease, one `testpilot.Prepare`, the serial iterations, each iteration's record, admission, assessment and publication, cleanup, and the lease's release; and `tools/canary/cmd/umpire-canary` with the closed mode `run --output <dir> --recovery <file>` (both paths outside the model; the output directory must exist) and no Case, target, Driver, checker, retry, executable, endpoint, credential or release flag. It writes one bounded JSON summary on stdout (each iteration's status, receipt and provenance identities, cleanup) and bounded progress on stderr through the Redactor, and exits 0 when every iteration is accepted, 1 when any is rejected or incomplete, 2 for a lost iteration or cleanup uncertainty, 3 for a preflight or tooling failure. `make canary-build` builds it.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/...`

**Files:** `tools/canary/controller/controller.go`, `tools/canary/controller/controller_test.go`, `tools/canary/cmd/umpire-canary/**`, `Makefile`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Stage order and the exit statuses distinguish accepted, rejected or incomplete, lost or uncertain, and tooling failure.
- [ ] The prepared Case is reused across iterations while each Run has a fresh Driver and fresh fenced identities.
- [ ] No arbitrary Case, target, Driver, checker, retry, executable, endpoint, credential or release option exists.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
