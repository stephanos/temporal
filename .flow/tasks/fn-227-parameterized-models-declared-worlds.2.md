---
satisfies: [R2, R3]
---
# fn-227-parameterized-models-declared-worlds.2 After .1: Finite and action inputs in derive(Action); action_inputs fixture with TLC

## Description
Add trait Finite with derive for fieldless enums and impls for bool and range newtypes; derive(Action) admits tuple/struct variants whose fields are Finite or carry #[domain(fn)] receiving &World; actions() enumerates the domain product in declaration order; catalog limit refusal (default 4096) names the count. Lift inputs as finite existential choice. Add shared fixture action_inputs (bool input plus a world-sized range input, one claim over the input) with full goldens and TLC agreement; trybuild pins a non-Finite field. All pre-existing goldens byte-identical.

## Acceptance
- [ ] Finite, derive(Action) inputs and #[domain] work; non-Finite field refused by trybuild
- [ ] Catalog over the limit is a Refusal naming the count
- [ ] action_inputs fixture goldens present and TLC agrees
- [ ] Pre-existing goldens byte-identical; gate and model-time pasted

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
