---
satisfies: [R1, R6]
---
# fn-250-shared-value-type-and-trace-layer-for.1 Literal in umpire-ir with one admit function

## Description
Implement **Literal in the IR (A2)** from the spec: move `Literal` from `umpire-tla` into `umpire-ir`, add `From<umpire_dsl::Value> for Literal`, add `Literal::admit(&self, &ir::Type) -> Result<ir::Value, Mismatch>`, and replace every listed conversion/admission copy (native `interpret_values.rs`, tla `bind.rs`, tlc `module.rs` `binding_value`, apalache `replay.rs` `typed_literal`/`literal`, testing `backend.rs` `admits`, the dsl→Literal copies in `bound.rs`, `apalache/tests/replay.rs`, `tla/tests/world_fairness_binding.rs`). Add `Binding::of` and `Concrete::of::<M>` replacing the three `concrete::<M>()` copies.

## Acceptance
- [ ] One `Literal` and one `admit`; no former copy remains (grep evidence).
- [ ] Type mismatches and out-of-range values still refuse.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
One `Literal` in umpire-ir (`ir/src/literal.rs`) with `Literal::admit(&Type) -> Result<Value, Mismatch>` replaces every listed conversion/admission copy (native import, tla bind, tlc binding_value, apalache replay helpers, testing admits/bound) plus three copies in the activity feature. umpire-dsl supplies `From<Value> for Literal` and non-wasm `ConcreteOf`/`BindingOf` traits for `Concrete::of`/`Binding::of`; `Lifted` keeps the admitted binding. Native catalog import now also checks Int range; no fixture refused. Review SHIP; follow-ups a78813f296 (tuple cfg range check, named refused fields). No golden changed. Covered by the joined `UMPIRE_APALACHE=require make model-validate` on umpire (see fn-250.2's evidence).

Joined validation completed at 1c1e7fbf0767088a4c820adb64da75325af1f007: `UMPIRE_APALACHE=require make model-validate` with TLC required exited 0. All named stages, guest freshness, workspace tests, wasm controls and all three Go host modules passed. Gate 1549.246s exceeds the 60s soft target; inner loop 7.623s meets 10s. This was a cold task-owned cache with concurrent independent work, not a controlled performance comparison. Full log: `.flow/tmp/milestones-resume-20261024/joined-model-validate.log`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1c1e7fbf0767088a4c820adb64da75325af1f007
- Tests: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-milestones-20261024/joined CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 CARGO_BUILD_JOBS=4 UMPIRE_APALACHE=require make model-validate
- PRs: