---
satisfies: [R2]
---
# fn-174-gomad-wasm-helper-local-host-call-fast.2 Shared binary WASI transcript records in Go and the helper

## Description
Introduce one versioned binary WASI transcript record format, written by Go's environment (every profile), by Go's remote-reply builder, and by the helper; the digest is SHA-256 over the record bytes (R2 format half). Split from the JSON-free local path (task 3) so the format and both owners switch together while differential parity holds.

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasi/transcript.go` (new: encoder, strict decoder, reservations), `tools/gomad3/backends/wasm/wasi/transcript_test.go` (new), `tools/gomad3/backends/wasm/wasi/environment.go`, `tools/gomad3/backends/wasm/wasi/helper_model.go`, `tools/gomad3/backends/wasm/wasi/identity.go`, `tools/gomad3/backends/wasm/wasmhost/src/transcript.rs` (new), `tools/gomad3/backends/wasm/wasmhost/src/model.rs`, `tools/gomad3/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad3/backends/wasm/wasmhost/protocol.md`
**Touches:** [tools/gomad3/backends/wasm/wasi/transcript*.go, tools/gomad3/backends/wasm/wasi/environment.go, tools/gomad3/backends/wasm/wasi/helper_model.go, tools/gomad3/backends/wasm/wasi/identity.go, tools/gomad3/backends/wasm/wasi/integration_test.go, tools/gomad3/backends/wasm/wasmhost/src/transcript.rs, tools/gomad3/backends/wasm/wasmhost/src/model.rs, tools/gomad3/backends/wasm/wasmhost/src/engine.rs, tools/gomad3/backends/wasm/wasmhost/src/main.rs, tools/gomad3/backends/wasm/wasmhost/protocol.md]

### Approach
- Specify the record format in `protocol.md` (replace "canonical WASI JSON lines"): a stream header with magic/version, then tagged records per operation carrying call ID, typed inputs, errno and typed outputs. Large payloads (fd_write data, fd_read output) are length-prefixed raw bytes. Leave tag space for the clock run record (task 4) and derived-payload records (task 5).
- Go: one encoder module used by `Environment.handle` (`environment.go:263-353`, retained and digest modes at `:333-350`) and `helperCallbackReply` (`helper_model.go:77-106`). The decoder refuses truncation, trailing bytes, unknown tags and oversize lengths with the existing refusal classes and no hand-written JSON scanner. Re-derive `responseReservation` (`environment.go:354-397`) as binary worst-case bytes.
- Rust: one `transcript` module that every producer calls; replace the `format!` call/reply lines in `State::clock_time_get` (`engine.rs:80-120`), `Model::handle_input`/`clock_time_get` (`model.rs:320-463`) and the reservation constants (1028/1029, poll `4*input_len`) with the same per-record worst-case rules as Go. Remote splices (`Model::remote`, `model.rs:276-319`) hash Go's binary bytes unchanged. Keep `remote_transcript` chunking and the 8 MiB bound.
- Add the new Go file to the `//go:embed` and source list in `identity.go:17,33` so codec changes rotate the implementation identity.
- Update the JSONL-parsing test helper at `integration_test.go:4491` (op histogram) to use the decoder.
- Coordination: fn-153.20 rewrites `helper_model.go`/`identity.go`; start from its landed version (this spec depends on fn-153).

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasi/environment.go:263-397`
- `tools/gomad3/backends/wasm/wasi/helper_model.go:77-229`
- `tools/gomad3/backends/wasm/wasmhost/src/model.rs:234-463`
- `tools/gomad3/backends/wasm/wasmhost/src/engine.rs:80-268`
- `tools/gomad3/backends/wasm/wasi/integration_test.go:268-281` (`sameEvidence`) and `:1313`

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml --release --locked
make -C tools/gomad3/backends/wasm helper
(cd tools/gomad3 && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestTranscript|TestHelperModel|TestHelperBridge|TestCaptured|TestEnvironment|TestOffered' ./backends/wasm/wasi)
```

## Acceptance
- [ ] Go environment (all profiles), Go remote replies and the helper write the same binary records; `TestHelperModelPreservesGoOwnerEvidence` passes byte-identically, in retained and digest-only modes.
- [ ] Go decoder unit tests cover round trip plus refusal of truncation, trailing data, unknown tag and oversize length.
- [ ] Capacity boundary parity: a retained-transcript limit hit at the same call on the Go-owner and helper-owner paths (new test).
- [ ] `identity.go` binds the new codec source; protocol.md documents the record format and reservation rules.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
