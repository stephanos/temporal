---
satisfies: [R3, R5]
---
# fn-165-inputs-as-records-named-classes.8 Native record execution and structural shared-input selectors

## Description
Bind native record inputs through effects/rules and framework-owned shared-input helpers using .7's settled representation.

**Size:** M
**Files:** model/framework/dsl/{ActionInputs,Machine,CoreBindings,RuleBindings,Syntax,MemberPath}.scala, native input/field descriptor helpers and focused effect/input tests; reached framework realization helpers only.
**Touches:** [model/framework/dsl/**, model/framework/realize/**]

### Approach
- Add S=>Steps and (S,In)=>Steps paths sharing the existing rule execution seam. `In` is the actual declared record or scalar A; do not pass a synthetic one-field record/tuple to scalar effects. Preserve field values, class restrictions and selected input domains; no second evaluator.
- Consume .7's genuine Action[In], RecordAction/ScalarAction and retained SuppliedInput[In] seam. Native rule execution and class enumeration must retain actual inputs through Class copies, aliases, facets and derivations without repeating record constructors/default evaluation per callback. Native invocation uses one record/scalar adapter, including records beyond six fields, not a second arity ladder. LegacyAction and its old tuple invocation remain temporary .9 retirement work.
- Replace shared-token identity in generic native helpers with checked field selectors and structural same-label/same-type contracts. Reuse `MemberPath.directField` in model/framework/dsl/MemberPath.scala for direct-field recognition and computed/foreign-root refusals; add input-specific typed owner/field evidence, not another AST walker. `MemberSelection` is a composition-specific `(Any, Model)` carrier, not input-field evidence.
- Capture a typed field descriptor at the concrete record authoring boundary and forward it into a structural helper such as `[In <: { def scheduleToStart: Timeout }]`. Prove two distinct concrete records and descriptor forwarding; an opaque selector function or abstract bound alone supplies no concrete field evidence. Anchor expected `In` before typing `_.field`, following callable `RequirementFacts` in model/framework/dsl/Syntax.scala. Prove a framework-local helper if the historical deadlines consumer exists only in disabled Temporal.
- Add wrong arity/type, missing field, same-name wrong-type and foreign-selector negatives. Preserve source label/field order for later export without implementing capture.
- Adapt native callback/signature validation where it assumes one function parameter per flattened input: a native callback has state plus one actual In parameter. Keep no-input, class-specific and heterogeneous multi-target state-only rule behavior. Do not repair held exporter/capture consumers to make the old flattened contract appear current.

### Investigation targets
**Required:** model/framework/dsl/ActionInputs.scala; model/framework/dsl/Machine.scala; model/framework/dsl/CoreBindings.scala; model/framework/dsl/RuleBindings.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/MemberPath.scala; model/framework/dsl/InputRecords.test.scala.

### Verification
Run touched native Inputs/Effects/Rules/realization selections with nonzero tests and independent old/new step-value pins. Full exporter and Temporal deadlines migration remain held.

## Acceptance
- [ ] Native rule functions receive the exact supplied record or scalar A through (S,In), including scalar products and records beyond six fields, without duplicate constructor/default evaluation or another arity ladder. No-input, class-specific and heterogeneous multi-target state-only behavior retains independent pins.
- [ ] A structural shared-input helper reads the same-label/same-type field from two distinct concrete records using captured typed descriptors, including generic descriptor forwarding, without reflection, casts, unchecked string lookup or synthetic Model tuples. Opaque selectors without forwarded evidence are explicitly refused.
- [ ] Wrong fields, types, selector owners and arities reject; focused native regressions pass.
- [ ] No complete R5/Temporal consumer or lifted field-read proof is claimed.

## Done summary
Native actions now execute callbacks with the actual record or scalar input through the existing rule seam. Concrete typed field descriptors support structural shared-input helpers and generic forwarding through compiled getters.

stage: impl-review - skipped(config: REVIEW_MODE=none)
stage: plan-sync - skipped(config: planSync.enabled != true)

### Native acceptance

- `Action[In] ~> (S, In) => Steps` and native `on(action)` callbacks preserve the declared input type. `RuleInput` distinguishes the native input from the temporary legacy tuple callback shape. Direct and generic scalar Tuple1/Tuple2 bindings have positive controls. Wrong records, scalar shapes and flattened callback arities refuse compilation.
- Class-aware enumeration retains each selected catalog object's identity. Supplied classes, metadata copies, aliases, facets and replacements use the same native adapter. A changed-values copy reconstructs its record once and retains that recovered value for later callbacks. No record constructor or default runs per native callback.
- Seven-field records execute through one actual-input adapter. Independent expected Steps pin ordered alternatives/facts, class restrictions, replacements, scalar products/tuples, no-input actions and heterogeneous multi-target state-only rules, including native class targets.
- Native class rules retain the authored qualified target spelling before Dynamic constructor expansion. Their target expression and defaults execute once during registration. A bare native class target still refuses qualification.
- `inputField[In](_.field)` and `RecordAction.field(_.field)` anchor the concrete owner before lambda typing. `InputField[In,A]` retains a label and typed getter. Two distinct records forward descriptors through a structural shared-input helper. Missing/wrong fields, foreign owners/roots, computed selectors, opaque functions, abstract owners without forwarded evidence and scalar-product attachment refuse. Capture reuses `MemberPath.directField`; descriptor reads use no reflection, casts or string lookup.
- Core callback arity validation recognizes state plus one actual native input. Rule and derivation validation share the selected record-schema-root check while scalar and legacy signatures retain their declared shape. Metadata-only tests use throwing catalogs and add no enumeration.
- Temporary compatibility receivers now name `LegacyAction` where their tuple overloads otherwise intercepted native scalar-tuple calls. Channel delivery/loss actions retain that explicit legacy discriminator until .9. Widened `Action[Tuple]` empty-call admission remains in `InputSource` for .9 retirement.
- Framework-local guidance documents native execution and concrete descriptor forwarding.

### Verification

Baseline was green. `mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests` exited 0 with 288 tests before edits. Log `.flow/tmp/fn1658-baseline.log`. The native receipt check missed, so the suite ran.

Final formatted source verification:

- The same full native command exited 0 with 301 executed tests. Log `.flow/tmp/fn1658-native-final.log`. This includes seven `InputExecution`, four `InputFieldsTest`, and one new native metadata test each in `CoreBindingsSuite` and `RuleBindingsSuite`.
- `mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite'` exited 0 with 29 tests. The native block-form scan visited 102 files and found zero findings. Log `.flow/tmp/fn1658-lint-final.log`.
- Scoped Scalafmt on 15 changed Scala files passed, its second pass preserved every SHA-256, and `fmt --check` exited 0. Logs `.flow/tmp/fn1658-format{,-second,-check}.log`; manifest `.flow/tmp/fn1658-format-before.sha256`. `git diff --check` passed.
- `mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1658-external-recipe.sh` exited 0. A fresh native jar, separate declarations jar and external runtime client passed. Four separately compiled negative clients refused another record, computed field, foreign-root field and opaque selector at their pinned call sites. Logs `.flow/tmp/fn1658-external-final.log`, `.flow/tmp/fn1658-native-package.log`, `.flow/tmp/fn1658-external-{package,positive,wrong-record,computed-field,foreign-field,opaque-field}.log`. The runtime client drives native rules through public `StepTables.lines`, retains selected record identity, captures descriptors across jars and forwards generic bindings.
- Gate classification was FULL, exit 1, because `model/framework/README.md` is unmatched. Log `.flow/tmp/fn1658-gate-classify.log`. Native receipt `.flow/tmp/green-receipts/7e209546-native-unittest.json` certifies only the exact native command above. No gate skip is claimed.

Red-to-green evidence is retained. `fn1658-execution-red.log` failed the seven-field arity cap and missing native bindings. `fn1658-fields-red2.log` refused the missing descriptor surface before implementation. `fn1658-signatures-red3.log` failed both old flattened native arity checks. `fn1658-default-class-red.log` observed duplicated target/default construction. `fn1658-native-focused5.log` exposed direct scalar-Tuple1 binding interception; focused6 passed all 33 input tests after narrowing legacy receivers. The final native run passes these cases, including bare-target refusal and recovered-copy caching.

Intermediate compiler fixture errors and the empty selection in `fn1658-signatures-red2.log` are inconclusive observations and earn no passing or behavior-red credit. Lint's missing backticked core-form documentation was corrected in `Syntax.scala`; no test assertion, gate, exemption or source root was weakened.

### Handover

Base `3575e66bfcc9b70b61a62875a6fa865e876d13bd`. Implementation commit `7e209546b2b23dcc8a6149e28a08818e4eda2703`. The range also contains conductor-owned plan-only commit `b319c6534eee5dccbad60dd9d53c096b1a276ff7`, which is not .8 implementation. Two delegated workers returned a read-only test survey and the field descriptor files. This worker owned compiler commands and commits.

Only native .8 closes here. .9 retains caller migration, the migration recipe and retirement of LegacyAction, tokens/Assigned/class `:=`, InputAt, tuple empty admission, arity bindings and temporary RuleInput legacy evidence. Original .1-.6 remain held. Complete R3/R5, the Temporal deadlines consumer, lifted input-field reads/refusals, exact IR/Param/Class/default parity, schemas, Go/backends, exporter integration and regeneration are not claimed. No Temporal source was read or restored. No MILESTONES edit, broad Make gate, old Model/API jar or project.scala root was used. Review remains at the owner's source-batch boundary.

Tracker sync: n/a (bridge inactive).
## Evidence
- Commits: b319c6534eee5dccbad60dd9d53c096b1a276ff7, 7e209546b2b23dcc8a6149e28a08818e4eda2703
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests, mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1658-external-recipe.sh, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/dsl/ActionInputs.scala model/framework/dsl/Channel.scala model/framework/dsl/CoreBindings.scala model/framework/dsl/CoreBindings.test.scala model/framework/dsl/DerivationBindings.scala model/framework/dsl/InputExecution.test.scala model/framework/dsl/InputField.scala model/framework/dsl/InputField.test.scala model/framework/dsl/Machine.scala model/framework/dsl/RuleBindings.scala model/framework/dsl/RuleBindings.test.scala model/framework/dsl/RuleInput.scala model/framework/dsl/RuleSource.scala model/framework/dsl/StepTables.scala model/framework/dsl/Syntax.scala, sha256sum -c .flow/tmp/fn1658-format-before.sha256, git diff --check
- PRs: