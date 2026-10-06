# fn-143-rename-modelumpire-to-modelframework.1 Move model/umpire to model/framework, package umpire to framework; regenerate; paths-only proof; docs

## Description
**Size:** M
**Touches:** [model/**, tools/umpire/**, Makefile, .plans/UMPIRE_MODULES.md, .plans/UMPIRE4_SPEC.md, MILESTONES.md]

`git mv model/umpire model/framework`; rename the package `umpire` to `framework` in every declaration and import; update the Makefile source and scalafix lists, `.scalafix.conf`, the syntax lint's framework paths, the lifter's fully qualified framework names, and the gate tests. Regenerate once and classify the IR/Case diff (R2). Search the repository for leftover `model/umpire` and `umpire.` framework references; keep product names (`tools/umpire`, `umpire-*` targets, `umpire.v1`).
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
