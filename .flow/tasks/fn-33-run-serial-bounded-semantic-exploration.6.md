---
satisfies: [R2, R6]
---
# fn-33-run-serial-bounded-semantic-exploration.6 Enforce the serial process-local coordinator boundary

## Description
Model process-local idle, planning, preparing, running, observing, and finished states with one transition at a time. Bound candidate count, aggregate Case bytes/static work, Run time/work, event references, and report bytes; define stop/crash handling without durable recovery.

**Size:** M
**Files:** `tools/umpire/campaign/session.go`, `tools/umpire/campaign/session_test.go`
**Touches:** [tools/umpire/campaign/session.go, tools/umpire/campaign/session_test.go]

### Approach
- One state machine value owned by the coordinator; every transition consumes the previous state, so two outstanding candidates cannot be represented.
- SIGINT during a Run: bounded cleanup, `stopped` with a lost iteration named by identity; SIGINT between candidates: `stopped` with none lost.
- Caps are declared once in the command's configuration and enforced before the action they bound; exceeding one is `limit-reached`, never truncation.
- Tests drive the state machine with fakes for the bridge and the facade; a 10x candidate volume stops at the cap with bounded retained state.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/facade_external_test.go` — cleanup outcomes.
- `tools/umpire/cmd/umpire-run/run.go:40-60` — teardown bounds to reuse.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/campaign/... && GOLANGCI_LINT_BASE_REV=<base> make lint-code-fast`

### Re-plan note (2026-09-21)
Re-planned on fn-85's exploratory set after fn-86 R6 deleted the variation Space this task was first written against; see the spec's **Re-plan on fn-85** section. Start only after the spec's fresh plan review.
## Acceptance
- [ ] No state admits a second outstanding candidate; every transition is pinned.
- [ ] Every cap is enforced before the action it bounds and reports `limit-reached`.
- [ ] Stop and lost-iteration handling never synthesizes a Verdict or coverage; 10x volume stops at the cap with bounded retained state.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
