---
satisfies: [R42]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.16 Name the wasm ABI explicitly and remove export rewriting

## Description
Rename the application exports to `umpire_buffer_alloc`, `umpire_buffer_release` and `umpire_call`, using direct Rust exports on the stock toolchain. The owner approved these names after the ABI explanation on 2026-10-18. Remove the obsolete linker wrapper, export-table rewrite implementation and rewrite-only tests/configuration. Retain actual compiled-module export/signature validation and its negative controls.

This is a transport naming change. Preserve protobuf messages and method IDs, the packed length/pointer return, live-allocation ownership, size limits, typed refusals, trap invalidation and the same native/Go report and row goldens. Update Go lookups, diagnostics, test controls and ABI/host/workspace/DESIGN documentation. Rebuild the real guest and prove host parity with the direct exports. Remove documentation describing export rewriting as part of the Rust recommendation.

**Touches:** Makefile, model2/.cargo/config.toml, model2/umpire/wasm/**, model2/umpire/proto/README.md, model2/hosts/go/**, model2/scripts/**, model2/README.md, model2/DESIGN.md

**Scheduling:** runs alone after .5 closes and before .15 moves the workspace. Shared Makefile and build configuration make this an always-serial lane. No dependency, schema, generated protobuf or Scala changes. Root owns MILESTONES and Flow closure.

**Quick commands:** `make model2-check`; `make model2-time`; `cd model2 && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked`

**R30:** capture the actual warm timing output. Inner loop at most 3 s and full gate at most 30 s; fix a breach or obtain an owner exception. Do not claim historical module size or host latency measurements describe the new binary; record its new size and identify older measurements honestly until .7 refreshes them.

## Acceptance
- The actual built guest directly exports the three approved names with the existing signatures. Old alloc/free/call and umpire_free application exports are absent.
- No custom linker or post-link export rewriting remains; direct locked Cargo guest build succeeds.
- Go resolves the new names, rejects missing/unexpected/wrong-signature exports, and all allocation/refusal/trap/parity controls pass with unchanged report and row goldens.
- The full gate and warm timing budgets pass; documentation describes the direct ABI and removes the obsolete workaround.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
