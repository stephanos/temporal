---
satisfies: [R3]
---
# fn-245-gomad-typed-boundaries-for-the-rust-host.6 Replace small hand-written codecs and test wasm bytes with workspace crates

## Description
Crate amendment (owner, 2026-10-24): replace the hand-written CRC32 in `measurement/warm_driver.rs` (`crc32_update`) with `crc32fast`; replace the hand-written base64 helpers (`qualify/cells.rs` `base64`, `campaign/exploration/mod.rs` and `simulation.rs` `base64`, `campaign/scenario.rs` `canonical_base64`) with the `base64` crate, keeping strict/canonical decoding where the callers require it; replace hand-assembled wasm byte strings in tests (`measurement/*_tests.rs`, `warm_driver.rs`, `qualify/*tests.rs`, `evidence/search_context.rs`) with `wat::parse_str`. All three crates are already in `Cargo.lock` via `wasmhost`/wasmtime; move them to `[workspace.dependencies]` pinned `=`. Varint helpers are covered by fn-247.4/.6. Skip any site fn-242 deleted. Record before/after counts in `.flow/artifacts/fn-245/counts.md`.

## Acceptance
- [ ] No hand-written CRC32 or base64 codec remains in host crates; non-canonical base64 is still refused where it was before, with a test.
- [ ] Test wasm modules are written in WAT; `cargo test --workspace` green; counts recorded with revisions.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
