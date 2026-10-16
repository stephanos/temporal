---
satisfies: [R1, R2, R9, R10]
---
# fn-165-inputs-as-records-named-classes.7 Native record inputs: inventory, named classes and declared defaults

## Description
Deliver the native record declaration/class frontend. Original .1-.6 retain their IR/integration obligations. The whole fn-141 closure and fn-174.24 IR reconciliation do not gate this native work.

**Size:** M
**Files:** model/framework/dsl/{Action,InputSource,Domain,Syntax}.scala and Inputs/InputDefaults tests.
**Touches:** [model/framework/dsl/**]

### Approach
- Refresh the framework-only inventory from .flow/tmp/fn165-framework-source-inventory.md and fn165-framework-reanchor.md. Record every native action/default/order and token consumer, including throwing/empty/restricted Finite witnesses. Do not inspect disabled Temporal.
- Prove native `In = Start`, named/empty/record-value calls, and no-input calls before broad migration. Compile a generic `Action[In]` helper and a separately compiled external client: the frontend must preserve compiler-checked named/default arguments, not substitute tuple inputs or unchecked Dynamic/Any label assignment.
- Preserve actual scalar `In = A`, not merely value-call ergonomics through wrappers. Use required/defaulted scalar declaration overloads `.input[A](name = label)` / `.input[A](name = label, default = value)` with explicit exact labels and the selected `Finite[A]` witness. This returns an action, never a token or appendable input ladder. One-field records remain optional. Bare `.input[A]` must not manufacture a label from action/type names or domain order.
- Store scalar versus record shape explicitly; a scalar domain may itself be a product. Prove scalar value calls/examples, required versus defaulted omission behavior, non-first defaults, Option-valued defaults distinct from absence, and selected restricted/throwing witnesses without metadata-time enumeration.
- Preserve declaration labels, input type/order, explicit non-first defaults and selected Finite witness. Enumerate only when the caller requests classes, never to discover a default. Missing/default-outside-domain/duplicate-or-unknown-label/wrong-type forms must fail clearly.
- fn-159.10 is already source-landed and superseded in representation only; preserve its explicit-default behavior. Record native baseline evidence now; exact IR baseline is separately held.

### Investigation targets
**Required:** model/framework/dsl/Action.scala; model/framework/dsl/InputSource.scala; model/framework/dsl/Domain.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Inputs.test.scala; model/framework/dsl/InputDefaults.test.scala.

### Verification
Use isolated native compiler/test selection from fn164.2 with positive executed counts. Pin old/new native class order and defaults against independently recorded examples. No capture/exporter/IR/Go/Temporal checks.

## Acceptance
- [ ] Inventory covers native inputs, class calls, defaults, effect arities and shared/token identity consumers with no guessed label provenance.
- [ ] Native input type is the declared record, not a tuple surrogate. Empty, reordered named, record-value and exact-label one-field value calls compile and run, including a generic helper and separately compiled external client; unknown/duplicate labels, omitted required fields, another record and wrong field types fail compilation with positive controls.
- [ ] Genuine scalar actions retain `In = A` and value calls/examples without a required one-field record, using exact explicit labels and required/defaulted metadata. Scalar product domains are not misclassified as records; required omission rejects, declared defaults remain exact (including Option values), and metadata construction preserves the selected witness without eager enumeration.
- [ ] Class order and explicit defaults preserve independent native pins, including non-first defaults and custom Finite witnesses without eager enumeration.
- [ ] Focused native tests pass; deferred exact Param/Class/default IR proof and original .1-.6 remain open.

## Done summary
Implemented native record declarations, checked class calls and explicit scalar declarations with genuine `Action[In]` types. Record/scalar adapters retain selected finite witnesses, declared defaults and actual supplied input values; the native author guide documents the precise-shape helper contract.

stage: impl-review - skipped(config: REVIEW_MODE=none)

### Native acceptance and boundary

- `.inputs[Start]` produces `RecordAction[Start] <: Action[Start]`; `.input[A](name = label)` and its `default` overload produce `ScalarAction[A] <: Action[A]`. A product scalar remains one explicitly labeled scalar. No label comes from an action/type name or catalog order. Native declarations cannot be appended as an input ladder.
- Empty/named record calls lower authored-order `NamedArg` trees through Scala 3.9's checked `Select.overloaded` constructor application, with expected `In`. The compiler supplies defaults and preserves argument evaluation order. The receiver runs once before arguments. A one-field record value shorthand uses that record's actual field label. No unchecked field assignment or tuple surrogate implements these calls.
- Generic `Action[In]` helpers forward actual input values. Shape-dependent helpers preserve `RecordAction[In]`/`ScalarAction[In]`; non-tuple shape erasure deliberately refuses named/empty calls. Required record omissions compile-refuse. Required scalar omission remains a runtime refusal naming the explicit input. During coexistence only, widened `Action[Tuple]` empty calls retain legacy admission; the runtime adapter still keeps an actual scalar tuple as one labeled value. Remove that exception in .9.
- Selected scalar, record-field and whole-record witnesses are retained. Derived product/sum/Option catalogs are lazy, so constructing metadata does not enumerate throwing witnesses. Restricted/empty whole-record and scalar catalogs, default-outside-domain refusals and supplied native Int domains have native controls.
- Independent pins cover seven repeated Boolean fields, 128 classes with the last field varying fastest, non-first defaults, false/zero/None/Some/nested values and explicit labels. `None` defaults remain `Some(None)` metadata rather than required omissions. Declaration modifiers/examples preserve precise shape and witnesses.
- `Class.supplied: Option[SuppliedInput[?]]` retains the actual typed value, including record object identity. Metadata-only `Class.copy` retains it; changing values drops the stale carrier. .8 must carry actual inputs through whole-class enumeration/callbacks, add concrete typed field descriptors and reuse `MemberPath.directField` for selectors. Current `classesOf` still returns flattened core values; no record execution/selector completion is claimed.
- Baseline inventory: `.flow/tmp/fn1657-native-input-inventory.md` and `.json`, pinned to `8eafcf7ffd`, with detailed earlier manifest references and final disposition here. Existing native token identity, class order, defaults and effect arity consumers remain inventoried. Fn-159.10's explicit-default behavior is source-landed and preserved; only its representation is superseded.

### Verification

Baseline: green, 276 executed native tests before edits, exit 0, `.flow/tmp/fn1657-baseline-observed.log`. The first baseline output handle was lost by output extraction; `.flow/tmp/fn1657-baseline.log` is inconclusive and earns no exit-code credit.

Fresh verification of committed source `841fb0b5833859a7742f5e44e21397e215a0265f`:

- `mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests`: exit 0, 288 tests, including all 12 `framework.dsl.InputRecords` tests. Log `.flow/tmp/fn1657-native-committed-final.log`. Receipt `.flow/tmp/green-receipts/841fb0b5-native-unittest.json` certifies this exact native command only.
- `mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite'`: exit 0, 29 tests; native block-form scan visits 98 files with zero findings. Log `.flow/tmp/fn1657-check-final2.log`.
- Pinned formatter on the seven changed Scala files: second pass unchanged and `fmt --check` exit 0. Logs `.flow/tmp/fn1657-format-{second,idempotence,check}.log`; hashes `.flow/tmp/fn1657-format-before.sha256`. `git diff --check` passed.
- Fresh native library packaging exit 0, `.flow/tmp/fn1657-native-package-final.log`. Separate declaration-library packaging, separate external positive-control compilation and external runtime client all exited 0 through `.flow/tmp/fn1657-external-recipe.sh`; final aggregate log `.flow/tmp/fn1657-external-final.log`, runtime marker in `.flow/tmp/fn1657-external-positive.log`. This uses Scala 3.9.0/JVM 27 and the existing IR support jar, without compiling exporter/system inputs.
- Nine separate external negative clients refused unknown/duplicate labels, named/empty required omissions, wrong field/scalar types, another record, and erased named/empty shape. The recipe pins each intended diagnostic reason and call-site line; logs `.flow/tmp/fn1657-external-<case>.log` were inspected. Positive helpers cross the compiled library boundary: generic actual-value forwarding, precise named forwarding and inline generic defaults. Additional external pins cover no-input calls, one-field exact labels, modifier chains and scalar products.

Red-to-green: `.flow/tmp/fn1657-red.log` failed both missing frontend assertions. `.flow/tmp/fn1657-behavior-red3.log` failed authored evaluation order and metadata non-enumeration, then the eight-case green selection passed after compiler-owned named construction and lazy catalogs. `.flow/tmp/fn1657-one-field-red.log` refused `finish(true)` before the checked one-field lowering. Final committed verification passes all corresponding tests. Intermediate compiler/warning failures and the obsolete one-field diagnostic needle remain in their logs; no failed observation is credited as green. Full native coexistence errors were repaired with the bounded legacy empty-call adapter. Lint's reserved `record` helper-name finding was fixed by naming the core helper `recordDeclaration`, without changing the gate.

Gate classification: FULL, exit 1 (`unmatched: model/framework/README.md`), `.flow/tmp/fn1657-gate-classify.log`. The owner-amended native gates above are authoritative for .7. No fabricated docs-only/receipt skips or broad integration receipts are claimed.

### Handover

Source commit `841fb0b583`; base `8eafcf7ffdf649b39979cef0d089ee7428794410`. The evidence range also includes the conductor's plan-only `55dc59061f`. Two disjoint assistants returned inventory and external-proof scaffolding; the worker owned compiler commands and commits.

Next native task .8 owns record execution, typed field descriptors and structural selectors; .9 owns native caller migration, its migration rule/proof, and removal of `LegacyAction`, token/Assigned/`:=` class forms, `InputAt`, tuple admission and arity APIs. Do not release fn-173.10 before committed .9.

Not run/claimed: `make check-framework`, `make model-check-changed`, original whole-model Quick commands, capture/exporter integration gates, exact Param/Class/default IR parity, schemas, Go/backend work, regeneration or Temporal restoration/migration. Original .1-.6 and complete R4/R6/R7/integration obligations remain held; this is native .7 closure only. Review remains at the owner's native source batch boundary. MILESTONES.md was not edited.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 55dc59061faa63ab252c7c9955fb96a23618964c, 841fb0b5833859a7742f5e44e21397e215a0265f
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests, mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/dsl/Action.scala model/framework/dsl/ActionInputs.scala model/framework/dsl/Domain.scala model/framework/dsl/InputRecords.test.scala model/framework/dsl/InputSource.scala model/framework/dsl/Machine.scala model/framework/dsl/Syntax.scala, sha256sum --check .flow/tmp/fn1657-format-before.sha256, mise exec -- scala-cli --power package --server=false --library model/framework/dsl model/framework/realize -f -o .flow/tmp/fn1657-native-dsl.jar, mise exec -- env NATIVE_DSL_CLASSPATH=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1657-native-dsl.jar:/Users/stephan/Workspace/skunkworks/umpire/temporal/model/build/ir-scalapb.jar /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1657-external-recipe.sh, git diff --check, INCONCLUSIVE: initial baseline output handle lost (.flow/tmp/fn1657-baseline.log); no exit-code credit
- PRs: