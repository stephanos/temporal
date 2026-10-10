# Remaining spinning-loop findings

Source closure can remove these two lint findings while retaining intentional CPU activity. The narrow design is an atomic stop flag for the conformance load workers and an explicit atomic increment loop for the unresponsive supervisor fixture. Neither loop should block, sleep or call `runtime.Gosched`. This is a lint correction with behavioral preservation evidence. Source inspection found no existing behavioral defect that would honestly produce a new failing regression on the baseline.

This read-only research proposes a new bounded owner. Task 64 admits only a checked watchdog readiness write and explicitly preserves its wait loop. It does not authorize either change below. Root owns admission, Flow lifecycle, execution gates and native deferrals. No product file, test, configuration, index or commit was changed, and no Go, build, test, lint or checker command was run for this report.

## Evidence binding

The inspected joined checkout is `/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/joined-current`, HEAD `b88c7dcb3c2681e58b2692eee9c49b18026d36fe`. The primary checkout supplied for this investigation is HEAD `6af88c48ac6ac5c934dfc5e938b380504bbb5c2a`. Product observations below refer to the joined checkout.

The retained `combined-64-65/integrated-lint.json` reports the original-base configured lint command exiting 2, with frozen before/after source, on 2026-10-10. `lint-comparison.json` records 53 baseline findings and 52 current findings, zero introduced findings and zero mapping gaps. Both loop findings remain in the current blocks. No aggregate green result is claimed.

| Input | SHA-256 |
| --- | --- |
| `combined-64-65/lint-comparison.json` | `f2d25a22c9785b8e115e0775e4147370dd7ff74fbbbb5d876bac414b8b17dd59` |
| `combined-64-65/integrated-lint.json` | `a3f59c238127c468bafdf3cccec2a8dbded9373262b8e22f719c8b31516cc265` |
| `combined-64-65/integrated-lint.log` | `0ed26fe36ddd89dc5ac542e0083c07a1d4213423bed526368db4f9bba0e5fb6b` |
| `tools/gomad3/internal/gomadtool/conformance/runtime_repeatability.go` | `833989fd248cd1526395e5be90d954aa24fe4108adbcad406a7d6384cc1ab549` |
| `tools/gomad3/runner/internal/execution/process_test.go` | `4a94563b09e82dec418088309cc4ae01f49f7899389f4c0d184ecb0e6579bb59` |
| `tools/gomad3/qualification/soak/soak.go` | `29d0fdd667d03ed3de82f5a9902a822e8f61b39868a8a0f214efb76155252b3a` |

## SA5004 belongs to the host load generator

`internal/gomadtool/conformance/runtime_repeatability.go:295-337` defines `startCPULoadWorkers`. Each worker locks an OS thread, announces startup, polls a shared stop channel through an empty-default select and exits only when the channel closes. Startup has a one-second bound per acknowledgement. The returned stop function uses `sync.Once`, closes the channel and joins all workers. Startup failure also closes and joins before returning its existing error.

Its sole caller is `requireHostLoad` in `internal/gomadtool/conformance/runtime_load.go:59-94`. The caller records nine fixture baselines, starts two workers, compares eight repetitions of each fixture under load, and stops the workers both explicitly and through deferred cleanup. `runtime_scheduling.go:69` includes that check in runtime conformance. The workers run in the host harness. Seeded fixture processes are separate commands with their own `GOMAXPROCS=1` and `GOMADSEED=1` environment (`runtime_load.go:97-102`).

The empty default is intentional CPU load. Removing it would park the workers and invalidate the host-load comparison. A yield, timer, sleep or blocking receive would also change that premise.

`qualification/soak/soak.go:668-704` provides the exact established alternative. Its `startLoad` uses a local `atomic.Bool`, spins on `!stopped.Load()` on locked OS threads, stores true on stop, and joins the workers under `sync.Once`. The preemption fixture also uses an atomic flag spin (`internal/gomadtool/conformance/testdata/preemption/main.go:12-17`). `sync/atomic` is standard library code already used by the module; no dependency is needed.

Recommended implementation is to mirror the stop-flag representation inside `startCPULoadWorkers` only. Preserve count, startup channel, locked-thread lifetime, timeout/error text, startup-failure join, stop return type and idempotent join. Both former `close(stop)` paths must store true before waiting. No shared load abstraction or change to the soak package is needed. Atomic polling changes the CPU instruction mix, which is acceptable for unrelated busy load, but it establishes no identical host scheduling or native repeatability claim.

Cheap ordinary-source preservation tests can exercise zero and two workers, successful stop, repeated stop and concurrent stop calls, all under an external wall bound. Run the potentially hanging lifecycle exercise in a subprocess so a broken stop cannot hang the whole test process. These controls should pass on both versions. Do not invent a negative-count contract or add global hooks just to manufacture a failing test. A deterministic startup-timeout test would require a new seam and exceeds this correction.

## SA5002 belongs to the deliberately hung supervisor

`runner/internal/execution/process_test.go:1428-1434` defines `TestUnresponsiveSupervisorHelper`. It skips without `GOMAD3_PROCESS_SUPERVISOR=1`, then enters an unconditional empty loop. Its only caller is `TestRunBoundsUnresponsiveSupervisor` at lines 810-825. The caller provides a 300 ms execution timeout and 100 ms grace, then requires a supervisor-related error and completion within two seconds.

`runner/internal/execution/process_unix.go:295-296` launches that helper as the host supervisor and adds the marker to the inherited host environment. The child does not run the normal supervisor protocol or launch a seeded target. The parent writes the request, waits for supervisor reports, and eventually kills/reaps an unresponsive child through its wall-time path (`process_unix.go:515-558`). This fixture therefore exercises a process that consumes CPU while ignoring all protocol requests.

The existing parent assertions alone cannot prove preservation. An early child exit can also produce a supervisor error within two seconds. Replacing the loop with `select {}`, a never-ready receive, sleeping or `runtime.Gosched` can preserve those assertions while changing the failure mode. An empty select can additionally encounter runtime deadlock behavior depending on the remaining test-harness goroutines and timers. None is the recommended fix.

The smallest source-only candidate that preserves CPU activity is a local `atomic.Uint64` plus an unconditional loop calling `Add(1)`. That expresses a non-elidable busy workload with no voluntary yield, I/O, protocol response, allocation in the loop or terminating counter condition. Unsigned wraparound must remain harmless; never turn the counter into an eventual exit condition. Keep the environment guard and existing parent assertions unchanged. This is a test-fixture implementation choice, not a production-runtime or deterministic scheduler change. It changes the work performed by the CPU and needs fresh review rather than an assertion of instruction-level equivalence.

A direct subprocess preservation control may start the actual helper, observe that it stays alive until the parent kills it, then verify a signalled/reaped child and absence of supervisor-protocol output. Bound the observation and cleanup externally. This detects accidental normal return or an early fatal exit. It does not prove CPU saturation, instruction equivalence or deterministic scheduling. Avoid brittle CPU-time thresholds or tests that inspect the AST for an atomic increment. The unchanged `TestRunBoundsUnresponsiveSupervisor` remains the end-to-end timeout check.

If the owner requires an unchanged instruction stream or a behavioral baseline failure, there is no supported correction recommendation for this second site under those constraints. The intentional spin currently satisfies its behavioral purpose. A lint exception would require separate explicit policy authority, which this report neither assumes nor requests.

## Corrective owner and verification boundary

The recommended owner admits exactly these two existing paths and, if lifecycle controls are required, one additive test file in each containing package. It excludes `watchdog_io_test.go`, production execution, toolchain/runtime overlays, native spin fixtures, all existing assertions, lint policy, format migration and the 42 capitalization findings.

The real RED is the retained configured lint result with SA5004 and SA5002. After admission, a worker can reproduce it on the frozen baseline and retain a configured unfiltered affected-package comparison after the correction. The expected result is removal of exactly these two blocks with no introduced finding. Other configured failures may keep the command red; removal of two diagnostics is not a green aggregate gate. Whole required lint and source acceptance remain open until their owners resolve the other findings. No artificial behavioral RED should be represented as evidence for a defect this research did not find.

Run ordinary source preservation with the pinned host Go and `-tags test_dep`, including the new lifecycle controls and unchanged `TestRunBoundsUnresponsiveSupervisor`. The existing `test-harness` recipe establishes ordinary host coverage for `./internal/gomadtool/conformance`; `test-host` includes execution tests but also seeded targets. A focused stock-Go execution test is partial portable evidence only. Root serializes those commands with configured lint, affected vet/errortype, formatting, generated validation and the both-source-set static checks retained by source acceptance. No new checker framework is warranted.

Native `test-runtime` must eventually re-establish the actual nine-fixture host-load comparisons with the corrected load generator. Native full `test-host`, seeded non-progress controls and supported-platform qualification remain with fn-149/fn-128 under `.flow/artifacts/native-scope-transfer-2026-10-07.md`. Neither portable lifecycle coverage nor absence of the two lint findings proves those native gates. `README.md:992-995` and `MILESTONES.md` explicitly preserve the distinction between CPU/polling spins and virtual-time progress. This owner must retain it.
