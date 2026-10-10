---
satisfies: [R14]
---
# fn-159-qualified-action-names-in-rules.7 enters(<phase>) replaces trivial phase-only effects

## Description
Add a framework step, `enters(<phase>)`, that rules (and `choose` alternatives) can use directly, and delete the trivial effects that only assign a phase. A rule then reads `when[Held] ~> enters(Phase.completed)`, so its target phase shows at the rule, and `effects` keeps only steps that carry logic: decisions, extra facts, rejections.

**Size:** M
**Files:** `model/framework/Syntax.scala` (the new step; `Phased` if it needs a writer), the lifter's effect and rule lifting in `model/irgen` (`Declarations.scala`, `Expressions.scala`), irgen fixtures and tests, `model/README.md` and `model/SEMANTICS.md` if the core form is documented there, and the models with trivial effects (survey 2026-10-10): activity standalone `product/Product.scala` (10), `system/System.scala` (7), `system/Dispatch.scala` (1); nexus `product/Product.scala` (6), nexus standalone `system/System.scala` (5)
**Touches:** [model/framework/Syntax.scala, model/framework/*.test.scala, model/irgen/*.scala, model/irgen/testdata/lifts/**, model/README.md, model/SEMANTICS.md, model/temporal/features/**/*.scala, model/ir/*]

### Approach
- **Meaning:** `enters(p)` is a step function of the state alone. It takes one accepted step into the state with its phase set to `p`, recording `p.status` when the phase type is `Recorded`, and nothing otherwise. That is exactly what `effect { phase = p }` does today. Core form: `enter(s.copy(phase = p), p.status)`.
- **Writing the phase:** `Phased[S, P](_.phase)` only reads the phase. Pick one way for the framework to write it, and say why in the done summary:
  - (a) `Phased` takes a writer beside its projection, `Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p))`, for machines that use `enters`;
  - (b) `enters` uses the level's setter accessor (`phase_=`) through a `Draft`, as `effect { phase = p }` does;
  - (c) an inline/macro expansion to `effect { phase = p }`.
  
  The lifter must lift it to the same core form every time.
- **Where it's allowed:** as a rule effect (`~> enters(Phase.completed)`) and as a `choose` alternative (`retryPaused -> enters(Phase.paused)(s)`, or whatever shape the chosen design gives). If alternatives can't take it cleanly, keep the named effect for those uses and say so.
- **Migrate:** replace each effect whose whole body is setting the phase (`effect { phase = X }`, or `enter(s.copy(phase = X), <X's status>)` where the fact is exactly the phase's status) with `enters(Phase.X)` at its uses: rules, `choose` alternatives, derived machines and tests that call `effects.<name>`. Then delete the effect. Keep an effect that records any other fact, rejects, or chooses. Aliases that exist only for their name (`resetWaiting = retry`, `resetKeepsPaused = pause`) become `enters(...)` at the rule too.
- **Qualified phases:** write `enters(Phase.X)` qualified, consistent with fn-159.3.
- **IR:** rows, outcomes, facts and Cases must stay the same. Function names and content hashes may change, because rules call one shared step instead of named effects, and any `<function>$<choice>` copies change too. Check that the regenerated diff contains nothing else, and record what moved.
- **Batching (see MILESTONES.md):** the construct (framework, lifter, fixtures) lands in Batch 3b with no model migrated and the IR byte-identical. The model migration lands in Batch 5b, after fn-159.1 and fn-159.3, with the batch's single regeneration.

### Prerequisite
- A working tree that builds the lifter and passes the Cases check (the in-progress IR `Duration` migration, plus `make protoc`).

### Quick commands
```sh
make umpire-gen-model
make umpire-check-model
make umpire-check-lint
```
## Acceptance
- [ ] `enters(<phase>)` exists in the framework, is documented with its core form, and lifts to the same steps as the effect it replaces; fixtures cover a rule effect and a `choose` alternative (or the done summary states why alternatives keep named effects).
- [ ] No model keeps an effect whose whole body is setting the phase (recording at most that phase's status); effects that remain carry other facts, rejections or decisions.
- [ ] Regenerated IR keeps every row, outcome, fact and Case; differences are limited to function names and hashes, as stated in the done summary.
- [ ] `make umpire-check-model` and `make umpire-check-lint` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
