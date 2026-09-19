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
behavior built from a feature's entities, actions, observations and machines. Its logic is ordinary
Lean; the command carries only what Lean cannot infer. A machine declares the entity it tracks
(`for:`), its per-instance state as a `structure` whose fields are all finite (`enum`, `Bool`, or a
`count` bounded by Limits), the `phase` values that end an instance (`ends:`), its setup parameters,
its timers, and one **step function** per action class group:

```lean
def handlerReply (op : Operation) : Reply → List (Operation × Outcome)
  | .async => if op.phase = .scheduled then [({ op with phase := .started }, .acknowledged)] else []
  | .handlerError true =>
      if op.phase = .scheduled then [({ op with phase := .backingOff, attempts := op.attempts + 1 }, .retried)] else []
  | ...
```

- a step returns every successor the server may take with the Model Outcome of each (SEM-07); an
  empty list is a rejection (the action changes nothing); several elements are alternatives, which
  is how a `system` step or a product machine says "either of these";
- `match` arms are the rows: the first matching arm applies, Lean reports a redundant arm, and the
  command reports a state from which an action has no successor with that concrete state as the
  witness unless the state is listed under `ends:`;
- a timer is a step function with no input, enabled while it returns a successor; a fault is an
  ordinary action of a declared party;
- **evidence** is declared in the command, keyed by action class and outcome, each optionally
  guarded by `when` over the state before the step, or `unobservable`, which becomes a Known Gap in
  every Case whose path uses that step.

The command enumerates each step function over the derived finite domain at elaboration into the
same finite table the `model` command produced, so the Behavior Fingerprint, Search, lowering and
inspection see a table and never a function (AUT-05, PLN-02). A `Nat` field, a field of a
non-finite type, or a step whose signature is not `State → Input → List (State × Outcome)` rejects in
place. Rejections are steps that return the `rejected` outcome first, so an action carries no
separate validation block.

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

- each action class to a Testpilot instruction or RPC and to the Program entrypoint that performs
  it (controller, workflow, handler); the binding is per action class, not per party, because one
  party's actions may run from different entrypoints (a handler's completion is a controller
  instruction today);
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

The replaced instructions are removed by fn-86 R3 together with the hand-written typed Nexus
example, the last Producer that emits them; this spec's Cases never use them.

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
  state:    <structure>                        -- fields: <enum> | Bool | count <bound>
  setup:    (<parameter>: <enum> | Bool)*
  timers:   [<timer>, ...]
  steps:    [<step function>, ...]             -- each State → <action input> → List (State × Outcome)
  evidence:
    <action> (<pattern>) → <outcome>: <observation> [when <guard>] (, ...)* | unobservable
    after (<timer>) → <outcome>: <observation> [when <guard>] | unobservable

property <name>
  machine:  <machine>
  holds:    <predicate>                        -- Step → Bool, or Step → Step → Bool (before, after)

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
- **R3:** A machine declares `for:`, `ends:`, a finite state `structure`, setup parameters, timers
  and its step functions under Machines; the command enumerates each step function over the derived
  finite domain into the finite table at elaboration, with a Behavior Fingerprint equal to the one an
  equivalent row table produces; evidence is declared per action class and outcome with optional
  `when` guards; a step with no input is a `system` step. Errors: a state or input field of a
  non-finite type, a step with another signature, a state outside `ends:` from which some action has
  no successor (reported with that state as the witness), `terminal` in a machine without `ends:`, a
  timer no step names, an evidence line naming an outcome the step never returns, and a `system` or
  timer step without evidence or `unobservable` reject in place, pinned by `#guard_msgs`; a redundant
  `match` arm is Lean's own error. A `property` names a machine and a Lean predicate by the same rule,
  `Step → Bool` or `Step → Step → Bool` over the step before and the step after, which the command
  enumerates over the machine's finite table into the existing `PropertyClause` records with the
  fingerprint the keyed form produced; the keyed `require:` form retires with `model`. Errors: a
  predicate over another machine's `Step` type and a predicate that is not decidable reject in place,
  pinned by `#guard_msgs`.
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
  naming the action, field, class and example as a row of fn-87 R13's structured `CaseProvenance`;
  single-value classes carry none.
  Errors: a missing example for a realized multi-value class rejects at Case production naming the
  class.
- **R9:** A Temporal `Realization` in `Temporal.Case` binds action classes (each to its instruction or RPC and its entrypoint), result values,
  the observation catalog and derived observations, timers, setup parameters, switches and
  references; the Producer assembles Program and Contract from a Query's path and the realization;
  whole-Program templates and the `case` command no longer exist; `Umpire.*` passes `lint-model`
  under MOD-01 and SCP-02. Errors: an action class, result value, observation, timer, setup parameter
  or switch the realization does not bind, and a `driven` fault action with no Testpilot fault kind,
  reject at Case production naming it.
- **R10:** Worker instructions carry the Temporal API messages under Typed worker instructions (a
  workflow command's attributes, a Nexus `StartOperationResponse` or `HandlerError` reply, a
  completion payload or failure); no Case the realization produces uses `StartNexusOperation`,
  `RespondNexus`, `NexusResponseKind` or `CompleteNexusOperation`'s untyped result, which stay in
  the protocol only for the hand-written typed Nexus example until fn-86 R3 migrates it and removes
  them (fn-87 left them unrenamed for that removal); a Case declares each observation once and
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
  commands and retiring `model` as a command, an AUT-09 amendment admitting a `structure` of finite
  fields and a step function enumerated into the finite table as author-provided domains and
  behavior, and a MOD-02 amendment allowing realization Evidence mappings in `Temporal.Case`; `DESIGN.md` points at the spec and the Model; fn-83 tasks .4, .5, .6,
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

Task fn-85-model-side-effects-as-typed-actions-and.1 is that proof: it adds the records and the
path-driven Producer with no command syntax, hand-builds Query 2 and compares against the fixture
masked. If it fails, re-evaluate the per-action-class binding before fn-85 .2 and later.

**Amended while implementing .1 (2026-09-14): the comparison is the Program, not the Case.** A
Contract is derived from the checked Property's clauses and the Scenario's action order, and a
correlated clause embeds its trigger action's Model Value and that action's occurrence bound in the
Contract itself, not only in provenance. Re-authoring two waits as three side effects therefore
changes the Contract by construction -- which is the point of the re-authoring, not a defect, and
which no identity mask covers. What the party-to-entrypoint design is answerable for is the Program,
and that is what .1 pins: byte-identical to the template's for the same identity. The Contract's
shape is settled once the Model is authored through the commands in .2 and .3. The stop condition is
unchanged and did not fire.

## Quick commands

```bash
# Model-file specimens and the command surface
cd model && LEAN_NUM_THREADS=1 mise exec -- lake build TemporalModelTests UmpireTests
# Fixtures, protocol and conformance
make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance
# Live tests (once per switch value from fn-85 .5 on)
CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-live-tests
# Full gate and the import rules
CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression
LEAN_NUM_THREADS=1 make lint-model      # baseline 163
```

## Planning decisions

Decided while breaking the spec into tasks (2026-09-12), from the repository and gap scans; each
narrows a requirement without changing its intent, and the task that owns it records the outcome.

- **Records before syntax.** The early proof point builds Umpire records and the path-driven
  Producer first; the `entity`, `action`, `observation`, `machine` and `set` commands elaborate into
  those records afterwards, so the stop condition is measured on the assembly, not on a grammar.
- **Per-action-class binding.** A realization binds each action class to its instruction or RPC and
  to the entrypoint that performs it; a per-party binding is wrong because a handler's completion
  runs as a controller instruction today.
- **Structured machine state is this spec's protocol change.** Per-instance state fields are carried
  in the correlated Contract as named fields (additive on fn-87's shapes), so `attempts` compares as a
  number; the Producer does not flatten a machine's fields into one value.
- **`count` is bounded.** A `count` field is `Fin (bound + 1)` with the bound from Limits, saturating
  to `limitReached`; instance count is a Limit checked before enumeration.
- **`system` rows carry a reserved action.** A row with no action gets a synthesized non-drivable
  action so `Machine.steps` stays action-indexed; it counts toward the step Limit.
- **`schema:` stores only the message name.** Members and examples are checked against the generated
  schema at elaboration and the descriptor is discarded, so a Case's identity never embeds a schema.
- **Refinement witnesses are synthesized.** `refines:`/`map:` discharge the forward simulation by
  `decide` (or `native_decide` as `Property.checked` does), never by an authored proof.
- **Switch values run at the suite level.** The live harness runs one fixture under one environment
  per switch value, the way the upstream HSM and CHASM suites do; setup parameters are dynamic config
  values the environment sets per Run and the Profile records.
- **No wait-for-duration instruction by default.** Timeouts are observed through the timed-out
  history event under a Contract elapsed deadline; the read observation polls one RPC with a bound.
- **The old Nexus instructions stay until fn-86 R3.** The Cases this spec produces never use them.
- **`model` is retired by the `machine` command** (AUT-07a amendment), and the command specimens are
  respelled in the same task.
- **Step functions, not a row grammar (decided with the user 2026-09-12).** A machine's logic is
  an ordinary Lean function over a structure of finite fields, enumerated at elaboration into the
  same finite table; the command keeps declarations, identity, `ends:`, timers, setup parameters and
  evidence links. The row grammar of `.plans/UMPIRE_CMP_FIZZBEE.md` section 4.1's comparison is not
  built, not even as sugar (AUT-07). The first commit of fn-85 .3 is the prototype: the success Model
  as a step function with its fingerprint shown equal to the row form's and its elaboration time
  recorded against the Race baselines (6 to 12 ms per check); if it cannot reproduce the fingerprint,
  fall back to rows with that evidence.
- **A state field is meant by the machine's capability (decided while delivering `.4`,
  2026-09-19).** The Model-side Property evaluator admits a value only through a meaning of the
  capability the Property requires, so a field a Property names -- `phase`, `attempts`, or an
  instance's slot -- is meant beside the states that hold it. Every Property over a machine reads
  through that capability, so every one's fingerprint moved once, by the same cause; the `.15` pins
  showed equality with the keyed form at fcbc068 and now pin the moved values.
- **Instances are a Search view, declared on the Scenario (decided while delivering `.4`).** A
  Scenario's `instances:` count runs the Search over the product of that many copies of the
  machine, built at the data level from the checked declaration with the machine's own law as its
  authority; each slot is a state field of the product state, which is what lets one instance's
  Property be read over the product on the acting slot. A Case follows each operation through one
  sequence, so every instance performs the same actions, and the Producer reads the first instance
  back with every instance's actions as the Program's path. The count is checked against the
  enumeration bound where it is written and is at most nine, because the product's keys number
  instances by one digit and the canonical catalog order Search admits is the order of those keys.
- **A projection rule's confirmed steps are a sequence, not alternatives (recorded while
  delivering `.4`).** One evidence kind confirms one step, continuing from the operation's own
  state, so a structured machine whose action leads to different results from different counts is
  confirmed by a kind per result. The Producer emits a Case's rules from the concrete steps of its
  own path, which is one result per kind on the seven functional Queries; a path that took the same
  kind to two results would need `.9`'s observation declarations to say which.
- **A refinement is a stuttering forward simulation, decided over the tables (decided while
  delivering `.6`).** `refines:` names the product machine and `map:` a Lean function from this
  machine's state to its state, by the rule that replaced rows with step functions. Every row's
  result is a product step from the mapped state -- the product action of the row's own name where
  one carries it, else any that does -- or a stutter when the mapped states are equal; any other row
  rejects at the `map:` line. Outcomes and facts read as the product's value of the same name, a
  fact's constructor covering its members; a fact the product does not name is hidden, an outcome
  it does not name rejects, and a product step may record less than the step it carries, never
  more. The witness is `Umpire.TableRefinement.ofChecked (by decide +kernel)`, synthesized by the
  command and never written, and `Umpire.ImplementationLink.Refinement` carries the obligations to
  the kernels and to traces. A refinement is not an Implementation Link (SEM-08).
- **A product Property is read on the refining machine through a state field (decided while
  delivering `.6`).** A refining machine carries the product state each state reads as in a field
  named after the product machine, so a product claim about a state -- a prior-state trigger, or
  the state a clause fixes -- is a claim about that field, read apart from the state the way any
  field is since `.4`; outcomes, facts and trigger Actions are the refining machine's values of the
  same name, and a Query rejects at `find:` naming one the refining machine lacks. The Property
  keeps its own identity. A stutter is checked like any step, so a product transition claim that
  requires the state to change fails on a stutter; the Nexus Properties are stutter-invariant.
- **A step out of an end state is admitted (decided while delivering `.6`).** `ends:` names the
  phases an instance finishes in, and `DESIGN.md` writes steps out of them -- a completion after
  the operation is over is `notFound`, a worker stopping afterwards records its fault -- which a
  Search takes like any row. The `model`-era refusal of such a step is retired; the Success test
  that pinned it now pins the table as noncanonical, which is the reason that remains.
- **Property bodies are predicates, by the same rule (decided with the user 2026-09-12).** A
  `property` names a machine and a Lean predicate: `Step → Bool` for a same-step claim, or
  `Step → Step → Bool` for a transition claim over the step before and the step after. The command
  enumerates the predicate over the machine's finite table into the existing `PropertyClause`
  records, so Search, fingerprints and Contract lowering never see a function; the keyed
  `require: state:/outcome:/fact:` form of fn-83 is not built. fn-85 .3 shows the success Model's
  `successfulResult` written as a predicate has the same fingerprint as its keyed form. Guards need no
  keyword (an `if` in a step function), and reachability stays the `find` Query, not a second
  `exists` word (SEM-19). Source: `.plans/UMPIRE_CMP_FIZZBEE.md` sections 4.1 and 4.8.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Entities with references and a key; several instances | .2, .4 | — |
| R2 | Actions, classes, schema, results, examples; derived observations | .2 | — |
| R3 | The machine command, its step functions and predicate Properties | .3, .14, .15 | — |
| R4 | Timers, faults, interleavings, Limits accounting | .14, .4 | — |
| R5 | Setup parameters, switches, per-value runs, divergence | .5 | — |
| R6 | Refinement checked by the forward simulation | .6 | — |
| R7 | Sets, bindings, derived Case identity, `umpire-case --list` | .7, .12 | — |
| R8 | Abstraction claims in provenance | .7 | — |
| R9 | Realization, path-driven Producer, templates and `case` removed | .1, .11 | — |
| R10 | Typed worker instructions; one observation declaration per Case | .8, .9 | — |
| R11 | The Nexus Model, seven Queries, `COVERAGE.md`, live tests | .10, .11 | — |
| R12 | Canary and exploratory sets admitted with coverage targets | .12 | — |
| R13 | `AUTHORING.md`, concept entries, rule drafts, fn-83 closure, gate | .13 | — |

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
- **A `steps:` row grammar with guards, alternatives, wildcards, bound names and `+1` updates**
  (2026-09-12): it re-implements `match`, `if`, record update and `List` inside a macro with its own
  diagnostics and learning curve; a Lean step function enumerated into the same table gives the same
  fingerprint, Lean's redundancy check, hover and located errors for free, and is what AUT-01 asks
  for (`.plans/UMPIRE_CMP_FIZZBEE.md` section 4.1).
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

- GOV-02 approval of the rules R13 drafts.
