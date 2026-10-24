---
satisfies: [R14]
---
# fn-160-one-ir-file-per-feature-collected-from.7 Flatten features/: a kind folder for shared parts, <kind>_<form> folders for forms

## Description
Flatten `features/` so that no form is nested under its kind (fn-160 R14, owner's decision of 2026-10-11): a kind folder holds only what its forms share, and each form is a top-level `<kind>_<form>` folder.

**Size:** M
**Files:** `model/temporal/features/activity/standalone/**` → `model/temporal/features/activity_standalone/**`; `features/nexus/workflow/**` → `features/nexus_workflow/**`; `features/nexus/standalone/**` → `features/nexus_standalone/**`; `features/nexus/shared/**` → `features/nexus/**` (the kind folder is the shared folder); the mini feature at `features/activity_mini/` if it exists by then; `model/irgen/Structure.scala` (layout check), `model/irgen/testdata/layout*` fixtures, `model/README.md` (layout), `.flow/specs/fn-166-*.md` and `MILESTONES.md` paths, the batch rename map, module-map lint rules, CODEOWNERS lines
**Touches:** [model/temporal/features/**, model/irgen/Structure.scala, model/irgen/testdata/layout/**, model/irgen/testdata/layoutRefusals/**, model/README.md, model/testdata/equivalence/*.map, CODEOWNERS]

### Approach
- Move with `git mv`, one form per commit, renaming packages to follow folders (`temporal.features.activity_standalone`, `temporal.features.nexus_workflow`, `temporal.features.nexus`). The kind file stays in the kind folder; `nexus/shared/*` moves up into `nexus/`, so a kind folder is exactly "what the forms share".
- Each form folder holds its levels (`product/`, `protocol/`, `realization/`) and its `generated/` tree directly; no level folder appears in a kind folder.
- Lint: the layout check refuses a form folder nested under a kind folder, and a level folder in a kind folder, naming the folder; `layoutRefusals` fixtures for both. The module-map import rules follow the new packages.
- Identities: packages are part of Definition IDs, so every id under a moved form changes; list them in the batch rename map and check with the steps-only equivalence. Lint acceptances and `generated/` trees move with their form.
- Docs: README layout rules and reading order; the fn-166 paths and `MILESTONES.md` already say `activity_mini`.

### Batching
- Batch 1d (Scala DSL; see MILESTONES.md): with the other fn-160 layout moves, after fn-160.3 so files move once. Scala build, Model and lifter tests, steps-only equivalence under the rename map.

## Acceptance
- [ ] `features/` has no nested form folders: `activity/`, `activity_standalone/`, `activity_mini/` (if present), `nexus/`, `nexus_workflow/`, `nexus_standalone/`, each form with its levels and `generated/` tree directly inside; packages follow folders.
- [ ] A kind folder holds only the kind file and what its forms share; `nexus/shared/` is gone.
- [ ] The layout lint refuses a nested form folder and a level folder in a kind folder, with fixtures.
- [ ] The rename map lists every moved id and the steps-only equivalence passes; README and the module map describe the flat layout.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: