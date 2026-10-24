---
satisfies: [R1, R3]
---
# fn-244-package-layout-of-modelumpire.1 Rust crate layout: nesting, annotations/catalog naming, descriptions, crate map

## Description
After fn-238 and fn-242 close (they edit these crates). Nest ir-derive and catalog-build under their parents; resolve the annotations/catalog naming (rename or merge, recorded); fix stale descriptions; add the crate map to model/README.md; update MIGRATE.md and AGENTS/module-map paths. Goldens and guest bytes unchanged except listed path-embedding files.

## Acceptance
- [ ] Consistent nesting; annotations/catalog split clear
- [ ] Crate map in model/README.md; descriptions current
- [ ] Default and required model gates pass; changed files listed

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
