---
satisfies: [R2, R3]
---
# fn-33-run-serial-bounded-semantic-exploration.3 Execute one candidate through the shared runner and Run Evaluation

## Description
Implement the serial path from one checked candidate through cleanup, Result admission, and observe.

**Size:** M
**Files:** `tools/umpire/campaign/run.go`, `tools/umpire/campaign/run_test.go`
**Touches:** [tools/umpire/campaign/run.go, tools/umpire/campaign/run_test.go]

### Approach
- Execute only the outstanding candidate through the existing runner with the exact environment and Limits.
- Complete cleanup before invoking existing Run Evaluation and admitting the Result to the bridge.
- Preserve every Space, policy, planner, environment, definition, fingerprint, checksum, and Limit binding.
- Keep cancellation, runner failure, cleanup failure, rejected Result, and bridge failure distinct.

### Investigation targets
**Required** (read before coding):
- Existing fn-19 runner and cleanup contracts.
- Existing fn-20 Run Evaluation admission.
- Tasks `.1`, `.2`, and `.6`.

## Acceptance
- [ ] Exactly one Execution is active and its cleanup completes before observation.
- [ ] Only one complete admitted Result advances the Lean session.
- [ ] Positive and representative fail-closed campaign tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
