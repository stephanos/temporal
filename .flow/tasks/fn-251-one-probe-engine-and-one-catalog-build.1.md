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
`proto::Graph` (catalog/syntax) is the one descriptor graph: catalog/build's wire schema and the realization schema both build on it; the private walkers are gone. `#[realization]` expansion names no Temporal crate: the API root and runtime root come from the Temporal macro crate. Review NEEDS_WORK → 901d423e5e (runtime root, ambiguous short service error, strict-refusal/nested-enum/keyword tests). Generated code byte-identical; no snapshot changed.

Joined validation completed at 1c1e7fbf0767088a4c820adb64da75325af1f007: `UMPIRE_APALACHE=require make model-validate` with TLC required exited 0. All named stages, guest freshness, workspace tests, wasm controls and all three Go host modules passed. Gate 1549.246s exceeds the 60s soft target; inner loop 7.623s meets 10s. This was a cold task-owned cache with concurrent independent work, not a controlled performance comparison. Full log: `.flow/tmp/milestones-resume-20261024/joined-model-validate.log`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1c1e7fbf0767088a4c820adb64da75325af1f007
- Tests: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-milestones-20261024/joined CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 CARGO_BUILD_JOBS=4 UMPIRE_APALACHE=require make model-validate
- PRs: