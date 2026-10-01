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
Fixed the darwin/arm64 replay divergence of TestSignalWorkflowTestSuiteChasm and restored its darwin expectation to `qualified` in the generated manifest. Nothing is committed; the changes are in the working tree. linux/amd64 was not run and D12 stays open.

Cause. `lock2` samples lock wait time (`mLockProfile.start`) just before an M sleeps on a contended runtime lock, and under Gomad that draw came from the process-wide seeded stream whenever the waiting M held the P. Whether the scheduler lock outlasts the waiter's spin is host timing. In the instrumented divergent replay the P's M waited in `findRunnable` at decision 7629 while another M parked in `stopm`. The draw shifted every later seeded draw by one, so type-assertion caches grew at other call sites: first on the cluster-start goroutine (17), in `database/sql` and `fx`. At decision 11483 goroutine 12 then grew the `fx.(*module).provide` cache from another size, span classes 8 and 11 were refilled in swapped order, and the replay diverged at choice ordinal 35573. A contended draw at startup shifts the stream too but left the evidence unchanged.

Fix. One added hunk in `lock_spinbit.go` (already allowlisted) calls `gomadLockProfileStart`, added to the overlay. It delegates to the upstream `start` when Gomad is disabled and otherwise draws from the M's own stream, as lock hand-off, work-steal order and pcvalue eviction already do. `mprof.go` is a prohibited area and is not patched; the allowlist and descriptor are unchanged. `make generate` updated four digest consumers. The README Contract paragraph names the new host-side draw. New toolchain key 8d28bd44.

Reproducer. `TestProfileSchedulerLockContentionLeavesSeededStreamInPlace` with the `io_handoff_contention` fixture (test-host tier) runs 12 executions that hand the P off for runner syscalls while wake-ups take the scheduler lock, then compares where the seeded stream stands. It failed 20 of 20 invocations on the unfixed toolchain and passed 20 of 20 on the fixed one.

Evidence for the cause, unfixed toolchain with scratch instrumentation, seed 11 under host load: 5 of 350 replays diverged. None of the 345 matching replays slept on a runtime lock while holding the P. The one divergent replay with a dump did so once, just before its first differing event.

Verification on the fixed toolchain, darwin/arm64 (Apple M2), tracing on, under host load:
- 400 exact replays per seed of a retained success on seeds 11 and 17, 0 divergences. At the historical 1-in-28 rate that outcome has a probability below 1e-6 per seed; at the 1-in-70 rate measured here, 0.3 percent.
- `gomad qualify --repeat=4 --choices --replay-successes` qualified on both seeds.
- Four runs of the generated workload (seeds 11 and 17, repeat 2, traced, replayed) met the restored `qualified` expectation.
- `make -C tools/gomad3 test-runtime` and the full `make -C tools/gomad3 test` passed. A first full run alongside six replay loops failed with 10 s watchdog timeouts in test-host and is recorded as inconclusive.
- `make gomad3-smoke-qualification` met expectations 4/4; `make -C tools/gomad3 generate validate` exit 0.

Not done or not green:
- `make lint-code-fast` exits 2 on three inherited staticcheck reports in `tests/` files this task does not touch, and it cannot type-check the nested gomad3 module. gofmt and go vet are clean for this task's Go files.
- The representative set and the full `./tests` set were not rerun on the fixed toolchain.
- The same draw is a plausible D12 channel on linux/amd64. That is an unverified hypothesis.

Evidence index: `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d14-evidence.json`; review: `task14-review.md`.

baseline: none (the spec defines no Quick commands)

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden) (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: go test -tags test_dep -count=1 -run TestProfileSchedulerLockContentionLeavesSeededStreamInPlace ./runner/internal/execution (unfixed key c0661e38: 20/20 fail; fixed key 8d28bd44: 20/20 pass), gomad qualify --seed=11|17 --repeat=4 --choices --replay-successes go-test ./tests -- -test.run=^TestSignalWorkflowTestSuiteChasm$ (qualified, both seeds, under host load), gomad replay <retained success> x400 per seed, seeds 11 and 17, under host load (800 exact, 0 diverged), gomad qualify-set --manifest=<tests.json reduced to TestSignalWorkflowTestSuiteChasm> x4 (expectations-met=true, choice_replay_exact on both seeds), make -C tools/gomad3 test-runtime (passed), make -C tools/gomad3 test (passed on a quiet machine; an earlier run under replay load failed with watchdog timeouts and is inconclusive), make gomad3-smoke-qualification (expectations-met=true 4/4), make -C tools/gomad3 generate validate (exit 0), make lint-code-fast (exit 2: inherited staticcheck reports in tests/, nested module not type-checkable)
- PRs: