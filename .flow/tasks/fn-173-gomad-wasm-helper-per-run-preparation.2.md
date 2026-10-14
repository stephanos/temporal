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
R3 accepted after task1: cache lookup/deserialization returns before the module loader on hits; misses read/hash inside the per-key lock before compile/publication. No-cache runs still verify. Retained default/diagnostic Rust suites pass deleted/replaced module hits, mismatched miss/no-publication and missing-file infrastructure controls. Diagnostic hit counts for module_read/module_hash are zero. Current protocol and implementation agree.

Acceptance uses the joined post-rename qualification bound to75835196d7, with original driver exit1 at the documentation-only HEAD guard and separately reviewed continuation exit0, not a relabelled uninterrupted pass. Current WASM production source is byte-identical through11b54c7161. Formal fn173.1–.4/R1–R5 fanout ea61c8cf8806fbfb5841f56de511f991 finalized SHIP on all three axes, no findings/unaddressed requirements; receipt .flow/tmp/impl-review-fn1731to4-current-20261014.json, gpt-6.1-sol/high, primary01a13b87-6885-71c3-ac6a-0c88a5fcc417. Same-family/read-only review independently checked actual retained source and receipts. Task5 and whole-spec completion remain open; native execution deferred.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7963bb8473b6df17ebeff7c3bb78714293cc6666, 9e0d09b5e6779f78f3c2f658761956e82223f569, 75835196d7345cc0f2d9886d79851fdd3b05eeb3, 50c0dec14337d8e27aef363c51151be4fe2bc8f7, 11b54c7161
- Tests: Retained full post-rename qualification: /Users/stephan/Workspace/skunkworks/gomad-postrename-retry-20261014.TMcAas/run-qualification.sh; original terminal exit1 at documentation-only HEAD guard after passed gates, joined with reviewed /Users/stephan/Workspace/skunkworks/gomad-postrename-doc-continuation-20261014.bL7lrR/continue-qualification.sh terminal exit0; preserve raw stages/receipts/source hashes., Pinned release default and diagnostic Rust tests/all-target Clippy, ordinary Go WASM backend/cluster/wasi and actual Temporal module natural terminal/reap measurements retained in the joined qualification; native execution deferred per unchanged owners/ledger., bash /Users/stephan/Workspace/skunkworks/gomad-crc-qualified-20261014.FFOu3a/measure.sh: terminal0; cache::tests::crc_throughput 1GiB23.786GB/s on real Linux/aarch64; source/compiler/CPU/boot bindings verified before/after., Formal implementation review ea61c8cf8806fbfb5841f56de511f991, fn173.1–.4/R1–R5, finalized SHIP correctness/contracts/integration; receipt .flow/tmp/impl-review-fn1731to4-current-20261014.json; actual gpt-6.1-sol/high, same-family, zero findings/unaddressed; read-only source/retained-evidence review, no invented test execution.
- PRs: