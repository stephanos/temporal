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
  Nexus/Caller/Model.lean  unchanged; gains compose nexusCaller (nexusProtocol ∥ worker) and a
                           cross-entity verify Query beside the existing sets
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
  constructor per member wrapping that member's type, keyed `<field>.<member key>`, so catalog keys
  never collide and predicates write `step.facts.contains (.operation .nexusOperationCompleted)`. An
  unsynchronized action is keyed `<field>.<member key>` and a synchronized action by its `sync:`
  name; Scenario `actions:` and Property `when:` resolve against those keys, so the elaborator's
  last-component action lookup and `-`-split phase lookup are extended for compositions.
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
  enabled is dropped from the catalog with a located warning. This walk is the command's own. fn-88 is a
  dependency because it closes before this spec starts in either of its modes; if it closed with a
  defer decision, the composed Models' `verify` Queries run on the `reference` backend within Limits
  declared to complete on the two shipped compositions.
- **Proofs.** A generic theorem shows that a decidable check, `composedTableAgrees members literal
  = true`, implies soundness and completeness of the literal with respect to the composition
  function over reachable sources. The `compose` command emits that check for each composition by
  `decide +kernel`, as the refinement decision does today, and refuses the composition with a located
  error when it fails. The kernel time of that check and of the law proof is measured on both
  shipped compositions. If either exceeds the elaboration budget, the composed table may be admitted
  through `native_decide` only with the explicit policy approval `LEAN_GUIDELINES.md` §5 requires,
  recorded per module. `compose` elaborates to `Umpire.DraftModel` through `Umpire.checkModel`.
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
  workflow.awaitCompletion ∥ worker.serve`, and verifies that a stopped worker completes no workflow. `nexusCaller` composes `operation ∥ worker` with
  `sync: workerStop: operation.workerStop ∥ worker.workerStop` and `handlerReply:
  operation.handlerReply ∥ worker.serve`, and verifies that `handlerReply` never fires while the
  worker is stopped. The Outage functional set and the seven caller Queries keep their machines.
- **Lint.** `make lint-model` gains `feature-entity-uniqueness`: under `Temporal.Feature` production
  modules, an `entity` or `action` name declared twice is a violation unless allowlisted; the initial
  allowlist records the duplicates that exist once the worker module lands (`workflow` in Caller,
  Start, and Outage; `startWorkflow` in Start and Outage; `workerStop` in Caller, Outage, and Worker;
  `workerResume` in Outage and Worker) each with the follow-up spec that removes it, test and
  specimen modules are excluded, and the diagnostic names both modules. Any other duplicate fails. It reads `Registry.entities` in the environment-importing driver, since the
  import-graph rules cannot see declarations.

## API Contracts
<!-- scope: technical -->

- `compose` produces a `CheckedModel`; `property`, `scenario`, `limits`, and `query` accept it as
  they accept a machine. A `set` over a composed Model is rejected in version one with a located
  error naming this spec's boundary; `case` needs no separate check because it realizes a set.
- A `verify` Query over a composed Model is searched by whichever backend fn-88 left in place; its
  Limits are declared so that it completes on `reference` for the two compositions this spec ships,
  and its outcome is pinned either way.
- Every located error names the construct: unsynchronized shared action name, `sync:` participant
  input domain mismatch, `restrict:` naming a timer or an absent action, `extend:` result at a
  disabled source or duplicate, product above 16 results, reachable enumeration above the bound with
  its counts, a failed `composedTableAgrees` check, a `set` over a composition.
- Definition IDs of a composed Model hang off a `-compose-<name>` owner under the enclosing
  namespace; member facts and outcomes keep their own IDs inside the union's constructors.

## Quick commands

```bash
cd model && lake build Umpire.Command Temporal.Feature
cd model && lake build Umpire.Command.Tests Temporal.Feature.Nexus.Tests Temporal.Feature.NexusTests
make umpire-gen-case-runtime-conformance && make umpire-check-case-runtime-conformance
make umpire-check-goldens && make canary-check-case
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
- **Ordering with fn-89 and fn-90.** Both touch the pair fixture; this spec does not. The Control
  rewrite over `pair` depends on the pair machine fn-90 leaves, which is why fn-90 is a dependency.
- **fn-88 closed deferred.** The compositions still elaborate; their `verify` Queries run on
  `reference` and must complete within declared Limits, and the measured reachable counts are the
  argument for revisiting Limits.
- **Bound pressure.** `nexusCaller` is at most 158 reachable operation states times 2 worker phases
  over about 25 action classes, about 8k evaluations, under `enumerationBound`; the measured count
  is a golden. A composition that exceeds the bound is refused with its counts; raising the bound is
  a GOV-02 question only if a shipped composition needs it.
- **Kernel cost.** The `composedTableAgrees` check and the law proof on a literal table of a few
  hundred rows are measured on `workerOutage` first, which is the proof point; `native_decide` is a
  fallback only with the explicit policy approval the Lean guidelines require.
- **Worker semantics differ per use case.** The handler's task-queue worker and the workflow's
  task-queue worker are two instances of one `worker` entity keyed by task queue; each composition
  names which one it composes through the member field.
- **No `workerResume` in `nexusCaller`.** After a stop the operation's timers remain enabled, so no
  state is stuck; resume is left to the Outage composition where it is the point.
- **Drift markers.** The AUTHORING drift test maps one marker file today; it gains a marker-to-file
  map so quoted regions from `Worker/Model.lean` and the Outage module are checked.

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
  `ends:`, generates the tagged-union Action, Outcome, and Fact types with member-prefixed keys,
  keys synchronized actions by their `sync:` name, resolves Scenario `actions:` and Property `when:`
  against those keys, and elaborates to a `CheckedModel`. Errors: each located error listed under
  API Contracts, pinned with `#guard_msgs`.
- **R4:** The composed table is enumerated by a frontier walk over reachable states only, emitted as
  a literal, with the state catalog sorted by lowered order key; reordering `members:` lines or
  `sync:` lines leaves the Behavior Fingerprint unchanged, pinned by a test. Errors: reachable
  states times actions above `enumerationBound` is a located error with both counts.
- **R5:** A generic theorem shows `composedTableAgrees members literal = true` implies soundness and
  completeness over reachable sources, with axiom inventories limited to `propext`, `Quot.sound`,
  `Classical.choice`; `compose` emits the check by `decide +kernel` for every composition; and the
  kernel time of the check and of the law proof is recorded for `workerOutage` and `nexusCaller`.
  Errors: a failed check is a located error; any other axiom fails the pin.
- **R6:** `restrict:` and `extend:` derive machines as pure functions of the source table with the
  filtering and ordering rules the Architecture section states; a derived machine inherits neither
  `refines:` nor the abstract field and re-owns its catalogs under its own namespace;
  `Nexus/Control` is `from: pair extend:` with one extra-results function and no other step
  function. Errors: the located errors listed under API Contracts.
- **R7:** The `nexusCaller` composition verifies that `handlerReply` never fires while the worker
  is stopped, and the `workerOutage` composition verifies that a stopped worker completes no
  workflow, both as `verify` Queries whose outcomes are pinned. Errors: no error surface beyond the
  Queries' outcomes.
- **R8:** The reachable state counts of `nexusCaller` and `workerOutage` are pinned as goldens and
  below `enumerationBound`. Errors: a count change fails the pin.
- **R9:** Every committed Case fixture, the canary fixture and policy Case identity, the recorded
  control Run under `tools/umpire/replay/testdata`, the exploration coverage golden, and the replay
  bridge goldens are byte-identical after this spec. Errors: any change fails
  `make umpire-check-case-runtime-conformance`, `make canary-check-case`, or the Go replay tests.
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
- No composition of refinements; a composed Model does not `refines:`.
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
- **Rules to draft under GOV-02** with the `*(drafted by fn-92; awaiting GOV-02 approval.)*` marker:
  AUT-07a (add `compose`, `restrict:`, `extend:` to the surface); glossary Machine (derived
  machines), Refinement (a composition does not refine), Table (a composition's table holds
  reachable rows), a new Composition entry, Party (distinguish the `compose` command from the
  `system` party); AUT-09 note that a reachable domain computed from declared member tables is
  author-provided; MOD-11 amendment and a new MOD rule for `feature-entity-uniqueness`.
- **Rejected:** moving the caller module now (canary re-pin without a decision); a shared workflow
  lifecycle for Start and Outage (different start evidence and different keys); running the caller
  set over the composition (rewrites seven fixtures and the canary); name-based synchronization
  (silent cross-module coupling); full-product enumeration (millions of evaluations); a
  message-passing actor composition (second language).

## Early proof point

Task fn-92-compose-entity-machines-into-one-system.3 validates the core approach: it declares the
`workerOutage` composition over the unchanged Outage machine and the new worker machine, its
reachable table is under the bound, its `composedTableAgrees` check and law proof close in the
kernel within budget, and the stopped-worker claim verifies. If the kernel checks do not close within
budget and `native_decide` is not approved at this seam, stop and re-plan the proof strategy before
the Control rewrite and the `nexusCaller` composition.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1  | Worker entity module | .1 | — |
| R2  | `feature-entity-uniqueness` lint with allowlist | .1 | — |
| R3  | `compose` command, generated unions, key resolution, located errors | .2 | — |
| R4  | Reachable literal table, order-independent fingerprint | .2 | — |
| R5  | Agreement theorem, per-composition kernel check, kernel cost recorded | .3, .5 | — |
| R6  | `restrict:` and `extend:`; Control over Pair | .4 | — |
| R7  | Two cross-entity verify Queries | .3, .5 | — |
| R8  | Reachable count goldens | .3, .5 | — |
| R9  | Every fixture and golden byte-identical | .1, .4, .5 | — |
| R10 | Outcomes and witnesses unchanged | .4, .5 | — |
| R11 | `set` over a composition rejected | .2 | — |
| R12 | AUTHORING section and multi-file drift markers | .6 | — |
| R13 | GOV-02 drafts and MOD-15 | .6 | — |

## References

- `.plans/UMPIRE4_DIRECTION.md`; `model/Temporal/Feature/Nexus/DESIGN.md` §2.1, §2.3, §2.5, §4
- `model/Umpire/Command/Instances.lean`, `model/Umpire/Command/Syntax.lean`,
  `model/Umpire/Command/Finite.lean`, `model/Umpire/Command/Refinement.lean`
- `model/Temporal/Feature/Nexus/Caller/Model.lean`, `Pair/Model.lean`, `Control/Model.lean`,
  `Workflow/Outage/Model.lean`, `model/Temporal/Case/Realization/Workflow.lean:147-156,238`
- `tools/umpire/authoring/drift_test.go`, `tools/canary/casebinding/testdata/`
