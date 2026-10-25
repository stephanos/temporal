---
satisfies: [R2, R7]
---
# fn-251-one-probe-engine-and-one-catalog-build.2 System::generate replaces build-configuration layers

## Description
Implement **Build configuration (B10)**: `umpire_catalog_build::System{..}.generate(descriptor, out_dir)`; remove `services::Config`'s derived fields, the `temporal/api/{services,wire_schema,codegen}.rs` and `annotate/syntax/src/services.rs` wrappers, and the duplicated descriptor build in `realization/macros/build.rs`.

## Acceptance
- [ ] Kettle and Temporal build through `System::generate`.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
`System { service, prefix, rust_root, request_roots, catalog_root }.generate(descriptor, out_dir)` replaces services::Config, wire_schema::Config and the Temporal/Kettle wrapper layers; Kettle's build.rs is one call; the Temporal descriptor build lives once in temporal/api/system.rs. services.rs and descriptor.rs now read `proto::Graph`, so production code decodes descriptors only in proto.rs. Review SHIP; 5335295d27 makes root rebasing path-boundary safe, refuses numberless enum values and adds malformed-descriptor and method-order tests. Generated code byte-identical. Follow-ups for .3+: realization macros' dependency on catalog-build, shared System::services/wire_schema helpers for tests, rust_root validation, clearer generate errors.

Joined validation completed at 1c1e7fbf0767088a4c820adb64da75325af1f007: `UMPIRE_APALACHE=require make model-validate` with TLC required exited 0. All named stages, guest freshness, workspace tests, wasm controls and all three Go host modules passed. Gate 1549.246s exceeds the 60s soft target; inner loop 7.623s meets 10s. This was a cold task-owned cache with concurrent independent work, not a controlled performance comparison. Full log: `.flow/tmp/milestones-resume-20261024/joined-model-validate.log`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1c1e7fbf0767088a4c820adb64da75325af1f007
- Tests: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-milestones-20261024/joined CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 CARGO_BUILD_JOBS=4 UMPIRE_APALACHE=require make model-validate
- PRs: