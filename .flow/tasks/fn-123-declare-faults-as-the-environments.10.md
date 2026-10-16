---
satisfies: [R2, R3, R4, R5, R6]
---
# fn-123-declare-faults-as-the-environments.10 Native durability, crash derivation and storage-loss assumptions

## Description
Complete native durability/crashes behavior from .2 and the native portions of .4/.5. Original tasks retain IR, Go crash parity and provider migration proofs.

**Size:** M
**Files:** framework DSL Machine/RuleBindings/Assume/Syntax, native tests, ReliableBroadcast classification and local guidance.
**Touches:** [model/framework/dsl/**, model/framework/README.md, model/framework/migrate/README.md, model/framework/samples/ReliableBroadcast.scala]

### Approach
- Classify state fields with durable/ephemeral/inMemory and bind crashes through the existing native rule mechanism. Derive a native crash from the same retained classification, not a second hand-authored semantics.
- Validate full field coverage, whole/inner path overlap, durable fallbacks, duplicate mappings, out-of-domain reset, classification without crash and crash without classification. Budget fields on crashing machines must remain durable. Validate on construction/registration before first evaluation; no dormant declaration bypass.
- Preserve valid inherited classification through restriction, including removal of all crashes, but validate every retained nonempty descriptor set. A derived machine's own new classification still requires an active crash; derivation alone is not an admission exemption. Apply budget-path durability checks only while a crash is active. Pin retained restriction, new derived classification refusal and crash-free authored noncrash budget extension without callbacks or a new consumption wrapper.
- Preserve authored-over-derived rebind meaning and storageLoss's ordered one-per-fault derived assumption, refusing hand-authored duplicates.
- Native assuming currently discards its arguments: add ordered assumption storage and propagation through built/derived/restrict/rebind/extend before relying on storageLoss deduplication. Preserve authored-before-derived order and one stable assumption identity for the same fault across machines; no serialized RuleBindings packet may become a native evaluator.
- Follow the parent's Native declaration seams for completed-table/metadata admission timing, typed reset-validity evidence and exact authored-assumption multiplicity. Pin successive authored A,B then C,A as A,B,C,A; only generated storage-loss entries coalesce. Preserve internal fairness duplicates/empty segments and refuse authored/generated collisions. Invalid never-invoked crashes must refuse before executable native acquisition, without base-constructor forcing or state callback execution.
- Implement the parent spec's explicit 2026-10-16 budgeted derived-crash amendment: durability reset preserves every budget field, then the same atomic transition consumes only the firing derived crash's own budget once (`true` to `false`, positive `n` to `n-1`). Exhaustion disables before reset evaluation; other budgets are unchanged. All budget fields remain classified durable. Authored fault effects and authored crash overrides manage their own consumption and must not be automatically decremented again. Unbudgeted crashes stay unchanged; no hidden counter or added state field. Use fn-173 plain-Int/reached-state contracts, not whole-state catalogs; global Go budget enforcement remains held.
- Apply the parent's negative-Int boundary policy only when a generated crash consumes its own budget. Refuse a negative value with fault/binding location before reset/decrement; independently pin negative refusal, zero exhaustion and positive consumption with a poison reset. Do not evaluate model state at registration or add native enforcement wrappers around authored effects/overrides.
- Apply the parent's own-budget alias boundary: reject an active generated crash whose own resolved state field path is also another active fault binding's budget in that machine. Refuse at completed admission before callbacks, distinguish separate bindings from repeated internal rule aggregation, and retain distinct-field positive controls. Authored effects/overrides alone gain no blanket shared-path ban, automatic consumption or global budget enforcement.
- Pin native derived results against independently authored ordinary crash functions on small generic fixtures, including every fallback and rebind case. Do not read historical Temporal providers.
- Add the required durability classification to ReliableBroadcast's existing authored crash without changing its guard, effect, facts, outcome or enabledness. Do not replace that callback with an unconditional generated crash; the broader native sample representation rewrite remains held under fn-173.14.

### Investigation targets
**Required:** model/framework/dsl/Machine.scala; model/framework/dsl/RuleBindings.scala; model/framework/dsl/Assume.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Rules.test.scala; model/framework/dsl/Domain.scala.

### Verification
Run native fault/rule/assumption/refinement tests with nonzero counts. No Go, schema/exporter, regeneration or model/temporal inputs.

## Acceptance
- [ ] Native crash derivation matches independent per-field truth/result pins and supports authored override through rebind.
- [ ] All invalid classification/fallback/crash/budget combinations reject at the owning native declaration boundary, including new classification on a crash-free derived machine. Valid inherited restriction and crash-free authored budget extension remain accepted without bypassing retained descriptor validation.
- [ ] Independent Boolean/numeric pins prove exactly-once own-budget consumption, exhausted disabling before reset evaluation, other-budget/durable-field preservation, unchanged unbudgeted results and no double consumption of authored overrides.
- [ ] A negative plain-Int own budget refuses before reset with a located fault/binding diagnostic; zero and positive paired controls preserve the adopted semantics without state enumeration or registration-time callback execution.
- [ ] Generated own-budget aliasing with another active fault binding refuses before callbacks; distinct fields pass, and an authored override does not retain the generated-only alias restriction or consumption wrapper.
- [ ] Storage-loss assumptions retain deterministic identity/order, coalesce generated entries per fault and reject authored contributions duplicating a generated fault assumption.
- [ ] Authored assumptions retain exact multiplicity/order through copies and fault restriction; native acquisition rejects invalid dormant crash declarations without forcing uninitialized subclass state or evaluating model callbacks.
- [ ] Focused native checks pass; complete R3/R4 Go/provider proofs and all IR obligations remain open.

## Done summary
# Native durability task .10

Framework authors can classify typed state fields with `durable`, `ephemeral` and `inMemory`, then derive an atomic native crash through `crashes`. Completed native admission checks dormant declarations before callbacks, preserves authored overrides and retains exact authored assumption occurrences followed by fault-local generated storage-loss assumptions.

Task: fn-123-declare-faults-as-the-environments.10
Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none)

### Source and ownership

- Branch `umpire` in `/Users/stephan/Workspace/skunkworks/umpire/temporal`.
- Initial anchor `64d1672442ff8e66b2404c4651d6a57755734949`.
- Persisted pre-edit base `faf5b9df3f94ad03d82476943132097b7da278bf`.
- Source commit `7d6e1443453bf0be39764c6ec4d6f523cb5145c7`.
- Root-owned metadata commits in the range are `5b354525a7e21baaf38abf9306919923c3d08767` and `e9da129802962313b77a6d25d58a95a54ba28314`. They record the adopted alias and inherited-classification boundaries and are excluded from implementation attribution.
- The source commit changes only the declared DSL, framework guidance and exact ReliableBroadcast sample paths. ReliableBroadcast retains its authored crash guard, effect, facts, outcome and enabledness. Its classification maps each sending Origin to the corresponding crashed payload and preserves p1, p2 and network.
- Implementation children `assumptions` and `durability_fields` completed without commits or running commands. Root's external-fixture helper completed preparation only. This worker owned every compiler command and Git/Flow mutation outside the root's explicit metadata windows.

### Admission and execution pins

`DurabilityFieldsSuite` covers exact leaf coverage, nested/whole overlap, wrong owner, budget durability, duplicate mappings, durable fallback closure, forged bounded resets, selected narrower finite evidence, recursive collection membership and unsupported opaque evidence. Structural Data comparison alone admits no reset value.

`NativeCrashSuite` independently pins unmapped and mapped custody values, explicit outcomes/fact order/empty explanation, Boolean/plain-Int/UpTo consumption, exhausted and negative-own budgets before a poison reset, untouched other budgets, nested reset/write composition and authored overrides. Completed binding and empty custom realization acquisition reject invalid dormant declarations before init/end/effects/dependencies. Generated own-budget aliases refuse; distinct paths and all-authored shared paths pass. Crash-free restrictions retain validated classification, new descendant classification requires an active crash, and authored budgets on retained ephemeral fields keep their existing behavior.

`NativeAssumptionsSuite` pins authored `A,B` then `C,A` as `A,B,C,A` through seven derivations, fairness duplicates and empty segments, generated-only coalescing, unrelated same-spelled fault identities, authored collisions and inferred definition names across aliases and reversed acquisition order. No global token ownership or serialized RuleBindings evaluator was introduced.

### Verification

The baseline RulesTest command exited 0 with 26 tests before source edits. Its log is `.flow/tmp/fn12310-baseline-rules.log`.

Every native command below uses `timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only <selector>`. All completed with exit 0 and nonzero tests.

| Selector | Tests | Log |
| --- | ---: | --- |
| framework.dsl.RulesTest | 26 | fn12310-verify-RulesTest.log |
| framework.dsl.FacetsTest | 12 | fn12310-verify-FacetsTest.log |
| framework.dsl.Choices | 4 | fn12310-verify-Choices.log |
| framework.dsl.BudgetFieldTest | 4 | fn12310-verify-BudgetFieldTest.log |
| framework.dsl.Input* | 51 | fn12310-verify-Input*.log |
| framework.dsl.SugarNativeTest | 5 | fn12310-verify-SugarNativeTest.log |
| framework.dsl.SampleExpectTest | 6 | fn12310-current-SampleExpectTest.log |
| framework.realize.NativePerformanceSuite | 6 | fn12310-current-NativePerformanceSuite.log |
| framework.dsl.NativeDataTest | 9 | fn12310-current-NativeDataTest.log |
| framework.dsl.Native*Suite | 25 | fn12310-native-derived-final.log |
| framework.dsl.DurabilityFieldsSuite | 8 | fn12310-fields-current-final.log |

The sequence totals 156 selected native checks. The final admission change reran the 25 affected Native suites and 8 field checks; the other 123 compatibility checks are retained passes from the preceding verification stages, not reruns against the last guard. All logs live under `.flow/tmp/`.

- Syntax/comment lint exited 0 with 29 tests using `timeout 600 mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite'`. Log `fn12310-lint-final.log`.
- Formatting check exited 0 for all 12 changed Scala files with the existing `model/.scalafmt.conf`. Log `fn12310-format-final.log`. `git diff --check` exited 0.
- `bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn12310-external-recipe.sh` exited 0 against committed source `7d6e1443453bf0be39764c6ec4d6f523cb5145c7`. Log `fn12310-external-committed-final.log`. It rebuilt the native library, a separately compiled author library and independent public clients with Werror, Wsafe-init and the full strict flags intact. It passed 22 positive checks, 13 located runtime refusals with callback counts 0, and 6 compile refusals at line 10 with intended diagnostics. Wrong outcome/fact checks require E134, crashes, the exact Boolean/String overload and unmatched arguments.
- External evidence establishes public admission, metadata, assumptions and type/refusal contracts. Public APIs expose no typed machine executor. Actual reset/budget/negative-own/rebind execution proofs remain the independent internal native tests; no casts or new public executor were added to external fixtures.
- Gate classification returned exit 1, `FULL: unmatched: model/framework/README.md`. No docs-only skip or broad full-suite green receipt is claimed. Owner-scoped native gates replaced the explicitly held whole-model/IR/Go commands.

### Red-to-green and inconclusive observations

- Genuine semantic red logs are `fn12310-red-duplicate-crash2.log`, `fn12310-red-inferred-name2.log`, `fn12310-red-budget-alias.log` and `fn12310-red-derived-durability.log`. Each exited 1 because its expected native refusal or retained name was missing. The final 25 Native tests cover these regressions and exit 0.
- Initial missing-API red logs also contained test-harness shadowing or inherited State alias errors. They are recorded as mixed API/harness failures, not isolated behavioral proof. Integration compile errors were corrected before green claims.
- `fn12310-sample-selector-inconclusive.log` selected the wrong sample package and ran zero tests. The corrected framework.dsl.SampleExpectTest ran 6 tests with exit 0.
- The first external aggregate stopped at a proof-fixture Subject ambiguity before line 10. A later suffix stopped because its matcher omitted the intended E134 overload diagnostic. Those aggregates are inconclusive; their compile checks receive no pass credit. The corrected complete committed-source recipe supersedes them.
- The old `fn12310-crash-green1.log` contains compile errors despite its filename and receives no pass credit.

### Held limits and closure

Tasks .1-.8, fn-153 .7 admission, parent closure, all IR/exporter/schema/Go/backend/toolchain/provider migration proofs, global budget enforcement, whole-state catalog admission and the broader ReliableBroadcast representation rewrite remain held. No model/temporal or disabled Scala body was read, restored, compiled or migrated. No archive, generated artifact, old Model/Temporal API jar or whole-catalog gate was used. The existing IR dependency jar served only native compilation.

Framework-local before/after guidance documents durability/crashes and assumption migration. No Scalafix implementation or feature-map edit was made. Every child and compiler/gate command is terminal. The final receipt/status/spec-validation results and final range are recorded in the paired evidence file after closure.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5b354525a7e21baaf38abf9306919923c3d08767, e9da129802962313b77a6d25d58a95a54ba28314, 7d6e1443453bf0be39764c6ec4d6f523cb5145c7
- Tests: timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.RulesTest, timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.FacetsTest, timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.Choices, timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.BudgetFieldTest, timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.Input*, timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.SugarNativeTest, timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.SampleExpectTest, timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.realize.NativePerformanceSuite, timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.NativeDataTest, timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.Native*Suite, timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.DurabilityFieldsSuite, timeout 600 mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only framework.check.lint.*RuleSuite, timeout 600 mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/dsl/Action.scala model/framework/dsl/Assume.scala model/framework/dsl/Budget.scala model/framework/dsl/Machine.scala model/framework/dsl/StateField.scala model/framework/dsl/Syntax.scala model/framework/dsl/Crash.scala model/framework/dsl/Durability.scala model/framework/dsl/DurabilityFields.test.scala model/framework/dsl/NativeAssumptions.test.scala model/framework/dsl/NativeCrash.test.scala model/framework/samples/ReliableBroadcast.scala, bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn12310-external-recipe.sh, git diff --check
- PRs: