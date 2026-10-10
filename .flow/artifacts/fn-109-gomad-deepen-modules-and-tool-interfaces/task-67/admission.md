# Preserve busy CPU activity while correcting two spin findings

The conductor admits the two source-backed loop corrections from [spinning-loop-research.md](../combined-64-65/spinning-loop-research.md), SHA-256 `583de5b19fecd9c6fcf106f9ae8deb412a626fd9b02d248a86a46c778a73c711`. The observed source retains SA5004 in `internal/gomadtool/conformance/runtime_repeatability.go` and SA5002 in `runner/internal/execution/process_test.go`, among 52 aggregate findings. Root selects the existing soak atomic-stop pattern and an unconditional atomic-increment busy fixture under the user's instruction to follow recommendations instead of asking when in doubt.

Both loops intentionally consume CPU. Blocking receives, sleeps, timers, empty selects and voluntary `runtime.Gosched` would alter that premise. A lint exception would need separate authority. Atomic polling preserves busy host load and the existing stop/join lifecycle; unconditional unsigned atomic increment preserves an unresponsive busy supervisor without overflow termination. Neither establishes an identical instruction stream, scheduling pattern, native repeatability or measured CPU saturation.

## Exact implementation boundary

The two existing product paths are:

- `tools/gomad3/internal/gomadtool/conformance/runtime_repeatability.go`
- `tools/gomad3/runner/internal/execution/process_test.go`

Inside `startCPULoadWorkers`, replace only the stop-channel representation with a local `atomic.Bool`, mirroring `qualification/soak/soak.go`. Workers retain locked OS threads, startup acknowledgement, count, wait-group ownership and busy polling until stopped. Both startup-failure cleanup and the returned idempotent stop store true before joining. Preserve existing timeout, error text, return type and `sync.Once`. Do not change the soak package or add a shared abstraction.

Inside `TestUnresponsiveSupervisorHelper`, retain the existing environment guard and replace only its empty loop with a local `atomic.Uint64` and unconditional `Add(1)`. No counter condition, yield, protocol response, I/O or allocation in the loop. Keep all original parent assertions and production execution unchanged.

Two additive files may provide preservation controls:

- `tools/gomad3/internal/gomadtool/conformance/cpu_load_lifecycle_test.go`
- `tools/gomad3/runner/internal/execution/unresponsive_supervisor_lifecycle_test.go`

Run load lifecycle exercises in a bounded subprocess, covering zero/two workers and repeated/concurrent stop. A broken join must not hang the whole suite. For the supervisor, exercise the actual helper and retain a bounded observation of continued liveness, no protocol output and killed/reaped child. The unchanged parent timeout test remains the end-to-end check; early normal exit can satisfy its existing assertions, so it is insufficient alone. Use actual process cleanup with checked errors and appropriate host constraints. Avoid CPU-time thresholds, source/AST assertions, global hooks or artificial negative-count/startup-timeout seams.

## Failing check and preservation

The actual configured-lint findings supply the meaningful RED for this lint correction. Baseline lifecycle controls should pass because research found no runtime behavioral defect. Do not invent or label a passing lifecycle control as a behavioral RED. Retain baseline configured unfiltered affected lint with both original complete blocks before production edits, then verify exact removal with no introduced finding. Remaining findings keep aggregate acceptance red.

Retain ordinary source lifecycle controls before and after, the unchanged `TestRunBoundsUnresponsiveSupervisor`, affected vet/errortype and formatting, architecture/package checks after imports, generated validation and repository fast lint. Use `test_dep` for all tests, pinned tools, no fixes and bound immutable raw receipts. The root owns one original-base aggregate comparison at the next frozen batch against the retained 52. Native full runtime/host-load/seeded checks remain transferred and unverified under fn-149/fn-128. Source passes cannot substitute for them.

Worker code scope excludes watchdog readiness, production execution, seeded/native spin fixtures, patched runtime/toolchain overlays, canonical/error-literal changes, all existing assertions and lint policy. Atomic instruction mix changes are explicitly admitted for these two host-side workloads only. No deterministic scheduler or virtual-time mechanism changes.

Task 66 independently owns ten calls in `runner_test.go`. Its source surface is disjoint and may be implemented in a separate worktree in parallel. Shared Go/build/lint/vet/generator commands require root's exclusive lane grant. Root retains lifecycle, review, integration, commits and completion. This correction supports source closure without admitting dependent simulation/storage tasks or skipping existing acceptance. No native execution, CI, PR or push is authorized.
