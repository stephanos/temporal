---
satisfies: [R2, R3, R10, R11]
---
# fn-228-gomad-helper-owned-replay-tape.2 Helper preflight: accept trace frames, verify digests and counts, refuse before guest start

Touches: wasmhost/src/model/guest_state.rs, wasmhost/src/model.rs, wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasi/helper_model.go, wasi/process.go

## Description
Replace `tape_chunks` in the helper model config with a versioned replay request: `{schema, mode, limit, execution_identity, trace_sha256, tape_sha256, decisions, trace_frames}`. In `Config::validate`, read `trace_frames` binary frames bounded by `limit`, hash and compare against `trace_sha256`, project through the module from task 1, compare tape digest and decision count, and on failure return a preflight reply `{error:"replay_preflight", check}` before any instantiation. Keep `mode` 1 (record) rejecting any trace. On the Go side, `helperConfig` emits the new request and `process.go` sends the trace payload as frames instead of JSON decision chunks; unknown schema versions are refused with both versions named. Keep the per-decision comparison in `imports.rs` untouched.

## Acceptance
A replay with a tampered trace byte, a wrong expected tape digest, a wrong decision count, a frame over the limit, or a mismatched schema version fails preflight with the named check and the Runner's existing status code; a valid replay starts the guest; record mode with a trace is refused.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
