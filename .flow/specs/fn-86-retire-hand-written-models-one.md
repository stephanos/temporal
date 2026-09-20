## Goal & Context
<!-- scope: business -->

After fn-85, a developer can model a feature entirely through the commands: entities, actions,
machines, observations, Properties, Queries and sets, with a realization producing the Cases. The
tree still carries a second, hand-written way to do the same thing. About 7,000 lines build Umpire
records or Testpilot Cases directly in Lean: two typed Nexus examples, four generations of Nexus
models (Race, Lifecycle, Operations, Experimental), two Cases with no Model at all (worker outage,
get-system-info), and Umpire's own Switch example. AUT-08 still names that path as an "expert
alternative".

Two authoring paths mean two things to learn, two things to keep working, and examples that teach
the path we no longer want developers to take. This spec makes the commands the only authoring path
for feature Models and keeps every piece of coverage the hand-written files provide: it is migrated
onto the commands, recorded in the open or deferred spec that owns it, or listed as deliberately
dropped. Nothing is deleted before its coverage has a destination.

Two decisions frame it (recorded with the user on 2026-09-10):

- **The typed examples migrate; they are not deleted first.** They are the only proof that
  Properties reach real request and response fields ("the submitted `workflow_type.name` equals the
  one recorded in history"), so this spec adds that capability to the commands and moves them onto
  it.
- **`Temporal.System.Nexus` stays, as the single exception.** It is the one Implementation Link
  between `Temporal.Feature` and `Temporal.System` that SEM-08 and MOD-04 describe. Removing it would
  amend those rules; keeping it is revisited once the commands can express a system model.

## Architecture & Data Models
<!-- scope: technical -->

### What is hand-written today

| Area | Covers | Destination |
| --- | --- | --- |
| Typed examples (`TypedUnary`, `TypedNexus`) and their tests, fixtures, Go artifact tests and live tests | generated RPC bindings; Properties comparing a request field to a recorded event field; two Nexus operations correlated by identity; crossed-pairing mutations | migrated onto the commands (R2, R3) |
| Worker outage and get-system-info Cases | a fault-bearing workflow Case with no Model; an RPC-only Case | re-authored as command Models with workflow and RPC realizations (R4) |
| `Umpire.Examples.Switch` and its goldens, used across Umpire's own tests | Umpire's Temporal-free worked example | re-authored with the commands inside Umpire (R5) |
| Nexus `Race` models | the cancellation race | deleted; the race behavior recorded in fn-79, which re-authors cancellation on fn-85 when resumed (R6) |
| Nexus `Lifecycle`, `Operations`, `Observation` and their goldens | earlier generations of the Nexus model | deleted when the inventory shows fn-85's Nexus Model covers them; otherwise the gap is migrated or recorded (R1, R6) |
| Nexus `Experimental` (`VariationSpace`, `Exploration`, `AutoClose`) | exploration inputs and the auto-close study | deleted; what fn-33 needs recorded in fn-33 as exploratory-set inputs (R6) |
| `Temporal.System.Nexus` (`Core`, `Evidence`, `ImplementationLink`) | the SEM-08 Implementation Link | kept as the only exception |
| Testpilot conformance and synthetic Cases | runtime conformance with expected Verdicts; they test Testpilot, not authoring | kept |

The first task confirms this table file by file before anything moves (R1).

### Field relations

A **field relation** is a Property clause that compares typed fields across the actions and
observations of a path: an action's input field, an action's result field, or an observation's
field, each addressed through the schema of the action or observation that carries it. It is the
command-level form of what `PropertyFieldPath` and the typed examples express in hand-written Lean
today, and it lowers to the Contract the same way, so the runtime reads exactly the fields the
Property compares.

```lean
property submittedTypeIsRecorded
  machine: workflowStart
  when: startWorkflow
  require: startWorkflow.input.workflow_type.name = workflowExecutionStarted.workflow_type.name
```

Fields are checked against the generated schema while the file compiles. An observation used in a
field relation needs a schema, which the realization's catalog supplies for history events.

### One authoring path, enforced

`lint-model` gains an import rule: a non-test module under `Temporal.Feature` or `Umpire.Examples`
imports Umpire's authoring owners (`Umpire.Model`, `Umpire.Property`, `Umpire.Scenario`,
`Umpire.Query`, `Umpire.Operation`, `Umpire.Case`) only through `Umpire.Command`. Realizations in
`Temporal.Case` and the `Temporal.System.Nexus` Implementation Link are outside the rule. Umpire's
own test trees keep building records directly, because they test those records.

## API Contracts
<!-- scope: technical -->

Field relation clause, in the fn-85 command style (exact grammar is a task decision):

```text
property <name>
  machine:  <machine>
  when:     <action pattern>
  require:  <field path> (= | ≠) <field path> | <field path> present
```

```text
<field path> ::= <action>.input.<schema field>(.<schema field>)*
               | <action>.result.<schema field>(.<schema field>)*
               | <observation>.<schema field>(.<schema field>)*
```

A field path through a repeated field selects the element that the path's observation correlates to
the row's entity instance through its key. A path through an optional submessage is present only
when every segment is set.

Lint diagnostic for R7, in the import-graph checker's existing one-line form (the Makefile asserts
this shape byte for byte for the planted violation). The rule is a direct-import rule, because every
module reaches the owners transitively through `Umpire.Command`:

```text
[model-import-graph/authoring-path-isolation] forbidden direct import: <module> -> <Umpire owner>
```

## Edge Cases & Constraints
<!-- scope: technical -->

- **Nothing without a destination.** A file is deleted only when the R1 inventory names, for each
  Property, golden, fixture, live test and tool reading it, where that coverage went or that it is
  dropped and why.
- **Deferred owners.** fn-79 is deferred until the user resumes it and fn-33 is unplanned. Deleting
  the Race and Experimental files does not wait for them; the behavior they cover is written into
  those specs' text so their re-planning starts from it.
- **Goldens and fixtures** change only through their generators (ART-11, ART-12); every regenerated
  file's diff is listed in the task receipt, and a byte-identical regeneration is stated as such.
- **Typed mutations.** The typed examples' crossed-pairing tests prove that a crossed correlation is
  a Property violation, not an admission rejection. The migrated form keeps that distinction with a
  realization or observation mutation, not by deleting the test.
- **Implementation Link exception.** `Temporal.System.Nexus` keeps importing Feature modules under
  MOD-10's existing exception; the new lint rule does not widen it. Today the link imports
  `Temporal.Feature.Nexus.Lifecycle` and `Temporal.Feature.Nexus.Race.Terminal`, both on the
  deletion list, so before either is deleted the link is re-anchored on fn-85's Nexus product
  machine with its forward simulation still proved; the link's Lean tests are the pin.
- **Tools.** The golden, inspect and inventory tools lose their hand-written inputs; each is pointed
  at a command-authored Model or its dependency is removed, and its gate still passes.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A committed inventory lists every non-test module under `Temporal.Feature`,
  `Temporal.Testpilot` and `Umpire.Examples` that builds Umpire records or Testpilot Cases without
  the commands, with each Property, golden, fixture, live test and tool that reads it and a
  destination: migrate, delete with coverage recorded in a named spec, keep with the reason, or drop
  with a reason. Errors:
  a module that builds records but is missing from the inventory fails `lint-model`'s existing
  inventory reconciliation over the import graph (a new inventory issue kind, not a new Go tool),
  naming the module.
- **R2:** A Property clause compares action input, action result and observation fields through
  their schemas, checked while the Model file compiles and lowered to Contract field reads identical
  in shape to today's hand-written field Properties. Errors: a path segment not in the schema, a type
  mismatch between the compared fields, an observation without a schema, and a repeated-field path
  with no correlation key reject in place, pinned by `#guard_msgs`.
- **R3:** The typed unary and typed Nexus examples are Model files authored with the commands and
  field relations plus a realization; their Cases regenerate with the diff listed; their Go artifact
  tests and live tests pass; the crossed-pairing mutation still yields a Property violation, not an
  admission rejection; the hand-written modules, their Lean tests and their `register_case` lines
  are deleted; the Testpilot instructions only the typed Nexus example still emitted
  (`StartNexusOperation`, `RespondNexus`, `NexusResponseKind`, `CompleteNexusOperation`'s untyped
  result) are removed from the protocol and their names added to the retired-vocabulary gate, as
  fn-85 R10 defers to this spec. Errors: no error surface beyond R2 and fn-85.
- **R4:** The worker-outage and get-system-info Cases are produced from command Models (a workflow
  Model with `workerStop` and `workerResume` fault actions, and an RPC Model) through workflow and
  RPC realizations; the hand-written Case modules are deleted; both live tests pass; fixture diffs are
  listed. Errors: no error surface beyond fn-85.
- **R5:** `Umpire.Examples.Switch` is re-authored with the commands; every Umpire test and tool that
  imports it builds; its goldens regenerate byte-identical, or the diff is listed with the reason.
  Errors: no error surface beyond fn-85.
- **R6:** Every module the R1 inventory marks for deletion is removed with the goldens and fixtures
  only it produced; the Race behavior is recorded in fn-79's spec and the exploration inputs in
  fn-33's; the golden, inspect and inventory tools build and their gates pass. Errors: a remaining
  import of a deleted module fails the build.
- **R7:** `lint-model` rejects a non-test module under `Temporal.Feature` or `Umpire.Examples` that
  imports `Umpire.Model`, `Umpire.Property`, `Umpire.Scenario`, `Umpire.Query`, `Umpire.Operation`
  or `Umpire.Case` other than through `Umpire.Command`; realizations in `Temporal.Case` and
  `Temporal.System.Nexus` are outside the rule. Errors: a violation prints the module and the import;
  a fixture module with a planted violation proves the gate fails closed.
- **R8:** `UMPIRE4_SPEC.md` drafts under GOV-02 an AUT-08 amendment that removes the expert
  alternative of constructing `Umpire.Machine` directly, an AUT-07a amendment that names the
  commands as the only authoring path for feature Models, and a MOD rule for R7 enforced under
  MOD-11; the architecture documents, `model/README.md`, `model/AUTHORING.md` and
  `UMPIRE4_ORDER.md` describe one authoring path; `make umpire-check-regression` passes. Errors: no
  error surface beyond R7.

## Early proof point

R1's inventory and R2's field relations proven on the typed unary example (R3's smaller half)
before any deletion. If the typed unary Property cannot be expressed as a field relation that lowers
to the same Contract field reads, stop: the migrate-first decision for the typed examples needs
revisiting before anything is removed.

Tasks fn-86-retire-hand-written-models-one.1 (the inventory and the typed-unary Contract baseline)
and .2 (field relations and the typed unary migration compared against that baseline) are that
proof. If .2 cannot reproduce the baseline's field reads, stop before .3.

## Quick commands

```bash
# Import rules, inventory reconciliation and the planted violation
LEAN_NUM_THREADS=1 make lint-model
# Goldens, regression views and fixtures after each deletion or migration
make umpire-check-goldens umpire-check-regression-views umpire-check-case-runtime-conformance
# Full gate
CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression
```

## Planning decisions

Decided while breaking the spec into tasks (2026-09-12), from the repository and gap scans.

- **The Implementation Link is re-anchored before anything is deleted.** It imports `Lifecycle` and
  `Race.Terminal` and its evidence tests import `Operations`; task .4 moves it onto fn-85's product
  machine and re-pins its evidence on the Caller Model's Queries, then task .5 deletes.
- **Field relations lower onto the existing field-Property structure.** `PropertyFieldPath` already
  has index, select, cardinality, establish and capture steps; R2 adds a clause form, not a clause kind.
- **A field relation is a `relates:` line of `property`** (task .2, 2026-09-20): `property <name>
  machine: <m> when: <action> relates: <operand> = <operand>`, `≠`, or `<operand> present`, beside
  `holds:` rather than under the retired `require:` key. An operand is `<action>.input.<path>`,
  `<action>.result.<path>` (the action the claim is about, through its `schema:` request or the
  method's response) or `<kind>.<path>` (a recorded event kind the machine's `evidence:` lines
  name, read through the history response), and the platform resolves the dotted path to
  `PropertyFieldPath` steps, presence reads and a scalar type through an installed resolver
  (`Umpire.Command.installFieldResolver`, answered by `Temporal.Case.FieldPath`). The Producer
  lowers the relations whose action a path performs to one monitor rule `<property>.relation`
  through `Umpire.Case.Projection.lower`; a `present` relation admits but lowers to no rule.
- **An observation of an earlier step is captured** (task .3, 2026-09-20): a `relates:` operand
  `<kind>.<path>` whose fact the `when:` action's rows do not record is an earlier step's event,
  read at the state the step starts from (the first state a row of the action starts from, in
  table order) and lowered through `Umpire.Case.Projection.lower`'s cross-event capture: the rule
  retains the instance's own event, selected by the field the realization's evidence source names
  (`EvidenceSource.selector`, resolved against the schema at definition time) against the literal
  the confirming action's binding assigns under that spelling, and matches the later event's read
  against the retained one. A field of the history event itself (`event_id`) is read off the
  event whichever arm the kind names. A result operand is read under the step's outcome.
- **Several instances are placed, not flattened** (task .3): the Program path carries the instance
  performing each action, an `ActionBinding` builds its node from a `Placement` (the Case, the
  instance, the count), every id of an instance's node, slot and entrypoint carries `-<n>` on a
  Case over several instances and nothing on one over one, an entrypoint may be emitted per
  instance (`EntrypointPlan.perInstance`, the Nexus handler), a `whenOnPath` item emits once per
  instance that performs the class, slots may be declared per instance, and a relation lowers to
  one rule per instance (`relation-<n>`). The typed Nexus example is
  `Temporal/Feature/Nexus/Pair/Model.lean`: two instances of the caller Model's operation on a
  machine that keeps the asynchronous success path, produced through the caller realization, whose
  instances address `<operation>-<n>` on their own handlers.
- **The Search does not extend a prefix the Scenario admits no extension of** (task .3): an action
  out of an exact sequence's order, one past an occurrence maximum, or one the Scenario forbids is
  not enumerated (`CheckedScenario.admitsPrefix`), so the admitted traces and their order are the
  same and the candidates counted against `limits.search` are the ones that could be admitted. The
  Operations compatibility artifacts, which record explored counts, were regenerated.
- **The Implementation Link's destination is a Target derived from the product machine's rows**
  (task .4, 2026-09-20): `nexusProduct`'s own table is not admissible as a Target (two classes
  without a row, one start), so the link derives `productTarget` from its rows -- the same rows,
  `scheduled` and `started` as starts, `ends:` as the terminal condition, its own Target and kernel
  identity -- and names every element from the checked Target's vocabulary. The forward simulation
  is decided over that Target (the System's authoritative cases, the Target's soundness laws, a
  `native_decide` membership each), the cancellation projection confirms `canceled` and `completed`
  as the product's completion classes and treats the cancellation request as irrelevant (no
  product row; fn-79), and the evidence tests pin the Caller Model's Queries. The Target is
  irreducible so a goal over its machine is never unfolded into the admission.
- **The retired models are gone and their behavior is spec text** (task .5, 2026-09-20): Race,
  Lifecycle, Operations, Observation and Experimental are deleted with their goldens, tests, docs,
  the `NexusDiscovery` tool and the `TemporalExperimentalTests` root; the race Model's rows,
  Properties and Scenarios are recorded in fn-79's spec and the variation Space and Exploration
  inputs in fn-33's; the facade `Temporal.Feature.Nexus` is the Caller and Pair Models; the
  inspector's registry is the Caller Model's Queries beside Switch, `list` prints it and `explain`
  prints one Query's checked lineage; no Temporal-side compatibility family remains, `switch` is
  pinned by `UmpireTests` until .7.
- **The two Model-less Cases are command Models and the outage-order rule is Producer-derived**
  (task .6, 2026-09-20): the worker outage is `Temporal/Feature/Workflow/Outage/Model.lean` --
  `workerStop` and `workerResume` as actions of the `worker` party, bound by the workflow
  realization to fault instructions on the Case's task-queue role -- and the system-info call is
  `Temporal/Feature/System/Info/Model.lean` through a unary realization whose evidence is the
  instruction-completed Run Event. `Umpire.Case.Producer.outageOrderRules` derives the
  bounded-liveness rule from the assembled Program (one per role whose faults stop then resume,
  `rule_events` deadline from `Realization.outageDeadline`), so any fault-bearing path carries it
  without a line of its own. A workflow's history events carry no one key across the started and
  completed events, so the outage Model's one evidence kind is the completed event keyed by the
  workflow task that completed it, and the three steps before it are Known Gaps. `register_case`
  is gone: every checked-in Case is a `case` block's.
- **The Switch example's ids moved with it (.7, 2026-09-20)**: the commands derive
  `umpire.switch.<kind>.twoState.<member>` under a new `Umpire.Examples.Conventions` root, each
  state and fact is its own definition, and the `power` field is one more; no golden is
  byte-identical and each regenerated file is listed in the task receipt with its reason. The
  exported names stayed, defined as views over the command's declarations, and the commands'
  leading words became non-reserved so the importers keep binding `query`, `property` and `limits`.
- **The inventory check lives in `lint-model`'s reconciliation**, as a new inventory issue kind.
- **The lint rule is a direct-import rule** in the checker's diagnostic form, scoped by the existing
  production-module predicate, with `Temporal.Case` and the Implementation Link as named carve-outs.
- **Tools that served only Operations are deleted** (`NexusDiscovery`, `Inspect`, the
  `umpire-inspect/list/explain` targets) and recorded as a deliberate drop, unless the user asks to
  re-point them at the Caller Model.
- **The offline Observation evaluation is a recorded drop**, with the Umpire evidence tests named as
  the remaining prover.
- **The superseded Nexus instructions are removed with the typed Nexus example** (task .3), where
  fn-85 R10 deferred them.
- **The outage-order rule becomes Producer-derived** for fault-bearing paths (fn-83 .5's dropped
  concern), in its own commit inside task .6.
- **Adjusted 2026-09-19 after fn-85 .1 to .7 landed (tasks .8 to .13 still open).** Four things
  the fn-85 tree now says that this spec's text predates; each is recorded on the task it touches.
  (1) fn-85 .15 made `property` take `machine:`, an optional `when:` and `holds:` with a Lean
  predicate, and rejects `require:` at the key ("the keyed form is retired"); the field-relation
  grammar under API Contracts (`when:` + `require: <path> = <path>`) therefore needs another key
  or another form, a task .2 decision that keeps R2's semantics (three relation forms, four
  rejections, lowering onto `PropertyFieldPath`). (2) fn-85 .7 added the Temporal `case <name>
  realizes <set> as <template> evidence <lines>` block as the Case-producing command, with identity
  `temporal.case.<set>.<query>` and fixture `<set>-<query>-case.json`; the migrated Cases of R3
  and R4 are produced through it (or the shape fn-85 .11 leaves), so their fixture names move and
  `register_case` goes with the last hand-written Case, as .6 says. (3) fn-85 .11 removes the
  template-era `Hook`/`FaultLine` machinery; the design's `workerStop` is an action of the `worker`
  party bound to `FAULT_KIND_WORKER_STOP`, so R4's outage-order rule derives from fault actions on
  the path, not from `fault` lines. (4) A machine's `setup:` parameters are bound by the
  realization and recorded by the Profile, switches are registered with `register_switch`, `ends:`
  is required and steps out of end states are admitted, a Property's fingerprint reads through the
  machine's state fields (fn-85 .4), and Definition IDs derive from `model_conventions`; R5's
  Switch re-authoring meets all of these, so byte-identical goldens are unlikely and the listed
  diff is the expected outcome.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Committed inventory with readers and destinations; reconciliation check | .1 | — |
| R2 | Field relations in `property` | .2 | — |
| R3 | Typed examples migrated; superseded instructions removed | .2, .3 | — |
| R4 | Worker-outage and get-system-info as command Models | .6 | — |
| R5 | Switch re-authored with the commands | .7 | — |
| R6 | Deletions with coverage recorded; Implementation Link re-anchored | .4, .5 | — |
| R7 | The authoring-path-isolation lint rule | .8 | — |
| R8 | Rule drafts, documents, full gate | .9 | — |

## Boundaries
<!-- scope: business -->

- **`Temporal.System.Nexus` is kept**, as the only hand-written Model and the SEM-08 Implementation
  Link; retiring it is a separate decision once the commands can express a system model.
- **Umpire's own unit tests** keep constructing records directly; the gate covers feature Models and
  examples only.
- **Testpilot conformance and synthetic Cases** are kept; they test the runtime.
- **No cancellation** re-authoring: the Race behavior moves into fn-79's text, and fn-79 stays
  deferred.
- **No exploration execution**: the Experimental inputs move into fn-33's text.
- **No Testpilot protocol changes** beyond what the workflow and RPC realizations of R4 need, which
  use existing instructions and fault kinds.
- **No new command concepts** beyond field relations.
- **One monitor per entity** is a follow-up after this spec: a Contract today repeats each rule per
  entity instance (the typed Nexus Case carries its operation rules twice), and once every Case comes
  from the commands, one Contract monitor declared per entity and instantiated per instance can
  replace the copies. It needs its own spec because it changes how the runtime evaluates rules.
- **No edits to historical `.plans` documents** other than `UMPIRE4_ORDER.md` and the drafted rules
  in R8.
- **Depends on fn-85**, whose commands and realization this spec migrates onto.

## Decision Context
<!-- scope: both — conditionally substructured -->

The commands become the only authoring path because two paths double what a developer learns and
what the tree maintains, and the hand-written examples teach the path the design moved away from.

- **Migrate the typed examples rather than delete them first**: they carry the only field-level
  Property coverage; deleting first would drop it silently or leave it as a Known Gap with no owner.
- **Keep `Temporal.System.Nexus`**: it is the spec's only Implementation Link; deleting it amends
  SEM-08 and MOD-04 for no authoring benefit, since developers do not write system models.
- **Delete Race and Experimental files without waiting for fn-79 and fn-33**: both specs are
  deferred or unplanned, so waiting would keep the second path indefinitely; writing the covered
  behavior into their text preserves it.
- **Enforce by imports, not by reviewing constructions**: `lint-model` already checks the import
  graph (MOD-11), and "feature Models import Umpire only through `Umpire.Command`" is mechanical.
- **Re-author the Switch example rather than keep it raw**: an example that builds records by hand
  teaches the path this spec retires; Umpire's internal tests may still build records directly.

Rejected: keeping AUT-08's expert alternative for "Models whose authority is specified
independently" (no remaining feature Model needs it once field relations exist), and a per-file
allowlist in the lint (it would become the second path under another name).

## Parked unknowns

- GOV-02 approval of the rules R8 drafts.
