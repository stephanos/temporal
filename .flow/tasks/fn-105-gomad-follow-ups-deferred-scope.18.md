---
satisfies: [R18]
---
# fn-105-gomad-follow-ups-deferred-scope.18 D18: investigate worker cancellation delivery and timeout budgets

## Description
Investigation approved on 2026-09-30 for TestWorkerCommandsTaskSuite/TestDispatchCancelOnWorkflowTimeout, following the seed-11 walkthrough and blanket approval of remaining investigations. The cancellation command is not received before the 90-second test context has less than the two-second long-poll minimum remaining, while Awaitf permits 120 seconds. Seed 17 passes. Identify why delivery is late and propose the correction; implementation remains a subsequent decision.

## Acceptance
- Reproduce the unskipped test on seeds 11 and 17 and compare native Go with Gomad; retain commands, identities, platform, virtual-time observations, and outcomes.
- Trace workflow timeout, cancellation generation, persistence/transfer, worker-control-queue delivery, and polling to locate the causal delay.
- Account for the 90-second parent context, five-second child polls, two-second server minimum, and 120-second Awaitf; establish which budget and delivery assumptions are valid.
- Record the cause, correction owner, proposed next action, and regression/qualification criteria in fn-105. Retain any needed fix as explicit open work for a subsequent decision.
- Preserve the cancellation-delivery assertion and keep the skip until verification supports removal; a longer timeout or classification alone does not resolve the issue.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
