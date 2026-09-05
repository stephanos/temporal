# Private Program execution

Admission freezes descriptors, expressions, assignment/projection paths and the Program DAGs.
The value data plane builds requests from those compiled objects and stages immutable outcome,
Slot and Observation values without Host I/O, recording or Monitor calls. Raw RPC payloads are
validated against the pinned response descriptor and discarded after declared projections. Equivalent
generated descriptors are accepted through a bounded structural check; reusing a protobuf full name
with different fields, nested messages or enum definitions is rejected. Exact descriptor identity
skips this compatibility walk, and cyclic type graphs terminate through a per-call visited set.
Opaque capabilities belong to the Host bridge and never enter this store.

A `valueStore` owns one Run. Controller activations share its ordinary Slots; worker activations
have separate Slots and outcome references. Activation IDs are unique across the store and each
controller entrypoint activates once. Each accepted attempt keeps an independent outcome snapshot.
`activationValues.request` evaluates the admitted guard before required assignments; false guards
produce no request. `stage` publishes nothing, and `commit` preflights every Slot and attempt before
changing any store state. Batches and their values are package-private, immutable after staging;
consumers must not mutate their outcome, field, Slot or Observation pointers.

The scheduler supplies admitted coordinates and owns activation authority. It stages values
before entering the recorder's publication boundary, checks recorder capacity, and commits the complete
batch there. Each staged fact retains its projection ordinal and protobuf element index, so source
IDs distinguish separate projections without losing EmitEach order. The recorder supplies source,
causal, sequence and elapsed coordinates and copies facts into its own immutable recording state;
those recorder copies count against recorder work/capacity. Raw payloads and Slots are not evidence.

The Executor wires `valueStore.seal` into the recorder's closure critical section. `Run` invokes that
closure boundary and owns effect draining, cleanup and Host/opaque-bridge shutdown. Sealing rejects
new request/stage operations and rejects every later commit, including batches staged before seal.
An operation already staging may finish, but cannot publish after sealing. A failed store commit
must never publish its staged facts. Recorder admission and Stop decisions remain the recorder's authority.

Runtime work is independent of IR binding work. The data plane derives a finite ceiling from the
admitted graph's compiled expression nodes, assignments, outcomes and projection sinks, maximum
payload bytes, path fanout and
expression depth, with checked arithmetic, and caches it during admission. Callers use `workLimit()` or a tighter positive
budget. One budget aggregates evaluation, validation, traversal, serialization and ownership copies
across the complete request or stage; exact wire byte ceilings are checked separately from that
accumulated work. No static rebinding occurs during request construction or response projection.

Execution expressions use `Expression.EvaluateExecution`, which shares the evaluator implementation
and semantics with `Evaluate` while charging intermediate decoding, encoding and ownership copies.
Contract evaluation retains its existing admission accounting through `Evaluate`; bounded descriptor
paths keep that existing work finite. Neither entry point rebinds expressions or changes presence,
short-circuiting or comparison behavior.

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

`admit` invokes a bounded Host admission operation and then its ownership-registration callback
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
proved violation survives failure. Returned Run, Verdict and callback inputs have independent mutable
protobuf storage; callers own their snapshots. Repeated closure rejects without transferring again.
`Run` owns actual Host/bridge closure, cleanup, drain and quarantine, and reports failures
accepted before this boundary before calling it. After closure, publication calls only the injected
Host diagnostic sink, keyed by Run identity, with a bounded call count; sink failure disables further
calls. No late path can mutate the frozen recording or returned data.

The production Monitor freezes rule transitions when it observes execution incompleteness, before
processing that event. Thus failure followed by a potentially violating in-flight fact remains
incomplete/inconclusive; only a violation committed before failure survives as stopped/violated.
The recorder publishes the failure marker before the callback and does not reinterpret a committed
Monitor result or mask Verdicts differently from offline replay.

The private `scheduler` runs each ordinary controller entrypoint once, using the compiled ready
order and one attempt per enabled node. Independent nodes admit in stable queue order and wait
concurrently; authored dependencies release only after atomic outcome/Slot/fact publication.
False guards release dependencies without creating an outcome. MaxAttempts is an admission ceiling,
not a retry count. Value-Slot readiness uses store notifications; opaque readiness and consumption
stay in the Host bridge. Neither wait runs under the Monitor barrier. Recorder Stop/failure/closure
wakes the scheduler even when every accepted effect is still waiting.

Reservation admission validates exact counts, zero-based ordinals, unique IDs and matching origins
before the triggering effect. Identity snapshots are captured once during acceptance. All nonnil
returned handles remain in `outstanding`, including partial and malformed Host returns. Reservation
completion is an activation-level diagnostic fact at its controller origin, causally linked to the
trigger's start; its source includes the declaration and ordinal. This does not claim a worker
activation opened or closed: the Host's separate Consume coordinate may use a different ActivationID.
Workers retain their own replay-local DAG state and emit no per-SDK-instruction central stream.

Reservation carrier authority is separate from ordinary endpoint method authorization. Each endpoint
policy names unary carrier methods plus maximum counts for supported workflow and Nexus-handler target
contexts. Admission checks each reservation-bearing RPC against that policy and compiles its authored
reservation order once. Every potential StartNexusOperation source, including guarded sources, maps to
one explicitly reserved handler by service and operation. Route order follows the prepared workflow
node order, then workflow ordinal; handler ordinals count within the declared handler reservation.
Missing, ambiguous, crossed or count-mismatched routes reject before Host I/O.

`PreparedProgram.ReservationCarrier` is the immutable Host seam. Its exact reservation topology lets
the Host bind returned reservation identities by entrypoint and ordinal without scanning source
instructions per Run. Its routes bind workflow entrypoint and ordinal plus the prepared SDK source to
the corresponding handler entrypoint and ordinal. Both returned slices are independent copies; the
compiled lookup remains shared and read-only across Runs.

`Run` supplies the bounded context to `execute`. On Stop or failure, it takes the retained
handles and cancellation functions, cancels/drains outside the recorder lock, and processes all
completions accepted before its drain boundary through `publishCompletion` before calling `close`.
It must distinguish expected cancellation from pre-existing Host failures. Completions beyond that
boundary are post-close diagnostics, never mutations of ordinary values or evidence. The buffered
completion channel can hold every admitted node and reservation once; cooperative waiters finish
without a consumer, while uncooperative Host waits require quarantine and cannot delay closure.
`waits` must only be joined when Host cooperation is established; it is not an unbounded drain gate.

Worker adapters use the root `EntrypointPlan.RuntimeWorkLimit` and `InstructionPlan` methods
`OutcomeType`, `EvaluateInput` and `ValidateOutcome`. `OutcomeType` returns a cloned declared schema;
`ValidateOutcome` returns an activation-owned `OutcomeSnapshot` with independently copied outcome and
declared fields. Mutating those results cannot mutate the plan or a subsequent validation result.
StartNexusOperation cannot declare VALUE because its SDK future is an opaque runtime handle; Await
validates the target result against its declared VALUE type. Finish and every RespondNexus variant
retain their evaluated result expressions and may declare their typed VALUE.

`EvaluateInput` evaluates the compiled guard first and skips the input on false. Its callback must
read only that activation's previously validated, immutable field/Slot snapshots, returning nil for
absence. The adapter must not change those values during evaluation, return unvalidated target
payloads, perform SDK/I/O calls or consult mutable Host/controller state from the lookup. The returned
input is an independent copy. Missing required reads fail through the existing IR evaluator, including
when a success guard passes but its required result is absent. These methods construct no store,
locks, goroutines or SDK objects; activation scheduling, futures and cancellation remain adapter-owned.

Each operation takes a positive work budget no greater than the prepared runtime ceiling and returns
consumed work on success or failure. Adapters subtract consumed work from their activation allowance;
the immutable plan does not keep a mutable budget. Validation charges traversal and ordering before
serialization/copies. Runtime protobuf fields use field-number order and map keys use typed order;
declared outcome fields use declaration order, making failure precedence and tight-budget exhaustion
repeatable. Static binding retains its existing finite work accounting. The same validator serves
controller staging; only raw RPC response validation/projection adds controller work afterward.
