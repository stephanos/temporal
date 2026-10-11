---
satisfies: [R1, R2, R3, R5]
---
# fn-149-safety-and-liveness-groups-for-object.6 Migrate Scala claim groups and commit native/lift equivalence

## Description
**Size:** M
**Files:** grouped Model/capability declarations, runtime/lifter discovery, native behavior pins and scratch-lift fixtures.
**Touches:** [model/temporal/**/*.scala, model/framework/IrFile.scala, model/framework/IrFile.test.scala, model/irgen/Capabilities.scala, model/irgen/Lifting.scala, model/irgen/Structure.scala, model/irgen/Order.scala, model/irgen/Conventions.scala, model/irgen/test/**, model/irgen/testdata/**]

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

### Conductor activation — attempt 1 (2026-10-11)

Discovery .1 and placement .2 source/closure commits are integrated locally. Placement source is 77825c5bd2ab402e9552f6c3a5def9570be23c0d; closure is 25e12fa6a1a440a65e072732f43c2f16dc2cae9e. Integration tracking is 7282ade176. Root independently authenticated the complete 3,654-file placement seal, all 591 source/config/documentation copies against the source commit, the exact 1,894 model class/TASTy members, actual runtime main/test outputs and dependencies, every fixture jar's compiled bytes, 29 source-attributed crossed-kind refusals, seven independent wrong-owner roots, all 12 final placement tests and 48 other targeted tests. Manifest SHA is 81c2def73044e66438d8973a8299fe2f222a09ff409f8feab589816f931cc1fc. Root audits are main .flow/tmp/scala-dsl/fn149-2-root-seal-audit.py/.json and fn149-2-root-final-placement-audit.json. The immutable placement proof is w-1d .flow/tmp/scala-dsl/fn149-2-final-proof.

Retain the immutable witness seal w-1d .flow/tmp/scala-dsl/fn140-10-retry-seal (manifest e8a7dbf4a9e51ae442cff4ae72de521ef4aff7340dd43ed358cd199a5acecd1e), discovery seal fn149-1-final-proof (manifest 630b1ef1a7c73993886dc9d6f608d26bbe7c4365b6e4d05ca3d07360278417b6), and placement seal read-only. Root's exact bridge is main .flow/tmp/scala-dsl/fn149-2-root-witness-bridge-audit.py/.json, also copied in the placement seal. The current declaration-side lift adds exactly 19 properties-section source roots and the existing, previously unqueried nexusProtocol.asyncStarts Property plus its literal source-derived predicate function, relative to fn140.10. All other complete IR fields and all three waiver artifacts remain identical. This is .1 discovery admission, not an added Model promise or a grouping behavior change. Preserve this explicit ledger when comparing .6 against fn140; do not silently delete, ignore or normalize this claim away. Placement's original-runner/current-runner seven complete production artifacts are byte-identical and frozen in original-source-ir/current-source-ir.

A fresh-context single implementation worker owns w-1d. Attempt 1 activation starts 14:26 UTC, with deadline 17:26 UTC (180 minutes); at most two consecutive non-done returns, no third respawn or automatic extension. No green baseline handoff is supplied. Before edits, run the current Quick source baseline and freeze a complete independent declaration/native/lift universe from the immutable inputs and actual current source. Missing inventories, unexplained identities or provenance, uncovered rows and unsupported predicate forms must stop the source seal rather than earn a count-only or table-only pass.

Only the declared Scala source Touches and worker lifecycle writes are authorized. Heavy commands serialize on /tmp/umpire-heavy-gates.lock with owned foreground handles. Root owns MILESTONES and downstream admission. Docs/layout final seal .7, fn164 expectation/colon work and later meaning-changing work remain held until their predecessors' committed closures. Existing Go/report checkpoint .3 and every original .4/.5 consumer obligation remain deferred and are not reset or closed. Review mode none records a held skip, not SHIP. No Go commands/edits, managed IR/Cases/goldens/mirrors regeneration, full gates, independent review, live runs, publishing, rebase or history rewrite.

The exact canonical equivalence Make route may require owner-held linked-API generation. Inspect its prerequisites without executing Go/generation. If unavailable in scope, record the real disposition and run the same direct Scala steps-only gate using independently authenticated current jars and the immutable baseline's absolute main-checkout path; no fabricated Make failure or canonical Make credit. Keep complete identity/span and predicate/bound/assumption/Query/realization/origin/waiver/monitor comparisons separate from native StepTables.

Preserve the main user-owned SEMANTICS draft SHA89e1fbcb204ae44fb2cbfe4129f3a89f826af00e666707bf8d639a0dd2774841 and protected played_test.go SHAc63deec916b93b86b679785a5791ecbbe5b436f9bb21b006570882bc583bdc77. Neither file is a .6 implementation touch.

### Owner cancellation — stopped before implementation (2026-10-11)

The owner cleared the autonomous goal after requesting a stop boundary. The fresh worker stood down before grouping implementation, with no tracked source edits or task commits. Task .6 remains in_progress and resumable, not done; no later task was admitted. All owned handles 45490, 25556 and 51855 are terminal. Baseline-only handovers are w-1d .flow/tmp/scala-dsl/fn149-6-summary.md and fn149-6-evidence.json. Root authenticated all 511 archived source/config files against activation commit 7c5ebf831160c49fecab07fbdce67eb2cf87e3ee, the seven complete frozen IR artifacts and model jar. Scoped baseline compile, grouping12, discovery7, capability9, ActivitySubjectRegression4, syntax and direct source steps-only equivalence passed; the already-launched native laws suite passed2. A guessed empty selector and provisional native-capture compile error receive no credit. The required complete declaration/native/predicate oracle, grouping migration, both capability scans, flat refusal, caller/reference/export updates, exact identity/span mappings, mutation controls and final source seal remain unfinished. Original consumer failures and .4/.5 obligations remain open. Resumption requires an explicit owner instruction and fresh admission/timebox; the cancelled attempt grants no automatic retry or deadline extension.

### Conductor scope amendment — current WIP TASTy compatibility (2026-10-11)

Authorize the narrow `model/irgen/Conventions.scala` change and its focused regression under `model/irgen/test/**` needed to distinguish compiler-synthetic `framework.Syntax$package.in` references with source-impossible coordinates from authored undotted `in`. Preserve every real `R4.dotted-in` refusal and the exact migrated predicates; no broad suppression or source rewrite is allowed. This directly integrates the owner-committed `63cd2f2b` WIP baseline and does not expand into other convention rules.
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
