# Gomad v3 Next: Distributed-System Simulation

**Roadmap date:** 2026-08-14

> **Status note:** This is the detailed track design. Current implementation status and cross-track ordering live in [GOMADv3_NEXT.md](GOMADv3_NEXT.md). The module designs, invariants, verification plan, and exit criteria here remain normative.

## Goal

Give Gomad v3 at least the behavioral simulation capabilities of Gomad v2,
then expand the simulation model where representative Temporal workloads show
additional value.

The target is behavioral parity, not source or API compatibility with v2. A
v3-native test may use a different harness, artifact schema, and implementation
while still being able to express and reproduce the same system behavior:

- multiple isolated logical nodes;
- deterministic communication between nodes;
- partitions and link delay;
- graceful stop, harsh crash, and restart;
- per-node durable storage and partial persistence after a crash;
- bounded enumeration of valid storage crash states;
- repeatable nemesis actions; and
- deterministic time, evidence, inspection, and replay across the cluster.

This roadmap is a fourth investment track beside
[bug-finding power](GOMADv3_NEXT_BUG_FINDING.md),
[compatibility](GOMADv3_NEXT_COMPATIBILITY.md), and
[productionization](GOMADv3_NEXT_PRODUCTIONIZATION.md). It depends on the
shared trust gate in [GOMADv3_NEXT.md](GOMADv3_NEXT.md). That gate is complete
and remains a required regression boundary for all simulation work.

## Executive design

Use a tiered hybrid with one execution backend per simulated cluster:

1. **In-process backend:** all logical nodes run in one patched-Go process.
   This is the default because it shares one deterministic scheduler and
   virtual clock, calls simulation models directly, starts quickly, and can
   explore more seeds and faults per compute-hour.
2. **Process-backed backend:** each logical node incarnation runs in a fresh
   patched-Go child process. This fidelity tier is required when a harsh crash
   must discard package globals, arbitrary goroutines, descriptors, allocator
   state, and package initialization state.

Both backends use the same cluster, network, volume, fault, scenario, record,
and replay contracts. They differ only at the execution seam. A cluster does
not mix backend types initially. Every artifact records its backend and replays
only through that backend.

The process backend is not the default and does not make ordinary Gomad v3
tests distributed. It exists because separate processes uniquely provide hard
failure isolation. Network faults, storage faults, client workloads, and most
distributed protocols are faster and simpler to simulate with in-process
logical nodes.

## Terminology

- **Cluster:** one deterministic simulation containing nodes, topology,
  volumes, scenario actors, faults, and one logical environment.
- **Node:** a simulated host or service instance, such as a Temporal frontend,
  history host, matching host, worker, database replica, or client. A node is
  not a physical computer.
- **Incarnation:** one execution lifetime of a node. Restart preserves the
  node identity and attached durable volumes but creates a new incarnation.
- **Runtime domain:** the in-process runtime identity inherited by every
  goroutine created by one node incarnation.
- **Backend:** the adapter that executes node incarnations either in-process or
  in child processes.
- **Model:** deterministic state and semantics outside application memory, such
  as the virtual network or a durable volume.
- **Scenario:** the test-owned actors, lifecycle actions, topology changes, and
  outcome checks applied to a cluster.
- **Fault plan:** a bounded, versioned sequence or search space of modeled
  failures matched against stable operation identities.
- **Oracle:** a module that consumes detached observations or operation
  histories and reports a stable invariant result.

## What parity means

V2 behavior is the baseline, but v3 should preserve only behavior worth
depending on, not accidental v2 implementation details.

| Capability | V2 behavior | Required v3 behavior | Backend needed |
| --- | --- | --- | --- |
| Local concurrency | Seeded cooperative scheduling and virtual time | Existing seeded runtime choices and native virtual time remain deterministic within a node | Both |
| Multiple nodes | Independent machine identity, address, globals, goroutines, disk, and network stack | Stable node identity, per-incarnation execution, node-aware I/O, per-node network and volume state | In-process for explicit node state; process-backed for arbitrary globals |
| Stop | Stop node work, close network gracefully, persist outstanding writes | Bounded graceful shutdown, modeled FIN/close, and declared volume flush | Both |
| Crash | Stop node work without cleanup or implicit disk flush | Revoke the incarnation, drop volatile resources, apply an explicit crash-persistence outcome, and prevent stale work from mutating models | Both; process-backed for hard isolation |
| Restart | Fresh globals and runtime state with retained disk and address | New incarnation with the same node identity, addresses, boot identity, and durable volumes | Process-backed for fresh package globals |
| TCP | Per-node listeners and streams over a virtual network | Deterministic TCP byte-stream behavior across node addresses, including deadlines and close/reset behavior | Both |
| Partition | Disable communication between host groups | Directional link state plus symmetric partition/heal helpers; sends over a disabled direction follow declared drop/reset semantics | Both |
| Delay | Fixed simulated latency between hosts | Directional deterministic link delay using cluster logical time | Both |
| Filesystem | POSIX-like files, directories, handles, rename, truncate, mmap, and fsync | At least the v2 operation set needed by its parity corpus, with explicit errors and bounds | Both |
| Persistence | Memory and persisted views plus dependency-aware pending operations | Durable volume state, volatile view, pending-operation dependency graph, file and directory sync semantics | Both |
| Partial crash | Random valid subset of pending writes may persist | Fault tape selects a dependency-closed crash outcome before restart | Both |
| Crash-state iteration | Iterate possible pending-write subsets | Lazy, canonical, bounded enumeration with resume and capacity outcomes | Both |
| Nemesis | Sequence/repeat random partitions and restart actions | Typed, recorded, replayable lifecycle, topology, and storage-fault plans | Both |
| Diagnostics | Machine/goroutine-aware logs and determinism checks | Node/incarnation-aware output, transition records, choice/fault tapes, stable failure identity, inspect, and exact replay | Both |

The parity claim has two levels:

- **Simulation-model parity:** every v2 network, storage, lifecycle, and nemesis
  case has an equivalent v3 behavior test passing on at least one backend.
- **Hard-isolation parity:** cases that depend on fresh package globals or on
  killing uncooperative/leaked goroutines pass on the process-backed backend.

Do not report complete v2 behavioral parity until both levels pass. In-process
support alone is intentionally useful but insufficient for that claim.

## Non-goals

- Preserving v2 package names, function signatures, translator directives, or
  generated syscall ABI.
- Modeling physical machines or communicating over a live host network.
- Kernel-accurate TCP/IP, filesystem, or hardware emulation.
- Treating either backend as a security sandbox for hostile target code.
- Cross-backend replay of one artifact.
- Mixing in-process and process-backed nodes within one cluster initially.
- True parallel multi-P record/replay or replacement of the race detector.
- Snapshotting arbitrary Go heaps as part of the first parity implementation.
- Automatically widening the model to every `os`, `net`, `syscall`, or
  third-party API.

## Architecture

```text
Scenario / oracle
        |
        v
Cluster ------------------------------------------------- Evidence
  |            |                |             |              |
  |       Node registry    Fault controller   World      artifact/replay
  |            |                |             |
  |            +-------- deterministic models-+
  |                         |           |
  |                      Network      Volumes
  |
  +---- execution seam -------------------------------+
       |                                              |
       v                                              v
 In-process adapter                           Process adapter
 runtime domains                              child processes
 direct model calls                           typed bounded IPC
 shared scheduler/time                        boundary/time arbitration
```

The models own external state. An application node never owns the canonical
network topology or durable volume state. This lets a crash discard an
incarnation without deleting the state that is supposed to survive it.

### System invariants

1. A cluster selects exactly one backend before starting and records it in its
   identity.
2. Node IDs are stable and never reused. Incarnation numbers increase
   monotonically per node.
3. Every modeled operation carries node and incarnation identity. An operation
   from a stale or revoked incarnation fails before reading or mutating model
   state.
4. Node-visible model transitions are validated against replay before they are
   applied.
5. Scheduler, scenario, network, storage, and fault choices use independent,
   domain-separated random streams or tapes. Adding a network choice must not
   silently renumber all storage-fault decisions.
6. All collections, events, and artifacts have explicit count and byte limits.
   Overflow is a typed capacity result, never truncation or host fallback.
7. Host scheduling, pipe-read order, pointers, process IDs, descriptor numbers,
   and wall timestamps never determine modeled behavior.
8. A restart retains only state declared durable by the cluster model. Handles,
   connections, timers, goroutines, and incarnation-local identifiers never
   survive.
9. Unsupported operations fail closed. The simulator never performs ambient
   network or filesystem I/O on behalf of a node.
10. An artifact promises exact replay only for the same target, platform
    bundle, backend, model identities, inputs, limits, and tapes.

## Module design

The modules below are deliberately deep: callers learn a small interface while
each module owns validation, state transitions, bounds, snapshots, recording,
and replay for its domain.

### Cluster module

The Cluster module is the primary interface for tests and the test surface for
end-to-end simulation behavior. Its conceptual entry point is:

```go
Run(ctx context.Context, spec Spec, scenario Scenario) (Result, error)
```

`Spec` contains the backend, seed, limits, initial nodes, boot identities,
addresses, volumes, links, and optional fault plan. `Scenario` receives a
controlled cluster handle for lifecycle and topology actions. `Result` returns
detached observations, histories, transition identities, retained evidence,
and a typed terminal outcome.

The Cluster implementation owns:

- construction and validation;
- node and incarnation state;
- backend selection;
- model composition;
- scenario and fault execution;
- logical completion and deadlock classification;
- evidence finalization; and
- cleanup of every incarnation and model resource.

Callers do not construct the network, volume, backend, or record modules
independently. Internal tests may replace those modules at internal seams, but
those seams should not leak into the Cluster interface.

### Application-facing harness

Temporal tests need a small package in the root module rather than an import of
the nested Gomad tool module. The proposed placement is
`tools/gomadv3sim`. It should have no third-party dependencies and expose only:

- registration of a stable boot identity with a boot function;
- construction of a cluster specification;
- a scenario callback;
- typed node handles and actions; and
- detached observations and terminal results.

A boot registry solves the hybrid problem without serializing function
pointers:

```go
RegisterBoot(id BootID, boot BootFunc)
```

In-process execution invokes the registered function inside a new runtime
domain. Process-backed execution starts the same prepared test binary in a
private node mode and selects the same boot ID. Artifacts record the boot ID,
prepared target identity, and configuration, never a pointer value.

The exact public names should be finalized in SIM-0 after two representative
Temporal harnesses prove the interface. Avoid exposing low-level sockets,
inodes, World requests, IPC frames, or runtime domains through this package.

### Backend seam

The backend seam is real because it has two adapters. Its internal interface
owns incarnation execution, not simulation semantics:

- start one boot identity with a node/incarnation configuration;
- request graceful stop;
- perform harsh crash;
- wait for a typed terminal state;
- exchange modeled-operation requests; and
- release all execution resources.

The in-process adapter uses runtime domains and direct calls. The process
adapter uses Runner-owned launch, supervision, and IPC. Neither backend decides
what a network write, `fsync`, partition, or storage failure means.

### Node registry module

The Node registry owns:

- stable ID allocation and uniqueness;
- labels and configured addresses;
- boot identity and boot configuration;
- attached durable volume identities;
- lifecycle state and current incarnation;
- incarnation-owned listeners, connections, handles, and pending operations;
- transition validation; and
- stale-incarnation rejection.

Externally visible lifecycle states are `defined`, `running`, `stopped`,
`crashed`, and `failed`. Starting or restarting creates the next incarnation.
The implementation may use private `starting` and `stopping` states, but callers
should observe only committed transitions.

Invalid transitions, such as restarting a running node or starting an unknown
boot identity, fail without partially changing state or consuming fault-tape
entries.

### Runtime-domain module

The in-process backend needs a small runtime concept analogous to v2's runtime
machine identity without restoring the source translator.

- A scenario/controller goroutine starts outside all node domains.
- A node boot function enters a new domain carrying node and incarnation IDs.
- Goroutines inherit their parent's domain at creation.
- Compiler-inserted `os`, `net`, clock, entropy, and model hooks resolve the
  current domain.
- Runtime choice traces include the stable domain identity where relevant.
- Crashing a node revokes its domain at a controlled point and makes later
  model calls from it fail as stale.

The runtime cannot generally unwind or safely kill an arbitrary goroutine.
After crash, the in-process adapter must prevent revoked work from re-entering
models and should stop scheduling it at reviewed safe points where runtime
invariants permit. A leaked CPU loop remains a wall-watchdog failure. Package
globals also remain shared. Tests requiring stronger semantics select the
process backend.

### Network module

The Network module owns a deterministic TCP byte-stream model. It does not
claim kernel packet fidelity.

For parity it must implement:

- unique IPv4 addresses and deterministic ephemeral ports;
- interfaces scoped to node identity;
- listen, dial, accept, read, write, half-close/full-close, and deadlines;
- bounded listener backlogs, connection buffers, queued delivery bytes, and
  total resources;
- deterministic connection and delivery identities;
- directional connectivity and fixed directional latency;
- symmetric partition/heal and delay helpers;
- ordered delivery of bytes sent over a healthy stream;
- connection refusal for absent listeners or nodes;
- incarnation-safe delayed delivery;
- graceful node stop with declared close behavior; and
- harsh crash with declared reset/drop behavior.

The parity contract should preserve v2's useful behavior while fixing its
ambiguities. A disabled link drops newly submitted deliveries in that
direction. A topology change does not retroactively reorder a delivery already
committed to the event queue: already queued deliveries remain eligible at
their scheduled time unless their endpoint or incarnation no longer exists.
Delayed data addressed to an old incarnation or connection identity is
discarded and recorded rather than delivered to a reused address or port.

The module presents typed network operations and results. The current `os` and
`net` interception code becomes a standard-library adapter that projects Go
types into those operations. In-process calls enter the model directly;
process-backed calls use the generated IPC adapter.

Network snapshots include topology, address and port allocation, listeners,
connections, buffers, scheduled deliveries, limits, and transition digest.
Pointers, channels, runtime timers, and descriptor numbers are excluded.

### Volume and filesystem module

The Volume module owns durable storage semantics. It should deepen the current
in-memory filesystem rather than layer a second persistence model around it.

Each volume contains:

- a persisted state;
- the current incarnation's volatile state;
- open handles and mappings;
- pending data and metadata operations;
- a dependency graph describing valid persistence order;
- capacity accounting; and
- a canonical transition and snapshot identity.

The parity operation set includes files and directories, open/close, positional
and sequential read/write, append, create/exclusive/truncate, stat, directory
iteration, remove, rename, working directory, sync, and the mmap behavior used
by the v2 corpus. Errors must use stable modeled classes corresponding to the
Go/Unix behavior under test.

Persistence rules are explicit:

- a normal write updates the volatile view and appends pending operations;
- file sync persists the required data, size, allocation, and dependencies for
  that file according to the declared model;
- directory sync persists required namespace changes for that directory;
- graceful stop flushes every pending operation;
- harsh crash discards the volatile view and rebuilds it from a selected valid
  dependency-closed subset of pending operations applied to persisted state;
- restart starts with no open handles or mappings; and
- no crash outcome may contain an operation without all required predecessors.

Crash-state enumeration is lazy and canonical. It must accept maximum state,
operation, depth, byte, and wall budgets. Reaching a bound returns a capacity
outcome and a resumable frontier; it never silently samples while claiming
enumeration. Copy-on-write state and content-addressed chunks should keep
branching from copying complete volumes for every candidate.

Read-only host mounts remain explicit imported inputs. They are captured into
the volume model and replayed from the artifact; a restart or process-backed
node must never reopen the original host path.

### Shared model source and endpoint adapters

The patched standard library cannot casually import the host Runner packages.
Do not maintain hand-copied network or filesystem implementations. Define one
dependency-free canonical model source and generate or compile equivalent
target-side and host-side packages from it, like the existing binary protocol
generation. Validation must reject drift.

The target endpoint supplies Go standard-library adapters. The host endpoint
supplies process-broker adapters and artifact inspection. Golden transition,
snapshot, malformed-input, capacity, and fuzz tests run against both generated
endpoints.

### World and logical-time module

World remains the pure event-ordering and replay module. Network and Volume own
their semantics and use World only for external readiness, cancellation,
logical delivery times, and stable transition ordering.

In-process clusters already share one patched runtime and one native virtual
clock. Model events should use a native timer wake-up for the earliest World
event, then deliver every event at that instant in stable World order. Native
application timers remain runtime-owned. Equal-time competition must be
recorded through the existing or planned choice mechanism rather than host
callback order.

Process-backed clusters require limited cross-process arbitration because each
process otherwise advances time independently and host scheduling changes the
outcome. This coordination is restricted to shared model and time seams:

1. a node parks when it submits a modeled operation or reaches local
   quiescence;
2. it reports its operation or earliest native timer with node-local ordinal;
3. an epoch waits until every live participant is parked, terminal, or
   explicitly unable to affect the current logical instant;
4. the coordinator orders pending model operations from stable logical
   identities and a recorded choice when operations are intentionally
   equivalent;
5. when no participant is runnable, it advances the cluster to the earliest
   node timer or World event and makes every event at that instant eligible;
6. selected results wake only the affected participants.

There is no shared application memory and no instruction-level lockstep.
Ordinary local computation remains isolated inside each process. A CPU loop or
unsupported host block prevents a quiescence epoch and ends in a typed watchdog
observation.

### Fault-controller module

The Fault controller owns fault matching, selection, bounds, recording, replay,
and shrinking. A fault matches stable fields such as:

- node and incarnation or a declared node class;
- model and resource identity;
- operation kind;
- occurrence ordinal;
- scenario phase; and
- optional equivalence class.

It must record the planned and realized fault independently. Exact replay
validates the next expected fault before applying it. A missing, extra,
reordered, inapplicable, or changed fault is a replay divergence.

Parity actions are:

- graceful stop;
- harsh crash;
- restart with persisted-only or selected partial volume state;
- directional disconnect and reconnect;
- symmetric partition and heal;
- fixed directional delay;
- repeat and sequence; and
- deterministic selection of a target node or partition grouping.

Initial beyond-v2 actions are typed adapter errors, delayed readiness,
cancellation, dropped modeled delivery, and declared capacity exhaustion. More
realistic network and storage faults come only after the parity models are
correct and useful workloads justify their state-space cost.

### Scenario and oracle modules

Scenarios own workload intent, not scheduler or model internals. A scenario may:

- start client actors;
- apply node lifecycle actions;
- change topology or delay;
- schedule a fault plan;
- wait on logical conditions;
- query detached model or application observations; and
- submit histories to an oracle.

Avoid a stringly typed chaos DSL. Actions and observations use versioned types
with stable identities. Sequence, repeat, choose, and bounded parallel
composition are library functions over those types.

An Oracle module receives detached values and returns a result containing a
stable invariant name, pass/fail state, and bounded evidence. It must not hold
model locks, run callbacks inside World, or read host time. Initial oracles cover
state invariants, eventual convergence within logical time, absence of
unexpected duplicate/lost operations, and exact expected histories. A later
linearizability checker should be a separate bounded module with explicit
operation and search limits.

### Record and artifact modules

A cluster artifact extends the current exact-replay envelope with:

- cluster schema and backend identity;
- target and platform-bundle identity;
- node specifications, boot identities, and incarnation history;
- initial topology and volume identities;
- model versions and limits;
- scheduler, scenario, network, storage, and fault tapes;
- canonical network, volume, lifecycle, and World transitions;
- planned and realized faults;
- node/incarnation-labelled stdout and stderr hashes plus bounded output;
- scenario observations and oracle results;
- terminal model snapshots and digests; and
- normalized failure identity.

Replay validates all static identities and initial snapshots before starting.
It validates each tape entry and transition before mutation, then requires tape
exhaustion and matching terminal snapshots, histories, output hashes, and
outcome. Inspect reports both human and stable JSON projections without
reimplementing model semantics.

## Lifecycle semantics

### Define and start

A Node specification declares a stable ID or lets Cluster allocate one,
label, boot ID, addresses, volume attachments, environment, and resource
limits. Definition is pure validation. Start commits a new incarnation only
after boot inputs and backend resources are ready. Partial startup becomes an
explicit failed incarnation and is cleaned up before another start.

### Graceful stop

Graceful stop:

1. records the stop request;
2. asks the boot function to shut down;
3. waits within a logical shutdown bound;
4. closes listeners and connections according to graceful network semantics;
5. persists all pending modeled volume operations;
6. verifies owned resources are closed or records bounded leak diagnostics; and
7. commits `stopped`.

A missed logical shutdown bound is not silently converted to a crash. The
scenario or fault plan must explicitly choose whether to follow with a harsh
crash.

### Harsh crash

Harsh crash is one atomic modeled transition with an explicit storage outcome.
It revokes the incarnation, prevents new operations from it, drops
incarnation-local handles and network endpoints, applies the selected valid
volume crash state, and commits `crashed`.

The in-process backend does not run application cleanup and quarantines revoked
runtime-domain work at reviewed safe points. It cannot promise arbitrary global
reset or reclamation of every leaked goroutine. The process backend terminates
and reaps the node process group without running application cleanup.

### Restart

Restart is valid only from `stopped`, `crashed`, or a cleanly classified failed
incarnation. It preserves the stable node ID, configured addresses, boot ID,
and attached durable volume identities. It allocates the next incarnation,
starts with no old descriptors/connections/handles, and invokes the boot
function again.

Changing a boot ID is allowed only while stopped or crashed and is itself a
recorded configuration transition.

## Exploration

Simulation adds several choice dimensions. Explore them in this order:

1. seed sampling;
2. runtime choice-tape recording and exact replay;
3. bounded alternative schedule prefixes;
4. bounded scenario and fault alternatives;
5. dependency-valid storage crash states;
6. combined schedule/fault frontier; and
7. schedule, fault, and typed scenario-input minimization.

The Explorer module owns the frontier and budgets. Every candidate starts a
fresh cluster from the same immutable specification and consumes explicit
tapes. Initial parity does not snapshot a live Go heap; replay-from-start is the
correct baseline.

Search plans record maximum runs, runtime-choice depth, scenario choices,
faults per run, crash states, frontier bytes, logical duration, wall duration,
parallel clusters, retained artifacts, and total bytes. Frontier exhaustion is
a successful bounded result. Hitting a declared limit is a distinct capacity
result with resumable state.

Minimization preserves a normalized failure identity and may:

- remove a suffix of forced runtime choices;
- replace forced choices with their seeded defaults;
- remove or simplify fault actions;
- reduce partition membership or fault duration;
- reduce selected valid storage operations; and
- invoke an opt-in typed scenario-input shrinker.

Do not claim general schedule completeness, input shrinking, or causal
minimality.

## Beyond-v2 simulation gaps

Parity is the first commitment. The following gaps should be measured and
designed now, then implemented in workload-value order.

### Network and topology

- asymmetric links as a first-class interface rather than only symmetric
  helpers;
- packet or segment loss, duplication, reordering, and corruption;
- latency distributions, jitter, queueing, bandwidth, and backpressure;
- half-open and black-holed connections;
- DNS and service discovery from explicit versioned records;
- multiple interfaces, subnets, routing, and NAT when a real scenario needs
  them;
- Unix-domain streams, UDP, and IPv6 as separate models rather than flags on
  TCP; and
- proxy, load-balancer, and connection-pool adapters for representative
  Temporal deployments.

Loss and reordering require a richer transport model than v2's fixed-delay
stream. They must not be approximated by arbitrary byte deletion that violates
the declared TCP abstraction.

### Storage

- `ENOSPC`, quota, inode exhaustion, `EIO`, and read-only transitions;
- torn or short writes at declared atomic units;
- latent corruption and checksum mismatch;
- device detach, remount, snapshot, and restore;
- permissions, hard links, symbolic links, locks, and metadata used by real
  Temporal dependencies;
- multiple volumes per node and shared/remote volume adapters; and
- purpose-built deterministic object-store, queue, or database adapters where
  filesystem emulation is the wrong abstraction.

Every storage fault must state its persistence and atomicity model. “Like
Linux” is not a sufficient contract.

### Node and environment faults

- pause/resume without restart;
- boot failure and repeated crash loops;
- per-node wall-clock offset and drift while preserving a deterministic
  monotonic clock contract;
- CPU scheduling quotas or stalls;
- memory, file-descriptor, goroutine, listener, and connection pressure;
- controlled entropy exhaustion; and
- rolling restart and upgrade scenarios with different boot identities.

True multi-P and weak-memory behavior remain a separate research project. A
CPU quota model must not imply that single-P execution covers hardware races.

### Scenario generation and correctness

- typed workload actors for Temporal client, worker, and persistence actions;
- generated but replayable operation sequences;
- typed shrinkers supplied by the scenario owner;
- bounded linearizability and serializability histories;
- convergence and reconciliation oracles;
- state-hash deduplication that excludes unstable identifiers;
- partial-order reduction only after choice and operation histories are
  trustworthy; and
- differential scenarios against a real local Temporal cluster where the
  comparison contract is meaningful.

### Model fidelity and support evidence

Each model publishes:

- supported operations and explicit simplifications;
- capacity and performance limits;
- positive and negative conformance fixtures;
- replay and host-escape canaries;
- workloads unlocked by the model; and
- known semantic differences from the host platform and v2.

Support reporting separates “scenario expectation matched,” “model operation
supported,” “backend fidelity supported,” and “workload completed.”

## Error handling and failure domains

Keep these outcomes distinct:

- **Node/application failure:** a boot function exits unexpectedly, panics, or
  returns a target error.
- **Scenario failure:** an actor or oracle reports a stable invariant failure.
- **Modeled fault outcome:** a planned error, drop, crash, partition, or
  resource failure was applied correctly.
- **Modeled capacity:** a network, volume, history, frontier, or other declared
  model limit was reached before partial mutation.
- **Unsupported model operation:** the target crossed a reviewed but unmodeled
  interface and failed closed.
- **Unsupported backend fidelity:** a scenario requires hard global or process
  isolation while using the in-process backend.
- **Replay divergence:** the next operation, choice, fault, transition, or
  terminal snapshot differs from the artifact.
- **Watchdog observation:** runnable CPU work or unsupported blocking prevented
  logical progress.
- **Runner/coordinator failure:** preparation, IPC, supervision, containment,
  persistence, or cleanup failed, so no simulation claim can be made.

Important failure handling:

- A process dying during an IPC operation produces an incomplete-incarnation
  result; the model operation is either uncommitted or committed with a durable
  transition ordinal, never ambiguous.
- A crash concurrent with a model request is ordered before mutation through
  the Cluster transition log.
- A stale incarnation cannot consume replay or fault entries.
- A leaked in-process goroutine that attempts model I/O receives a stable stale
  error; one that spins is bounded by the wall watchdog.
- Invalid crash-state dependencies fail model validation rather than producing
  a corrupt volume.
- Artifact or journal write failure preserves the last valid state and reports
  infrastructure failure.
- Cleanup signals only independently verified process identities.

## Performance and scalability

### Expected costs

The in-process backend has the best campaign throughput: one toolchain process,
one scheduler, direct model calls, shared immutable chunks, and no per-node IPC.
Its main costs are larger runtime state, global model locks, event queues, and
the inability to reclaim arbitrary crashed-node heap state.

The process-backed backend pays for one process per live node incarnation,
process startup, pipes or shared memory, output capture, quiescence epochs,
process-group cleanup, and larger artifacts. It should be selected by explicit
fidelity need, not used as the default benchmark path.

### Ten-times-load behavior

At 10× nodes, connections, pending writes, or faults:

- Cluster and model limits reject work before allocation exceeds the plan;
- event and transition records stream into bounded segmented storage;
- network queues use per-link and global byte/count budgets;
- volumes use content-addressed chunks and copy-on-write snapshots;
- crash-state enumeration remains lazy and resumable;
- the Explorer applies backpressure rather than materializing the full
  schedule/fault cross-product;
- node/process concurrency stays below an explicit limit;
- process-mode descriptors and output have aggregate budgets; and
- inspection streams large histories rather than decoding them all into
  memory.

Benchmarks must report operations per simulated second, seeds per wall second,
bytes per transition, memory per node/connection/volume, process startup cost,
IPC round trips, and replay overhead. Performance regressions are evaluated per
backend rather than averaged together.

## Security and data handling

Both backends remain trusted-test tools. Runtime domains and child processes
are failure-isolation mechanisms, not protection against hostile code. Raw
syscalls can bypass reviewed standard-library adapters unless separately
blocked by the host sandbox.

Cluster artifacts can contain node output, client inputs, network payloads,
volume contents, imported mount data, operation histories, and fault plans.
They inherit the productionization roadmap's classification, quota, retention,
redaction, export, and cleanup policy. Full payloads should be retained only
when exact replay requires them; human reports should default to hashes and
bounded summaries.

## Verification plan

### Pure model tests

1. Node lifecycle state-machine property tests covering every valid and invalid
   transition.
2. Network unit and property tests for address/port exhaustion, backlog,
   stream ordering, partial reads/writes, deadlines, close races, delayed stale
   delivery, partition/heal, and capacity rollback.
3. Volume tests for namespace operations, handles, rename, truncate, mmap,
   file/directory sync, dependency closure, crash application, and resource
   limits.
4. Fault-controller tests for matching, domain-separated selection, replay,
   unused/extra/reordered faults, and shrinking.
5. Snapshot and codec golden, malformed-input, allocation-bound, and fuzz
   tests for every model.

### Behavioral parity corpus

Translate v2 behavior—not its package structure—into named black-box cases:

- independent node addresses and lifecycle;
- deterministic multi-node request/response;
- fixed link latency;
- partition, timeout, heal, and reconnect;
- graceful stop versus harsh crash connection behavior;
- restart with durable state and fresh volatile state;
- file and directory sync behavior;
- rename and truncate crash dependencies;
- random valid partial persistence;
- bounded enumeration of all valid small crash states;
- repeated partition and random restart nemeses; and
- same-seed equality plus different-seed diversity.

Each case documents the v2 source test or behavior it replaces, the stronger v3
contract, supported backends, and any intentionally rejected v2 ambiguity.

### Backend conformance

Run the same scenario corpus on both backends wherever both claim support.
Compare detached model transitions, topology, volume snapshots, operation
histories, oracle results, and normalized outcomes. Do not require identical
runtime choice tapes, raw logs, process exits, or timing diagnostics.

Add process-only cases proving:

- package globals reset after restart;
- leaked goroutines and descriptors disappear on crash;
- package initialization runs once per incarnation;
- process groups are killed and reaped;
- a child crash during every IPC phase cannot ambiguously commit a model
  transition; and
- host scheduling/load does not change the modeled transition digest.

### End-to-end replay and recovery

1. Change each schedule, scenario, network, storage, and fault tape entry and
   prove replay rejects the first mismatch before mutation.
2. Interrupt cluster execution and artifact publication at every journal
   boundary, then inspect/resume or recover the last valid state.
3. Corrupt or remove node binaries, volume snapshots, transitions, and model
   identities and prove open fails before execution.
4. Repeatedly crash/restart nodes while output, network, and storage operations
   are active.
5. Run 10× soak cases for nodes, connections, pending volume operations,
   process churn, descriptors, histories, and artifact growth.

### Temporal qualification

Qualification should progress from mechanisms to representative systems:

1. two small service instances communicating over the modeled network;
2. a Temporal frontend-style pair with independent injected state;
3. worker/frontend or matching/history interactions with retries and timeout;
4. persistence restart and recovery against modeled volumes or an exact
   deterministic adapter;
5. partition/heal with convergence oracle;
6. crash after acknowledged operation with durability oracle; and
7. the same hard-crash case under the process backend to expose global or
   goroutine leaks hidden by graceful construction.

## Delivery roadmap

### SIM-0: restore trust and define the parity contract — complete

Preserve the completed shared entry gate from `GOMADv3_NEXT.md`, including the
TCP close/data, World replay, and network resource-limit guarantees. Create the
v2-derived behavioral parity manifest and two representative
Temporal scenario prototypes. Use them to finalize the small application-facing
harness, boot registry, cluster schemas, fidelity declarations, and limits.

The canonical `gomadv3.simulation-parity/v1` manifest now maps thirteen
v2-derived behaviors to named planned v3 cases, exact source tests, explicit
replacement decisions, delivery stages, and backend/fidelity requirements.
The root `tools/gomadv3sim` package defines the bounded
`gomadv3.simulation-spec/v2` application contract, stable node/incarnation
handles, fail-closed boot registration, and cluster lifecycle seam. Private
contract tests originally expressed the required request/response and restart
prototypes without a backend claim; SIM-1 below now executes those contracts.
They still do not satisfy the network or storage parity cases assigned to
later stages.

Exit criteria:

- every claimed v2 behavior has a named expected v3 case;
- ambiguous or incorrect v2 behavior is explicitly replaced, not copied;
- the harness can express one two-node request/response and one restart case;
- backend and fidelity requirements are machine-readable; and
- no new model code is built on a known false replay or TCP guarantee.

### SIM-1: cluster core and in-process runtime domains — complete

Implement Cluster, Node registry, runtime-domain identity/inheritance, boot
registry, lifecycle state machine, node-aware transparent hooks, stale
incarnation revocation, node-labelled output, and base cluster records. Start
with no cross-node network or crash-persistent storage beyond minimal fixtures.

The production `tools/gomadv3sim.Run` path now owns stable node configuration,
monotonic incarnations, concurrent boot execution, validate-before-commit
lifecycle transitions, authenticated cluster replay plans, bounded canonical
records, per-incarnation terminal results, and context-bounded teardown. The
pinned runtime propagates opaque node domains through ordinary goroutine
creation; `os.Hostname`, stdout, and stderr resolve the active incarnation,
and revocation is atomic with output admission. Both SIM-0 prototypes execute
through this path and exactly replay their lifecycle and terminal evidence.

The in-process fidelity limits remain explicit: package globals are shared,
revoked crash work may remain computationally live, CPU loops require an outer
watchdog, and hard isolation requires the process backend. SIM-1 does
not claim cross-node TCP, durable volumes, partitions, crash persistence, or
fresh package initialization.

Graceful Stop checks caller cancellation before admission. After it delivers
cancellation to arbitrary boot code, it waits for a terminal commit so a
context error never hides a partially stopped incarnation. A boot that ignores
cancellation is therefore an outer-watchdog failure in this tier; bounded hard
termination belongs to the process backend.

Exit criteria:

- multiple boot functions run concurrently in distinct node domains;
- I/O hooks resolve the correct node/incarnation;
- stop, crash, restart, and wait transitions replay exactly;
- stale incarnations cannot mutate model state;
- the limitations around shared globals and leaked CPU loops are enforced and
  reported; and
- ordinary single-process Gomad v3 behavior is unchanged when cluster mode is
  disabled.

### SIM-2: virtual network parity — complete

Deepen the current loopback network into the node-aware Network module. Add
multi-address TCP, deterministic ports and resources, link topology, fixed
delay, partition/heal, lifecycle behavior, snapshots, recording, replay, and
inspection.

The in-process backend now intercepts ordinary `net` TCP operations inside a
runtime domain and routes them through a run-scoped network. Nodes retain
independent addresses and deterministic port sequences; links carry explicit
enabled state and delay; listeners, connections, queued deliveries, bytes, and
transitions are bounded before mutation. Partition/heal and delay changes are
recorded topology operations. Every connection and delayed delivery carries
both endpoint incarnations, so crash revocation resets established operations,
drops stale queued data, and cannot deliver it after restart. Graceful stop
produces close/EOF behavior instead of a crash reset.

Cluster record and replay schemas v2 contain canonical causal-lane history and
a digest-bound final network snapshot. Replay matches each semantic operation
against the next transition for its resource before mutation. Topology changes
share one ordered lane, while host arrival order between independent resources
is normalized. Missing, changed, or causally reordered transitions fail, and
exact final state is checked. Endpoint, payload, delay, outcome, resource, and
incarnation identity are all part of the match.

Exit criteria:

- every network parity case passes in-process;
- same-seed transition digests repeat under host load;
- delayed data cannot cross incarnation identity;
- all close/data/dial/accept/deadline races have adversarial tests;
- limits fail before partial mutation; and
- disabled cluster mode preserves qualified upstream behavior.

### SIM-3: durable volume parity — complete

Deepen `gomadfs` into the per-node Volume module. Add persisted and volatile
views, dependency-aware pending operations, sync semantics, graceful flush,
crash selection, lazy crash-state enumeration, copy-on-write snapshots,
recording, replay, and inspection.

The in-process backend now gives every mounted volume separate persisted and
volatile views with explicit allocation, content, resize, metadata, and
namespace dependencies. File sync and directory sync persist distinct
dependency closures, graceful stop flushes all pending operations, and crash
uses a deterministic dependency-valid selection. Crash-state enumeration is
bounded by state, operation, depth, byte, and wall budgets; incomplete pages
carry an identity-bound canonical frontier and resume without changing order.

Volume transitions and final snapshots are independently digest-bound in the
cluster record and replayed before mutation. Lifecycle changes fork the durable
state, revoke handles and read-only mappings with `ESTALE`, and clear retained
mapping bytes. The exact modernc libc adapter exposes bounded read-only shared
mmap/munmap behavior without a host syscall. Captured read-only mount replay is
constructed only from retained inputs and contains no host source path.

Exit criteria:

- every storage parity case passes in-process;
- all enumerated crash states are dependency-valid and small fixtures are
  complete;
- partial persistence is exactly replayable;
- restart drops handles and mappings but retains selected durable state;
- mount replay never reopens host input; and
- bounded enumeration can stop and resume without changing order.

### SIM-4: scenarios, nemeses, records, and oracles — complete

Add typed scenario composition, parity nemeses, Fault controller, stable
operation histories, initial oracles, complete cluster artifacts, inspect
projections, and exact replay from cluster start.

The v4 cluster contract now composes typed sequence, repeat, deterministic
choice, and bounded-parallel scenario steps. Versioned fault plans bind stable
match fields and controller-owned occurrence ordinals, select candidate targets
through a domain-separated function, record planned and realized actions
independently, and reject missing, extra, reordered, changed, or inapplicable
faults before their model mutation. Directional disconnect/reconnect/delay,
atomic group partition/heal, graceful stop, persisted-only or selected-partial
crash, and prior-target restart all have patched-runtime coverage.

Cluster records bind static target/platform identity, independent lifecycle,
network, volume, fault, scenario, history, observation, oracle, and output
evidence, normalized failures, bounded inspection projections, and terminal
model snapshots. A representative Temporal matching retry scenario uses the
production `common/collection.SyncMap`, loses an acknowledgement across an
injected directional disconnect, observes duplicate task delivery, fails the
duplicate/lost oracle, and exactly replays the same failure identity and
bounded artifact. The three SIM-4 parity cases now carry named in-process
prototypes; SIM-5 supplies the process-backed evidence.

Exit criteria:

- partition/restart and partial-disk nemeses reproduce from artifacts;
- unused, extra, reordered, and changed faults diverge before application;
- at least one Temporal scenario finds and replays a failure path;
- invariant failures have stable identities independent of seed; and
- cluster artifacts stay within declared limits at the qualification size.

### SIM-5: process-backed fidelity tier — complete

Add Runner-owned node launch and supervision, private node bootstrap, generated
bounded operation protocols, host-side models, time and model arbitration,
hard crash/reap, fresh-incarnation initialization, process-aware output, and
cross-backend conformance.

The Runner now launches each process-backed incarnation through a private,
bounded bootstrap and routes generated model operations to the host-owned
network, volume, lifecycle, and time arbiters. Wait admission, in-flight model
operations, crash, response delivery, and time advancement have explicit
acceptance and completion states, so process death cannot leave an ambiguous
model commit. Hard crash reaps the exact incarnation and its descendants;
restart receives fresh package globals, goroutines, and descriptors while
retaining only declared model state. Root process tests cover shared TCP,
durable restart, clock synchronization, hard isolation, detached-model
equivalence, completion-order-independent digests, and deterministic draining
of operations interrupted by crash.

Exit criteria:

- process-only global/goroutine/descriptor reset cases pass;
- shared model cases produce equivalent detached outcomes on both backends;
- node process death cannot ambiguously commit an operation;
- host load and process completion order do not change model digests;
- all descendants are bounded, terminated, and reaped; and
- the complete v2 behavioral parity corpus passes on its declared backend.

SIM-5 completes the evidence required to claim full v2 behavioral simulation
parity at the declared fidelity tiers.

### SIM-6: controlled schedule and fault exploration

Integrate runtime choice tapes from the bug-finding roadmap with scenario,
network, storage, and fault tapes. Add bounded combined frontiers, durable
resume, crash-state search, semantic deduplication, and minimization.

Exit criteria:

- a benchmark corpus shows controlled exploration reaches alternate distributed
  outcomes more efficiently than seed-only sampling, or clearly demonstrates
  where it does not;
- frontiers survive interruption without rediscovering completed candidates;
- minimization reduces at least one schedule-plus-fault reproduction while
  preserving its failure identity; and
- all search dimensions and remaining work are inspectable and bounded.

### SIM-7: evidence-driven expansion beyond v2

Run the Temporal corpus to rank missing simulation semantics by workloads
unlocked. Add the smallest deep model that provides the highest value, starting
from the beyond-v2 gap inventory above. Each addition needs a semantic
contract, host-escape canary, exact replay, failure and capacity tests,
performance evidence, and named Temporal consumers.

Do not implement broad UDP, DNS, routing, storage corruption, resource
pressure, or external-service emulation merely for API-count parity.

## Adjacent v2 gaps

V2 also has behavior around metatesting, trace checksums, race-aware runtime
support, translated global duplication, and Go-test presentation. This roadmap
addresses their simulation consequences as follows:

- same-seed equality and different-seed diversity become qualification and
  parity-corpus requirements;
- trace checksums are superseded by typed choice and model transition digests;
- translated global duplication is provided behaviorally only by the
  process-backed tier;
- node-aware logs and standard Go-test results are part of Cluster evidence;
- race-detector support remains a separate runtime/toolchain research item and
  must be reported as unsupported rather than implied by single-P simulation.

The project may claim v2 behavioral **simulation** parity. It must
not claim complete product feature parity while a relied-upon adjacent feature,
such as race mode, remains unsupported.

## Recommended next slice

Implement SIM-6 behind the completed runtime-choice, lifecycle, network,
volume, fault, scenario, oracle, and process seams. Start with one canonical,
bounded combined-candidate identity and breadth-first frontier, then make its
round checkpoint durable before adding semantic failure deduplication and
schedule-plus-fault minimization.
