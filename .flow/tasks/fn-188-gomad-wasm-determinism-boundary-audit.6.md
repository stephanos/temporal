---
satisfies: [R5, R6]
---
# fn-188-gomad-wasm-determinism-boundary-audit.6 Regression gates, version pinning, validate wiring and the documented contract

## Description
**Touches:** tools/gomad/internal/backends/wasm/determinism/**, tools/gomad/docs/wasm/README.md, tools/gomad/docs/README.md, tools/gomad/Makefile and tools/gomad/rust/** for Rust receiving-gate integration with fn-232.13.

Turn every escape or divergence found in tasks .1 to .5 into a named regression test (fn-187.1 transcript corpus case where end-to-end, otherwise an engine refusal test or Go host test) referenced by its inventory row; the completeness test fails when a named test disappears. Pin the inventory `reviewed` versions (Go toolchain from the WASM toolchain descriptor, Wasmtime from Cargo.lock, Rust from rust-toolchain.toml) and fail the inventory test on mismatch until the pin and a review date are updated together. Wire the inventory, source-scan, archcheck and boundary tests into `make -C internal/backends/wasm test` and `make -C tools/gomad validate`. Docs: add a determinism contract section to the WASM README listing dispositions by class, escapes with justification and the retained stress evidence, worded as a tested contract and never as elimination of all nondeterminism; in the top-level README determinism section, state for each catalogued native escape (GC stamps, FIPS monoTime, tracer snapshot, Gettimeofday, cputicks, enlistWorker) whether it applies under wasip1, cross-referencing the WASM README; update ARCHITECTURE.md where it describes the soak and diagnostics. If any .1-.5 fix rotated helper, model or runtime identities, record the rotation and rerun the WASM gates on the frozen candidate.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/determinism/*, tools/gomad/internal/backends/wasm/Makefile, tools/gomad/Makefile, tools/gomad/docs/wasm/README.md, tools/gomad/docs/README.md, tools/gomad/docs/ARCHITECTURE.md, tools/gomad/internal/backends/wasm/wasmhost/tests/testdata/helper_transcript/**

**Re-homing (owner decision 2026-10-20, Rust host rewrite):** complete the named regression tests, inventory pins and README contract here. The `make validate` and WASM test-target wiring that lives in the Go host build is re-homed to fn-232.13 (Rust host retirement), which wires the inventory and boundary tests into the Rust binary's validate path; do not add Go-host-only gate code.

## Acceptance
- [ ] Receiving Rust gate coverage inventories the replacement Runner/CLI/evidence owners as well as the helper: clock/entropy reads, thread/commit ordering and map iteration cannot disappear merely because a host-go file is retired. Extend site classification and completeness checks to the actual Rust owners, with named behavioral tests and explicit disposition for each transferred site. Go-source checks alone do not establish Rust determinism.
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

## Handover 2026-10-23 (worker B, branch fn188-gates)
- Reconciled the 5 unresolved retired-Go evidence references (19 rows) to live receivers, without deleting rows. TestDeterminismExecutedDivergenceArtifacts maps to `campaign::counterfactual::produce_the_four_original_counterfactual_artifact_pairs` plus the executed-divergence driver control. TestCooperativeRuntimeQualification maps to `runtime_qualification` and the runtime-gate driver controls; the stream rows also get the native first-divergence test. TestTerminalExitMeasurementControls maps to the `measurement::terminal_controls` tests and the driver control. TestTransportRejectsInvalidTerminalClaims now resolves through its migration contract: invented-output and excess-fuel are `native`, received by the resealed public-intake refusals in `record_intake_tests`. TestInstalledOperatorClusterFailureEndsPromptly maps to the new helper-corpus case `manual_cluster_trap_ends_run_without_control_frame.json` (the fn-235 control reader with input left open) plus the installed-cluster-failure driver control.
- Added the `driver:tests/test_<x>.py:test_<y>` evidence kind. It resolves to a unittest method in `rust/driver/tests`, with shape and removed-reference negatives. `validate-wasm-determinism` now runs the driver suite, and `test-wasm` depends on `validate-wasm-determinism`.
- Source drift on gomad head is reconciled. Production commits 7a4e859c37 (run_observed→run_observed_using) and c8878afd57 (stop_live cleanup-grace watcher) had left the inventory red. Changes: 4 host relocations, 3 new stop_live rows, linked_cluster.rs thread::scope 1→2 (accepted-prefix retirement) and recv_timeout 3→4 (multiplicity migration). The 276-record prefix is untouched; identity-history and effect-migrations are append-only.
- Docs: the WASM README contract now covers dispositions by class, reference kinds, the retired-Go receiver map, stress evidence and "Known determinism escapes" (none accepted; the anchor already linked from ARCHITECTURE now exists). The top README adds per-escape wasip1 applicability. ARCHITECTURE stress text now describes the linked engine.
- Unchanged pins: go1.27.1 / Wasmtime 47.0.3 / Rust 1.99.1, review date 2026-10-23. No helper/model/runtime identity rotation: only tests, data and docs changed.
- Evidence: `make validate-wasm-determinism` exit 0 (inventory 40/40 including `inventory_all_evidence_references_resolve`, driver 551 OK, archcheck ok). The named receiving Rust tests pass. The new corpus case passes. Resource-gated `#[ignore]` receivers resolve but earn behavioral credit only in the frozen cycle (fn-232.13).
- Open, outside this task: the reference-helper `helper_corpus` is red on gomad head before this change, with 19 pre-existing cases plus `closed_cluster_cross_case_raw_and_evidence_guards`; this blocks `test-rust-engine`/`test-wasm`.
