---
satisfies: [R1, R3]
---
# fn-137-capabilities-read-phase-roles.6 Closable reads the Closed role through Phased (needs fn-134, fn-136)

## Description
Closable loses its `status` and `terminal` fields and reads "closed" as the `Closed` role of the machine's `Phased` phase, through a type witness. Builds on fn-134's `capabilities` section, so start only after fn-134 and fn-136 have landed. Regeneration keeps every Query's name, answer, receipt and Definition ID.

**Size:** M
**Files:** model/temporal/capabilities/Closable.scala (fn-134's one-file-per-capability layout), model/temporal/capabilities/Catalog.test.scala, model/irgen/Capabilities.scala, model/irgen/testdata/lifts/Capabilities.scala (+ expected), the Closable declaration sites: activity product Product.scala, Record.scala, WithTaskQueue.scala (both compositions), nexus standalone System.scala
**Touches:** [model/temporal/capabilities/**, model/irgen/Capabilities.scala, model/irgen/testdata/**, model/temporal/features/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Before fn-134 `Closable[S, P, O](status, terminal, rejected)` was at Capabilities.scala:12; after fn-134 it is a companion-defined capability in `Closable.scala`. Remove `status` and `terminal`, and read the projection from the declaring object's `Phased` given. For compositions, today's `through(_.activity, ...)` projection comes from the composition's own `Phased`.
- The role witness is a `using` parameter resolved at the declaration against the concrete enum, so the build has no unchecked warning. On derived objects (`trustingActivityRecord`, `trustingRecordOverQueue`, `trustingRecordOverMatching`, `recordOverLossyMatching`) the phase type comes from the source's typed given re-exported by task 1.
- Extend fn-134's binding rule: reading a role the capability owns counts as reading its field. Closable owns `Closed`, so `terminalStatesAreFinal` is not refused for "reads no field of its own capability" and is still brought exactly where Closable is declared. Put the role ownership in the capability companion next to its fields.
- Lifter (irgen/Capabilities.scala:402-414, :453-456): drop the def-field checks for the removed fields, build the Closable Property from the `Phased` projection (or a derivation source's) and the `Closed` case set, and refuse Closable on a machine that is neither `Phased` nor derived from one, or whose phase has no `Closed` case, naming the machine. Reuse task 5's role-case check.
- Once the `terminal` field is gone, delete the `states.terminal` predicates that fn-136 task .5 kept only for it (activity product, ActivityRecord, Nexus standalone), and rewrite any remaining callers to `in[Closed]` / `is[Closed]`. This is an IR change within the bound of fn-136 R9: the Function is removed and its case set inlined. Classify the diff the same way.
- If a site's old `terminal` set differs from its `Closed` cases, the regeneration shows it. Stop and resolve it with the owner, never absorb it.

#- Derived compositions (spec Decision Context, 2026-10-06): make composition derivations return `Composition[S] & DerivedFrom[this.type]`, as machine derivations do after fn-137.1, so `trustingRecordOverQueue`, `trustingRecordOverMatching` and `recordOverLossyMatching` read the source's typed `Phasing[S, P]`.

## Acceptance
- [ ] Every Closable declaration, including the four on derived objects, binds only the rejection outcome.
- [ ] `terminalStatesAreFinal` is brought on exactly the machines that declare Closable (fn-134's two-machine test still passes).
- [ ] Regeneration leaves every Query's name, answer, receipt and Definition ID unchanged.
- [ ] Refusal fixtures: Closable on a non-`Phased` machine, and on a phase with no `Closed` case, each naming the machine.
- [ ] The build shows no unchecked warning, and `make umpire-check-model` and `make lint-model` pass.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
