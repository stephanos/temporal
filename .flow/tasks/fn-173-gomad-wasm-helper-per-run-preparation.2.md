---
satisfies: [R3]
---
# fn-173-gomad-wasm-helper-per-run-preparation.2 Skip module bytes on compile-cache hits and verify inside the miss path

## Description
Restructure helper preparation so a cache hit never opens or hashes the module file. A miss (and every no-cache run) reads, verifies the digest, then compiles and publishes (R3).

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/cache.rs`, `tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs`, `tools/gomad/internal/backends/wasm/wasmhost/protocol.md`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/src/cache.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/protocol.md]

### Approach
- Move the module read (`engine.rs:436-444`) and SHA-256 check (`engine.rs:449`) behind a loader that `cache::module` (`cache.rs:241`) calls only on a miss, inside the per-key lock and before compilation (`cache.rs:265-338`). The cache key already uses only the claimed digest.
- The `compile_cache: None` path (`engine.rs:470`) keeps the full read and verify.
- Outcomes: mismatch on a miss → `invalid "module SHA-256 mismatch"`, nothing published, no staging file left behind. Missing file on a hit → runs normally. Missing file on a miss or with no cache → `infrastructure`, as today.
- Diagnostic phases: on a hit, `module_read` and `module_hash` record count 0. Keep the fixed phase map.
- protocol.md: `:25` ("Input bytes are SHA-256 checked before compilation" now applies to misses), `prepare_nanos` wording (`:183-188`).
- Tests (new, in `tests/engine.rs`): a hit with the module file deleted; a hit with the module file replaced by other bytes runs the cached artifact; a miss with mismatched bytes publishes no entry; a no-cache run with mismatched bytes is `invalid`.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs:430-560`
- `tools/gomad/internal/backends/wasm/wasmhost/src/cache.rs:241-380`
- `tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs:344-600`, `:607`, `:896`

### Key context
- The `diagnostic-profile` gated spans in these blocks stay (fn-166 R8 amendment, 2026-10-13); keep them attributing the new hit/miss paths.

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --features diagnostic-profile --target-dir /tmp/gomad-wasmhost-diag
```

## Acceptance
- [ ] A hit never opens the module path (test: deleted or replaced module still runs the cached artifact).
- [ ] A miss with mismatched bytes yields `invalid` and publishes nothing; a missing file on a miss is `infrastructure`; no-cache runs still verify (tests).
- [ ] Default and diagnostic builds pass; diagnostic hit runs show zero `module_read`/`module_hash`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
