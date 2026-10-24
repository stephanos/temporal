---
satisfies: [R2, R5, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.2 Extract gomad-evidence crate with boundary tests

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Description
Move gomad/src/evidence/ to rust/gomad-evidence. lib.rs contract doc (owns / refuses), #![warn(missing_docs)], tests/ boundary tests over store open/publish, symbolize, inspection via public API; prior inline tests moved or kept with named invariant. Early proof: first hour, use-path edits only, cargo test --workspace green; if more is needed stop and report. List every pub added for a test with reason (target zero). Own branch, merge before .3. Does not overlap fn-232 (no wasmhost touch). R11: the extracted crate's public functions return its own error enum (no Result<_, String> in pub items; map_err to_string chains become From impls); typed error kinds and documented refusal behavior remain tested; message text and goldens may change. R12: remove the #[allow(too_many_lines|cognitive_complexity)] on any function this task rewrites and update .flow/artifacts/fn-240/lint-allows.md.

## Acceptance
cargo test --workspace and e2e fixture suite pass unchanged; cargo test -p gomad-evidence runs boundary tests in isolation; contract doc present; pub-for-test list recorded in .flow/artifacts/fn-240/gomad-evidence.md with lines/build-time/public-surface measurements. No pub Result<_, String> in the crate; allow-list did not grow.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
