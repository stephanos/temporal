---
satisfies: [R2, R3, R8]
---
# fn-173-reachable-exploration-drop-full-tables.12 Native reach/open/checker authoring and Query total retirement

## Description
Finish native authoring portions currently mixed into .2/.3/.8. Backend execution, export coverage admission and reached-graph semantics stay held.

**Size:** M
**Files:** framework DSL Claims/Machine/Syntax, native Query tests/examples and local guidance.
**Touches:** [model/framework/dsl/**, model/framework/samples/**, model/framework/README.md, model/framework/migrate/README.md]

### Approach
- Implement reach/open/checkedBy/confirm under the parent's Native query configuration policy. Reuse Limits in Claims.scala:90 and the shared Declares ownership seam in Machine.scala:60, including Composition. Do not introduce a separate composition settings language or mutable global default.
- Pin frameworkReach's exact fields independently. Validate reach at native consumption, without toIR/export and without globally changing ordinary Limits zero validity. Preserve all inherited/replaced reach settings and avoid base-constructor reads of overridden vals.
- Consolidate configuration copying across Query live/explore at Claims.scala:115 and Witness requiring/ends/limits/live/explore at Claims.scala:223. Preserve specialized Witness authoring after new modifiers, exact predicates/selector/owners/names/limits/expectedRun/exploration, and source reach from the Scenario owner rather than the Property owner.
- Retire public Query.total authoring and migrate native callers/docs with no compatibility alias. Internal IR/carrier reader retirement remains .3's later obligation; do not force a schema migration now.
- Apply the explicit single-assignment/open/checker/confirmation policy; preserve exact reason text and authored confirmation order. Accept all three checker identifiers as metadata without availability/coverage checks. Prove the checkedBy-before-confirm ordering rule, not universal setter commutativity.
- Remove the four native ReliableBroadcast total clauses and obsolete count comment, and replace Witness.test.scala's public total getter assertion with independent declaration pins. External negative clients must refuse both getter and writer, including Witness. A private compile-only carrier bridge, if needed, earns no held integration credit.
- Keep configuration distinct from a checked answer. No backend invocation, export, replay, coverage admission or apparent successful result is introduced by this task.

### Investigation targets
**Required:** model/framework/dsl/Claims.scala; model/framework/dsl/Machine.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/IrFile.test.scala; model/design/reachable-exploration.md.

### Verification
Run native query/machine/syntax suites and external author compile tests with nonzero counts. No schema/IR/exporter/Go/backends or Temporal inputs.

Use the focused native gate shape `mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only '<native suite>'`. Re-anchor concrete selectors after committed .11. Do not run whole-state table helpers as admission gates. External clients use the native strict warning flags, including Werror/deprecation, and no old Model/Temporal jars. Independent poison callbacks must show metadata construction, copying and inspection performs no model execution.

## Acceptance
- [ ] Native reach/open/checker/confirmation authoring retains exact typed configuration, defaults, subject and limits; malformed configuration rejects.
- [ ] Machine/composition defaults, intentional overrides, derivations/replacement and refined-Property/Scenario ownership retain reach without initialization hazards; bounds reject negatives/zero search while zero depth remains valid.
- [ ] Find/verify and ordinary Query/Witness chains preserve every unrelated field and specialized authoring in both modifier orders where permitted. Nonblank/repeated open, repeated primary, empty/duplicate/repeated/self confirmation, and primary-after-confirm cases have independent diagnostics/control tests.
- [ ] Public total authoring is inaccessible externally and native callers/docs use the final surface.
- [ ] Independent ordinary Query declaration pins and native tests pass; configuration is never reported as backend execution or a checked answer.
- [ ] Full R2/R3 engine/carrier retirement and R8 coverage/replay/backend proofs remain open.

## Done summary
Native authors can retain reach, open acceptance and typed checker selections across ordinary Query and specialized Witness chains. Public total getters and writers are retired, and the four ReliableBroadcast clauses and obsolete count comment are removed.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: session (jev-unavailable(no_key))

### Native acceptance

- `frameworkReach` independently pins name `frameworkReach`, steps 256, actions 0 and search 65536. Machine and Composition share Declares.reach. Derived/Built operations and both composition derivation forms inherit source reach, with deliberate val overrides retained. Default compositions keep the framework default even when a member uses custom reach.
- Query.reach consumes its Scenario owner's bounds, including a refined owner's Property. Native validation rejects negative steps/actions and nonpositive search. Zero depth and positive compatibility actions remain valid. Ordinary all-zero Query Limits remain valid at native construction, and existing residual-signature tests retain their independent serialization rules.
- Immutable configuration preserves exact reasons, ordered confirmations, explicit-primary assignment, expectedRun and exploration through one Query copy seam and specialized Witness copies. Umpire, Apalache and TLC are accepted metadata. Open and checkedBy reject repeated assignment. Confirmation rejects empty, duplicate, repeated and self-confirming selections. checkedBy after confirm reports the required ordering.
- QueryConfigurationTest has independent find/verify declaration, owner, selector, predicate, name, start, schedule and Limits pins. WitnessTest exercises new modifiers before and after typed recordsWhen/excludes/ends/limits/live/explore refinements, with independent true/false predicate controls. Registered poison guard/effect, init/end, ordinary Property and Witness callbacks remain unexecuted by metadata construction, copies and inspection.
- Separate author/client jars compile under all native warning flags, including Werror/deprecation. The client reads public fields directly, preserves specialized Witness returns, passes 48 runtime checks and observes eight line-5 compile refusals covering Query/Witness total getters/writers and unknown/wrongly typed checker arguments.

### Initialization controls

Local Derived reach-val overrides exposed the existing constructor diagnostic's cold-this escape through name/objectName(this). The same defect was independently reproduced in a local Composition override. Constructor diagnostics now pass getClass to the shared naming algorithm, retaining val authoring and strict Wsafe-init. The unchanged Derived tests failed in `fn17312-query-verified.log` and passed in `fn17312-query-initialization-control.log`; restoring only Composition's old diagnostic failed in `fn17312-composition-initialization-red.log`, and the final 17-test run passed with both corrections.

Throwing expectation blocks, including a named helper's transitive throw, retained their existing refusal. The framework poison Property uses existing private sameStep construction; the public proof keeps the ordinary pure `check(after.state)` expectation. No expectation validator or public authoring boundary changed. Existing derivation construction's native init/refinement reads are unchanged; metadata configuration adds none.

### Verification

Baseline: green. The actual pre-edit WitnessTest command exited 0 with 6 tests at `71ed33293ef0394a1b01f30a785f505e6bdb1446`; log `fn17312-baseline-witness.log`. No baseline handoff or full-suite receipt was reused.

All final native suites use `mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only '<selection>'` and exited 0. Selections and observed passes are QueryConfigurationTest 17, WitnessTest 7, *Suite 90, RulesTest 26, Expect* 19, SugarNativeTest 5 and SampleExpectTest 6, all in framework.dsl. These total 170 focused passes. `fn17312-focused-final.log` and `fn17312-verified-*.log` retain every command's result and nonzero count.

- `mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite'` exited 0 with 29 tests, scanning 108 native files with zero findings; `fn17312-lint-final.log`.
- `mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn17312-external-recipe.sh` exited 0. Fresh native package, separate author package and public runtime stages each exited 0; all eight negative clients exited 1 with the intended typed/access diagnostic at line 5. Logs and scaffold are `fn17312-external-*`.
- Scoped Scalafmt formatted six changed Scala files; its final --check exited 0. Git diff --check passed. Logs `fn17312-format-final.log` and `fn17312-format-check.log`.
- Classification returned FULL, exit 1, for unmatched model/framework/README.md. Focused receipts under `.flow/tmp/green-receipts/ea02eb02-*` certify only their exact named commands. Spec validation returned valid with no errors or warnings. No docs-only skip or broad/native-table/law gate is claimed.

Initial missing-API red evidence is `fn17312-query-red-clean.log`. Earlier fixture import and invalid expectation-body failures were corrected without production API workarounds. The initial relative recipe launcher exited 127 with `bash: .flow/tmp/fn17312-focused-recipe.sh: No such file or directory` before any suite started; that observation is inconclusive and earns no passing credit. The absolute recipe launch supplied the final observed 170 passes.

### Handover and held work

Source commit is `ea02eb02db7370bb3bd0ef70f13aa924d0ab70d0`. The complete base range also contains conductor-owned fn-123 metadata-only commits `5a90a18ded36d9c2003d5359fddca314a952ecc1` and `98f47a25511b46ea6db97d5fc9cc93acdc14c8b6`; neither implements .12. Three disjoint assistants owned production, native tests and external/docs preparation. This worker reconciled them and owned every compiler, source commit and lifecycle operation; all assistants and started commands are terminal.

The minimal private[framework] total reader returns None solely for held exporter source compilation. It remains explicit carrier debt. Full R2/R3 engine/carrier retirement, R8 export coverage/replay/backend decisions, interpreted numeric parity, reached-graph proofs and original .2-.9 remain open. These declarations are metadata and supply no checked answer, backend availability result or exploration adequacy evidence. No Temporal sources, exporter/reify/schema/Go/backend/toolchain changes, regeneration or MILESTONES edits were added by this worker. Formal review and broad gates remain the owner's native batch-end work. The conductor retains fn-123.9 admission and parent lifecycle ownership.

Tracker sync: n/a (bridge inactive).
stage: plan-sync - skipped(config: planSync.enabled=false)
## Evidence
- Commits: 5a90a18ded36d9c2003d5359fddca314a952ecc1, 98f47a25511b46ea6db97d5fc9cc93acdc14c8b6, ea02eb02db7370bb3bd0ef70f13aa924d0ab70d0
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.QueryConfigurationTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.WitnessTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.*Suite', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.RulesTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.Expect*', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.SugarNativeTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.SampleExpectTest', mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn17312-external-recipe.sh, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/dsl/Claims.scala model/framework/dsl/Machine.scala model/framework/dsl/Compose.scala model/framework/dsl/QueryConfiguration.test.scala model/framework/dsl/Witness.test.scala model/framework/samples/ReliableBroadcast.scala, git diff --check, INCONCLUSIVE: initial bash .flow/tmp/fn17312-focused-recipe.sh exited 127: bash: .flow/tmp/fn17312-focused-recipe.sh: No such file or directory; no suite started. Repaired absolute recipe supplied 170 focused passes.
- PRs: