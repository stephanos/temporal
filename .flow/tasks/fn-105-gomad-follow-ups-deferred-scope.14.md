---
satisfies: [R14]
---
# fn-105-gomad-follow-ups-deferred-scope.14 D14: fix Darwin Chasm replay divergence and restore qualification

## Description
Origin: F7 (2026-09-29). TestSignalWorkflowTestSuiteChasm is intermittent on darwin: about one seed-11 replay in 28 differs in stderr because two heap-span refills (size classes 11 and 50) swap order at decision 46 of cluster start; the allocating goroutine is not identified. The cause may be shared with D12, but that connection is unproven.

Decision on 2026-09-30: this is a required fix. Remove the previous D12/rate-rise deferral. Instrument the allocations on darwin/arm64, identify the allocating goroutine and first divergent event, fix the cause, and retain a regression reproducer. A D12 fix may be reused if it resolves this failure, but this task needs its own Darwin verification and stays open until that evidence passes.

## Acceptance
- Identify the allocating goroutine and causal runtime path behind the heap-span refill ordering difference on darwin/arm64.
- Fix the cause without widening the deterministic boundary or suppressing differing stderr or replay evidence. A retained regression reproducer demonstrates the original failure and passes with the fix.
- Repeated qualification with choice tracing enabled and exact replay of TestSignalWorkflowTestSuiteChasm pass on seeds 11 and 17 on darwin/arm64, including host-load runs. Retain commands, platform identity, repetition counts, and outcomes; a shared Linux fix or environment blocker is not Darwin qualification evidence. Disabling tracing cannot satisfy this acceptance criterion.
- After verification passes, restore the suite's Darwin expectation to `qualified`; the existing qualification gates reject nondeterministic and replay-divergence outcomes for it.
- Diagnosis, classification, or an unverified shared-cause hypothesis alone cannot satisfy R14 or close this task.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
