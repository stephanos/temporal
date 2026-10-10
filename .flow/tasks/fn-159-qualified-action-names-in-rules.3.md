---
satisfies: [R9]
---
# fn-159-qualified-action-names-in-rules.3 Qualify every phase reference; refuse phase-member imports

## Description
Name every phase through its enum: `when(Phase.scheduled)`, `phase = Phase.started`, `State(Phase.scheduled)`, `s.phase == Phase.paused`. Do this in rule cases, effects, init, states and properties alike. The IR generator refuses any import of a phase type's members, so bare phase names can't come back.

**Size:** M
**Files:** `model/irgen/Declarations.scala` (and the lifter for imports, if it sits elsewhere), `model/framework/Syntax.scala` doc comments, `model/README.md`, the 9 model files that import phase members, `model/irgen/testdata/lifts/Admission.scala` and new refusal fixtures, the pin tests that import phases
**Touches:** [model/irgen/*.scala, model/irgen/testdata/lifts/*.scala, model/irgen/testdata/lifts/expected/*.json, model/framework/Syntax.scala, model/README.md, model/temporal/features/activity/standalone/product/Product.scala, model/temporal/features/activity/standalone/system/System.scala, model/temporal/features/activity/standalone/system/Dispatch.scala, model/temporal/features/activity/standalone/system/DispatchRaces.scala, model/temporal/features/nexus/product/Product.scala, model/temporal/features/nexus/standalone/system/System.scala, model/temporal/features/nexus/workflow/system/System.scala, model/temporal/foundations/taskqueue/product/Product.scala, model/temporal/features/activity/standalone/StandaloneActivityPins.test.scala, model/temporal/features/nexus/product/NexusOutcomePins.test.scala]

### Approach
- **What counts as a phase type:** the `P` of a machine's `Phased[S, P]` projection, plus the phase type of a composition or derived machine that inherits it. The enums have different names: `Phase`, `AdmissionPhase`, the task queue's `Outstanding`, and testdata's `ProductPhase`. Key the rule on the projection type, not on the name `Phase`.
- **irgen, imports:** refuse an import of a phase type's members in a lifted source, whether wildcard (`import Phase.*`, `import product.Phase.*`, `import Outstanding.*`) or selective (`import Phase.{started, paused}`). It doesn't matter where the import sits: file level, machine object, or a nested section or `rules` block. The message names the import and its position, and says to write `Phase.<case>`.
- **irgen, rule cases:** also require each argument of the phase form `when(p1, …)` to be a selection: a case selected on its phase type (`Phase.scheduled`), or a named set selected on `states` (`states.terminal`). Leave the role form `when[Closed]` alone. Don't touch `m.property.when(...)` in capabilities; that's the Property builder's `when`, not a rule case. This check backs up the import ban, which already makes bare names fail to compile.
- **Framework:** no runtime check. The phase form of `when` takes varargs that aren't inline, so it can't read their source text (see the `codeOf` note on the multi-target `on`). Update the doc comments and examples that show `when(placed, open)` with bare names.
- **Migrate:** in each machine object, drop the phase imports and qualify every phase reference: rule cases, effects (`phase = Phase.started`), `init`, `states`, properties, and `choose` / `enter` arguments (`s.copy(phase = Phase.scheduled)`). The task queue product's `import Outstanding.*` gives `when(Outstanding.empty)`. The test files that import phases (activity standalone pins, nexus outcome pins) migrate for consistency, but irgen doesn't lift tests, so they aren't enforced.
- **IR stays the same:** the change is syntactic. Check that `make umpire-check-model` passes with no checked-in IR file regenerated differently. The lifted rule-case headings are the exception: if irgen records the written text, e.g. `when(scheduled)` becoming `when(Phase.scheduled)` in the IR or the lint table, regenerate and check that this is the only diff. Record which it was in the done summary.
- **Coordination:** fn-159.1 touches the same irgen and README sections and the same `rules` blocks. Land fn-159.1 first, or do both in one batch.

### Investigation targets
**Required:**
- `model/irgen/Declarations.scala`: the machine-object and rules lift, and where `when` cases are read (`cases(...)`)
- `model/framework/Syntax.scala`: the `when` overloads in `Rules`/`Firing`, and `Phased`
- `model/README.md`: the rules and effects spec sections

### Batching
- Batch 1d (Scala DSL, immediate; see MILESTONES.md): land the Scala changes now. Checked-in IR, Cases and Go consumers may go stale. Verify with the Scala build, framework and model tests, and fn-160.5's step-table comparison under this task's rename map. Declare the IR change this task will cause; the IR is regenerated once, in Batch 5b, whose shared close supplies the gate evidence for the `make umpire-check-*` criteria.

### Quick commands
```sh
make umpire-check-model
make umpire-check-lint
```

## Acceptance
- [ ] The IR generator refuses any import of a machine's phase-type members (wildcard or selective, at any level of a lifted source), naming the import and position and spelling out `Phase.<case>`.
- [ ] The IR generator refuses a bare phase argument to the phase form of `when(...)`; `when(Phase.x)`, `when(states.set)` and `when[Role]` are accepted.
- [ ] No lifted model imports phase members; every phase reference in rule cases, effects, init, states and properties is qualified; the pin tests are migrated too.
- [ ] Refusal fixtures exist for wildcard, selective and nested phase imports and a bare `when` argument; accepted fixtures cover the qualified forms.
- [ ] The README and framework doc comments show only qualified phases; `make umpire-check-model` and `make umpire-check-lint` pass, with checked-in IR unchanged except rule-case heading text if the IR records it (stated in the done summary).


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
