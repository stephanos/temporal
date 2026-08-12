# Gomad v3 architecture

This document records the design decisions that are not obvious from the code.
The [README](README.md) is authoritative for supported commands and current
behavior. Public types, wire schemas, defaults, and limits are authoritative in
the implementation and its tests.

## System boundary

Gomad makes runtime-controlled choices repeatable when the toolchain, target,
architecture, deterministic inputs, and seed are unchanged. It does not claim
schedule stability across source or toolchain changes, exhaustive exploration,
or deterministic execution of arbitrary host I/O.

The system has four ownership boundaries:

```text
Runner ---- prepares and supervises one target process per seed
  |
  +---- Record/Artifact ---- identity, persistence, and replay envelope
  |
  +---- target process ---- runtime choices and native virtual time
                              |
                              +---- I/O profile ---- transparent target-specific I/O
                              |
                              +---- World ---- explicit external-event model
```

These boundaries intentionally do not collapse into one controller:

- the runtime owns goroutine scheduling, native timers, maps, and synchronization;
- Runner owns host process lifetime, resource bounds, and artifact publication;
- Record owns raw bytes, hashes, and the outer replay envelope;
- World owns external-event identities, ordering, state, and semantic digests;
- each adapter owns its domain semantics; and
- an I/O profile owns the reviewed transparent boundary for one exact target.

The mode is for trusted tests. The process boundary and fail-closed shims reduce
accidental host dependence; they are not an operating-system sandbox against a
target deliberately issuing raw syscalls.

## Runtime choices and virtual time

### Activation

A directly launched target activates Gomad with `GOMADSEED`. A Runner-managed
I/O profile instead supplies the seed in an inherited, identity-bound bootstrap
configuration and uses `GOMADV3_IO_PROFILE` only to select that bootstrap path.
Both paths converge before package initialization on the same runtime state.

Activation forces the initial `GOMAXPROCS` to one, disables asynchronous
preemption and the system monitor, initializes the seeded runtime choice state,
and starts the process clock at midnight UTC on 2000-01-01. Disabled execution
retains the upstream runtime paths.

### Why process faketime

Gomad activates the dormant process-wide `faketime` machinery in the pinned Go
runtime. This preserves ordinary `runtime.main`, package initialization,
binaries, and the `testing` harness while covering standard `time` and context
deadlines transparently.

The rejected alternatives have materially larger or weaker boundaries:

- wrapping the process in a `testing/synctest` bubble changes main-goroutine and
  initialization structure and conflicts with tests that create nested bubbles;
- injecting a clock misses package initialization, third-party `time` calls,
  standard-library deadlines, and the test harness; and
- source or package rewriting has incomplete coverage and introduces a second
  timer controller.

Explicit `testing/synctest` bubbles keep their private clocks and take precedence
over the process clock.

### Quiescence and native timers

The native runtime timer heaps remain the only queues for `time.Sleep`, timers,
tickers, callbacks, and context deadlines. The runtime advances directly to the
earliest deadline only after its deadlock accounting proves that no goroutine is
runnable. All timers at that instant become eligible before scheduling resumes.

A runnable goroutine, including a busy loop or polling `select`, prevents time
advancement. Unsupported blocking host I/O also cannot be converted into a
logical clock event. Runner's wall watchdog bounds both cases without letting
host elapsed time affect supported timer delivery.

Timers with equal deadlines use a seeded runtime tie-break. The tie-break may
change when unrelated runtime choices are added to the program; reproducibility
is for an unchanged target, not a stable global choice numbering scheme.

Native timers must not be copied into World. If an external adapter needs to
compete with a native deadline, a runtime hook is justified only by a minimized
case that cannot be coordinated outside the runtime. Such a hook must compare
the earliest native and World events at the existing quiescence point, advance
one logical instant, and make every event at that instant eligible without
moving external payload or adapter policy into the runtime.

## Runner and process containment

Runner prepares one immutable target and launches every seed in a fresh process
and working directory. Building once makes target identity independent of seed;
fresh processes prevent globals, goroutines, descriptors, allocator state, and
runtime randomness from leaking between seeds. Parallelism is across processes,
not through multiple Ps inside one target.

The Go build driver runs outside deterministic mode. `go-run` and `go-test`
produce a target first, while `exec` requires trusted provenance for the supplied
binary. Runner validates and hashes the prepared bytes before execution and
again before publication.

The target environment starts empty. Runner adds only its activation values,
UTC, and explicitly supplied validated entries; runtime, toolchain, and dynamic
loader controls are reserved. Ambient credentials and host configuration
therefore cannot enter a deterministic run or artifact accidentally.

On Unix, a supervisor places the target at the head of a new process group. A
liveness channel and an independently known absolute deadline allow the
supervisor to terminate the group if Runner cancels, stalls, or exits. Shutdown
sends `SIGTERM`, waits only within the existing deadline, escalates to `SIGKILL`,
reaps the leader, and verifies that the group is gone. This contains ordinary
bugs and unsupported subprocess use, not adversarial descendants that escape
their session.

Per-run and overall deadlines are host safeguards. They never advance logical
time. A logical `go test` timeout is a target result; a wall watchdog expiry is a
bounded diagnostic observation; failure to terminate or reap the target is a
Runner/host failure.

Runner drains stdout and stderr concurrently, hashes every byte, and retains a
bounded head and tail. Output timing and host completion order are diagnostics
and never enter runtime or World decisions.

## Records, artifacts, and replay

Record defines the outer versioned envelope and canonical identities. It treats
World snapshots, transitions, adapter data, and I/O transcripts as validated
payloads owned by their respective modules rather than reimplementing their
semantics.

Record and failure hashes exclude diagnostic host timestamps and paths. Failure
signatures also exclude the seed so byte-equivalent observations from different
seeds can be grouped. Full stream hashes, not retained output fragments, enter
the identity.

Artifact publication uses private staging, bounded files, content hashes,
durability operations, and a no-replace rename. A manifest is written last.
Interrupted work may leave explicit partial diagnostics but can never appear as
a complete replayable artifact. Existing content-addressed artifacts are reused
only after complete validation.

Replay performs all identity and payload validation before starting the stored
target. It never rebuilds from source, substitutes a local binary, silently
migrates a schema, or falls back to live host input. Exact replay compares the
new semantic result with the artifact. Watchdog replay remains diagnostic
because host elapsed time is not deterministic.

## World

World is a pure in-memory model for deterministic events outside the runtime.
It performs no host I/O, starts no goroutines, invokes no callbacks, and does not
read the runtime's random state or clock. Its public methods accept and return
detached data under one mutex, allowing adapters to wake application code only
after World releases its lock.

### Lifecycle and identity

Requests and readiness events receive monotonically increasing identities that
are never reused during a run. A request progresses from pending to queued and
then delivered, or cancellation wins before delivery. Duplicate readiness,
unknown identities, time regression, invalid input, and exhausted identity
space fail without partially mutating state or consuming replay entries.

World retains terminal metadata so duplicate operations and replay results stay
stable. Recorded lifetime limits bound that retention; capacity failure never
permits dropping history or falling back to host behavior.

### Event ordering

World orders queued events lexicographically by:

1. logical time;
2. semantic priority;
3. canonical adapter, resource kind, and resource key;
4. request and event kinds;
5. equivalence class; and
6. registration sequence or a seed-derived choice rank, followed by registration
   sequence as the collision fallback.

Ordinary events retain semantic/FIFO order. An adapter may assign a nonempty
equivalence class only when exchanging those events cannot change its semantics
apart from the intentional choice being explored. The choice rank is a
stateless, versioned, domain-separated HMAC over the seed and stable event
identity. It never consumes the runtime's private random stream.

Pointers, goroutine identities, host timestamps, callback arrival order, map
iteration, heap position, and OS resource numbers are not ordering inputs.

### Quiescence

`World.Quiesce` is an assertion by its caller that application work in the
claimed deterministic region cannot proceed. The standalone module cannot prove
runtime quiescence.

When events are queued, World advances to the earliest event time and atomically
delivers every event at that instant in queue order. With pending requests but
no queued events it reports World deadlock. With neither it reports idle. These
results remain distinct from native runtime deadlock and Runner's wall timeout.

### Snapshots and replay

Snapshots contain only versioned, bounded, serializable data. Collections are
canonically sorted; implementation maps, heap positions, pointers, callbacks,
and channels are excluded. Restore validates the complete schema, ordering,
cross-references, capacity accounting, sequence space, and semantic digest
before publishing a usable World.

External-event replay validates each requested transition before applying it.
The first incompatible input, result, order, missing operation, or extra
operation reports a stable divergence without advancing state or the replay
cursor. Runner separately validates exhaustion and the final state digest at
process exit.

Adapter state is not an opaque registry inside World. Each adapter defines its
own versioned snapshot, and Runner composes adapter and World data into one
record generation. This keeps the event core deep without making it responsible
for filesystem, network, persistence, or process semantics.

## Transparent I/O profiles

World is an explicit event model; a transparent I/O profile is a different
integration boundary. A profile is restricted to an exact target and reviewed
dependency closure. Standard-library shims and generated overlays replace only
the inventoried operations needed by that target, bind their implementation and
inventory identities into the artifact, and fail closed at unsupported reviewed
entry points.

Modeled I/O is appended to a bounded deterministic transcript. Replay supplies
the recorded transcript and stops at the first mismatching operation. Host data
that is intentionally imported, such as a read-only mount, is captured through
a Runner-owned boundary and must replay from artifact data without reopening the
original host source.

Profiles need not route operations through World when a synchronous, explicitly
ordered transcript is sufficient. A profile should adopt World only when it
needs modeled external readiness, competing events, cancellation, or logical
time coordination. This avoids imposing a speculative event scheduler on simple
deterministic shims while preserving one World contract for adapters that do
need those semantics.

### Binary protocol ownership

The deterministic-I/O protocols retain transports selected for their distinct
runtime properties: fixed shared-memory transcript records, fixed bootstrap and
terminal frames, and bounded synchronous pipes for lazy read-only mounts. World
configuration and recording remain separately owned, explicit encodings; they
are not part of a universal serialization layer.

Cross-endpoint layouts are declared once in `protocol/iowire.json`. The protocol
generator emits dependency-free typed codecs and the same golden, truncation,
validation, allocation-bound, and fuzz tests for the Runner module and patched
standard library. `make generate` updates checked-in output, while `make
validate` rejects drift. Protocol changes require an explicit version and
compatibility decision.

Callers do not own offsets, byte order, magic, reserved bytes, or enum checks.
Runner-side bootstrap, transcript, and mount packages use the generated host
codec. In the target, `internal/gomadwire` owns the layouts,
`internal/gomadtrace` owns typed transcript recording and replay, and the typed
`internal/gomadio/mount` client owns descriptors, framing, bounds, serialization,
and request ordinals below the `os` adapter. This keeps the patched dependency
closure small while making malformed input fail before allocation or exposure.

## Failure domains

Gomad keeps these outcomes distinct:

- target failure: the process completed with a deterministic exit, signal,
  logical test timeout, runtime fatal, or structured World/profile failure;
- watchdog timeout: host time bounded a process that did not produce a complete
  target result;
- replay divergence: current deterministic interaction differs from the record;
- capacity or invalid input: a modeled boundary rejected an operation before
  partial mutation; and
- Runner/host failure: preparation, launch, containment, capture, integrity, or
  publication failed, so the batch cannot be claimed trustworthy.

No error path silently falls back to host time, host readiness, live replay
input, an approximate schema, or an unbounded allocation.

## Maintenance gates

The runtime patch and transparent I/O overlays are pinned implementation costs.
Every Go upgrade requires disabled-mode compatibility checks and a fresh audit
of the touched runtime, linker, and standard-library paths.

Broader runtime or compiler changes require a minimized real workload showing
that Runner, World, adapters, and records cannot satisfy the contract. Runtime
choice tracing, deterministic GC control, compiler checkpoints, and multi-P
execution remain separate research projects rather than incremental extensions
of the current design.
