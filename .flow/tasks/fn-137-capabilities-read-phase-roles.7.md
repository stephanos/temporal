---
satisfies: [R2, R3]
---
# fn-137-capabilities-read-phase-roles.7 Pausable reads Suspended and Held; settle pausedWhileHeld; capability docs

## Description
Pausable reads "paused" as `Suspended` and Pollable reads "running" as `Held` of the `Phased` phase. They stay separate capabilities (fn-134). `pausedIsNotDispatched` reads both owned roles and is still brought only where both are declared. This task records `pausedWhileHeld` as `Held` and finishes the capability docs; the conductor owns the batched MILESTONES update.

**Size:** M
**Files:** model/temporal/capabilities/{Pausable,Pollable}.scala (fn-134's one-file-per-capability layout), Catalog.test.scala, model/irgen/Capabilities.scala, model/irgen/testdata/lifts/Capabilities.scala (+ expected), Pausable sites: activity product Product.scala, Record.scala, WithTaskQueue.scala; model/README.md (capabilities), MILESTONES.md
**Touches:** [model/temporal/capabilities/**, model/irgen/Capabilities.scala, model/irgen/testdata/**, model/temporal/features/activity/**, model/README.md, MILESTONES.md]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Remove `paused` (Pausable.scala after fn-134) and `running` (Pollable.scala). Read them through `Phased` with `Suspended` and `Held` witnesses, following task 6's pattern. The Property body (Pausable.scala's companion; before fn-134 Pause.scala, `pausedIsNotDispatched`) takes role-derived inputs.
- Lifter refusals: Pausable on a phase with no `Suspended` case, and the pause-with-polling Property on a phase with no `Held` case, each naming the missing role.
- Owned roles count as fields (task 6's rule): Pausable owns `Suspended` and Pollable owns `Held`, so `pausedIsNotDispatched` is brought exactly where both are declared.
- `pausedWhileHeld`: give it `Held`, not `Suspended`. It is an admitted attempt that remains held and owned while pause is requested; `paused` remains the dispatch-blocking `Suspended` phase. Record the decision and expected batch delta in the spec, move it out of Parked unknowns, and defer regeneration so no unapproved generated Query change is absorbed.
- Docs: capability descriptions in the README and the Closable.scala, Pausable.scala and Pollable.scala header comments. The conductor updates MILESTONES.md after the DSL batch.

### Acceptance
- [ ] No Pausable or Pollable declaration binds a paused, running or projection field, including on derived objects.
- [ ] `pausedIsNotDispatched` is brought exactly where Pausable and Pollable are both declared.
- [ ] `pausedWhileHeld` carries `Held`, and the spec records why it is not `Suspended`.
- [ ] Regeneration leaves Query names, answers, receipts and Definition IDs unchanged, except an ActivityRecord change the owner approves, recorded in the spec.
- [ ] Both refusal fixtures pass, naming the missing role.
- [ ] The README and capability comments are updated; focused model, lifter, packaging and scoped lint gates pass. MILESTONES and regeneration remain conductor-owned.

## Acceptance
- [ ] TBD

## Done summary
Pausable now binds only pause/unpause and owns Suspended; Pollable binds only dispatch and owns Held. Their Property reads the declaring object typed Phasing and is brought only for the pair. All eight direct/derived owners migrated; three predicate aliases retired. pausedWhileHeld is Held because the admitted attempt remains owned while pause is pending. Runtime, focused lifter, package, format and scoped lint evidence passed; regeneration remains deferred by DSL batch policy.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a8a669c1208c6f875c9249096394c80d5b478da8
- Tests: scala-cli test model/project.scala model/umpire model/temporal (62 passed), scala-cli test model/irgen --test-only *PhaseCapabilities* after fresh model package/classpath (2 passed), make model/build/model-scala.jar (passed, no unchecked warnings), make lint-model-models lint-model-irgen lint-model-syntax (exit 0), git diff --check (exit 0)
- PRs: