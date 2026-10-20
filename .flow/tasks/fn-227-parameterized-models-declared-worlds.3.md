---
satisfies: [R3, R4]
---
# fn-227-parameterized-models-declared-worlds.3 After .2: lifter admits impl Phased and reject! arms; activity Model gains maximum_attempts and RespondFailed { retryable }

## Description
Extend umpire-lift to admit the trait impl Phased and reject! arms in a machine module so the activity Model is no longer refused, with tlc goldens for it. Extend temporal/features/activity/model.rs in place: World { maximum_attempts: 1..=3 }, state attempt, Worker::RespondFailed { retryable: bool }, Worker::Poll carrying the expected attempt (wrong attempt Rejected), retry to Scheduled with attempt + 1 below the cap, Failed at the cap or on non-retryable. Pin the maximum_attempts = 1 world as a fixture that reproduces the six-phase goldens exactly; the default fixture world is maximum_attempts = 3 with new goldens. Update tests.rs matrices and terminal-law controls over attempts, and fn-218's activity cover test with its Cases(n). Keep the fn-216 realization and live slice compiling against the extended Model (Answer maps Fail to retryable = false until .5).

## Acceptance
- [ ] Activity Model lifts and agrees with TLC
- [ ] maximum_attempts = 1 world reproduces six-phase goldens byte-identical
- [ ] Default world goldens, cover test and terminal controls updated
- [ ] Live slice still passes; gate and model-time pasted

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
