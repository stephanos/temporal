# Gomad v3 Protobuf IPC Design

## Goal

Replace Gomad v3's private coordinator, supervisor, and target-bootstrap JSON
messages with versioned protobuf messages. Keep the public artifact, World
snapshot, transition, provenance, and semantic-digest contracts unchanged.

This is an IPC implementation change. It must not change Runner behavior,
failure classification, replay identity, process containment, or any persisted
bytes.

## Alternatives

The preferred design uses protobuf only for private process IPC. Converting all
persisted data would lose the stable canonical encoding required for hashing and
would still require custom bounds and semantic validation. Keeping JSON
everywhere would avoid a dependency but retain duplicated private wire structs
and ad hoc encoding. Defining protobuf schemas while continuing to serialize
the same messages as JSON would add a second type system without removing code.

## Schema and Ownership

One proto file defines the complete private IPC schema:

```text
tools/gomad3/proto/gomad3/ipc/v1/message.proto
```

Generated Go types live in the standalone Gomad module:

```text
tools/gomad3/internal/ipcpb/v1/message.pb.go
```

The schema contains coordinator request/response and summary messages,
supervisor request/report messages, the target-bootstrap request, and their
nested target and outcome types. Closed protocol vocabularies such as target
kind, failure policy, stop reason, report kind, and termination kind use proto
enums. Open diagnostic reasons and details remain strings. Durations are signed
nanoseconds so conversion to `time.Duration` is exact and does not introduce a
well-known-type dependency.

Generated messages remain private wire representations. The existing
`runner`, `process`, and `target` domain types remain their module interfaces.
Each owning package converts at its IPC seam and validates enum values, integer
ranges, durations, and required fields before using a decoded message.

## Framing

`tools/gomad3/internal/ipc` is a small, deep framing module shared by the
coordinator and supervisor implementations. Every message is encoded as a
four-byte unsigned big-endian payload length followed by one protobuf payload.
Callers supply the existing per-channel byte limit. The decoder validates the
declared size before allocating, reads exactly that payload, uses a bounded
protobuf recursion limit, and rejects retained unknown fields.

One-message channels require EOF after their frame. The supervisor report pipe
accepts an ordered sequence of frames until EOF; the existing report state
machine continues to reject missing, duplicate, out-of-order, or extra reports.
The fixed target identity frame and fixed World child-configuration frame do
not move to protobuf because their current formats are already smaller and
simpler than a generated message.

## Generation

The root Makefile gains a `gomad3-proto` target using Temporal's pinned
`protoc-gen-go` version and the repository's existing `protoc` convention. The
root `proto` workflow includes that target. The tool-local Makefile exposes a
matching `proto` target for focused development.

Because `tools/gomad3` builds with `GOWORK=off`, its own `go.mod` and `go.sum`
declare the protobuf runtime dependency explicitly. Generated code is checked
in, and normal Runner builds do not require `protoc`.

## Errors and Limits

Malformed lengths, oversized frames, truncated payloads, protobuf decode
errors, unknown fields, invalid enums, invalid numeric conversions, and
unexpected EOF remain protocol failures. Error text identifies the channel and
operation but does not include unbounded peer-controlled data. Existing process
cleanup and deadline paths run unchanged after any protocol failure.

The framing module never allocates more than the caller's existing message
limit. Protobuf does not replace application validation: environment, target,
duration, World, ordering, and process identity checks remain owned by their
current modules.

## Verification

Tests are written before each implementation step and cover:

- deterministic frame round trips, multiple frames, and clean EOF;
- oversized lengths, truncation, unknown fields, and trailing frames;
- lossless coordinator, supervisor, bootstrap, target, and summary conversion;
- rejection of every unknown enum and invalid duration or integer conversion;
- unchanged coordinator isolation, supervisor protocol state, containment,
  output bounds, artifact publication, and exact replay behavior;
- regeneration with no generated diff; and
- tagged Gomad tests, World race tests, cross-platform compilation, vet, and
  the complete Gomad v3 black-box suite.

Encoding and decoding remain linear in message size. At ten times the seed
count, coordinator IPC is still one request and one response, while supervisor
IPC remains a constant number of bounded messages per seed. A crash or partial
write produces a truncated-frame protocol failure and cannot publish a complete
record.
