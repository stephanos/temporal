---
satisfies: [R1, R2]
---
# fn-149-safety-and-liveness-groups-for-object.1 Discover grouped claims and prove registration equivalence

## Description
Discover grouped claims and prove registration equivalence. Advances R1, R2 of the parent spec.

**Size:** M
**Files:** `model/framework/IrFile.scala`, `model/framework/Rules.test.scala`, `model/irgen/Lifting.scala`, `model/irgen/Claims.scala`, `model/irgen/test/Fixtures.test.scala`, `model/irgen/testdata/propertyGroups/**`
**Touches:** [model/framework/IrFile.scala, model/framework/Rules.test.scala, model/irgen/Lifting.scala, model/irgen/Claims.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/propertyGroups/**]

### Approach
- Make designated properties.safety and properties.liveness groups discoverable for Machine and Composition owners through the actual framework IrFile and TASTy discovery paths. Fn-141 runs later; use no hypothetical replacement exporter.
- Keep traversal bounded to declared sections, not arbitrary nested objects. Cover direct vals, imported aliases, shared factories/bundles and progress declarations that no Query references.
- Align IrFile construction with lifted discovery, preserving lazy initialization behavior and registering each declaration once. Do not move monitor definitions or attach references as new monitors.
- Keep existing flat declarations temporarily accepted until task 4 migrates their owners and enables the final layout rule.
- Pin a small mixed Property/Progress/monitor-reference specimen independently before and after regrouping. Compare declaration inventory, predicates, owner/claim identities and Query/check results; explain each qualified-name/identity/source-coordinate difference. Prove runtime initialization and exactly-once registration separately from lifted goldens, including explicit Progress roots with no Query consumer. A dropped whole group must fail the independent inventory comparison.

### Investigation targets
**Required:**
- `model/framework/IrFile.scala:29` - runtime construction and Query reflection.
- `model/framework/Rules.test.scala:275` - independent lazy section initialization and IrFile tests.
- `model/irgen/Lifting.scala:24` - root enumeration and claim kinds.
- `model/irgen/Claims.scala:91` - duplicate registration and folded-reference memo.
- `model/irgen/test/Fixtures.test.scala:2053` - bundle and declaration identity fixture patterns.
- `model/temporal/features/nexus/workflow/Workflow.scala:133` - no-Query explicit Progress roots.

### Key context
Conditional Batch 2 entry follows the committed fn-140.6 witness seal. Re-anchor to the actual fn-155 mapping, fn-156 enforcement and fn-140 vocabulary; future APIs remain unknown here. Fn-141 comes later. Root owns placement/source gates; .5's final grouping seal precedes fn-123.1, and closure waits fn-123.8. Shared heavy work uses the real `/tmp/umpire-heavy-gates.lock`.

### Quick commands
```bash
mise exec -- scala-cli test model/irgen --test-only Fixtures
```

## Acceptance
- [ ] A focused fixture discovers every nested claim exactly once, including Progress without a Query consumer; expected counts come from an independent declared baseline and no zero-selected proof passes.
- [ ] Independent runtime initialization/registration and lifted grouped/flat baselines have the same semantic inventory and check outcomes; a dropped whole group fails, and every identity/source difference is explained.
- [ ] Helpers/arbitrary nested objects are not silently exported, and referencing a monitor changes neither attachment nor initialization.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
