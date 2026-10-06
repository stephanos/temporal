---
satisfies: [R1, R2, R3, R4]
---
# fn-134-capabilities-own-their-properties.2 Capabilities section, capability Properties, bounds and waivers in the lifter (additive)

## Description
The early proof. Add the new form beside the old one: the `object capabilities extends Capabilities` section, capability Properties defined in the kind's companion, bounds in `queries`, `origin` emission, and waiver reasons written by the model gate. Prove it on irgen fixtures. The old `Implements`, `Law` and `Catalog` path stays working and untouched, so the gate stays green. A machine declaring both forms is refused.

**Size:** M
**Files:** `model/umpire/{Capabilities,Machine,Compose,IrFile}.scala`, `model/irgen/{Capabilities,Claims,Lifting,Declarations,Structure,Order,Context}.scala`, `model/check/Gate.scala` (waiver writer), new irgen fixtures under `model/irgen/testdata/lifts/` and `crossed/`, `model/irgen/test/Fixtures.test.scala`
**Touches:** [model/umpire/**, model/irgen/**, model/check/Gate.scala, model/check/test/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Name clash first: rename the value class `Capabilities[S]` (umpire/Capabilities.scala:39) to an internal name, and update its matches (IrFile.scala:50, Machine.scala:96-111, irgen `capable` ~Capabilities.scala:188-196, Order.scala:288, which matches the fully qualified name). Then add `abstract class Capabilities` as the section base. It takes the machine's types from the `Declaring` given (Machine.scala:179/336, Compose.scala:55), so mistyped capabilities do not compile and phantom state types resolve. `except`/`overriding` are protected statements of its body.
- Section recognition: follow `sectionOf`/`parentArguments` (Declarations.scala:206-215) and `isSection`/`requiredClass` (Context.scala:290-305). Add `capabilities` to `formSections` in the slot `implements` holds (Structure.scala:590-601) and to the order lint (Order.scala:619, message :313). Keep `implements` until task 4.
- Shared sets: support `object capabilities extends SharedBase(this)`. The lifter must read the base's vals and its `except` statements (today `implementsOf` reads only the object's own body, Capabilities.scala:211-236). If that is not feasible, implement the shared-def fallback (each section's vals call a shared def) and record it in the spec's Decision Context. Either way a design cannot drop a waiver its shared set states.
- Expansion: a capability Property is a def in the kind's companion taking `(model, <field params>…)`. Reuse `expand` (Capabilities.scala:649-738), whose binding by parameter name already spans capabilities. Bring the Property when every capability its parameters read is declared. Emit `origin` (fully qualified def name, def position). An `overriding` Property keeps the origin of the Property it replaces. Errors per R1/R2: reads no field of its own capability, unbound parameter, parameter bound twice, capability declared twice (both positions), waiver of an unbrought Property, empty reason, both sections on one machine.
- Bounds: in `queries`, one statement bounds every generated Query of the machine's capabilities, with a per-Property override. Errors per R3: brought Property unbounded, override of an unbrought or waived Property. Query names stay `<machine>.<property>`.
- Waiver reasons: on `--update` the model gate writes each new-form waiver into `<file>.lint.json` as the `waived-law`-keyed acceptance lint's forward writes today, preserving existing key order and hand-written entries. Check mode fails on a missing or stale one. Old-form machines keep the old forward, and keys never overlap because no machine has both forms.
- Init order: a section extending a parameterised base is a new init edge. Update the cycle detector and add one fixture (testdata/initOrder).

### Investigation targets
**Required:**
- `model/irgen/Capabilities.scala:183-236, 297-353, 649-738`
- `model/irgen/Declarations.scala:206-215`, `model/irgen/Context.scala:290-305`
- `model/irgen/Structure.scala:587-601`, `model/irgen/Order.scala:288, 313, 619`
- `model/umpire/Capabilities.scala`, `model/umpire/Machine.scala:96-111, 179, 336`
- `model/check/Gate.scala:360-366, 492-505`
**Optional:**
- `model/irgen/test/Fixtures.test.scala:137-193, 849-897`
- `tools/umpire/lint/laws.go:56-67, 161` (today's forward, for the acceptance shape)

### Key context
Memory `consolidated-extractor-dropped-a-2026-09-27`: when two paths are folded together, carry over every validation the old path made (the duplicate-capability and waiver checks in `declare`, Capabilities.scala:316-353).

## Acceptance
- [ ] Fixtures lift: a `capabilities` section, and one extending a shared base with inherited waivers; generated Properties carry `origin`; a Property reading two capabilities is brought only when both are declared.
- [ ] Bounds from `queries`, including a per-Property override; Query names `<machine>.<property>`.
- [ ] One refusal fixture per error in R1, R2 and R3, plus one for both sections on one machine; messages name the Property, capabilities and positions.
- [ ] The model gate writes and checks new-form waiver acceptances; a stale or missing one fails check mode.
- [ ] Init-order fixture for the base-class edge.
- [ ] Old `Implements`/`Law` path unchanged: `make umpire-check-model` passes with `model/ir` byte-identical.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
