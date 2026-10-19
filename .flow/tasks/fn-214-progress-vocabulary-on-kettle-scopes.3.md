---
satisfies: [R4, R5]
---
# fn-214-progress-vocabulary-on-kettle-scopes.3 Kettle's progress questions as fixtures; fairness and emitter contract in DESIGN.md

## Description
Kettle parity with its archived test's progress questions, as shared checker fixtures (the `tests.rs` form follows in fn-213.4): `boils` in `never_switched_off` verified with longest 81 from room temperature (20) and 141 from the coldest start (-40); on the whole kettle from room temperature a `Cycle` `depress, press` with the `only` hint; assuming `only([User::PressButton])` a `Deadlock` ending in `Model { temperature: 20, heating_on: true, beeping: false }`; `boils.within(80)` from room temperature a `MissedDeadline`; budget 10 `Unknown`. Ask the per-question variants as `Model::boils.assuming(only([User::PressButton]))` and `Model::boils.within(80)` from the fixture side, so the Model stays as in .6. Existing Kettle goldens byte-identical. `DESIGN.md` records the fairness rule and fn-212's emitter contract (`fair` entry = `WF_vars(A)`, nothing else fair) in one sentence each (R5).

**Touches:** `model/umpire/tests/fixtures/kettle/**` (new expected files only), `model/umpire/checker/**` (per-question scope override if needed), `model/DESIGN.md`. After .1 and .2. **Feedback loop (R30):** paste `make model-time` into the done summary.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
