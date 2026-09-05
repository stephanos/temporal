# Temporal controller Host

`New(Options)` freezes the public Profile and symbolic endpoint bindings, creates lazy shared gRPC
channels, and performs no target calls. Supply its `Snapshot` through `umpire.PrepareCase`. The root
facade passes an admitted `PreparedProgram` to `Open`; this package copies its node bounds and
coordinates without rebinding descriptors, expressions, assignments or projections. Unknown,
streaming and unauthorized methods fail public preparation. Runtime dispatch accepts the prepared
unary descriptor and an already constructed message with that exact input descriptor. Generated
response equivalence and all response projections remain execution responsibilities.

The configured Profile identity must change when authorization, ceilings or endpoint bindings change.
Addresses, gRPC transport/per-call credentials and injected metadata remain Host configuration;
credential rotation for the same authorization does not require a new identity. Channels are reused
across sessions. Call errors expose only canonical status codes, with deadline/cancellation typed
separately; error messages, status details, headers and trailers are never returned. Authorized raw
protobuf responses are ordinary data and are not filtered. The diagnostic method accepts a bounded
number of calls and retains no supplied diagnostic payload.

Profile `MaxActivations` bounds concurrent sessions and total completion-capability creation per
session. `MaxAttempts` bounds attempted effects per session and unfinished effects across the shared
Host, including quarantined effects. Every accepted effect retains that shared capacity until the
actual transport call returns. Wait/Cancel/Drain and all serialized Host operations honor their contexts, including cancellation
while waiting for the Host lock. Close cancels effects, destroys
bridge authority and rejects future operations; it does not wait for transports. A closed session
with unfinished effects retains its Run identity until those effects return. Quarantine is idempotent
and retains existing ownership rather than allocating more capacity. Late completion releases Host
capacity and can only be read through an independently copied effect result; the Host owns no Run
recording or Verdict. The internal Executor owns draining, cleanup, closure ordering and late diagnostics.

`Reserve` rejects worker reservations. The composite Host joins this Session with the worker Session;
worker registration, activation reservation/consumption, and worker DAG cancellation remain there.
The composite must close both sessions and direct quarantine to the owning session.

## Reserved worker delivery

The internal `delivery` ledger is the handoff between the worker Session and the composite Host.
The worker Session passes every newly acquired reservation handle through
`RetainReservation` before returning it to the Executor. The returned proxy is the exact handle the
Executor waits, cancels, drains or quarantines. Later, the composite binds those same proxies to the
prepared `ReservationCarrierPlan`, controller coordinate and requested workflow binding with
`CreateBundle`; foreign, duplicate, partially supplied or already-bound handles reject atomically.

`PrepareRPC` clones a reserved StartWorkflowExecution request, verifies its namespace, workflow ID,
workflow type and task queue, and adds only the private delivery header. A reserved-key collision or
final request-size excess rejects before server `InvokeRPC`; calls without a bundle pass through
unchanged. Workflow admission consumes the exact reservation once and pins the Temporal Run ID.
Matching replay reuses its immutable activation. `PrepareNexus` carries the preassigned
workflow-source/handler route without changing the complete `umpire.Value`; first Nexus admission
pins the Request ID and matching retries reuse it, including after the parent becomes terminal. A
bundle whose handles finish before its start result retains a bounded route-ledger slot until trigger
finalization, so a matching response can still pin and retire it without unbounded tombstones.

Trigger and parent-terminal transitions report only unused reservation release. Failed or uncertain
triggers retire unconsumed routes and request cancellation for admitted work; a terminal parent
releases only its remaining unconsumed Nexus routes. `Stop` is the admission barrier and retries
failed cancellation without releasing unfinished capacity. Quarantine registration unwraps an exact
same-ledger proxy only for the worker callback. Its one-shot completion notification releases ledger
capacity when the raw activation actually ends; registration itself does not release ownership and
requires no watcher goroutine. Completed registrations are idempotent. A concurrent registration
reports a retryable lifecycle error, while a synchronous completion remains authoritative even if
the registration callback subsequently returns an error.

## Completion authority handoff

The composite Host injects a callback into the worker Host that calls the server Session's
`NewCompletionCapability(ctx, originalCoordinate, CompletionInfo)`. The callback translates worker
callback URL, headers, operation token and start time into `CompletionInfo`; the worker need not
import this package. Only trusted Host glue can mint capabilities, after the worker has checked
reservation/activation authority. The original Run/entrypoint/activation coordinate is retained in
the private capability. The server validates the Run and Nexus-handler entrypoint and bounds minting.

The composite exposes the server Session's `CapabilityBridge` to the worker and controller. Publish
requires the exact original coordinate and a declared opaque Slot, rejects conflicting or cross-Run
publication, and permits exact duplicate publication before consumption. Await is context-cooperative;
Consume returns a private claim while the bridge retains the underlying authority. Completion accepts
only the current claim for a capability minted by that Session and consumes completion authority at
effect acceptance. Rejection releases the claim without acquiring a rollback lock; cancellation of
the claim context also permits a fresh cleanup Consume. Replaced claims cannot accept an effect,
and cancellation after successful acceptance cannot restore used authority. A full effect pool
therefore leaves authority recoverable by cleanup; successful acceptance prevents any second completion. The private capability has no
payload accessor, and neither it nor `CompletionInfo` enters expression/projection code. Closing the
Session clears both capability secrets and Slot bindings.

`CompleteNexusOperation` uses the existing Nexus completion HTTP client and Temporal
`PayloadSerializer`. The result is the complete `umpire.Value` protobuf, encoded as a Temporal Payload
with `encoding=binary/protobuf` and `messageType=temporal.server.api.umpire.v1.Value`. This produces
standard Nexus protobuf content and round-trips through Temporal's default SDK data converter into
`*umpirespb.Value`. Worker start/await/finish/result paths must preserve that same generic value
convention; there is no scenario-specific result encoding.

HTTP redirects are disabled so completion credentials cannot follow a redirect. Standard HTTP
transports are cloned with bounded headers/connections; response bodies are bounded and closed on
all outcomes. A supplied custom RoundTripper, like gRPC credential providers, must honor its context
and bound its own internal resources. The effect handle still retains ownership if a transport
violates that contract; no timeout goroutine conceals the unfinished call.

Focused tests use real in-process gRPC/HTTP services. Server tests exercise the Session with admitted
controller metadata and isolated capability fixtures; root preparation covers method-policy rejection,
and the public facade plus async Nexus integration cover composite Run behavior.
