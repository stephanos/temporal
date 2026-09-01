---
satisfies: [R2, R3]
---
# fn-33-run-serial-bounded-semantic-exploration.3 Execute one candidate through the shared runner and Run Evaluation

## Description
Implement the serial path from one checked candidate through cleanup, Result admission, and observe.

**Size:** L
**Files:** `tools/umpire/campaign/run.go`, `tools/umpire/campaign/run_test.go`, `tools/umpire/runevaluation/**`, `model/Temporal/Tool/RunEvaluation.lean`, `model/Temporal/Tool/RunEvaluation/Protocol.lean`, `model/Temporal/Tool/RunEvaluationTests.lean`
**Touches:** [tools/umpire/campaign/run.go, tools/umpire/campaign/run_test.go, tools/umpire/runevaluation/**, model/Temporal/Tool/RunEvaluation.lean, model/Temporal/Tool/RunEvaluation/Protocol.lean, model/Temporal/Tool/RunEvaluationTests.lean]

### Approach
- Execute only the outstanding candidate through the existing runner with the exact environment and Limits.
- Extend the Go and Lean Run Evaluation allowlists with the full compiled `CallerClosureFault` baseline-point Experiment binding, paired only with the existing ordinary caller-closure RuntimeConfiguration; retain exact rejection for every other or crossed profile.
- Complete cleanup before invoking existing Run Evaluation and admitting the Result to the bridge.
- Preserve the exact caller-closure fault Space, policy, planner, fixed baseline/duplicate-delivery RuntimeConfiguration mapping, environment, definition, fingerprint, checksum, and Limit bindings.
- Keep cancellation, runner failure, cleanup failure, rejected Result, and bridge failure distinct.
- Call `observe` only for a complete admitted Result with successful operation, complete cleanup, and accepted Observation Evaluation; admitted operational failure terminates without coverage credit.

### Investigation targets
**Required** (read before coding):
- Existing fn-19 runner and cleanup contracts.
- Existing fn-20 Run Evaluation admission.
- Existing Run Evaluation subject snapshots and mutation tests — pin every field of the new baseline-point Experiment/profile pair rather than its RuntimeConfiguration alone.
- Tasks `.1`, `.2`, and `.6`.

## Acceptance
- [ ] Exactly one Execution is active and its cleanup completes before observation.
- [ ] Only one complete admitted Result advances the Lean session.
- [ ] The existing baseline and duplicate-delivery closures both run; basic-lifecycle and mismatched RuntimeConfiguration candidates fail before execution.
- [ ] Go and Lean Run Evaluation agree on the exact baseline-point checksum, query ID, Behavior Fingerprint, provenance, properties, and ordinary configuration; one-field and crossed-profile mutations remain unsupported.
- [ ] Positive and representative fail-closed campaign tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
