# Compose entity machines into one Model

> HTML render lens (local): open `.flow/artifacts/fn-92-compose-entity-machines-into-one-system/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Umpire4 architecture reconciliation

This spec adds a `compose` command to `Umpire.Command` that builds one Model from several entity
machines with declared action synchronization over a reachable-state enumeration, adds `restrict:`
and `extend:` keys on `machine` that derive a machine from a source table, introduces the first entity
module, `worker`, under `Temporal.Feature`, and hosts the first cross-entity Properties as `verify`
Queries. Every construction is a pure function of checked tables, so the `Machine` proofs
and the Definition ID scheme carry over, and no behavior is defined outside a step function an
author wrote (AUT-07a, AUT-09). The word is `compose`, not `system`: `system` is the reserved party
for the implementation under test (glossary Party) and the elaborator enforces it.

Version one composes only for `verify` Queries. No functional, canary, or exploratory set runs over
a composed Model, so the Producer, the realizations, Case production, Testpilot, every Case fixture,
the production canary's pinned Case identity, the recorded control Run, the exploration coverage
golden, and the replay bridge are unchanged: every committed fixture and golden is byte-identical
after this spec. The caller module stays where it is, and so do the Start and Outage entities: moving
the `operation` entity would rewrite every `temporal.nexus.caller.*` ID and re-pin the canary, and a
shared `workflow` entity needs a key decision the grammar cannot express today (Start correlates by
`firstExecutionRunId`, Outage by `workflowTaskCompletedEventId` because its only evidence carries no
run id). Both are recorded under Boundaries as the follow-up. Rule text that needs a
GOV-02 draft is listed in Decision Context. The direction note `.plans/UMPIRE4_DIRECTION.md` and the
conversation of 2026-09-26 record the reasoning.

## Overview

Today `Temporal.Feature` holds seven machines shaped around the upstream tests they replace, three
of them copies of the Nexus reply logic, and no Property can relate a workflow to the worker that
serves it or an operation to the worker that answers it. This spec gives the model a way to compose
entities into one Model for cross-entity claims, adds the first entity module, and removes one of the
three copies. The full unification, which moves the operation entity, re-pins the canary, and needs
an entity key that varies by use case, is named as the follow-up.

## Goal & Context
<!-- scope: business -->

The vision's first line is one model for software behavior. The tree has seven: `Workflow/Start` and
`Workflow/Outage` each declare `entity workflow` with a different key and their own `startWorkflow`;
`Nexus/Pair` and `Nexus/Control` copy the caller's reply logic over new phase enums; `System/Info`
has an `entity server` nothing else mentions. Only `Nexus/Caller` has the intended shape.

Three structural causes: a machine tracks one entity and nothing composes machines of different
entities; the search enumerates paths, so a composed state space was unaffordable before fn-88; and
the Case pipeline made the upstream test the unit of authoring. The concrete symptom this spec fixes
first: `workerStop` is a stutter row on the operation machine, so the schedule-to-start timeout
Scenario orders the stop before the request by convention. With a worker machine composed in,
`handlerReply` is enabled only while the worker polls, and that ordering becomes a checked fact.

Who benefits: an author writing a cross-entity claim has a Model to write it against; the worker
becomes a first-class entity instead of a stutter row on two other machines; the Control model
becomes a visible one-row delta on the Pair machine instead of a third copy.

## Architecture & Data Models
<!-- scope: technical -->

```text
Temporal/Feature/
  Worker/Model.lean        entity worker (key taskQueue), workerStop, workerResume, serve,
                           machine polling                                                     new
  Workflow/Outage/Model.lean unchanged; gains compose workerOutage (outage machine ∥ worker) and a
                           cross-entity verify Query beside its functional set
  Nexus/Caller/Model.lean  unchanged; gains machine handlerWorker (from: polling restrict:), compose
                           nexusCaller (nexusProtocol ∥ handlerWorker), and a cross-entity verify
                           Query beside the existing sets
  Nexus/Control/Model.lean machine nexusControl from: pair extend: …, byte-identical Case        rewritten
  Nexus/Pair, Nexus/Success, Workflow/Start, System/Info                                        unchanged
```

Constructions, all in `Umpire.Command`, all pure functions of checked tables:

- **`compose` command.** Keys: `for:` the entities tracked; `state:` an author-written `structure`
  with one field per member (field names are what predicates read; the structure derives no
  `Finite`); `members:` one `field: machine` line per member;
  `sync:` one line per synchronized action, `name: field.action ∥ field.action …`, naming which
  member's action each participant is, first-named member's outcome reported; `starts:` and
  `ends:` over the composed state, since members like the worker have no natural end. The command
  generates `<name>.Action`, `<name>.Outcome`, and `<name>.Fact` as tagged unions with one
  constructor per member wrapping that member's type, so catalog keys never collide and predicates
  write `step.facts.contains (.operation .nexusOperationCompleted)`. Catalog keys obey
  `FiniteCatalog.validKey` (alphanumerics, `-`, `_`), so an unsynchronized action is keyed
  `<field>_<member key>` and a composed state `_`-joins its members' state keys in field order, the
  spelling `instances:` already uses for `1_accepted` and its product states; a member field name
  containing `_` is a located error. A synchronized action is keyed by its `sync:` name. Authors
  refer to members with dots, `operation.handlerReply (async)`, and Scenario `actions:`, Scenario
  `starts:`, and Property `when:` resolve those references and `sync:` names to the catalog keys,
  so the elaborator's last-component action lookup and `-`-split phase lookup are extended for
  compositions.
  Semantics: an unsynchronized action steps its owner and leaves the others unchanged; a
  synchronized step is enabled only when every participant has a row from its current state, and
  its results are the ordered product of the participants' results, sorted by the existing step
  order key, refused above 16 results with the counts; a participant may own a classless action
  synchronized with a classed one, in which case its row applies to every class; two members
  owning an action of the same name that no `sync:` line names is a located error; timers never
  synchronize and are keyed per member.
- **Reachable enumeration.** The composed table is built by a frontier breadth-first walk from the
  composed start states through the members' tables, evaluated at elaboration and emitted as a
  literal table. The state catalog is the reachable set sorted by lowered order key, as `instances:`
  orders its catalogs, so the fingerprint is independent of discovery and member order. The bound
  check is reachable states times actions against `enumerationBound`. An action never jointly
  enabled is dropped from the catalog with a located warning. This walk is the command's own; the
  literal is sorted by a total key before emission, since a stable sort would let ties carry the
  `members:` order into the fingerprint.
- **Composed state fields, starts, ends, timers.** The composed Model lowers every member's state
  fields as its own state fields keyed `<field>_<memberField>` (a one-field member as `<field>`);
  the member field itself is not a state field, because a whole-structure spelling would make every
  nested predicate a disjunction the clause language refuses.
- **Field-addressed requirements.** The `property` command's predicate enumeration fixes whole
  states, outcomes, and facts today, so a claim on one member's field, `step.state.worker.phase ==
  .polling` while the other member varies, fixes nothing and is refused. This spec adds one
  requirement kind: a state field is fixed when every accepted step carries that field's value and
  changing it to any other value of that field's catalog is rejected; it lowers to the same
  transition-contract clause on the field's model value, which both the reference evaluator and the
  Property monitors already match, because each offers a state's fields beside the state to a
  resulting-state pattern. A claim that names every field of a unique composed state, as the
  Outage claim can, is carried as a whole-state requirement without it. `starts:` and `ends:` on
  `compose` name member-qualified values (`workflow.completed`); a composed state is terminal when
  the named member field holds a listed value, and the machine rule that `ends:` values share one
  carrying field does not apply. Each member's `timers:`, `unobservable:` entries, and `evidence:`
  lines are lifted under `<field>.` keys; a `sync:` line naming a timer is a located error. A member
  may itself refine another machine; the composition carries neither `refines:` nor an abstract
  field.
- **Search backend.** fn-88 is a dependency and closes before this spec starts: `Selection.select`
  chooses the backend per Query with no author-facing override, `veil` when the Property lowers to
  monitors and the Scenario is pinned, `reference` otherwise with a recorded reason, and
  `Selection.cutover` is true. The two shipped claims are same-step `when:` Properties with a
  state-field requirement over a `scenario` declaration, which lowers, so each runs on `veil` with
  reason `default`; the kernel replay gate runs on every result as it does for every Query. The
  backend differential is registry-driven and covers a Query once the module declaring it is
  imported by the sweep module and its expected line is pinned, so every Query this spec declares,
  fixture or shipped, is listed there.
- **Proofs.** A generic theorem shows that a decidable check, `composedTableAgrees members literal
  = true`, implies soundness and completeness of the literal with respect to the composition
  function over reachable sources. The `compose` command emits that check for each composition by
  `decide +kernel`, as the refinement decision does today, and refuses the composition with a located
  error when it fails. The kernel time of that check and of the law proof is measured on both
  shipped compositions, beside the predicate-enumeration seconds of their Properties. If either
  exceeds the elaboration budget, the first fallback is a proof-shape change (per-row lemma
  batching, `Nat` constructor-index comparisons instead of `String` equality inside the decided
  `Bool`); `native_decide` comes only with the explicit policy approval `LEAN_GUIDELINES.md` §5
  requires, recorded per module, and on Lean 4.32 it mints a per-call axiom rather than
  `Lean.ofReduceBool`, so the R5 axiom pin fails on it by construction. `compose` elaborates to
  `Umpire.Command.DeclaredModel` as `machine` does; a Query's `check` over it yields
  `Umpire.Command.CheckedModel`, which still carries a `realizable` field as it does for every
  `verify` Query, and nothing reads it because no set names a composition.
- **`restrict:` and `extend:` on `machine`.** `machine x from: source restrict: [actions]` keeps only
  the listed actions' rows and drops the rest from the catalog; `evidence:` lines for facts no
  remaining row returns, `timers:` and `unobservable:` entries without rows are filtered. `extend:
  action: step` appends the author's results to the named action's rows, with a located error for a
  result at a source state where the action has no row and for a duplicate result; results are
  sorted by the step order key. Both keys may appear on one machine; `restrict:` applies first. A
  derived machine inherits neither `refines:` nor the abstract state field. `Nexus/Control` becomes
  `from: pair extend: handlerReply: forgedStep` with the machine name, catalog keys, and result order
  it has today, so its Case fixture and the recorded control Run under `tools/umpire/replay/testdata`
  stay byte-identical; a derived machine re-owns its catalogs under its own namespace.
- **Entity module.** `Worker/Model.lean` declares `entity worker` keyed by task queue, with
  `workerStop`, `workerResume`, a classless `serve` action (a polling worker serves; a stopped one
  has no row), and machine `polling` with `starts: [polling]` and `ends: [polling, stopped]`. The
  handler's task-queue worker and the workflow's task-queue worker are two instances of this entity,
  named by the member field of the composition that uses them. The caller and Outage modules keep
  their own `workerStop` actions in version one; the compositions synchronize with them by name.
- **Two compositions.** `workerOutage` composes `workflowOutage ∥ worker` with `sync: workerStop:
  workflow.workerStop ∥ worker.workerStop`, `workerResume` likewise, and `awaitCompletion:
  workflow.awaitCompletion ∥ worker.serve`, and verifies that a stopped worker completes no
  workflow, as a claim on the unique composed state a completion leaves. `nexusCaller` composes
  `operation ∥ handlerWorker`, where `handlerWorker` is `from: polling restrict: [workerStop,
  serve]` declared in the caller module, with `sync: workerStop: operation.workerStop ∥
  worker.workerStop` and `handlerReply: operation.handlerReply ∥ worker.serve`, and verifies with a
  bare `when: handlerReply`, which covers every reply class, and a field-addressed requirement that
  `handlerReply` never fires while the worker is stopped. The restriction is what keeps
  `workerResume` out of that composition; composing the full `polling` machine would admit a
  stop-resume-reply path the boundary excludes. The Outage functional set and the seven caller
  Queries keep their machines.
- **Lint.** `make lint-model` gains `feature-entity-uniqueness`: under `Temporal.Feature` production
  modules, an `entity` or `action` name declared twice is a violation unless allowlisted; the initial
  allowlist records the duplicates that exist once the worker module lands (`workflow` in Caller,
  Start, and Outage; `startWorkflow` in Start and Outage; `workerStop` in Caller, Outage, and Worker;
  `workerResume` in Outage and Worker) each with the follow-up spec that removes it, test and
  specimen modules are excluded, and the diagnostic names both modules. Any other duplicate fails. It reads `Registry.entities` in the environment-importing driver, since the
  import-graph rules cannot see declarations.

## API Contracts
<!-- scope: technical -->

- `compose` produces an `Umpire.Command.DeclaredModel`; `property`, `scenario`, `limits`, and
  `query` accept it as they accept a machine, and a Query's `check` yields
  `Umpire.Command.CheckedModel`. A `set` over a composed Model is rejected in version one with a
  located error naming this spec's boundary; `case` needs no separate check because it realizes a
  set.
- A `verify` Query over a composed Model is searched by the backend `Selection.select` picks for
  it. Its Limits are two-unit: on `veil`, `search` bounds product states and the depth is the
  smaller of `steps` and `actions`; on `reference`, `search` bounds candidate paths. The Limits of
  the two shipped Queries are declared so that both backends answer verified within limits, and the
  differential line pins the backend, the reason, and both explored counts.
- Every located error names the construct: unsynchronized shared action name, `sync:` participant
  input domain mismatch, a `sync:` line naming a timer, a member field name containing `_`,
  `restrict:` naming a timer or an absent action, `extend:` result at a disabled source or
  duplicate, product above 16 results, reachable enumeration above the bound with its counts, a
  failed `composedTableAgrees` check, a `set` over a composition, a `starts:` or `ends:` value that
  names no member field.
- A same-step Property may fix a state field; the refusals the predicate enumeration reports today
  (`fixesNothing`, `notCarried` with the distinguishing step, `neverHolds`) are unchanged in shape.
- Definition IDs of a composed Model hang off a `-compose-<name>` owner under the enclosing
  namespace; member facts and outcomes keep their own IDs inside the union's constructors.

## Quick commands

```bash
cd model && lake build Umpire.Command Temporal.Feature
cd model && lake build UmpireTests TemporalModelTests   # every #guard_msgs pin, both differential sweeps
make umpire-gen-case-runtime-conformance && make umpire-check-case-runtime-conformance
make umpire-check-goldens && make canary-check-case
go test -tags test_dep ./tools/umpire/replay/... ./tools/umpire/evaluation/... ./tools/umpire/cmd/umpire-assess/... ./tools/umpire/authoring/... ./tools/umpire/vocabulary/...
LEAN_NUM_THREADS=1 make lint-model
make umpire-check-regression   # final gate
```

## Edge Cases & Constraints
<!-- scope: technical -->

- **No fixture changes.** No entity, action, or machine moves in version one, and the Control rewrite
  preserves its machine name, catalog keys, and result order, so every Case fixture, the canary
  fixture, the canary policy's Case identity, the recorded control Run that three Go tests read, the
  exploration coverage golden, and the replay bridge goldens are byte-identical. A byte change
  anywhere is a finding, not a re-pin.
- **Ordering with fn-89 and fn-90.** Both touched the pair fixture and `Pair/Tests.lean`; neither
  touched `Pair/Model.lean`, the Control module, its fixture, or the recorded control Run. The
  Control rewrite over `pair` reads the pair machine as it stands. fn-89's per-instance Rules live
  in the Case Producer, which no composition reaches in version one, and `Registry` records no
  entity key or instance count on a machine, so `compose` has nothing of fn-89's to respect.
- **Ordering with fn-88.** This spec starts after fn-88 closes, so every pin is taken with
  `Selection.cutover` true and after fn-88.10's one-commit golden re-pin, and the docs edits of
  tasks .5 and .6 rebase on fn-88.7's edits to `model/AUTHORING.md`, `model/README.md`, and both
  `ARCHITECTURE.md` files. A pin taken against `cutover` false would move at the flip.
- **Backend per Query.** No author mechanism forces a backend. A composed Property written with
  `branches`, a guarded within-form, or a correlated rule, or a Limits unit of logical time,
  search, or plans, falls back to `reference` with a receipt reason; the two shipped claims avoid
  those forms so their differential lines read `veil default`. A future claim that needs one of
  them still elaborates and is still compared by the differential.
- **Predicate enumeration cost.** A same-step Property is enumerated over the composed table by
  mutating each state field across the catalog; on `nexusCaller` that is about 316 reachable states
  by 25 action classes. The seconds are recorded beside the kernel seconds and count against the
  same elaboration budget.
- **Bound pressure.** `nexusCaller` is at most 158 reachable operation states times 2 worker phases
  over about 25 action classes, about 8k evaluations, under `enumerationBound`; the measured count
  is a golden. A composition that exceeds the bound is refused with its counts; raising the bound is
  a GOV-02 question only if a shipped composition needs it.
- **Kernel cost.** The `composedTableAgrees` check and the law proof on a literal table of a few
  hundred rows are measured on `workerOutage` first, which is the proof point. The decided `Bool`
  compares constructor indices as `Nat`, which the kernel accelerates, and never sorts inside the
  proof, because `List.mergeSort` is well-founded recursion the kernel cannot unfold; the kernel
  honours `maxHeartbeats` and ignores `maxRecDepth`. The recorded 128 s of `native_decide` on three
  theorems (memory entry of 2026-09-09) is why a proof-shape change comes before that fallback.
- **Worker semantics differ per use case.** The handler's task-queue worker and the workflow's
  task-queue worker are two instances of one `worker` entity keyed by task queue; each composition
  names which one it composes through the member field.
- **No `workerResume` in `nexusCaller`.** After a stop the operation's timers remain enabled, so no
  state is stuck; resume is left to the Outage composition where it is the point. An
  unsynchronized member action stays independently executable, so the exclusion is a `restrict:`
  on the worker member, `handlerWorker`, whose catalog is pinned to have no resume.
- **Whole-state or field requirement.** The predicate enumeration fixes a whole state only when
  the accepted steps share one; on a composed table the other member varies, so a claim on one
  member's field is carried only by the field-addressed requirement task .7 adds. The Outage claim
  names the unique state `(completed, polling)` and works either way, which keeps the proof point
  independent of .7; the caller claim needs .7.
- **Drift markers.** The AUTHORING drift test maps one marker file today and rejects a duplicate
  block name across the whole walkthrough; it gains a marker-to-file map so quoted regions from
  `Worker/Model.lean` and the Outage module are checked, and the new regions take names no Caller
  region uses. Code inserted into `Caller/Model.lean` between two existing markers is absorbed into
  the preceding region and breaks its byte match, so the `nexusCaller` composition lands under its
  own marker with its walkthrough block in the same commit.
- **Test roots are explicit import lists.** `model/UmpireTests.lean` and
  `model/TemporalModelTests.lean` are the roots the `UmpireTests` and `TemporalModelTests`
  libraries build; a new test module is reached only once one of them imports it. The existing
  `Umpire.Command.Tests.*` modules are imported from `Nexus/Success/Tests.lean`; the new ones go
  into `model/UmpireTests.lean`.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `Temporal.Feature.Worker.Model` exists, declaring `entity worker` once with key
  `taskQueue`, actions `workerStop`, `workerResume`, `serve`, and machine `polling` with both phases
  terminal. Errors: covered by R2's lint and the machine command's existing checks.
- **R2:** `make lint-model` enforces `feature-entity-uniqueness`: a second `entity` or `action`
  declaration of one name across `Temporal.Feature` production modules fails with a diagnostic
  naming both modules unless allowlisted; the initial allowlist names exactly the duplicates listed
  under Architecture, each with the follow-up spec, and test and specimen modules are excluded. Errors: no error surface
  beyond the lint.
- **R3:** The `compose` command exists with `for:`, `state:`, `members:`, `sync:`, `starts:`, and
  `ends:`, generates the tagged-union Action, Outcome, and Fact types with `validKey`-legal
  member-prefixed keys (`<field>_<member key>`, states `_`-joined in field order), keys synchronized
  actions by their `sync:` name, resolves dotted Scenario `actions:` and Property `when:`
  references against those keys, and elaborates to an `Umpire.Command.DeclaredModel` a Query's `check` turns
  into an `Umpire.Command.CheckedModel`. Errors: each located error listed under API Contracts,
  pinned with `#guard_msgs`.
- **R4:** The composed table is enumerated by a frontier walk over reachable states only, emitted as
  a literal, with the state catalog sorted by lowered order key; reordering `members:` lines or
  `sync:` lines leaves the Behavior Fingerprint unchanged, pinned by a test, while reordering the
  fields of the author's `state:` structure changes it, as it does for a machine. Errors: reachable
  states times actions above `enumerationBound` is a located error with both counts.
- **R5:** A generic theorem shows `composedTableAgrees members literal = true` implies soundness and
  completeness over reachable sources, with axiom inventories limited to `propext`, `Quot.sound`,
  `Classical.choice`; `compose` emits the check by `decide +kernel` for every composition; and the
  kernel time of the check and of the law proof, with the predicate-enumeration time of each
  shipped Property, is recorded for `workerOutage` and `nexusCaller`. Errors: a failed check is a
  located error; any other axiom, including the per-call axiom `native_decide` mints, fails the
  pin.
- **R6:** `restrict:` and `extend:` derive machines as pure functions of the source table with the
  filtering and ordering rules the Architecture section states; a derived machine inherits neither
  `refines:` nor the abstract field and re-owns its catalogs under its own namespace;
  `Nexus/Control` is `from: pair extend:` with one extra-results function and no other step
  function, and `nexusCaller`'s worker member is `from: polling restrict: [workerStop, serve]`
  with a catalog pinned to hold no `workerResume`. Errors: the located errors listed under API
  Contracts.
- **R7:** The `nexusCaller` composition verifies that `handlerReply` never fires while the worker
  is stopped, and the `workerOutage` composition verifies that a stopped worker completes no
  workflow, both as `verify` Queries whose outcomes are pinned. Each is a same-step `when:`
  Property over a `scenario` declaration, the Outage claim fixing the unique whole state a
  completion leaves and the caller claim, with a bare `when: handlerReply` over every class,
  fixing the worker member's phase field, so `Selection.select` runs each on `veil` with reason
  `default`, and its Limits let both backends answer verified within limits. Errors: a Query the
  differential reports on `reference`, or as limit-reached on either backend, fails the pin; a
  claim the enumeration refuses as `fixesNothing` fails elaboration; no error surface beyond that
  and the outcomes.
- **R8:** The reachable state counts of `nexusCaller` and `workerOutage` are pinned as goldens and
  below `enumerationBound`. Errors: a count change fails the pin.
- **R9:** Every committed Case fixture, the canary fixture and policy Case identity, the recorded
  control Run under `tools/umpire/replay/testdata`, the exploration coverage golden, and the replay
  bridge goldens are byte-identical after this spec, and `make umpire-rerecord-pinned-runs` is not
  run. Errors: any change fails `make umpire-check-case-runtime-conformance`,
  `make canary-check-case`, `make umpire-check-goldens`, or one of the four readers of the
  recorded control Run (the replay, evaluation, and umpire-assess Go tests and the live assess
  test).
- **R10:** The Start, Outage, Pair, Control, and seven caller Queries report the same
  `PlanningOutcome` and witness as before. Errors: a differing outcome or witness fails the golden.
- **R11:** A `set` over a composed Model is rejected with a located error naming this boundary.
  Errors: that error is the whole surface.
- **R12:** `model/AUTHORING.md` gains a section on the entity module, `compose`, `restrict:`, and
  `extend:`, quoted from `Worker/Model.lean`, the Outage module, and `Nexus/Control/Model.lean`
  under `-- authoring:` markers, and the drift test maps markers to files. Errors: the drift test
  fails on a quoted region that differs from its source.
- **R13:** The GOV-02 drafts listed under Decision Context exist with the fn-92 marker, and the
  MOD-15 spec-names test passes. Errors: a missing draft or an untagged new dotted name fails review
  or the test.
- **R14:** Every Query declared over a composed Model or a derived machine, fixture or shipped, is
  listed in a backend differential's expected block with its backend and reason: the Umpire sweep
  for the fixture compositions of `Umpire.Command.Tests`, the Temporal sweep for `workerOutage`
  and `nexusCaller`, and at least one fixture `find` Query whose witness is a composed trace, so
  the kernel replay gate re-executes union-typed facts and member-prefixed catalog members.
  Errors: an unlisted Query fails the sweep's `#guard_msgs`; a witness the gate rejects reports
  `unreplayableWitness` and fails the pin.
- **R15:** A same-step Property may fix a state field: the predicate enumeration fixes a field when
  every accepted step carries its value and changing it to any other value of the field's catalog
  is rejected, the requirement lowers to a transition contract on the field's model value, and the
  reference evaluator and the Property monitors both evaluate it; the composed Model lowers each
  member's state fields keyed `<field>_<memberField>` so a nested-field claim reaches it. Pinned by
  a nested-field Property on a fixture composition, accepted with the other member varying and
  refused when the field varies, whose differential line reads `veil default` with both backends
  agreeing, and by an unchanged enumeration of every existing Property. Errors: a predicate the
  clause language still cannot carry is refused with the distinguishing step, as for a machine.

## Boundaries
<!-- scope: business -->

- No set, Case, realization, or Producer change over a composed Model; cross-entity claims are
  `verify` Queries only. Realizing a composed Model is the next spec.
- No move of the `operation` entity, the caller actions, or the caller machines out of
  `Temporal.Feature.Nexus.Caller`. That move re-pins every `temporal.nexus.caller.*` ID, the seven
  caller fixtures, the canary fixture, the canary policy's Case identity, and the exploration golden.
- No shared `workflow` entity module. Start correlates by `firstExecutionRunId` and Outage by
  `workflowTaskCompletedEventId`, and Outage's start row records nothing on purpose; one entity with
  one key cannot serve both until the grammar lets a use case choose its key. That grammar change,
  the operation move, and the resulting fixture and canary re-pin are the follow-up spec.
- No inline `restrict:` on a composition member.
- No change to `Nexus/Pair`; a `restrict:` that also projects state fields, which Pair over
  `instances: 2` needs to stay under the bound, is out of scope.
- No `instances:` inside `compose` in version one.
- No composition of refinements; a composed Model does not `refines:`, though a member may.
- No author-facing backend override per Query and no change to `Umpire.Search`, `Selection`, or
  the monitor lowering; a composed claim that needs an unsupported clause kind runs on `reference`
  as any machine's would.
- No change to `Nexus/Success`, `Workflow/Start`, `System/Info`, or `Temporal.System`.
- No cancellation (fn-79).

## Decision Context
<!-- scope: both -->

### Motivation
<!-- scope: business -->

Seven machines for seven use cases is the test suite rewritten in Lean. The user's observation on
2026-09-26 that the models read as separate scenarios rather than one model is correct. Version one
ships the composition primitive and the one entity module that costs no Case re-pin, verifies the
first two cross-entity claims, and names the canary-re-pinning move and the entity-key grammar as
the follow-up decision.

### Implementation Tradeoffs
<!-- scope: technical -->

- **`compose`, not `system`.** `system` is the reserved party and a Lean identifier in the elaborator;
  reusing it collides with both.
- **Generated unions, author-written state.** Predicates need field names, so the author writes the
  state structure; catalog keys collide across members (`accepted` twice), so the command generates
  the Action, Outcome, and Fact unions with member-prefixed keys. This trades the earlier "generate
  nothing" stance for the only type shape that typechecks `step.facts`.
- **Declared synchronization, not name matching.** Two modules' `workerStop` actions have
  different Definition IDs, so synchronization pairs actions explicitly; a shared name without a
  `sync:` line is an error rather than a silent independent step.
- **Classless participants.** The worker's `serve` synchronizes with the classed `handlerReply`
  without importing the caller's `Reply` domain into the worker module.
- **Reachable enumeration, literal table.** The full product exceeds the bound; a BFS at
  elaboration emitted as a literal keeps the kernel proof over a value rather than a computation.
- **verify only in version one.** Realizing a composed Model needs a multi-member `Realizable`, an
  acting-member resolution in the Producer, and fn-89's per-entity Rules across entities. That is a
  Producer spec of its own; shipping composition for claims first is the smallest useful step.
- **Control over Pair, Pair unchanged, bytes unchanged.** `extend:` removes one copy of the reply
  logic now while the recorded control Run keeps its bytes; Pair needs a field-projecting `restrict:`
  to fit `instances: 2` under the bound, which is deferred.
- **Per-composition kernel check.** A generic theorem about the composition function says nothing
  about a given literal, so each composition carries its own decided check, the same shape the
  refinement decision uses.
- **Backend by selection, not by declaration.** fn-88 gave `Selection.select` the choice per
  Query and no author key; this spec keeps it so and ships claims in the clause kinds that lower,
  because a per-composition backend key would be a second selection rule the differential cannot
  see.
- **Member fields flattened, member structures kept.** The author's `state:` structure keeps one
  field per member for predicates to read; the lowered state fields are the members' fields under
  `<field>_` keys, because the clause language fixes one field at a time and refuses a disjunction
  across fields.
- **Underscore keys, dotted references.** `FiniteCatalog.validKey` admits no `.`, and `instances:`
  already spells its product keys with `_`; a second separator would be a second grammar for the
  same thing. Authors keep the dotted spelling and the elaborator resolves it.
- **Field-addressed requirements as their own task.** The enumeration fixes whole states today, so
  a nested-field claim on a composed table fixes nothing; the Outage claim can name its unique
  state, the caller claim cannot, and the field requirement is the smallest addition that makes a
  member's field a claim. It lands after the compose command and before the caller composition,
  and its evaluators need no change.
- **A restricted worker for the caller.** An unsynchronized member action is independently
  executable, so "no resume" is a `restrict:` on the member and not a sentence.
- Maintainability (plan review): duplication - none identified; structure - task .7's machine-side
  test first imported `Temporal.Feature.Nexus.Caller.Model` into the Umpire test tree (round 2), a
  back-edge `lint-model` forbids; replaced by a local synthetic multi-field machine.
- **Refreshed 2026-09-27 against fn-88, fn-89, fn-90, and fn-94.** fn-88's seam, cutover, replay
  gate, and registry-driven differential replaced the "reference within Limits" contingency; fn-89's
  instanced Rules stay on the Case side, which no composition reaches; fn-90 left the Control
  module and its Run untouched; fn-94 removed nothing this spec cites.
- **Rules to draft under GOV-02**, new rules with the `*(drafted by fn-92; awaiting GOV-02
  approval.)*` marker after the rule title and amendments as `*Amendment (drafted by fn-92; awaiting
  GOV-02 approval.)*` continuation paragraphs, the two forms fn-88's drafts use: AUT-07a (add
  `compose`, `restrict:`, `extend:` to the surface); glossary Machine (derived machines),
  Refinement (a composition does not refine), Table (a composition's table holds reachable rows), a
  new Composition entry, Party (distinguish the `compose` command from the `system` party); AUT-09
  note that a reachable domain computed from declared member tables is author-provided; MOD-11
  amendment and a new MOD-18 for `feature-entity-uniqueness`, with the `UMPIRE4_SPEC_MODEL_ARCH.md`
  sentence that the import-graph phase is the single enforcement mechanism amended to name the
  declaration-level rule.
- **Rejected:** moving the caller module now (canary re-pin without a decision); a shared workflow
  lifecycle for Start and Outage (different start evidence and different keys); running the caller
  set over the composition (rewrites seven fixtures and the canary); name-based synchronization
  (silent cross-module coupling); full-product enumeration (millions of evaluations); a
  message-passing actor composition (second language).

## Early proof point

Task fn-92-compose-entity-machines-into-one-system.3 validates the core approach: it declares the
`workerOutage` composition over the unchanged Outage machine and the new worker machine, its
reachable table is under the bound, its `composedTableAgrees` check and law proof close in the
kernel within budget, the stopped-worker claim verifies, and its Temporal differential line reads
`veil default` with both backends agreeing. If the kernel checks do not close within budget after a
proof-shape change and `native_decide` is not approved at this seam, or the claim falls to
`reference`, stop and re-plan the proof strategy or the Property shape before the Control rewrite
and the `nexusCaller` composition.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1  | Worker entity module | .1 | — |
| R2  | `feature-entity-uniqueness` lint with allowlist | .1 | — |
| R3  | `compose` command, generated unions, key resolution, located errors | .2 | — |
| R4  | Reachable literal table, order-independent fingerprint | .2 | — |
| R5  | Agreement theorem, per-composition kernel check, kernel cost recorded | .3, .5 | — |
| R6  | `restrict:` and `extend:`; Control over Pair; the caller's restricted worker | .4, .5 | — |
| R7  | Two cross-entity verify Queries | .3, .5, .7 | — |
| R8  | Reachable count goldens | .3, .5 | — |
| R9  | Every fixture and golden byte-identical | .1, .4, .5 | — |
| R10 | Outcomes and witnesses unchanged | .4, .5 | — |
| R11 | `set` over a composition rejected | .2 | — |
| R12 | AUTHORING section and multi-file drift markers | .6 | — |
| R13 | GOV-02 drafts and MOD-15 | .6 | — |
| R14 | Every composed or derived Query in a backend differential's expected block; a composed witness through the replay gate | .2, .3, .4, .5, .7 | — |
| R15 | Field-addressed requirements; member state fields lowered as `<field>_<memberField>` | .2, .7 | — |

## Resolved via Research
<!-- provenance: plan (docs-scout, practice-scout) on 2026-09-27; refine --scope=research writes the same section -->

### docs-scout
- **Lean 4.32.0 `decide`** — `DecideConfig` has `kernel`, `native`, `zetaReduce`, `revert`;
  `+kernel` and `+native` together are an error; the kernel path goes through `mkAuxLemma` and
  honours `maxHeartbeats`, while `maxRecDepth` governs only the elaborator's diagnosis path.
  Source: `~/.elan/toolchains/leanprover--lean4---v4.32.0/src/lean/Init/Tactics.lean:1377-1422`,
  `Lean/Elab/Tactic/Decide.lean:78-130`, `Lean/Environment.lean:297`.
- **`native_decide` on 4.32** mints a per-call axiom `_native.<tactic>.ax…` asserting `e = true`;
  `Lean.ofReduceBool` is deprecated since 2026-02-01. Source: `Lean/Meta/Native.lean:37-85`,
  `Init/Core.lean:2437`.
- **`#guard_msgs` options**: `whitespace := exact|normalized|lax`, `ordering := exact|sorted`,
  `positions`, `substring`. Source:
  https://lean-lang.org/doc/reference/4.32.0/Interacting-with-Lean/#hash-guard_msgs
- **`List.mergeSort` is stable** (`mergeSort_zipIdx`), `Array.qsort` carries no stability lemma.
  Source: `Init/Data/List/Sort/Basic.lean:79`, `Init/Data/List/Sort/Lemmas.lean:334-346`,
  `Init/Data/Array/QSort/Basic.lean:64`.
- **Veil concrete checker** needs `StateFingerprint σ σₕ` (`BEq`, `LawfulBEq`, `Hashable`,
  `LawfulHashable`, `StateView`) and `ActionStatUpdate`; the repo satisfies it once in
  `model/Umpire/Search/Backend/Veil.lean:123-132` over the product state, so a composed State needs
  only the `DecidableEq`/`BEq` instances `DeclaredModel` already demands. Veil has no product or
  composition API. Source: Veil checkout `Veil/Core/Tools/ModelChecker/Concrete/Core.lean:19-27`,
  `Sequential.lean:82`; Veil `lean-toolchain` at the pinned revision is `v4.32.0`.

### practice-scout
- **Gotcha:** `Decidable` instances built by well-founded recursion or `Eq.rec` get stuck under
  `decide +kernel`; sort at elaboration time and decide only membership and equality over the
  emitted literal. Source: `Init/Tactics.lean:1419-1422`,
  https://github.com/leanprover/lean4/issues/2171
- **Gotcha:** `String` equality unfolds to `List Char` in the kernel; compare constructor indices
  as `Nat` inside the decided `Bool`. Source: `Init/Prelude.lean:3535`,
  https://github.com/leanprover/comparator/issues/93
- **Gotcha:** `mergeSort` is stable, so an injective sort key is what makes the fingerprint
  independent of `members:` order. Source: `Init/Data/List/Sort/Basic.lean:69`.
- **Pattern:** generate inductives by quoting `inductive … deriving …` into `elabCommand`
  (`elabInductiveViews` is private on 4.32) and read elaboration results back with the repo's
  `unsafe … Meta.evalExpr` plus `@[implemented_by]` pattern; wrap generated proofs in
  `elabGenerated`. Source: `model/Umpire/Command/Syntax.lean:148-150, 337-351, 1803`,
  `Lean/Elab/MutualInductive.lean:1563`.
- **Pattern:** pin a kernel failure with `substring := true`, as `Search/VisibilityTests.lean:54`
  does. Source: `Lean/Elab/Tactic/Decide.lean:121-127`.

## References

- `.plans/UMPIRE4_DIRECTION.md`; `model/Temporal/Feature/Nexus/DESIGN.md` §2.1, §2.3, §2.5, §4
- `model/Umpire/Command/Instances.lean`, `model/Umpire/Command/Syntax.lean`,
  `model/Umpire/Command/Finite.lean`, `model/Umpire/Command/Refinement.lean`,
  `model/Umpire/Command/Predicate.lean`
- `model/Umpire/Search/Selection.lean`, `model/Umpire/Search/Product/Monitor.lean`,
  `model/Umpire/Search/Product/Scenario.lean`, `model/Umpire/Search/Tests/Differential.lean`,
  `model/TemporalModelTests/SearchDifferential.lean`
- `model/Temporal/Feature/Nexus/Caller/Model.lean`, `Pair/Model.lean`, `Control/Model.lean`,
  `Workflow/Outage/Model.lean`, `model/Temporal/Case/Realization/Workflow.lean`
- `tools/umpire/authoring/drift_test.go`, `tools/umpire/replay/key_test.go`,
  `tools/canary/casebinding/testdata/`

