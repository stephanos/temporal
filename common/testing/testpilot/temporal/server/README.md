# Temporal controller transport

`New(Options)` freezes the public Profile and configured transport endpoints, creates lazy shared
gRPC channels, and performs no target calls. Supply its `Snapshot` through `testpilot.Prepare`. The
root facade passes an admitted `PreparedProgram` to `Open`; this package copies its controller node
bounds and coordinates without rebinding descriptors, expressions, assignments, or projections.

Runtime dispatch accepts a prepared unary descriptor and an already constructed message with that
exact input descriptor. Unknown, streaming, and unauthorized methods fail public preparation.
Authorized raw protobuf responses and protocol status are returned without filtering. Request
construction, response projection, Run recording, and Verdict evaluation remain internal execution
responsibilities.

The configured Profile identity changes when authorization or ceilings change. Prepared and Driver
identity also include the complete immutable Profile binding fingerprint. Symbolic endpoint and
resource binding IDs never carry network addresses. Addresses, gRPC transport credentials,
per-call credentials, and injected metadata remain Driver configuration. Channels are reused across
sessions, and credential rotation for unchanged authorization does not require a new identity.

Profile `MaxActivations` bounds concurrent sessions and capability creation per session.
`MaxAttempts` bounds attempted effects per session and unfinished effects across the shared Driver,
including quarantined effects. Every accepted effect retains shared capacity until its transport
returns. Wait, Cancel, Drain, and serialized Driver operations honor their contexts, including
cancellation while waiting for the Driver lock. Close cancels effects, destroys capability bridge
authority, and rejects future operations without waiting for transports. A closed session with
unfinished effects retains its Run identity until those effects return.

Opaque capabilities carry generic Driver-provided effects. Publish requires the exact originating
coordinate and a declared opaque Slot, rejects conflicting or cross-Run publication, and permits an
exact duplicate before consumption. Await is context-cooperative. Consume returns a private claim
while the bridge retains authority. Invocation accepts only the current claim, clones the typed
input, applies the prepared controller instruction bounds, and consumes authority when the effect is
accepted. A rejected invocation releases its claim so cleanup can consume the capability again.
Replaced claims cannot invoke an effect, and cancellation after successful acceptance cannot restore
used authority. Closing the Session clears effect closures and Slot bindings.

Quarantine is idempotent and retains existing ownership rather than allocating more capacity. Late
completion releases Driver capacity and can only be read through an independently copied effect
result. The diagnostic method accepts a bounded number of calls and retains no supplied payload.
`Reserve` rejects worker reservations; the composite Driver joins this Session with an independently
owned worker Session and directs quarantine to the component that owns each effect.

Focused tests use a real in-process gRPC service for unary transport and generic capability fixtures
for ownership, cancellation, cleanup recovery, limits, and cross-Run isolation.
