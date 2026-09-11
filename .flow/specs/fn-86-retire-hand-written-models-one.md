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

Lint diagnostic for R7:

```text
<module>: imports <Umpire owner> directly; feature Models are authored through Umpire.Command
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
  MOD-10's existing exception; the new lint rule does not widen it.
- **Tools.** The golden, inspect and inventory tools lose their hand-written inputs; each is pointed
  at a command-authored Model or its dependency is removed, and its gate still passes.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A committed inventory lists every non-test module under `Temporal.Feature`,
  `Temporal.Testpilot` and `Umpire.Examples` that builds Umpire records or Testpilot Cases without
  the commands, with each Property, golden, fixture, live test and tool that reads it and a
  destination: migrate, delete with coverage recorded in a named spec, or drop with a reason. Errors:
  a module that builds records but is missing from the inventory fails a Go check that compares the
  inventory to the import graph.
- **R2:** A Property clause compares action input, action result and observation fields through
  their schemas, checked while the Model file compiles and lowered to Contract field reads identical
  in shape to today's hand-written field Properties. Errors: a path segment not in the schema, a type
  mismatch between the compared fields, an observation without a schema, and a repeated-field path
  with no correlation key reject in place, pinned by `#guard_msgs`.
- **R3:** The typed unary and typed Nexus examples are Model files authored with the commands and
  field relations plus a realization; their Cases regenerate with the diff listed; their Go artifact
  tests and live tests pass; the crossed-pairing mutation still yields a Property violation, not an
  admission rejection; the hand-written modules, their Lean tests and their `register_case` lines
  are deleted. Errors: no error surface beyond R2 and fn-85.
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
