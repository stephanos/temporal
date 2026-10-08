---
satisfies: [R1, R2, R3]
---
# fn-149-safety-and-liveness-groups-for-object.2 Enforce safety and liveness declaration placement

## Description
Enforce safety and liveness declaration placement. Advances R1, R2, R3 of the parent spec.

**Size:** M
**Files:** `model/irgen/Structure.scala`, `model/irgen/Order.scala`, `model/irgen/Claims.scala`, `model/irgen/test/Fixtures.test.scala`, `model/irgen/testdata/propertyGroups/**`
**Touches:** [model/irgen/Structure.scala, model/irgen/Order.scala, model/irgen/Claims.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/propertyGroups/**]

### Approach
- Enforce placement from resolved declaration kind: Property and safety monitor references in safety, Progress in liveness. Resolve aliases and factory/bundle outputs before checking; attribute the misplaced authored binding to its source.
- Reuse section/order lint machinery (or its fn-141 successor), not a second semantic classifier. Accept omitted empty groups and allow helper definitions without treating them as claims.
- Retain existing progress bound/assumption admission and compiler type checks. Add source-attributed negative specimens for crossed kinds, aliases that conceal a crossed kind, bad bounds and incompatible owners.
- Prepare the final flat-layout refusal, activated with task 4's migration so intermediate tasks retain a usable tree.

### Investigation targets
**Required:**
- `model/irgen/Structure.scala:545` - section ownership.
- `model/irgen/Order.scala` - source-order rules.
- `model/irgen/Claims.scala:372` - progress bound admission.
- `model/irgen/test/Fixtures.test.scala` - refusal assertions.

### Key context
Re-anchor paths and interfaces against completed fn-140/fn-141 and the approved schema/package moves before editing. Keep this work outside the activity batch and serialize shared regeneration with it and the schema chain; no new spec-close dependency is implied.

### Quick commands
```bash
mise exec -- scala-cli test model/irgen --test-only Fixtures
```

## Acceptance
- [ ] Valid, empty and omitted groups pass; direct and aliased kind mismatches fail at their source with the expected group.
- [ ] Positive, missing and non-positive bound cases retain the correct compiler/lifter/checker disposition.
- [ ] Helper/bundle and shared-capability fixtures neither bypass placement checks nor duplicate registrations.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
