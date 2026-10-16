---
satisfies: [R5, R6]
---
# fn-188-gomad-wasm-determinism-boundary-audit.6 Regression gates, version pinning, validate wiring and the documented contract

## Description
Turn every escape or divergence found in tasks .1 to .5 into a named regression test (fn-187.1 transcript corpus case where end-to-end, otherwise an engine refusal test or Go host test) referenced by its inventory row; the completeness test fails when a named test disappears. Pin the inventory `reviewed` versions (Go toolchain from the WASM toolchain descriptor, Wasmtime from Cargo.lock, Rust from rust-toolchain.toml) and fail the inventory test on mismatch until the pin and a review date are updated together. Wire the inventory, source-scan, archcheck and boundary tests into `make -C internal/backends/wasm test` and `make -C tools/gomad validate`. Docs: add a determinism contract section to the WASM README listing dispositions by class, escapes with justification and the retained stress evidence, worded as a tested contract and never as elimination of all nondeterminism; in the top-level README determinism section, state for each catalogued native escape (GC stamps, FIPS monoTime, tracer snapshot, Gettimeofday, cputicks, enlistWorker) whether it applies under wasip1, cross-referencing the WASM README; update ARCHITECTURE.md where it describes the soak and diagnostics. If any .1-.5 fix rotated helper, model or runtime identities, record the rotation and rerun the WASM gates on the frozen candidate.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/determinism/*, tools/gomad/internal/backends/wasm/Makefile, tools/gomad/Makefile, tools/gomad/internal/backends/wasm/README.md, tools/gomad/README.md, tools/gomad/ARCHITECTURE.md, tools/gomad/internal/backends/wasm/wasmhost/tests/testdata/helper_transcript/**

## Acceptance
- [ ] Every escape or divergence found has a named test referenced by its row; deleting it fails the completeness test
- [ ] Inventory pinned to Go toolchain, Wasmtime and Rust versions; mismatch fails until pin and review date updated
- [ ] Inventory and boundary tests run in the WASM test target and in `make -C tools/gomad validate`
- [ ] WASM README states the contract by disposition and class; top README records wasip1 applicability per native escape; no elimination claim
- [ ] Identity rotations, if any, recorded with rerun WASM gates

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
