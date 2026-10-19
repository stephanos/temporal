---
satisfies: [R6]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.19 Claims as check! statements; bare #[within(..)]

## Description
**Owner decision (2026-10-18):** invariants and postconditions are methods with no return type whose body makes `check!` statements, as Scala's `check`; a claim holds on a state or edge iff no check failed. `check!(e)` expands to `if !(e) { ::umpire_dsl::claim::fail(stringify!(e)) }` and records the failure (expression text) in a per-evaluation recorder that the generated predicate reads after calling the method; `unknown!("reason")` records a `Hole` the same way. No panics and no unwinding, since the wasm guest is `panic=abort`; a claim method stays an ordinary method callable from a unit test (the recorder is simply empty there). Counterexamples name the failing check: `fails_over_at_boiling: check!(self.beeping) failed at Model { .. }`. Also: `#[within(..)]` takes the bare scope name (`#[within(never_switched_off)]`), since `#[machine]` resolves it against the `#[scopes]` block; `#[after(..)]` keeps its variant path.

```rust
#[invariants]
impl Model {
    fn beeps_only_when_off(&self) {
        if self.beeping {
            check!(!self.heating_on);
        }
    }
}

#[postconditions]
impl Model {
    fn warms_only_when_on(&self, before: &Self) {
        if !before.heating_on {
            check!(self.temperature <= before.temperature);
        }
    }
    #[after(Heater::Failover)]
    fn fails_over_at_boiling(&self) {
        check!(self.temperature == BOILING_TEMPERATURE);
        check!(self.beeping);
    }
}

#[progress]
impl Model {
    #[within(never_switched_off)]
    fn boils(&self) -> Progress { whenever(self.heating_on).eventually(self.beeping) }
}
```

Do: `check!`/`unknown!` and the recorder in `umpire-dsl` (`claim.rs`); the generated invariant and postcondition predicates in `dsl/macros/src/machine.rs` wrap each call with the recorder and map it to `Verdict`; refuse a claim method with a return type (`invariants and postconditions return nothing; use check!`) and `#[within(Self::..)]` with a message naming the bare form, each pinned by a trybuild `.stderr` fixture; rewrite Kettle's claims and the `machine` macro fixture; the counterexample text gains the failing check (the only permitted golden change, in `report.txt`/`paths/` of the attribute Kettle and the checker's refusal/hole fixtures that use the attribute surface); `kettle_trait` and its goldens unchanged; DESIGN.md notes the shape and the recorder in one sentence each.

**Touches:** `model/umpire/dsl/**` (`claim.rs`, macros), `model/umpire/checker/**` (reading the recorder, counterexample text), `model/umpire/tests/fixtures/kettle/**`, `model/DESIGN.md`.

**Feedback loop (R30):** paste `make model-time` into the done summary.

## Acceptance
- Invariant and postcondition methods have no return type and use `check!` / `unknown!`; a claim holds iff no check failed; the recorder never panics or unwinds, and the wasm guest's behaviour is unchanged.
- `#[within(..)]` takes the bare scope name; `Self::` is refused with a message naming the bare form; a claim method with a return type is refused; both pinned by trybuild `.stderr` fixtures.
- Kettle's claims and the `machine` macro fixture use the new form; the only golden change is the failing-check text in counterexamples; `kettle_trait` and its goldens are byte-identical.
- A claim method called directly in a unit test behaves as an ordinary method.
- DESIGN.md records the shape and the recorder; the full local gate passes within R30.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
