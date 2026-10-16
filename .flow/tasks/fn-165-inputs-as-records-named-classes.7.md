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
- Preserve value-call ergonomics through exact-label one-field records, including `Finish(result: Result)` and `Prepare(rm: RM)`. Genuine scalar `.input[A]` remains unresolved unless exact declaration-label provenance is proven; never infer labels from action/type names or domain order.
- Preserve declaration labels, input type/order, explicit non-first defaults and selected Finite witness. Enumerate only when the caller requests classes, never to discover a default. Missing/default-outside-domain/duplicate-or-unknown-label/wrong-type forms must fail clearly.
- fn-159.10 is already source-landed and superseded in representation only; preserve its explicit-default behavior. Record native baseline evidence now; exact IR baseline is separately held.

### Investigation targets
**Required:** model/framework/dsl/Action.scala; model/framework/dsl/InputSource.scala; model/framework/dsl/Domain.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Inputs.test.scala; model/framework/dsl/InputDefaults.test.scala.

### Verification
Use isolated native compiler/test selection from fn164.2 with positive executed counts. Pin old/new native class order and defaults against independently recorded examples. No capture/exporter/IR/Go/Temporal checks.

## Acceptance
- [ ] Inventory covers native inputs, class calls, defaults, effect arities and shared/token identity consumers with no guessed label provenance.
- [ ] Native input type is the declared record, not a tuple surrogate. Empty, reordered named, record-value and exact-label one-field value calls compile and run, including a generic helper and separately compiled external client; unknown/duplicate labels, omitted required fields, another record and wrong field types fail compilation with positive controls.
- [ ] Class order and explicit defaults preserve independent native pins, including non-first defaults and custom Finite witnesses without eager enumeration.
- [ ] Focused native tests pass; deferred exact Param/Class/default IR proof and original .1-.6 remain open.

## Done summary
Blocked:
Owner-ordered native DSL queue. Release only after fn-164.10's native caller/API/colon checkpoint is committed on umpire. This is a task-level admission hold because flowctl forbids cross-spec task edges; do not replace it with whole-spec fn-164 closure or an IR gate. Native sequence is fn164.2 -> .10 -> fn165.7 -> .8 -> .9 -> fn173.10 -> .11 -> fn123.9 -> .10 -> fn153.7. IR/Go/backend work follows all native DSL work. Ignore model/temporal entirely.

Blocked:
Owner-ordered native DSL queue. Release only after fn-164.10's native caller/API/colon checkpoint is committed on umpire. This is a task-level admission hold because flowctl forbids cross-spec task edges; do not replace it with whole-spec fn-164 closure or an IR gate. Native sequence is fn164.2 -> .10 -> fn165.7 -> .8 -> .9 -> fn173.10 -> .11 -> .12 -> fn123.9 -> .10 -> fn153.7. IR/Go/backend work follows all native DSL work. Ignore model/temporal entirely.
## Evidence
- Commits:
- Tests:
- PRs:
