# Umpire walkthrough: how a Model works, from nothing

Written 2026-09-30 for a reader who has not seen the model layer before. It uses the Nexus caller
Model (`model/Temporal/Feature/Nexus/Caller/Model.lean`) as the running example and ends with how
the pieces fit the system-level vision in `UMPIRE4_VISION.md`.

Think of it as a board game, a scripted playthrough, and a referee. The model is the rulebook for a
small game. A Case is one playthrough written down in advance. The Go runtime plays that script
against the real Temporal server, and the referee checks that what actually happened is a legal
sequence of moves in the rulebook. Umpire is the referee. Everything else is machinery to write
rulebooks, generate playthroughs, and compare.

## The pieces, bottom up

**1. Vocabulary.** You start by naming things. An entity is a thing with identity, like an
operation or a worker. A party is someone who acts: caller, handler, worker, network, or the system
for timers. An action is a move a party makes, with a small finite menu of inputs. In the Nexus
model, `handlerReply` has one input, the reply, which is one of six values. Each combination of
action and input value is an "action class", so `handlerReply(syncSuccess)` and
`handlerReply(handlerError true)` are two classes. Faults are not special. A worker stopping is
just an action of the worker party.

**2. State.** Each entity has a state, a small record of enums. The Nexus operation's protocol
state is its phase plus an attempt counter plus three deadline flags. Because every field is a
finite enum, the whole state space is finite and can be listed. The protocol state has 192
possible values. That finiteness is the foundation for everything that follows.

**3. Step functions.** For each action you write one ordinary function: given the current state
and the action's input, return the list of allowed outcomes. Each outcome is a triple: the next
state, an outcome label like accepted or notFound, and the facts this step leaves behind. Facts are
the evidence you would expect to see in the real system, such as a history event. An empty list
means the action is not allowed in that state. This is the only real code in a model, and it is
deliberately dumb. No loops, no I/O, just conditions.

```lean
def handlerReplyStep (state : ProductState) (reply : Reply) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase != .scheduled then [] else
  match reply with
  | .syncSuccess => productStep .succeeded .nexusOperationCompleted
  | .async => productStep .started .nexusOperationStarted
  | .operationFailed => productStep .failed .nexusOperationFailed
  | .operationCanceled => productStep .canceled .nexusOperationCanceled
  | .handlerError true => []
  | .handlerError false => productStep .failed .nexusOperationFailed
```

**4. The machine and its table.** A machine bundles a state type, the step functions for its
actions, which states it starts in, which it ends in, and which actions are timers. The checker
then runs every step function on every state for every action class and writes down the results.
The result is a table: 192 states times 23 action classes, each cell holding the allowed steps.
That table is the rulebook in its explicit form. Once it exists, the step functions have done
their job. Everything downstream reads the table, not the functions. This is why the logic can be
shipped to Go as data rather than transpiled: for a finite machine the table is the logic.

```lean
machine nexusProtocol
  for: operation
  state: ProtocolState
  refines: nexusProduct
  map: productOf
  starts: [unscheduled]
  ends: [succeeded, failed, canceled, timedOut]
  timers: [backoff, scheduleToClose, scheduleToStart, startToClose]
  unobservable: [backoff]
  evidence:
    nexusOperationCompleted: nexusOperationCompleted
    pendingAttempts: pendingAttempts
  steps:
    schedule: scheduleStep
    handlerReply: protocolHandlerReplyStep
    complete: protocolCompleteStep
```

**5. Two levels of the same thing.** A product machine describes what the user sees: an operation
is scheduled, then started, then succeeded. A protocol machine describes how the server gets
there: the retry, the backoff timer, which of three deadlines fired. The protocol machine declares
that it refines the product machine and provides a map from protocol states to product states.
The checker walks every row of the protocol table through the map and requires that it either
lands on a product row with the same outcome and at least the product's facts, or is a stutter,
meaning the product state did not change. This check caught two spec mistakes in one afternoon
while the `cmp/` samples were being written. It is also the seed of the two-level idea in the
vision: the feature-level model and the integration-level model related by a checked map.

**6. Properties.** A property is a claim about steps. A same-step claim says: when this action
fires, the resulting step looks like this. A transition claim says something about consecutive
states. Properties are boolean functions over rows of the table.

```lean
property syncSucceeds
  machine: nexusProtocol
  when: handlerReply (syncSuccess)
  holds: fun step =>
    step.state.phase == .succeeded && step.facts.contains .nexusOperationCompleted

property terminalIsFinal
  machine: nexusProduct
  holds: fun before after =>
    !(productTerminal before.state) || after.state.phase == before.state.phase
```

**7. Scenarios.** A scenario is a path: a start state and an ordered list of action classes. It
is the shape of one test, not yet a test.

```lean
scenario syncReplied
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (syncSuccess)]
```

**8. Queries.** A query joins a property, a scenario, and limits. A find query asks: along this
scenario, within these bounds, is there a step where the property holds? The checker searches the
table and either finds the witness path or fails. A verify query asks: does the property hold on
every trace this scenario admits? Find queries become tests. Verify queries are proofs over the
table and produce nothing to run.

```lean
query syncCompletion
  find: syncSucceeds
  in: syncReplied
  limits: two

query terminalHolds
  verify: terminalIsFinal
  in: asyncThenSucceeded
  limits: three
```

**9. Sets.** A set groups queries for a purpose and says which parties the harness drives and
which it only observes. The functional set drives caller, handler and worker. The canary set
drives the caller but only observes the handler, because in production someone else's handler
answers. The exploratory set names no queries. It names a machine and asks the checker to cover
every row, result and class member within a budget, generating one candidate per target.

```lean
set nexusCallerTests
  purpose: functional
  bind:
    caller: driven
    handler: driven
    network: observed
    worker: driven
  repeat: implementation
  queries: [syncCompletion, asyncCompletion, asyncFailure, handlerError, retry,
    scheduleToStartTimeout, startToCloseTimeout]

set nexusCallerExploration
  purpose: exploratory
  bind:
    caller: driven
    handler: driven
    network: observed
    worker: driven
  machine: nexusProtocol
  cover: rows | results | classMembers
  budget: four
```

**10. Cases.** A Case is the query's witness path lowered into something executable: a Program,
which is the script of RPC calls and worker instructions, and a Contract, which is the property
turned into a rule over the evidence the run will produce. The lowering needs a realization, the
piece that knows how `handlerReply(syncSuccess)` becomes a real Nexus handler response. Cases are
protobuf (`proto/internal/temporal/server/api/testpilot/v1/`), are checked in as fixtures, and
are deterministic: the same model yields the same bytes.

```lean
case nexusCallerCases
  realizes nexusCallerTests
  as (Temporal.Case.Realization.asyncNexus "umpire.case.service" "complete")
```

**11. Runtime and verdict.** The Go runtime (`common/testing/testpilot/`) prepares a Case against
a profile, plays the Program against a real server with a real SDK worker, records every event,
and evaluates the Contract. The verdict is satisfied, violated, or inconclusive. Missing evidence
never counts as success. The runtime never interprets the model; it only runs Cases.

## How it grows gradually

The rulebook never has to be complete, and the design already has three ways to say so.

**Empty steps and stutters.** An action the machine does not model returns the empty list. The
product machine cannot see a transport fault at all. The protocol machine records the worker
stopping as a stutter: state kept, nothing recorded. Both are honest statements that this machine
has no opinion, and the refinement check treats stutters as legal.

**Known Gaps.** When a Case cannot check something, it carries a named gap saying what it does
not verify. A gap cannot make a property pass. It only documents the boundary. Cancellation is
not modeled in Nexus today, and the model says so rather than pretending.

**Unrealizable.** When exploration plans a path that performs an action the realization has no
binding for, the candidate is credited as unrealizable and listed. The model can be ahead of the
harness, and the harness reports exactly how far.

The system-level vision extends this in one direction: the rulebook becomes a graph of components
connected by task edges, and a feature model attaches to nodes and edges. The refinement map
between protocol and product is the same mechanism you would use between a feature model and its
integration path. Scoping to a subgraph is choosing which tables the search may read. Fault
injection points are edges. Gaps, stutters and unrealizable targets are the vocabulary for the
parts nobody has specified yet, and the checker's job is to report them, not to hide them.

## Composition: the first taste of the graph

The worker has its own two-state machine, polling or stopped, in
`model/Temporal/Feature/Worker/Model.lean`. Composing it with the operation synchronizes "handler
replies" with "worker serves", so a reply is only possible while the worker polls.

```lean
compose nexusCaller
  for: [operation, Worker.worker]
  state: NexusCallerState
  members:
    operation: nexusProtocol
    worker: handlerWorker
  sync:
    workerStop: operation.workerStop ∥ worker.workerStop
    handlerReply: operation.handlerReply ∥ worker.serve
  starts: [operation.unscheduled, worker.polling]
  ends: [operation.succeeded, operation.failed, operation.canceled, operation.timedOut]

property repliedByPollingWorker
  machine: nexusCaller
  when: handlerReply
  holds: fun step => step.state.worker.phase == .polling
```

The cross-entity claim, that no handler replies while its worker is stopped, is then verifiable
over the product of the two tables. That is the pattern the vision scales up: small tables,
explicit synchronization points, claims checked across them.

## The mental model in one picture

Author writes vocabulary and step functions. Checker expands them into a finite table and proves
the two levels agree. Properties are claims about rows. Scenarios are paths through the table.
Queries search paths for claims and produce witnesses. Sets choose which witnesses become tests,
canaries or exploration. Realizations turn witnesses into executable Cases. The Go runtime plays
Cases against Temporal and referees the result against the model's own evidence expectations.
Everything not modeled is a stutter, a gap, or an unrealizable target, and each is visible.

## Where to look next

- `model/AUTHORING.md` walks the Nexus caller file region by region.
- `model/README.md` covers the Case format, the runtime split, and the exploration and replay
  bridges.
- `cmp/` holds the same two Models written in ten languages, with `cmp/EVAL.md` as the review.
- `UMPIRE4_VISION.md` is the target this walkthrough is measured against.
