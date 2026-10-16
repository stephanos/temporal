---
satisfies: [R3]
---
# fn-173-reachable-exploration-drop-full-tables.16 Retire native StepTables and catalog-based refinement/law walks

Touches: [model/framework/dsl/StepTables*, model/framework/dsl/Refine.scala, model/framework/dsl/*Laws*.test.scala, model/framework/dsl/IrFile.test.scala, model/framework/dsl/Compose.scala, model/framework/dsl/IrFile.scala, model/framework/README.md, model/framework/migrate/README.md, model/README.md]

## Description
Retire the native producer/walk portion of R3 after the completed six-sample fixture/map bundle. Original .3 retains Go replacements, orchestration, carriers and integration; this is not a replacement semantic seal.

**Size:** M, bounded deletion and concrete framework-test migration. Split before expanding into new checker or compatibility machinery.
**Files:** framework/dsl/StepTables.scala and its test; Refine.scala; MachineLaws.test.scala, MachineLawsHole.test.scala, IrFile.test.scala; comment references in Compose.scala/IrFile.scala; framework README/migrate README and model/README.md law/seal guidance.

### Approach
- Remove the StepTables producer, writeStepTables main and dedicated producer tests. Scoped pinned searches found no other executable framework symbol caller; refresh that audit first. Excluded Temporal callers are held debt, not a reason to retain aliases/stubs or inspect disabled sources.
- Remove Refinement.unclosed and MachineLaws all-catalog image/totality, role-closedness and repeated relation/state/outcome/fact membership walks. Keep Refinement/declaredBy and actual binding-order/duplicate-binding checks under clear concrete structural helpers, not a MachineLaws.check facade returning pass/empty/skipped semantic results.
- Re-anchor each IrFile.test.scala caller (pinned :286,322,335,351,389,400,408,432,451,463,477) and MachineLawsHole tests. Preserve discovery, owners, derivations, restrictions, actual Hole identity/exception examples and concrete callback cases where meaningful. Catalogue removed universal guarantees in held .3; do not rename the same catalog checker or delete assertions as supposed replacement proof.
- Preserve the completed old-source/map/expectation bundle. Leave private Query.total None and exporter/carriers held. Retain pure canonical comparison, RenameMap, lift comparison and synthetic negative controls; do not blanket-delete check/equivalence or disable Samples' lift-and-Go harness.
- ChangeCheck/Makefile historical equivalence orchestration and consumer reconciliation remain original .3 work. Document that the old source-stage seal cannot run after producer retirement; do not mark unavailable or unrun stages passed. No Makefile/exporter/schema/Go/Temporal changes in this task.

### Investigation targets
Required: dsl/StepTables.scala/test, Refine.scala, MachineLaws.test.scala, MachineLawsHole.test.scala, IrFile.test.scala; completed retained sample bundle; parent source-tail contract and original .3 proof inventory. Read scoped framework callers only, not model/temporal or archived Temporal source.

### Verification
Compile framework native roots and run real nonempty IrFile discovery, concrete binding/Hole and touched syntax selections, scoped lint/format, plus scoped retired-symbol/walk reference audit. No whole-state table/law execution or new explorer. Native count reductions must match explicit retirement, not broad skips. No old baseline/equivalence, exporter, Go, backend, Temporal or regeneration gate.

## Acceptance
- [ ] StepTables/writeStepTables and Refinement.unclosed/native catalog law evaluation are retired with framework callers reconciled; no compatibility facade or new semantic checker remains.
- [ ] Concrete binding-order, discovery/ownership, derivation and useful Hole/callback controls survive; actual native construction does not enumerate state or evaluate callbacks. Nonzero focused suites pass.
- [ ] Source/utility references and guidance distinguish retired native producers from retained pure comparison and unavailable historical orchestration; six old fixtures/maps remain available.
- [ ] Lost universal law/coverage/role/refinement guarantees and complete R3/R4 replacements, total carriers and consumer compatibility remain explicit held integration obligations, not passed seals.

## Done summary
Native StepTables/writeStepTables and their four producer tests, Refinement.unclosed and the MachineLaws catalog evaluator are retired. IrFile retains eight concrete discovery/ownership/binding/callback tests, and NativeHoleCallbackTest preserves seven explicit Hole/exception/output cases without a law-result facade or state scan. Typed Refinement and declaredBy remain.

Baseline commit 99044786112f62cb28b272541a50ce8731da7282 is persisted in .flow/tmp/base_commit and .flow/tmp/fn17316-native-baseline. Source commit 4dc412e26ce99e4750c5825cc8eb660853227f95. Tier: session (jev-unavailable(no_key)). Host execution metadata supplies no actual-model identifier. One disjoint test child completed and joined without compiler/Git/Flow authority; this worker owned production/docs, compilation and commits.

baseline: green via explicit owner source/test handoff from .15 source 3d47b0cd5caa8759d971a5bbdc5508985ba1e570 and receipt efa853fa22abfa5a62aec0654b7a7e2f9f360124, with final 77 native and 29 lint cases rc0. The intervening admission commit changed only MILESTONES.md; reuse follows the owner's scoped source/test override across that non-.flow path. Baseline lint was rerun before edits and passed 29. The old catalog suites were deliberately not executed, and this retirement claims no behavioral red-to-green proof.

All 11 former IrFile law callers were reconciled. Exact Added/Original discovery and advance identity remain. Added returns literal open/closed ordered Steps with done/no facts and distinct authored choices. Permuted retains actual retreat/advance against independent advance/retreat; DuplicateBinding retains two identical advance declarations. Poison state catalogs/effects prove those structural inspections stay inert. The complete eight-owner root set remains. Mapped/Rebound keep Original and the concrete map/rebound callback; restriction drops refinement. BadImage returns closed targeting Narrow; two Unrepeatable calls return open then closed. BadClosedness maps settled Closed to pending Waiting. Partial maps open and explicitly throws unmapped closed for closed. Filesystem/path refusal, collisions and designated Property initialization controls remain.

The seven Hole cases pin the exact Hole object on repeated calls, two distinct same-name Holes, Hole/known alternation, an ordinary IllegalStateException and literal invalid state/outcome/fact results after a Hole input. They provide bounded callback observations only. The four StepTables tests retired with the producer. One aggregate Hole scan test became seven concrete tests; IrFile remains eight, so the affected registered-test count changes from 13 to 15. Universal result rows/cardinality/dispositions and their reason protocol are removed.

Frozen-source verification passed IrFileDiscoveryTest8, NativeHoleCallbackTest7, NativeDataTest9, RulesTest26, Choices4 and SampleExpectTest6. Final scoped lint RuleSuites passed29, giving 89 unique focused cases. Initial fmtcheck returned rc1 for one for/do newline in the Hole table; the exact formatter layout was applied, final fmtcheck returned rc0 and the Hole suite recompiled/reran7 rc0. git diff --check returned rc0. Logs, exact commands/counts and terminal command handles are in .flow/tmp/fn17316-native-run-evidence.json. Source/test writes were frozen during each compiler epoch.

Scoped executable-symbol and touched-walk audits passed. The only remaining framework Scala StepTables mentions are three historical format comments in retained pure equivalence files. Pure comparison, RenameMap, lift comparison, synthetic negative controls and Samples' lift-and-Go harness have no diff. All six old source snapshots compare byte-identically with 53f05f9bce8434ee069ec7717212700217358bbc; the complete six maps and 25 intended answers also have no diff. Cache's many-to-one history projection, Voting's guard/projection deltas and Broadcast's interpreter-owned dequeue remain explicit.

Framework, migration and model guidance now identifies native source-state tables/text seals as unavailable. Original .3/R3/R4 retains universal refinement totality/target membership, role coverage/closedness, all-state ordered repeatability, state/outcome/fact membership and unknown-row/Hole handling. ChangeCheck/Makefile consumer reconciliation, the private Query total bridge and exporter carriers remain held. Complete R3/R4 and R7 graph/Query, numeric, lifting, IR/Go/backend and Temporal work stays open. No new checker, Result/skip facade, Scalafix or proof manifest was added.

flowctl classification returned rc1 FULL for unmatched model/README.md. Verification follows the owner's scoped native checkpoint exception. No held broad native, Model/mini, catalog, exporter, schema, Go, backend, regeneration or Temporal gate ran. Formal review remains the conductor's next phase after this checkpoint. MILESTONES.md was not edited. Only .16 closes; the parent and held tasks remain open.

stage: impl-review - skipped(config: REVIEW_MODE=none; owner defers formal review until after .16)

stage: plan-sync - skipped(config: planSync.enabled=false)
## Evidence
- Commits: 4dc412e26ce99e4750c5825cc8eb660853227f95
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.IrFileDiscoveryTest, mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.NativeHoleCallbackTest, mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.NativeDataTest, mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.RulesTest, mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.Choices, mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.SampleExpectTest, mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only framework.check.lint.*RuleSuite, mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/dsl/Compose.scala model/framework/dsl/IrFile.scala model/framework/dsl/IrFile.test.scala model/framework/dsl/MachineLawsHole.test.scala model/framework/dsl/Refine.scala, git diff --check, six fixed native-migration snapshots cmp against 53f05f9bce8434ee069ec7717212700217358bbc (commands/log evidence .flow/tmp/fn17316-native-run-evidence.json), scoped retired-symbol and former catalog-walk audits (expected no matches; .flow/tmp/fn17316-native-run-evidence.json)
- PRs: