---
satisfies: [R1, R7]
---
# fn-251-one-probe-engine-and-one-catalog-build.1 Shared descriptor graph builder; system-neutral #[realization]

## Description
Implement **Descriptor graph (B3)**: `umpire_catalog_syntax::proto::Graph::new(bytes, rust_root)` replaces `catalog/build/src/wire_schema.rs` `Schema` and `temporal/annotate/syntax/src/realization.rs:467-556`; remove the hardcoded `::umpire_temporal_api`.

## Acceptance
- [ ] One graph builder used by catalog/build and realization.
- [ ] `#[realization]` names no Temporal crate.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
