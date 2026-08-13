# Gomad v3 TODO

Status snapshot: 2026-08-12.

This is the prioritized backlog for making Gomad v3 simpler to operate,
maintain, and upgrade without weakening its deterministic guarantees. The
[README](tools/gomadv3/README.md) remains the user contract and
[ARCHITECTURE](tools/gomadv3/ARCHITECTURE.md) remains the durable design record.

## Current verdict

Gomad v3 has the right high-level seams: a small runtime patch, compiler-level
standard-library interception, a process-isolated Runner, immutable replay
artifacts, an in-memory I/O implementation, and an explicit World model. The
recent extraction of `BatchJournal`, outcome classification, launch-resource
planning, and safe-file handling improved the module boundaries.

The recommended direction is:

1. close runtime and real-workload validation gaps;
2. improve diagnostics and everyday commands;
3. add semantic guided coverage; and
4. expand modeled capabilities only from observed, minimized workloads.

Do not overlay complete upstream Go files. Whole-file overlays hide the upstream
diff, collide with existing source by design, and turn every Go update into a
manual merge. Continue using the compiler interception seam for ordinary call
redirection, keep patches limited to irreducible runtime/compiler/linker hooks,
and use overlays only for new files. The remaining maintainability problem is
proving that the generated inventory is complete, not the seam itself.

This seam works because Go invocations ultimately bind to compiled function or
method definitions. Prepending one validated prologue to the definition covers
direct calls, method values/expressions, and interface dispatch once it selects
that method, without parsing and rewriting every target package as Gomad v2 did.
It does not cover an undeclared sibling/internal function or a raw host call,
which is why completeness must be proven separately.

## P1: incomplete capabilities and validation

### Requalify real Temporal workloads

The dated functional sweep predates universal deterministic I/O and completion
of lazy read-only mount replay. Its 80/81 SQLite-schema blockers no longer
describe the current first failure.

Rerun the inventory in bounded batches and, for every suite, retain:

- exact target identity and command;
- same-seed repeated result and transcript identity;
- first unsupported boundary or progress blocker;
- replay result for failures; and
- semantic coverage/probe summary.

Do not add capabilities from the old blocker list without reproducing the new
first boundary. Prioritize a small, representative suite set before another
repository-wide sweep.

### Complete only evidence-backed models

Current transparent I/O is intentionally narrower than the standard library:
it covers an in-memory filesystem, entropy, hostname, loopback TCP, read-only
mounts, and one pinned modernc/libc adapter. World currently demonstrates one
mailbox adapter. Missing or incomplete areas include DNS, UDP/Unix sockets,
subprocess semantics, signals, broader foreign runtimes, network fault models,
persistence conflicts, and coordination between external events and native
timer quiescence.

For each observed gap, first decide among:

1. deny it with a stable error;
2. implement a synchronous deterministic shim and transcript;
3. implement a World adapter for readiness, cancellation, or competing events;
4. capture it as an explicit immutable input; or
5. leave it outside the contract and reject the target.

Do not turn the I/O profile into a registry of Temporal test names. Keep models
generic and bind exact adapter versions and implementation digests into replay
identity.

### Finish the testing backlog

Existing tests are strong in several areas: fresh-process repeatability,
enabled/disabled comparisons, timers and logical timeouts, artifact validation,
process-group cleanup, bounded output, protocol goldens, mount replay, compiler
hook validation, and focused upstream packages.

Testing is nevertheless insufficient today because the dated qualification
sweep is stale.

Promote these into CI tiers by cost:

- per-change: validation, unit tests, compiler canaries, focused black-box tests;
- scheduled: stress repetitions, host-load perturbations, syscall audit,
  upstream shards; and
- release/upgrade: full platform matrix, corpus requalification, and Temporal
  suite inventory.

Every behavioral fixture needs a separate semantic oracle. Same output for the
same seed proves repeatability, not correctness; cross-seed diversity proves
variation, not validity.

### Make Go upgrades a generated qualification dossier

Run the generated dossier on supported CI with a retained-corpus report and
keep that artifact as the release qualification record. Local qualification now
produces the source diff, boundary-manifest diff, interception report, checked
overlay-collision report, bounded gate evidence, mandatory-probe policy, and
platform result from the canonical version descriptor. Schedule identity may
legitimately change across versions; semantic invariants and explicit
dispositions may not silently disappear.

## P2: code and module improvements

### Keep shell at the process boundary; move policy into Go

The scripts are shell because the earliest operations are inherently external:
finding a bootstrap Go, downloading and checking a tarball, setting a sterile
environment, applying a patch, invoking `make.bash`, running command fixtures,
and supervising tools before the custom toolchain exists. Rewriting all of
that in Go would introduce a bootstrap binary and reduce command transparency
without removing the need for a shell entry point.

Shell remains a poor fit for lock recovery, fixture metadata, result
classification, and other state machines in `build.sh` and `test.sh`. Continue
moving those typed, unit-testable operations into `internal/hosttool`. Keep thin
shell launchers for environment isolation, download/tool invocation, signals,
and portability; do not do a wholesale shell rewrite.

### Reduce code by removing duplicate knowledge

Raw line count is not the goal. Some large files are deep modules, generated
code, or exhaustive tests. Prioritize deletions and single sources of truth:

- store the modernc adapter body as ordinary checked source/template data
  instead of a 180-line Go string, then validate its digest;
- generate repetitive standard-library hook boilerplate only where operation
  descriptors fully describe error wrapping and transcript behavior;
- centralize version/platform constants rather than repeating Go 1.26.4 and
  `darwin/arm64` in shell, Go, filenames, expected output, and docs;
- replace repeated test command sequences with a typed fixture table; and
- delete obsolete compatibility/profile paths once artifact-schema compatibility
  requirements are documented.

Large-file follow-up should target ownership, not arbitrary splitting:

- `runner.go`: extract the seed-campaign scheduling/aggregation state machine
  behind a small interface; leave preparation and manifest construction with
  their existing owners.
- `process_unix.go`: separate host-side launch orchestration from supervisor
  process-group lifecycle; the existing launch plan is the boundary to preserve.
- `record.go`: separate validation and identity projection into files within the
  same package; do not create forwarding packages.
- `os/gomad.go` and `gomadfs/fs.go`: keep the filesystem engine deep and
  independently tested; reduce adapter translation duplication before splitting
  the engine by file length.
- `test.sh`: move policy and fixture metadata to the host tool while retaining
  small, readable shell entry points.

### Preserve the modular boundaries that are working

The following modules have cohesive ownership and should remain independent:

- runtime: scheduling, maps, synchronization, and native virtual time;
- `gomadfs`/`gomadio`: deterministic synchronous I/O semantics;
- World and its adapters: external readiness, ordering, and snapshots;
- process launch plan/supervisor: descriptors, containment, and reaping;
- target: immutable preparation and provenance;
- `BatchJournal`/artifact/record: durable state, publication, and identity;
- replay: validation and re-execution; and
- outcome: execution-result classification.

Modularity is therefore mostly working. The main leaks are duplicated version
and boundary knowledge, the legacy “profile” terminology for one universal
boundary, Runner's large campaign loop, and adapter implementation embedded as
source-rewrite strings. Avoid a generic plugin framework: immutable registries
and narrow operation interfaces are easier to audit.

### Failure and scale review

At 10× the current seed count, process isolation scales horizontally but target
build cost, artifact disk, captured mount bytes, transcript capacity, and host
process limits become the constraints. Preserve build-once/run-many, bound all
retention, stream aggregation instead of retaining all results, and expose
capacity failures rather than silently dropping evidence.

Crashes already leave partial journals and publish manifests last. Resume must
verify prepared-target identity and every completed run before reuse. Guided
coverage must update its corpus transactionally so a crash cannot mark a probe
as covered without retaining the corresponding reproducible case.

## Guided semantic coverage

Yes, Gomad can reward runs that find new paths, but schedule seeds are opaque:
nearby seeds do not imply nearby executions. The first useful feature is a
coverage-guided corpus, not a numeric seed mutator.

### Coverage model

Track separate, versioned dimensions:

- target code regions or edges when built with Go coverage instrumentation;
- runtime choice sites, choice family, alternative count, and selected class;
- standard-library interceptor and result class;
- I/O operation × resource kind × semantic result, excluding host paths and
  secrets;
- World state transition, event type, cancellation/fault, and terminal state;
- application oracle/invariant reached; and
- distinct failure signature.

Stable semantic IDs must be derived from versioned domains and source/operation
identity, never from a global execution ordinal, pointer, descriptor, goroutine
ID, or map iteration. Code coverage and semantic coverage must be reported
separately: executing a line does not prove that a boundary result was checked.

### Reward and corpus

Score completed runs in this order:

1. new reproducible failure signature;
2. new required invariant failure or terminal state;
3. new World transition, I/O result, or runtime-choice outcome;
4. new operation/fault or transition pair;
5. new code edge; and
6. a smaller or faster reproduction of an existing finding.

Persist only bounded, identity-bound corpus entries: seed, target/toolchain and
manifest versions, realized transcripts, captured inputs, coverage digest,
score reasons, and replay result. Merge parallel results in deterministic
selection-ordinal order and use atomic publication.

Semantic probe observation must not start target goroutines, consume runtime
randomness, call host clocks, or perform unbounded allocation. Prefer
fixed/bounded counters or append-only probe buffers exported after the run. Any
instrumentation can still perturb compilation and memory layout, so bind its
configuration into target identity and compare only like-for-like runs; ordinary
Go code coverage remains a diagnostic dimension, not proof that an uninstrumented
schedule took the same path. Runner chooses the next campaign only from completed
runs; for parallel exploration, select a whole batch from one corpus snapshot,
then deterministically merge the batch before choosing again.

### Delivery sequence

1. Add `--guide --corpus DIR` to allocate future batches across under-covered
   domains while reserving a fixed fraction for unguided seeds.
2. Add guided World scenario, fault, and input generation, where mutations have
   semantic locality.
3. Consider stable runtime choice traces or forced choices only if retained
   seeds and realized transcripts cannot reproduce minimized failures.

An upgrade invalidates probe identities whose owning implementation changed.
Carry forward semantic requirements and replayable old artifacts, not raw
coverage bitmaps.

## Lessons from other DST systems

### Research question

Which practices from TigerBeetle and FoundationDB improve Gomad's completeness,
maintainability, testing, and guided exploration without replacing its native-Go
architecture?

### Findings

Gomad should borrow their testing disciplines, not their architectures
wholesale. Both systems were designed around an explicit simulated world.
Gomad instead runs ordinary Go code in a real child process, controls native
runtime scheduling and time, and intercepts a reviewed I/O surface.

- TigerBeetle's VOPR runs production code while stubbing clock, network, and
  disk operations. It derives topology, workload, and injected faults from a
  seed, then uses assertions and independent state/storage checkers as oracles.
  Reproduction requires both the seed and Git commit. Its documented faults
  include packet loss, reordering, partitions, and corrupt disk reads/writes.
  [TigerBeetle VOPR documentation](https://github.com/tigerbeetle/tigerbeetle/blob/main/docs/internals/vopr.md)
- TigerBeetle separates safety exploration from a liveness phase: after damage,
  the simulator heals a viable core, disables further faults, and requires
  convergence. This exposes progress failures that continuous fault injection
  can mask.
  [TigerBeetle liveness testing](https://tigerbeetle.com/blog/2023-07-06-simulation-testing-for-liveness/)
- TigerBeetle tests stock binaries from outside VOPR because the simulated
  network/storage implementations and native client boundary are not themselves
  exercised by VOPR. This is defense in depth, not a competing strategy.
  [TigerBeetle Vortex](https://tigerbeetle.com/blog/2025-02-13-a-descent-into-the-vortex/)
- TigerBeetle identifies independent generator and oracle blind spots: a
  workload may never generate the trigger, or its checks may not recognize the
  invalid result. Less-structured generation and a more detailed independent
  model address different failures.
  [TigerBeetle fuzzer blind spots](https://tigerbeetle.com/blog/2025-06-06-fuzzer-blind-spots-meet-jepsen/)
- FoundationDB runs production database code, randomized workloads, and
  injected failures in a deterministic, single-process, discrete-event
  simulation. Flow abstracts network, disk, time, and randomness; thin
  implementations connect production code to the simulator. Swarm testing
  varies features, configurations, and enabled fault points.
  [FoundationDB paper, section 4](https://www.foundationdb.org/files/fdb-paper.pdf)
- FoundationDB conditional probes count semantically important situations
  across simulation ensembles. Missing expected probes fail the test and guide
  changes to workloads, fault distributions, or buggification. This is stronger
  evidence than aggregate line coverage alone.
  [FoundationDB internal testing tools](https://apple.github.io/foundationdb/internal-dev-tools.html)
- FoundationDB supplements simulation with live performance and hardware-fault
  testing because simulation cannot validate every third-party, OS, filesystem,
  or performance behavior.
  [FoundationDB testing overview](https://apple.github.io/foundationdb/testing.html)

The transferable practices are realized transcripts plus exact build identity,
independent semantic oracles, conditional probes, separate generator/oracle
coverage, separate safety and recovery campaigns, outside-in boundary tests,
and an upgrade qualification dossier. Gomad already has strong foundations for
the first item; the boundary manifest and guided semantic coverage provide the
next two.

Important differences remain:

- FoundationDB's Flow actors and TigerBeetle's single-threaded event loop were
  designed for simulation. Replacing Go's scheduler and ordinary process model
  would be a different system, not a simplification.
- Their database network and disk APIs are narrow, owned interfaces. Gomad's
  transparent support for arbitrary trusted Go code needs stronger source-diff,
  dependency, and escape gates.
- Synchronous I/O transcripts are appropriate until readiness or competing
  events matter. Partitions, disk latency, crash/recovery, and similar policies
  belong in explicit World adapters rather than every standard-library shim.
- Large randomized fleets do not replace regressions, real-boundary tests,
  performance tests, or application invariants.

### Primary sources

- [TigerBeetle: VOPR](https://github.com/tigerbeetle/tigerbeetle/blob/main/docs/internals/vopr.md)
- [TigerBeetle: simulation testing for liveness](https://tigerbeetle.com/blog/2023-07-06-simulation-testing-for-liveness/)
- [TigerBeetle: Vortex](https://tigerbeetle.com/blog/2025-02-13-a-descent-into-the-vortex/)
- [TigerBeetle: fuzzer blind spots](https://tigerbeetle.com/blog/2025-06-06-fuzzer-blind-spots-meet-jepsen/)
- [FoundationDB paper](https://www.foundationdb.org/files/fdb-paper.pdf)
- [FoundationDB internal testing tools](https://apple.github.io/foundationdb/internal-dev-tools.html)
- [FoundationDB testing overview](https://apple.github.io/foundationdb/testing.html)

## Recommended execution plan

1. **Runtime confidence:** close the remaining CI gaps and rerun representative
   Temporal qualification.
2. **Guided coverage:** ship passive semantic reporting, then bounded novelty
   corpus retention and guided campaigns.
3. **Evidence-driven models:** implement only the generic I/O or World adapters
   required by newly minimized, qualified workloads.

Gomad v3 is ready for broader use when the supported platform has a green CI
gate, every claimed nondeterministic boundary is classified and tested, enabled
runs cannot silently reach ambient I/O through the reviewed surface, failures
replay from immutable evidence, and representative unchanged Temporal tests
exercise meaningful semantic and schedule diversity.
