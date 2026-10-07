# fn-143-rename-modelumpire-to-modelframework.1 Move model/umpire to model/framework, package umpire to framework; regenerate; prove the diff is the umpire.→framework. mapping; docs

## Description
**Size:** M
**Touches:** [model/**, tools/umpire/**, Makefile, .plans/UMPIRE_MODULES.md, .plans/UMPIRE4_SPEC.md, MILESTONES.md]

`git mv model/umpire model/framework`; rename the package `umpire` to `framework` in every declaration and import; update the Makefile source and scalafix lists, `.scalafix.conf`, the syntax lint's framework paths, the lifter's fully qualified framework names, and the gate tests. Regenerate once and classify the IR/Case diff (R2). Search the repository for leftover `model/umpire` and `umpire.` framework references; keep product names (`tools/umpire`, `umpire-*` targets, `umpire.v1`).
## Acceptance
- [ ] `model/framework/` with package `framework`; no `model/umpire` path or `umpire.` framework reference remains outside dated history (R1)
- [ ] Regenerated IR/Case diff contains only paths, positions, the `umpire.` → `framework.` mapping and the fingerprints it implies; Definition IDs, Query answers and receipts unchanged (R2)
- [ ] Lifter, gate, Model tests, `make lint-model`, `make umpire-check-model` and the Go tooling suite pass (R3)
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
