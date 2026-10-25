# Mini activity realization

`umpire-temporal` binds the independent `feature/activity/model.rs` to the
real generated Temporal API. `Realization` implements `Realize<activity::World>`;
`runtime()` creates the explicit three-attempt checker fixture. The guest uses
`annotation_runtime()`, whose Temporal-owned live dispatcher admits one actual execution
inside the shared stateless annotation wrapper.
Generated `prost::Name` implementations supply canonical Any URLs through the
generic `Any::pack`/`unpack` helpers; no hand-maintained type URL registry is needed.

`#[realization(activity)]` on an inherent impl generates the `Realize<activity::World>`
dispatch. `#[observe(WorkflowService::Method, reads(path), ignores(path = "reason"))]`
selects an RPC before decoding its response. The handler receives that typed response,
or `Result<Response, i32>` when it handles raw gRPC error codes. A `Projection` is used
directly; `Step<A> { action, expected }` proposes one action. A state-valued handler must
declare `project(Self::mapping)`, where the Realization-owned function takes the current
Model state and returned value and constructs a `Projection`. The activity mapping
checks the observed attempt, matches the current phase/attempt, and proposes Completed,
nonretryable Failed, or retryable Failed when actual Describe reports the next Scheduled
attempt. It never infers Poll from Started.

`#[answer(Task, reads(path), ignores(path = "reason"),
produces(Request, fields(path), ignores(path = "reason")))]` declares incoming callback
and outgoing request contexts separately. Each answer declares at least one produced
request type and returns `Reply<A> { delivery, action, request }`. `request` is the raw
typed request packed with `Any::pack(&request)`. Generated code validates its declared
field presence and wraps it in the shared `umpire.realization.v1.Reply` envelope exactly once. These are
partial SDK requests: the macro does not run whole-request catalog constraints or require
SDK-filled task tokens. `#[propose]` is reserved and refuses compilation.

Field paths and RPC edges come from the existing admitted catalog declarations and
linked upstream descriptors. Unknown and overlapping paths, empty ignore reasons,
duplicate handlers, unclassified return types and wildcard arms at generated enum
positions refuse compilation. Byte-policy fallbacks remain valid. Unlisted methods
are Unsupported. A present decoded field omitted from both lists also refuses;
optional messages and oneof alternatives use their actual protobuf presence, including
opaque catalog fields. Proto3 scalar defaults have no decoded wire presence. Classifying
a parent stops traversal below it; an opaque parent cannot be traversed. The attributes
declare what the handler reads; unused declared reads are not detected.

Enum admission resolves same-file module imports, grouped imports, type aliases,
local aliases and imported variants, preserving local Model and generic-name
shadowing. Wildcard coverage through unknown external reexports or inferred
helper-return subjects refuses with a located diagnostic; use canonical enum paths or explicit
variants. Conditional or unsupported aliases are treated as unresolved. This is
bounded source analysis, not Rust compiler name or type inference. The attributed
impl must occur literally in readable source; unavailable source scope refuses
compilation. Its source file is tracked as an expansion input. Match expressions
are inspected without changing their evaluation, ownership or moves.
Dereferencing follows only proven built-in references; wildcard coverage after
custom `Deref` is unresolved.
Method-call and `?` result shapes are unresolved unless an explicit binding type
establishes them. Tuple rests use proved arity to align prefixes and suffixes;
unproved enum-like rest coverage refuses.

`Realization::REALIZATION` retains the mapped/ignored/unsupported paths and ignore
reasons for each observed response, incoming callback and outgoing request. Outgoing
mapped paths mean produced fields. The expansion is pinned after formatting with
prettyplease. The generic runtime owns Model replay and first-Unexplained freezing;
the Temporal facade owns the immutable Start receipt and admission refusal.

Judge accepts `umpire.realization.v1.Observation` containing the exact Describe RPC
method `/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution`
and a `temporal.api.workflowservice.v1.DescribeActivityExecutionResponse` Any.
Running status requires Scheduled or Started run state and the current attempt, or
the next Scheduled attempt after a retryable failure. Terminal statuses use
Unspecified run state, as emitted by the server. Judge never manufactures Poll;
only a validated callback supplies delivery evidence.

Observation carries a result oneof, with response Any at tag 2 or error at tag 3.
The local `umpire.realization.v1.Error` has code and message because `google.rpc.Status`
is absent from the upstream descriptor closure used by generation. The host transports
the envelope as opaque Any; the Temporal realization owns numeric status interpretation.

Optional request Any at tag 4 carries the same invocation's actual completed request.
Request-absent bytes and tags 1 to 3 are unchanged; Go emits the oneof after regular
fields while prost keeps number order, and both decode identically. `Observed::request`
decodes it on demand. The live facade additionally retains the original raw `Any.value`
bytes, including unknown fields, and requires actual requests on Describe and cancel.
The request-absent wire encoding remains unchanged; it cannot establish live evidence.
The live transport owner is the after-fill SDK interceptor: a caller-owned context sink
receives one capture per attempt, cloning the request immediately before invoke and the
raw result immediately after it. An RPC error and a later local validation failure are
recorded separately, and a request changed after capture is refused. Association is the
interceptor's trusted same-invocation assertion; the Start result does not echo the
request ID.

For `/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution`,
the actual `RequestCancelActivityExecutionResponse` proposes one `Client::Cancel` and
matches only `Accepted`. An actual error with NotFound code 5 proposes the same step and
matches only `Rejected(NotFound)`, leaving the closed phase unchanged. The realization maps
the recognized status vocabulary and compares the replayed outcome; contradictory or
unknown codes are Unexplained and freeze the run. An unknown method is Unsupported.
Explanations carry parallel steps/outcomes for the complete retained witness from Reset.
Repeated current Describe and validated Start replay return that full witness; initial
Start admission retains its actual observation anchor with an empty path.

The live SDK slice proves successful cancellation while the worker remains stopped and
NotFound after completion, using fresh request IDs and actual RPC results. The client
returns service errors, converted with `serviceerror.ToStatus` to preserve code/message.
It also mutates a copy of the observed NotFound code to FailedPrecondition and checks
Unexplained plus freezing on later genuine observations. Live cancellation in Started
is excluded because the real server passes through CancelRequested, which the independent
six-phase Model omits. Repeated request IDs, repeated cancel and late replies are unmodeled.

Answer accepts `umpire.realization.v1.Callback` with a partial
`temporal.api.workflowservice.v1.PollActivityTaskQueueResponse`. The input must
contain exactly one Payload with exactly one metadata entry,
`encoding = binary/plain`, and bytes `Complete`, `Fail` or `RetryThenComplete`.
Callbacks carry the actual SDK namespace, activity ID, activity run ID and attempt;
the facade checks the execution tuple before the state-aware delivery guard.

The returned `umpire.realization.v1.Reply` contains a partial completed request with
one binary/plain `Complete` result, or a partial failed request with message
`Fail`, application failure type `MiniActivityFailure`, and non_retryable true.
`RetryThenComplete` returns a retryable failure on attempt one and completes later attempts.
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
`umpire.Annotate` helper transports these envelopes through `Host.Annotate`.

The same Annotate Any also accepts `ProbeRequest`. Inventory derives expectation
identities from typed catalogs and registers their RPC contexts, including nested
failure variants consumed by Failed. It checks registration against the macro's
independent declaration inventory and the three supplemental catalogs. Prepare
uses supplied namespace/activity/queue/run/token prerequisites, constructs a valid
baseline, and proves the mutated protobuf after encode/decode against every
applicable catalog. Required-field omissions have separate case identities.
Overlong request IDs necessarily also violate their supplemental UUID intent;
only that exact `Accepts` companion is permitted and appears in the report.

Large witnesses remain materialized and validated in Rust. Their response carries
a bounded literal/repeat recipe with exact decoded length and FNV-1a 64 integrity
check, allowing the host to reconstruct the actual RPC bytes without increasing
the ABI buffer limit. FNV detects accidental corruption and is not authentication.
Classify requires exactly one observation per contextual case and renders the
value-free gap report; generation, prerequisite and coverage failures are refusals.

## Live World admission

Only a canonical Start request and successful Start response from one captured invocation
can admit a World. The transport owner supplies that association; protobuf bytes cannot
authenticate it. Admission validates the existing catalogs, nonempty namespace/activity/
request/run identity and an optional activity Link before checking World domains. It never
fills, redacts or repairs captured evidence. Diagnostics name fields without including values.

| Captured Start field | Admission in fn-227.5 |
| --- | --- |
| `retry_policy.maximum_attempts` | Must be explicitly present in `1..=3`; selects the exact World before construction. |
| `schedule_to_start_timeout` | Recorded exactly with the receipt; not admitted. fn-237.4 owns future admission. |
| `start_to_close_timeout` | Recorded exactly with the receipt; not admitted. fn-237.4 owns future admission. |
| `schedule_to_close_timeout` | Recorded exactly with the receipt; not admitted. fn-237.4 owns future admission. |
| `heartbeat_timeout` | Recorded exactly with the receipt; not admitted. fn-237.4 owns future admission. |

Present valid timeout values are neither refused nor quantized into Model parameters.
A successful new-run result (`started=true`) prepares a whole replacement Runtime with
that World's unique Scheduled/attempt-one start and no exploration. Before admission,
Explore/Rows concern only the explicit fixture World; Judge/Answer cannot use it as a live
realization. Each execution needs a fresh guest. Original exact request/run replay is zero-step
after progress and Reset, including `started=false` only with retained original successful
provenance. Same-ID changed bytes, changed response evidence or crossed executions refuse.

Describe and cancel require the bound namespace/activity/run tuple in their actual request;
an omitted run ID is insufficient. Describe's outer run ID and Info activity/run IDs must
agree. Describe attempt must match the Model or evidence its next Scheduled retry. Callback
identity and attempt are checked before delivery; a rejected stale Poll cannot authorize a reply.

The first domain failure prepares field-named Unsupported plus an owned admission-freeze
transaction. It retains the old Runtime and exploration. Malformed, failed and crossed evidence
cannot overwrite it, and stateless Annotate/Probe remain available. Reset composes Model Reset
with admission-freeze clearing, retaining World, receipt and execution identity. A rejected
prebinding candidate remains unadmitted after Reset and cannot be replaced by changed options.

All effects commit as owned moves after guest response publication. Native false/panicking
publishers, oversized responses and actual Wasm exhaustion of all 16 MiB of live ABI buffers
leave World/state/exploration/receipt/freeze unchanged; retry publishes once. Admission success
invalidates exploration only when committed. This boundary is the guest-owned response buffer:
a later host read failure cannot undo its commit. A remote successful Start is also irreversible;
the live owner retains the same capture to retry publication idempotently.

The live SDK proof admits Start before worker startup and holds callbacks while synchronously
reading and judging actual Describe. `RetryThenComplete` serializes actual Poll RPCs with one delivery permit. Empty and failed
polls restore permission; a nonempty delivery consumes it. After the real failed-attempt RPC
succeeds, the proof delays Describe beyond the retry interval while the next Poll remains
blocked. It judges fresh Scheduled attempt two, releases the next Poll and answers that real
second delivery, then observes completion. The six existing Complete, Fail,
HostRefusal, context-Cancellation and before/after-start cancellation controls remain separate.
