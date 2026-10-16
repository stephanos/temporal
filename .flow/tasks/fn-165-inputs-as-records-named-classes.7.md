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
- Prove named, empty and record-value calls, plus no-input and scalar-input forms before broad migration. A generic apply cannot accept arbitrary named/default arguments; use the compile-time checked frontend proposed in the reanchor or prove an equivalent solution without reflection or runtime unchecked labels.
- Preserve declaration labels, input type/order, explicit non-first defaults and selected Finite witness. Enumerate only when the caller requests classes, never to discover a default. Missing/default-outside-domain/duplicate-or-unknown-label/wrong-type forms must fail clearly.
- fn-159.10 is already source-landed and superseded in representation only; preserve its explicit-default behavior. Record native baseline evidence now; exact IR baseline is separately held.

### Investigation targets
**Required:** model/framework/dsl/Action.scala; model/framework/dsl/InputSource.scala; model/framework/dsl/Domain.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Inputs.test.scala; model/framework/dsl/InputDefaults.test.scala.

### Verification
Use isolated native compiler/test selection from fn164.2 with positive executed counts. Pin old/new native class order and defaults against independently recorded examples. No capture/exporter/IR/Go/Temporal checks.

## Acceptance
- [ ] Inventory covers native inputs, class calls, defaults, effect arities and shared/token identity consumers with no guessed label provenance.
- [ ] Empty/named/value record calls and scalar/no-input forms compile and run; malformed labels, missing/default violations and wrong types reject.
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
