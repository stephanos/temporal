---
satisfies: [R2]
---
# fn-188-gomad-wasm-determinism-boundary-audit.3 Fail-closed request, reply and engine-setting boundaries on both sides

## Description
Audit the Go-to-helper execute request and the helper-to-Go callback path as boundaries. Helper: strict decoding of the execute request refuses unknown fields and kinds before `started` with the existing `unsupported` class and a message naming the field or kind. Go: strict decoding of helper replies and callback requests (namespace_call, storage transport, network, the fn-176 split-mode remote path) refuses unknown reply kinds or fields and classifies the terminal as unsupported; no refusal after guest code ran. Engine settings: the Go side pins the expected engine configuration revision and refuses a helper whose `started` frame reports a setting outside the pin. Add tests in Rust (`tests/engine.rs` or the fn-187.1 transcript corpus once it exists) and Go (`wasi`) for each refusal, including the split-mode remote reply. Keep message shapes for the existing refusals (`unknown import m::n`, `operation outside stock WASI profile: <op>`) unchanged. Each new refusal is an inventory row with disposition `rejected`.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/wasmhost/src/{protocol.rs,engine.rs,namespace_call.rs,namespace_transport.rs}, tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs, tools/gomad/internal/backends/wasm/wasi/{process.go,helper_model.go,cluster.go,*_test.go}, tools/gomad/internal/backends/wasm/determinism/inventory.json

## Acceptance
- [ ] Unknown execute-request field or kind refused before `started` with `unsupported`, tested in Rust and Go
- [ ] Unknown callback reply kind or field refused on the Go side, including the split-mode remote path, with a classified terminal
- [ ] Off-pin engine setting in `started` refused by Go
- [ ] Existing refusal messages unchanged; every new refusal is an inventory row

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
