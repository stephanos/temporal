# fn-156 workflow residual: attribution before optimization

Read-only investigation, 2026-10-11. This note is the only repository write. No builds, guest execution, profiling, package installation, tracker changes or acceptance decisions were performed. The helper owner supplied observations and is preparing a separate current matched pair; this research does not duplicate that work.

## Recommendation

Keep the original five-suite, two-seed, authorization-to-validated-terminal 5x gate and all resource limits unchanged. The next useful evidence is a bounded diagnostic separation of host-transition work from time between transitions, followed by a small typed-versus-dynamic import characterization if that boundary is material. Do not assume the 390 ms residual is guest CPU, or implement a broad engine/configuration change based on it.

There is no overlooked global optimization-off setting: pinned Cranelift already defaults to speed optimization. Typed safe host bindings are a plausible narrower lead, not a measured cause or a prediction of 5x. Adding diagnostics changes the helper hash, so current strict cache identity prevents silently reusing another helper's compiled cache.

## Correction: final producer GOMEMLIMIT

My earlier instruction to unset `GOMEMLIMIT` for the final Temporal producer was too broad. I found no primary source/user contract requiring it. The original matched helper performance control explicitly kept it unset; preserve that condition for that comparison. The separate final producer may declare `GOMEMLIMIT=1536MiB` while still passing its actual physical 2 GiB lifetime RSS, CPU/wall, compiler/helper and host-admission gates.

This is managed-Go-memory tuning, not a hard physical cap. [Go's GC guide](https://go.dev/doc/gc-guide#Memory_limit) explains the soft limit and CPU/memory tradeoff; memory outside runtime accounting and transient overshoot still require headroom. `qualification/publication_receipt.go:42` captures actual producer usage after Result publication; `:104` enforces 2 GiB. No fabricated or pre-publication substitute follows from tuning.

Important inheritance detail: `backend/provider.go:565` preserves `GOMEMLIMIT` in compiler environments, so 1536 MiB affects compiler subprocesses too; their measured time and unchanged limits still apply. `wasi/process.go:363` launches the Rust helper with an empty environment; guest environment comes from sealed configuration. The setting does not implicitly alter guest GC. `qualification/manifest.go:19` has no explicit producer-GOMEMLIMIT field: retain exact wrapper/environment/hash alongside the guardian and receipt rather than inventing a schema or claiming it is already bound there.

## Current retained observations

Primary run directory: `.flow/tmp/fn156-helper-matched/resume-workflow-profile-v2/`. Each row below comes from the corresponding `workflow-diagnostic-seed*-metrics.json`; the diagnostic helper is `4148647a…`, DEFAULT is `60e65db4…`, immutable legacy module `daf29914…`, runtime `aa92d3e4…`, model `e0da60c2…`. These are diagnostic legacy-workload observations, not current-ABI corpus acceptance.

| Diagnostic measurement | Seed 0 | Seed 1000003 |
| --- | ---: | ---: |
| Execute wall | 592.905 ms | 571.894 ms |
| Execute helper user + system CPU | 465.952 ms | 464.417 ms |
| Residual execute wall | 391.790 ms | 389.352 ms |
| Remote ABI wall, 1,480 calls | 163.891 ms | 145.747 ms |
| Nested remote read/wait | 133.613 ms | 114.475 ms |
| Local clock ABI | 26.945 ms / 27,536 calls | 26.440 ms / 27,407 calls |
| Scheduler + select ABI | 9.612 ms | 9.483 ms |
| Instantiation wall | 48.458 ms | 54.178 ms |

DEFAULT authorization totals are 633,850,667 + 603,391,292 = 1,237,241,959 ns, versus the prior matched candidate 1,350,280,917 ns. The historical original baseline is 4,761,416,669 ns. Dividing that old baseline by this diagnostic-window DEFAULT aggregate gives about 3.848, **not a fresh matched ratio**. A fixed baseline of that size would require at most 952,283,334 ns for 5x: another roughly 285 ms / 23% reduction from the current DEFAULT aggregate. Neither measurement noise nor the diagnostic helper can supply acceptance.

The remote read/wait is nested within remote ABI: never add them. It includes waiting for Go model work, pipe/wakeup scheduling and deserialization readiness, not just Rust parsing. Even eliminating the approximately 53 ms combined clock ABI would be much smaller than the gap. Eliminating all 310 ms combined remote ABI would be an unrealistic upper bound; actual synchronous storage semantics, replies and validation cannot disappear. Cross-helper instrumented numbers are only scale estimates, not transferable speedup predictions.

`diagnostic.rs:142` computes residual as phase wall minus disjoint ABI wall. The ABI span starts at `abi.rs:204`, after Wasmtime has entered the closure; it ends before Wasmtime converts the result back. Thus residual includes guest runtime/application instructions, fuel machinery, dynamic import conversion, other engine work, instrumentation outside spans and off-CPU delay. Existing phase CPU proves the execute phase consumes substantial helper CPU, but does not allocate it among those categories. Whole-helper wait4 CPU additionally includes preparation. `wasi/helper_profile_test.go:279` profiles the Go parent around the whole Run, not the helper/JIT. Retained `/tmp/fn156-resume-parent-cpu-top.log` has 1.01 s samples, dominated by 730 ms zeroing and 180 ms SHA work before authorization; it does not support a large execution-path JSON speedup.

## Exact configuration and what is already enabled

Local owners: `wasmhost/Cargo.toml:11`, `Cargo.lock:156` and `:886`; `wasmhost/src/engine.rs:451`; `tools/gomad_wasm/Makefile:9`. Wasmtime is exactly 47.0.3, Cranelift 0.134.3, release Rust build. Enabled Wasmtime features are `cranelift,runtime,std,wat,reexport-wasmparser`; defaults are disabled. The engine explicitly enables fuel and canonical NaNs, disables relaxed SIMD, memory64 and multi-memory. The inspected module must expose one unshared memory32 (`engine.rs:511`). Fuel is set from the sealed request (`:361`). The entrypoint already uses typed `_start` (`:391`).

Pinned [Config source](https://github.com/bytecodealliance/wasmtime/blob/v47.0.3/crates/wasmtime/src/config.rs) and local registry `wasmtime-47.0.3/src/config.rs` establish: speed optimization (`:1509`), backtracking register allocation (`:1537`), lazy tables (`:2149`) and Linux COW initialization (`:2245`) default on; compiler inlining defaults off (`:2356`). Parallel compilation is a Cargo feature absent here: relevant to cold compilation, not cached execute residual. Inlining/lazy-table tuning changes generated artifacts and may trade compilation/RSS against execution; no such substitution is recommended under the frozen experiment. COW applicability to this module has not been measured. Pooling is absent and not an obvious win for one fresh process/instance. Fuel/NaN safety policy must not be disabled for speed.

## Attribution routes, in order

### 1. Deeper host-transition accounting, diagnostic only

Pinned [Store source](https://github.com/bytecodealliance/wasmtime/blob/v47.0.3/crates/wasmtime/src/runtime/store.rs), local `:965`, exposes `Store::call_hook` only with the currently absent `call-hook` feature. [Trap-handler source](https://github.com/bytecodealliance/wasmtime/blob/v47.0.3/crates/wasmtime/src/runtime/vm/traphandlers.rs), local `:266`, wraps the host closure with `CallingHost` and `ReturningFromHost`. The wrapper includes argument/result marshaling around Gomad's ABI function. It also sees Result-returning engine libcalls; simple non-Result libcalls are excluded. It is not the entire generated machine trampoline.

A bounded diagnostic can count/time transitions and associate the existing ABI span with the enclosing host interval. Separate host intervals containing a Gomad import from other engine intervals; do not mislabel extra hooks as guest work. Compute host-wrapper minus ABI time only for matched enclosing intervals. Handle traps/unwind and the outer `ReturningFromWasm` explicitly; never fabricate a missing return interval. The observer must always return `Ok(())`, never change fuel, guest memory, replies or scheduling. Use bounded aggregate counters rather than per-call output; preserve complete evidence comparisons.

Call hooks add real overhead. An enabled-but-inactive diagnostic control and timer/aggregation calibration are required before interpreting small deltas. Per-import `getrusage` would itself be substantial work; retain existing phase CPU and use transition wall for attribution, not as a CPU surrogate. Time outside the hook remains guest-plus-unhooked-transition wall, not pure guest time. A timer only around the existing `func_new` closure cannot capture Wasmtime's conversion before/after that closure; it mostly overlaps the existing ABI span.

### 2. Safe typed import characterization if boundary cost is material

All imports currently use dynamic `Linker::func_new` (`engine.rs:374`), despite a closed signature table (`abi.rs:9`). Pinned [function implementation](https://github.com/bytecodealliance/wasmtime/blob/v47.0.3/crates/wasmtime/src/runtime/func.rs), local `:2469`, performs `load_untyped_params` before the closure and `store_untyped_results` afterward. This work lies outside today's ABI span. Parameter storage is reused (`:2534`, `:2573`), so **do not claim a fresh Vec allocation on every call**. The API recommends typed `Func::wrap` because signatures are statically known (`:343`); that recommendation does not establish workload cost.

Small safe `func_wrap` adapters for the closed high-frequency signatures could retain the same ABI body using stack-local `Val` arrays, before considering any typed-body refactor. First characterize dynamic versus typed calls on bounded tiny guests with equivalent validation/trap behavior. Full qualification must retain import signature rejection, signed/unsigned bit interpretation, guest pointer and iovec checks, capacity/error precedence, callback counts, terminal status, canonical bytes, fuel remaining and exact control replay. Do not use `new_unchecked`/unchecked calls or cache guest memory slices across callbacks. The exact gain remains unknown; most residual may be real guest computation.

### 3. Parent remote-callback attribution

The 1,480 remote calls and 114–134 ms read/wait make this worth measuring, but existing parent profile mixes preparation with execution. Attribute existing parent callback handling during the authorization window and separate decode, namespace/storage model, transcript/reply encoding and blocked time before changing ownership. `wasmhost/src/engine.rs:209` reads the reply; `wasi/namespace.go:481` owns fd_write/fd_pwrite. Preserve Go storage ownership and each synchronous result. No batching, local filesystem reimplementation or deferred errors merely to reduce callback counts.

### 4. Smaller local cleanup only after attribution

Typed clock handling already exists (`abi.rs:224`, `engine.rs:81`). It still formats canonical call/reply text (`engine.rs:87`, `model.rs:331`) and flush-checks (`model.rs:236`). Reusable bounded buffers/export-index lookup are plausible modest savings, bounded by the small measured local-ABI share. Scheduler/select together are only about 19 ms across the two diagnostic records. Nested encoding/hash/flush measures overlap: optimizing or summing all of them cannot manufacture independent savings. Preserve exact stream bytes and limits.

## Sampling feasibility and cache constraint

- Wasmtime [Linux perf guidance](https://docs.wasmtime.dev/examples-profiling-perf.html) describes PerfMap/JitDump symbols. Pinned `src/profiling_agent.rs:19` supports PerfMap on Unix+std; JitDump requires the absent `profiling` feature. Read-only checks found no `perf` executable and `perf_event_paranoid=2`. That value does not itself prove user-space sampling impossible, but no working profiler permission/tool path is established. Do not alter kernel policy or install tools without separate authority. If available later, sample only the owned helper, preserve its guard and phase boundaries, and distinguish JIT guest symbols, host marshaling and kernel/wakeup cost. System profiling has its own overhead and cannot replace the uninstrumented matched result.
- Pinned [GuestProfiler source](https://github.com/bytecodealliance/wasmtime/blob/v47.0.3/crates/wasmtime/src/runtime/profiling.rs), local `:31`, requires sampling while the guest is on the same stack/thread. Ordinary sampling uses epoch callbacks; fuel sampling is biased. Neither changing epoch nor fuel control is in scope. `sample` is not async-signal-safe (`:54`), so a signal-handler shortcut is invalid. Sampling only at existing imports could reveal biased callsites, not CPU distribution within long guest stretches. `GuestProfiler::call_hook` emits host markers and captures stacks, adding more overhead than plain counters. Its `profiling` feature is currently absent.
- A new instrumented helper cannot use the existing helper's cache unchanged: `cache.rs:73` binds helper SHA, `:243` hashes the manifest into the filename, `:293` compares exact identity, `:298` verifies the artifact digest before deserialization. Store call hooks need not semantically change the guest module, but their build changes helper identity. Under the unchanged no-rebinding/no-substitution rule, budget a separately authorized fresh cold preparation for that diagnostic helper, followed by warm bounded samples. Do not copy/relabel cache envelopes or weaken validation to avoid compilation.

## Decision boundary

No measured evidence yet attributes the residual to dynamic marshaling, fuel instrumentation, Go runtime loops or a particular JIT-generated function. The practical next step is to retain the already-planned current uninstrumented pair, then choose the smallest separately authorized diagnostic that can resolve this question. If deeper instrumentation needs a cold compile that the resource window cannot accommodate, report that prerequisite rather than claim existing wrapper spans or parent pprof answer it. Every resulting optimization still needs fresh matched all-five evidence under original controls; no threshold, resource, host-speed, engine-backend or timing-boundary waiver is proposed.
