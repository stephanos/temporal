---
satisfies: [R4]
---
# fn-174-gomad-wasm-helper-local-host-call-fast.5 Answer random_get, args and environ inside the helper

## Description
Carry argv, environment and the entropy key in the local model configuration and answer `random_get`, `args_sizes_get`/`args_get` and `environ_sizes_get`/`environ_get` in the helper whenever it owns the model (R4). Transcript records hold length+offset for entropy and the operation only for args/environ (spec planning decision).

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasi/helper_model.go`, `tools/gomad/internal/backends/wasm/wasi/environment.go`, `tools/gomad/internal/backends/wasm/wasi/transcript.go`, `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs`, `tools/gomad/internal/backends/wasm/wasmhost/protocol.md`
**Touches:** [tools/gomad/internal/backends/wasm/wasi/helper_model*.go, tools/gomad/internal/backends/wasm/wasi/environment*.go, tools/gomad/internal/backends/wasm/wasi/transcript*.go, tools/gomad/internal/backends/wasm/wasi/integration_test.go, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs, tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasmhost/src/transcript.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/protocol.md]

### Approach
- Extend `helperModelConfig` (`helper_model.go:15-27`, built at `:68-75`) and Rust `model::Config` (`model.rs:14-32`, `deny_unknown_fields`, validated at `:133-163`) with args, environment and entropy key; validate in Rust as `NewEnvironment` does (`environment.go:131-151`).
- Port the entropy stream exactly: byte at offset o = SHA-256("gomad3.wasi-entropy-sha256-counter/v1" || key || LE64(o/32))[o%32] (`environment.go:420-441`); `sha2` is already a dependency. Keep the 4 MiB request bound and offset-overflow capacity outcome (`entropy-bytes`).
- Keep the helper's "sizes must precede get" state (`engine.rs:68-69`, `abi.rs:251`) and output writing (`abi.rs:470-518`).
- Go-owner path (`Environment.handle`, `environment.go:398-445`) writes the same derived-payload records so parity holds.
- Golden vectors: a Go test emits entropy bytes for fixed key/offsets/lengths; a Rust unit test asserts the same bytes.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/environment.go:20-44`, `:131-151`, `:225-243`, `:398-445`
- `tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs:246-266`, `:470-518`
- `tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs:680` (`writes_host_entropy_into_guest_memory`), `:826` (`packs_captured_arguments_using_preceding_sizes`)

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked
make -C tools/gomad/internal/backends/wasm helper
(cd tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestCapturedArgumentsEnvironmentAndEntropy|TestOffered|TestHelperModel' ./internal/backends/wasm/wasi)
```

## Acceptance
- [ ] Helper-owned runs make no remote callback for these five imports (callback count test).
- [ ] Guest-visible bytes match the Go adapter for shared vectors (Rust golden test plus Go owner/helper parity on a fixture using argv, env and multiple `random_get` calls).
- [ ] Oversize/overflow entropy keeps the `entropy-bytes` capacity outcome; range errors keep their errno; get-before-sizes keeps its classification.

## Done summary
The helper answers random_get, args_* and environ_* from argv, environment and the 32-byte entropy key carried in the local model configuration, with Go golden entropy vectors shared by a Rust test; records hold length+offset (entropy) or the operation only. Range, entropy-bytes and get-before-sizes outcomes are unchanged. Impl review: SHIP.

Native execution remains deferred (fn-128/fn-149). Performance proof belongs to the .6 task.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fast-batch tip 08f92e58ee (frozen qualification candidate d0372e7a20)
- Tests: frozen qualification /home/agent/wt/fast-qual-20261014T1345: run-qualification.sh + continuations (stages.log ends PASS frozen-qualification module=gomad), make -C tools/gomad/internal/backends/wasm test-conformance, cargo +1.94.1 test/clippy -D warnings (default, diagnostic-profile, conformance)
- PRs: