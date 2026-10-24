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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
