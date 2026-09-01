---
satisfies: [R4]
---
# fn-33-run-serial-bounded-semantic-exploration.4 Expose the closed serial umpire-fuzz run command

## Description
Add the bounded public command and honest terminal output over the completed serial loop.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-fuzz/**`, `Makefile`
**Touches:** [tools/umpire/cmd/umpire-fuzz/**, Makefile]

### Approach
- Accept only the checked two-choice caller-closure fault Space, retained fn-17 policy, fn-40 PlannerPolicy, fixed local-loopback environment, candidate Limit 1..2, and positive wall-clock Limit up to 240 seconds.
- Drive the serial coordinator until exhausted, Limit Reached, stopped, or tooling failure.
- Emit the parent's closed `umpire-fuzz-run-summary/v1` field order/nullability and exit mapping, or the closed pre-summary `umpire-fuzz-run-error/v1` stderr object.
- Test every transition row: exhaustion; candidate/wall-clock Limit; cancellation; bridge, runner, cleanup, and evaluation failure; admitted operational failure; and reporting failure, including exact counter/coverage advancement.
- Keep command output operational and inspectable without defining a persisted Artifact.
- Expose no in-flight progress stream, timestamps, durations, or arbitrary error text.

### Investigation targets
**Required** (read before coding):
- Existing `tools/umpire/cmd` command conventions.
- Task `.3` serial runner interface.
- Parent spec terminal output contract.

## Acceptance
- [ ] `umpire-fuzz run` has one closed bounded command shape and no arbitrary executable option.
- [ ] Exhausted, Limit Reached, stopped, and tooling failure remain distinct.
- [ ] Canonical stdout/stderr bytes, exit codes, counters, coverage exclusion, failure phase/code, and write-failure behavior match the parent contract.
- [ ] Focused command tests and root Make wiring pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
