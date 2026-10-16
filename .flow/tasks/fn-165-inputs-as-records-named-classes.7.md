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
Admission hold released: fn-164.10 is verified done with committed receipt 2806e4c53b on umpire. This task is not complete. Execute native .7 -> .8 -> .9 before releasing fn-173.10; IR/Go/backend work and model/temporal remain excluded. No whole-spec fn-164 closure or IR gate is a prerequisite.
## Evidence
- Commits:
- Tests:
- PRs:
