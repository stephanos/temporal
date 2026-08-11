# Gomad After v3

## Purpose

Gomad v3 establishes a narrow runtime contract: with deterministic external
inputs, a fixed patched Go 1.26.4 toolchain, program, architecture, and seed
produce repeatable runtime-controlled scheduling, `select`, map, and proven
synchronization choices.

This document begins after that experiment. It outlines the work required to
turn the runtime primitive into a useful deterministic execution environment
and exploration workflow. It does not repeat v3 testing gaps; those are tracked
in `GOMAD_TESTS.md`.

## Starting assumptions

- V3 remains opt-in and side-by-side with the repository's normal Go toolchain.
- `GOMADSEED` remains the runtime activation switch.
- Programs continue to use one P and native Go goroutines, channels, maps, and
  synchronization.
- The production Go patch remains small. External behavior belongs outside the
  runtime unless a minimized failure proves that an external solution cannot
  satisfy the contract.
- Different seeds run in separate processes to use multiple cores.
- A schedule is reproducible only for the same toolchain, architecture,
  program, deterministic external inputs, and seed.
- The system is for trusted tests, not production execution.

## Architectural direction

Post-v3 work should be organized around four deep modules with small public
interfaces:

1. **Runner** launches one isolated child per seed, enforces limits, and writes
   reproducibility artifacts.
2. **World** owns deterministic external state and an ordered event queue.
3. **Adapters** implement time, filesystem, network, and process boundaries
   against the World rather than the host.
4. **Record** stores the versioned inputs and decisions needed to reproduce or
   replay a failure.

The intended flow is:

```text
seed + command + world snapshot
              |
              v
           Runner
              |
              v
   Gomad process <-> Adapters <-> World/event queue
              |
              v
    result + record + diagnostics
```

This boundary keeps host timing and readiness out of runtime decisions without
requiring the Go runtime to simulate operating systems or public packages.

## Workstream 1: deterministic external event model

### Goal

Define how external events become runnable work without using host completion
order as an input. This is the foundation for virtual time and all I/O
adapters.

### Work

- Inventory external boundaries used by candidate Temporal tests: clocks,
  timers, filesystem access, sockets, DNS, subprocesses, signals, environment,
  and entropy.
- Classify each boundary as deterministic input, simulated state, explicitly
  unsupported behavior, or an operation that must occur outside the
  deterministic region.
- Define a World event with a logical timestamp, event kind, stable resource
  identity, payload, and deterministic tie-break sequence.
- Define how operations register interest, how events become ready, how
  cancellation removes interest, and how completion is delivered.
- Make quiescence explicit. Do not infer it from elapsed wall time.
- Specify bounded queues and deterministic failure behavior for capacity,
  invalid operations, and deadlock.

### Design constraints

- Event ordering must not use pointers, goroutine IDs, host timestamps, map
  iteration, or OS callback order.
- A single seed may break ties only after all semantic ordering rules apply.
- The World derives domain-separated external-choice state from the configured
  seed; it never consumes the runtime's private random stream.
- The same World implementation must serve every adapter; each boundary must
  not invent its own scheduler or random stream.
- The event API must be testable without the custom Go toolchain.

### Exit criteria

- A pure in-memory test can register, cancel, order, and deliver competing
  events deterministically.
- Snapshot plus seed reproduces the same event sequence across processes.
- Capacity errors, deadlock, cancellation races, and invalid resource access
  have stable results.
- The model identifies where runtime readiness begins without adding a runtime
  hook prematurely.

## Workstream 2: virtual time

### Goal

Remove wall-clock timing and host timer delivery from supported deterministic
tests.

### Work

- Add a virtual clock to the World with explicit current time.
- Model one-shot timers, sleeps, deadlines, tickers, stop, reset, and
  cancellation.
- Order simultaneous deadlines deterministically through the shared event
  model.
- Advance time only through an explicit driver operation or a proven
  quiescence protocol. Do not poll wall time to decide when to advance.
- Define overflow, zero-duration, negative-duration, and far-future behavior.
- Provide adapter interfaces that application code can receive through normal
  dependency injection before considering standard-library interception.

### Trade-off

Explicit clock injection requires application seams but keeps the runtime patch
small and the time model independently testable. Intercepting `time` or runtime
timers would support more unmodified programs, but it couples Gomad to unstable
runtime timer internals and should require a concrete adoption blocker.

### Exit criteria

- Timer-heavy tests complete without sleeping or reading host time.
- Same-time timer races repeat for a seed and vary across seeds only where the
  model permits a choice.
- Advancing virtual time cannot skip runnable deterministic work.
- Deadlock and “no future events” are distinguishable outcomes.

## Workstream 3: deterministic external adapters

### Filesystem

Implement an in-memory filesystem backed by an explicit initial snapshot.
Model path normalization, files, directories, metadata, permissions, atomic
rename, failure injection, and deterministic directory enumeration. Host file
access must be an import/export operation outside the deterministic region.

### Persistence

Prefer existing in-memory Temporal persistence implementations where their
semantics are sufficient. Otherwise model requests, transactions, iteration,
conflicts, failures, and completion as World operations. Initial database state
must be part of the snapshot, and no deterministic assertion may depend on a
real database's response timing or unspecified row order.

### Network

Model endpoints, connections, messages, delivery, reordering, duplication,
loss, partition, and closure as World events. DNS and address assignment must
come from the snapshot. Do not reuse host sockets or readiness notification
inside a deterministic run.

### Processes

Represent allowed commands as registered deterministic handlers or simulated
processes with explicit stdin, stdout, stderr, exit status, and lifecycle
events. Arbitrary host subprocess execution remains outside the contract.

### Environment and entropy

Capture environment values in the World snapshot and expose a controlled
entropy source for application-level randomness. This source is separate from
the seeded private runtime stream and must have an explicit ownership model.

### Common requirements

- All adapters use the same event queue and cancellation semantics.
- Results are bounded and serializable into a reproduction artifact.
- Fault injection is data, not control flow hidden in a test.
- Adapters validate logical state separately from scheduling order.
- Security-sensitive host capabilities are denied by default.

### Exit criteria

- Representative tests run with no host clock, file, socket, DNS, or process
  readiness inside their deterministic region.
- Each adapter has standalone semantic tests and cross-adapter event-ordering
  tests.
- A saved World snapshot reproduces success and failure cases.

## Workstream 4: seed exploration and failure minimization

### Goal

Turn the one-seed runtime interface into a bounded exploration tool without
changing runtime activation or running multiple seeds in one process.

### Runner responsibilities

- Launch a fresh process for every seed.
- Run seed ranges and explicit seed sets with bounded cross-process
  parallelism.
- Apply per-run and overall timeouts and kill complete process trees.
- Capture stdout, stderr, exit status, seed, command, arguments, environment
  contract, architecture, and Gomad toolchain build key.
- Group equivalent failures by stable result signature.
- Stop on first failure, collect all failures, or enforce a configurable
  failure budget.
- Write artifacts atomically so a crashed runner cannot publish a complete
  record for an incomplete execution.

### Minimization

A seed is an opaque selector, not a value that can be numerically minimized.
Minimize the reproducible case instead:

- deterministic external input and World snapshot;
- event and fault-injection set;
- test selection and application arguments; and
- eventually, a recorded choice sequence if replay support justifies it.

Use delta debugging only when every candidate is rerun enough times to prove it
still reproduces under the same contract.

### Scale and failure handling

- Stream records to disk rather than retaining all output in memory.
- Bound concurrent children independently of the one-P runtime constraint.
- Deduplicate artifacts by content hash.
- Preserve partial diagnostics when a worker crashes or times out.
- Treat a host failure separately from a deterministic program failure.

### Exit criteria

- A developer can run a bounded seed set and receive one self-contained
  artifact per distinct failure.
- Re-running an artifact invokes exactly one seed and reproduces the same
  result.
- A 10x larger seed set increases work approximately linearly without growing
  per-run memory or leaking children.

## Workstream 5: versioned records and replay

### Goal

Make failures portable and diagnosable without promising replay across changed
programs or toolchains.

### Record contents

- schema version;
- Gomad toolchain build key and Go version;
- target OS and architecture;
- program identity and arguments;
- seed and deterministic environment settings;
- World snapshot identity;
- ordered external events and injected faults;
- stdout, stderr, exit status, timeout, and result signature; and
- optional diagnostic choice information when available without broadening the
  production patch.

### Replay levels

1. **Seed replay** reruns the same program, World snapshot, and seed.
2. **External-event replay** supplies the recorded World events and validates
   that the program requests compatible operations.
3. **Runtime-choice replay** controls internal scheduling choices directly.

Implement levels 1 and 2 first. Level 3 requires runtime observation or
control hooks, stable choice identities, and substantially more maintenance.
Add it only if seed plus external-event replay cannot reproduce a minimized
failure.

### Compatibility

Records are exact-version artifacts. A reader must reject unsupported schema,
toolchain, architecture, program, or snapshot identities rather than silently
attempting approximate replay. Migration tools may copy metadata forward, but
must not claim behavioral equivalence without rerunning the case.

### Exit criteria

- A record is self-describing and validates all required identities before
  execution.
- Seed and external-event replay reproduce known failures.
- Divergence reports the first incompatible external request or result.
- Truncated or partially written records fail clearly.

## Workstream 6: Temporal integration

### Goal

Prove the environment against real Temporal code rather than only synthetic
runtime fixtures.

### Work

- Select a small package whose concurrency matters and whose external
  dependencies can be injected.
- Inventory its clock, persistence, network, process, entropy, and background
  goroutine boundaries.
- Route those boundaries through the post-v3 interfaces without changing its
  production behavior.
- Define a deterministic scenario with multiple legitimate schedules and
  stable semantic invariants.
- Run a seed range, retain the first distinct schedules, and minimize any
  invariant failure.
- Measure runtime, artifact size, explored schedules, duplicate schedules, and
  failure reproduction rate.

### Adoption rules

- Keep the normal test path unchanged and make Gomad opt-in.
- Do not label an integration deterministic while any host readiness source
  remains inside the claimed region.
- Prefer narrow dependency-injection seams over package-wide abstractions.
- Add adapters only for boundaries exercised by the selected pilot.

### Exit criteria

- One real Temporal test exercises multiple schedules and repeats exactly for
  each seed.
- Its external inputs are fully captured by a World snapshot.
- A failure artifact can be replayed by another developer using the same
  toolchain and architecture.
- The integration cost is understood before expanding to more packages.

## Workstream 7: evidence-gated runtime or compiler extensions

The following extensions are not default next steps. Each requires a minimized
failure, an explanation of why the Runner, World, adapters, and records cannot
solve it, and explicit review of production patch cost.

### Domain-separated runtime random streams

Consider separate scheduling, map, and synchronization streams only if harmless
random-consuming operations make failures too unstable to minimize or replay.
This would improve change isolation but would add runtime state and diverge
further from upstream behavior.

### Deterministic GC triggering

Consider an allocation-count or explicitly driven GC trigger only if identical
supported workloads still diverge because host-dependent GC timing changes the
runnable set. Prefer documented heap bounds or explicit test GC first.

### Runtime choice observation and control

Stable choice IDs, traces, and forced decisions may enable runtime-choice
replay and schedule minimization. They also add hot-path instrumentation,
versioned semantics, and larger patches. Seed replay remains preferred.

### Compiler scheduling checkpoints

Compiler-inserted checkpoints could explore CPU-bound loops that never reach a
runtime scheduling point. This changes program execution and compiler output,
increases overhead, and creates a new compatibility surface. Pursue it only
when a real target workload cannot expose necessary choices cooperatively.

### Multiple-P deterministic execution

Deterministic parallel execution would require controlling true concurrent
memory interactions, not only choosing among runnable goroutines. It is a
separate research project and must not be treated as a small extension of the
one-P model.

## Recommended sequence

Two tracks can begin after v3:

```text
External determinism                         Exploration ergonomics
--------------------                         ----------------------
event model                                  seed runner
    |                                             |
virtual time                                failure artifacts
    |                                             |
external adapters --------------------------> versioned records
    |                                             |
Temporal pilot -----------------------------> replay
    |
evidence-gated runtime/compiler extensions
```

The event model and seed runner are independent starting points. Versioned
records should be designed jointly before either track publishes long-lived
artifacts. Runtime or compiler expansion comes only after the Temporal pilot
identifies a concrete blocker.

## Post-v3 success criteria

Post-v3 work has produced a useful deterministic testing system when:

- supported tests receive time and external readiness exclusively through a
  deterministic World;
- separate-process seed exploration is bounded, parallel, and reproducible;
- every failure artifact identifies its exact toolchain, program, seed, and
  external snapshot;
- seed and external-event replay reproduce failures without host timing;
- at least one real Temporal package finds or meaningfully explores a
  concurrency behavior using the system;
- normal production builds and tests remain unchanged; and
- any additional runtime or compiler patch is justified by a minimized
  integration failure rather than speculative capability growth.

## Non-goals

- Deterministic execution of arbitrary unmodified programs that use host I/O.
- Production deployment with deterministic map seeds.
- Exhaustive exploration of data races or CPU instruction interleavings.
- Schedule stability across program, toolchain, or architecture changes.
- Multi-P deterministic parallelism in the initial post-v3 roadmap.
- Windows support.
- Runtime or compiler hooks added solely to make diagnostics more convenient.
