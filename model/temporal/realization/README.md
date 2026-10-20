# Mini activity realization

`umpire-temporal` binds the independent `features/activity/model.rs` to the
real generated Temporal API. `Realization` implements `Realize<activity::World>`;
`runtime()` uses the macro-exported world directly. Each runtime owns one run.
Generated `prost::Name` implementations supply canonical Any URLs through the
generic `Any::pack`/`unpack` helpers; no hand-maintained type URL registry is needed.

Judge accepts `umpire.temporal.v1.Observation` containing the exact Describe RPC
method `/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution`
and a `temporal.api.workflowservice.v1.DescribeActivityExecutionResponse` Any.
Running status requires Scheduled or Started run state. Terminal statuses use
Unspecified run state, as emitted by the server. Judge never manufactures Poll;
only a validated callback supplies delivery evidence.

Observation carries a result oneof, with response Any at tag 2 or error at tag 3.
The local `umpire.temporal.v1.Error` has code and message because `google.rpc.Status`
is absent from the upstream descriptor closure used by generation. The host transports
the envelope as opaque Any; the Temporal binding owns numeric status interpretation.

For `/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution`,
the actual `RequestCancelActivityExecutionResponse` proposes one `Client::Cancel` and
matches only `Accepted`. An actual error with NotFound code 5 proposes the same step and
matches only `Rejected(NotFound)`, leaving the closed phase unchanged. The binding maps
the recognized status vocabulary and compares the replayed outcome; contradictory or
unknown codes are Unexplained and freeze the run. An unknown method is Unsupported.
Explanations carry parallel steps/outcomes; repeated current Describe carries neither.

The live SDK slice proves successful cancellation while the worker remains stopped and
NotFound after completion, using fresh request IDs and actual RPC results. The client
returns service errors, converted with `serviceerror.ToStatus` to preserve code/message.
It also mutates a copy of the observed NotFound code to FailedPrecondition and checks
Unexplained plus freezing on later genuine observations. Live cancellation in Started
is excluded because the real server passes through CancelRequested, which the independent
six-phase Model omits. Repeated request IDs, repeated cancel and late replies are unmodeled.

Answer accepts `umpire.temporal.v1.Callback` with a partial
`temporal.api.workflowservice.v1.PollActivityTaskQueueResponse`. The input must
contain exactly one Payload with exactly one metadata entry,
`encoding = binary/plain`, and bytes `Complete` or `Fail`.

The returned `umpire.temporal.v1.Reply` contains a partial completed request with
one binary/plain `Complete` result, or a partial failed request with message
`Fail`, application failure type `MiniActivityFailure`, and non_retryable true.
The SDK owns task tokens, identity, timestamps, namespace, worker metadata and
request transport. The callback projection requires only input: the SDK callback
does not expose a whole server poll response. Whole-request equality is not claimed.

Answer validates the entire callback and selected enabled response before Poll
is committed; it leaves the Model Started. The generic runtime commits state only
after successful response publication, handles freezing and Reset, and owns all
ABI allocation behavior. The Temporal guest uses its public export hook.

`annotation_runtime()` wraps the existing `runtime()` with stateless Annotate
dispatch. Its typed `AnnotationRequest` carries the actual RPC method, direction,
protobuf bytes and seed to `activity_api::annotate`. The response retains filled
request bytes or unchanged server response bytes, violations and any explicit
OutOfScope reason. This validates full SDK wire messages separately from the partial
callback/reply projection above, including after Judge freezes the run. The Go
`temporal.Annotate` helper transports these envelopes through `Host.Annotate`.
