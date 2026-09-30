---
satisfies: [R13]
---
# fn-105-gomad-follow-ups-deferred-scope.13 D13: make choice tracing opt-in for routine qualification

## Description
Origin: fn-106.3 (2026-09-30). TestTaskQueueStats_Pri_Suite, TestVersioning3FunctionalSuite, TestVersioning3QueryFunctionalSuite, and TestWorkerDeploymentSuite exceed the 64 MiB choice-trace maximum and currently run without a choice trace. Their recorded same-seed repeatability evidence remains valid; it does not establish choice-tape replay.

Decision on 2026-09-30: make runtime choice tracing opt-in for routine full-suite functional qualification. The CLI already defaults seed runs to tracing off; the full-suite generator currently enables it by default. Change that generator policy and its generated manifest, preserve explicitly traced replay/conformance gates and choice-based exploration, and distinguish seed repeatability from verified choice-tape replay in reports and documentation. D12 and D14 remain required fixes with tracing-enabled verification. Larger-trace support is separately deferred under R15 and is not required for completion of this policy task.

## Acceptance
- Default routine full-suite generation to choice_bytes 0 and replay_successes false, with consistent zero success-retention limits. Individual workloads may explicitly opt into bounded tracing and replay.
- Preserve the CLI's existing opt-in controls and tracing requirements for choice-based exploration. Preserve explicitly traced representative replay/conformance gates.
- Reports and documentation distinguish same-seed repeatability, tape availability, and verified choice-tape replay. Untraced runs make no choice-tape replay claim and continue to reject same-seed evidence mismatches and target/infrastructure failures.
- D12 and D14 retain explicitly tracing-enabled qualification and exact-replay verification; disabling tracing or weakening their existing traced gates cannot close either fix task.
- Verify both generator defaults and explicit tracing overrides, and regenerate the full-suite manifest. Closing this task claims only the opt-in policy change; the larger-trace extension remains recorded and deferred under R15.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
