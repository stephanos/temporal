---
satisfies: [R2, R6]
---
# fn-33-run-serial-bounded-semantic-exploration.6 Enforce the serial process-local coordinator boundary

## Description
Model process-local idle, planning, preparing, running, observing, and finished states with one transition at a time. Runs before `.4`: the command's exit codes rest on this state machine, which supersedes `.3`'s local guard in the bridge client. Bound candidate count, aggregate Case bytes/static work, Run time/work, event references, and report bytes; define stop/crash handling without durable recovery.

**Size:** M
**Files:** `tools/umpire/campaign/session.go`, `tools/umpire/campaign/session_test.go`
**Touches:** [tools/umpire/campaign/session.go, tools/umpire/campaign/session_test.go]

### Approach
- One state machine value owned by the coordinator; every transition consumes the previous state, so two outstanding candidates cannot be represented.
- SIGINT during a Run: bounded cleanup, `stopped` with a lost iteration named by identity; SIGINT between candidates: `stopped` with none lost.
- Caps are the campaign's own counters (candidate cap, aggregate Case bytes, report bytes), declared once in the command's configuration and enforced before the action they bound; exceeding one is `limit-reached`, never truncation. The budget's `search` limit bounds one Search and is never a campaign cap.
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
- [x] No state admits a second outstanding candidate; every transition is pinned.
- [x] Every cap is enforced before the action it bounds and reports `limit-reached`.
- [x] Stop and lost-iteration handling never synthesizes a Verdict or coverage; 10x volume stops at the cap with bounded retained state.
## Done summary
The serial process-local coordinator. `campaign.Session` models idle, planning, preparing, running, observing and finished as one value: every transition consumes the state it starts from (a consumed value refuses everything with `ErrConsumed`) and is admitted only from the states it names, so a second outstanding candidate cannot be represented. The caps are the campaign's own counters, enforced before the action each bounds: candidate count, aggregate Case bytes and aggregate Run Events before each `next` (a Case arriving over the byte cap is never bound), the Run timeout on the Run's context before it opens, the report cap on the rendered report; exceeding one ends the campaign as `limit-reached`, never by truncation. A stop during a Run names the lost iteration by identity, releases it and observes nothing; a stop between candidates loses none; any other failure is `tooling-failure`. `campaign.Drive` is the loop: task .3's serial path reports its steps to the session (so `RunCandidate` and the coordinator share one path), one candidate at a time, with one progress line per candidate, and asks the bridge for its summary on a bounded context of its own once the campaign ended. Tests pin every transition from every state, the consumed-state refusal, each cap before its action, stop with and without a Run in flight, and drive the loop over the fake bridge and fake binders through exhaustion, the candidate cap at ten times its volume with bounded retained state, aggregate Case bytes, a stop mid-Run, a binding failure, preparation rejections, the bridge's own tooling failure and a rejected `next`.
## Evidence
- Commits: d79c6a5b0eca7fd44addbe5cf35b74f235762764, 498bd39d8d07a0cebd41772a3ce5d34fae7377df, 63ef9869540997adfb5e8a55aa3110bcbc9c3195
- Tests: go test -count=1 -timeout 180s -tags test_dep ./tools/umpire/campaign/... ./tools/umpire/binding/... ./tools/umpire/cmd/umpire-run/..., go vet -tags 'test_dep integration' ./tools/umpire/campaign/, GOLANGCI_LINT_BASE_REV=HEAD~1 make lint-code-fast
- PRs: