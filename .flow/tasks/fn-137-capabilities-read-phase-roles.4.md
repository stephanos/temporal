---
satisfies: [R5]
---
# fn-137-capabilities-read-phase-roles.4 Retire Rules(projection): framework, lifter fallback, fixtures, docs

## Description
Removes the old form now that no Model uses it. The `Rules` constructor parameter and the lifter fallback are deleted, the framework and lifter fixtures move to `Phased`, and the docs describe the one form.

**Size:** M
**Files:** model/umpire/Syntax.scala, model/umpire/Rules.test.scala, model/irgen/Declarations.scala, model/irgen/testdata/{crossed/Rules.scala, objectForms/Invalid.scala, sectionOrder/SectionOrder.scala, lifts/Rules.scala, lifts/Rejects.scala, lifts/*.scala}, model/irgen/testdata/lifts/expected/*.json, model/irgen/test/Fixtures.test.scala, model/README.md, model/SEMANTICS.md, .plans/DSL_OPERATORS.md, MILESTONES.md
**Touches:** [model/umpire/Syntax.scala, model/umpire/Rules.test.scala, model/irgen/**, model/README.md, model/SEMANTICS.md, .plans/DSL_OPERATORS.md, MILESTONES.md]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Syntax.scala:266: drop the `phase` constructor parameter and its throwing default. `Rules` reads only the `Phased` given.
- Declarations.scala: delete the `parentArguments(rules)` fallback added in task 2. A `Rules(...)` with an argument no longer compiles, so the lifter needs no refusal for it.
- Fixtures: `grep -rn 'Rules(' model/irgen/testdata model/umpire` and convert each one, e.g. lifts/Rejects.scala:1044-1186 (`_.glow`) and Rules.test.scala:51, :74 (`_.light`). crossed/Rules.scala becomes the "not Phased" refusal fixture; update its header comment.
- Fixtures.test.scala:505-524 asserts line:col positions (e.g. `Rules.scala:16:30`). Recompute them, don't loosen them.
- Docs: README item 6 (:662) and the reading order (add `Phased` to the header item), :509, :891; SEMANTICS.md:171, :179; .plans/DSL_OPERATORS.md:55, :210, :230. Leave .plans/MODEL_VISUALIZATION.md:183 as is, since it is a historical design note. Keep existing comments; comments are `//` only.

#- Merge the two "the projection's phase type is not `Nothing`" evidences that coexist after the batch's integration, fn-136.4's `ProjectsPhases[P]` and fn-137.1's `PhasesOf[P, Q]`, into one (conductor, 2026-10-06).

## Acceptance
- [ ] `grep -rn 'extends Rules(' model/` finds nothing.
- [ ] The lifter's "not Phased" refusal fixture passes with recomputed positions.
- [ ] README, SEMANTICS and the DSL vocabulary describe `Phased` plus argument-less `Rules` only.
- [ ] `make umpire-gen-model` leaves model/ir and model/cases unchanged, and `make umpire-check-model` and `make lint-model` pass.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
