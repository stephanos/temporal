# Mini activity realization

`umpire-temporal` binds the independent `features/activity_mini/model.rs` to the
real generated Temporal API. `Realization` works with that Model's world; `runtime()`
supplies a named delegating world for the guest. Each runtime owns one run.

Judge accepts `umpire.temporal.v1.Observation` containing the exact Describe RPC
method `/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution`
and a `temporal.api.workflowservice.v1.DescribeActivityExecutionResponse` Any.
Running status requires Scheduled or Started run state. Terminal statuses use
Unspecified run state, as emitted by the server. Judge never manufactures Poll;
only a validated callback supplies delivery evidence.

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
