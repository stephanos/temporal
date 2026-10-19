# Use standard Rust formatting instead of Spell

The owner approved Display for Models/actions and standard Debug for their fields, including formatting-only golden changes such as Option::Some { 0: 1 } becoming Some(1).

## Acceptance Criteria

- **R1:** Remove the custom Spell trait, implementations, exports and active callers. Machine state and action values use Display; domain outcomes and facts use Debug so unit, Option, Vec and tuples work without custom implementations.
- **R2:** State/Action derives and generated aggregate actions provide idiomatic standard formatting. State fields use Debug, supporting ordinary Rust collections and custom Debug-only fields. Preserve action naming where no field-format change is needed. Formatter errors must propagate in fmt implementations.
- **R3:** Reports, counterexamples, progress diagnostics and native/Wasm row rendering use standard formatting consistently. Retain ordering, claim semantics, row structure and the distinction between model verdicts and domain outcomes.
- **R4:** Migrate manual Models, fixtures, compile-fail tests, macro expansion snapshots and current documentation. Only formatting implementations/derives may change in the independent Temporal Model; no transition, assumption, verdict or protocol semantics changes. Update formatting-only goldens and prove native/Wasm/Go parity.

## Verification

Add or adapt executable formatting coverage first and record its expected failing result before implementation. Cover Debug-only collection/nested fields, actions including the empty action enum, and formatter error propagation. Run focused DSL/checker tests, make model-check and make umpire-check-model-slice, serializing heavy Go builds. Review golden diffs for formatting-only changes.

## Scope

**Touches:** model/umpire/**, model/temporal/**, model/README.md, model/DESIGN.md (excluding generated protocol files and all model/archived/**).

This is a standalone simplification request, not permission to resume the milestone queue or change archived code, protocol schema, Temporal adapter policy, or unrelated planning files. Current branch umpire. Local commits only; no push or PR.
