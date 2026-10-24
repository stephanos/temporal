# Programmed workers

Design of record for fn-243. Proposed 2026-10-23 after the owner deferred the fn-228.5 Shim; the
decisions below were walked through with the owner one by one on 2026-10-24 and accepted.

## Problem

R40 connects every SDK worker to the Model while it runs: in-process Go workers call Answer, and
workers in other languages would reach it through a local Shim. The owner's direction: workers
should know what to do *before* they execute, because in a cloud setup there is no connection
between a worker and Umpire. The Model still decides every worker response; it decides earlier,
and Umpire judges what happened afterwards from the server's history.

## The vision

- For every feature, the Realization defines the raw worker responses: the WorkflowService
  requests a worker sends. For a feature with an SDK, it may also define scripted responses, the
  form an SDK handler consumes (a kitchen sink action set for a workflow).
- Each SDK has one pre-programmable handler per feature, driven by the program. A single activity
  function and a single workflow function serve every scenario because the scenarios are rows in
  the program, not branches in the code.
- The program's only logic is telling scenarios apart within one worker. It is a table of exact
  keys, never a language. Raw only would be wrong: an SDK worker handed raw commands would have to
  reverse the SDK's own translation, which is the layer under test.

## The program

**A table, not a script.** A program maps what a worker can see when a task arrives to what it
sends back. Keys are exact matches over a few task fields. No predicate language (R35 removed CEL
and none returns). A behaviour the key cannot express is not programmed; the Realization refuses it.

**A property of the Model and the World.** Issued once, on Reset, through `ResetResponse.program`,
constant across Runs and seeds. Per-Run variation travels in the inputs the Model proposes: the
Realization stamps an `umpire-scenario` header when it fills a Start, and the worker reads it from
the task. A worker provisioned once from `umpire program` serves any number of Runs; each Run
carries the program in its Reset event, so replay judges against the program in force.

**Why it scales.** Program size is scenarios times visible situations per task kind, both fixed by
the Model. The activity Model is at most nine rows (three scenarios, three admitted attempts), for
one Run or a thousand. A workflow needs one rule: execute the action set carried in the input, which
is how kitchen sink already works.

**Envelope.** `umpire.temporal.v1.WorkerProgram`: guards (namespace, task queue) and ordered rules,
each a key `Any` and response `Any`s; first match wins. Each feature defines its own key and
response types; the envelope never changes for a new feature. A worker handed a key type it does
not know refuses the whole program at start. No fallback field: a task no rule matches is failed
non-retryably with type `umpire.unprogrammed`, a fixed executor contract that a program cannot mask.

**Where it stops.** A behaviour depending on something the worker cannot see at task time has no
key; the Model stamps it into the input when it proposes the Start, or the Realization refuses.
Situations that become visible later (cancellation requested, in a heartbeat response) become key
fields on a heartbeat rule when a Model needs them. Executors keep no state across tasks.

## Executors

Two kinds behind one worker interface per language. `raw` polls with the plain gRPC client and
sends the response as is: the only option for a feature without SDK support (the "fake worker"),
the shortest proof of "no connection", and a byte comparison against the program. `sdk` registers
one generic handler per feature with the SDK. Raw lands first; the SDK executor must produce
identical judgments on the live slice. The worker is part of the system under test: executors live
under `model/temporal/hosts/go/reference/worker`, import only the Temporal API or SDK and the
program proto, and the Run records which executor kind and version ran so a deviation is attributed
to worker, SDK or server correctly.

## Proposals and faults

A proposal is an action Umpire takes, a bootstrap, or a read. Worker actions are observed, never
proposed: `PROPOSAL_KIND_READ` is a Describe loop with no committed action. The Driver's worker gate
and `worker_tasks` go away.

A delay is a fault Umpire injects, so it belongs in a proposal's instruction, not in the program: a
hold keeps worker polls from reaching the server until the proposed request is acknowledged. The
Realization fills it from the seed; the Driver implements it on a connection it owns; an unsupported
hold is a DriverFault, never a verdict. It gives cancel-before-start its determinism and is what
fn-237's schedule-to-start timeout needs.

## Judgment

Server history only. The Describe projection reads `outcome` and `last_failure`, projects multi-step
paths (Scheduled to Completed in one read is Poll then RespondCompleted), and projects nothing when
the observed result or failure disagrees with the program row: that observation is Unexplained. An
unprogrammed task inside the World is a Realization defect, Refused by name. The program is derived
from the one reply policy; there is no second path and no agreement check, replay pins drift, and a
unit test checks coverage.

## Decisions (2026-10-24)

1. Read proposal kind. 2. No agreement check. 3. Scenario in the header. 4. Hold fault, not
`start_delay`, not a program delay. 5. Program on Reset plus `umpire program`. 6. Executors under
`model/temporal/hosts/go/reference/worker`, raw and sdk. 7. After fn-228.6 and fn-244.2, before
fn-228.7, before fn-237.

## Consequences

- R40 is written once, in fn-228.7, in its final form. The Shim (fn-228.5) is superseded; its
  partial start was deleted at the owner's request.
- `ProposeResponse` is unchanged; `ResetResponse` gains the program; `ProposalKind` gains Read.
- The Temporal Driver shrinks to provisioning, the hold, acting and observing.
