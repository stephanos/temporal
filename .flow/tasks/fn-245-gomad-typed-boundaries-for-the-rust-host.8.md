# fn-245-gomad-typed-boundaries-for-the-rust-host.8 Snapshot tests with insta instead of hand-rolled corpus regeneration

## Description
Complexity review item 5 (owner, 2026-10-24). Replace hand-rolled golden regeneration (`GOMAD_HELPER_CORPUS_REGENERATE`, `GOMAD_CLUSTER_NETWORK_FIXTURE_REGENERATE`, `GOMAD_BROKER_ORDERS_REGENERATE`) and the Rust literal corpora (~9.5k lines of `*_literals.rs`/`*_corpus.rs` at 2026-10-24) with `insta` snapshots (pinned `=`, dev-dependency). Expected outputs move into snapshot files and are reviewed with `cargo insta review`. Skip corpora that fn-249 deletes. Keep corpora whose inputs are themselves the contract as data files, not snapshots.

## Acceptance
- [ ] No custom `*_REGENERATE` switch remains; converted corpora use `insta` snapshots; `cargo test --workspace` green.
- [ ] Before/after Rust test line counts recorded with revisions.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
