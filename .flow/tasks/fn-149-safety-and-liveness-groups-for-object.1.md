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
Implemented designated properties.safety and properties.liveness discovery for Machine and Composition through native IrFile construction and TASTy lifting. Backed private sections, private unqueried Progress vals, supported bundles and Property subtypes retain their declarations; helper methods and arbitrary nested objects remain excluded, and monitor aliases resolve only to an existing home attachment.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none)

The dedicated PropertyGroupsFixtures suite has seven independently selected tests. Its literal baseline requires six Properties and four Progress claims, including direct and bundle-only non-form owners without Query consumers. Both flat and grouped native harnesses check all 30 applicable predicate rows and eight monitor rows. Complete six-file IR equality uses explicit root/name mappings and authenticated authored source-line mappings, with 546 position visits. Query selection, exploration limits, Progress bounds 1/2/3/4 and fair assumptions, owner identities and exactly one monitor attachment are pinned independently. Native Query total remains unset while lifting derives total=2 from its scenario domain; this existing metadata distinction is checked explicitly. Dropping a whole group from both candidates fails the literal inventory even when candidate-to-candidate equality still passes.

Runtime tests independently prove lazy, exactly-once initialization and retained reference identity. They exclude ordinary and reserved-section helper defs and cover backed private properties/safety/liveness access. Existing strict assertions and declaration/layout gates remain enabled. Order admits only designated nested structure and aliases to already attached monitors; wrong-kind/bound admission remains task .2 and final flat refusal remains .6.

Baseline IrFileDiscoveryTest passed six tests and RulesTest passed 24. The absent dedicated suite selected zero and is inconclusive, not green. Initial red1 through red7 were setup/layout/compile failures with no regression credit. Preserved red8 reproduces five missing unqueried Properties in both successful candidate lifts. Subsequent private-val, helper-def, reserved-section, subtype and private-section controls reproduce their exact discovery/runtime omissions before their fixes. The subtype test retains the existing never sugar and independently pins its NOT(Call) shape, helper parameters and NE body. Specimen revisions needed for existing duplicate-promise and bundle-field admission receive the fresh final native and full-metadata proofs, not substituted old-red evidence. A wrong-glob empty selection and formatter/CWD preflight errors are retained as inconclusive; corrected nonzero/configured runs supply final evidence.

Final rebuilt-source Quick commands passed PropertyGroupsFixtures 7, IrFileDiscoveryTest 7 and RulesTest 25 tests, all terminal rc0 and --require-tests. Proportionate additional source checks passed framework 77, conventions 9, overlap 2, qualified names 4, checker rules 19, strict step-ledger controls 8, equivalence 11, equality 1 and exported native laws 2 tests covering 43 subjects and 172 law rows. Current source syntax, comments and block-form gates, configured scoped formatting for all 23 declared Scala files and git diff --check passed. Test/gate logs and command records are frozen with the proof.

The owner-approved direct source --check-equivalence fallback passed against the absolute immutable fn160 baseline, using the unchanged map and ledger. It generated 46 native keys, 45 gzip payloads and 2,189,510 indexed rows. Current index SHA-256 is 3a8f32405bfc69ce0f940097398575d0c3313facdd43fb88dd2bac5d8b8f3bcd. Root independently authenticated every compressed payload against the immutable witness seal and every decompressed digest/row count. This is native steps-only evidence, not canonical Make, IR provenance, Go interpreter, Cases, outcomes, counterexamples or live assessment credit.

Immutable final proof is .flow/tmp/scala-dsl/fn149-1-final-proof. Its manifest SHA-256 is 630b1ef1a7c73993886dc9d6f608d26bbe7c4365b6e4d05ca3d07360278417b6; all 792 copied files were independently rehashed successfully. It contains the final fixture scratch property-groups.14401525082804577412, complete four-jar class/TASTy/source provenance, all six IRs and native/negative logs, 476 current Scala sources, 20 pinned dependency jars, current model jar and classpath, source diff, native step tables, historical map/ledger/index inputs and five root audit receipts. Model jar SHA-256 is 94f8b794809f98ca52ee03636691dfb589e7988ab01fc4aece226435e47fe3ca, with 1,397 class and 497 TASTy entries. A native audit arriving after the seal is frozen separately in .flow/tmp/scala-dsl/fn149-1-native-audit-proof; SHA256SUMS SHA-256 is 0d45f2096dcd804d197e35da02c227d4887b12d7e420f9f9c83799786ce63a4b. Earlier red proofs and authenticated green7/current snapshots remain unchanged.

The original fn140 witness manifest remains e8a7dbf4a9e51ae442cff4ae72de521ef4aff7340dd43ed358cd199a5acecd1e. No Go, managed production IR/Cases/golden/mirror regeneration, full gates, independent review, live run or publication ran. Gate classify requests FULL; those gates stay held under the explicit owner source-lane override and receive no passing/skip receipt. Consumer obligations .3/.4/.5, migration .6, docs/seal .7 and fn150 composition-progress limits remain deferred. No other task was implemented or admitted. All owned command handles terminated before completion.

Source commit b751748caef54e2ec8b7ffa666098a4443015658 contains only the 23 declared Scala files. Root metadata commits were safely merged without history rewrite. Base commit is bb5dfb6a87c34a8eb30482fb29c78e90253ae247.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ed13789459a048cb2bd8b8d0a9b85a49faffc388, 0085f99ee3d79c11eaa566d650a47eff5ad2bed1, 670d1ec4dad6c738a88de4faff2aefde7446ba79, 7a6012ed94bf36a1d6a2ab9980da997a3ff5e34a, 9936b54916b76753db4e0f0d825eaffe02f07d19, b751748caef54e2ec8b7ffa666098a4443015658
- Tests: mise exec -- scala-cli test model/irgen --server=false --require-tests --test-only '*PropertyGroupsFixtures*', mise exec -- scala-cli test model/project.scala model/framework --server=false --require-tests --test-only '*IrFileDiscoveryTest*', mise exec -- scala-cli test model/project.scala model/framework --server=false --require-tests --test-only '*RulesTest*', mise exec -- scala-cli --power package model/project.scala model/framework model/temporal --server=false --library -f -o model/build/model-scala.jar, mise exec -- scala-cli run model/check --server=false -- --check-syntax, mise exec -- scala-cli run model/check --server=false -- --check-comments, mise exec -- scala-cli run model/check --server=false -- --check-block-form, mise exec -- scala-cli run model/check --server=false -- --check-equivalence /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn160/baseline-ae3f36ee51 model/testdata/equivalence/batch-1d.map --steps-only, mise exec -- scala-cli fmt --scalafmt-conf /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/w-1d/model/.scalafmt.conf --check model/framework/IrFile.scala model/framework/IrFile.test.scala model/framework/Rules.test.scala model/irgen/Claims.scala model/irgen/Lifting.scala model/irgen/Order.scala model/irgen/test/PropertyGroups.test.scala $(rg --files model/irgen/testdata/propertyGroups -g '*.scala'), git diff --check
- PRs: