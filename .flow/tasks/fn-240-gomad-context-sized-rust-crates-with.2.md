---
satisfies: [R2, R5]
---
# fn-240-gomad-context-sized-rust-crates-with.2 Extract gomad-evidence crate with boundary tests

## Description
Move gomad/src/evidence/ to rust/gomad-evidence. lib.rs contract doc (owns / refuses), #![warn(missing_docs)], tests/ boundary tests over store open/publish, symbolize, inspection via public API; prior inline tests moved or kept with named invariant. Early proof: first hour, use-path edits only, cargo test --workspace green; if more is needed stop and report. List every pub added for a test with reason (target zero). Own branch, merge before .3. Does not overlap fn-232 (no wasmhost touch).

## Acceptance
cargo test --workspace and e2e fixture suite pass unchanged; cargo test -p gomad-evidence runs boundary tests in isolation; contract doc present; pub-for-test list recorded in .flow/artifacts/fn-240/gomad-evidence.md with lines/build-time/public-surface measurements.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
