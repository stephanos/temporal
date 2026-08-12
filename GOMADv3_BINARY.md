# Gomad v3 binary protocols

## Decision

Gomad retains its existing transport choices:

- a fixed-layout shared-memory log for deterministic I/O transcripts;
- small fixed bootstrap and completion frames for process activation and final
  status;
- bounded request/response pipes for lazy read-only mount lookup;
- simple length-prefixed envelopes around canonical World data; and
- explicit canonical byte encodings for cryptographic identities and choice
  ranks.

These uses of binary encoding solve different problems and must not be treated as
one general serialization subsystem. Protobuf is not adopted as a universal
replacement. The target-side implementations live inside a patched Go standard
library, have a deliberately constrained dependency graph, and in some cases
require fixed offsets or in-place access that a general message codec does not
provide.

The low-level encoding details should nevertheless become private implementation
details. Callers should operate on typed events, requests, responses, and status
values through narrow modules. Where the same wire format must be implemented in
both the Runner and the patched standard library, one declarative format
definition should generate dependency-free codecs for both endpoints and golden
tests should pin their compatibility.

## Protocol roles

| Protocol | Transport | Purpose | Required property |
|---|---|---|---|
| I/O transcript | Shared memory | Append and replay modeled target I/O | Fixed capacity, no per-event blocking, direct ordinal comparison |
| I/O terminal status | Pipe | Declare completion, overflow, or first replay divergence | Small integrity-checked final frame |
| I/O bootstrap | Inherited fixed frame | Bind profile, target, Runner, arguments, and seed before package initialization | Fixed size and dependency-free decoding |
| Read-only mount lookup | Request/response pipes | Ask a Runner-owned broker for an approved host entry | Bounded framing, synchronous errors, variable-sized results |
| World child configuration | Pipe | Supply bounded seed, limits, and optional replay input | Simple length-bounded framing |
| World recording envelope | Pipe and artifact payload | Carry canonical initial state, final state, and terminal result | Bounded length prefixes around canonical data |
| World identities and choice ranks | In-process canonical bytes | Hash semantic state and rank equivalent events | Stable, unambiguous cryptographic input |

The fact that these protocols use `binary.BigEndian` does not make them the same
kind of interface. Some are wire protocols, some are shared-memory layouts, and
some are merely canonical inputs to a hash function.

## Why the I/O transcript uses shared memory

The I/O transcript records operations performed inside an exact transparent I/O
profile. Recording itself must not introduce a new source of scheduling behavior.
If every operation wrote to a pipe, the target could block on pipe capacity or a
reader's progress. A reader goroutine or helper process would then participate in
host scheduling while Gomad is trying to observe deterministic target behavior.

Runner instead creates a private, bounded backing file, unlinks its name, sizes it
before target execution, and passes the descriptor through the supervised launch
chain. The target maps it with `MAP_SHARED` and appends fixed-size records under
its own lock. Each record contains an ordinal, operation identity, argument and
content digests, result fields, and deterministic stream positions.

This layout provides the properties the transcript needs:

- capacity is fixed before the target starts;
- recording does not require a syscall or host reader for every operation;
- record `N` has a constant offset and can be compared directly with expected
  record `N` during replay;
- overflow is detected before an unbounded allocation or silent truncation;
- replay input can be mapped read-only; and
- Runner can hash and retain exactly the bytes the target produced.

Shared memory is therefore used to avoid adding scheduling-sensitive backpressure
to the observed operation stream, not merely as a performance optimization.

Shared memory alone cannot prove that a target completed its transcript. A crash
may leave a valid prefix in the mapping. The target consequently writes a small
terminal frame over a separate pipe after freezing the log. That frame reports
completion, overflow, or replay divergence together with the final length, record
count, digest, and checksum. Runner accepts the mapped bytes only after validating
this frame. The pipe is suitable here because it is used once at the process
boundary rather than once per modeled operation.

## Why read-only mounts use pipes

A lazy read-only mount has a different ownership model. The target must not open
an approved host root directly. A Runner-owned broker pins and validates that
root, captures an entry on first observation, and returns only the modeled entry
to the target.

This is a synchronous request/response interaction:

```text
target os operation
        |
        | lookup ordinal and normalized path
        v
Runner-owned mount broker
        |
        | status, metadata, children, and bounded contents
        v
target in-memory file view
```

Pipes fit this interaction because each request requires an answer or an explicit
failure before the target can continue. EOF and broker termination are meaningful
failure signals, while response sizes vary with file contents and directory
entries. Shared memory would still require a notification and ownership protocol
for request and response slots, adding a ring buffer or mailbox without removing
the need for synchronization.

The mount transport must remain bounded and fail closed. It validates magic,
version, operation, ordinal, path length, content length, entry count, entry kind,
and status before allocating or exposing data. A request for an entry absent from
recorded replay input is a divergence; it must never fall back to the original
host root.

## Why protobuf is not the default

Protobuf is useful when independently evolving applications exchange structured
messages and can share generated types plus a supported runtime. Those are not
the dominant constraints here.

The target-side endpoints are compiled into a patched Go standard library. In
particular, the filesystem adapter is inside `os` and the transcript implementation
sits below packages that already depend on `os`. Importing a protobuf runtime
would expand the pinned overlay and can introduce standard-library import cycles.
Gomad intentionally keeps this dependency closure narrow and auditable.

Protobuf also does not provide the fixed offsets and in-place updates required by
the shared transcript. Using it there would require serializing variable-length
messages into another framing or indexing layer, reintroducing allocation,
capacity, and lookup machinery while losing direct ordinal comparison.

For the mount pipe, protobuf is technically possible but not automatically
simpler. The decoder would still need explicit limits for total message bytes,
paths, contents, children, allocations, unknown fields, and nesting. Hand-written
use of `protowire` would exchange fixed offsets for tag and varint handling without
removing low-level code. A generated protobuf runtime is disproportionate for one
lookup operation and its response.

Protobuf remains an option if a future protocol grows multiple independently
evolving message families and the patched endpoint can consume a small,
dependency-safe generated codec. Such a change must demonstrate less total
interface complexity while preserving strict bounds, exact replay identity, and
the overlay dependency audit.

## Rejected alternatives

### Buffered transcript pipe

A pipe offers a convenient stream interface, but its bounded kernel buffer and
reader progress can block the target. Increasing the buffer only moves the limit;
adding a concurrent reader makes host scheduling part of the recording path.

### JSON for target-side protocols

JSON is appropriate for low-frequency supervisor requests and reports, where
clarity is more valuable than fixed layout. It is a poor fit for the transcript
and mount hot paths because it adds variable allocation, escaping, numeric parsing,
and a larger decoder dependency surface. Canonical JSON remains appropriate for
durable records and semantic World snapshots.

### `encoding/binary.Write` over Go structs

This can shorten individual encoders but does not establish one schema, prevent
the host and overlay copies from drifting, centralize validation, or hide layout
knowledge from callers. It also introduces temporary buffers or reflective work
where direct fixed-layout access is intentional.

### Memory-mapped Go structs using `unsafe`

Mapping structs directly would make the format depend on Go layout, padding,
alignment, architecture, and memory-model details. Concurrent field publication
would additionally require explicit atomic semantics. An explicit byte format is
more portable and auditable.

### Shared memory for mount lookup

The mount broker would require slot ownership, notification, variable payload
storage, cancellation, EOF, and crash recovery. This is more machinery than the
bounded synchronous pipe protocol it would replace.

### Eagerly capturing entire mounted trees

Pre-capture removes the request protocol but makes preparation time, memory, and
artifact size proportional to the complete approved tree rather than the entries
the target observes. It also broadens semantic input unnecessarily.

### FUSE or another host filesystem service

An OS filesystem interface is familiar but introduces platform-specific
deployment, kernel or daemon readiness, host scheduling, and a much larger
security surface. It is not justified for exact target-specific profiles.

## Ergonomic module shape

The transport choices do not require callers to manipulate byte offsets. The
desired interfaces are typed and narrow:

```go
type TranscriptEvent struct {
	Ordinal      uint64
	Operation    string
	ArgumentHash [32]byte
	ContentHash  [32]byte
	Count        uint64
	Result       uint32
}

type Transcript interface {
	Append(TranscriptEvent) error
	Finish() (TranscriptStatus, error)
}

type MountClient interface {
	Lookup(path string) (MountEntry, MountStatus, error)
}
```

These examples describe module interfaces, not mandatory exported types. The
actual interfaces should remain internal and concrete unless a second adapter
requires abstraction. Callers should know the semantic event or lookup and its
documented errors; they should not know magic bytes, offsets, byte order,
descriptor numbers, or frame sizes.

The implementation should be organized as follows:

```text
I/O profile operations
        |
        v
typed transcript recorder/replayer
        |
        v
private fixed-layout shared-memory codec

overlaid os adapter
        |
        v
typed mount client
        |
        v
private bounded pipe codec <----> Runner mount broker
```

The mount client belongs below the `os` adapter, in the overlay's existing
`internal/gomadio` area. The `os` package should translate between ordinary
filesystem operations and typed mount results; it should not own framing,
ordinals, descriptor I/O, response allocation, or entry caching.

## One format definition, two dependency-free codecs

The Runner and patched standard library cannot import one another's Go package.
Copying protocol constants and offsets by hand is nevertheless brittle. The mount
format and any other duplicated cross-endpoint format should have one declarative
source that generates:

- a host codec under the Gomad module;
- an overlay codec with no dependencies outside its reviewed standard-library
  closure; and
- golden vectors consumed by tests on both sides.

The generator need not create a general serialization framework. It should
support only the small set of fixed integers, bounded byte strings, enums, and
repeated bounded entries used by the protocol. Generated files must expose typed
encode/decode operations and keep `binary.BigEndian`, offsets, and reserved bytes
inside the codec implementation.

Generation is justified only for formats duplicated across the host/overlay
seam. In-process World hash encodings and single-owner envelopes should retain
small explicit helpers rather than being forced into a universal schema system.

## Compatibility and validation

Binary protocol compatibility is an explicit contract. Each wire or memory
layout must define:

- distinct magic and version fields;
- total and per-field size bounds checked before allocation;
- legal enum values and reserved-byte behavior;
- ordinal and completion rules;
- checksum or digest coverage where partial publication matters;
- whether unknown versions or fields fail closed; and
- the exact bytes that enter replay or semantic identity.

Tests should include:

- golden bytes for the smallest and largest legal frames;
- host-generated frames decoded by the overlay codec and the reverse;
- truncated input at every field boundary;
- invalid magic, version, enum, ordinal, and reserved values;
- oversized lengths and counts before allocation;
- transcript capacity exhaustion and incomplete terminal status;
- first-record, middle-record, extra-record, and missing-record replay
  divergence; and
- fuzzing of every bounded decoder with an allocation ceiling and no panics.

Changing a canonical transcript, bootstrap, or replay format requires an explicit
version and compatibility decision. Refactoring the ergonomic wrapper must not
silently change existing bytes.

## Consequences

Gomad continues to carry some low-level binary implementation because its process
and patched-standard-library seams require properties that general serialization
libraries do not provide cheaply. That complexity is load-bearing only inside
the transport modules.

Moving layout knowledge behind typed modules increases locality: protocol changes,
bounds, errors, and compatibility tests live together. Callers gain leverage by
expressing semantic events and filesystem lookups without reproducing framing
rules. Generated dependency-free codecs remove the most brittle host/overlay
duplication without expanding the target dependency graph or weakening replay.

This decision does not require replacing the current transports. It requires
making their low-level nature an implementation detail rather than a property of
the surrounding code.
