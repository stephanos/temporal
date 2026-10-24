---
satisfies: [R1]
---
# fn-245-gomad-typed-boundaries-for-the-rust-host.4 Engine state machine without Option unwraps

## Description
wasmhost engine.rs has 113 unwrap/expect sites in production, most on optional state (self.model.as_ref().unwrap(), self.namespace.as_ref().unwrap(), self.broker.take().unwrap(), host_nanos.as_mut().unwrap()) that is only Some after a phase transition. Replace the Option fields with a typed phase (e.g. Prepared/Running/Terminal variants or a single accessor returning HostError) so the invariant is in the type and a violation is a classified infrastructure outcome, not a panic that kills the helper mid-evidence. Scope engine.rs and the linked_* helpers only (fn-240.8's wasm-engine); volume/state.rs unwraps (252) are index invariants on internal maps and are out of scope unless the audit shows input-dependent ones.

## Acceptance
No unwrap/expect on Option state fields in crates/wasm-engine production code; invariants enumerated in lib.rs; a fixture that drives an import before preparation yields a classified outcome, not a panic; helper corpus and linked tests unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
