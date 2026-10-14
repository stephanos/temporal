---
satisfies: [R5]
---
# fn-173-gomad-wasm-helper-per-run-preparation.4 Compiled-artifact envelope v3 with a CRC-32 integrity check

## Description
Move the compile cache to a new envelope format whose artifact integrity check is hardware-accelerated CRC-32. Earlier-format entries are ignored and preserved (R5).

Performance measurements follow the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml`, `tools/gomad/internal/backends/wasm/wasmhost/Cargo.lock`, `tools/gomad/internal/backends/wasm/wasmhost/src/cache.rs`, `tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs`, `tools/gomad/internal/backends/wasm/wasmhost/protocol.md`, `tools/gomad/internal/backends/wasm/README.md`, `tools/gomad/internal/backends/wasm/CLI.md`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml, tools/gomad/internal/backends/wasm/wasmhost/Cargo.lock, tools/gomad/internal/backends/wasm/wasmhost/src/cache.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/protocol.md, tools/gomad/internal/backends/wasm/README.md, tools/gomad/internal/backends/wasm/CLI.md]

### Approach
- Add `crc32fast = "=1.5.2"` as a direct dependency (already in `Cargo.lock` via `object`; no new package). Confirm with `cargo tree`.
- New `MAGIC` (`GOMADC03`), header with a u32 CRC-32 replacing the 32-byte SHA-256, manifest `"version":3`, filename suffix `.gomad-cache-v3` (`cache.rs:15-18`, `:95-134`, `:251`, `:293-309`, `:370-377`). v2 and `.cwasm` entries are never opened and stay on disk.
- Truncated, oversized, mis-identified or checksum-failing v3 entries stay a host failure before `started` (`corrupt_entry_fails_closed_as_infrastructure_naming_entry`).
- Update tests that hard-code layout: `.gomad-cache-v2` (`cache.rs:475`, `:553`; `tests/engine.rs:412`, `:575`), offsets (`tests/engine.rs:418`, `:582-583`; `cache.rs:478`). Add a v2-entry-preserved test.
- Throughput: an ignored Rust test that checks ≥ 1 GiB with the CRC and prints GB/s; record the linux/arm64 result in the task summary (≥ 2 GB/s).
- Docs: protocol.md cache section (`:200-227`), README (`:148-155`), CLI.md (`:345-355`).
- Gate: land with fn-173 task 1's rotation, inside or right after the rotation-batch candidate.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasmhost/src/cache.rs:1-410`
- `tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs:344-600`

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked -- --ignored crc_throughput --nocapture
cargo +1.94.1 tree --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml -i crc32fast
```

## Acceptance
- [ ] `Cargo.lock` gains no new package; the envelope uses CRC-32 under a new magic, version and suffix.
- [ ] v2 and legacy entries are ignored and preserved; corrupt v3 entries fail closed before `started` (tests).
- [ ] Measured CRC throughput ≥ 2 GB/s on linux/arm64, recorded.
- [ ] protocol.md, README and CLI.md describe the v3 envelope and trust model.

## Done summary
R5 accepted after task2: GOMADC03/version3/.gomad-cache-v3 uses CRC-32; direct crc32fast1.5.2 adds one edge to an already locked package, no new package. Default/diagnostic and engine controls retain v2/legacy preservation and corrupt/truncated/oversized/misidentified fail-closed behavior before started. Protocol, README and CLI describe the same-UID trusted cache owner and corruption-not-forgery guarantee. Verified owned-buffer deserialization remains; no file-backed mapping was introduced.

Fresh clean50c0dec143 Linux/aarch64 ignored CRC test processed1,073,741,824bytes in0.045142s, reporting23.786GB/s against≥2GB/s, terminal0. Evidence gomad-crc-qualified-20261014.FFOu3a binds CPU/boot/compiler/source before/after and retains exact command/log/hash. Prior APuleL attempt failed preflight and ran no benchmark. This measurement reran only the ignored throughput test; integration coverage is supplied by the joined post-rename qualification, transparently retaining original driver exit1 at a doc-only guard and separate continuation exit0. Executable WASM source remains unchanged through11b54c7161.

Formal fn173.1–.4/R1–R5 fanout ea61c8cf8806fbfb5841f56de511f991 finalized SHIP on all three axes with no findings/unaddressed; receipt .flow/tmp/impl-review-fn1731to4-current-20261014.json, actual gpt-6.1-sol/high, primary01a13b87-6885-71c3-ac6a-0c88a5fcc417. Same-family/read-only review checked actual retained evidence. Task5 warm comparison remains unrun and the spec remains open; native execution deferred.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7963bb8473b6df17ebeff7c3bb78714293cc6666, 9e0d09b5e6779f78f3c2f658761956e82223f569, 75835196d7345cc0f2d9886d79851fdd3b05eeb3, 50c0dec14337d8e27aef363c51151be4fe2bc8f7, 11b54c7161
- Tests: Retained full post-rename qualification: /Users/stephan/Workspace/skunkworks/gomad-postrename-retry-20261014.TMcAas/run-qualification.sh; original terminal exit1 at documentation-only HEAD guard after passed gates, joined with reviewed /Users/stephan/Workspace/skunkworks/gomad-postrename-doc-continuation-20261014.bL7lrR/continue-qualification.sh terminal exit0; preserve raw stages/receipts/source hashes., Pinned release default and diagnostic Rust tests/all-target Clippy, ordinary Go WASM backend/cluster/wasi and actual Temporal module natural terminal/reap measurements retained in the joined qualification; native execution deferred per unchanged owners/ledger., bash /Users/stephan/Workspace/skunkworks/gomad-crc-qualified-20261014.FFOu3a/measure.sh: terminal0; cache::tests::crc_throughput 1GiB23.786GB/s on real Linux/aarch64; source/compiler/CPU/boot bindings verified before/after., Formal implementation review ea61c8cf8806fbfb5841f56de511f991, fn173.1–.4/R1–R5, finalized SHIP correctness/contracts/integration; receipt .flow/tmp/impl-review-fn1731to4-current-20261014.json; actual gpt-6.1-sol/high, same-family, zero findings/unaddressed; read-only source/retained-evidence review, no invented test execution.
- PRs: