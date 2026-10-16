---
satisfies: [R1, R2, R6]
---
# fn-188-gomad-wasm-determinism-boundary-audit.2 Complete the site classes: runtime clock/draw patches, helper source scan, Go archcheck rule, engine settings, seed findings

## Description
Extend the completeness tests to the remaining site classes from the spec. (a) wasip1 runtime rows in the native `reviewedDrawReferences`/`reviewedHostClockReferences` row grammar, derived from the WASM overlay patch list (`overlay.go` string patches: runnext, mrandinit seed, cheaprand, select, timer rand, nanotime1/walltime1) and from every host-time read in `runtime_wasm.go.txt`; the native tables are read for grammar only, not edited. (b) Helper source-scan test over `wasmhost/src` for `Instant::now`, `SystemTime`, `/dev/urandom`, `thread::spawn` and hash-map iteration; every hit must be an inventory row or in an allowlist whose entries are themselves rows (diagnostic.rs, metrics in broker/cluster/engine, cache.rs temp-file names). (c) archcheck extension for the Go host packages (`wasi`, `cluster`, `backend`): `time.Now`, goroutine fan-out and map ranging that feeds evidence must be rows; current exempt files (process.go, broker_process.go, cluster.go) get rows instead of blanket exemption where feasible. (d) Every Wasmtime `Config` setting set in engine.rs (consume_fuel, native_unwind_info, cranelift_nan_canonicalization, wasm_relaxed_simd, memory64, multi_memory, threads feature off, epoch_interruption under guest-profile) has a row, and the test fails if the engine sets a setting without a row. (e) Dispose of the seed findings with rows and recorded decisions: host-timed `enlistWorker` draw in the GC pacer under wasip1; GC `LastGC`/`PauseEnd` stamps under wasip1; `cputicks`, tracer snapshot and FIPS `monoTime` under wasip1; one OS thread per cluster instance under the baton; map range feeding volume evidence in cluster/provider.go before sorting; `sort.Slice` keys; serde_json map ordering; SIMD with NaN canonicalization. A finding that needs a semantic fix is referred to a follow-up spec in the row, not fixed here.

**Size:** L
**Files:** tools/gomad/internal/backends/wasm/determinism/*, tools/gomad/internal/backends/wasm/toolchain/{overlay.go,overlay_test.go,runtime_wasm.go.txt} (read), tools/gomad/internal/backends/wasm/wasmhost/tests/source_scan.rs, tools/gomad/internal/backends/wasm/architecture_test.go, tools/gomad/internal/archcheck/*, tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs (read)

## Acceptance
- [ ] Overlay clock/draw patches and runtime host-time reads each have a wasip1 row in the shared grammar; a new patch without a row fails
- [ ] Helper source scan passes only when every time/entropy/thread/hash-map site is a row or an allowlisted row
- [ ] archcheck rule covers the Go host packages; exemptions are rows
- [ ] Every Wasmtime Config setting is a row; an unlisted setting fails
- [ ] Each seed finding in the spec has a row with a recorded decision or follow-up referral

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
