---
satisfies: [R1, R5, R9]
---
# fn-123-declare-faults-as-the-environments.9 Native fault declarations, budgets and realization choices

## Description
Implement the framework-native authoring portions of .1/.5/.6 after the settled input/domain APIs. Go exploration, IR/schema carriers, lowering and Temporal providers stay held.

**Size:** M
**Files:** model/framework/dsl/Action.scala, native binding/syntax seams, realize and focused tests.
**Touches:** [model/framework/dsl/**, model/framework/realize/**, model/framework/README.md, model/framework/samples/ReliableBroadcast.scala]

### Approach
- Re-anchor fault-party actions to current Action/Declares; add crash/responseLoss/storageLoss metadata and nonempty modelOnly reasons without alternate actors or compatibility spellings.
- Implement typed budgetedBy field binding on Boolean or Int domains, including bounded numeric carriers still supported by the final API. The approved fn173 amendment permits plain Int; do not require a finite state catalog or introduce a new bounded wrapper. Reject duplicate/wrong-field/wrong-type selectors and preserve the budget as ordinary native state. Global reached-state Go budget checks remain later; native declaration acceptance does not prove them.
- Retain native realization choosing metadata and check performed/model-only action/choice contracts at the native boundary that owns them; do not claim witness-lowering or manifest enforcement.
- Follow the parent's Native declaration seams: couple exact ordered choice tokens and deferred typed arms in one native program accepted by existing binding adapters. Preserve ordinary eager choose and callback behavior; do not introduce detached inventory assertions, execute effects to inspect choices, or retain stale choice metadata after rebind. Use identity rather than token names/Step equality.
- Enforce explicit selected-token membership at machine-aware completed realization/kit admission, with direct Performance/modelOnly bypass controls. Omitted selection means all, not an empty subset; opaque evidence cannot justify explicit selection. Add poison-arm controls for metadata construction and admission, and record/scalar/class-filter controls for program execution.
- Reuse ActionDecl copy/boundTo metadata and the native Choice/Performance seams. Retain choice identity explicitly rather than executing arbitrary effects to rediscover flattened choices. Migrate the framework ReliableBroadcast fault declaration if missing-kind enforcement reaches it; no disabled Temporal caller migration.
- Add framework-only fixtures and local author guidance. No task-queue/mini migration or new schemas.

### Investigation targets
**Required:** model/framework/dsl/Action.scala; model/framework/dsl/RuleBindings.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Machine.scala; model/framework/realize.

### Verification
Run touched native action/rule/realization/check selections, with external-author type/refusal probes and nonzero executed tests. Preserve independent no-fault native rule pins.

## Acceptance
- [ ] Fault kinds and nonempty modelOnly declarations survive native construction; missing/duplicate/invalid metadata rejects.
- [ ] Native budget selector/choice authoring is typed, rejects invalid bindings and preserves ordinary state/default/step behavior.
- [ ] Choice inventory and executable arms are structurally coupled; exact-token selection, duplicates, empty selection, wrong machine/class and opaque evidence refusals have independent controls without evaluating callbacks during admission.
- [ ] Framework-only examples and native positive/negative tests pass; global budget, witness lowering, schema and Temporal proofs remain open.

## Done summary
# Native fault declaration checkpoint

Implemented .9's framework-native fault kinds/modelOnly metadata, typed per-binding Boolean/Int/UpTo budget fields, immutable deferred choice programs, and exact-token native performance selection. Fault attachment preserves ActionDecl/Actor identity and RecordAction/ScalarAction shapes; budget attachment neither reads state nor executes or alters authored effects. Derivations retain coupled programs; unrelated rebind effects drop stale choice inventories while retaining binding budgets.

Completed Machine table acquisition validates fault/budget metadata. Performance construction rejects modelOnly declarations, including direct construction; final nativePerformances/dependencies acquisition enforces machine/class/choice membership for generic and custom kits, without a voluntary hook or callback evaluation. Custom kits must supply protected declaredPerformances and declaredDependencies. Unknown custom serialized script agreement is not claimed. ReliableBroadcast received only its missing crash tag; README documents the native authoring route and kit protocol.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none)

### Evidence

Baseline: green, pre-edit framework.dsl.RulesTest executed 26 tests with exit 0 (.flow/tmp/fn1239-baseline-rules.log). The earlier unsupported union/ChoicesTest selector collected zero tests and exited 1; .flow/tmp/fn1239-baseline-native.log is inconclusive, not passing credit. No baseline handoff was used.

Final verification: every command was awaited to terminal completion; no parallel compilers. 132 focused native tests passed: Rules26, Facets12, Choices4, BudgetField4, Input51, Sugar5, SampleExpect6, Native18, NativePerformance6. Logs: .flow/tmp/fn1239-verify-1.log through -8.log and fn1239-native-located-final.log. The six performance tests include enforced custom-kit acquisition controls with no manual validation hook. Lint selection passed 29 tests (fn1239-lint-final.log). Formatting check passed all 15 changed Scala files (fn1239-format-check-final.log); git diff --check passed.

Independent external proof rebuilt a framework library, a separate public author library, and a separate client under the native strict warning flags. Client executed 18 positive checks with callbacks=0. Eleven independent runtime refusal clients covered wrong actor, foreign same-spelled choice, foreign action, empty/duplicate choice selections, direct modelOnly performance, opaque evidence, wrong budget owner, final kit native/dependency acquisition, and wrong class. Four compile-refusal clients rejected wrong budget type, foreign root, computed field, and wrong state at their pinned source line. Recipe: .flow/tmp/fn1239-external-recipe.sh; aggregate log: fn1239-external-final.log. All terminal exit observations were checked. Initial external harness path/name errors were inconclusive and corrected in ignored fixtures, not production workarounds.

TDD evidence includes pre-implementation missing-API failures (fn1239-red-native.log, fn1239-red-budget.log); real red-to-green controls for late modelOnly mutation (red-stale-performance4), Rules budget source spelling (red-budget-derivation), budget retention after rebind (native-integration1), multiple guarded program token union (native-union-red), enforced kit protocol (red-kit-protocol), and located metadata failures (red-locations). Prefix each basename with .flow/tmp/fn1239- and append .log where omitted.

R1/R5/R9 receive native authoring/admission credit only. NativeFault, NativeBudget, BudgetField, NativeChoiceExecution and NativePerformance tests exercise the task acceptance controls; existing Rules/Facets/Input/Choices/Sugar/SampleExpect pins preserve ordinary nonfault behavior and inference. Metadata construction/admission poison callbacks stay unexecuted; dedicated execution controls subsequently pin ordered record/scalar arms, class filtering, Nil arms, and per-arm step cardinality.

Gate classification returned FULL (exit 1, unmatched MILESTONES.md in the cumulative range). No docs-only/full-suite skip or blanket green receipt is claimed. Per owner scope, verification is the focused native selections/check/lint and strict external proof above, not historical IR-bearing parent gates. Source warning flags, including Werror/deprecation, remain unchanged. Runtime JDK Unsafe warnings are inherited library behavior, not suppressed source diagnostics.

### Scope and provenance

Source commit: 36e1fe870f0f434927fb9773a0cba582df54a643.
Pre-edit base: bac173188b63b52031d942db7d67c0286f636d95.
The full contiguous range also contains root-owned metadata-only commits f2d93a1bcc6d6d8034b8f629c14c51ba7ac18df6 (exact sample Touches), 4b92f38168cee4166d5d713a4f3bfc7e516118f4 (held fn173 tail/MILESTONES), and adc3669159cd0385bb1c7f9451439b15d70a9983 (fn153 policy). They are not .9 source implementation.

Two delegated assistants (budget_fields and native_performance) completed and were reconciled before final verification; neither owns a live command or pending write. All compiler/verification handles are terminal. Root metadata windows were explicitly released before source staging.

Limits: .10 durability/generated crashes/assumptions/runtime budget enforcement remains next and unstarted. No automatic budget guard/decrement, global Go reached-state budget rules, capture/IR/schema/transport agreement, lowering/witness/manifest proof, exporter/Go/backend/toolchain/regeneration, disabled Temporal migration, full-model tests, whole-state StepTables/MachineLaws admission, or original .1-.8 hold release. Formal review and broad gates remain deferred to the owner's native batch end. Existing generic ir-scalapb.jar was the only allowed native generated dependency. No Temporal sources or legacy model/API jars were used.

The source/Flow lane is released only after the committed receipt and final clean-tree verification reported by this worker.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f2d93a1bcc6d6d8034b8f629c14c51ba7ac18df6, 4b92f38168cee4166d5d713a4f3bfc7e516118f4, adc3669159cd0385bb1c7f9451439b15d70a9983, 36e1fe870f0f434927fb9773a0cba582df54a643
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.RulesTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.FacetsTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.Choices', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.BudgetFieldTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.Input*', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.SugarNativeTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.SampleExpectTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.Native*', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.realize.NativePerformanceSuite', mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1239-external-recipe.sh, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/dsl/Action.scala model/framework/dsl/Budget.scala model/framework/dsl/StateField.scala model/framework/dsl/Machine.scala model/framework/dsl/Syntax.scala model/framework/dsl/Facets.test.scala model/framework/dsl/BudgetField.test.scala model/framework/dsl/NativeBudget.test.scala model/framework/dsl/NativeChoiceExecution.test.scala model/framework/dsl/NativeFault.test.scala model/framework/realize/Constructed.scala model/framework/realize/Realize.scala model/framework/realize/NativePerformance.test.scala model/framework/realize/Conversion.test.scala model/framework/samples/ReliableBroadcast.scala, git diff --check
- PRs: