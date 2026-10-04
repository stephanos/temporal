# Simulation progress lifecycle decision

Task 16 should give the coordinator one closed, typed transition entry on a
private progress owner under the existing arbiter mutex. Each event describes a
complete protocol step. The owner validates the entire step before consuming
arrival credits, reserving work, waking a waiter or changing response phases.
This choice removes caller sequencing while retaining aggregate wire arrivals
and simultaneous blocking operations on one participant.

Task 15 supplies behavioral tests and this decision only. R11 remains open until
task 16 implements the owner and passes state-machine and real process
conformance. The unchanged production source is the task 13/14 candidate on
`0dd05b313acd0986312da7fd3159520e6a21f1bf`. This development host runs stock
Go 1.27.1 on linux/arm64 and has no patched `.toolchain/bin/go`. These tests
provide no native-runtime, cross-process timing or exact-replay qualification.

## Current guarantees and exposed gaps

`simulation_time.go` owns time, participant membership, generation validation,
external work, active handling, delivered credits and quiescent waiters.
`simulation_unix.go` separately owns coordinator response phases. Its callers
sequence updates across the coordinator mutex and arbiter mutex.
`simulation_model.go` owns pending and abandoned exchange correlation, including
the cancellation race where an already claimed reply wins cancellation.
`process_unix.go` connects model transport callbacks to participant accounting.

The added tests exercise coordinator frames, time requests, real framed pipes
and the real model transport. They assert complete time responses, quiescence,
error classification, callback selection and retained World state. The commit
fixture uses `serveSimulationModels` with a real World registration and delays
the response until cancellation has returned. It proves transport cancellation
does not undo or repeat that committed handler mutation. It does not prove the
network or volume process backend, native timers or hard isolation.

| Behavior | Characterization |
| --- | --- |
| Admission at an installed waiter wakes External without advancing time; a delivered unconsumed reply returns Retry | `TestSimulationTimeProgressArrivalAtInstalledQuiescence` |
| Two operations from the same node remain pending and correlate independently in both reply orders; one consumed arrival leaves the other operation external | `TestSimulationModelProgressConcurrentOperationsOnOneParticipant` |
| Acknowledging one of two delivered replies leaves Retry; consuming the second permits Advance | `TestSimulationTimeProgressPartialAcknowledgementKeepsOtherDeliveryRunnable` |
| Unknown admission acknowledgements fail before retaining an exploration record or advancing time | `TestSimulationTimeProgressUnknownAcknowledgementBeforeAdmission` |
| Unknown forwarding and arrival-transfer acknowledgements retain their error classification and leave both participants able to reach the next epoch | `TestSimulationTimeProgressUnknownAcknowledgementBeforeForwardAndTransfer` |
| Removal errors an installed waiter; a new incarnation activates at current time; stale arrival and terminal frames fail before terminal mutation | `TestSimulationTimeProgressDeathAndRestartRejectStaleIncarnation` |
| Cancellation after commitment returns Canceled; removal leaves the coordinator's delivered credit runnable until the known late reply is discarded; the committed model remains unchanged | `TestSimulationModelProgressCancellationKeepsCommitAndDiscardsLateArrival` |
| Unknown, duplicate and already discarded duplicate responses fail before arrival or discard callbacks | `TestSimulationModelProgressRejectsUnknownAndDuplicateResponsesBeforeCallbacks` |
| A known abandoned reply with too many acknowledged arrivals fails without clearing the runnable delivery | `TestSimulationModelProgressUnknownDiscardAcknowledgementKeepsDeliveryRunnable` |

Existing `simulation_time_test.go` tests additionally pin earliest-deadline
arbitration, strict versus forward epochs, acknowledgement-before-forwarding,
atomic transfer, delivered-work exclusion and restart activation. New tests
assert no private counter values or response-map contents. The installed-waiter
fixture uses the context `Done()` evaluation after waiter installation as a
channel handshake, with bounded cancellation and goroutine cleanup. It does not
use sleeps or a negative polling interval.

Two explicit historical tests expose existing violations of R11's required
validation-before-mutation contract. They pass against the unchanged source
because task 15 characterizes existing behavior.

| Gap | Source mechanism | Task 16 correction |
| --- | --- | --- |
| `TestSimulationTimeProgressPreexistingMalformedWaitWakesQuiescence` returns Retry to an installed waiter even though unknown arrivals reject acceptance | `handleWaitAcceptance` calls `runnable` before `acknowledgeExternal` | Validate the acknowledgement and response identity before waking or reserving work; a strengthened negative test must leave the waiter installed |
| `TestSimulationTimeProgressPreexistingDuplicateAdmissionConsumesArrival` leaves External after rejecting a duplicate request that consumed a delivered credit | `beginResponseBarrier` admits work and consumes arrivals before checking the response map; `endExternal` cannot restore the consumed credit | Validate duplication and arrivals together before mutation; a strengthened negative test must still see Retry for the unconsumed reply |

Task 16 keeps all valid-path test bodies in `simulation_progress_test.go`
byte-identical. It may adapt migration-sensitive callback wiring in
`simulation_progress_fixture_test.go` when it removes the old accounting
helpers. It must strengthen the two named `Preexisting` negative tests,
demonstrate their failure against the old production source, and record this
scoped correction. Keeping their incorrect behavior would conflict with R11.
Production compatibility wrappers for removed accounting methods would retain
the sequencing burden and are excluded from the selected design.

## Alternative A. Operation handles

A concrete handle design could expose these private operations.

```go
progress.Accept(participant, arrivals, responseID) (*operation, error)
progress.Forward(source, arrivals, current, destination) (*operation, error)
operation.Dispatch() error
operation.Arrive(arrivals uint32) error
operation.Discard(arrivals uint32) error
operation.SuspendWait() error
operation.ReserveCompletion() error
progress.Remove(participant) error
```

The owner would keep each operation's source, destination, response phase and
terminal disposition. Handles could ensure only one terminal method succeeds
and could reject old participant pointers before applying an operation. Accept
and Forward would consume aggregate arrival credits under the arbiter mutex
only after validating both identities and the response reservation. Arrive
would acknowledge the source and deliver to the destination under the same
mutex. Multiple handles would represent simultaneous operations.

Callers would still retain and pass the right handle through request handling,
response callbacks, suspended waits, process completion and transport
cancellation. The existing coordinator response map could move into the owner,
but a request-to-handle index would still be needed wherever a later callback
has only a frame. Transport acknowledgements carry an aggregate arrival count,
not the identity of a consumed operation. The owner therefore needs a separate
aggregate credit ledger even with live handles; it cannot safely mark a
particular handle consumed from that count or assume FIFO consumption.

The 25 current accounting sites could call handle methods instead of counter
methods, but separate Dispatch, terminal-disposition and wait-reservation calls
would preserve several caller sequences. Complete Accept/Forward/Arrive
operations could reduce this, at the cost of additional handle retention and
correlation. The method ordering itself becomes another lifecycle the callers
must understand. Duplicate responses still need transport-level rejection
before a handle callback. Host arrival order cannot define handle identity or
semantic replay order.

## Alternative B. Closed typed semantic transitions

The selected design keeps progress and coordinator response phases in one
private owner with this shape. Names illustrate responsibilities rather than a
new public API or wire format.

```go
type simulationProgressEvent interface { simulationProgressEvent() }

func (progress *simulationProgress) apply(event simulationProgressEvent) error

type coordinatorRequestAccepted struct { /* participant, responseID, arrivals */ }
type coordinatorControlForwarded struct { /* participant, node, responseID, arrivals */ }
type coordinatorWaitAccepted struct { /* participant, responseID, arrivals */ }
type coordinatorWaitSuspended struct { /* participant, responseID */ }
type coordinatorCompletionReserved struct { /* participant, responseID */ }
type coordinatorResponseDelivered struct { /* participant, responseID */ }
type participantRequestAccepted struct { /* participant, arrivals */ }
type participantResponseDelivered struct { /* participant */ }
type modelRequestForwarded struct { /* source, destination, arrivals, current */ }
type modelRequestDispatched struct { /* coordinator */ }
type modelDispatchUnavailable struct { /* coordinator */ }
type modelResponseArrived struct { /* coordinator, node, arrivals */ }
type modelAbandonedResponseDiscarded struct { /* coordinator, arrivals */ }
type arrivalCreditsConsumed struct { /* participant, arrivals */ }
type participantRemoved struct { /* participant */ }
```

These are closed variants for actual protocol boundaries. Each type carries
only its required inputs; a generic bag of optional participants, booleans,
phase values and counters is excluded. The implementation may use fewer types
where two boundaries truly have identical complete semantics. It must not
relabel the eight accounting methods as eight events.

`apply` holds the arbiter mutex for validation, acknowledgement consumption,
work reservation, response phase change, waiter wake and settlement. A normal
coordinator admission validates both the aggregate acknowledgement and response
identity before reserving handling work. Wait acceptance additionally changes
the waiter state in that same transaction. Wait suspension releases the
reservation and moves its response phase together. Completion reservation and
response delivery consult that phase and perform all required accounting
together, including delivery from a suspended-wait phase. No caller chooses
which counter updates are needed for the phase.

Model forwarding validates both participant pointers, the reported epoch and
source arrivals before consuming source credits and reserving source external
work plus destination handling work. Model arrival validates the active
destination and acknowledgement before atomically consuming coordinator
credits and making the destination runnable. Known abandoned replies consume
only the legitimate coordinator credits through the discard transition. An
unknown or duplicate response never reaches this owner because transport
correlation rejects it first. An invalid discard acknowledgement remains an
error and preserves the delivered work.

The owner moves the existing coordinator response phases inside its private
state. Counted participant work continues to represent simultaneous operations;
the design adds no per-model-operation token registry or caller-owned credit
map. The existing response request IDs remain private correlation in their
current coordinator request namespace. Model transport has its own pending
and abandoned request namespace. The owner must not join those namespaces by
assuming numeric request IDs are globally unique.

Native quiescence remains the arbiter's existing blocking operation. It uses
the owner's private validation and credit state while holding the same mutex.
Registration, activation, monotone epochs and generation validation keep their
existing semantics. Removal validates the current participant pointer,
completes its waiter with an error, deletes its progress state and settles in
one transaction. A later old-incarnation callback cannot mutate a replacement.

## Caller migration

The task text cites 24 accounting sites in `simulation_unix.go` plus three in
`process_unix.go`. The current source has 22 plus three. The following mapping
covers all 25 extant sites; the historical 27-site figure is not a measured
reduction claim. Three runnable calls and three participant-removal calls are
additional lifecycle boundaries.

| Current caller boundary | Accounting sites | Complete owner responsibility |
| --- | --- | --- |
| `handleCoordinatorDelivery` | 3 | Deliver the reserved response, handling its private wait phase without caller begin/deliver sequencing |
| `handleWaitAcceptance` | 3 | Validate arrivals and duplicate reservation, consume credits, wake the waiter and reserve the wait response atomically; remove rollback |
| `handleModelArrival` | 1 | Validate and transfer source credit to the active node atomically |
| Node `SimulationCapability` delivery/arrival callbacks | 2 | Deliver participant response and consume aggregate credits through the owner |
| `handleNodeFrame` admission and unavailable-model branch | 3 | Admit a participant request or forward model work; terminate coordinator handling when model dispatch is unavailable |
| `stop` graceful/crash delivery | 2 | Deliver the participant control response through the owner |
| `retainCompletionUntilResponse` | 1 | Reserve completion once according to the private response phase |
| `beginResponseBarrier` | 2 | Admit and reserve a coordinator response as one event; remove admit/rollback sequencing |
| `beginForwardedResponseBarrier` | 3 | Validate and reserve source/destination response work atomically; remove both rollback updates |
| `releaseWaitResponseBarrier` / `reserveWaitResponseBarrier` | 2 | Suspend or resume the private response reservation and accounting in one transaction |
| `process_unix.go` arrival/model dispatch/discard callbacks | 3 | Consume arrivals, mark dispatch or discard legitimate late credits at the actual transport boundary |

Task 16 removes the eight external accounting methods,
`simulationResponseBarrier`, `coordinator.responses` and their begin/release/
reserve helpers. It may keep coordinator transport callback names where those
names are the actual stable boundary, with one complete typed transition as
their implementation. The unavailable-model terminal event is necessary to
release its handling reservation, not an optional cleanup extension.
Participant runnable notifications and removal also route into the owner;
no second progress bookkeeping structure lives in the coordinator.

## Ownership, invalid inputs and reversal criterion

The transport retains bounded pending/abandoned maps, pipe serialization,
response correlation and cancellation. The process supervisor retains process
lifetime, hard crash/reap and completion synchronization. Domain handlers retain
model mutation and classify an operation as committed or uncommitted before
answering. A cancelled committed operation keeps its mutation exactly once,
uses the late discard path and never delivers a cancelled node arrival.
The progress owner accounts for that disposition; it does not redo or undo
domain work. Runtime native timers and generated time-wire codecs stay with
their delivered owners.

Every transition validates all referenced participants, acknowledgement bounds,
response phase/duplication and applicable epochs before any state change. The
same input that fails once must fail again without consuming credits or waking
another waiter. The transition holds one mutex across transfer; it never
settles in the interval between source release and destination delivery.
The owner must preserve existing valid error precedence, including the
unknown-acknowledgement classification characterized by the tests.

Request IDs, pipe completion order, participant map iteration and callback
arrival order remain correlation and implementation details. Semantic replay
continues to validate the independently owned World/network/volume/fault/runtime
transitions. Neither design adds host arrival order to replay identity.

Choose typed transitions because the present protocol supplies aggregate
credits and framed callbacks, so complete semantic events fit its boundaries
without an operation-handle registry. Revisit handles only if a future wire
contract explicitly names each operation's acknowledgement and terminal
disposition, or if the typed owner cannot implement a listed boundary without
caller-side phase/counter sequencing. In the latter case, reconsider the
boundary before adding generic flags or a second bookkeeping map. Source
tests alone cannot decide native timing; task 16 still needs real process
conformance on the supported platforms and unchanged D12/D14 dispositions.
