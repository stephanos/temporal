# Gomad v3 architecture

This document records the design decisions that are not obvious from the code.
The [SPEC](SPEC.md) owns product requirements and the
[canonical vocabulary](SPEC.md#productvocabulary-ubiquitous-language).
The [README](README.md) describes supported commands and current behavior.
Public types, wire schemas, defaults, and limits are authoritative in the
implementation and its tests.

## System boundary

Gomad makes runtime-controlled choices repeatable when the toolchain, target,
architecture, deterministic inputs, and seed are unchanged. It does not claim
schedule stability across source or toolchain changes, exhaustive exploration,
or deterministic execution of arbitrary host I/O.

The complete Runner and deterministic-I/O contract is qualified on
`darwin/arm64` and `linux/amd64`; artifacts replay only on their recorded
platform. Qualification applies to declared workloads, rather than proving
arbitrary Go programs deterministic. The
[milestones](../../MILESTONES.md#open-findings) record remaining
functional-suite replay divergences and known host-clock escapes, including
`MemStats.LastGC`. Garbage-collector timing is not fully controlled.

The system separates these ownership boundaries:

```text
Runner ---- prepares a Target and supervises each isolated Execution
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

## Cluster simulation

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

### Incarnation, backend, and fidelity

A cluster selects one backend and declares its fidelity for its lifetime.
Backend specifies the execution mechanism; Fidelity specifies the guarantee
being claimed. The in-process backend supports Model Fidelity; the process
backend supports Model Fidelity or Hard Isolation. Node IDs are stable and
incarnations increase monotonically. Restart retains configured addresses, boot
identity, and declared durable volumes while revoking old handles, mappings,
connections, and model access. Stale work fails before consuming fault/replay
entries or mutating model state.

In-process revocation cannot reset arbitrary package globals or terminate a CPU
loop. Graceful Stop checks cancellation before admission, then waits for the
boot's terminal commit; boot code that ignores cancellation requires the outer
watchdog. Fresh package initialization and hard crash/reap belong to the process
backend. Invalid lifecycle transitions cannot partially commit model state.

### Network and persistence semantics

Directional link changes affect newly submitted deliveries. Already queued
deliveries keep their scheduled order and time unless an endpoint or incarnation
is revoked. Delayed data cannot reach a restarted node through an old connection
or a reused address/port.

A volume write updates the volatile view and adds dependency-tracked pending
operations. File sync persists required data, size, allocation, and their
dependency closure; directory sync persists namespace changes. Graceful stop
flushes pending operations. Crash restores a selected dependency-closed
persistence outcome, and restart creates fresh handles and mappings.

Crash-state enumeration is canonical, bounded, and resumable. Reaching a state,
operation, depth, byte, or time bound produces explicit incomplete evidence and
remaining work. Captured read-only mounts are replay inputs; restarted nodes
never reopen their original host paths.

### Process arbitration and model evidence

Process nodes submit model operations and native timer/quiescence reports through
private bounded IPC. The coordinator owns shared model state and logical-time
advancement. Request identities correlate bounded IPC exchanges. Time advances
to the earliest participant deadline only after runnable work, in-flight handlers,
and delivered-but-unconsumed model operations are accounted for. Participants
blocked on external model work are excluded until their results arrive. Replay
validates semantic transitions rather than raw IPC arrival order. A node that
spins prevents quiescence and reaches the wall watchdog. Process death must leave
an operation classified as committed or uncommitted, with no ambiguous partial
model transition.

Runtime, scenario, network, storage, and fault evidence retain independent
identities. Replay validates static inputs before node activation, validates
choices and transitions before mutation, requires complete tape consumption, and
checks terminal model states, histories, output, and outcomes. Count/byte limits
belong to the recorded identity; overflow is a typed capacity result.

Backend conformance compares detached model transitions, topology, volume state,
histories, oracles, and normalized outcomes. Runtime tapes, raw logs, process
exits, and diagnostic timing need not match across backends. An artifact replays
only on its recorded backend and platform.

Oracles consume detached values without holding model locks, calling back from
World, or reading host time. Their histories and search work have separate bounds.
An explicit-model digest cannot establish equivalence of arbitrary native Go
execution state or justify pruning its schedule frontier.

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

Darwin CI also runs a privileged DTrace escape audit. A marker in
an unsigned probe binary activates observation only after runtime startup; an
unseeded positive control must reach both `clock_gettime` and
`mach_absolute_time`, while the seeded execution must reach neither. Missing
privileges, missing probes, or an unobserved marker fail the gate rather than
silently skipping it.

Both qualified platforms also pin standard-library host-clock references to a
reviewed static inventory. On Linux the vDSO clock path is not observable by a
syscall tracer, so static inventory is the current escape gate. The inventory
classifies known escapes; it does not claim they have all been removed.

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

The default `strict` clock-tick policy leaves `time.Now` unchanged while work is
runnable. The optional `forward` policy adds a cumulative 1–1024 nanosecond
increment at each `time.Now` read, using a separate seed-derived stream. Only
`time.Now` observes that offset; the native timer clock, scheduler clock reads,
and simulation time retain the idle-driven clock. Application calculations
that derive deadlines or durations from `time.Now` can still observe the
offset. For a reading that still carries its monotonic value, `time.Since`
and `time.Until` read the idle-driven clock rather than the ticked one, so an
elapsed time measured against such a reading is short by that reading's offset
and negative within one busy stretch, and a deadline derived from it expires
later than a native timer armed for the same duration. A reading stripped of
its monotonic value, as by `Round(0)`, serialization, or parsing, is compared
against a fresh ticked `time.Now` instead. Distinct readings are a
nanosecond-resolution property of process
`time.Now`; a `testing/synctest` bubble keeps its own clock, and timestamps
truncated to a coarser unit can still tie. The recorded environment binds the
policy into Campaign, Artifact,
and portable-plan identity, and replay restores it. Separating the offset
keeps timestamp ticking out of native timer-clock advancement.

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

### Choice traces, decision tapes, and exploration

When choice recording is enabled, the runtime records a bounded Choice Trace
containing logical decisions and observations. Exact runtime replay projects a
complete v2 trace into a Decision Tape containing only branching decisions;
observations and single-alternative decisions remain trace evidence. The tape
binds the Prepared Target, toolchain build key, platform, and choice
implementation. Stable logical alternative
identities and canonical alternative sets avoid treating physical run-queue
order as replay identity. Replay validates a decision before applying it,
requires complete tape consumption, and still compares the final Record.

Choice Exploration uses forced prefixes from one base Seed. A forced-prefix
candidate divergence invalidates search confidence and returns CLI status 3,
including mixed failures. Ordinary seeded and World replay divergence retain
status 1. Its pure controller
orders candidates by prefix length and identity in bounded breadth-first
rounds. Runner executes candidates in fresh processes and commits completed
results in candidate order; host completion order cannot alter the frontier.
Each completed round is an immutable, hash-linked transaction. Resume archives
an interrupted round and reruns it in full, keeping recovery attempts separate
from completed logical work. Outcome deduplication affects retained evidence
without pruning distinct prefixes.

The plan's start ordinal limits expansion to later replay-plan decisions while
preserving earlier decisions in forced prefixes. Select readiness is trace
evidence: polls with fewer than two ready cases do not expand the frontier,
and the controller counts their omitted alternatives separately. Resume keeps
both decisions under the frozen plan and controller identity.

Combined Exploration keeps runtime, scenario, network, storage, fault, and
crash decisions in separate dimensions with explicit global and per-dimension
bounds. A detached model digest cannot establish native-state
equivalence or justify pruning an unexplored runtime prefix. Bound exhaustion
reports the incomplete search envelope; completion is a claim only about the
declared envelope.

## Runner and process containment

Runner prepares a Target once and launches each Execution of that Prepared
Target in a fresh process and working directory. Building once makes target
identity independent of seed; fresh processes prevent globals, goroutines,
descriptors, allocator state, and runtime randomness from leaking between
seeds. Parallelism is across processes, not through multiple Ps inside one
target.

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

### Targets outside this repository

A target may live in any module, including one that depends on the server
through a local `replace`. `--working-dir` names its module root; build
adapters are selected from the go.mod of the module that owns the target, and
the build environment is the same forced one (no workspaces, no vendoring).
Packs for that module's own dependencies stay in its tree: an external pack
root is authored with `--compatibility-root` and loaded with
`GOMAD3_COMPATIBILITY_PACKS` under the same validation as embedded packs, and
each selected pack's identity binds the target, so replay without it fails
closed. Adapters stay embedded and exact, one version per module. What a
downstream module must still supply itself are source seams for its own
subprocess, signal, and host-filesystem calls and in-process substitutes for
services that run outside the process; Gomad does not model them.

## Records, artifacts, and replay

Record (`record.ExecutionRecord` in the implementation) defines the outer
versioned envelope and canonical identities. It treats
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

Campaign, corpus, and minimizer stores keep a content-addressed target pool
outside staged Campaign directories. Artifacts hard-link their prepared binary
from the pool, retain ordinary payload manifests, and validate the same bytes
on open and replay. Retained-byte accounting charges a shared target once per
pool; a host that cannot hard-link uses and counts private copies.

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

Minimization persists one state per parent below its output root, binding
attempt order, budget, implementation, and accepted artifacts. Resume validates
that state and its replay evidence before continuing; it preserves consumed
attempts and never mutates the parent. Exclusive ownership prevents concurrent
minimizers from advancing the same parent.

### Guided semantic exploration

Guide is a deep module around a private bounded corpus. Runner opens it only
after preparing the target, then selects the complete Campaign from that one
immutable snapshot. Rarity within higher-value semantic domains orders retained
seeds. Ordinary guidance excludes answered requested seeds and substitutes
nothing; a fully answered selection executes no seeds. Regression guidance
reuses corpus cases for no more than three quarters of a Campaign, leaving at
least `ceil(count/4)` requested seeds unguided. The recorded Campaign plan binds
the mode, snapshot hash, and final selection. Resume and shards use that
selection without consulting a later snapshot for scheduling.

A corpus identity binds the execution-relevant target projection, explicit
environment and clock-tick policy, toolchain,
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

Guidance reuses realized seeds and captured transcripts; it does not synthesize
World scenarios, faults, or inputs and never forces runtime choices. Choice
Exploration and Combined Exploration have separate bounded controllers
and journals rather than extending the corpus selector. This keeps corpus
admission and seed ranking independent from forced-decision frontier ownership.

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
standard library. `make -C tools/gomad3 generate` updates checked-in output,
while `make -C tools/gomad3 validate` rejects drift. Protocol changes require an explicit version and
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
host-tool tests and the complete runtime qualification gate run on both
`darwin/arm64` and `linux/amd64`. The macOS sandbox test and privileged DTrace
audit remain Darwin-specific.

## Maintenance gates

`toolchain/version/version.json` is the canonical release descriptor. It owns
the Go archive and digest, supported platforms, patch name, boundary-manifest
version, adapter versions, and exact patch/overlay source sets. Generation
produces its Make, Go, and human-guide consumers; validation requires the
allowlists to equal the actual patch and overlay tree rather than merely
containing them.

`pin-impact` compares complete candidate and baseline module identities
against adapter, compatibility-pack, interception, and clock-inventory pins.
It reports an unavailable identity as unknown and therefore invalidated.
`adapter-regenerate` derives rewrites with exact-occurrence anchors in a
private cache, binds the changed source and proposed anchors to a review digest,
and publishes the descriptor, adapter, fixtures, and generated consumers only
after matching approval. Publication checks the input snapshot under an
exclusive lock and recovers an interrupted transaction before another apply.
`compatibility-pack refresh` reads the same request-to-target table as the
qualification Make target; it batches fresh discovery and review while leaving
generation behind each request's exact approval. Other-platform requests remain
untouched, and a changed review invalidates its former approval. Both supported
hosts must run validation, pack qualification, core qualification, and the
complete test gate for an accepted pin update.

The runtime patch and transparent I/O overlays are pinned implementation costs.
Every Go upgrade runs the typed `gomadtool upgrade-dossier` host command, which records the
complete upstream patch, semantic boundary diff, interception evidence,
archive-based overlay collision audit, disabled-mode upstream compatibility,
mandatory probes, optional retained-corpus evidence, and platform qualification
in one JSON dossier. The supported-host gate must also rerun the
platform's host-clock inventory and, on Darwin, the positive-controlled clock
trace because dynamic imports and probe names are platform implementation
details. The dossier is published on failure and uploaded by CI, so a rejected
upgrade retains its first failing gate and bounded output. Publication uses the
shared host-filesystem replacement primitive, so a failed write, file or
directory synchronization, rename, or temporary-file cleanup is returned as a
host error ahead of the gate result, and a partial dossier never replaces the
prior complete one. Boundary comparison
canonicalizes complete manifest metadata, intercepts, and hook policies so a
field unknown to an older comparator cannot disappear
from upgrade evidence.

Broader runtime or compiler changes require a minimized real workload showing
that Runner, World, adapters, and records cannot satisfy the contract.
Deterministic GC control, compiler checkpoints, multi-P execution, DPOR, and
preemption bounding remain separate research projects rather than guarantees
of bounded choice tracing and exploration.
