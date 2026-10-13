---
satisfies: [R1, R4]
---
# fn-175-gomad-filesystem-namespace-in-the-rust.6 Activity-selection gate: zero fs callbacks, >=25% faster on darwin/arm64, docs

## Description
Prove the spec on the real workload and close the docs (R1, R4): the Temporal activity selection runs with zero filesystem callbacks to Go, and authorized execution on darwin/arm64 drops by at least 25% against a same-seed pre-change baseline with identical choice traces.

**Size:** S
**Files:** `tools/gomad3/backends/wasm/wasi/integration_test.go` (assert the Go fs-served counter in the matched-suite gate), `tools/gomad3/backends/wasm/README.md`, `tools/gomad3/backends/wasm/wasmhost/protocol.md`, `tools/gomad3/ARCHITECTURE.md` (only if it names the WASI namespace owner), `.flow/artifacts/` (measurement record)
**Touches:** [tools/gomad3/backends/wasm/wasi/integration_test.go, tools/gomad3/backends/wasm/README.md, tools/gomad3/backends/wasm/wasmhost/protocol.md, tools/gomad3/ARCHITECTURE.md]

### Approach
- Baseline (decision): no retained darwin/arm64 baseline for the activity selection exists. The linux/arm64 numbers in `wasm-section-acceptance-2026-10-12.md` and the ≈ 2.3 s session profile are not usable, and fn-172 records no plain-namespace baseline. Build the helper at the fn-175 base commit (before fn-175.5) as `GOMAD_WASM_MATCHED_BASELINE_HELPER`.
  - If fn-172 has since retained a same-seed darwin/arm64 activity baseline, use it and cite it.
- Run `TestHelperMatchedSuiteQualification` (integration_test.go:4068) with `GOMAD_WASM_MATCHED_SUITE=activity` and the same module, seeds and alternating baseline/candidate order. Its `sameEvidence` and equal `Calls`/`TimeAdvances`/`EnvironmentIdentity` checks and the candidate replay of the baseline tape establish identical choice traces.
  - Note: `Calls` stays equal because local calls still consume global IDs.
- Extend that gate to require the Go fs-served counter (fn-175.5) to be 0 for the candidate.
- Record results under `.flow/artifacts/`: per-seed authorized nanos, speedup, host, helper digests and commit.
- Docs:
  - `wasmhost/protocol.md`: Go no longer "owns descriptors and files" for gated runs; namespace image frames; local fs ops; `stdout`/`stderr` streams; the 33-op table local/remote split.
  - `README.md`: model table "Default namespace" row; the "filesystem … callbacks retain canonical transcript ordering" paragraph; the directory owner table.

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasi/integration_test.go:4068-4240` (matched-suite gate and env vars)
- `tools/gomad3/backends/wasm/README.md:5-20,112-130`
- `tools/gomad3/backends/wasm/wasmhost/protocol.md:1-60`

### Key context
- Run the Temporal gate alone on the host (memory budget: cold helper up to 6 GiB during compile). It needs `GOMAD_WASM_MATCHED_MODULE`, `GOMAD_WASM_MATCHED_BASELINE_HELPER`, `GOMAD_WASM_MATCHED_CACHE`, `GOMAD_WASM_MATCHED_BASELINE_CACHE`, `GOMAD_WASM_MATCHED_SUITE=activity` and `GOMAD_WASM_PROTOTYPE_ROOT`.

## Acceptance
- [ ] Matched activity gate on darwin/arm64: candidate authorized execution ≤ 0.75 × baseline on every seed, with identical choice traces and `sameEvidence`; Go fs-served counter = 0.
- [ ] Measurement record retained under `.flow/artifacts/` with exact command, seeds, digests and numbers.
- [ ] README and protocol.md describe the helper-owned namespace, gating and new frames/streams; no stale "Go owns descriptors and files" statement remains for gated runs.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
