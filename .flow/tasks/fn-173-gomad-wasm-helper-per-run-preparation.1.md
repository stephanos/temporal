---
satisfies: [R1, R2]
---
# fn-173-gomad-wasm-helper-per-run-preparation.1 Disable native unwind info and rotate the engine configuration revision

## Description
Turn off native unwind info in the helper engine, rotate the configuration revision consistently across the helper, cache identity, Go engine identity, tests and docs, and remove the `mem::forget` teardown workaround once the exit bound holds without it (R1, R2).

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad3/backends/wasm/wasmhost/src/cache.rs`, `tools/gomad3/backends/wasm/wasmhost/tests/engine.rs`, `tools/gomad3/backends/wasm/wasi/process.go`, `tools/gomad3/backends/wasm/wasmhost/protocol.md`
**Touches:** [tools/gomad3/backends/wasm/wasmhost/src/engine.rs, tools/gomad3/backends/wasm/wasmhost/src/cache.rs, tools/gomad3/backends/wasm/wasmhost/tests/engine.rs, tools/gomad3/backends/wasm/wasi/process.go, tools/gomad3/backends/wasm/wasmhost/protocol.md]

### Approach
- Add `native_unwind_info(false)` to the engine `Config` (`engine.rs:456-462`) and to the test engines (`engine.rs:676-678`, `cache.rs:658`).
- Replace the three Rust literals (`engine.rs:318` authorize check, `engine.rs:550` `started`, `cache.rs:130` cache identity) with one constant and a new revision name (for example `cranelift-fuel-nan-canonical-nounwind-v2`). Update Go `PinnedEngine()` (`process.go:217`), the hard-coded authorize in `tests/engine.rs:78` and protocol.md examples (`:34`, `:44`) plus the "configuration revision binds" paragraph (`:235-250`).
- Do NOT edit `backend/testdata/premove-stock-artifact/**` (pinned historical negative fixture) or the legacy `.cwasm` poison name at `tests/engine.rs:492`.
- Extend `cache_identity_binds_helper_source_platform_and_engine_settings` (`cache.rs:649`) so an engine built with native unwind info yields a different identity. Deserialize does not check the unwind flag on non-Windows, so the manifest identity is the only guard.
- Remove `std::mem::forget((store, linker, module, engine))` (`engine.rs:427`) if a release helper exits within 1 s after the terminal frame on the success path and on a post-load error path (authorization mismatch), measured with the Temporal `./tests` module on darwin/arm64. Otherwise keep it with a why-comment and record the numbers.
- Gate: land inside the fn-153/fn-154/fn-166 rotation candidate or right after it, not as a standalone requalification. Coordinate with fn-172 task 1, which also centralizes this constant.

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasmhost/src/engine.rs:273-470`
- `tools/gomad3/backends/wasm/wasmhost/src/cache.rs:86-134`, `:649`
- `tools/gomad3/backends/wasm/wasi/process.go:210-220`, `:653-690`

**Optional:**
- `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/wasmtime-47.0.3/src/config.rs` (`native_unwind_info` docs)

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml --release --locked
make -C tools/gomad3/backends/wasm helper
(cd tools/gomad3 && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestGuestTransport|TestTransport|TestProvenanceRejectsChangedIdentity' ./backends/wasm/wasi ./backends/wasm/backend)
grep -rn 'cranelift-fuel-nan-canonical-v1' tools/gomad3 --exclude-dir=target | grep -v premove-stock-artifact | grep -v '\.cwasm'
```

## Acceptance
- [ ] One Rust revision constant; the old revision string remains only in the pinned premove fixture and the legacy poison filename.
- [ ] Authorization naming the old revision is refused before instantiation (test); traps, fuel exhaustion and capacity outcomes classify unchanged.
- [ ] Exit within 1 s after the terminal frame on the success path and one post-load error path is measured on darwin/arm64; `mem::forget` removed, or kept with a comment and recorded numbers.
- [ ] Rust and Go focused tests pass; protocol.md names the new revision and unwind setting.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
