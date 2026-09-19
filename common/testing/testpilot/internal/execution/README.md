# Private Program execution

Admission freezes descriptors, expressions, assignment and response-read paths and the Program DAGs.
Instruction inputs and guards bind in the Program expression context, which admits Slot, instruction
outcome, Run and environment references; any other reference rejects with category `unknown` at
its path, such as `program.entrypoints[controller].instructions[call].guard.reference.observation_id`.
An environment reference is admitted only as a whole request assignment value, which admission
resolves to a literal before binding. A Program declares no bindings: `EnvironmentBindingIDs` derives
its binding graph (each binding its roles name, role order, then each other binding an expression
references), and admission resolves exactly that set, rejecting a binding the Profile does not supply
with category `unknown` at `environment`. An instruction's limits are the ones it writes, or the
Profile's `InstructionDefaults` where it writes none; an omitted limit the Profile has no default for
rejects `malformed`.
The value data plane builds requests from those compiled objects and stages immutable outcome,
Slot and Observation values without Driver I/O, recording or Monitor calls. Raw RPC payloads are
validated against the pinned response descriptor and discarded after declared response reads. Equivalent
generated descriptors are accepted through a bounded structural check; reusing a protobuf full name
with different fields, nested messages or enum definitions is rejected. Exact descriptor identity
skips this compatibility walk, and cyclic type graphs terminate through a per-call visited set.
Opaque handles belong to the Driver bridge and never enter this store.

The Driver-facing vocabulary this package schedules against (`Session`, effect and reservation
handles, `Coordinate`, the handle bridge, Profile role policy and `Opcode`) is declared in the
`common/testing/testpilot/contract` leaf and used here directly. This package keeps `Driver`,
`Profile` and the prepared plans, whose methods expose IR, and the public facade re-exports the leaf
by alias, so the handles a Session returns reach execution and come back to it unwrapped.

A `valueStore` owns one Run. Controller activations share its ordinary Slots; worker activations
have separate Slots and outcome references. Activation IDs are unique across the store and each
controller entrypoint activates once. Each accepted attempt keeps an independent outcome snapshot.
`activationValues.request` evaluates the admitted guard before required assignments; false guards
produce no request. `stage` publishes nothing, and `commit` preflights every Slot and attempt before
changing any store state. Batches and their values are package-private, immutable after staging;
consumers must not mutate their outcome, field, Slot or Observation pointers.

The scheduler supplies admitted coordinates and owns activation authority. It stages values
before entering the recorder's publication boundary, checks recorder capacity, and commits the complete
batch there. Each staged fact retains its response read ordinal and protobuf element index, so source
IDs distinguish separate response reads without losing EmitEach order. The recorder supplies source,
causal, sequence and elapsed coordinates and copies facts into its own immutable recording state;
those recorder copies count against recorder work/capacity. Raw payloads and Slots are not evidence.

The Executor wires `valueStore.seal` into the recorder's closure critical section. `Run` invokes that
closure boundary and owns effect draining, cleanup and Driver/opaque-bridge shutdown. Sealing rejects
new request/stage operations and rejects every later commit, including batches staged before seal.
An operation already staging may finish, but cannot publish after sealing. A failed store commit
must never publish its staged facts. Recorder admission and Stop decisions remain the recorder's authority.

Runtime work is independent of IR binding work. The data plane derives a finite ceiling from the
admitted graph's compiled expression nodes, assignments, outcomes and response-read targets, maximum
payload bytes, path fanout and
expression depth, with checked arithmetic, and caches it during admission. Callers use `workLimit()` or a tighter positive
budget. One budget aggregates evaluation, validation, traversal, serialization and ownership copies
across the complete request or stage; exact wire byte ceilings are checked separately from that
accumulated work. No static rebinding occurs during request construction or response reads.

Execution expressions use `Expression.EvaluateExecution`, which shares the evaluator implementation
and semantics with `Evaluate` while charging intermediate decoding, encoding and ownership copies.
Contract evaluation retains its existing admission accounting through `Evaluate`; bounded descriptor
paths keep that existing work finite. Neither entry point rebinds expressions or changes presence,
short-circuiting or comparison behavior. Binding admits an absent comparison operand without a
presence guard, and both entry points evaluate a comparison with an absent operand to false under
every operator, so `NOT_EQUAL` is false there while `not(EQUAL)` is true. An absent value used any
other way (a bare boolean, an instruction input) still needs a guard.

The private `recorder` serializes publication, synchronous Monitor callbacks, ordinary admission,
and closure under one mutex. The scheduler creates it from the prepared view, supplies a monotonic clock
(`time.Now` in production), and first publishes `RUN_OPENED`. Producers supply stable source IDs,
causes and coordinates; the recorder replaces sequence and arrival elapsed values. Source IDs use
the admitted identifier grammar; the recorder reserves its own closure identity. Equal redeliveries
skip both store commit and Observe. Deduplication compares producer failure markers independently
of recorder-latched incompleteness.

`publish` preflights and copies every fact before invoking the supplied store commit. A failing
commit publishes nothing. Once a batch commits, every fact is recorded in order, including facts
after a Stop in the same batch; no new admission can interleave. Observe failures record the exact
first uncommitted evaluation coordinate, freeze subsequent Monitor observation, and retain the
append-only prefix. Successful callback return commits even when cancellation follows it.

`admit` invokes a bounded Driver admission operation and then its ownership-registration callback
before unlocking. Registration receives all returned handles, including partial results on error.
The scheduler validates reservation/effect identities and ceilings there and retains handles for `Run`;
no callback may reenter the recorder. Wait, Cancel, Drain, quarantine and blocking Slot readiness
remain outside this boundary. Stop and incompleteness reject ordinary admission; `Run`'s separately
bounded cleanup is not ordinary admission and must remain unsuppressible.

Event count and source state are bounded by `MaxRunEvents`. Surface validation uses the shared IR
hard ceilings; event metadata and Observation bytes are additionally bounded from the prepared
response ceiling and declarations. Aggregate recording work charges surface traversal, size walks,
deduplication and ownership copies, including redeliveries. Closure retains its independent bounded
snapshot work (at most the admitted event prefix and prepared Monitor result), so exhaustion cannot
prevent terminal transfer. Failure latches once without recursively emitting error events, and
diagnostics retain at most min(`MaxRunEvents`, 64) entries of bounded text.

`Run` supplies the fixed terminal disposition and cleanup outcome to `close`. The recorder seals
ordinary stores under the same barrier, appends a centrally timed closure fact when capacity permits,
and calls Monitor.Close exactly once, even with cancelled context or recording failure. A previously
proved violation survives failure. A close that fails is returned to the caller beside the Run and
the Verdict, which are still produced and unchanged: the recorder's failure is reported, never
swallowed, and never allowed to revise a conclusion the Contract already reached. Returned Run, Verdict and callback inputs have independent mutable
protobuf storage; callers own their snapshots. Repeated closure rejects without transferring again.
`Run` owns actual Driver/bridge closure, cleanup, drain and quarantine, and reports failures
accepted before this boundary before calling it. After closure, publication calls only the injected
Driver diagnostic sink, keyed by Run identity, with a bounded call count; sink failure disables further
calls. No late path can mutate the frozen recording or returned data.

The production Monitor freezes rule transitions when it observes execution incompleteness, before
processing that event. Thus failure followed by a potentially violating in-flight fact remains
incomplete/inconclusive; only a violation committed before failure survives as stopped/violated.
The recorder publishes the failure marker before the callback and does not reinterpret a committed
Monitor result or mask Verdicts differently from offline replay.

The private `scheduler` runs each ordinary controller entrypoint once, using the compiled ready
order and one attempt per enabled node. A node runs after the previous node of its entrypoint unless
its `after` names another set of the same entrypoint (none makes it a root), and without a guard it is
enabled only when every node it runs after succeeded; preparation binds that default as the node's
guard, so its success facts make dependency outcomes available exactly as a written success guard
does, and a literal `true` guard runs the node regardless. An `after` entry naming an unknown node,
the node itself, a node twice, another entrypoint's node or a cycle rejects at its located path.
Independent nodes admit in stable queue order and wait concurrently; dependencies release only after
atomic outcome/Slot/fact publication. False guards release dependencies without creating an
outcome. MaxAttempts is an admission ceiling, not a retry count. Value-Slot readiness uses store notifications; opaque readiness and consumption
stay in the Driver bridge. Neither wait runs under the Monitor barrier. Recorder Stop/failure/closure
wakes the scheduler even when every accepted effect is still waiting.

Reservation admission validates exact counts, zero-based ordinals, unique IDs and matching origins
before the triggering effect. Identity snapshots are captured once during acceptance. All nonnil
returned handles remain in `outstanding`, including partial and malformed Driver returns. Reservation
completion is an activation-level diagnostic fact at its controller origin, causally linked to the
trigger's start; its source includes the reservation's position and ordinal. This does not claim a worker
activation opened or closed: the Driver's separate Consume coordinate may use a different ActivationID.
A reservation that completes with any outcome but success fails the Run, with one exception: a
reserved entrypoint that carries no instruction may go undelivered, so its reservation released
canceled when the parent activation finished is recorded as that diagnostic and the Run goes on. A
Case whose path performs nothing on the handler still reserves the handler's activation, because
the carrier can activate the entrypoint, not because the path needs it to.
Workers retain their own replay-local DAG state and emit no per-SDK-instruction central stream.

Reservation carrier authority is separate from ordinary endpoint method authorization. Each endpoint
policy names unary carrier methods plus maximum counts for supported workflow and Nexus-handler target
contexts. A Case declares no reservations: an ordinary controller instruction invoking a carrier method
on that endpoint reserves one activation of each workflow and Nexus-handler entrypoint whose kind the
carrier's shapes admit, in entrypoint declaration order, and two instructions that could carry the
same entrypoint reject `unsupported` naming both. Admission checks those reservations against the
carrier's maximum counts and compiles their order once. Every potential StartNexusOperation source, including guarded sources, maps to
one explicitly reserved handler by service and operation. Route order follows the prepared workflow
node order, then workflow ordinal; handler ordinals count within the declared handler reservation.
Missing, ambiguous, crossed or count-mismatched routes reject before Driver I/O.

`PreparedProgram.ReservationCarrier` is the immutable Driver seam and returns a
`contract.ReservationCarrierPlan` from the `common/testing/testpilot/contract` leaf. Its exact
reservation topology lets the Driver bind returned reservation identities by entrypoint and ordinal without scanning source
instructions per Run. Its routes bind workflow entrypoint and ordinal plus the prepared SDK source to
the corresponding handler entrypoint and ordinal. Both returned slices are independent copies; the
compiled lookup remains shared and read-only across Runs.

`Run` supplies the bounded context to `execute`. On Stop or failure, it takes the retained
handles and cancellation functions, cancels/drains outside the recorder lock, and processes all
completions accepted before its drain boundary through `publishCompletion` before calling `close`.
It must distinguish expected cancellation from pre-existing Driver failures. Completions beyond that
boundary are post-close diagnostics, never mutations of ordinary values or evidence. The buffered
completion channel can hold every admitted node and reservation once; cooperative waiters finish
without a consumer, while uncooperative Driver waits require quarantine and cannot delay closure.
`waits` must only be joined when Driver cooperation is established; it is not an unbounded drain gate.

Worker adapters use the root `EntrypointPlan.RuntimeWorkLimit` and `InstructionPlan` methods
`OutcomeType`, `EvaluateInput`, `ValidateOutcome`, `TimeoutMilliseconds`, `MaxAttempts` and
`Reservations`. An instruction's outcome fields are derived from it: every instruction has a status and
a detail, `InvokeRpc`, `CompleteNexusOperation` and `NexusOperationCompletion` a protocol code, a
workflow or Nexus-handler instruction an SDK failure code, and `AwaitInstruction` its operation's
result as VALUE: text after a `StartNexusOperation`, the handler's payload as an `Any` after a
`WorkflowCommand` that schedules one.
`OutcomeType` returns a cloned derived schema; `ValidateOutcome` returns an activation-owned
`contract.OutcomeSnapshot` with independently copied outcome and derived fields. Mutating those results
cannot mutate the plan or a subsequent validation result. An RPC response is read only through
response reads, StartNexusOperation's SDK future is an opaque runtime handle, and a Finish or
RespondNexus result ends its activation, so none of them has a VALUE.

A response read target may be a `CorrelatedEvidence` lift rather than a Slot or an Observation. Its
rules are tried in declaration order and the first whose guard is true builds the evidence value
from paths read out of the projected value; a value no rule claims emits nothing. A guard is a
boolean `Expression` in the evidence-lift context, whose only admitted reference is the projected
value: `present(path(projected_value, p))` fires where `p` resolves, and a text requirement conjoins
`compare(EQUAL, path(projected_value, p), text)`. It binds and evaluates through the IR like any
other expression, so an unguarded absent read, a non-boolean guard and a reference outside the
context reject at preparation, the last at the reference's path. Each scope field and evidence field
is a `NamedExpression` whose value is a text literal or `path(projected_value, p)`; the lift reads
those paths itself, so any other expression rejects, a foreign reference at its path. Admission
requires the sink to be
the exact declared `CorrelatedEvidence` Observation, every bound path to read a scalar the portable
evidence domain admits, and the lift to sit on one instruction of a controller entrypoint whose
declared source no other instruction claims — a source ordinal is the position in that source's own
dense stream, and only the emitting instruction can count it.

A rule may instead name one of the Program's evidence declarations (`evidence_id`) and spell nothing
else: the declaration must be a history event kind and the projected value the recorded
`HistoryEvent`, and the rule's guard is the presence of the declared attributes arm, its scope,
operation key and fields the declaration's. `bindEvidence` admits the declarations before the
instructions: each identity once, each source and operation key path once, a history arm that the
event's attributes oneof carries, a Run Event kind that carries a payload, a read method the catalog
knows whose path ends in repeated messages, and every path typed against the recorded value. A Run
Event declaration is lifted by `scheduler.liftRunEvents` as the event is recorded, into the Program's
one `CorrelatedEvidence` Observation, with ordinals dense per source across the Run. Every lift names
as its parent the operation's previously lifted evidence when that came from another source
(`valueStore.chainEvidence`): ordinals order one source's evidence, only a parent orders evidence
across sources, and the Run's own order is the order the Program's instructions took, so an
operation read back by a poll and then by a history read is one comparable chain to the verifier. A read
declaration is polled by a `ReadEvidence` instruction: the Session's `PollRPC` repeats the
declaration's method with the request the assignments build until `readSatisfied` finds an element
of the declared path satisfying `until`, and the instruction's one synthesized response read then
lifts every element the condition selects, `until` doubling as the lift's guard.

`EvaluateInput` evaluates the compiled guard first and skips the input on false. Its callback must
read only that activation's previously validated, immutable field/Slot snapshots, returning nil for
absence. The adapter must not change those values during evaluation, return unvalidated target
payloads, perform SDK/I/O calls or consult mutable Driver/controller state from the lookup. The returned
input is an independent copy. Missing required reads fail through the existing IR evaluator, including
when a success guard passes but its required result is absent. These methods construct no store,
locks, goroutines or SDK objects; activation scheduling, futures and cancellation remain adapter-owned.

Each operation takes a positive work budget no greater than the prepared runtime ceiling and returns
consumed work on success or failure. Callers must subtract consumed work from their activation allowance;
the immutable plan does not keep a mutable budget. The Temporal worker composes these methods through
its private `temporal/internal/activation` state: `Evaluate` resolves local references and consumes
evaluation work, and `Admit` consumes validation work and atomically retains owned fields. Each actual
interpretation gets fresh state; SDK traversal, futures, cancellation and delivery authority remain
in the worker. This composition does not change the public methods or their work units.

Validation charges traversal and ordering before serialization/copies. Runtime protobuf fields use field-number order and map keys use typed order;
derived outcome fields use field-number order, making failure precedence and tight-budget exhaustion
repeatable. Static binding retains its existing finite work accounting. The same validator serves
controller staging; only raw RPC response validation and reads add controller work afterward.
