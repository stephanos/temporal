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
The new form now sits beside the old one in the framework, the lifter and the gate. A machine's `object capabilities extends Capabilities` section, or one extending a shared base, `object capabilities extends Shared(this)`, brings each Property defined in a declared kind's companion. The lifter expands each into `<machine>.<property>` with an inert `origin`, bounds it from a `queries` statement `capabilities.bound(limits, <Capability>.<property> -> limits)`, and the gate writes the section's waivers into `<file>.lint.json`. The old `Implements`/`Law`/`Catalog` path is untouched: its fixtures (capabilities.json, capabilities.laws.json, every other expected file) lift byte-identical.

stage: impl-review - skipped(config: REVIEW_MODE=none, DSL batch reviews at the batch end)

Tier: IMPLEMENTER claude-opus-5-5 at high (actual_model: claude-opus-5-5)

### What was built
- Framework (`model/umpire/Capabilities.scala`): the old value class `Capabilities[S]` is now `LawDeclaration[S]`; its matches in Machine.scala, IrFile.scala, irgen `capable`, Order.scala `kindOf` and Lifting.scala are updated. The new `abstract class Capabilities[S, O, F](using Declaring[S, O, F])` provides:
  - `protected type Capability = CapabilityOf[S, O, F]`
  - protected `except` and `overriding`
  - `claim(property)`
  - `bound(limits, overrides*)`

  `IrFile.construct` admits the section as a root.
- Lifter (`model/irgen/Capabilities.scala`): `capabilitiesSectionOf` reads the section's body and the chain of source base classes it extends. Base constructor params are bound to the section's folded arguments, accessors included, so shared vals such as `c.own(_.left, hold)` and `through(...)` resolve. A capability Property is a companion def returning `Property`. It is brought when every parameter after the model is a field of exactly one declared capability. `expand` is shared with the old path, which still builds its `LawClaim` the same way. The generated Property takes its position from the capability's val. Its origin is the def's qualified name and position, and an `overriding` def keeps the origin of the Property it replaces.
- Refusals, each naming the Property, the capabilities and positions:
  - R1: reads no field of its own kind; a parameter no kind of the lifted sources has; a parameter two declared capabilities both bind; a kind declared twice (both positions).
  - R2: a waiver of an unbrought Property; an empty reason.
  - R3: a brought Property no `queries` statement bounds; an override of an unbrought or a waived Property; a second bound statement.
  - Both forms on one machine, refused from either section.
- Waivers: the lifter writes `<file>.waivers.json` (the section machines plus `(machine, <machine>.<property>, reason)` for each waiver) beside the IR. `Gate.settle` ignores it. `Gate.acceptWaivers` merges it into `<file>.lint.json` under `waived-law`:
  - an existing acceptance keeps its place and takes the current reason;
  - a stale one owned by a section machine is dropped;
  - a new one is appended;
  - every other entry stays.

  `--update` writes the merged file. Check mode fails, naming the missing and stale subjects. The encoder matches Go's `Accepted.Encode` byte for byte, and a test round-trips every `model/ir/*.lint.json`. The JSON reader uses only the standard library, as the gate requires.
- Order and structure: `capabilities` follows `implements` in `formSections` and in the order lint. The cycle detector now counts the body of every source base class a section object extends as part of that object's own initialization. Red-first: without that edge the new initOrder fixture's cycle went unreported.

### Tests (acceptance)
- `lifts/CapabilitySections.scala` → `expected/capabilitySections.json` and `capabilitySections.waivers.json`:
  - Task has its own section, its `queries` bounds it with three plus a `sealedIsRefused -> two` override, and a `claim` Query reads a generated Property.
  - Chore declares Holdable without Takeable, so it gets no `heldIsNotTaken`; it waives one Property with `except` and replaces one with `overriding`.
  - TaskPair and MirrorPair are compositions extending `PairCapabilities(this)` and inherit its waiver; MirrorPair adds a waiver of its own.
  - The new test "a generated Property's origin is the capability Property it was expanded from" pins every origin and checks no other Property carries one.
- `lifts/CapabilitySectionRejects.scala` (10 roots, lines in `rejects.txt`): one per R1/R2/R3 error plus both forms.
- `crossed/Sections.scala`: a capability of another state, outcome or fact type does not compile (3 positions).
- `initOrder`: Dimmer.capabilities → DimmerRealization → Dimmer.capabilities, through the base class Dimmed.
- Gate tests cover merge in place, stale drop, other entries kept, check-mode failure text, no file created for zero waivers, and the Go-format round trip.
- The lift fixtures need my own kinds (Sealable, Holdable, Takeable): `CapabilityVocabularySuite` forbids naming Temporal kinds in `model/irgen` and `model/umpire`, and the kit's companions only gain Property defs in .3.

### Declared IR delta
None for `model/ir`, `model/cases` or Model lift expectations: no Model uses the new form yet, and the old path's output is unchanged (verified on its fixtures). The new fixture files and the 10 new rejects.txt lines are this task's deliberate fixture changes. The Scala IR jar was repackaged because fn-134.1's ir.proto was newer than it.

### Decisions (recorded in the spec's Decision Context)
- The shared base is feasible, so the shared-def fallback was not used. A base takes `(using Declaring[S, O, F])` to pass the machine's types on.
- Capability vals are typed `: Capability`. This is what makes a mistyped capability fail to compile and gives a phantom state type the machine's. The lifter also refuses an untyped val that does not conform.
- `bound` is a statement rather than a val, so `IrFile.queriesOf` reads nothing new. The lifter searches every `queries` section of the lifted sources for it, so the five Record designs and the WithTaskQueue designs in .3 can be bounded wherever their Queries live.
- A shared def can take the section as a `Capabilities[S, O, F]` parameter and call `.claim` on it, which .3 needs for Record and WithTaskQueue (`valued` in Claims.scala now folds such a parameter). At run time `claim`'s Property has an empty name; the IR generator names it.

### Notes for the conductor and later tasks
- Go conflict until .5: `umpire-lint --update` runs `lint.Forward`, which deletes every `waived-law` acceptance the law sidecar does not list, and its check fails on the same difference. Once .3 migrates the Models and the sidecars are gone, umpire-lint would strip the acceptances this gate writes until .5 removes the Forward. That ordering is the batch plan (.5 runs before the batch gates). Running `make umpire-check-model` between .3 and .5 would fail.
- No `queries` fixture of the Models used: the lifter reads the bounds wherever the statement sits, so .3 needs no particular `queries` layout.
- Not covered: a machine declared with both the old function form `capabilities(m, limits)(...)` and a new section is refused only when both declare the same kind. Section-versus-section is refused outright.
- Model owner edits seen during the run, all left untouched: `MILESTONES.md`, `.flow/specs/fn-123-*`, untracked `.flow/tasks/fn-123-*` and `fn-141-*`, `.plans/PROTO_ANNOTATIONS.md`. The owner also committed 5be6e3060a between my base and my commits, so the evidence lists my two commits instead of a range.
- No MILESTONES.md edit is needed from this task.
- Follow-up for .6 (docs): model/README.md and SEMANTICS.md should describe the section, `Capability`, `bound` and the waiver file.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4409cccffc, ff50aef53f
- Tests: make model/build/ir-scalapb.jar model/build/api-scalapb.jar model/build/model-scala.jar (packaging only; ir jar was stale against fn-134.1's ir.proto), baseline: green (mise exec -- scala-cli test model/irgen: 89 passed, 1 skipped, pre-edit), UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (writes this task's own fixture expectations only), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (check mode: 92 passed, 1 skipped), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check (69 passed), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal (23 passed), make lint-model-irgen lint-model-check lint-model-models lint-model-irgen-lifts lint-model-syntax (each rc=0, run one by one), scala-cli fmt --check on every changed Scala file, GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model and the byte-identical model/ir check run at the batch's single regeneration, GATE_SKIPPED:go-suite:batch - DSL batch rule: the full Go suite runs at the batch end
- PRs: