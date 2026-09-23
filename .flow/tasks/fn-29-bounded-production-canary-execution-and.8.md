---
satisfies: [R3, R4, R5, R6, R7, R8, R10]
---

# fn-29-bounded-production-canary-execution-and.8 Compose the canary controller and its run mode

## Description
Add `tools/canary/controller/controller.go` composing, in order: preflight (which prepares the Case once), the lease, the serial iterations with each iteration's record, admission, assessment and rendering held in memory, the cleanup attempt and the lease's release (or, when cleanup is uncertain, the lease left held), then publication whatever that outcome; and `tools/canary/cmd/umpire-canary` with the closed mode `run --output <dir> --records <dir> --recovery <file>` (all outside the model; the directories must exist; `--records` is runner-local and never uploaded) and no Case, target, Driver, checker, retry, executable, endpoint, credential or release flag. It writes one bounded JSON summary on stdout (each iteration's status, receipt and provenance identities, cleanup) and bounded progress on stderr through the Redactor, and exits by precedence 3 > 2 > 1 > 0: 3 for a preflight or tooling failure or a publication it could not report (each with a named status), 2 for `lease-unreconciled` (the lease found open or closed without a canary termination) or when cleanup is uncertain and the lease stays held, 1 when an iteration is rejected or incomplete, 0 when every iteration is accepted. `make canary-build` builds it.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/...`

**Files:** `tools/canary/controller/controller.go`, `tools/canary/controller/controller_test.go`, `tools/canary/cmd/umpire-canary/**`, `Makefile`
**Touches:** `tools/canary/controller/controller.go`, `tools/canary/controller/controller_test.go`, `tools/canary/cmd/umpire-canary/**`, `Makefile`

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
