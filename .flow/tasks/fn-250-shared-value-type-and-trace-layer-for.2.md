---
satisfies: [R2, R6]
---
# fn-250-shared-value-type-and-trace-layer-for.2 Schema keyed by ir::Type; remove the wire Type and dead Set/Record decoding

## Description
Implement **One type system (A1)**: delete the wire `Type` (`tla/src/values.rs`), key `Schema` by `ir::Type` with one `accepts`, add emitter-owned `Module::schema()`/`safety_schema()` replacing the five derivations (tlc `module.rs`, apalache `replay.rs`, three tla tests), serve ITF `varTypes` from `tla/src/snowcat.rs` (remove `apalache/src/decode.rs` `wire_type`), and remove wire Set/Record with their TLC/ITF decoders and `default_value`.

## Acceptance
- [ ] No wire `Type`; one Snowcat printer; no Set/Record decoding code.
- [ ] Values outside their declared type refuse decoding in TLC and Apalache.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
`Schema` is keyed by `ir::Type` and built by the emitter (`Module::schema`, `Module::safety_schema`); the wire `Type`, TLC/Apalache schema builders and three test copies are gone. One `values::accepts` and one `validate_type` serve both back ends; `snowcat::ty` produces ITF varTypes; wire Set/Record decoding is removed with refusal tests. Review NEEDS_WORK → 57de37eac1: Apalache keeps refusing Models with non-unit edge facts until fn-250.3 adds a require-mode fact replay test; empty tuples refused; ITF refusals name state and field. No golden changed. Focused tests passed with TLC and Apalache required; joined gate below.

Joined validation completed at 1c1e7fbf0767088a4c820adb64da75325af1f007: `UMPIRE_APALACHE=require make model-validate` with TLC required exited 0. All named stages, guest freshness, workspace tests, wasm controls and all three Go host modules passed. Gate 1549.246s exceeds the 60s soft target; inner loop 7.623s meets 10s. This was a cold task-owned cache with concurrent independent work, not a controlled performance comparison. Full log: `.flow/tmp/milestones-resume-20261024/joined-model-validate.log`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1c1e7fbf0767088a4c820adb64da75325af1f007
- Tests: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-milestones-20261024/joined CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 CARGO_BUILD_JOBS=4 UMPIRE_APALACHE=require make model-validate
- PRs: