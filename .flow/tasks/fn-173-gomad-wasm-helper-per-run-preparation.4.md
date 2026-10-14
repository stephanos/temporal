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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
