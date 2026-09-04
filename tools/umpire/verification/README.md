# Contract evaluation

`Prepare` binds a Contract to the immutable Program Observation/bounds view. The prepared
Contract implements the internal execution MonitorFactory; `New` creates a fresh `Evaluator`
for every Run. The public Case facade binds that factory and does not accept replacement monitors.

`Observe` processes an appended event synchronously. It stages rule transitions, scalar captures,
and supporting sequence references, checks cancellation, then commits the entire event atomically.
Predicates see pre-transition captures. Event-kind indexes preserve declaration order and only the
first matching transition runs. The first committed violation returns `Stop`; later drain/cleanup
events cannot extend or erase the proved bad prefix. Captures retain independent values and their
producing event sequences. Rules and Runs never share mutable state.

Pending liveness expires before transitions at the first recorded elapsed coordinate greater than
or equal to its Run-relative horizon. A witness must be strictly earlier. Early completed closure
is inconclusive. `RunEvent.execution_incomplete` takes effect before expiry and remains effective
for later events, even when they omit the flag. Pending rules then stay inconclusive past their
horizon; time and late witnesses cannot manufacture a result.

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
Run event ceiling; runtime checks both ceilings and capture count/bytes before commit. Run input
validation has separate bounded IR surface/type/fanout checks under Program response limits.
The shared `internal/ir` interpreter only resolves values from its supplied typed environment;
verification supplies declared Observations, captures, and the closed Run metadata fields.

`Evaluate` and live `Close` produce independent protobuf Verdicts. The internal ordered transition
trace and support sequences are deterministic and tested alongside deterministic Verdict bytes
for completed, stopped, and incomplete Runs. Callbacks on one Evaluator must be serialized;
a PreparedContract supports concurrent independent Runs.
