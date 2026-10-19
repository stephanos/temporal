---
satisfies: [R6]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.18 Handlers as fn(&mut self): enabled iff the state changed

## Description
**Owner decision (2026-10-18):** a transition handler is `fn name(&mut self)` with no return type; the action is enabled on a state iff the handler changed it. `#[machine]`'s dispatch already clones `next` from `state`, so it emits the step iff `next != *state` (`State: Eq`). This replaces `-> Option<()>` with `.then(|| ..)` or `Some(())`, and it is TLA+'s stuttering semantics: an unchanged state is indistinguishable from a disabled action (relevant to fn-212's agreement). Consequence, accepted: an action that writes the same value counts as disabled on that state, so an `#[after(..)]` postcondition is not checked on that stutter; Kettle has no such action; the trait form can still express an explicit no-change step, and no escape hatch is added now.

```rust
#[transitions]
impl Model {
    #[on(Heater::Heat)]
    fn heat(&mut self) {
        if self.heating_on && self.temperature < BOILING_TEMPERATURE {
            self.temperature += 1;
        }
    }
}
```

Do: change the dispatch in `dsl/macros/src/machine.rs`; refuse a handler with a return type (`#[on] handlers return nothing; enabledness is a changed state`) with a trybuild `.stderr` fixture `machine_handler_returns`; rewrite Kettle's four handlers and the `machine` macro fixture; the canonical goldens must not change (same enabled set for Kettle); update .6's "kept" note in DESIGN.md. `kettle_trait` is untouched.

**Touches:** `model/umpire/dsl/macros/**`, `model/umpire/dsl/tests/fixtures/machine/**`, `model/umpire/tests/fixtures/kettle/model.rs`, `model/DESIGN.md`

**Integration clarification:** the macro expansion fixture reads the runtime DSL fixture at `dsl/tests/fixtures/machine/model.rs`; adapt that fixture and its values golden to the changed-state rule too. All existing refusal fixtures must continue to pin their intended error after handler signatures change. Canonical Kettle and trait-reference goldens remain byte-identical; DSL expansion/runtime fixture deltas are reviewed separately.

**Feedback loop (R30):** paste `make model-time` into the done summary.
## Acceptance
- Transition handlers are `fn name(&mut self)` with no return type; `#[machine]` emits a step iff the handler changed the state.
- A handler with a return type is refused by `#[machine]` with a message naming the rule, pinned by a trybuild `.stderr` fixture.
- Kettle's handlers and the `machine` macro fixture use the new form; all canonical goldens are byte-identical to before this task; `kettle_trait` is unchanged.
- DESIGN.md records the changed-state enabledness and its stutter consequence in one sentence each; the full local gate passes within R30.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
