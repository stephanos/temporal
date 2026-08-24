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
  +---- Guide ---- bounded semantic corpus and seed selection
  |
  +---- Record/Artifact ---- identity, persistence, and replay envelope
  |
  +---- target process ---- runtime choices and native virtual time
                              |
                              +---- deterministic I/O ---- transparent reviewed boundary
                              |
                              +---- World ---- explicit external-event model
```

These boundaries intentionally do not collapse into one controller:

- the runtime owns goroutine scheduling, native timers, maps, and synchronization;
- Runner owns host process lifetime, resource bounds, scheduling, and failure policy;
- Guide owns corpus identity, semantic prioritization, and atomic corpus updates;
- Campaign owns durable execution journaling; Artifact owns durable publication;
- Record owns raw bytes, hashes, and the outer replay envelope;
- World owns external-event identities, ordering, state, and semantic digests;
- each adapter owns its domain semantics; and
- deterministic I/O owns the reviewed transparent boundary for every
  Runner-managed target on one qualified toolchain and platform.

Within Runner, the seed campaign is a pure control state machine over pending
ordinals, parallel slots, aggregate counters, resume state, and failure-policy
stops. The orchestration loop owns process launches and hands completed results
to artifact publication; the campaign never prepares targets or writes files.
Parallel results enter semantic publication in selection-ordinal order, so host
completion timing cannot change the Campaign journal or guided corpus.

The mode is for trusted tests. The process boundary and fail-closed shims reduce
accidental host dependence; they are not an operating-system sandbox against a
target deliberately issuing raw syscalls.

## In-process cluster simulation

`tools/gomad3sim` owns the application-facing cluster seam. An Execution selects one
backend and records bounded node, topology, lifecycle, output, network, volume,
fault, scenario, history, and oracle evidence. The in-process backend assigns
inheritable runtime domains to logical
node incarnations; stale domains fail before model mutation. Package globals
and computationally live crashed goroutines remain shared-process limitations,
so fresh initialization and hard cleanup are provided only by the process
backend.

The virtual network is a separate Execution-scoped deep module below ordinary `net`
TCP calls. It owns node addresses, deterministic ports, directional links,
fixed delay, partition/heal, listeners, streams, queued deliveries, lifecycle
revocation, capacity, snapshots, and replay. Every endpoint and queued delivery
is incarnation-bound. Graceful stop closes the local side and exposes EOF;
crash resets both sides and removes pending traffic before a restart can reuse
the stable node address.

Network transitions are partitioned into canonical causal lanes. Operations on
one connection or listener retain order, and all topology changes share one
ordered lane; arrival order between independent resources is normalized.
Replay validates the next transition in the affected lane before mutation and
also requires exact final state. This network identity remains independent from
runtime choice, lifecycle, output, volume, fault, scenario, and oracle
identities.

The durable-volume and fault controllers are separate deep modules. Volume
owns persisted and volatile views, dependency-aware operations, sync, crash
selection, enumeration, snapshots, and replay. Fault plans own stable matching,
occurrence counting, target selection, application bounds, and fail-before-
mutation replay. Typed scenarios and semantic oracles consume those modules
through the Cluster seam without taking ownership of their state machines.

## Runtime choices and virtual time

### Activation

A directly launched target activates Gomad with `GOMADSEED`. A Runner-managed
target instead supplies the seed in an inherited, identity-bound bootstrap
configuration and uses the private `GOMAD3_IO_PROFILE` marker only to select
that bootstrap path. The marker names a versioned artifact identity, not a
user-selectable or target-specific profile. Both paths converge before package
initialization on the same runtime state.

Activation forces the initial `GOMAXPROCS` to one, disables asynchronous
preemption and the system monitor, initializes the seeded runtime choice state,
and starts the process clock at midnight UTC on 2000-01-01. Disabled execution
retains the upstream runtime paths.

The supported CI host also runs a privileged DTrace escape audit. A marker in
an unsigned probe binary activates observation only after runtime startup; an
unseeded positive control must reach both `clock_gettime` and
`mach_absolute_time`, while the seeded execution must reach neither. Missing
privileges, missing probes, or an unobserved marker fail the gate rather than
silently skipping it.

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
produce a target first. `exec` requires canonical v3 provenance containing the
same policy-versioned package-closure review: direct and test-only imports,
foreign sources, overlay-resolved source hashes, module identities, and the
generated test main are all explicit evidence. Runner checks standard-package
claims against the pinned toolchain and module claims against the executable's
embedded build information before accepting the binary. The provenance remains
a declaration made by trusted build tooling; its binary hash binds that
declaration to the exact supplied bytes. Runner validates and hashes the
prepared bytes before execution and again before publication.

The target environment starts empty. Runner adds only its activation values,
UTC, and explicitly supplied validated entries; runtime, toolchain, and dynamic
loader controls are reserved. Ambient credentials and host configuration
therefore cannot enter a deterministic Execution or Artifact accidentally.

On Unix, a supervisor places the target at the head of a new process group. A
liveness channel and an independently known absolute deadline allow the
supervisor to terminate the group if Runner cancels, stalls, or exits. Shutdown
sends `SIGTERM`, waits only within the existing deadline, escalates to `SIGKILL`,
reaps the leader, and verifies that the group is gone. This contains ordinary
bugs and unsupported subprocess use, not adversarial descendants that escape
their session.

Per-Execution and overall deadlines are host safeguards. They never advance logical
time. A logical `go test` timeout is a target result; a wall watchdog expiry is a
bounded diagnostic observation; failure to terminate or reap the target is a
Runner/host failure.

Runner drains stdout and stderr concurrently, hashes every byte, and retains a
bounded head and tail. Output timing and host completion order are diagnostics
and never enter runtime or World decisions.

`runner/internal/execution.Spec` groups World and deterministic-I/O inputs as typed execution
capabilities. On Unix, one process-owned launch-resource plan creates the pipes
and backings, fixes every stage's descriptor numbers and inheritance order, and
defines which ends close after each process start. The supervisor and bootstrap
remain separate containment stages; neither caller reconstructs `ExtraFiles` or
the final `dup2` layout independently. Host launch orchestration and output
collection live in `process_unix.go`; supervisor activation, process-group
termination, reaping, and cleanup live in `supervisor_unix.go` while sharing
that unchanged launch plan.

World transport remains enabled for every Runner-managed target. Although the
launch plan now represents World explicitly, making its descriptors optional is
deferred until external targets have been audited for calls to
`world/process.Open` and migrated to an explicit declaration. Until then, an empty
child record continues to become the canonical `none` World record. This keeps
the descriptor refactor compatible rather than silently disconnecting an
existing World-aware target.

## Records, artifacts, and replay

Record defines the outer versioned envelope and canonical identities. It treats
World snapshots, transitions, adapter data, and I/O transcripts as validated
payloads owned by their respective modules rather than reimplementing their
semantics. `record.go` owns the public hashing and manifest finalization entry
points, `validation.go` owns envelope validation, and `identity.go` owns the
record and failure identity projections. These remain files in one package so
the internal split does not add forwarding APIs.

Record and failure hashes exclude diagnostic host timestamps and paths. Failure
signatures also exclude the seed so byte-equivalent observations from different
seeds can be grouped. Full stream hashes, not retained output fragments, enter
the identity.

Manifest schema v2 keeps its existing JSON shape but requires the universal
deterministic-I/O identity and matching `GOMAD3_IO_PROFILE` environment entry
for every artifact. Previously accepted profile-less v2 data is treated as an
incomplete artifact and rejected rather than migrated or replayed through host
I/O. Artifacts emitted with the deterministic-I/O identity retain their schema
and identity compatibility.

Artifact publication uses private staging, bounded files, content hashes,
durability operations, and a no-replace rename. A manifest is written last.
Interrupted work may leave explicit partial diagnostics but can never appear as
a complete replayable artifact. Existing content-addressed artifacts are reused
only after complete validation.

`runner/internal/campaign` owns the durable Campaign state machine: planned,
prepared, running, committing, published, and recoverable-failure state;
preparation and per-Execution partial directories; bounded immutable Execution
segments; compact index and `campaign.json` publication; inspection; locked
recovery; and resume preflight. Campaign plan v1 declares journal, simultaneous
partial-Execution, transcript, retained-success, failure, aggregate Artifact
ceilings, and optional portable-plan shard identity. Campaign v1 binds every
closed segment through `executions/index.json`; sharded Campaigns also bind the
exact external plan and ordinal partition. The validated final manifest is
authoritative. Recovery reconstructs validated state, incorporates at most one
contiguous post-rename segment, archives an active partial before trimming a
torn terminal record, and never edits a closed segment. Injected create, sync,
rename, and delete failures must leave a published or resumable Campaign. Runner
advances semantic Execution states but does not implement filesystem publication
or integrity decisions.

The portable campaign-plan module separates immutable work identity from
execution. A `gomad3.campaign-plan/v1` file binds the Runner, toolchain,
deterministic profiles, environment, complete selection, ordinal mapping,
bounds, prepared target, and captured read-only mount digest. Its adjacent
private bundle contains only the verified target and path-independent numbered
mount trees; the plan file is published last. Static seed shards own disjoint
global ordinals by `ordinal % count`, and resume preserves that assignment.
Merge validates every Campaign v1 source through the Campaign module, requires one
plan identity, rejects overlap or unexplained gaps, stores content-deduplicated
evidence metadata once in a bounded segmented journal, and publishes an
immutable `gomad3.merged-campaign/v1` without changing source artifacts.

Replay performs all identity and payload validation before starting the stored
target. It never rebuilds from source, substitutes a local binary, silently
migrates a schema, or falls back to live host input. Exact replay compares the
new semantic result with the artifact. Watchdog replay remains diagnostic
because host elapsed time is not deterministic.

### Guided semantic exploration

Guide is a deep module around a private bounded corpus. Runner opens it only
after preparing the target, then selects the complete Campaign from that one
immutable snapshot. Rarity within higher-value semantic domains orders retained
seeds; no more than three quarters of a Campaign may come from the corpus, leaving
at least `ceil(count/4)` requested seeds unguided. The recorded Campaign plan binds
the snapshot hash and final mixed selection. Resume uses that selection without
consulting a later snapshot for scheduling.

A corpus identity binds the execution-relevant target projection, toolchain,
generated boundary manifest, semantic probe instrumentation, manifest schema,
and record contract. Each entry binds its seed and record hash to the retained
exact-replay artifact, payload size, I/O transcript, World inputs and
transitions, read-only mounts, semantic coverage, novelty reasons, and verified
matching replay. Opening validates the complete index and every referenced
artifact before exposing any seeds. One nonblocking filesystem lock permits a
single writer, and fixed limits of 1,024 entries and 1 GiB keep validation and
selection bounded.

Candidate artifacts are durably content-addressed before replay. Only an
interesting candidate whose exact replay verifies and matches can enter a new
canonical index written by file sync, rename, and directory sync. A crash may
therefore leave an unreferenced immutable case but cannot claim its coverage;
the next open removes such cases. Parallel candidates merge in selection order.

Features use stable failure identities, abstract World state changes and
transition outcomes, operation and transition pairs, I/O names and results,
and generated boundary-probe IDs. World features omit seeds, sequence and
request/event identities, logical times, resource keys, and payloads. The
feature schema and probe instrumentation jointly enter the corpus identity.
Observation consumes neither runtime randomness nor host time. Reproducible
failures outrank invariant and terminal states, World and I/O outcomes,
operation pairs, and boundary probes; payload size breaks remaining ties and
rewards smaller reproductions. Code-edge coverage remains a separate,
lower-priority input for a future independent producer rather than changing the
versioned semantic-probe contract.

The first guidance stage deliberately reuses realized seeds and captured
transcripts. World scenario, fault, and input generation is gated on evidence
that seed guidance improves exploration, and forced runtime choices require a
minimized failure that retained seeds and transcripts cannot reproduce.

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

## Transparent deterministic I/O

World is an explicit event model; transparent deterministic I/O is a different
integration boundary. Every Runner-managed target uses the same versioned
boundary for the qualified toolchain and platform. Standard-library shims cover
the inventoried operations independently of the target package or arguments,
bind their implementation and inventory identities into the artifact, and fail
closed at unsupported reviewed entry points.

`boundary/manifest.json` is the canonical inventory of the currently reviewed
standard-library entry points. It records each target's signature, semantic
operation, stable probe, disposition, hook or delegated boundary, permitted
adapter closure, and fixtures. Generation emits the version-specific compiler
table, applied-interception report, and human-readable inventory; validation
rejects malformed, duplicate, or stale declarations before a toolchain build.
The generated compiler table also carries the formatted declaration fingerprint
of every intercepted definition. The compiler rereads and hashes the selected
source declaration before inserting a prologue, so a signature-compatible body
change cannot silently retain an obsolete interception decision.
Qualification also discovers public callers of host-capability sinks and
methods on capability-bearing handles. Every discovered target must be directly
intercepted or carry an explicit transitive, dynamic, unreachable, patch, or
upstream disposition; static delegates must still reach a declared hook.

Each compiler prologue records its generated stable probe ID once per process,
before hook dispatch. The generated semantic-canary test runs the filesystem
and network fixtures and fails if any manifest hook is unobserved, while the
fixtures independently assert the modeled result or stable rejection.

The manifest also generates uniform denial hooks only when an interception
names a complete hook policy. That policy fixes disabled execution as an
upstream fallback, transcript observation as the compiler probe, zero result
values, the exact unsupported error, and error wrapping. Stateful denials and
modeled operations remain handwritten; adding `disposition: deny` alone never
opts a hook into generation.

The implementation retains one immutable internal profile specification because
bootstrap frames and existing artifact schemas need a stable name, target
contract, inventory, identities, and build-overlay policy. It has no name-based
selection path and is not a public registry or extension mechanism.
Foreign-runtime adapters form a separate,
closed, version-pinned registry selected from target build metadata. The current
`modernc.org/libc` adapter is generic to that reviewed dependency version; it is
not keyed to SQLite, Temporal, or an individual test.

Modeled I/O is appended to a bounded deterministic transcript. Replay supplies
the recorded transcript and stops at the first mismatching operation. Host data
that is intentionally imported, such as a read-only mount, is captured through
a Runner-owned boundary and must replay from artifact data without reopening the
original host source.

Deterministic I/O need not route operations through World when a synchronous,
explicitly ordered transcript is sufficient. An adapter should adopt World only
when it needs modeled external readiness, competing events, cancellation, or
logical time coordination. This avoids imposing a speculative event scheduler
on simple deterministic shims while preserving one World contract for adapters
that do need those semantics.

### Filesystem and mount ownership

`internal/gomadfs` is a purpose-built in-memory filesystem shared by the `os`
adapter and reviewed foreign-runtime adapters. It owns namespace and handle
semantics, deterministic timestamps, stable directory order, mount
immutability, and explicit capacity accounting behind a small operation
interface. Gomad does not use Afero here: Afero imports `os`, cannot sit below a
patched `os` package without a cycle, and does not cover unchanged libc callers
or Gomad's transcript, replay, and capacity contracts. The mount loader accepts
the generated mount wire value types directly, so `os` does not copy every
field between identical transport and filesystem structures. The `os` adapter
also owns one `gomadfs.Entry`-to-`FileInfo` projection shared by path stat,
handle stat, and directory reads; the filesystem keeps its richer stat result
and operation semantics private.

Explicit read-only mounts are the only brokered host filesystem input. A
Runner-owned broker pins each approved root, resolves descendants without
following symlinks, validates stable bounded captures, and sends typed entries
to the target. The target installs each first observation as an immutable
in-memory node. Replay serves only the artifact's captured entries and never
reopens the original host path; an uncaptured lookup diverges instead of falling
back to live input.

### Binary protocol ownership

The deterministic-I/O protocols retain transports selected for their distinct
runtime properties: fixed shared-memory transcript records, fixed bootstrap and
terminal frames, and bounded synchronous pipes for lazy read-only mounts. World
configuration and recording remain separately owned, explicit encodings; they
are not part of a universal serialization layer.

Cross-endpoint layouts are declared once in the deterministic-I/O, Choice, and
simulation schemas under their respective `schema` directories. The protocol
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
  logical test timeout, runtime fatal, or structured World/I/O failure;
- watchdog timeout: host time bounded a process that did not produce a complete
  target result;
- replay divergence: current deterministic interaction differs from the record;
- capacity or invalid input: a modeled boundary rejected an operation before
  partial mutation; and
- Runner/host failure: preparation, launch, containment, capture, integrity, or
  publication failed, so the Campaign cannot be claimed trustworthy.

No error path silently falls back to host time, host readiness, live replay
input, an approximate schema, or an unbounded allocation.

## Host tooling boundary

Host policy is split into deep, typed modules: source archives, patch sets,
toolchain publication, command supervision, bounded output capture, and the
black-box test campaign each expose a narrow Go interface.
`cmd/gomadtool` is their command adapter; Make retains stable target names but does not own
lifecycle or result-classification policy. The test driver records one bounded
case result per external command and keeps equality, diversity, diagnostics,
timeouts, and mandatory semantic markers as distinct oracles.

Shell is limited to reviewed argv and platform boundaries. The patch-regeneration
scripts are owned by `internal/gomadtool/conformance/scripts`: `exec.sh` and
`compiler_test_exec.sh` adapt upstream Go hooks, while `clock_audit_test.sh` owns the
Darwin DTrace invocation. A Go-owned content check rejects new script owners,
Bash outside the explicit platform adapter, and Perl policy. Platform-neutral
host-tool tests run on Linux, while runtime qualification remains exclusively
the complete `darwin/arm64` gate.

## Maintenance gates

`version.json` is the canonical release descriptor. It owns the Go archive and
digest, supported platforms, patch name, boundary-manifest version, adapter
versions, and exact patch/overlay source sets. Generation produces its Make,
Go, and human-guide consumers; validation requires the allowlists to equal
the actual patch and overlay tree rather than merely containing them.

The runtime patch and transparent I/O overlays are pinned implementation costs.
Every Go upgrade runs the typed `gomadtool upgrade-dossier` host command, which records the
complete upstream patch, semantic boundary diff, interception evidence,
archive-based overlay collision audit, disabled-mode upstream compatibility,
mandatory probes, optional retained-corpus evidence, and platform qualification
in one JSON dossier. The supported-host gate must also rerun the
positive-controlled host-clock trace because dynamic imports and probe names are
platform implementation details. The dossier is published on failure and
uploaded by CI, so a rejected upgrade retains its first failing gate and bounded
output. Boundary comparison canonicalizes complete manifest metadata, intercepts,
and hook policies so a field unknown to an older comparator cannot disappear
from upgrade evidence.

Broader runtime or compiler changes require a minimized real workload showing
that Runner, World, adapters, and records cannot satisfy the contract. Runtime
choice tracing, deterministic GC control, compiler checkpoints, and multi-P
execution remain separate research projects rather than incremental extensions
of the current design.
