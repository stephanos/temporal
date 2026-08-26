---
satisfies: [R5, R6]
---
# fn-32-add-umpire-refinement-and-the-first.5 Enforce Refinement imports and synchronize architecture guidance

## Description
Close R5/R6 with import guards, aggregate tests, and authoring/conformance documentation.

**Size:** S
**Files:** `model/UmpireTests.lean`, `model/TemporalModelTests.lean`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`
**Touches:** [model/UmpireTests.lean, model/TemporalModelTests.lean, model/README.md, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md]

### Approach
- Mechanically enforce the Feature/System/refinement import graph.
- Document semantic altitude, authored-to-checked lifecycle, derivations, and separate failures after interfaces stabilize.
- Include the first teaching progression from Feature through System refinement.

### Investigation targets
**Required** (read before coding):
- `model/UmpireTests.lean` — reusable aggregate
- `model/TemporalModelTests.lean` — ordinary Temporal aggregate
- `model/ARCHITECTURE.md` — current package/lifecycle map
- `model/Umpire/ARCHITECTURE.md` — current deep-module contracts

### Acceptance
- [ ] Import guards prove only the focused refinement leaf composes Feature and System.
- [ ] Aggregate tests and regression fixtures pass.
- [ ] Documentation distinguishes Observation, Refinement, and Property outcomes.

## Acceptance
- [ ] R5 mutation isolation and R6 facade/import checks pass.
- [ ] Documentation reflects implemented contracts and preserves comments.
- [ ] Full model and regression gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
