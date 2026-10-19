---
satisfies: [R1]
---
# fn-214-progress-vocabulary-on-kettle-scopes.1 Scope: only and fair; fairness explicit, never implied by #[internal]

## Description
`Assumption` (renamed from `Scope`, with `#[assumptions]` replacing `#[scopes]`, owner 2026-10-19) in `umpire-dsl` gains `only([..])` and `fair([..])` beside `without([..])`, chaining, over actor-level action values; `#[machine]` maps the lists into `<mod>::Action` (as .6 does for `without`). `allows` = only minus without (only wins when both present), `is_fair` from the fair list. Nothing is fair by default: remove the checker's `world.internal(action)` fairness and use `scope.is_fair` (`Machine::internal` stays, meaning only "not driven by the environment"). Unknown actions in an assumption are refused naming them (compile-time where `#[machine]` sees them, `Refusal` at runtime otherwise); an empty admitted set is reported as an empty assumption. Kettle's goldens must not change.

**Prerequisite:** fn-210.19 done (bare `#[within]`, nested surface). **Touches:** `model/umpire/dsl/src/assumption.rs` (renamed from `scope.rs`), `dsl/macros/src/machine.rs`, `checker/src/progress.rs`, checker fixtures. **Feedback loop (R30):** paste `make model-time` into the done summary.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
