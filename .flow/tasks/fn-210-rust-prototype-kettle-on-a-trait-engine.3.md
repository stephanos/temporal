---
satisfies: [R3]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.3 Progress: whenever/eventually within a scope, weak fairness on internal steps, lasso counterexamples

## Description
Implement progress checking over the explored graph restricted to the scope's action set: from every state where `whenever` holds, every fair path reaches a state where `eventually` holds; weak fairness on internal steps; a refutation is a lasso (prefix plus cycle) or a dead end. Verify Kettle's `boils` within `without(DepressButton)`; refute it on a control Model whose heater never fires.

**Files:** model2/framework/checker/src/progress.rs, tests.

**Quick commands:** `mise exec -- cargo test -p framework-checker -p samples`

## Acceptance
- `boils` verified; control refuted with a lasso path.
- Unknown scope action refuses with its name.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
