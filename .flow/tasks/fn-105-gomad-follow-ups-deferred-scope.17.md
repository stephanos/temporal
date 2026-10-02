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
Investigation only; nothing is fixed and the TestNexusOTELSuite/TestOperation skip stays. Cause: `TEMPORAL_TEST_DEDICATED_CLUSTERS=2` never reaches the test. The Runner records a supplied `--env` entry and passes it to the process, and the patched `syscall.copyenv` then replaces the environment Go code reads with `TZ=UTC` in deterministic mode (go1.27.1.patch:892-896), so the dedicated pool keeps one slot. The test waits on itself at test_cluster_pool.go:109 with no virtual deadline, and the wall watchdog kills the target at a host-timed virtual instant; the first differing evidence field is `virtual_time_elapsed_nanos`, reported as `nondeterministic`. A killed target writes no terminal choice frame, its artifact replays untraced, and the seed-11 replay's stderr differs in one `%p` address. No evidence of a cause shared with D14 was found and its fix does not change the outcome; D12 is unassessed (no linux run). With the pool forced to two slots in a throwaway copy the unskipped suite is `qualified` on seeds 11 and 17 with exact choice-tape replay (12 executions) and passes seeds 1 to 17; native passes with 8 and 2 slots and waits the same way with 1.

Proposed correction, open work in fn-105 for a decision: deliver supplied environment entries to the target (owner: Gomad runtime patch and Runner), or reject the flag and size the pool under the `gomad` build tag in tests/testcore. Not built or verified. Report: docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md; evidence: .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d17-evidence.json. linux/amd64 not run. baseline: none (the spec defines no Quick commands).

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden) (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: none (the spec defines no Quick commands); flowctl gate classify: FULL (unmatched MILESTONES.md), no gate command is defined to run, gomad qualify-set, env=2, skip lifted, traced, seeds 11 and 17: nondeterministic (4 of 4 executions watchdog_timeout), gomad explore leaf with -test.timeout=2m, seeds 11 and 17, three runs: self-wait at test_cluster_pool.go:109, byte-identical, gomad env probe: os.Environ() is [TZ=UTC] with supplied --env entries, gomad qualify-set, pool forced to two slots (throwaway copy), skip lifted, traced, repeat 2 and repeat 4, seeds 11 and 17: qualified, exact choice-tape replay, gomad qualify-set, same, strict tick: target_failure, deterministic, gomad qualify-set, TestOperation skipped, traced, seeds 11 and 17: qualified, go test -tags test_dep ./tests -run '^TestNexusOTELSuite$' -count=3 (8 slots and 2 slots): 12 of 12 PASS each; 1 slot: timeout at test_cluster_pool.go:109, review: raw codex bridge gpt-5.6-sol, 3 rounds; round 3 no blocker, one should-fix applied-unreviewed (.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/task17-review.md)
- PRs: