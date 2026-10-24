---
satisfies: [R1, R5]
---
# fn-248-off-the-shelf-crates-and-consolidation.6 Serde-derived ITF envelope in the Apalache back end

## Description
Serde-derived ITF envelope in the Apalache back end.

- ITF envelope (`model/umpire/backends/apalache/src/decode.rs` ~130-160): `#[derive(Deserialize)]` with `#[serde(deny_unknown_fields)]` for the top-level object instead of manual `keys()`/`get()` checks. The strict value visitor, pre-scan budget, `#bigint`/`#tup`/`#set` handling and schema checks stay. Enable serde's `derive` feature only if `serde_derive` is already resolved in `model/Cargo.lock`; otherwise record the admission under fn-210 R18.

The `sha2` replacement of the `shasum` subprocess moved to fn-250.5 (2026-10-24, throughput ordering): it lands once, inside the consolidated `PinnedJar::admit`, instead of being rewritten in two runners first.
## Acceptance
- [ ] The ITF envelope decodes through serde derive; unknown or duplicate top-level keys still refuse.
- [ ] `make model-check` passes (Apalache mode where available).
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
