# Contract evaluation

`Prepare` binds a Contract to the immutable Program Observation/bounds view under the Profile's
Contract and correlated ceilings; a Contract declares none of its own. The prepared
Contract implements the internal execution MonitorFactory; `New` creates a fresh `Evaluator`
for every Run. The public Case facade binds that factory and does not accept replacement monitors.

`Observe` processes an appended event synchronously. It stages rule transitions, typed captures,
and supporting sequence references, checks cancellation, then commits the entire event atomically.
Predicates see pre-transition captures. Event-kind indexes preserve declaration order and only the
first matching transition runs. The first committed violation returns `Stop`; later drain/cleanup
events cannot extend or erase the proved bad prefix. Captures retain independent values and their
producing event sequences. Rules and Runs never share mutable state.

Message captures retain one whole declared Observation when later predicates must correlate
multiple fields from the same event. Separate scalar captures cannot preserve that pairing. The
descriptor is bound exactly during preparation, and the same capture-count and byte ceilings bound
the immutable runtime copy; both are Profile ceilings. A capture is typed by a `SingularType` that must be a scalar, enum or
message; preparation rejects any other at the capture.

A bounded-liveness rule's `Deadline` sets one positive bound in its `bound` oneof; preparation rejects
a deadline with no bound or a non-positive one at the rule's deadline. `elapsed_milliseconds` expires before
transitions at the first recorded elapsed coordinate greater than or equal to its Run-relative
deadline; it depends on the clock of the host that produced the Run. `rule_events` expires after
exactly that many Run Events the rule evaluated since its last transition, which is a count of what
the Run recorded and nothing else. One helper owns the counter, and the online `Evaluator.Observe`
path and the offline `PreparedContract.Evaluate` path both reach it through that helper, so neither
can tick on its own terms. The counter resets on each transition into a new state, stops once the
rule reaches a terminal state, and freezes with every other rule effect once execution becomes
incomplete, so no expiry is ever concluded from a truncated Run. A witness must be
strictly earlier than expiry. Early completed closure is inconclusive. `RunEvent.execution_incomplete` takes effect before expiry and remains effective
for later events, even when they omit the flag. Pending rules then stay inconclusive past their
deadline; time and late witnesses cannot manufacture a result.

The Executor records `Run.evaluation_failure_sequence` when an Observe callback fails after event
append. It identifies the first callback whose staged evaluation did not commit. The evaluator
freezes on that failure; later cleanup events may still be recorded. A successful Observe return
is the commit boundary: the Executor must not reinterpret a later `ctx.Err()` as failure of that
already committed event. Offline `Evaluate` replays
only events before that coordinate through the same `Observe` implementation. A Close failure
makes the final disposition incomplete without suppressing proofs already committed by Observe.
No final disposition is retroactively applied to earlier events.

Contract work counts indexed rule visits, expression operations/value bytes, projection traversal,
and capture copies/references. Static preparation bounds that work per event and for the admitted
Run event ceiling; runtime checks both ceilings and capture count/bytes before commit. Every one of
these ceilings is the prepared Profile's snapshot. Run input validation has separate bounded IR
surface/type/fanout checks under the Profile's Program response ceiling.
The shared `internal/ir` interpreter only resolves values from its supplied typed environment;
verification supplies declared Observations, captures, and the closed Run metadata fields.

Kind-specific Run Event data is the event's `payload` oneof, read through a path from the payload
reference whose first segment names the arm (`fault_injected.kind`). One table in `internal/ir`
says which arm each event kind may carry and which kind requires its arm. A transition declares the
arms its filtered kinds may carry, so a path into an arm none of them carries rejects at
preparation, located at that segment. For each evaluated kind, only the arm the kind requires is
available; a path into an arm the event may lack is absent and needs a presence guard.

A correlated capability's trigger, response and correlation are `Expression`s in the correlated
context, but the capability admits and evaluates them itself rather than binding them through the
IR: a FACT step reference is an existential over the step's facts, which no single-valued reference
expresses, and the capability's depth and work ceilings count conditions rather than expression
nodes. Preparation checks their references with `ir.AdmitReferences`, which locates a rejection at
the path binding would report, and checks their shapes against the rule: a trigger reads only the
step's action, and a response only its outcome, state or facts.

Verdict rule results and support references are maintained incrementally at the same atomic event
commit, without recopying prior support history. `Close` polls cancellation while validating the
Run and checks it once more at its successful-return boundary. It transfers the frozen Verdict
once, including on error; further Observe/Close callbacks are rejected. Cancellation cannot
require an uncancelable rebuild or erase a committed proof.

`Evaluate` and live `Close` produce independent protobuf Verdicts. The internal ordered transition
trace and support sequences are deterministic and tested alongside deterministic Verdict bytes
for completed, stopped, and incomplete Runs. Callbacks on one Evaluator must be serialized;
a PreparedContract supports concurrent independent Runs.

Failure ordering also governs positive transitions: once execution is incomplete, later events
still advance validated sequence/elapsed coordinates but cannot commit transitions, captures or
new support. The event first marking incompleteness is already beyond that boundary. A violation
committed earlier remains authoritative; a potential violation on or after failure remains
inconclusive. Live callbacks and offline replay share this rule.
