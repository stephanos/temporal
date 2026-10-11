---
satisfies: [R1, R2, R3, R5]
---
# fn-149-safety-and-liveness-groups-for-object.6 Migrate Scala claim groups and commit native/lift equivalence

## Description
**Size:** M
**Files:** grouped Model/capability declarations, runtime/lifter discovery, native behavior pins and scratch-lift fixtures.
**Touches:** [model/temporal/**/*.scala, model/framework/IrFile.scala, model/framework/IrFile.test.scala, model/irgen/Capabilities.scala, model/irgen/Lifting.scala, model/irgen/Structure.scala, model/irgen/Order.scala, model/irgen/test/**, model/irgen/testdata/**]

Implement the Scala migration after .1/.2 and the conductor-verified committed fn-140.10 witness source seal (R1/R2/R3/R5). Legacy .4 retains every complete consumer, assessment and Case comparison. This task changes no Go or managed artifact and does not satisfy .4's consumer proof.

### Approach
- Freeze the complete declaration-side baseline from fn-140.10 before edits. Inventory every Activity and Nexus owner, foundation/capability law, derived design, composition, explicit unqueried Progress, helper bundle and attached monitor. Consume actual fn-155 source identity maps and fn-156 enforcement receipts, without claiming their deferred full closures.
- Move authored claims into designated properties.safety/liveness groups, update references and exports, then activate the final flat-layout refusal. Classify by resolved declaration kind. Helpers remain helpers; omitted empty groups remain valid. Preserve every existing composition form and its unsupported/deferred disposition; add no progress semantics.
- Update runtime and TASTy capability scans together if grouping changes discovery. Keep the independently frozen seven companions/thirteen-law baseline and current per-instance origins, waiver references and monitor attachments. Pin native initialization/exactly-once registration separately from lifted discovery, including a non-form owner with Progress and no Query.
- Compare the complete native inventory and scratch-lift artifacts against the frozen source baseline. Evaluate unchanged predicates on every applicable native state/action/input/result row, including disabled pairs. Compare Progress predicates, declared bounds and assumptions, Query metadata/expected-run presence, realization membership and monitor attachment. Use the existing StepTables/Gate table-equivalence seam for transition behavior.
- Account individually for qualified names, function/declaration identities and edited source spans in an explicit ledger. Native table projection omits provenance/fingerprints, so keep the complete scratch-lift identity/span comparison separate. Apply only approved fn-140/fn-155 mappings. An unexplained delta stops the seal.
- Seed omitted whole groups and a law omitted by both candidate scanners, plus predicate, bound, assumption and Query-population changes. Each must fail its independent frozen oracle. Keep nonzero expected/matched populations. Go receipt, canonical Case/checksum and expected/actual live/replay assessment deltas remain mandatory controls in deferred .4.
- Retain strict inherited completion/fatal/pause-resume failures with exact before/after attribution. A still-red baseline earns no passing credit and stays assigned to its existing Activity correction tasks.

### Investigation targets
**Required:**
- model/framework/IrFile.scala:138 and model/framework/IrFile.test.scala:74 - native package discovery and lazy unqueried Progress.
- model/irgen/Lifting.scala:130 - independent Progress roots.
- model/temporal/capabilities/Properties.test.scala:38 and model/irgen/Capabilities.scala:372 - fixed law inventory, both scans, origins and waivers.
- model/temporal/features/activity/standalone/ActivitySubjectRegression.test.scala:34 - explicit facet inventory, native row predicates and Query metadata.
- model/temporal/features/nexus/workflow/protocol/RejectAfterClose.scala:155 - shared bundles, safety claims and unqueried Progress; AckByOriginal.scala:24 supplies another explicit Progress.
- model/check/StepTables.scala:35 and model/check/Gate.scala:519 - source enumeration and independent table comparison.

### Quick commands
Run serially, using a dedicated nonempty PropertyGroupsFixtures suite rather than the unrelated eager golden campaign.

```sh
mise exec -- scala-cli compile model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/irgen --test-only '*PropertyGroupsFixtures*'
mise exec -- scala-cli test model/project.scala model/framework --test-only '*IrFileDiscoveryTest'
mise exec -- scala-cli test model/project.scala model/framework model/temporal --test-only '*CapabilityPropertiesTest'
mise exec -- scala-cli test model/project.scala model/framework model/temporal --test-only '*ActivitySubjectRegression'
make lint-model-syntax
make umpire-check-equivalence BASE=.flow/tmp/fn160/baseline-ae3f36ee51 MAP=model/testdata/equivalence/batch-1d.map STEPS_ONLY=1
```

Check scoped formatting and git diff --check. Confirm every selector executes a nonzero population. Do not run Go, managed regeneration, full gates, independent review or live work.

## Acceptance
- [ ] Every eligible authored Model and shared-law reference uses the final groups; retired flat, direct/aliased/factory crossed-kind and incompatible-owner refusals retain exact source coordinates.
- [ ] Independent native initialization and lifted inventory preserve the complete frozen universe, including unqueried Progress and fixed capability-law counts; seeded whole-group and dual-scanner law omissions fail.
- [ ] Native tables and applicable-row predicates, Progress bounds/assumptions, Query metadata, realization membership, origins/waivers and monitor attachments match the baseline under exact authorized identity/span maps.
- [ ] Predicate/bound/assumption/population mutations fail; strict inherited semantic failures remain attributed and earn no green credit.
- [ ] Nonempty Scala Quick checks and the source equivalence seal are committed for .7; the complete consumer receipt/Case/assessment proof stays open in legacy .4.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
