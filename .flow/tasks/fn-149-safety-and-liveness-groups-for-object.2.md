---
satisfies: [R1, R2, R3]
---
# fn-149-safety-and-liveness-groups-for-object.2 Enforce safety and liveness declaration placement

## Description
Enforce safety and liveness declaration placement. Advances R1, R2, R3 of the parent spec.

**Size:** M
**Files:** `model/irgen/Structure.scala`, `model/irgen/Order.scala`, `model/irgen/Claims.scala`, `model/irgen/test/Fixtures.test.scala`, `model/irgen/test/PropertyGroups.test.scala`, `model/irgen/testdata/propertyGroups/**`
**Touches:** [model/irgen/Structure.scala, model/irgen/Order.scala, model/irgen/Claims.scala, model/irgen/test/Fixtures.test.scala, model/irgen/test/PropertyGroups.test.scala, model/irgen/testdata/propertyGroups/**]

### Approach
- Enforce placement from resolved declaration kind: Property and safety monitor references in safety, Progress in liveness. Resolve aliases and factory/bundle outputs before checking; attribute the misplaced authored binding to its source.
- Check every resolved claim-bearing field of a shared bundle, not only its first classified field. Include mixed-kind and aliased/factory bundle negatives at the authored binding, with an independent expected population in the dedicated PropertyGroupsFixtures suite.
- Reuse the actual section/order lint machinery left by fn-156, not a hypothetical fn-141 successor or a second semantic classifier. Accept omitted empty groups and allow helper definitions without treating them as claims.
- Retain existing progress bound/assumption admission and compiler type checks. Add source-attributed negative specimens for crossed kinds, aliases that conceal a crossed kind, bad bounds and incompatible owners.
- Prepare the final flat-layout refusal, activated with task .6's migration so intermediate tasks retain a usable tree.

### Investigation targets
**Required:**
- `model/irgen/Structure.scala:655` - section ownership.
- `model/irgen/Order.scala:325` - current unified Claim kind and nested-section refusal at :694.
- `model/irgen/Claims.scala:444` - progress bound admission.
- `model/irgen/test/Fixtures.test.scala` - refusal assertions.

### Key context
Selected Batch 2 entry follows the committed fn-140.10 witness source seal. Re-anchor to the actual fn-155 mapping, fn-156 enforcement and fn-140 vocabulary; future APIs remain unknown here. Fn-141 comes later. Root owns activation/readiness and source gates; .7's committed Scala seal releases fn-164; its expectation/colon seal and a separate pre-fault step comparison precede fn-123.1. Legacy .4/.5 retain deferred consumer/full closure. Shared heavy work uses the real `/tmp/umpire-heavy-gates.lock`.

Owner-mode source scope: the dedicated PropertyGroupsFixtures suite must execute a nonzero independently expected population. Native initialization/registration and full scratch-lift metadata are separate proofs, including a non-form owner with unqueried Progress. Go Query/progress outcomes, finite-bound counterexamples/incomplete checks and Cases/assessments remain required in deferred .4/.5. Only Scala source work runs here; managed artifacts/full gates/review/live stay deferred.

### Quick commands
```bash
mise exec -- scala-cli test model/irgen --test-only '*PropertyGroupsFixtures*'
```

## Acceptance
- [ ] Valid, empty and omitted groups pass; direct and aliased kind mismatches fail at their source with the expected group.
- [ ] Positive, missing and non-positive bound cases retain the correct compiler/lifter disposition, with checker-result obligations handed to deferred .4/.5.
- [ ] Helper/bundle and shared-capability fixtures neither bypass placement checks nor duplicate registrations.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
