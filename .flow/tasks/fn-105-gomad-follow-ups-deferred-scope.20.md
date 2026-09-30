---
satisfies: [R20]
---
# fn-105-gomad-follow-ups-deferred-scope.20 D20: investigate heartbeat timeout counting under virtual time

## Description
Covered by the 2026-09-30 blanket investigation approval. TestWorkflowTaskTestSuite/TestWorkflowTaskHeartbeatingWithEmptyResult performs twelve one-second sleeps and expects two heartbeat timeouts. The recorded finding attributes the second native timeout to RPC wall-time overhead; Gomad observes one timeout in the corresponding twelve-second virtual interval. Establish the intended heartbeat semantics and propose a time-independent test correction or product fix from evidence.

## Acceptance
- Reproduce native and Gomad behavior on seeds 11 and 17, retaining heartbeat/task histories, timeout counts, configured deadlines, and elapsed logical/observed time.
- Establish the intended timeout/reset semantics and identify whether the count difference follows the test's timing assumption or a runtime/server defect.
- Propose an explicit semantic event/deadline condition that preserves coverage of heartbeat timeout and recovery without depending on incidental RPC latency.
- Record the cause, correction owner, proposed next action, and regression/qualification criteria in fn-105 for a subsequent decision; retain any needed fix as explicit open work.
- Keep the skip until evidence supports removal; do not merely change the expected count by execution mode or weaken the timeout/recovery assertions.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
