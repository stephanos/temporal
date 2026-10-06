---
satisfies: [R4]
---
# fn-132-group-the-nexus-and-activity-models-by.3 Structure lint and docs learn the kind level

## Description
**Size:** M
**Touches:** [model/irgen/Structure.scala, model/irgen/Order.scala, model/irgen/test/**, model/irgen/testdata/layout/**, model/irgen/testdata/lifts/**, tools/umpire/ir/layout_test.go, model/README.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

**Required investigation:** prerequisite grouping/fixture handover; `model/irgen/Structure.scala:239`; `model/irgen/test/Fixtures.test.scala:1796`; `tools/umpire/ir/layout_test.go:324`; README layout sections and module map.
Reuse the prerequisite's classifier and validation seam. This task pins the three named R4 refusal specimens, verifies the actual moved tree and completes docs; do not reimplement grouping. Preserve independent/compound-invalid level guards and own-kind refinement boundaries. Repeat only affected checks after task 2's full batch unless an actual implementation change invalidates broader results.

fn-126's structure lint (R20) learns the kind level. A `features/<kind>/` folder holds one general feature file named after the kind, an optional `product/`, and its forms as subfolders (`workflow/`, `standalone/`), each a feature folder by fn-126's rules. A form's `system/` may refine a machine in its kind's `product/`. The general feature file holds types and signature and no machine.

Add one refusal fixture each: a form folder (`workflow/` or `standalone/`) outside a kind folder; a machine in a kind's general file; a second general file in a kind folder. The R10 layout test learns the new folders.

Docs: `model/README.md` ("Writing a Model", "Where things are") describes the kind level with Nexus as the example; `.plans/UMPIRE_MODULES.md` rows follow.

## Acceptance
- [ ] The lint accepts the tree tasks 1 and 2 produced, and each refusal fixture is refused at its line.
- [ ] `model/README.md` and `.plans/UMPIRE_MODULES.md` show the kind level.
- [ ] `make lint-model` and the layout test pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
