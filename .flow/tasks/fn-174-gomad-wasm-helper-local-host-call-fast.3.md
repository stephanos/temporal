---
satisfies: [R2]
---
# fn-174-gomad-wasm-helper-local-host-call-fast.3 JSON-free typed local model path in the helper

## Description
Make the helper's local callbacks exchange typed values between the ABI layer and the model and write binary records directly: no `serde_json::Value`, base64 payload, JSON reply string or re-parse per local call (R2 path half). Rust-only.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/src/transcript.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs]

### Approach
- Give the model typed entry points for the local operations: `clock_time_get`, clock-only `poll_oneoff`, `sched_yield` and the `runtime_*` controls. They take decoded ABI arguments or borrowed guest-buffer slices and return typed results that the ABI layer writes to guest memory.
- Runtime controls already run on raw buffers (`Model::runtime`, `model.rs:464-655`); remove the base64 encode, `json!` reply and `serde_json::from_str` round trip at `model.rs:425-444` and the `format!("runtime_{op}")` allocation (`abi.rs:700-717`).
- `poll()` (`model.rs:792-847`) returns typed events instead of a JSON string.
- Remote (Go-owned) operations keep building the JSON callback frame, since that is the protocol; only the local path becomes JSON-free.
- Keep guest range, iovec, buffer and output bounds and their errno/capacity classifications (`abi.rs:246-266`, `:468-625`).

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs:320-847`
- `tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs:196-744`
- `tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs:80-146`

**Optional:**
- `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs:1094` (ignored `profile_local_import_stages_against_json_reference`)

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked
make -C tools/gomad/internal/backends/wasm helper
(cd tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestHelperModel|TestCooperative|TestRuntime' ./internal/backends/wasm/wasi)
```

## Acceptance
- [ ] Local clock, poll, yield and runtime-control calls allocate no `serde_json::Value`, base64 string or JSON reply (checked by code review and the diagnostic `encoding` span count on a local-only fixture).
- [ ] Go owner/helper parity and replay tests stay byte-identical.
- [ ] Malformed guest buffers keep their errno/capacity outcomes (existing `runtime_imports_*` and pointer-priority tests).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
