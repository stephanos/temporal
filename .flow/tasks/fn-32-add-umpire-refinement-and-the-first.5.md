---
satisfies: [R5, R6]
---

# fn-32-add-umpire-refinement-and-the-first.5 Enforce Implementation Link imports and synchronize architecture guidance

## Description
Close R5/R6 with import guards, aggregate tests, and authoring/Run Evaluation documentation.

### Review reconciliation (normative)

Extend fn-34's explicit import policy with exactly one composed-test class/root, `Temporal.ImplementationLinkTests.Nexus`, allowed to reach both the Feature family and `Temporal.System.Nexus.ImplementationLink`. Do not classify it as base System, do not use a prefix/suffix wildcard, and add near-miss tests proving sibling System and test modules remain rejected.

**Size:** S
**Files:** `model/ModelLint/ImportGraph.lean`, `model/ModelLint/ImportGraphTests.lean`, `model/UmpireTests.lean`, `model/TemporalModelTests.lean`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`
**Touches:** [model/ModelLint/ImportGraph.lean, model/ModelLint/ImportGraphTests.lean, model/UmpireTests.lean, model/TemporalModelTests.lean, model/README.md, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md]

### Approach
- Mechanically enforce the Feature/System/Implementation Link import graph, the single production Implementation Link leaf, and the single exact non-base-System test root.
- Document semantic altitude, authored-to-checked lifecycle, Evidence Links, and separate failures after interfaces stabilize.
- Include the first teaching progression from Feature through System Implementation Link.

### Investigation targets
**Required** (read before coding):
- `model/UmpireTests.lean` — reusable aggregate
- `model/TemporalModelTests.lean` — ordinary Temporal aggregate
- `model/ARCHITECTURE.md` — current package/lifecycle map
- `model/Umpire/ARCHITECTURE.md` — current deep-module contracts

### Acceptance
- [ ] `ModelLint.ImportGraph` classifies exactly `Temporal.ImplementationLinkTests.Nexus` as the composed-test root; import guards prove only it and the focused production leaf reach both sides, while sibling and prefix/suffix near misses fail.
- [ ] Aggregate tests and regression fixtures pass.
- [ ] Documentation distinguishes Observation, Implementation Link, and Property outcomes.
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
