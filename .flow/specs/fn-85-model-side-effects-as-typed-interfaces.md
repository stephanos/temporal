## Goal & Context
<!-- scope: business -->

A developer who models a Temporal feature today writes waits (`awaitStart`, `awaitSuccess`) and
then picks a hand-written Program template per Case. The side effects that actually decide the
outcome (which RPC, which request fields, which handler reply, which timeout, which cancel) are
invisible to the Model, so a Model cannot distinguish a retryable handler error from a
non-retryable one, and a Property cannot read the fields that make the difference. Precision lives
in templates, and every new behavior needs a new template.

The developer also wants to state coverage per purpose: which Queries run as checked-in functional
tests, which run against a real deployment as a canary, and which an exploratory run should cover.
Today the only unit is a `case` block per Query.

This spec makes three things true:

1. **Side effects are part of the Model.** A Model declares the actions its entities take part in
   (calls, commands, replies) and the observations that confirm them, with typed inputs grouped
   into behavior classes, typed results grouped into classes, and the party that performs each
   action. Umpire stays free of Temporal; a Temporal realization binds each action and observation
   to RPCs, workflow commands, handler replies and history events.
2. **Queries are grouped into sets by purpose.** A set binds each party to the test or to the
   environment and names its Queries or its coverage goal. A functional set compiles to one Case
   per Query.
3. **The Nexus caller-side operation is expressed this way end to end**, with seven functional
   Queries translated from the Nexus functional tests, each run under both Nexus implementations.

The design record, including the server behavior and the survey of all Nexus functional tests this
model must eventually express, is `model/Temporal/Feature/Nexus/DESIGN.md`. Its section 7 records
the five decisions this spec builds on.

This spec supersedes the `case`-block direction of fn-83. fn-83's tasks .4, .5, .6, .8, .16 and .17
are blocked on this redesign; this spec's final task closes them as superseded and names where
each concern went.

## Architecture & Data Models
<!-- scope: technical -->

### Layers

```text
Model file (Temporal.Feature.<Feature>)
  enum / entity / action / observation / environment / machine / link / property / scenario /
  limits / query / set
        |  elaborates into Umpire records (Temporal-free)
        v
Umpire.Command ── Entity, Action, Observation, Party, Dimension, ResultClass, Machine, Row,
                  EnvironmentStep, SetupParameter, Link, Set
        |  checked model + witness trace per Query
        v
Umpire.Case.Producer ── assembles Program and Contract from the witness and a Realization
        ^
        |  binds actions, results, observations, setup parameters and parties
Temporal.Realization.<Feature> ── Temporal-owned; the only place RPC methods, workflow commands,
                                  handler replies, history event kinds and dynamic config keys appear
        |
        v
Case per (set, Query) ── one checked-in fixture; the Profile binds setup parameters and switch values
        |
        v
live Go test per fixture ── runs once per switch value
```

### Entities and structured state

An **entity** is a kind of thing with identity and state: a Nexus operation, a workflow, an
endpoint. Its state is a record of finite fields: an `enum`, a bool, a count bounded by Limits, an
optional or set-valued reference over a finite pool of symbolic slots. An entity may refer to other
entities. A Model's state is a finite collection of entity instances; Limits bound the instance
count. Search stays finite and the existing admission, Search and checked-witness machinery apply
to the product state.

### Actions, observations, parties and kinds

An **action** is a named side effect a party performs. It extends the glossary's Action, which had
none of the following, with:

- a **kind**: `call`, `command` or `reply`;
- an optional **schema**: the protobuf message type, or alternatives, that types its input and
  results; an action whose payload has no protobuf message declares its classes as names the
  Realization interprets, and carries no member check;
- a **party**: the side that performs it, from a party list the feature declares (`caller`,
  `handler` for Nexus), plus the reserved parties `system` and `environment`;
- an **input**: *dimensions* (finite classes over input fields), *references* to entities,
  *defaults* for fields the Model claims do not matter, and *input rules* that normalize or reject
  before any state is read;
- **results**: finite classes of the typed result;
- **representatives**: for each multi-member input class, the one concrete member a functional
  realization uses.

An **observation** is recorded data that confirms a step: it names the entity it belongs to and the
key that identifies the instance, and it is read either from a recorded event or through a read
call. It extends the glossary's Observation with that entity and key.

`Umpire.Operation`'s kinds become `call` and `command` (from `unaryRpc` and `sdkCommand`), `reply`
is added, and `event` becomes the observation declaration. Schemas stay protobuf descriptors.

### Rows

A **machine** block holds step rows, the transition relation the glossary calls a Machine; the
Model is the checked behavior built from its machines, entities, actions and observations. Rows
match on entity state fields, action arguments and results, and
setup parameters, with alternatives (`a | b`), negation (`not terminal`) and omitted fields as
wildcards. Rows are ordered; the first matching row applies, and a row no input can reach is
rejected as shadowed. Each row that a `system` or `environment` party takes names its evidence: a
recorded-event observation keyed to the entity, a read observation (confirmed through a read
call), or an explicit `unobservable` that becomes a Known Gap in every Case whose witness uses
the row.

### Environment

The `environment` party owns timers and faults. A timer is enabled while its row guard holds and
fires as a step; a fault replaces a call's result with a declared fault class. When several entity
instances have enabled steps, any order is a trace. Limits gain an `environment` bound on
environment steps per trace.

### Setup parameters and switches

Machine `setup` parameters are finite, named after the behavior they control
(`concurrencyLimit: [atLimit, belowLimit]`), and bound by the Profile. A **switch** is a rollout
flag between implementations of the same behavior (Nexus HSM or CHASM); it is not a Model
parameter. A set's `repeat` names switch values; each Query's Case runs once per value under a
Profile that sets it, and a verdict that differs between values is reported as a divergence.

### Two levels and a link

A feature may declare a product machine, a mechanism machine and a `link` between them. The link
maps mechanism-machine state to product-machine state and mechanism steps to product steps or to
`stutter`; unmapped values become Known Gaps. It elaborates to `Umpire.ImplementationLink` and is
checked by its forward simulation. A Property declared on the product machine is carried to the
mechanism machine through the link.

### Sets

```text
set functional nexusCaller          bind parties; list Queries; optional repeat
set canary nexusCaller              bind parties; list Queries
set exploratory nexusCaller         bind parties; coverage goal; budget
```

- **functional**: every non-`system` party is bound; each listed `find` Query compiles to one Case
  and one checked-in fixture; fixture and Case identity derive from the set and Query names.
- **canary**: listed Queries; admission rejects a Query whose Case would carry a white-box Known
  Gap or bind a party to test that the canary environment cannot realize. Execution belongs to
  fn-70 and fn-29.
- **exploratory**: a coverage goal over rows, result classes and the members of each claimed input
  class, with a budget. Admission enumerates the coverage targets deterministically. Execution
  belongs to fn-33.

### Class claims

A multi-member input class (one class covering several concrete values) is an **abstraction
claim**. Every Case that realizes it records the claim in Provenance with the representative it
used. A functional realization uses the declared representative; an exploratory run that finds two
members with different verdicts reports a counterexample (execution and Promotion belong to fn-33).
Single-member classes carry no claim.

### Realization

A Temporal-owned `Realization` value per feature binds:

- each action to a Testpilot instruction or RPC and each party to a Program entrypoint
  (controller, workflow, handler);
- each result class to a classification of the concrete result;
- each observation to an observation source, a correlation key path and a kind; a read
  observation is a read RPC (for Nexus, describing the caller workflow's pending operations) whose
  response feeds the correlated evidence, and when the existing projection cannot carry a read
  response, R10 adds that evidence source shape to the protocol;
- each setup parameter and switch value to a dynamic config key and value;
- each entity reference to a runtime identifier (slot, run reference, operation key).

The Producer assembles the Program from the witness trace: steps whose party the set binds to test
become instructions in trace order; `system` steps become Contract expectations through the
projection; `environment` timers become Program-side waits bounded by the realized durations;
environment faults become fault instructions. Whole-Program templates and the `case` block are
removed.

### Testpilot additions for Nexus

The functional Queries need Program instructions the runtime does not have. All additions are
additive protobuf changes.

| Addition | Why |
| --- | --- |
| schedule-to-close, schedule-to-start and start-to-close durations on `StartNexusOperation` | timeout Queries |
| a `RespondNexus` reply form for operation failed; a handler error type and retry behavior on the error form | reply classes and retry Queries |
| an outcome (succeeded, failed) on `CompleteNexusOperation` | async failure Query |
| a read-response evidence source for the correlated projection, only if the existing projection cannot carry a describe response | retry Query's attempt-count observation |

## API Contracts
<!-- scope: technical -->

Command forms are normative; exact grammar is a task decision. They follow the fn-83 respelled
reading rule: a column-0 word is a declaration kind, an indented `word:` is a framework key,
everything else is an author name or value.

```text
entity <name>
  refer:  (<field>: <entity>)*
  key:    <observation key name>
  state:  (<field>: <enum> | bool | count | optional <slotPool> | set <slotPool>)+

action <name>
  kind:    call | command | reply
  party:   <party>
  schema:  <protobuf message> (| <protobuf message>)*   -- optional
  on:      <entity>                        -- or creates: <entity>
  input:   (<field>: [<class>, ...])*
  refer:   (<field>: <entity>)*
  rules:   (<guard> → reject: <result class>)*
  results: (<class>)*
  representatives: (<class pattern> → <concrete member>)*

observation <name> (, <name>)*
  on:   <entity>
  key:  <entity key>                       -- or read: <read call result field>

environment
  fault: (<fault class> on <action>)*
  timer: <name>, ...

machine <name>
  entity: <entity>
  setup:  (<parameter>: [<value>, ...])*
  steps:
    <state guard> + <action>(<argument pattern>)[ → <result class>]
      → <state update>, evidence: <observation> | unobservable

link <mechanism machine> refines <product machine>
  state: (<field>: <value> → <value> | hidden)*
  steps: (<action pattern> → <product step> | stutter)*

set functional | canary <name>
  bind:    (<party>: test | environment)+
  repeat:  <switch> [<value>, ...]         -- functional only
  queries: [<query>, ...]

set exploratory <name>
  bind:    (<party>: test | environment)+
  cover:   rows | results | classMembers   -- one or more
  budget:  <Limits name>
```

Umpire records (Temporal-free; names are contracts):

```lean
namespace Umpire.Command
inductive ActionKind | call | command | reply
inductive PartyBinding | test | environment
structure Dimension where field : String; classes : List ClassDecl
structure ClassDecl where name : String; members : List Value.Raw   -- one member: no claim
structure AbstractionClaim where action : DefinitionId; dimension : String; className : String;
  representative : Value.Raw
inductive SetKind | functional | canary | exploratory
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

- **State space.** Entity instances, slot pools, counts and environment steps are all bounded by
  Limits; exceeding a bound during Search reports `limitReached`, never truncates silently.
- **Shadowed and unreachable rows** reject at elaboration by row position. Two rows that differ
  only in evidence are shadowing.
- **Party binding.** A functional set that leaves a non-`system` party unbound rejects naming the
  party. A Scenario that selects a choice whose party the set binds to environment rejects.
- **Representatives.** A multi-member class without a representative rejects when a functional set
  realizes a witness that uses it; exploratory and canary sets do not require one.
- **Unobservable rows.** A witness through an `unobservable` row still produces a Case, with a Known
  Gap naming the row.
- **Timers in Cases.** A timer row realizes as a concrete duration from the Realization. The live
  test's wall-clock tolerance is a realization parameter, not a Model value; a Case whose witness
  needs a timer duration the Profile's limits reject fails at preparation.
- **Divergence.** When a Query's verdict differs across switch values, the live test fails naming
  the switch values and both verdicts; the spec treats this as a finding, not a flake.
- **Vacuity.** fn-80's early-response rule still applies to every clause the Producer derives.
- **Umpire independence.** `Umpire.*` names no Temporal RPC, command, event, config key or party;
  `lint-model` enforces MOD-01 and SCP-02 on every new module.
- **Fixtures** are generated only; hand edits fail the conformance check (ART-11, ART-12).
- **Protocol compatibility.** Every Testpilot proto change is additive and passes the buf breaking
  check.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A Model file declares entities with structured state fields and references; the model
  command admits several instances bounded by Limits; Search, admission and witness selection work
  over the product state. Errors: a field type outside the finite kinds, an unbounded count, a
  reference to an undeclared entity, and an instance bound of zero reject in place, pinned by
  `#guard_msgs`.
- **R2:** A Model file declares actions with kind, party, optional schema, input dimensions,
  references, defaults, input rules, result classes and representatives, and observations with the
  entity and key they confirm; `Umpire.Operation` kinds are `call`, `command` and `reply`, and
  events are observations; the transition block is spelled `machine`. Errors: unknown party,
  duplicate class, a schema that does not resolve to a protobuf message, a class member outside the
  field's schema, a representative that is not a member of its class, a rule on an undeclared
  field, and an observation on an undeclared entity or key reject in place, pinned by
  `#guard_msgs`.
- **R3:** Step rows guard on state fields, action arguments, results and setup parameters with
  alternatives, negation and wildcards; the first matching row applies; every `system` and
  `environment` row names evidence or `unobservable`. Errors: shadowed row, unreachable row, a
  `system` row without evidence, and evidence naming an undeclared observation reject in place,
  pinned by `#guard_msgs`.
- **R4:** The `environment` party declares timers and faults; timers fire only while their guard
  holds; interleavings across entity instances are traces; Limits bound environment steps. Errors:
  a fault on a non-`call` action and a timer no row guards reject in place; an exhausted
  environment bound reports `limitReached`.
- **R5:** Setup parameters are bound by the Profile and guard rows; a functional set's `repeat`
  runs each Query's Case once per switch value; a differing verdict fails the live test naming both
  values and verdicts. Errors: an unbindable setup parameter yields a Known Gap in the Case; an
  unknown switch value rejects at the set.
- **R6:** `link A refines B` elaborates to a checked `Umpire.ImplementationLink`; a Property on the
  product machine is checked on mechanism-machine traces through the link. Errors: a state mapping to an
  undeclared product value, a step mapping that breaks forward simulation, and an unmapped
  reachable value without a Known Gap reject, each pinned by `#guard_msgs` or a `#guard` on the
  checked link.
- **R7:** `set functional`, `set canary` and `set exploratory` are admitted as described under
  Sets; `umpire-case --list` lists exactly the Queries of functional sets; Case IDs and fixture names
  derive from set and Query names. Errors: an unbound party in a functional set, a `verify` Query
  in a functional or canary set, a canary Query whose Case carries a white-box Known Gap, a
  duplicate derived fixture, and an exploratory set without a coverage goal reject in place.
- **R8:** Every Case that realizes a multi-member input class carries an abstraction claim naming
  the action, dimension, class and representative in its Provenance; single-member classes carry
  none. Errors: a missing representative for a realized multi-member class rejects at Case
  production naming the class.
- **R9:** A Temporal `Realization` binds actions, parties, result classes, observations, setup
  parameters, switches and references; the Producer assembles Program and Contract from the witness
  and the Realization; whole-Program templates and the `case` command no longer exist; `Umpire.*`
  passes `lint-model` under MOD-01 and SCP-02. Errors: an action, result class, observation or
  setup parameter the Realization does not bind rejects at Case production naming it.
- **R10:** The Testpilot protocol, Lean generated declarations and Go runtime support the Nexus
  additions in the Architecture table that the Queries of R11 use; each has a Driver conformance case; buf breaking
  passes. Errors: an invalid duration and a reply form the handler activation does not admit
  reject at Case preparation with the existing preparation error categories.
- **R11:** The Nexus caller-side operation is re-authored as the mechanism machine, product machine
  and link of `DESIGN.md` section 4, without its cancel actions and cancel rows; a functional set `nexusCaller` with `repeat` over the HSM and CHASM
  switch contains exactly these Queries, each with a generated fixture and a passing live test under
  both switch values:
  1. sync success reply completes the operation;
  2. async reply then succeeded callback completes it;
  3. async reply then failed callback fails it;
  4. non-retryable handler error fails it;
  5. retryable handler error then sync success completes it after one backoff, with the attempt
     count observed through a read observation;
  6. schedule-to-start timeout while the handler's worker is stopped times it out;
  7. async reply with start-to-close timeout times it out.

  A `COVERAGE.md` beside the Model maps each assertion of the corresponding upstream tests
  (`TestNexusOperationSyncCompletion`, `TestNexusOperationAsyncCompletion`,
  `TestNexusOperationAsyncFailure`, `TestNexusSyncOperationErrorRehydration`,
  `TestNexusOperationRetriesAfterHTTPFault`, `TestNexusOperationScheduleToStartTimeout`,
  `TestNexusOperationStartToCloseTimeout`) to a Property, an
  observation, or a Known Gap. The async-Nexus fixture is replaced by Query 2's fixture and the
  receipt lists the Program and Contract diff against it. Errors: no error surface beyond R1 to
  R10.
- **R12:** A `set canary nexusCaller` over Queries 1 and 2 with `handler: environment` is admitted,
  and a canary set naming a Query with a white-box Known Gap rejects; a `set exploratory
  nexusCaller` enumerates its coverage targets, pinned by a golden. Errors: no error surface beyond
  R7.
- **R13:** `model/AUTHORING.md` walks from an empty file to a green live test using the Nexus
  Model, with every Lean block equal to a marked region of the Model file, enforced by a Go drift
  test; `UMPIRE4_SPEC.md` gains concept entries for Entity, Party, Set, Realization and Abstraction
  Claim, amends the Action, Observation and Machine entries, and drafts rules under GOV-02,
  including an AUT-07a amendment that names the `entity`, `action`, `observation`, `environment`,
  `machine`, `link` and `set` commands and retires `model` as a command; `DESIGN.md` points at the spec and the Model;
  fn-83 tasks .4, .5, .6, .8, .16 and .17 are closed as superseded with the destination of each
  concern; `make umpire-check-regression` passes. Errors: a missing or duplicate drift marker fails
  the drift test naming the marker.

## Early proof point

Before any Testpilot addition (R10), re-author today's async-Nexus Case as Query 2 on entities,
actions, observations, rows and a Realization (R1 to R3, R9). Its assembled Program and Contract must be
equivalent to the checked-in async-Nexus fixture with identities masked. If the assembled Program
needs a Nexus-specific branch in `Umpire.Case.Producer`, or cannot reproduce the template's
dependency edges from the witness order, stop: the party-to-entrypoint binding is wrong and R4 to
R12 build on it.

## Boundaries
<!-- scope: business -->

- **Composition** (a handler reply built from another entity's actions: update-, query- or
  activity-backed handlers; several callers sharing one handler workflow) is a follow-up spec.
- **History-rewriting actions** (reset and its reapply rules) are out of scope.
- **Eventually consistent observations** (visibility list and count) and **standalone Nexus
  operations** are out of scope.
- **Metrics, spans and logs** are not observations in this spec; assertions on them map to Known
  Gaps in `COVERAGE.md`.
- **Endpoint registry, matching dispatch and cross-cluster topology** are out of scope.
- **HTTP transport faults** get no fault kind; Query 5 reaches the same backoff through a retryable
  handler error, and `COVERAGE.md` records the transport-fault assertion as a Known Gap.
- **Exploratory execution** stays in fn-33 and **canary execution** in fn-70 and fn-29; this spec
  admits those sets and nothing more.
- **External HTTP callers** of Nexus operations and the **worker-outage** Model re-authoring are
  follow-ups.
- **The typed examples** (`TypedUnary`, `TypedNexus`) keep their Programs, Profiles and Contracts.
- **No schema interface**; protobuf descriptors remain the only schema.
- **No edits to historical `.plans` documents** other than `UMPIRE4_ORDER.md` and the
  `UMPIRE4_SPEC.md` concept entries and drafted rules in R13.
- **Nexus operation cancellation** (the cancel request, cancel delivery and canceled resolution,
  with their Testpilot instructions) stays in fn-79, which is deferred until the user asks to
  resume it and then re-plans on this spec's entities, actions and sets.
- **Depends on** fn-84 (recorded in Flow) and on fn-83 tasks .13 (optional Facts), .14 (located
  compile-time diagnostics) and .15 (respelled command surface), which Flow cannot record as a
  cross-spec task dependency; the first task of this spec starts only after fn-83 .15 is done.

## Decision Context
<!-- scope: both — conditionally substructured -->

The five design decisions are recorded in `DESIGN.md` section 7: a party is fixed on the action
and bound per set; the author claims classes and exploration owns their evidence; protobuf
descriptors stay the only schema with structural kind names; one level by default with a product
model when mechanism or several realizations call for it; one Model for HSM and CHASM, repeated per
switch value.

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
- **`interface` as the declaration word**: it reads as a method-set contract type to Go and Java
  readers, collides with the design's own "interface model", and duplicates the glossary's Action,
  which SEM-19 forbids.
- **`model` or `statemachine` for the transition block**: the glossary's word for a transition
  relation is Machine, and Model stays the checked behavior every Property, Scenario and Query
  shares.
- **A cancel Query in this slice**: it and its two Testpilot instructions are fn-79's deferred
  scope, which resumes only on an explicit user request.
- **Transport fault injection for Query 5**: it needs a server test hook the black-box Driver does
  not have; the retryable handler error exercises the same server path.

## Parked unknowns

- The wall-clock tolerance that keeps timer Queries 6 and 7 stable in CI; needs measured runs.
- GOV-02 approval of the rules R13 drafts.
