### Correctness axis

## Quality Audit — Correctness axis: fn-211.1

### Summary

- Files changed: 211 · Critical 0 · Should Fix 2 · Consider 0.
- Periodic checkpoint at `141df3cc41`; completion verdict deferred. Reviewer and writer share the GPT model family.

### Should Fix (High priority)

- **model/umpire/lift/src/safety.rs:127** (Conf 75): Walked: an overflowing first start range followed by an empty second range makes `Init` unsatisfiable, hiding the evaluated overflow. Example: `#[starts(1..=(LIMIT+1))] value:i8`, then `#[starts(0..0)] other:u8`, with `LIMIT=127`. Native enumeration evaluates the outer range first and faults — preserve an error initialization independently of later range constraints while respecting field evaluation order.
- **model/umpire/lift/src/safety.rs:327** (Conf 75): Walked: for accepted `struct Model {}`, a handler containing `stay!(); if 1/w.divisor>0 {}` with divisor zero emits the condition’s validity but never consumes it. No field update/check/outcome references that definition, so `ArithmeticSafe` remains true despite native division failure — retain evaluated condition roots in branch checks and add this control.

### Test Gaps

- Add the two fault controls above, including reversed field order to distinguish evaluated from unevaluated start expressions.
- Required full `model-time` remains pending; no gate rerun during this read-only audit.

### Test Budget

- Added Rust lines: ≤341 test versus 1,071 implementation; separate fixture file excluded.
- Existing `inputs.rs`, `phased.rs`, and `world.rs` assertions changed for annotations/declaration grouping; their intended checks remain.

### What’s Good

- Cited: retained actual-tool logs show six controls passing in 37.04s and supplemental payload/outcome/mutant coverage passing in 7.82s.
- Walked: preserving stutter, separate arithmetic metadata, exact action/history filters, and the TLC process extraction follow the intended contracts.

### Standards axis

## Quality Audit — Standards axis: fn-211.1 checkpoint `141df3cc41`

### Summary

- Files changed: 211 · Should Fix 3 · Consider 0 · Blocking: none possible (standards axis)

### Should Fix (High priority)

- **model/umpire/lift/src/snowcat.rs:22** (Conf 75): `annotate` duplicates `emit.rs` observer types and infers operator types from rendered strings with fallback guesses, creating two owners for declaration knowledge — reuse `observer_types` and attach operator types at their emission sites.
- **.flow/artifacts/fn-211-apalache-backend-for-umpire-in-rust/task1/baseline-inventory.json:1** (Conf 100): Mechanical file-size crossing, 0 → 2,789 lines — retain every entry using deterministic compact serialization or bounded manifest partitions.
- **.flow/artifacts/fn-211-apalache-backend-for-umpire-in-rust/task1/compatibility.json:1** (Conf 100): Mechanical file-size crossing, 0 → 2,246 lines — compact or partition the hash inventory while retaining its compatibility summary.

### What's Good

- **Walked:** TLC subprocess support is extracted and reused narrowly; dependencies remain local, and safety emission consumes the shared typed front end.
- **Cited:** Retained actual-tool controls and changed-crate clippy pass. The final required `model-time` remains pending; no gate was rerun during this read-only audit. Reviewer and writer use the same GPT family.

Correctness: 2 findings; worst tier Should Fix. Both accepted, contingent on discriminating red controls and supported-source confirmation.
Standards: 3 findings; worst tier Should Fix. All accepted; preserve every inventory entry and use existing typed owners rather than text guesses.

Conductor: source frozen pending normal review; add narrow shared-front evaluated-condition checks-root support to this task, without new authoring or IR constructs. Start-order correction remains in the safety emitter. Preserve exact native semantics, add reverse-order/untaken controls, recheck linearity. Front production changes invalidate prior unchanged-source guest reuse; refresh guest and actual slice/gaps if that change lands. No final full gate has run yet.

