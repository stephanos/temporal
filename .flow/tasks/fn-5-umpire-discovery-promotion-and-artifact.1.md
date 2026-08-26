---
satisfies: [R1, R3, R7]
---
# fn-5-umpire-discovery-promotion-and-artifact.1 Define the reusable checked semantic catalog

## Description
Create the domain-neutral catalog language and checker for R1/R3/R7.

**Size:** M
**Files:** `model/Umpire/Catalog.lean`, `model/Umpire/Catalog/Language.lean`, `model/Umpire/Catalog/Tests/Fixtures.lean`, `model/Umpire/Catalog/Tests/Validation.lean`, `model/Umpire.lean`
**Touches:** [model/Umpire/Catalog.lean, model/Umpire/Catalog/Language.lean, model/Umpire/Catalog/Tests/Fixtures.lean, model/Umpire/Catalog/Tests/Validation.lean, model/Umpire.lean]

### Approach

- Define catalog entries, aliases, deprecations, references, dispositions, checked graph identity, exact lookup, and structured errors over existing checked declaration metadata plus fn-16's `CheckedSpaceMetadata` projection.
- Validate the complete graph before constructing `CheckedCatalog`; do not copy clauses, traces, kernels, or planner logic.
- Make authoring-order permutations canonical and keep constructors for checked values private.

### Investigation targets

**Required:**
- `model/Umpire/Core.lean:8-105` — current declaration identity, kind, source, and semantic values.
- `model/Umpire/Core.lean:189-224` — structured error conventions.
- `model/Umpire/Core.lean:430-464` — identity and kind validation.
- `model/Umpire/Property/Language.lean:192-270` — authored-to-checked lifecycle.
- `model/Umpire/Space/Metadata.lean` — fn-16-owned checked metadata projection; consume without copying compiler logic.
- `model/Umpire/ARCHITECTURE.md:31-44` — package lifecycle boundary.

### Quick command

`cd model && mise exec -- lake build Umpire.Catalog.Tests.Validation`

## Acceptance
- [ ] Heterogeneous checked metadata forms one canonical catalog independent of authoring order.
- [ ] Duplicate/case-colliding identities, wrong kinds, missing digests/sources, dangling references, alias cycles, invalid replacements, and conflicting dispositions have exact error fixtures.
- [ ] Exact list/lookup primitives never normalize or silently redirect selectors.
- [ ] Catalog identity changes only for meaning-bearing metadata and graph changes.
- [ ] The package imports no Temporal module and contains no copied semantic bodies.
- [ ] Existing comments in touched files are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
