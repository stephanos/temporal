---
satisfies: [R4, R6]
---
# fn-31-deepen-umpire-target-and-simplify.5 Enforce Target architecture and authoring documentation

## Description
Close R4 and R6 with facade/import/mutation coverage and synchronized authoring guidance.

**Size:** M
**Files:** `model/Umpire.lean`, `model/UmpireTests.lean`, `model/TemporalModelTests.lean`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`
**Touches:** [model/Umpire.lean, model/UmpireTests.lean, model/TemporalModelTests.lean, model/README.md, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md]

### Approach
- Test public facades and forbidden import directions.
- Add independent mutations for provider, connector, law, enumeration, and kernel errors.
- Document ordinary versus maintainer authoring responsibilities after the API is final.

### Investigation targets
**Required** (read before coding):
- `model/Umpire.lean` — public aggregate
- `model/UmpireTests.lean` — aggregate test boundary
- `model/TemporalModelTests.lean` — ordinary Temporal test aggregate
- `model/Umpire/ARCHITECTURE.md:37-46` — current checked lifecycle

### Acceptance
- [ ] Import and domain-purity checks enforce the architecture.
- [ ] Mutation tests fail at the Target boundary with source-located diagnostics.
- [ ] Documentation teaches the small interface and retains explicit semantic choices.

## Acceptance
- [ ] R4 diagnostic mutations and R6 isolation checks pass.
- [ ] Aggregate Umpire/Temporal builds and regression gate pass.
- [ ] Documentation reflects the implemented interface without duplicating semantics.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
