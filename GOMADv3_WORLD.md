# Gomad World event model plan

## Decision summary

Implement World first as a pure-Go, in-memory module under
`tools/gomadv3/world`. World owns stable request and event identities, a bounded
priority queue, logical external-event time, lifecycle validation, cancellation,
atomic same-instant delivery, snapshots, and external-event replay validation.
It accepts data and returns data: it starts no goroutines, invokes no callbacks,
reads no environment or host clock, and performs no filesystem, network, DNS,
process, signal, or other host I/O.

The external interface is deliberately small. Adapters register a request, mark
its result ready, cancel it, or declare that their deterministic region is
quiescent. A quiescence call either returns every delivery at the next logical
instant in deterministic order, reports stable World deadlock, or reports idle.
All mutations are serialized under one lock, and returned values are detached
copies, so callbacks and adapter wakeups occur outside World.

Queue order is logical time, semantic priority, canonical resource identity,
request and event kinds, equivalence class, and request registration sequence.
Only events that an adapter explicitly places in the same nonempty equivalence
class may replace FIFO with a seed-derived choice rank before the final
registration-sequence fallback. The rank is a stateless, domain-separated
HMAC-SHA-256 value derived from the configured seed and stable event identity.
World never calls a runtime random function or consumes a stateful random
stream.

Do not add a runtime hook in the initial implementation. The native runtime
timer heaps documented by `GOMADv3_CLOCK.md` remain the only queue for standard
Go time. World owns external events only and may advance them only after an
explicit quiescence declaration. A native-timer coordination hook requires a
concrete adapter pilot and a minimized failure proving that adapter or Runner
coordination cannot meet the contract.

## Goals

- Provide one deep World module that every future external adapter can use for
  request lifecycle and event ordering.
- Make registration, readiness, cancellation, delivery, quiescence, snapshot,
  restore, and replay-divergence behavior explicit and testable through the
  public interface.
- Reproduce the same external-event sequence for the same World schema, initial
  snapshot, seed, and sequence of calls across fresh processes.
- Keep request IDs, event IDs, logical timestamps, priorities, resource IDs,
  payloads, state transitions, and errors bounded and serializable.
- Permit seed-based exploration only where the adapter has asserted semantic
  equivalence; ordinary FIFO and resource ordering must not vary by seed.
- Distinguish idle, deterministic World deadlock, capacity rejection, invalid
  input, replay divergence, native runtime deadlock, and Runner wall timeout.
- Make snapshots safe after any completed public call and reject corrupt,
  incompatible, or noncanonical snapshots without partially restoring state.
- Keep the initial implementation in the Go standard library and independent
  of the custom Go runtime so it can be tested with an ordinary Go toolchain.

## Non-goals

- Virtualizing native `time.Timer`, `time.Sleep`, tickers, or context deadlines.
  Those continue to use the runtime timer heaps from `GOMADv3_CLOCK.md`.
- Inferring runtime quiescence from wall time, polling, host netpoll, goroutine
  IDs, or application inactivity.
- Making arbitrary unmodified host filesystem, socket, DNS, database, process,
  signal, cgo, or foreign-thread behavior deterministic.
- Defining filesystem, network, persistence, process, environment, or entropy
  semantics before a selected adapter needs them.
- Calling adapter code, sending on adapter channels, or waking goroutines while
  the World lock is held.
- Runtime-choice replay, cross-version replay, or compatibility across changed
  programs, toolchains, architectures, World schemas, or adapter schemas.
- Using World's external-choice derivation for runtime scheduling, native
  timers, map iteration, application entropy, cryptographic entropy, or fault
  domains that have not been separately named and versioned.
- Multi-P deterministic parallelism or production execution.

## Module and seam

World is an in-process deep module. Its external seam is the concrete `World`
type and its data-only methods. There is no adapter interface in the first
phase: one implementation would make that seam hypothetical, and adapters need
not be callbacks into the queue. Each adapter owns its domain rules and invokes
the same World interface.

The deletion test explains the module's depth. Removing World would force every
adapter to duplicate stable identity allocation, queue bounds, ordering,
equivalence validation, cancellation races, quiescence outcomes, snapshot
canonicalization, and replay divergence. Keeping those behaviors behind a
small interface gives adapters leverage and keeps fixes local.

The World event core initially snapshots event-model state. Future adapter
snapshots are composed alongside it in the Runner record:

```text
record world snapshot
  |
  +-- event core snapshot          tools/gomadv3/world
  +-- filesystem snapshot          future filesystem adapter
  +-- network snapshot             future network adapter
  +-- process/environment state    future adapters
```

Adapter state is not accepted as an opaque callback or `any` value by the event
core. Each adapter defines a versioned, data-only snapshot and the record layer
bundles the parts. This keeps the core serializable without turning it into a
shallow generic state registry.

## Pattern survey

### Analogous features

- `GOMADv3_NEXT.md:31` organizes post-v3 work around Runner, World, Adapters,
  and Record as deep modules and makes World the owner of deterministic
  external state and ordered events.
- `GOMADv3_NEXT.md:70` requires logical timestamps, stable resource identity,
  deterministic tie-breaking, explicit interest registration, cancellation,
  readiness, delivery, bounded queues, and explicit quiescence.
- `GOMADv3_CLOCK.md:101` defines runtime-proven quiescence and direct advancement
  to the next native timer. World follows the same jump-to-next-instant model
  but does not claim the runtime proof in its standalone interface.
- `GOMADv3_CLOCK.md:662` requires native timers to remain in their runtime heaps
  and postpones cross-domain coordination until a concrete adapter establishes
  the need.
- `common/clock/event_time_source.go:126` has an `AdvanceNext` operation that
  finds the earliest future timer. It is useful semantic prior art, but its
  linear scan and timer-only interface are not sufficient for a bounded,
  cancelable external-event queue.
- `common/clock/event_time_source.go:157` fires callbacks while processing fake
  timers, and its comment at line 68 documents the resulting reentrancy and
  deadlock risk. World instead returns delivery data after releasing its lock.
- `common/collection/priority_queue.go:14` wraps `container/heap` behind a
  comparator, and `tools/fairsim/sim.go:398` orders heap entries by successive
  semantic fields with an insertion index as the stable final tie-break.
- `tools/gomadv2/internal/simulation/network/delayqueue.go:15` orders by time
  and index, but `delayQueue` at line 44 uses host timers, `time.Now`, and a
  condition variable. World retains the useful time/index ordering pattern and
  rejects the host-readiness mechanism.
- `tools/gomadv2/internal/simulation/simulation.go:37` places network,
  filesystem, machine, timeout, and waiter state in one simulation object. It
  demonstrates the state that later adapters must model, while its runtime,
  syscall, and background-network integration is broader than the v3 seam.

### Reusable utilities and convention anchors

- `tools/gomadv3/go.mod:1` is a separate Go 1.26.4 module with no declared
  dependencies. The World core should use `container/heap`, `crypto/hmac`,
  `crypto/sha256`, `encoding/binary`, `encoding/json`, `errors`, `fmt`, `sort`,
  `sync`, and `unicode/utf8` from the standard library only.
- `tools/gomadv3/overlay/src/runtime/gomad.go:13` parses `GOMADSEED` before user
  code, and line 30 retains the parsed `uint64`. Runner must pass that same
  numeric seed into World configuration; World must not link to this runtime
  variable.
- `tools/gomadv3/overlay/src/runtime/gomad.go:11` fixes the process clock at
  `946684800000000000`. World uses the same Unix-nanosecond initial logical
  instant so a later evidence-backed coordinator can compare timestamps without
  conversion.
- `tools/gomadv3/testlib.sh:3` already provides bounded child execution,
  process-group termination, status capture, and bounded diagnostics. Runner
  work should retain that failure classification rather than put watchdogs in
  World.
- `tools/gomadv3/README.md:52` states the reproducibility unit: fixed toolchain,
  architecture, program, deterministic inputs, and seed. A World schema and
  adapter snapshot identity become additional deterministic inputs, not a
  broader compatibility promise.
- `tools/gomadv3/README.md:58` excludes host-dependent readiness and identifies
  deterministic mode as trusted-test-only. World resource names confer no host
  capability and adapters must deny ambient host access.

### Proposed alignment

Use a local `container/heap` implementation because the nested
`tools/gomadv3` module cannot reuse `common/collection` without adding a module
dependency. Mirror the repository's lexicographic heap comparisons and stable
insertion sequence, but add indexed removal for cancellation. Mirror the
clock's direct advancement and same-instant eligibility while keeping the
quiescence assertion explicit. Retain v3's process seed as an input but derive
World choices with a named, stateless domain so runtime PRNG consumption cannot
change external event order.

## Alternatives

### A. Pure-Go explicit World core — recommended

Adapters invoke a concrete in-memory World and explicitly declare readiness,
cancellation, and quiescence.

Advantages:

- Independently unit-testable with the stock toolchain and race detector.
- Small runtime patch remains unchanged.
- One queue and lifecycle contract serves every adapter.
- Snapshots and replay can validate data before adapter integration exists.
- No host callback order, wall clock, runtime-private identity, or random stream
  enters the ordering rules.

Costs and risks:

- Standalone World cannot prove that the Go runtime has no runnable goroutine.
- Transparent adapter delivery must wait for an explicit driver or later
  evidence-backed coordination.
- Terminal request/event records consume bounded memory for the World lifetime
  so replay identities are never silently reused.

### B. Add a runtime external-event queue immediately

Patch `checkdead` and runtime scheduling now so World events participate beside
native timers.

Advantages:

- Could provide transparent runtime quiescence from the first adapter.
- Could compare native and external deadlines at the existing proof point.

Costs and risks:

- No adapter has yet demonstrated the required hook shape.
- Pulls serializable external payloads, cancellation, and replay policy into a
  runtime that should only schedule Go work.
- Broadens pinned-Go maintenance and risks a second native timer queue.
- Couples external choice randomness to runtime implementation details.

Reject initially. Reconsider only under the evidence gate below.

### C. Give each adapter its own queue and scheduler

Filesystem, network, and process adapters each order their own completions.

Advantages:

- Each adapter can encode its semantics locally.
- Small first adapter implementation.

Costs and risks:

- Cross-adapter ordering becomes unspecified.
- Capacity, cancellation, snapshots, and replay diverge by adapter.
- Domain-separated choices and quiescence are duplicated at every call site.
- Native-clock coordination would need to reconcile several queues.

Reject because it destroys World locality and makes same-time races dependent
on adapter call order.

### D. Record host completion order and replay it

Run real I/O once, record callback order and timestamps, then force that order
during replay.

Advantages:

- Lower simulation cost for a narrow happy path.
- Can reproduce one observed host run when the environment remains compatible.

Costs and risks:

- The first run is not deterministic and cannot systematically explore choices.
- Host timestamps, readiness, credentials, and machine-specific behavior leak
  into artifacts.
- Cancellation races and unseen alternatives are not modeled.
- A replay mismatch appears late and may be impossible to minimize.

Reject inside the deterministic region. Host import may only create an explicit
initial adapter snapshot before the run.

### E. Reuse `common/clock.EventTimeSource`

Advantages:

- Existing fake-time behavior and locking tests.
- Synchronous advancement is familiar to Temporal code.

Costs and risks:

- It is in the root module, uses callbacks, and does not model request identity,
  resource order, queue capacity, snapshots, or replay.
- It would introduce a second timer queue despite native time already being
  solved.

Retain it for existing Temporal unit tests, not as the World event core.

| Option | Pure Go | Stable external identity | Cross-adapter order | Snapshot/replay | Initial runtime change |
| --- | --- | --- | --- | --- | --- |
| A. Explicit World | Yes | Yes | One queue | Native design concern | None |
| B. Runtime queue | Partly | Runtime-coupled | One queue | Harder to version | Broad |
| C. Per-adapter queues | Yes | Adapter-specific | No | Fragmented | None initially |
| D. Host record/replay | No on first run | Host-derived | Observed only | Replay-only | Adapter hooks |
| E. EventTimeSource | Yes | No | Timer-only | No | None |

## Public interface

The first implementation should expose a concrete type rather than an
interface with one adapter. Names below are the implementation contract; tests
exercise this same seam.

```go
package world

const (
	SchemaVersion uint32      = 1
	InitialTime   LogicalTime = 946684800000000000
)

type LogicalTime int64
type Seed uint64
type RequestID uint64
type EventID uint64
type Sequence uint64
type Priority uint16
type Digest string

type Config struct {
	Seed   Seed   `json:"seed"`
	Limits Limits `json:"limits"`
}

type Limits struct {
	MaxRequests     uint64 `json:"maxRequests"`
	MaxEvents       uint64 `json:"maxEvents"`
	MaxQueuedEvents uint64 `json:"maxQueuedEvents"`
	MaxTransitions  uint64 `json:"maxTransitions"`
	MaxPayloadBytes uint64 `json:"maxPayloadBytes"`
	MaxStringBytes  uint32 `json:"maxStringBytes"`
}

type ResourceID struct {
	Adapter string `json:"adapter"`
	Kind    string `json:"kind"`
	Key     string `json:"key"`
}

type Request struct {
	Kind     string     `json:"kind"`
	Resource ResourceID `json:"resource"`
	Priority Priority   `json:"priority"`
	Payload  []byte     `json:"payload,omitempty"`
}

type Readiness struct {
	RequestID       RequestID   `json:"requestId"`
	At              LogicalTime `json:"at"`
	Kind            string      `json:"kind"`
	Payload         []byte      `json:"payload,omitempty"`
	EquivalenceClass string     `json:"equivalenceClass,omitempty"`
}

type CancelStatus string

const (
	CancelWon              CancelStatus = "won"
	CancelAlreadyCanceled  CancelStatus = "already-canceled"
	CancelAlreadyDelivered CancelStatus = "already-delivered"
)

type Cancellation struct {
	RequestID RequestID    `json:"requestId"`
	EventID   EventID      `json:"eventId,omitempty"`
	Status    CancelStatus `json:"status"`
}

type Delivery struct {
	RequestID RequestID   `json:"requestId"`
	EventID   EventID     `json:"eventId"`
	At        LogicalTime `json:"at"`
	Kind      string      `json:"kind"`
	Payload   []byte      `json:"payload,omitempty"`
}

type QuiescenceKind string

const (
	QuiescenceDelivered QuiescenceKind = "delivered"
	QuiescenceDeadlock  QuiescenceKind = "deadlock"
	QuiescenceIdle      QuiescenceKind = "idle"
)

type Quiescence struct {
	Kind       QuiescenceKind `json:"kind"`
	Before     LogicalTime    `json:"before"`
	After      LogicalTime    `json:"after"`
	Deliveries []Delivery     `json:"deliveries,omitempty"`
	Blocked    []RequestID    `json:"blocked,omitempty"`
}

type ReplayProgress struct {
	Cursor   uint64 `json:"cursor"`
	Expected uint64 `json:"expected"`
}

func New(config Config) (*World, error)
func Restore(snapshot Snapshot, replay *ReplayPlan) (*World, error)

func (w *World) Register(request Request) (RequestID, error)
func (w *World) Cancel(requestID RequestID) (Cancellation, error)
func (w *World) Ready(readiness Readiness) (EventID, error)
func (w *World) Quiesce() (Quiescence, error)
func (w *World) Snapshot() Snapshot
func (w *World) ReplayProgress() ReplayProgress
```

`New` requires every limit to be nonzero and starts at `InitialTime` with ID and
transition sequences beginning at one. `Restore` validates the entire snapshot
and optional replay plan before publishing a `World`; it never returns a
partially usable value. Passing `nil` disables replay checking.

All byte slices are copied on input and output. All methods are safe for
concurrent callers. They hold one mutex only while validating and changing
state, invoke no foreign code, and return detached data. The order in which
concurrent calls acquire that mutex is part of the program's runtime-controlled
schedule; for an unchanged Gomad program and seed it is repeatable.

## Serializable types and canonical encoding

Seed, public IDs, and sequences use `uint64` internally. Their JSON marshal
methods emit canonical base-10 strings and reject signs, whitespace, leading
zeros, overflow, and numeric JSON tokens on input. Seed zero is valid; identity
and sequence zero values are not. This avoids precision loss in record tooling
that represents JSON numbers as IEEE-754 values. Logical time uses the same
canonical decimal-string encoding and must be nonnegative.

`Priority` is a numeric semantic field where smaller values deliver first. The
core does not invent meanings for priority values. Each adapter publishes its
own named constants and records the numeric value; changing their meaning is an
adapter schema change.

`ResourceID.Adapter` and `ResourceID.Kind` must match
`[a-z][a-z0-9._-]{0,63}`. `ResourceID.Key`, request kind, event kind, and
equivalence class must be valid UTF-8 and fit `MaxStringBytes`; kinds must be
nonempty. The core compares exact UTF-8 bytes and performs no Unicode, path,
hostname, or case normalization. The adapter must supply one canonical key for
each logical resource before registration.

Payloads are bytes, not `any`, maps, functions, interfaces, pointers, or
channels. JSON represents them with standard base64. `MaxPayloadBytes` counts
all retained request, event, delivery-history, and replay payload copies, so an
accepted operation cannot make memory use silently unbounded.

Serializable structures contain only scalars, structs, and canonically sorted
slices. Maps are internal indexes and never appear in an artifact. A private
length-prefixed binary codec is the source for snapshot and transition SHA-256
digests; JSON field order or encoder behavior is never hashed. `Digest` is
exactly 64 lowercase hexadecimal characters. The digest detects corruption and
divergence; it is not an authenticity or secrecy mechanism.

## Identity and lifecycle contract

Request and event IDs are allocated monotonically and are never reused within
a World. Zero is invalid. `RequestID` is also the request registration sequence;
`EventID` is the readiness sequence. Exhausting either `uint64` is a stable
capacity error before mutation.

```text
Register
   |
   v
pending -------------------------> canceled
   |                                  ^
   | Ready                            | Cancel wins
   v                                  |
queued ----------------------------->+
   |
   | Quiesce atomically selects the next instant
   v
delivered
```

- `Register` validates and copies the request, reserves request and transcript
  capacity, assigns the next request ID, and leaves the request pending.
- `Ready` accepts only a pending request, requires `At >= World.now`, validates
  and copies the event, reserves event/queue/transcript capacity, assigns the
  next event ID, and inserts it into the heap. A request has at most one event.
- `Cancel` on pending or queued state wins. A queued event is removed by heap
  index in `O(log q)`. Repeated cancellation returns
  `CancelAlreadyCanceled`; cancellation after delivery returns
  `CancelAlreadyDelivered`. These idempotent outcomes are not errors.
- `Quiesce` atomically delivers a whole next-instant batch. Once its lock is
  acquired, cancellation cannot split the batch: cancellation either removes
  an event before the batch or observes it delivered afterward.
- Terminal request and event metadata are retained until the World ends. This
  makes duplicate operations and replay results stable. Lifetime limits bound
  the retained tombstones.
- Invalid request IDs, readiness for a terminal request, duplicate readiness,
  time regression, and malformed data return typed errors and leave IDs,
  counters, queue state, replay cursor, and transcript unchanged.

The transition transcript records successful registration/readiness mutations,
all cancellation outcomes, and every quiescence outcome. A transcript-capacity
error is terminal for further modeled operations and is reported separately by
Runner because it cannot safely append a record of itself.

## Ordering and genuinely equivalent events

The heap compares one static tuple, which makes the comparator total and
transitive for every mixture of ordinary and explicitly equivalent events:

1. `Readiness.At`, ascending;
2. `Request.Priority`, ascending;
3. `Request.Resource.Adapter`, then `Kind`, then `Key`, bytewise ascending;
4. request kind, then event kind, bytewise ascending;
5. equivalence class, bytewise ascending, with empty first;
6. registration sequence for an empty class, or seed-derived choice rank for a
   nonempty class; and
7. registration sequence as the final fallback.

Pointers, goroutine IDs, heap indexes, map iteration, host timestamps, callback
arrival order, and OS resource numbers are never compared.

An adapter may set a nonempty `EquivalenceClass` only when exchanging the order
of events with the same preceding tuple fields cannot change adapter semantics
except for the intentional choice being explored. Empty-class events retain
registration order. Different nonempty classes have stable bytewise order;
only members of one identical nonempty class compare choice rank:

```text
time, priority, resource, kinds, equivalence class
              |
              +-- empty class -----> registration sequence
              |
              +-- nonempty class --> choice rank, registration sequence
```

Registration sequence remains the total-order fallback for hash collisions.
An adapter must not label distinguishable network messages, file operations,
faults, recipients, or payload results equivalent merely to obtain more seed
variation. Such a declaration is an adapter correctness bug and must be caught
by adapter semantic tests.

Choice rank is stateless and versioned:

```text
key  = SHA-256("gomadv3/world/seed/v1\x00" || seed-as-big-endian-uint64)
rank = HMAC-SHA-256(
         key,
         "gomadv3/world/equivalent-event-order/v1\x00" ||
         length-prefixed-equivalence-class ||
         event-id-as-big-endian-uint64,
       )
```

Using a named domain prevents later external fault or entropy choices from
sharing state. Ranking does not consume a stream, but event IDs remain part of
the unchanged-program reproducibility contract: source or request-order changes
may change later ranks. The runtime seed is only an input value; World imports
no runtime symbol and does not use `math/rand`, `crypto/rand`, `hash/maphash`,
or the runtime's private PRNG.

## Quiescence, delivery, and deadlock

Calling `Quiesce` is an assertion by the caller that no work in the claimed
deterministic region can run until World returns. The pure-Go core validates its
own state but cannot prove this runtime fact. An adapter or test driver that
calls it while application work is runnable is outside the deterministic
contract.

Under the World lock, `Quiesce` behaves as follows:

```text
queued event exists?
  |
  +-- yes --> jump now to earliest event time
  |           pop every event at that instant in queue order
  |           mark the full batch delivered
  |           return QuiescenceDelivered
  |
  +-- no --> pending request exists?
              |
              +-- yes --> return QuiescenceDeadlock + sorted blocked IDs
              |
              +-- no  --> return QuiescenceIdle
```

Every event at the selected instant becomes delivered in one atomic transition
before adapter wakeups occur. The returned slice preserves queue order, but the
adapter performs the actual channel send, future resolution, or other wakeup
after World returns. A new event registered at the current instant after this
call is causally later and belongs to the next quiescence batch.

World deadlock means at least one live pending request and no queued event. It
does not mean native runtime deadlock: an adapter may have failed to mark a
completion ready, or unsupported runnable/host work may still exist. Runner
records the classifications separately:

- World idle: no pending or queued external operation;
- World deadlock: pending external requests and no future external event;
- native deadlock: runtime proves no runnable goroutine and no native timer;
- wall timeout: runnable spin, unsupported blocking host operation, crashed
  coordination, or another failure that prevents logical progress.

Quiescence outcomes do not permanently close World. This permits standalone
tests to inspect a deadlock and then diagnose the missing readiness, while
Runner normally stops a deterministic run at the first deadlock result.

## Internal state and queue

The implementation uses one mutex and the following private shape:

```go
type World struct {
	mu sync.Mutex

	config         Config
	now            LogicalTime
	nextRequestID  RequestID
	nextEventID    EventID
	nextTransition Sequence
	payloadBytes   uint64

	requests map[RequestID]*requestState
	events   map[EventID]*eventState
	queue    eventHeap
	history  []Transition
	replay   *replayState
}

type requestState struct {
	request Request
	state   RequestState
	eventID EventID
}

type eventState struct {
	readiness   Readiness
	request     *requestState
	heapIndex   int
	choiceRank  [sha256.Size]byte
}

type eventHeap []*eventState
```

`eventHeap.Swap` updates both indexes. `heap.Remove` supports queued
cancellation; `heap.Pop` clears the removed slot and sets its index to `-1` so
terminal payloads are not accidentally retained through the backing array.
Every comparison is total and side-effect free.

Mutation helpers follow plan-then-apply discipline. They validate input,
capacity, sequence space, and the expected replay transition; build the result
and next rolling transcript digest; then apply all fields together. A
divergence or error therefore consumes no ID, sequence, payload budget, or
replay entry.

World does not spawn a dispatcher goroutine. A caller can use it synchronously,
and concurrency tests can call it from several goroutines without introducing
an internal schedule beyond mutex acquisition.

## Capacity contract

All limits are part of `Config`, snapshot identity, and replay compatibility:

- `MaxRequests` bounds all request records, including terminal requests.
- `MaxEvents` bounds all event records, including delivered or canceled events.
- `MaxQueuedEvents` separately bounds active heap entries.
- `MaxTransitions` bounds transcript and replay work.
- `MaxPayloadBytes` bounds every retained payload copy accounted by World.
- `MaxStringBytes` bounds each externally supplied string.

Capacity is checked with subtraction rather than unchecked addition. An
operation either fits completely or returns a typed `CapacityError` identifying
the exhausted dimension, configured limit, current usage, and requested delta.
It never partially registers, queues, cancels, delivers, appends history, or
advances logical time.

Limits are lifetime bounds, not only concurrent-work bounds. This trades longer
single-process runs for stable tombstones and complete replay. Runner should
start a fresh process and World per seed, and choose limits from the scenario
rather than automatically resizing them. A capacity failure is a deterministic
test result, never permission to fall back to host behavior or drop an event.

## Snapshot and restore contract

The serializable snapshot shape is:

```go
type Snapshot struct {
	SchemaVersion  uint32            `json:"schemaVersion"`
	Config         Config            `json:"config"`
	Now            LogicalTime       `json:"now"`
	NextRequestID  RequestID         `json:"nextRequestId"`
	NextEventID    EventID           `json:"nextEventId"`
	NextTransition Sequence          `json:"nextTransition"`
	PayloadBytes   uint64            `json:"payloadBytes"`
	Requests       []RequestSnapshot `json:"requests"`
	Events         []EventSnapshot   `json:"events"`
	Transitions     []Transition      `json:"transitions"`
	TranscriptDigest Digest           `json:"transcriptDigest"`
	Replay          ReplayProgress    `json:"replay"`
	StateDigest     Digest            `json:"stateDigest"`
}
```

Requests are sorted by request ID, events by event ID, and transitions by
sequence. Heap backing-array order is never serialized. Event snapshots include
queued/terminal state and enough data to recompute ordering and choice rank;
choice rank and heap index are recomputed during restore. Terminal state,
counters, limits, and payload accounting are included.

`StateDigest` covers all semantic World fields, including the transition list
and rolling `TranscriptDigest`, but excludes `Replay` and `StateDigest` itself.
Replay cursor position is validation metadata and does not change World
semantics. Each transition digest hashes the previous transcript digest and the
canonical transition bytes, so normal operations do not rehash the full World.

`Snapshot` takes the lock, deep-copies all data, sorts copies, computes the
canonical digest, and releases the lock. It performs no encoding or I/O.
Runner is responsible for JSON encoding, bounded storage, and atomic file
publication.

`Restore` validates, in order:

1. schema and enum versions;
2. nonzero limits and seed compatibility;
3. canonical ID, event, and transition ordering;
4. unique IDs and legal state transitions;
5. request/event cross-references and queued counts;
6. nonnegative time and no queued event earlier than `Now`;
7. exact payload/string accounting and all capacity limits;
8. next sequences strictly beyond every retained identity; and
9. canonical state digest.

Only then does it allocate internal maps, recompute choice ranks, call
`heap.Init`, attach replay state, and return the World. JSON decoding rejects
unknown fields, and unsupported schema versions are rejected rather than
approximated.

### Crash behavior

A public method is atomic with respect to snapshots: a concurrent snapshot sees
the complete before-state or complete after-state, never a half transition. A
process crash loses mutations after the last snapshot that Runner durably
published. Runner must write a temporary artifact, sync and close it as its
platform contract requires, then atomically rename it; an incomplete artifact
must not be labeled replayable.

Restore never guesses whether an unrecorded external side effect occurred.
Because supported adapters perform no host I/O, recovery begins from the last
complete World plus adapter snapshot. A crash during future adapter state
mutation requires that adapter and event-core snapshots to share one record
generation and digest; mismatched generations are rejected.

## Replay and divergence contract

External-event replay validates the program's interaction with World; it does
not force runtime scheduling choices. A replay plan is exact-version data:

```go
type ReplayPlan struct {
	SchemaVersion uint32       `json:"schemaVersion"`
	InitialDigest Digest       `json:"initialDigest"`
	Transitions   []Transition `json:"transitions"`
	FinalDigest   Digest       `json:"finalDigest"`
}
```

Each `Transition` is a tagged data record with exactly one operation body:

| Kind | Input recorded | Result recorded |
| --- | --- | --- |
| `register` | canonical `Request` | assigned `RequestID` |
| `ready` | canonical `Readiness` | assigned `EventID` |
| `cancel` | `RequestID` | complete `Cancellation` |
| `quiesce` | no payload | complete `Quiescence` batch |

Every transition includes its sequence and previous and resulting rolling
transcript digests. Restore requires the current snapshot state digest to equal
`InitialDigest`. Before applying each call, World compares the actual operation,
computed outcome, and next transcript digest with the expected transition.
Comparison includes payload bytes, IDs, timestamps, ordering, cancellation
result, and blocked IDs.

The first mismatch returns `*ReplayDivergenceError` containing transition
index, expected and actual operation kinds, a stable field path, and expected
and actual digests. Its human-readable error omits raw payloads. The call leaves
World and replay cursor unchanged. A call after the plan is exhausted is an
unexpected-operation divergence. At process exit, Runner requires
`ReplayProgress.Cursor == ReplayProgress.Expected` and the final digest to
match; missing expected operations are divergence even if the process otherwise
exits successfully.

Replay plan validation is bounded by the same string, payload, request, event,
and transition limits before execution. Truncated JSON, duplicate sequences,
unknown transition kinds, inconsistent results, and incompatible initial or
final identities fail before user code. Seed replay without a replay plan
remains supported and does not claim operation-by-operation validation.

## Error handling and failure classification

Package errors support `errors.Is` with stable sentinel categories and typed
details:

- `ErrInvalidConfig`: zero limits, inconsistent bounds, or invalid seed policy;
- `ErrInvalidRequest`: malformed kind/resource/payload data;
- `ErrUnknownRequest`: request ID zero or absent;
- `ErrRequestState`: duplicate readiness or readiness after a terminal state;
- `ErrTimeRegression`: readiness timestamp before World time;
- `ErrCapacity`: queue, lifetime record, payload, string, transition, or
  sequence exhaustion;
- `ErrInvalidSnapshot`: schema, ordering, accounting, cross-reference, state,
  or digest failure; and
- `ErrReplayDivergence`: first incompatible operation or result.

Expected lifecycle outcomes such as repeated cancellation, cancellation after
delivery, World idle, and World deadlock are typed results rather than errors.
Errors use stable category and field names, do not include pointer values or map
formatting, and do not expose full payloads. Internal impossible states panic in
unit tests and should be returned as `ErrInvalidSnapshot` at the untrusted
restore seam; no error path silently drops an event, changes a limit, advances
time, skips replay validation, or uses host behavior.

## Proposed file impact

| File | Purpose |
| --- | --- |
| `tools/gomadv3/world/doc.go` | Package contract and no-host-I/O scope |
| `tools/gomadv3/world/types.go` | Public scalar, request, readiness, delivery, quiescence, and config types |
| `tools/gomadv3/world/world.go` | World construction, lifecycle methods, locking, and capacity accounting |
| `tools/gomadv3/world/queue.go` | Indexed `container/heap` implementation and total ordering |
| `tools/gomadv3/world/choice.go` | Versioned HMAC-SHA-256 seed derivation and equivalent-event ranks |
| `tools/gomadv3/world/snapshot.go` | Canonical snapshot, validation, digest, and restore |
| `tools/gomadv3/world/replay.go` | Transition recording, replay plan validation, and divergence |
| `tools/gomadv3/world/errors.go` | Stable sentinel and typed errors |
| `tools/gomadv3/world/*_test.go` | Interface-level lifecycle, ordering, race, bounds, snapshot, and replay tests |
| `tools/gomadv3/Makefile` | Focused World test target after the package exists |
| `tools/gomadv3/README.md` | Supported World contract only after implementation passes |

`tools/gomadv3/go.mod` should remain dependency-free. No production Temporal
package, Go runtime overlay, Go patch, root Make target, or existing comment is
changed by the initial World core.

## Detailed implementation plan

### Phase 0: freeze canonical fixtures and failure vocabulary

1. Add compile-time examples for every public type and method.
2. Add golden canonical binary encodings and SHA-256 digests for one empty and
   one populated snapshot, including seed zero and maximum `uint64` IDs.
3. Add tests for JSON decimal-string IDs, enum validation, UTF-8 validation,
   resource ordering, and each typed error category.
4. Fix default scenario limits in test helpers; production `New` receives
   explicit limits and has no hidden environment-derived defaults.

The phase is accepted when fixtures are architecture-independent and malformed
input fails without allocating proportional to attacker-declared lengths.

### Phase 1: implement identity, queue, and lifecycle

1. Implement validated constructors, deep-copy helpers, monotonic ID allocation,
   payload accounting, and terminal request state.
2. Implement the indexed heap and the ordinary comparison tuple.
3. Add stateless domain-separated equivalent-event ranking and collision
   fallback.
4. Implement `Register`, `Ready`, and `Cancel` with plan-then-apply atomicity.
5. Implement `Quiesce` as an atomic whole-instant delivery batch with sorted
   blocked IDs for deadlock.
6. Test solely through public World methods except focused heap and canonical
   codec property tests.

The phase is accepted when a pure in-memory test can reproduce registration,
readiness, cancellation, ordering, delivery, idle, and deadlock for a seed.

### Phase 2: implement snapshots and restoration

1. Define version-one snapshot structures without maps or implementation-only
   heap fields.
2. Canonically sort and digest detached copies under the documented accounting
   rules.
3. Validate complete snapshots before allocating a usable World.
4. Rebuild maps, request links, choice ranks, and heap indexes from valid data.
5. Compare continued execution from an original and restored World transition
   by transition and snapshot digest by snapshot digest.

The phase is accepted when snapshots taken in every lifecycle state restore to
the same future deliveries and all single-field corruptions fail clearly.

### Phase 3: implement transition replay and divergence

1. Record successful mutations and modeled outcomes with a rolling transcript
   digest.
2. Validate complete replay plans and initial identity before execution.
3. Precompare each planned transition so divergence never mutates state.
4. Return structured first-divergence details for input, result, ordering,
   exhaustion, early process exit, and final digest mismatch.
5. Re-run recorded successful, cancellation, deadlock, and capacity scenarios
   in fresh processes.

The phase is accepted when replay reproduces an exact event sequence and each
incompatible operation reports the first stable field path.

### Phase 4: harden concurrency, bounds, and scale

1. Exercise concurrent register/ready/cancel/quiesce calls with the stock race
   detector and deterministic barriers rather than sleeps.
2. Prove cancellation-versus-delivery linearization and atomic batch delivery.
3. Exercise every limit at `limit-1`, `limit`, and `limit+1`, including integer
   overflow attempts and failed-operation rollback.
4. Benchmark heap insertion, indexed cancellation, batch delivery, snapshot,
   restore, and replay at baseline and ten-times load.
5. Fuzz public decoders and snapshot validation with bounded inputs.

The phase is accepted when the race detector is clean, memory is bounded by
configuration, and ten-times load follows the scale expectations below.

### Phase 5: pilot one deterministic adapter without runtime changes

Select one narrow adapter whose test driver can explicitly establish
quiescence. The adapter must:

1. canonicalize its resource identities and publish semantic priority values;
2. keep all logical state in serializable data;
3. call only the World public interface for request lifecycle and order;
4. perform wakeups after World returns and never while holding adapter state
   locks needed by callbacks;
5. compose its snapshot with the event-core snapshot under one record identity;
6. prove same-seed repetition, permitted cross-seed equivalent choices,
   cancellation, injected failure, deadlock, snapshot, and replay; and
7. perform no host I/O inside the claimed deterministic region.

This pilot establishes whether explicit coordination is sufficient. It does
not advertise transparent native-timer composition.

### Phase 6: apply the native-timer evidence gate

Add no hook if the pilot succeeds through explicit coordination. If it fails,
first minimize a case showing that the adapter and Runner cannot determine the
next logical instant without runtime quiescence. A proposed hook is reviewable
only if it:

- enters at the existing runtime-proven quiescence point;
- compares the earliest native timer with the earliest external event;
- advances exactly one shared logical instant;
- makes all native and World events at that instant eligible before scheduling
  runnable work;
- leaves native Go timers in native heaps and external events in World;
- lets a runnable goroutine prevent both domains from advancing;
- preserves nested synctest precedence and all clock edge behavior;
- transports no arbitrary payload or adapter policy into the runtime; and
- keeps World choice derivation independent from runtime randomness.

The minimized case, hook interface, pinned-Go maintenance cost, disabled-mode
proof, and new black-box tests require explicit review before implementation.

## Test matrix

| Area | Required cases | Required result |
| --- | --- | --- |
| Construction | seed 0/max, each zero limit, inconsistent limits | valid seeds accepted; invalid config rejected before state publication |
| Registration | one/many resources, max ID, copied payload, malformed names | monotonic stable IDs; no aliasing; canonical errors |
| Readiness | now/future, distinct kinds, past time, duplicate, terminal request | one event per pending request; no regression or partial insert |
| Ordinary order | time, priority, resource, request/event kinds, class, registration sequence | exact lexicographic order independent of seed |
| Equivalent order | same class, empty/different class, hash collision fallback | same seed repeats; bounded seeds vary; only declared class varies |
| Cancellation | pending, queued, repeated, after delivery, unknown ID | idempotent statuses; queued removal is exact and bounded |
| Quiescence | earliest future, many at same instant, causal now event | direct jump; full instant delivered atomically in queue order |
| Idle/deadlock | empty World, terminal-only World, pending without event | stable distinct typed outcomes and sorted blocked IDs |
| Capacity | every dimension below/at/above limit, overflow | exact error dimension; no ID, time, transcript, or cursor change |
| Concurrency | parallel register/ready/cancel; cancel versus quiesce | race-free; each call linearizes; whole batch or cancellation wins |
| Callback safety | adapter callback reenters World after delivery return | no callback under World lock and no World-created goroutine |
| Snapshot | every request state, queued heap, terminal records, replay cursor | canonical sorted data and repeatable digest |
| Restore failure | schema, digest, order, duplicate ID, reference, accounting, limit corruption | full rejection and no partial World |
| Snapshot continuation | original versus restored World | identical subsequent results, IDs, transitions, and digest |
| Replay success | registration, ready, cancel outcomes, batch, deadlock, idle | every transition and final digest match |
| Replay divergence | wrong kind/resource/payload/time/ID/order/result, extra/missing call | first stable field reported; state and cursor unchanged |
| Crash | snapshot before/after each transition; truncated artifact | complete generation restores; incomplete generation rejected |
| Race detector | full World unit suite under stock `-race` | no data race; custom Gomad runtime is not involved |
| Fuzz | JSON IDs, snapshot decoder, canonical codec, replay plan | bounded, panic-free rejection of malformed data |
| Ten-times load | requests, queued events, payload, snapshots, replay | expected logarithmic queue cost and linear bounded storage |
| Host independence | scan imports; run with varied TZ/load/environment | identical results and no host I/O or wall-clock dependency |

Go unit tests in the dependency-free `tools/gomadv3` module use the standard
`testing` package rather than introducing `testify`. They use channels,
barriers, and `sync.WaitGroup`, never `time.Sleep`, for concurrency control.
Every Go test command includes `-tags test_dep` even though the World package
does not initially require the tag.

## Performance, scalability, complexity, and security

### Performance

- Registration and state lookup are expected `O(1)` map operations.
- Ready insertion and queued cancellation are `O(log q)` for `q` queued events.
- Delivering a batch of `k` events is `O(k log q)` with the simple heap; optimize
  only if profiles show batch extraction dominates.
- Snapshot and restore are `O(n log n + b)` for sorting `n` retained records and
  hashing `b` bounded bytes. Normal delivery does not serialize JSON.
- HMAC-SHA-256 costs one fixed digest per explicitly equivalent queued event;
  ordinary events do not consume choice work beyond comparison.

### Scalability and ten-times load

One World is intentionally serialized by one mutex because one Gomad target
starts with one P and cross-adapter total order is required. Parallelize seeds
across Runner child processes, not by sharding one World's order.

At ten times as many queued events, heap work grows from `O(n log n)` to
`O(10n log(10n))`; retained request/event/history memory and snapshot size grow
approximately tenfold until the configured limit. There is no unbounded
goroutine, channel, output, or tombstone growth. If the ten-times scenario
exceeds a configured bound, it fails immediately and reproducibly with
`CapacityError`; Runner may launch a new run with reviewed larger limits but
World never resizes past the recorded contract.

Large seed sets use fresh processes and bounded Runner concurrency. Per-run
World memory depends on scenario limits, not the number of seeds, and records
must be streamed rather than accumulated in World.

### Complexity

The principal complexity is localized in lifecycle validation, total ordering,
canonical encoding, and replay precomparison. Adapters do not learn heap,
choice, or tombstone rules. A concrete World avoids a premature interface and
has no internal callback seam. Focused private tests for heap indexing and the
canonical codec supplement interface tests where corruption cannot be produced
through valid calls.

Schema, equivalence domain, resource normalization, adapter priorities, and
adapter snapshot formats are versioned contracts. A change that can alter an
event sequence increments the responsible schema and rejects old exact replay
rather than silently migrating behavior.

### Security

- World is for trusted tests, but snapshot and replay inputs are still length
  checked before allocation and rejected on overflow or invalid UTF-8.
- Resource IDs are inert names. They never authorize opening a path, dialing an
  address, reading an environment variable, or starting a process.
- Payloads are bounded bytes and are not interpreted, formatted into errors, or
  executed by World.
- Snapshot digests detect accidental corruption, not malicious replacement.
  Runner records provenance and applies filesystem permissions outside World.
- Deterministic choice ranks are not random secrets. They must not be exposed as
  application entropy or cryptographic material.
- World imports no `os`, `net`, `time`, `syscall`, `plugin`, `unsafe`, or cgo
  package and receives no ambient credential or host handle.

## Clock integration boundary

World logical timestamps use absolute Unix nanoseconds and begin at the fixed
Gomad process instant. Nevertheless, standalone `World.Quiesce` advances only
World state. It neither reads nor changes runtime `faketime`, and adapters must
not claim that a native timer and external event have been jointly ordered.

The initial ownership is:

```text
native time.Timer/time.Sleep/context deadlines --> runtime timer heaps
external adapter requests and completions       --> World event queue
wall watchdog and process lifetime              --> Runner
```

There is no World entry for native timers and no runtime entry for external
payloads. Before a coordination hook exists, tests combining future external
events with native timers require an explicit driver that stops application
execution and advances the domains under a documented test protocol. If that
cannot preserve the same-instant contract, the pilot must stop and present the
minimized evidence rather than let either clock advance independently.

## Runner and record integration boundary

Runner must honor these cross-document contracts:

- Parse one canonical `uint64` seed, pass it to the target through the existing
  child-only `GOMADSEED` path, and pass the same value to World configuration.
  It must not copy runtime PRNG state into World.
- Start a fresh process and World per seed. World limits, schema, initial
  snapshot digest, and adapter snapshot identities are deterministic inputs.
- Treat World as the sole owner of snapshot, transition, and semantic-digest
  schemas. Record owns the outer artifact envelope and raw payload hashes;
  neither package imports the other.
- Store the initial composite snapshot, ordered World transitions, final
  snapshot digest, transcript digest, replay progress, and structured World
  error or quiescence result in the artifact.
- During replay, reject toolchain, architecture, program, schema, seed, limits,
  and initial snapshot mismatches before target execution.
- On target exit, require replay exhaustion and final digest equality. Report
  first World divergence separately from target failure.
- Keep World idle, World deadlock, native deadlock, capacity rejection, replay
  divergence, logical test timeout, wall-watchdog timeout, target exit/signal,
  and Runner/host failure as distinct result classes.
- Write composite snapshots and records atomically. A crash may publish bounded
  partial diagnostics but never a complete/replayable marker for an incomplete
  generation.
- Drain and bound stdout/stderr as nonsemantic diagnostics. Output timing must
  not drive World calls or ordering.
- Enforce process-tree wall time, memory, artifact, and child-concurrency limits
  outside World. A ten-times seed set grows total work approximately linearly
  without changing per-run World bounds.
- Never import host state into a live deterministic region. Files, sockets,
  environment, and other state enter only through a versioned initial adapter
  snapshot prepared before target activation.

## Verification commands

Run the pure-Go tests with the ordinary toolchain first. Every Go test includes
`test_dep`:

```sh
env -u GOMADSEED GOWORK=off go -C tools/gomadv3 test -tags test_dep ./world
env -u GOMADSEED GOWORK=off go -C tools/gomadv3 test -tags test_dep -run TestWorldOrdering -count 100 ./world
env -u GOMADSEED GOWORK=off go -C tools/gomadv3 test -tags test_dep -run TestWorldCancelQuiesceRace -count 1000 ./world
env -u GOMADSEED GOWORK=off go -C tools/gomadv3 test -race -tags test_dep ./world
env -u GOMADSEED GOWORK=off go -C tools/gomadv3 test -tags test_dep -fuzz FuzzRestore -fuzztime 30s ./world
env -u GOMADSEED GOWORK=off go -C tools/gomadv3 test -tags test_dep -run '^$' -bench World -benchmem ./world
make -C tools/gomadv3 test
make fmt-imports
make lint-code
```

The race command uses the stock runtime because enabled Gomad mode does not
support the race detector. The implementation suite should additionally scan
World imports for forbidden host packages and compare golden snapshots and
digests on every supported architecture.

## Completion criteria

- The public interface and serializable schema above are implemented in the
  dependency-free `tools/gomadv3` module.
- Pure in-memory tests register, mark ready, cancel, order, and atomically
  deliver external events with stable identities and bounded storage.
- Ordinary ordering is invariant across seeds; explicitly equivalent events
  repeat for one seed and demonstrate permitted diversity across a bounded seed
  set.
- World choice derivation is stateless, domain-separated, and independent of
  runtime and application random streams.
- Idle, World deadlock, cancellation outcomes, invalid input, capacity, native
  deadlock, and Runner wall timeout have distinct documented results.
- Snapshots are canonical, bounded, restorable after every completed public
  transition, and reject all tested corruption or incompatible schema cases.
- External-event replay validates every operation and result, reports the first
  divergence without mutation, detects early/extra calls, and verifies the
  final digest.
- Concurrent lifecycle tests pass the stock race detector, and no callback or
  goroutine runs inside World.
- A process crash can lose only work after the last atomically published
  composite snapshot; incomplete artifacts never claim replayability.
- Ten-times event load follows the documented heap and linear-memory behavior
  or fails at the recorded bound without partial state.
- The package performs no host I/O, uses no third-party dependency, and changes
  no Go runtime or native timer behavior.
- One deterministic adapter pilot passes before any native-timer coordination
  hook is proposed; any hook is backed by a minimized failure and satisfies the
  shared-instant evidence gate.
- Focused tests, the gomadv3 suite, formatting, and linting pass with every Go
  test invocation carrying `-tags test_dep`.
