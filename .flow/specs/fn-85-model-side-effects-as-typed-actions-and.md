## Goal & Context
<!-- scope: business -->

A developer who models a Temporal feature today writes waits (`awaitStart`, `awaitSuccess`) and
then picks a hand-written Program template per Case. The side effects that actually decide the
outcome (which RPC, which request fields, which handler reply, which timeout) are invisible to the
Model, so a Model cannot distinguish a retryable handler error from a non-retryable one, and a
Property cannot read the fields that make the difference. Precision lives in templates, and every
new behavior needs a new template.

The developer also wants to state coverage per purpose: which Queries run as checked-in functional
tests, which run against a real deployment as a canary, and which an exploratory run should cover.
Today the only unit is a `case` block per Query.

This spec makes three things true:

1. **Side effects are part of the Model.** A Model declares entities, the actions parties perform
   on them with typed inputs grouped into classes, the machines that track each entity's state, and
   the observations that confirm each step. Umpire stays free of Temporal; a Temporal realization
   binds actions, observations, timers and configuration to RPCs, Testpilot instructions, history
   events and dynamic config.
2. **Queries are grouped into sets by purpose.** A set binds each party to `driven` (the Case
   performs its actions) or `observed` (the verifier reads what happened) and names its Queries or
   its coverage goal. A functional set compiles to one Case per Query.
3. **The Nexus caller-side operation is expressed this way end to end**, with seven functional
   Queries translated from the Nexus functional tests, each run under both Nexus implementations.

The design record is `model/Temporal/Feature/Nexus/DESIGN.md`: the seven concepts (section 2), a
specimen (section 3), extension points for the remaining Nexus tests (section 4), and the decisions
this spec builds on (section 6), with the server behavior and the test survey as appendices.

This spec supersedes the `case`-block direction of fn-83. fn-83's tasks .4, .5, .6, .8, .16 and .17
are blocked on this redesign; this spec's final task closes them as superseded and names where
each concern went.

## Architecture & Data Models
<!-- scope: technical -->

### Layers

```text
Model file (Temporal.Feature.<Feature>)
  enum / entity / action / observation / machine / property / scenario / limits / query / set
        |  elaborates into Umpire records (Temporal-free)
        v
Umpire.Command ── Entity, Action, InputField, Example, Observation, Machine, Row, Timer,
                  SetupParameter, Refinement, Set, AbstractionClaim
        |  checked Model + one path per Query
        v
Umpire.Case.Producer ── assembles Program and Contract from the path and a Realization
        ^
        |  binds actions, observations, timers, setup parameters, switches and references
Temporal.Case.<Feature> realization ── Temporal-owned; the only place RPC methods, Testpilot
                                       instructions, history event kinds and dynamic config keys appear
        |
        v
Case per (set, Query) ── one checked-in fixture; the Profile binds setup parameters and switch values
        |
        v
live Go test per fixture ── runs once per switch value
```

### Entities

An **entity** is a kind of thing with identity: a Nexus operation, a workflow. It declares what it
refers to and the key recorded data uses to name an instance. It declares no state. A Model holds
several instances of each entity, bounded by Limits; references are compared, never interpreted.

### Actions and parties

An **action** is a side effect performed by a **party**. Parties are the names a feature uses
(`caller`, `handler`, `network`, `worker`); `system`, the server under test, is reserved and
performs no declared action. An action declares:

- the entity it acts `on:` or `creates:`, or neither;
- its **input fields**, each typed by a finite `enum`; a constructor may carry finite fields
  (`handlerError (retryable : Bool)`), mirroring a protobuf oneof, and each constructor is a
  **class** of concrete values claimed to behave alike;
- an optional **schema**: the protobuf message, or alternatives, that types its input and results;
  when present, class members and examples are checked against it while the file compiles, and an
  action whose payload has no protobuf message omits it;
- optional **results**: a finite enum of what it returns;
- **examples**: for each class with several concrete values, the one a functional Case uses.

An action has no kind. Whether it is realized as a call, a command or a reply is decided in the
realization, where `Umpire.Operation`'s existing kinds stay.

### Observations

An **observation** is recorded data that confirms a row. Evidence names resolve against the
realization's observation catalog (for Temporal, the generated history event kinds and the
Testpilot Run Events) and need no declaration. Only derived observations are declared, such as a
value read through a call (`observation pendingAttempts`, `read: attempts`). Recorded data finds its
instance through the entity's key. An observation may belong to another entity than the row's.

### Machines

A **machine** is the transition relation the glossary calls a Machine; the Model is the checked
behavior built from a feature's entities, actions, observations and machines. A machine declares the
entity it tracks (`for:`), the `phase` values that end an instance (`ends:`), the state it keeps per
instance, its setup parameters, its timers, and its ordered rows. A row reads
**guard** `+` **what happens** `→` **state changes**, **evidence**:

- **guard**: state fields and setup parameters, with alternatives and omitted fields as wildcards;
  `none` before an instance exists; `terminal` and `not terminal` for the `ends:` values;
- **what happens**: an action with a class pattern, alternatives across actions, `after (<timer>)`,
  or nothing, which is a `system` step the server takes on its own when the guard holds;
- **state changes**: field updates; a lowercase name bound in a pattern stored into a field of the
  same type; `reject` when the action changes no state; `result:` for an action with results;
- **evidence**: one or more observations, each optionally guarded by `when` over the state before
  the step, or `unobservable`, which becomes a Known Gap in every Case whose path uses the row.

The first matching row applies; a row no input can reach is rejected as shadowed. Rejections are
rows, so an action carries no separate validation block.

### Timers and faults

A timer is `system` behavior: it is enabled while a row guarded by `after (<timer>)` matches and
fires as a step. A fault is an ordinary action of a declared party (`transportFault` of `network`,
`workerStop` of `worker`). When several entity instances have enabled steps, any order is a path.
Timer firings and fault actions count toward the existing step and action Limits.

### Setup parameters and switches

Machine setup parameters are finite, named after the behavior they control
(`atConcurrencyLimit: Bool`, `recordCancelCompletion: Bool`), and bound by the Profile. A **switch**
is a rollout flag between implementations of the same behavior (Nexus HSM or CHASM); it is not a
Model parameter. The realization declares it, a functional set's `repeat` names it, each Query's
Case runs once per value under a Profile that sets it, and a verdict that differs between values is
reported as a divergence.

### Two machines and a refinement

A feature may declare a product machine and a protocol machine that refines it. Both describe
behavior observable at the API, so both belong to `Temporal.Feature`. The protocol machine declares
`refines:` and a state `map:`; values with the same name map to each other, so the map lists only
the differences, and fields the product machine does not have map to `hidden`. The step mapping is
derived: a protocol row whose mapped states form a product step is allowed, one whose mapped states
are equal is a stutter, and any other row rejects the refinement. The check reuses the forward
simulation inside `Umpire.ImplementationLink`; a refinement is not an Implementation Link, which
SEM-08 reserves for Feature-to-System connections. A Property declared on the product machine holds
on every protocol-machine path.

### Sets

A set names a `purpose:` and binds every party except `system`:

- `driven`: the Case's own Program performs the party's actions, using each class's example;
- `observed`: a real deployment or the world performs them, and the verifier reads which class
  occurred and checks the machine allows it.

A Scenario lists the actions of non-`system` parties in order; under `driven` the Case performs
them, under `observed` the verifier expects them.

- **functional**: each listed `find` Query compiles to one Case and one checked-in fixture; fixture
  and Case identity derive from the set and Query names.
- **canary**: listed `find` Queries; admission rejects a Query whose Case would carry a white-box
  Known Gap. Execution belongs to fn-70 and fn-29.
- **exploratory**: a coverage goal over rows, result values and the members of each claimed class,
  with a budget. Admission enumerates the coverage targets deterministically. Execution belongs to
  fn-33.

### Class claims

A class with several concrete values is an **abstraction claim**. Every Case that realizes it
records the claim in Provenance with the example it used. An exploratory run that finds two members
of one class with different verdicts reports a counterexample (execution and Promotion belong to
fn-33). Single-value classes carry no claim.

### Realization

A Temporal-owned `Realization` value per feature, in `Temporal.Case` beside the templates it
replaces, binds:

- each action class to a Testpilot instruction or RPC, and each party to a Program entrypoint
  (controller, workflow, handler);
- each result value to a classification of the concrete result;
- the observation catalog, each observation's correlation key path, and each derived observation
  to its read; for Nexus, `pendingAttempts` is `DescribeWorkflowExecution`'s
  `pending_nexus_operations.attempt`, declared once in the Case (R10);
- each timer to a concrete duration;
- each setup parameter and switch value to a dynamic config key and value;
- each entity reference to a runtime identifier (slot, run reference, operation key).

It cannot live in `Temporal.System`, because MOD-10 forbids `Temporal.System` from importing the
Feature machines it names; MOD-02 lists Evidence mappings under `Temporal.System`, so R13 drafts an
amendment.

The Producer assembles the Program from a Query's path: actions of `driven` parties become
instructions in path order, each carrying the API message its class example fills; actions of `observed` parties and `system` rows become Contract
expectations through the projection; timers become waits bounded by the realized durations; a
`driven` fault action becomes the Testpilot fault instruction the realization names. Whole-Program
templates and the `case` block are removed.

### Typed worker instructions

The functional Queries need side effects the runtime cannot express: timeouts on the schedule
command, a failed reply, a handler error's type and retry behavior, a failed completion. Instead of
a Testpilot field for each, a worker instruction carries the Temporal API message that already names
those fields, the same message the action's `schema:` names:

| Instruction | Carries | Replaces |
| --- | --- | --- |
| workflow command | a `temporal.api.command.v1.Command` attributes message; for Nexus, `ScheduleNexusOperationCommandAttributes`, which has schedule-to-close, schedule-to-start and start-to-close timeouts | `StartNexusOperation` |
| handler reply | `temporal.api.nexus.v1.StartOperationResponse` (sync success, async success, operation error) or `temporal.api.nexus.v1.HandlerError` (error type, failure, retry behavior) | `RespondNexus` and `NexusResponseKind` |
| operation completion | a `temporal.api.common.v1.Payload` result or a `temporal.api.failure.v1.Failure` | `CompleteNexusOperation`'s untyped result |

The Driver maps each message to the SDK call that produces it: a workflow command to the workflow
SDK call with its options, a reply to the handler's return value or error. The Profile admits
workflow commands per command type. An SDK option with no API field (Nexus `CancellationType`) would
go in an extension field beside the message; this spec adds none, because cancellation stays in
fn-79. Other command types (timers, activities, child workflows) take the same shape when a feature
needs them.

### One observation declaration per Case

A Case today names what it reads twice: Program waits filter history by event type, and the
Contract names the same event kinds, key paths and fields again. A Case instead declares each
observation once, with its source (a history event kind, a Run Event kind, or a read such as
`DescribeWorkflowExecution`), its correlation key path and the fields it exposes. Program waits and
Contract rules refer to it by name. The Producer emits the declarations from the Model's evidence
and the realization's catalog, so `pendingAttempts` gets its read source without a separate
projection shape.

## API Contracts
<!-- scope: technical -->

Command forms are normative; exact grammar is a task decision. They follow the fn-83 respelled
reading rule: a column-0 word is a declaration kind, an indented `word:` is a framework key,
everything else is an author name or value.

```text
entity <name>
  refer:    (<field>: <entity>)*
  key:      <key name>

action <name>
  party:    <party>                            -- any declared name except system
  on:       <entity>                           -- or creates: <entity>; or omitted
  schema:   <protobuf message> (| <protobuf message>)*   -- optional
  input:    (<field>: <enum>)*
  results:  <enum>                             -- optional
  examples: (<constructor pattern> → <concrete member>)*

observation <name>
  on:       <entity>
  read:     <read result field>

machine <name>
  for:      <entity>
  refines:  <product machine>                  -- optional
  map:      (<field>: <value> → <value> | <field>, ... → hidden)*   -- with refines
  ends:     [<value>, ...]
  state:    (<field>: <enum> | Bool | count | optional <slotPool> | set <slotPool>)+
  setup:    (<parameter>: <enum> | Bool)*
  timers:   [<timer>, ...]
  steps:
    <guard> [+ <action> (<pattern>) (| <action> (<pattern>))* | + after (<timer>)]
      → <updates> | reject [, result: <value>],
        evidence: <observation> [when <guard>] (, <observation> [when <guard>])* | unobservable

set <name>
  purpose:  functional | canary | exploratory
  bind:     (<party>: driven | observed)+
  repeat:   <switch>                           -- functional only
  queries:  [<query>, ...]                     -- functional and canary
  cover:    rows | results | classMembers      -- exploratory, one or more
  budget:   <Limits name>                      -- exploratory
```

Umpire records (Temporal-free; names are contracts):

```lean
namespace Umpire.Command
inductive PartyBinding | driven | observed
inductive SetPurpose | functional | canary | exploratory
structure InputField where name : String; domain : Lean.Name   -- a finite enum; each constructor is a class
structure Example where action : DefinitionId; field : String; pattern : String; member : Value.Raw
structure AbstractionClaim where action : DefinitionId; field : String; className : String;
  example : Value.Raw
end Umpire.Command
```

Case identity for a functional set: Case ID `temporal.case.<set>.<query>`, fixture file
`<set>-<query>-case.json`. The Case bytes do not depend on switch values; the Profile does.

```text
umpire-case --list        # <case-id> <fixture-name>, sorted by case-id, every functional-set Query
umpire-case --render <id> # canonical ProtoJSON on stdout
```

## Edge Cases & Constraints
<!-- scope: technical -->

- **State space.** Entity instances, slot pools, counts, timer firings and fault actions are all
  bounded by Limits; exceeding a bound during Search reports `limitReached`, never truncates
  silently.
- **Shadowed and unreachable rows** reject at elaboration by row position. Two rows that differ only
  in evidence are shadowing.
- **Party binding.** A set that leaves a non-`system` party unbound, or binds `system`, rejects
  naming the party. A functional set whose Query path needs an action of an `observed` party
  rejects naming the action, because the Case cannot perform it.
- **Examples.** A class with several concrete values and no example rejects when a functional set
  realizes a path that uses it; exploratory and canary sets do not require one.
- **Unobservable rows.** A path through an `unobservable` row still produces a Case, with a Known
  Gap naming the row.
- **Timers in Cases.** A timer realizes as a concrete duration from the realization. The live test's
  wall-clock tolerance is a realization parameter, not a Model value; a Case whose path needs a
  duration the Profile's limits reject fails at preparation.
- **Faults.** A `driven` fault action with no Testpilot fault kind rejects at Case production; bound
  `observed`, it may occur and the verifier checks the machine allows it.
- **Divergence.** When a Query's verdict differs across switch values, the live test fails naming
  the switch values and both verdicts; the spec treats this as a finding, not a flake.
- **Vacuity.** fn-80's early-response rule still applies to every clause the Producer derives.
- **Umpire independence.** `Umpire.*` names no Temporal RPC, instruction, event, config key or
  party; `lint-model` enforces MOD-01 and SCP-02 on every new module.
- **Fixtures** are generated only; hand edits fail the conformance check (ART-11, ART-12).
- **Protocol changes** build on fn-87's protocol and need not be additive; the buf breaking check
  ignores the Testpilot package. Every changed instruction keeps a Driver conformance case.
- **SDK reach.** A field of a carried API message that the Driver cannot set through the SDK rejects
  at preparation naming the field, never silently dropped.
- **One observation, two readers.** A Program wait and a Contract rule that read the same recorded
  data refer to one declaration; a Case that declares the same source and key path twice rejects.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A Model file declares entities with references and a key; a Model admits several instances
  of each, bounded by Limits; Search, admission and path selection work over every instance's
  machine state. Errors: a reference to an undeclared entity, a duplicate key name, and an instance
  bound of zero reject in place, pinned by `#guard_msgs`.
- **R2:** A Model file declares actions with a party, an optional `on:` or `creates:` entity, input
  fields over finite enums whose constructors may carry finite fields, an optional schema, optional
  results and examples, and declares derived observations; evidence names resolve against the
  realization's observation catalog or a declared observation; the transition block is spelled
  `machine`. Errors: `system` as an action's party, a non-finite input field or constructor field, a
  schema that does not resolve to a protobuf message, a class member or example outside the schema,
  an example that matches no class, and an evidence name that is neither catalogued nor declared
  reject in place, pinned by `#guard_msgs`.
- **R3:** A machine declares `for:`, `ends:`, state fields, setup parameters, timers and ordered rows
  with the guard, action or timer, state-change and evidence forms under Machines; the first
  matching row applies; a row without an action is a `system` step. Errors: shadowed row,
  unreachable row, `terminal` in a machine without `ends:`, `after` of an undeclared timer, a bound
  name stored into a field of another type, `result:` on an action without results, and a `system`
  row or timer row without evidence or `unobservable` reject in place, pinned by `#guard_msgs`.
- **R4:** Timers fire only while a row guarded by their `after` matches; faults are actions of
  declared parties; interleavings across entity instances are paths; timer firings and fault actions
  count toward the step and action Limits. Errors: a declared timer no row uses rejects in place; an
  exhausted bound reports `limitReached`.
- **R5:** Setup parameters are bound by the Profile and guard rows; a functional set's `repeat` runs
  each Query's Case once per switch value; a differing verdict fails the live test naming both
  values and verdicts. Errors: an unbindable setup parameter yields a Known Gap in the Case; an
  unknown switch rejects at the set.
- **R6:** A machine that declares `refines:` and `map:` is checked by the forward simulation inside
  `Umpire.ImplementationLink` with same-named values mapped by default and its step mapping derived;
  a Property on the product machine is checked on the protocol machine's paths. Errors: a mapping to
  an undeclared product value, a protocol value with no same-named product value and no map entry,
  a protocol row whose mapped states are neither a product step nor equal, and `map:` without
  `refines:` reject, each pinned by `#guard_msgs` or a `#guard` on the checked refinement.
- **R7:** `set` declarations with `purpose:` functional, canary or exploratory and `driven` or
  `observed` bindings are admitted as described under Sets; `umpire-case --list` lists exactly the
  Queries of functional sets; Case IDs and fixture names derive from set and Query names. Errors: an
  unbound party, a bound `system`, a functional Query whose path needs an action of an `observed`
  party, a `verify` Query in a functional or canary set, a canary Query whose Case carries a
  white-box Known Gap, a duplicate derived fixture, and an exploratory set without a coverage goal
  reject in place.
- **R8:** Every Case that realizes a class with several concrete values carries an abstraction claim
  naming the action, field, class and example in its Provenance; single-value classes carry none.
  Errors: a missing example for a realized multi-value class rejects at Case production naming the
  class.
- **R9:** A Temporal `Realization` in `Temporal.Case` binds action classes, parties, result values,
  the observation catalog and derived observations, timers, setup parameters, switches and
  references; the Producer assembles Program and Contract from a Query's path and the realization;
  whole-Program templates and the `case` command no longer exist; `Umpire.*` passes `lint-model`
  under MOD-01 and SCP-02. Errors: an action class, result value, observation, timer, setup parameter
  or switch the realization does not bind, and a `driven` fault action with no Testpilot fault kind,
  reject at Case production naming it.
- **R10:** Worker instructions carry the Temporal API messages under Typed worker instructions (a
  workflow command's attributes, a Nexus `StartOperationResponse` or `HandlerError` reply, a
  completion payload or failure), and `StartNexusOperation`, `RespondNexus`, `NexusResponseKind` and
  `CompleteNexusOperation`'s untyped result are removed; a Case declares each observation once and
  Program waits and Contract rules refer to it by name; the Lean generated declarations and the Go
  Driver support both, with a Driver conformance case per carried message and per observation
  source; the Profile admits workflow commands per command type. Errors: an invalid duration, a
  message field the Driver cannot set through the SDK, a reply the handler activation does not
  admit, a command type the Profile does not admit, a reference to an undeclared observation, and a
  duplicate observation declaration reject at Case preparation with the existing preparation error
  categories.
- **R11:** The Nexus caller-side operation is re-authored as the product machine and the protocol
  machine that refines it in `DESIGN.md` section 3, without its cancel actions and cancel rows; a
  functional set with `repeat` over the HSM and CHASM switch contains exactly these Queries, each
  with a generated fixture and a passing live test under both switch values:
  1. sync success reply completes the operation;
  2. async reply then succeeded callback completes it;
  3. async reply then failed callback fails it;
  4. non-retryable handler error fails it;
  5. retryable handler error then sync success completes it after one backoff, with the attempt
     count observed through the `pendingAttempts` read observation;
  6. schedule-to-start timeout after a driven `workerStop` action of the `worker` party, realized by
     the existing worker-stop fault, times it out;
  7. async reply with start-to-close timeout times it out.

  A `COVERAGE.md` beside the Model maps each assertion of the corresponding upstream tests
  (`TestNexusOperationSyncCompletion`, `TestNexusOperationAsyncCompletion`,
  `TestNexusOperationAsyncFailure`, `TestNexusSyncOperationErrorRehydration`,
  `TestNexusOperationRetriesAfterHTTPFault`, `TestNexusOperationScheduleToStartTimeout`,
  `TestNexusOperationStartToCloseTimeout`) to a Property, an observation, or a Known Gap. The
  async-Nexus fixture is replaced by Query 2's fixture and the receipt lists the Program and Contract
  diff against it. Errors: no error surface beyond R1 to R10.
- **R12:** A canary set over Queries 1 and 2 with `handler: observed` is admitted, and a canary set
  naming a Query with a white-box Known Gap rejects; an exploratory set over the protocol machine
  enumerates its coverage targets, pinned by a golden. Errors: no error surface beyond R7.
- **R13:** `model/AUTHORING.md` walks from an empty file to a green live test using the Nexus Model,
  with every Lean block equal to a marked region of the Model file, enforced by a Go drift test;
  `UMPIRE4_SPEC.md` gains concept entries for Entity, Party, Set, Realization, Refinement and
  Abstraction Claim, amends the Action, Observation and Machine entries, and drafts rules under
  GOV-02: an AUT-07a amendment naming the `entity`, `action`, `observation`, `machine` and `set`
  commands and retiring `model` as a command, and a MOD-02 amendment allowing realization Evidence
  mappings in `Temporal.Case`; `DESIGN.md` points at the spec and the Model; fn-83 tasks .4, .5, .6,
  .8, .16 and .17 are closed as superseded with the destination of each concern;
  `make umpire-check-regression` passes. Errors: a missing or duplicate drift marker fails the drift
  test naming the marker.

## Early proof point

Before the typed instructions and observation declarations (R10), re-author today's async-Nexus Case as Query 2 on entities,
actions, a machine, observations and a realization (R1 to R3, R9). Its assembled Program and
Contract must be equivalent to the checked-in async-Nexus fixture with identities masked. If the
assembled Program needs a Nexus-specific branch in `Umpire.Case.Producer`, or cannot reproduce the
template's dependency edges from the path order, stop: the party-to-entrypoint binding is wrong and
R4 to R12 build on it.

## Boundaries
<!-- scope: business -->

- **Composition** (a party bound to another machine: update-, query- or activity-backed handlers;
  several callers sharing one handler workflow) is a follow-up spec.
- **Reusable row fragments** (one retry structure shared by operations and callbacks) are a
  follow-up.
- **History-rewriting steps** (reset and its reapply rules) are out of scope.
- **Eventually consistent observations** (visibility list and count) and **standalone Nexus
  operations** are out of scope.
- **Metrics, spans and logs** are not observations in this spec; assertions on them map to Known
  Gaps in `COVERAGE.md`.
- **Endpoint registry, matching dispatch and cross-cluster topology** are out of scope.
- **HTTP transport faults** get no Testpilot fault kind; `transportFault` stays `observed`, Query 5
  reaches the same backoff through a retryable handler error, and `COVERAGE.md` records the
  transport-fault assertion as a Known Gap.
- **Exploratory execution** stays in fn-33 and **canary execution** in fn-70 and fn-29; this spec
  admits those sets and nothing more.
- **External HTTP callers** of Nexus operations are a follow-up.
- **Hand-written Models and Cases** (the typed examples, the worker-outage and get-system-info
  Cases, the Race, Lifecycle, Operations and Experimental Nexus models, and the Switch example) keep
  working unchanged here; fn-86 retires them onto the commands.
- **No schema interface**; protobuf descriptors remain the only schema. **No rename** of
  `Umpire.Operation`'s kinds; they stay a realization detail.
- **No edits to historical `.plans` documents** other than `UMPIRE4_ORDER.md` and the
  `UMPIRE4_SPEC.md` concept entries and drafted rules in R13.
- **Nexus operation cancellation** (the cancel request, cancel delivery and canceled resolution,
  with their Testpilot instructions) stays in fn-79, which is deferred until the user asks to resume
  it and then re-plans on this spec's entities, actions, machines and sets.
- **The Testpilot protocol's names, structure, expressions and defaults** are fn-87's; this spec
  adds typed worker instructions and Case observation declarations on top of it.
- **Depends on** fn-84 and fn-87 (both recorded in Flow) and builds on fn-83 tasks .13 (optional Facts), .14
  (located compile-time diagnostics) and .15 (respelled command surface), all done on 2026-09-10.

## Decision Context
<!-- scope: both — conditionally substructured -->

The decisions are recorded in `DESIGN.md` section 6: a party is fixed on the action and bound
`driven` or `observed` per set, with `system` the only reserved party; the author claims classes as
enum constructors and exploration owns their evidence, with examples for functional Cases; protobuf
descriptors stay the only schema and actions have no kind; one machine by default, with a product
machine refined by a protocol machine when protocol detail or several realizations call for it; one
Model for HSM and CHASM, repeated per switch value; state belongs to machines, not entities; event
observations resolve from a catalog; validation is rows; the realization lives in `Temporal.Case`.

Rejected:

- **A `case` block per Query with a whole-Program template** (fn-83 R1 and R2): precision lives in
  templates, and every new behavior needs a new template.
- **A separate Temporal binding that holds RPC request details outside the Model**: request fields
  such as a conflict policy or a handler error's retry behavior decide the outcome, so they are
  behavior and belong in the Model as classes.
- **An `implementation: [hsm, chasm]` Model parameter**: it would make the rollout flag part of the
  specified behavior instead of a conformance dimension.
- **Hand-listed concrete requests per Action** (`ParameterDomain` today): one Action per concrete
  request multiplies rows without adding meaning.
- **`interface` as the declaration word**: it reads as a method-set contract type and duplicates the
  glossary's Action, which SEM-19 forbids.
- **`model` or `statemachine` for the transition block**: the glossary's word for a transition
  relation is Machine, and Model stays the checked behavior.
- **A top-level `link` declaration and a "mechanism machine"**: SEM-08 reserves Implementation Link
  for Feature-to-System connections, MOD-02 gives "implementation mechanisms" to `Temporal.System`,
  and a written step mapping duplicated what the state map determines.
- **An action `kind` (call, command, reply)**: it changed nothing the verifier does.
- **An `environment` block and reserved `environment` party**: timers are the server's behavior and
  faults are ordinary actions, and "environment" also named a binding target.
- **Declared event observations, action `rules:` and entity `vary:`**: the event names already exist
  in the generated catalog, rejections are the first rows, and constant traits are setup
  parameters.
- **State on entities**: the product and protocol machines keep different state for the same
  entity.
- **A cancel Query in this slice**: it and its two Testpilot instructions are fn-79's deferred
  scope, which resumes only on an explicit user request.
- **A Testpilot field per server option** (timeouts on `StartNexusOperation`, a reply-kind enum):
  every feature would grow a parallel vocabulary for fields the API messages already name, and the
  action's schema and its instruction would type the same thing twice.
- **Program and Contract declaring observations separately**: the same event kind, key path and
  fields were written twice and could drift.
- **Transport fault injection for Query 5**: it needs a server test hook the black-box Driver does
  not have; the retryable handler error exercises the same server path.

## Parked unknowns

- The wall-clock tolerance that keeps timer Queries 6 and 7 stable in CI; needs measured runs.
- GOV-02 approval of the rules R13 drafts.
