---
satisfies: [R1, R2]
---
# fn-149-safety-and-liveness-groups-for-object.1 Discover grouped claims and prove registration equivalence

## Description
Discover grouped claims and prove registration equivalence. Advances R1, R2 of the parent spec.

**Size:** M
**Files:** `model/umpire/IrFile.scala`, `model/irgen/Lifting.scala`, `model/irgen/Claims.scala`, `model/irgen/test/Fixtures.test.scala`, `model/irgen/testdata/propertyGroups/**`
**Touches:** [model/umpire/IrFile.scala, model/irgen/Lifting.scala, model/irgen/Claims.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/propertyGroups/**]

### Approach
- Make designated properties.safety and properties.liveness groups discoverable for Machine and Composition owners through the existing declaration/export mechanism. Reuse the replacement exporter if fn-141 has landed.
- Keep traversal bounded to declared sections, not arbitrary nested objects. Cover direct vals, imported aliases, shared factories/bundles and progress declarations that no Query references.
- Align IrFile construction with lifted discovery, preserving lazy initialization behavior and registering each declaration once. Do not move monitor definitions or attach references as new monitors.
- Keep existing flat declarations temporarily accepted until task 4 migrates their owners and enables the final layout rule.
- Pin a small mixed Property/Progress/monitor-reference specimen before and after regrouping. Compare declaration inventory, predicates, owner/claim identities and Query/check results; explain source-path-only differences.

### Investigation targets
**Required:**
- `model/umpire/IrFile.scala:60` - runtime construction/discovery.
- `model/irgen/Lifting.scala:28` - current root enumeration.
- `model/irgen/Claims.scala` - folding and registration; locate exporter replacement after fn-141.
- `model/irgen/test/Fixtures.test.scala:2231` - golden/refusal fixture patterns.

### Key context
Re-anchor paths and interfaces against completed fn-140/fn-141 and the approved schema/package moves before editing. Keep this work outside the activity batch and serialize shared regeneration with it and the schema chain; no new spec-close dependency is implied.

### Quick commands
```bash
mise exec -- scala-cli test model/irgen --test-only Fixtures
```

## Acceptance
- [ ] A focused fixture discovers every nested claim exactly once, including Progress without a Query consumer.
- [ ] Grouped and flat fixture baselines have the same semantic declaration inventory and check outcomes; any identity/source differences are explained.
- [ ] Helpers/arbitrary nested objects are not silently exported, and referencing a monitor changes neither attachment nor initialization.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
