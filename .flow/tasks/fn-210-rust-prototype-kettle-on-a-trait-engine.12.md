---
satisfies: [R25, R27]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.12 Spell and Refusal: canonical value spelling in every output; misuse returns Result

## Description
Apply R25 and R27 to the code .2 delivered, before .3, .4 and .9 build on it.

- Add `trait Spell { fn spell(&self, out: &mut String); }` to `umpire-dsl` with impls for `bool`, the integer types, `Outcome`, `Option`, tuples, and (behind the `imbl` dependency when it is admitted into dsl) `imbl` maps and sets with sorted keys; records spell `Name { field: value, ... }`, enum values `Type::Variant` with fields like records.
- Require `State: Spell` and `Action: Spell` on `Machine`; implement it by hand for the trait Kettle and every checker fixture Model (derive comes in .4); delete Kettle's hand-written `Display` and every `{:?}` or `Display` that reaches an output in `umpire-testing` and `umpire-checker`'s report and path rendering.
- `Checker::explore`, `Checker::refuted` and the other questions return `Result<_, Refusal>`; `Refusal` is a `thiserror` enum (`Budget { given }`, and the variants the existing asserts guard); remove the `assert!`s and the `# Panics` docs.
- Re-render all goldens with `make model2-update`; the diff is the spelling change only (counts, order and verdicts unchanged) and is reviewed as such. A zero-budget fixture now pins the refusal text.

**Files:** model2/umpire/dsl/src/{lib.rs,spell.rs,machine.rs}, model2/umpire/checker/src/{explore.rs,report.rs,refusal.rs}, model2/umpire/testing/src/render.rs, model2/umpire/samples/src/kettle.rs, model2/umpire/checker/tests/fixtures/*/model.rs, all `expected/` goldens.

**Quick commands:** `make model2-check`; `grep -rn '{:?}' model2/umpire/testing model2/umpire/checker/src` returns nothing that reaches output

## Acceptance
- Every golden spells values through `Spell` (records with type name and braces, enums as `Type::Variant`); counts, order and verdicts unchanged.
- No `Debug` or hand-written `Display` reaches an output.
- A zero budget returns `Refusal::Budget { given: 0 }`, pinned by a fixture; no `assert!` guards API input.
- `make model2-check` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
