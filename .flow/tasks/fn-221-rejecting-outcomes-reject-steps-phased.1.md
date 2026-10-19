---
satisfies: [R1]
---
# fn-221-rejecting-outcomes-reject-steps-phased.1 Outcomes and reject!: Rejection, Outcome, the outcome frame, macro edge rules and outcome-taking postconditions

## Description
Add `Rejection`, `Outcome`, `outcome::{evaluate, reject}` and the `reject!` macro to `umpire_dsl`, mirroring `claim::{evaluate, fail}` and `check!`. `#[machine]` wraps each `#[on]` handler call in `outcome::evaluate`, applies the four-row edge table from the spec, selects `type Outcome` by the presence of `reject!` in the module, and lets a `#[postcondition]` take `outcome: &Outcome`. The checker refuses a rejecting step that changed the state, naming the step.

**Size:** M
**Touches:** [model/umpire/dsl/src/**, model/umpire/dsl/macros/src/machine.rs, model/umpire/dsl/macros/tests/**, model/umpire/tests/fixtures/**, model/umpire/checker/src/**]

### Approach
- New fixtures under `model/umpire/tests/fixtures/`: accepted, disabled, rejected self-loop, rejecting helper, and a rejecting-then-mutating control that expects the checker refusal. Rows and report spell `outcome=Rejected(NotFound)`.
- Existing goldens and the macro expansion snapshot for modules without `reject!` must not change; add a snapshot for a module with it.
- Doc comments state the helper and out-of-frame behavior.

## Acceptance
- [ ] R1 fixtures pass natively; rows/report spell outcomes; rejecting-then-mutating is a named checker refusal.
- [ ] Kettle and every pre-existing fixture golden and the existing macro snapshot are byte-identical.
- [ ] `make model-check` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
