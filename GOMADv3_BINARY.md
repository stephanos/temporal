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

## Current implementation map

The host-side implementations are split by protocol ownership rather than by
encoding mechanism:

| Protocol | Runner-side implementation | Target-side implementation |
|---|---|---|
| I/O bootstrap | `tools/gomadv3/internal/ioprofile/bootstrap.go` | `tools/gomadv3/overlay/src/runtime/gomad.go` and `tools/gomadv3/overlay/src/internal/gomadio/gomadio.go` |
| I/O transcript and terminal | `tools/gomadv3/internal/process/iotranscript.go` | `tools/gomadv3/overlay/src/internal/gomadtrace/trace.go` |
| Read-only mount lookup | `tools/gomadv3/internal/romount/wire.go` | `tools/gomadv3/overlay/src/os/gomad.go` |
| World child configuration | `tools/gomadv3/internal/worldpipe/config.go` | `tools/gomadv3/world/child/child.go` |
| World recording envelope | `tools/gomadv3/world/recording.go` and `tools/gomadv3/internal/worldrecord/worldrecord.go` | `tools/gomadv3/world/child/child.go` |
| World semantic identities | `tools/gomadv3/world/snapshot.go` and `tools/gomadv3/world/replay.go` | In-process only |
| World choice ranks | `tools/gomadv3/world/choice.go` | In-process only |

The bootstrap executable renumbers inherited descriptors immediately before
`exec` of the prepared target. The patched process therefore observes this fixed
descriptor set:

| Descriptor | Target access | Contents |
|---:|---|---|
| 3 | Read | World child configuration |
| 4 | Write | World recording envelope |
| 5 | Read, then close | I/O bootstrap frame |
| 6 | Read/write mapping | Produced I/O transcript |
| 7 | Write once | I/O terminal frame |
| 8 | Read-only mapping | Expected replay transcript |
| 9 | Write | Read-only mount requests |
| 10 | Read | Read-only mount responses |

Descriptor 5 is always installed, using an empty pipe when no I/O profile is
active. Descriptors 6 through 8 require transcript recording, and descriptors 9
and 10 require read-only mounts. Descriptor numbers are private launch ABI, not
values that profile operations or World adapters should expose.

## Implemented layouts

All multi-byte integers below are unsigned big-endian values. Reserved bytes are
currently written as zero.

### I/O bootstrap frame

`Profile.BootstrapFrame` produces exactly 212 bytes:

| Offset | Size | Field |
|---:|---:|---|
| 0 | 8 | Magic `GOMADIO\x01` |
| 8 | 2 | Format version, currently 1 |
| 10 | 2 | Frame kind, currently 1 |
| 12 | 32 | Raw SHA-256 of the profile inventory |
| 44 | 32 | Raw SHA-256 of the profile implementation |
| 76 | 32 | Raw SHA-256 of the prepared target |
| 108 | 32 | Raw SHA-256 of the Runner build |
| 140 | 32 | Raw SHA-256 of canonical target `argv` JSON |
| 172 | 8 | Schedule and deterministic-I/O seed |
| 180 | 32 | SHA-256 of bytes `[0:180]` |

The profile name is not carried as text. Host decoding resolves it from the
inventory and implementation digest pair. Encoding first re-resolves the named
profile and rejects stale inventory or implementation identities.

The supervisor sends this frame to the bootstrap executable inside its bounded
JSON control request. After activation, the bootstrap executable places the
frame alone in a new pipe, installs its read end as descriptor 5, and executes
the target. The patched runtime reads exactly 212 bytes before package
initialization, closes descriptor 5, and extracts the seed. `internal/gomadio`
then verifies the complete-frame checksum before enabling modeled I/O.

### Produced and expected I/O transcript mappings

The active overlay fixes both mappings at 64 MiB. A produced mapping begins with
a 64-byte header:

| Offset | Size | Field |
|---:|---:|---|
| 0 | 8 | Magic `GOMADTR\x01` |
| 8 | 4 | Format version, currently 1 |
| 12 | 4 | Reserved |
| 16 | 8 | Mapping capacity |
| 24 | 8 | Next record offset, initially 64 |
| 32 | 8 | Published record count, initially 0 |
| 40 | 24 | Reserved |

Runner creates, unlinks, sizes, and initializes the backing file before launch.
The target maps it with read/write `MAP_SHARED`. Publication is serialized by the
transcript mutex: the target copies a complete record first, then advances the
next offset and record count in the header while still holding the lock.

The expected replay mapping uses a related 64-byte header:

| Offset | Size | Field |
|---:|---:|---|
| 0 | 8 | Magic `GOMADXT\x01` |
| 8 | 4 | Format version, currently 1 |
| 12 | 1 | Replay-enabled flag |
| 13 | 3 | Reserved |
| 16 | 8 | Expected payload bytes |
| 24 | 8 | Expected record count |
| 32 | 32 | SHA-256 of the expected records |

Expected records start at offset 64. The target maps this file read-only,
requires a whole number of records, checks count and payload length agree, and
verifies the payload digest before recording its first operation.

Each transcript record is 128 bytes:

| Offset | Size | Field |
|---:|---:|---|
| 0 | 8 | Zero-based ordinal |
| 8 | 2 | Operation-name byte length, at most 22 |
| 10 | 22 | Operation name followed by zero padding |
| 32 | 32 | SHA-256 of canonical operation arguments |
| 64 | 32 | SHA-256 of observed or produced contents |
| 96 | 8 | Operation-specific byte or item count |
| 104 | 4 | Stable result class |
| 108 | 4 | Reserved |
| 112 | 8 | Deterministic entropy position before the operation |
| 120 | 8 | Deterministic entropy position after the operation |

Replay compares each newly encoded 128-byte record directly with the expected
record at the same ordinal. An extra or unequal record freezes the transcript at
that ordinal. Finalization also detects a missing suffix by comparing the final
record count with the expected count.

### I/O terminal frame

The target's exit hook freezes the mapping and writes one 104-byte frame to
descriptor 7:

| Offset | Size | Field |
|---:|---:|---|
| 0 | 8 | Magic `GOMADIT\x01` |
| 8 | 4 | Format version, currently 1 |
| 12 | 1 | State: 1 complete, 2 overflow, 3 replay divergence |
| 13 | 3 | Reserved |
| 16 | 8 | Record count |
| 24 | 8 | Mapping length including its 64-byte header |
| 32 | 32 | SHA-256 of the produced record bytes |
| 64 | 8 | First divergent ordinal when state is 3 |
| 72 | 32 | SHA-256 of bytes `[0:72]` |

The exit hook runs on normal exit and runtime failure paths that execute exit
hooks, and finalization is idempotent. Transcript overflow and an immediate
replay mismatch terminate the target with exit code 125 after attempting to
publish the terminal frame.
Runner reads at most 105 bytes from the pipe, thereby rejecting both truncation
and trailing bytes. It accepts complete and divergence frames, verifies the
checksum, bounds the reported length, re-reads exactly the published payload,
and checks its digest. Overflow is reported as an incomplete transcript.

### Read-only mount request and response

The patched `os` client serializes lookups under one mutex, so there is at most
one request in flight. The request header is 24 bytes followed by the normalized
absolute target path:

| Offset | Size | Field |
|---:|---:|---|
| 0 | 8 | Magic `GOMADRO\x01` |
| 8 | 2 | Format version, currently 1 |
| 10 | 2 | Operation, currently 1 for lookup |
| 12 | 8 | Zero-based request ordinal |
| 20 | 4 | Path byte length |

The response header is 40 bytes, followed first by file contents and then by
zero or more directory children:

| Offset | Size | Field |
|---:|---:|---|
| 0 | 8 | Magic `GOMADRS\x01` |
| 8 | 2 | Format version, currently 1 |
| 10 | 2 | Status: 0 OK, 1 unmounted, 2 not found |
| 12 | 8 | Echoed request ordinal |
| 20 | 1 | Entry kind: 1 regular file, 2 directory |
| 21 | 3 | Reserved |
| 24 | 4 | Permission bits |
| 28 | 8 | Content byte length |
| 36 | 4 | Directory-child count |

Each child is an 8-byte header followed by its name: a two-byte name length, a
one-byte kind, one reserved byte, and four permission bytes. The host enum also
defines status 3 for an error response, but the current broker does not emit it
and the patched `os` client does not accept it. The default broker limits are 4
KiB per path, 100,000 requests, 10,000 captured entries, 100,000 aggregate
directory entries, 16 MiB per regular file, and 64 MiB of aggregate file data.
The overlay independently fixes the path, single-file, and directory entry bounds
to their default values.

The broker requires consecutive ordinals and checks the path bound before
allocation. It opens each source with `os.OpenRoot`, rejects symbolic links,
hard-linked regular files, and unsupported entry kinds, and verifies identity,
mode, size, and modification time around capture. Successful and missing mounted
lookups are cached. Directory children and the persisted snapshot are sorted.
During replay the broker has no host roots; a lookup absent from the captured
entry and missing-path sets returns `ErrReplayDivergence` and stops the protocol
instead of consulting the host.

### World child configuration

The World configuration on descriptor 3 has a 32-byte header followed by an
optional expected initial snapshot:

| Offset | Size | Field |
|---:|---:|---|
| 0 | 8 | Magic `GOMADWC\x02` |
| 8 | 8 | Positive transition-byte limit |
| 16 | 8 | World seed |
| 24 | 8 | Expected-initial-snapshot byte length |

The expected snapshot is bounded at 64 MiB. The supervisor validates it as a
canonical World snapshot and checks its seed before target activation. The child
checks the same seed against the canonical decimal `GOMADSEED` environment
value. In replay it restores that snapshot; otherwise it verifies that the
application-supplied World has the configured seed.

### World recording envelope

Opening a World child session immediately writes the eight-byte magic
`GOMADW2\x00` to descriptor 4. This makes a child that opens a session but fails
before `Finish` distinguishable from one that never connected. Successful finish
appends the rest of this envelope:

```text
magic
u64 initial snapshot bytes | canonical initial snapshot JSON
u64 final snapshot bytes   | canonical final snapshot JSON
u64 terminal bytes         | canonical terminal JSON
```

Each snapshot is bounded at 64 MiB, the terminal is bounded at 1 MiB, and the
whole envelope is bounded at `2*64 MiB + 1 MiB + 40` bytes. Decoding rejects
unknown fields, trailing data, noncanonical JSON, invalid World state digests,
invalid transition history, and a terminal result inconsistent with the final
quiescence transition.

The envelope does not duplicate the transition delta. The final snapshot already
contains the complete transition history. `internal/worldrecord` verifies that
the initial history is an exact prefix of the final history, restores and replays
the delta, and then emits the durable initial snapshot, JSON-lines transition
delta, and final snapshot artifacts with their raw and semantic digests.

### World canonical hash input

World hashes use a separate in-process byte grammar. `uint32` and `uint64` are
fixed-width big-endian values; signed integers use their two's-complement bit
pattern as `uint64`; and strings and byte slices are encoded as an eight-byte
length followed by their bytes. Lists prefix their element count.
State and transcript hashes have distinct NUL-terminated domain prefixes:

- `gomadv3/world/state/v1\x00` covers the schema, configuration, logical time,
  next identifiers, payload accounting, requests, events, transitions, and
  transcript digest;
- `gomadv3/world/transcript/v1\x00` chains the previous digest with the next
  semantic transition; and
- `gomadv3/world/seed/v1\x00` derives a choice key from the seed.

Equivalent-event ranking computes
`HMAC-SHA256(key, domain || class-length || class || event-id)`, where `key` is
the SHA-256 seed derivation and the ranking domain is
`gomadv3/world/equivalent-event-order/v1\x00`. The event queue compares the
32-byte ranks lexicographically only after deterministic time, priority,
resource, operation, and equivalence-class fields compare equal. Request ID is
the final tie-breaker.

### Implementation status relative to this decision

The layouts above are implemented, but the encapsulation and generation work
described later in this document is not yet complete:

- transcript constants and SHA-256 logic are duplicated between
  `internal/process` and the patched `internal/gomadtrace` package;
- mount headers and constants are duplicated between `internal/romount` and the
  patched `os` package rather than generated from one definition;
- the target mount client still owns framing, descriptor I/O, response
  allocation, ordinals, and entry decoding directly inside `os`; and
- current decoders do not uniformly reject nonzero reserved bytes or validate
  every status and entry-kind enum at the codec boundary.

Toolchain integration tests exercise both sides together, while focused host
tests cover bootstrap mutation, transcript terminal validation, mount ordering
and bounds, World configuration framing, and World recording round trips. There
is not yet a generated cross-endpoint golden-vector suite. The module extraction,
schema generation, stricter boundary validation, and golden/fuzz coverage below
therefore remain follow-up work rather than descriptions of the current tree.

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
