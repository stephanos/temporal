---
satisfies: [R17]
---
# fn-105-gomad-follow-ups-deferred-scope.17 D17: investigate Nexus operation determinism with two clusters

## Description
Decision on 2026-09-30: investigation approved for TestNexusOTELSuite/TestOperation. The test needs two dedicated Temporal clusters in one Go process. TEMPORAL_TEST_DEDICATED_CLUSTERS=2 already resolves the single-slot pool wait, but recorded Darwin runs then diverged on seed 17, and a seed-11 replay differed in stderr and did not produce a terminal frame. Identify the remaining cause and propose the correction; a shared cause with D12/D14 is unproven. The investigation does not preselect a fix or claim that two clusters are inherently unsupported.

## Acceptance
- Reproduce with two dedicated-cluster slots and the skipped operation enabled on seeds 11 and 17, starting with the recorded darwin/arm64 case. Retain commands, identities, repetitions, environment, and outcomes; environment blockers are recorded explicitly.
- Separate the resolved pool-capacity wait from the remaining execution differences. Compare native Go and suitable single-cluster/two-cluster controls, preserving the operation's application-tracing assertions.
- Use diagnostic choice tracing where useful to locate the first divergent event and causal path. Explain the seed-11 missing terminal frame and distinguish target termination, watchdog/cancellation, runtime evidence failure, and Runner failure from ordinary replay divergence.
- Test any suspected connection with D12/D14 against retained evidence; diagnosis of either other issue alone is not evidence that this operation is resolved.
- Record the diagnosed cause, correction owner, proposed next action, and regression/qualification criteria in fn-105 for a subsequent decision. Retain any needed fix as explicit open work and keep the skip until verification supports its removal. Classification alone does not resolve the skipped operation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
