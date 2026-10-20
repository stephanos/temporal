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
Completed the remaining source-derived site classes and recorded all seed-finding decisions/referrals. The JSON inventory now contains 68 exact runtime entries (17 draw/clock patches, 22 host-clock references, 29 other patches), linked bidirectionally to production-derived sites and the shared reviewed grammar. Helper source scans, Go host-effect coverage and engine-setting coverage retain their negative mutation checks. Accepted retirement preserves identity but grants no wildcard exemption to new sites; unreviewed paired appends cannot claim retirement. All 68 runtime sites retain missing/stale/disposition/rule/evidence checks, and valid new active appends remain supported. No runtime/model/protocol change. Stress evidence and the final contract remain with .5/.6.

Evidence handover: `.flow/artifacts/fn188-source-acceptance-2026-10-20.md`. Integrated source `5dcd37ceaea62e95ba7e33703e247094fea6d158`: full default validate exit 0 and affected lint exit 0, zero issues.

stage: impl-review - SHIP (official SINGLE re-review, 2026-10-20T18:15:58Z; same session 01a15a74-37ad-7020-b5d4-830052a6a1a3; codex:gpt-6.1-sol:high, same-family; .flow/tmp/impl-review-fn188-source-batch.json; both findings fixed, zero open).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5dcd37ceaea62e95ba7e33703e247094fea6d158
- Tests: Full determinism, root WASM both-platform architecture and archcheck host suites: exit 0, t2-mutation-proofs.log: helper Instant, engine SIMD, Go time.Now and overlay patch additions fail coverage, Go/Rust paired-append retirement regressions: old-code exit 1/101, repaired tests exit 0, Retired helper allowlist fresh entropy site and invalid retirement: named failures before fix, 12 Rust inventory tests exit 0 after fix, Retirement-positive active append regression: old-code exit 1, full Go determinism exit 0 after fix, Affected Go lint and focused Rust Clippy: exit 0; integrated default validate d2410c5e exit 0, make -C tools/gomad validate on integrated5dcd37ce: exit0 (inventory12/12, corpus26/26, engine3/3, Go determinism/architecture/WASI); /home/agent/wt/evidence/integration-2026-10-20/validate-fn188-controller-tagged-fixes.log, make lint-code-gomad with Linux tools, FIX=false, basee0cd113841: exit0, zeroissues; /home/agent/wt/evidence/integration-2026-10-20/lint-fn188-controller-tagged-fixes.log
- PRs: