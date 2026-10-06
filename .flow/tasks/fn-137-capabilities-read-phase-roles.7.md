---
satisfies: [R2, R3]
---
# fn-137-capabilities-read-phase-roles.7 Pausable reads Suspended and Held; settle pausedWhileHeld; capability docs

## Description
Pausable reads "paused" as `Suspended` and Pollable reads "running" as `Held` of the `Phased` phase. They stay separate capabilities (fn-134). `pausedIsNotDispatched` reads both owned roles and is still brought only where both are declared. This task also settles the role of `pausedWhileHeld` with the owner, and finishes the capability docs and the MILESTONES entry.

**Size:** M
**Files:** model/temporal/capabilities/{Capabilities,Pause,Catalog}.scala (post fn-134), Catalog.test.scala, model/irgen/Capabilities.scala, model/irgen/testdata/lifts/Capabilities.scala (+ expected), Pausable sites: activity product Product.scala, Record.scala, WithTaskQueue.scala; model/README.md (capabilities), MILESTONES.md
**Touches:** [model/temporal/capabilities/**, model/irgen/Capabilities.scala, model/irgen/testdata/**, model/temporal/features/activity/**, model/README.md, MILESTONES.md]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Remove `paused` (Capabilities.scala:29 today) and `running` (Pollable, :50 today). Read them through `Phased` with `Suspended` and `Held` witnesses, following task 6's pattern. The law body (Pause.scala, `pausedIsNotDispatched`) takes role-derived inputs.
- Lifter refusals: Pausable on a phase with no `Suspended` case, and the pause-with-polling Property on a phase with no `Held` case, each naming the missing role.
- Owned roles count as fields (task 6's rule): Pausable owns `Suspended` and Pollable owns `Held`, so `pausedIsNotDispatched` is brought exactly where both are declared.
- `pausedWhileHeld`: before regenerating, ask the owner unconditionally whether it is `Suspended`, `Held`, or a model-specific role extending one (fn-136 R4 forbids both). Assign it on the record's admission-phase enum, then regenerate and compare. ActivityRecord today reads paused = `paused` only and running = `started` only (Record.scala:267-281). Record the decision in the spec, move it out of Parked unknowns, and stop on any Query change the owner hasn't approved.
- Docs: capability descriptions in the README and the Capabilities.scala, Close.scala and Pause.scala header comments. Mark fn-137 done in MILESTONES.md.

### Acceptance
- [ ] No Pausable or Pollable declaration binds a paused, running or projection field, including on derived objects.
- [ ] `pausedIsNotDispatched` is brought exactly where Pausable and Pollable are both declared.
- [ ] `pausedWhileHeld` carries the owner's chosen role on its enum, and the spec records the decision.
- [ ] Regeneration leaves Query names, answers, receipts and Definition IDs unchanged, except an ActivityRecord change the owner approves, recorded in the spec.
- [ ] Both refusal fixtures pass, naming the missing role.
- [ ] The README and MILESTONES are updated, and `make umpire-check-model` and `make lint-model` pass.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
