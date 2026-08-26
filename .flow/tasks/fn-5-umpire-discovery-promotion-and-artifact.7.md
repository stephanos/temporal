---
satisfies: [R1, R2, R3, R4, R6, R7]
---
# fn-5-umpire-discovery-promotion-and-artifact.7 Wire root commands and document catalog-to-promotion lifecycle

## Description
Integrate all fn-5 surfaces through the top-level Makefile and public model documentation for R1-R4/R6/R7.

**Size:** M
**Files:** `Makefile`, `model/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [Makefile, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach

- Add root-only generation, non-mutating stale-check, list, explain, catalog-check, and promotion-proposal targets with strict variable validation.
- Compose catalog generation before catalog-selected regression projection generation in the stable gate.
- Document the checked-catalog authority, exact promotion lifecycle, generated files, commands, and strict ownership transfer to fn-18.
- Reconcile C3/C5 status without claiming runtime discovery, live replay, or migration support.

### Investigation targets

**Required:**
- `Makefile:988-1032` — current Umpire generator/check targets.
- `Makefile:1254` — phony target declarations.
- `model/README.md:68-158` — current authoring and projection workflow.
- `model/Umpire/ARCHITECTURE.md:210-260` — planner/artifact boundaries.
- `.plans/UMPIRE4_COMPONENTS.md:250-288` — C4/C5 status.
- `.plans/UMPIRE4_COMPONENTS.md:331-369` — C8/C10 ownership handoff.

### Quick commands

`make umpire-check-catalog && make umpire-check-regression`

## Acceptance
- [ ] Root Make exposes deterministic generate/check/list/explain/promote entry points and rejects missing variables before invocation.
- [ ] `make umpire-check-catalog` and `make umpire-check-regression` call the generators' in-memory comparison modes, detect every generated/catalog/projection drift class in scope, and never regenerate or mutate files.
- [ ] No model-local Makefile or CI workflow is added.
- [ ] Documentation distinguishes checked catalog, generated projections, in-memory proposal, and future persisted replay/migration ownership.
- [ ] C3/C5 roadmap status matches verified implementation and leaves fn-18/live work open.
- [ ] Focused tests, aggregate model tests, regression checks, and `git diff --check` pass with comments preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
