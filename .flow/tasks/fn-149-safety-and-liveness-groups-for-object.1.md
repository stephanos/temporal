---
satisfies: [R1, R2]
---
# fn-149-safety-and-liveness-groups-for-object.1 Discover grouped claims and prove registration equivalence

## Description
Discover grouped claims and prove registration equivalence. Advances R1, R2 of the parent spec.

**Size:** M
**Files:** `model/framework/IrFile.scala`, `model/framework/Rules.test.scala`, `model/framework/IrFile.test.scala`, `model/irgen/Lifting.scala`, `model/irgen/Claims.scala`, `model/irgen/Order.scala`, `model/irgen/test/Fixtures.test.scala`, `model/irgen/test/PropertyGroups.test.scala`, `model/irgen/testdata/propertyGroups/**`
**Touches:** [model/framework/IrFile.scala, model/framework/Rules.test.scala, model/framework/IrFile.test.scala, model/irgen/Lifting.scala, model/irgen/Claims.scala, model/irgen/Order.scala, model/irgen/test/Fixtures.test.scala, model/irgen/test/PropertyGroups.test.scala, model/irgen/testdata/propertyGroups/**]

### Approach
- Make designated properties.safety and properties.liveness groups discoverable for Machine and Composition owners through the actual framework IrFile and TASTy discovery paths. Fn-141 runs later; use no hypothetical replacement exporter.
- Admit designated nested groups through the existing Order section-layout gate, with only the structural change required for this discovery proof. Monitor aliases must resolve to an existing monitor definition/attachment in its original home; do not admit new definitions in a property group. Wrong-kind and bound enforcement remain task .2, and helper/nested-object refusals remain intact.
- Keep traversal bounded to declared sections, not arbitrary nested objects. Cover direct vals, imported aliases, shared factories/bundles and progress declarations that no Query references.
- Align IrFile construction with lifted discovery, preserving lazy initialization behavior and registering each declaration once. Do not move monitor definitions or attach references as new monitors.
- Keep existing flat declarations temporarily accepted until task .6 migrates their owners and enables the final layout rule.
- Pin a small mixed Property/Progress/monitor-reference specimen independently before and after regrouping. Compare declaration inventory, predicates, owner/claim identities, native predicate truth and Query metadata; explain each qualified-name/identity/source-coordinate difference. Prove runtime initialization and exactly-once registration separately from lifted goldens, including explicit Progress roots with no Query consumer. A dropped whole group must fail the independent inventory comparison.

### Investigation targets
**Required:**
- `model/framework/IrFile.scala:29` - runtime construction and Query reflection.
- `model/framework/Rules.test.scala:275` - independent lazy section initialization and IrFile tests.
- `model/irgen/Lifting.scala:24` - root enumeration and claim kinds.
- `model/irgen/Claims.scala:91` - duplicate registration and folded-reference memo.
- `model/irgen/test/Fixtures.test.scala:2053` - bundle and declaration identity fixture patterns.
- `model/temporal/features/nexus/workflow/protocol/AckByOriginal.scala:24` - no-Query explicit Progress roots.

### Key context
Selected Batch 2 entry follows the committed fn-140.10 witness source seal. Re-anchor to the actual fn-155 mapping, fn-156 enforcement and fn-140 vocabulary; future APIs remain unknown here. Fn-141 comes later. Root owns activation/readiness and source gates; .7's committed Scala seal releases fn-164; its expectation/colon seal and a separate pre-fault step comparison precede fn-123.1. Legacy .4/.5 retain deferred consumer/full closure. Shared heavy work uses the real `/tmp/umpire-heavy-gates.lock`.

Owner-mode source scope: the dedicated PropertyGroupsFixtures suite must execute a nonzero independently expected population. Native initialization/registration and full scratch-lift metadata are separate proofs, including a non-form owner with unqueried Progress. Go Query/progress outcomes, finite-bound counterexamples/incomplete checks and Cases/assessments remain required in deferred .4/.5. Only Scala source work runs here; managed artifacts/full gates/review/live stay deferred.

### Quick commands
```bash
mise exec -- scala-cli test model/irgen --test-only '*PropertyGroupsFixtures*'
mise exec -- scala-cli test model/project.scala model/framework --test-only '*IrFileDiscoveryTest'
mise exec -- scala-cli test model/project.scala model/framework --test-only '*RulesTest'
```

### Conductor activation — attempt 1 (2026-10-11)

The conductor verified the committed fn-140.10 source and completion receipts
(`b5374fbb0bd13642dea60191f97619754a81acd8`,
`c59b5bdf21b6656bc9799135a9214951e4cbfe11`) and integrated acceptance tracking
(`9e398e81fa`) before admission. The immutable 5,272-file witness seal remains
read-only; its manifest SHA-256 is
`e8a7dbf4a9e51ae442cff4ae72de521ef4aff7340dd43ed358cd199a5acecd1e`.
Committed fn-155/fn-156 source receipts satisfy source admission only; their
consumer closures and fn-149.3/.4/.5 remain held.

Task claimed at 12:28:04.485981 UTC. Fresh-context worker
`/root/grouping_discovery` owns the existing `w-1d` checkout at base
`bb5dfb6a87c34a8eb30482fb29c78e90253ae247`. Attempt 1 starts at
12:28:05 UTC and expires at 14:28:05 UTC (120 minutes). At most two
consecutive non-done returns are allowed for this task; no third respawn.
Only this task's declared Scala source surface and lifecycle files are writable.
Heavy commands are serialized on `/tmp/umpire-heavy-gates.lock`; no other
implementation worker, Go, managed regeneration, full gates, independent review,
live run or publication is admitted. No inherited green baseline is handed off.

Baseline logs record RulesTest 24 tests and IrFileDiscoveryTest 6 tests with
terminal success. The new PropertyGroupsFixtures suite selected zero tests
before its creation: that run is inconclusive, never passing acceptance evidence.
Test-first nonzero independent grouping coverage is required before source changes.
This activation and baseline record claims no implementation acceptance.

At 12:40 UTC, actual focused fixture lifting exposed the missing write-surface
requirement: Order's section layout rejects designated groups and monitor references.
The conductor added `model/irgen/Order.scala` to Files/Touches for narrowly admitting
these structures. Existing layout gates remain enabled; setup/layout-red runs are
inconclusive, not intended discovery regression evidence. Full task acceptance and
the attempt deadline are unchanged.

### Conductor pre-acceptance discovery proof (2026-10-11 12:49 UTC)

The first seven dedicated-suite attempts were compile/setup/layout failures and
earn no intended regression credit. Attempt red8 reached successful flat and
grouped lifts, then failed the fixed six-Property inventory with only
`subject.closed`; all five unqueried Properties were absent in both candidates.
The conductor independently checked all four preserved IR files, original
specimen source hashes, Progress bounds 1/2/3 and `fair` assumptions, and Query
selection/limits in `.flow/tmp/scala-dsl/fn149-1-red-proof` under `w-1d`.
Complete saved jar member sets and bytes match the saved compiled directories:
common 17, flat 22, grouped 26. The snapshot includes 89 files (15 visible
artifacts plus 74 compiled/provenance files). Root read-only audit:
`.flow/tmp/scala-dsl/fn149-1-root-frozen-red-audit.json`.

This proves the intended discovery regression, not task completion. Subsequent
fixture adjustments for existing duplicate-promise and bundle-field admission
must remain explicit and receive fresh final-fixture native/metadata parity.
The bundle-only non-form Progress control additionally guards automatic file
admission. Native registration/truth, whole-group omission refusal and complete
identity/source-attribution proofs remain required before acceptance.

### Conductor final-fixture audit checkpoint (2026-10-11 13:09 UTC)

The conductor independently audited the final authored mixed specimen at
`model/build/history/property-groups.10115162974975750254` in `w-1d`.
All saved packaging sources match the authored bytes, allowing only the recorded
project jar-path substitution. Complete class/TASTy member sets and bytes match
the saved compiled directories: common 23, flat 31, grouped 36, negative 21.
All six actual IR files match after the literal root-universe mapping and an
independently token-authenticated source-line ledger; 534 source-position visits
were checked against real authored files. The frozen inventory is six Properties
and four Progress claims (bounds 1/2/3/4, `fair` assumptions), including private
unqueried Progress and a bundle-only non-form owner. Query selection/limits and
exact single-monitor attachment remain pinned. Both native logs report 30
predicate and eight monitor rows. Root receipt:
`.flow/tmp/scala-dsl/fn149-1-root-final-audit.json` in the main checkout.

The dedicated green7 log contains seven selected passing tests, with whole-group
omission, helper/new-monitor/unattached-reference refusal and independent lifted
predicate/metadata controls. Reserved-section helper red2 separately reproduces
accidental invocation of `def properties` and `def liveness`; the updated
IrFileDiscoveryTest green run checks backed section access, flat reserved claim
names and repeat initialization. These remain pre-acceptance evidence: rebuilding
the current model jar, final focused Quick commands, immutable final proof,
source commit and verified Flow completion receipt are still required. No
consumer/full-gate/review/live closure or task .2 admission is claimed.
## Acceptance
- [ ] A focused fixture discovers every nested claim exactly once, including Progress without a Query consumer; expected counts come from an independent declared baseline and no zero-selected proof passes.
- [ ] Independent runtime initialization/registration and lifted grouped/flat baselines have the same source semantic inventory and native predicate truth; a dropped whole group fails, and every identity/source difference is explained.
- [ ] Helpers/arbitrary nested objects are not silently exported, and referencing a monitor changes neither attachment nor initialization.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
