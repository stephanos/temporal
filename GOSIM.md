# Gosim adoption for Temporal

This document records the decision to adopt gosim and retire the original
Temporal-specific GoMaD implementation. It compares the abandoned
implementation, preserved under [`tools/gomad_old`](tools/gomad_old), with
[`jellevandenhooff/gosim`](https://github.com/jellevandenhooff/gosim) at commit
[`ffd3a613`](https://github.com/jellevandenhooff/gosim/tree/ffd3a613542675755e4cbf8186b5edaf404ed95c).
Both projects are experimental deterministic simulation systems for ordinary Go
code, but they choose different simulation boundaries. The imported and Go
1.26-ported gosim source lives under [`tools/gomad`](tools/gomad).

> **Decision:** stop developing and integrating `tools/gomad_old`. All new
> deterministic-simulation work will improve the gosim-derived engine in
> `tools/gomad` and integrate that engine with Temporal. The legacy source is
> historical reference only and should receive no features or compatibility
> fixes.

The shortest useful summary is:

> Legacy GoMaD replaces selected Go language and library operations so
> Temporal's in-process test cluster can run under a deterministic scheduler.
> Gosim replaces the Go runtime/standard-library boundary and Linux syscalls so
> one process can contain multiple simulated machines with realistic network,
> filesystem, and crash behavior.

The legacy approach reached Temporal sooner, but its growing collection of
high-level substitutes and native escape hatches is not the foundation we want
to maintain. Gosim's deeper runtime/syscall boundary, multi-machine model,
faultable network, and crash-aware disk are a better fit for Temporal's
distributed-systems failure modes. The remaining work is therefore gosim
compatibility and Temporal integration, not further comparison between two
active implementations.

## Side-by-side

| Dimension | Legacy GoMaD (abandoned) | gosim (active) |
| --- | --- | --- |
| Primary scope | Temporal-specific, in-tree functional and concurrency testing | General-purpose distributed-systems simulation for Go |
| Source integration | `go test` with a `gomad` build-tagged `TestMain` | A dedicated `gosim test` CLI modeled after `go test` |
| Translation boundary | Test package and non-stdlib dependencies; selected calls are rewritten to legacy simulation APIs | Nearly all program, dependency, and standard-library code is translated |
| Runtime integration | Higher-level replacements for language constructs and library APIs | A lightweight runtime plus Go-version-specific hooks for unexported standard-library/runtime entry points |
| Simulated unit | One simulator running an in-process Temporal test cluster | Multiple simulated machines, each with its own globals, disk, network stack, and lifecycle |
| Goroutine implementation | One native goroutine per simulated goroutine, controlled by scheduler handshakes | Coroutines based on the mechanism behind `iter.Pull` |
| Scheduling and time | Seeded cooperative scheduler; jumps to the next timed event | Seeded cooperative scheduler; jumps when all goroutines are waiting for time |
| Network | In-memory TCP connection pairs and selected HTTP/gRPC substitutes | A virtual TCP network with per-link delay and connectivity control |
| Filesystem | In-memory `afero` filesystem; no durability model | POSIX-style simulated filesystem with `fsync`, in-flight writes, crash loss, and recovery-state exploration |
| Failure injection | Scheduling variation and simulated time; failpoint rewriting exists, but there is no first-class host/network fault API comparable to gosim | Machine crash/restart, partial disk persistence, partitions, delays, and a small `nemesis` package |
| Nondeterminism detection | Optional lockstep dual-process execution that compares logs at each scheduler step | Execution checksums plus metatests that compare checksums and logs across runs |
| Reproduction | Printed seed and `-gomad.seed` | Printed seed, seed ranges, and replay through the gosim CLI |
| Debugging | Scheduler logs, source locations, native stack dump, deadlock diagnostics | Structured step logs, syscall tracing, stack capture, and Delve-at-step debugging |
| Race detector | Not a documented integration contract | Explicitly integrates the standard Go race detector with simulated happens-before edges |
| Time travel | Experimental operation-log checkpoint and replay | No general process-checkpoint API at the compared revision |
| Host API fidelity | Selective and Temporal-driven; skipped or fake boundaries are common | Broader Linux syscall model, but still explicitly incomplete |
| Version maintenance | High-level shims track APIs and Temporal dependencies | Low-level hooks must track Go runtime and standard-library internals |

## What the approaches share

Both projects avoid requiring production code to be rewritten as explicit
state machines. Instead, they perform source-to-source transformation and
compile the result with the standard Go compiler. Both replace goroutines,
channels, maps, random values, and time with deterministic equivalents, and
both make a seed the handle for reproducing a schedule.

That shared choice has two important consequences:

1. Existing Go application code can remain close to its production form.
2. Correctness depends on the completeness of the transformation boundary.
   Any operation that reaches uncontrolled native behavior can break
   determinism, block the cooperative scheduler, or make simulation semantics
   diverge from production.

The historical architectural disagreement was where to make that boundary
manageable. The project has chosen gosim's lower boundary.

## Legacy GoMaD: virtualize the APIs Temporal uses

Legacy GoMaD does not translate the Go standard library as a whole. It rewrites
language operations and selected package references, supplies cooperative
implementations of common APIs, copies a few library packages where necessary,
and leaves other dependency families on their real implementations. Its
[transformer configuration](tools/gomad_old/transformer/transform.go) contains
Temporal-specific gRPC/HTTP rules and a growing skip list for dependency
boundaries that otherwise produce incompatible Go types.

This made legacy GoMaD comparatively adaptable to the source tree in front of
it. When Temporal needed a particular behavior, the project could add a focused
shim, fake, or overlay without first implementing a complete host. It also let
the framework follow the repository's Go toolchain and dependency versions.

The trade-off is a broad semantic surface. Reimplementing `context`, `sync`,
timers, networking, HTTP, SQL, gRPC, OS calls, and their interactions at a high
level is difficult. A real API may compile against a fake but behave
differently around cancellation, buffering, connection lifecycle, errors, or
resource cleanup. Package skips can also let native behavior back into the
simulation. Legacy GoMaD mitigates these problems with stuck detection and an
optional two-process determinism check, but those mechanisms detect symptoms
rather than prove semantic equivalence.

## Gosim: virtualize below the standard library

Gosim's design deliberately moved away from replacing high-level APIs. It
translates the standard library and redirects its internal runtime hooks and
Linux syscalls into a simulated runtime and OS. The ordinary `net/http`,
`os.File`, or `sync.Mutex` implementation therefore remains in use above that
boundary. The architecture is described in gosim's pinned
[`docs/design.md`](https://github.com/jellevandenhooff/gosim/blob/ffd3a613542675755e4cbf8186b5edaf404ed95c/docs/design.md).

This is a deeper module: a relatively small runtime/syscall interface supports
many higher-level packages without separately faking each one. It enables a
coherent machine abstraction, where every machine has isolated package
globals, its own network stack and disk, and a crash/restart lifecycle. The
[`Machine` API](https://github.com/jellevandenhooff/gosim/blob/ffd3a613542675755e4cbf8186b5edaf404ed95c/machine.go)
can model graceful stops, hard crashes, partial persistence, and repeated disk
recovery states.

The cost moves into lower-level compatibility. Gosim has hooks for unexported
Go runtime and standard-library functions, plus architecture-specific syscall
bindings. The compared revision declared Go 1.23.2 and contained `go123` hooks.
The local port now passes gosim's translated behavior and nemesis suites on Go
1.26.1, but the amount of compatibility code required confirms that each Go
release is a meaningful maintenance event. Translating a dependency graph as
large and type-sensitive as Temporal's remains a substantial compatibility
exercise.

Gosim's deeper boundary is not complete emulation. Its own
[`README`](https://github.com/jellevandenhooff/gosim/blob/ffd3a613542675755e4cbf8186b5edaf404ed95c/README.md)
calls the project experimental and lists gaps such as UDP, hostnames, file
permissions, and links. Those gaps matter for a server with broad platform,
database, telemetry, and networking dependencies.

## Different notions of a distributed system

The legacy GoMaD world is effectively one simulated host. Temporal frontend,
history, matching, worker, persistence, and clients can run concurrently, but
they share the same simulation state and do not acquire independent process
globals or host lifecycles. In-process TCP and fake gRPC are primarily
coordination mechanisms; they are not faultable network links.

Gosim makes machines first-class. Package globals are rewritten into
per-machine storage and reinitialized on restart. Machines communicate through
the simulated OS, which can delay or disconnect network links. This model can
ask questions legacy GoMaD cannot express directly:

- What happens when one Temporal node crashes without running deferred cleanup?
- Does a connection fail and recover correctly across a partition?
- Which writes survive a crash around `fsync`?
- Does a restarted process reconstruct correct state from disk and peers?

For an ordering bug inside one process, gosim can run a single simulated
machine. We will support that simpler case within the same engine instead of
maintaining a second scheduler and transformation stack.

## Determinism and debugging

Legacy GoMaD's verification mode runs two OS processes in lockstep with the
same seed and compares their output after every scheduler step. This is simple
and useful for the Temporal code already emitting diagnostic logs. Its blind
spot is unlogged state: two runs can diverge internally and converge on the
same output, or only reveal the difference much later.

Gosim computes a running checksum over scheduling decisions and selected
runtime events. Its metatesting layer can rerun a seed and compare the checksum
and logs, providing a more direct signal that execution diverged. Gosim also
associates logs with machine, goroutine, and simulated time; numbers checksum
events by step; can trace syscalls; and can launch Delve stopped at a chosen
step. This is a stronger debugging product around the simulator, not only a
stronger simulation model.

Legacy GoMaD has one feature that gosim does not expose at the compared
revision: checkpoint-and-restore. It can record selected simulation operations
and replay to a checkpoint before exploring a continuation. That can eventually
reduce the cost of exploring deep Temporal scenarios. Today it is narrower
than a machine or process snapshot, so it should not be used to infer that
arbitrary Temporal heap state and side effects have been rolled back.

Checkpointing is not a reason to retain the legacy engine. If exploration cost
later justifies it, checkpointing should be designed against gosim's machine,
runtime, network, and disk state rather than porting the legacy operation-log
prototype.

## Fit for Temporal

### Why gosim is the chosen foundation

- Its machine/OS boundary is a better conceptual model for a distributed
  database and durable-execution server.
- Network partitions, node restarts, TCP connection failure, and disk crash
  consistency are part of the model instead of test-specific fakes.
- Per-machine globals more faithfully represent independent Temporal server
  processes.
- Runtime checksums, structured tracing, Delve integration, race-detector
  support, and metatesting form a more complete diagnosis workflow.
- A low-level syscall boundary can be deeper and more maintainable than an
  ever-growing collection of high-level substitutes once the Go-version port
  is paid for.

The legacy implementation's advantage was existing Temporal integration, not a
better simulation boundary. We will rebuild the useful integration ergonomics
on gosim rather than continue maintaining high-level replacements for Go and
third-party APIs.

### What will not be carried forward

- No new features, Go-version fixes, overlays, or API shims will be added to
  `tools/gomad_old`.
- Temporal tests will not be split permanently between two deterministic
  schedulers.
- Legacy fake HTTP, gRPC, SQL, filesystem, and synchronization behavior will
  not be ported when the translated standard library or simulated OS can supply
  the behavior.
- The legacy checkpoint prototype will not constrain gosim's runtime or machine
  design.

The old source may be consulted for Temporal-specific build, test-selection,
and lifecycle requirements. Once those requirements are represented in the
gosim integration, `tools/gomad_old` can be deleted.

### Integration questions and backlog

Improving gosim for Temporal must answer:

1. Can the local Go 1.26 runtime-hook port be kept current at an acceptable
   cost as Go's unexported internals continue to change?
2. Can the complete Temporal dependency graph be translated without breaking
   generated protobufs, reflection, unsafe code, gRPC/telemetry type identity,
   or build constraints?
3. Which persistence backends can run against the simulated filesystem, and
   what happens to CGO-backed or external databases?
4. Which required Linux syscalls, socket behaviors, DNS features, TLS paths,
   and file semantics are absent?
5. What is the transform, compile, memory, and execution cost for a Temporal
   functional test at this scale?
6. How should existing test-cluster lifecycle code map onto gosim machines and
   fault scenarios?

These questions define the implementation backlog. They are not gates for
resuming legacy GoMaD work and do not imply that both engines remain active.

## Go 1.26 foundation

The pinned gosim source is imported as a nested module under
[`tools/gomad`](tools/gomad), with provenance recorded in
[`UPSTREAM.md`](tools/gomad/UPSTREAM.md). The port was verified with Go 1.26.1
on Darwin/ARM64, and the nested module now declares Go 1.26.0.

The port passes the runtime unit target, translator tests, self-translation,
the translated behavior and nemesis suites, and those translated suites under
the race detector. The principal acceptance commands are:

```text
go test -ldflags=-checklinkname=0 -tags=linkname,test_dep ./gosimruntime
.gosim/gosimtool prepare-selftest
.gosim/gosimtool test ./internal/tests/behavior ./nemesis
.gosim/gosimtool test -race ./internal/tests/behavior ./nemesis
```

The compatibility layer covers the Go 1.26 changes that crossed gosim's
translation boundary:

- moved and added runtime/syscall entry points, ARM64 CPU probes, DIT and caller
  intrinsics, wait-group semaphores, environment clearing, and `vgetrandom`;
- Go 1.26 FIPS packages and assembly boundaries, including indicator and bypass
  state plus constant-time helpers;
- `reflect.TypeAssert`, `Value.Seq`, and `Value.Seq2` for translated maps;
- named-map generic constraints and Go 1.26's `internal/sync.HashTrieMap`;
- `internal/race`, `internal/synctest`, `weak`, and new time runtime hooks; and
- the Linux `O_DIRECTORY` behavior exercised by crash/disk tests.

Some adapters intentionally choose deterministic approximations where Go does
not expose a stable contract. Weak pointers retain identity, synctest bubble
bookkeeping is not modeled, FIPS state is simulation-goroutine-local rather
than a complete runtime clone, race adapters lose some object/PC fidelity, and
the internal hash-trie adapter uses a collision-correct constant hash that can
degrade to linear behavior. These are explicit limitations, not unverified
escape hatches in the acceptance suites.

The port establishes the baseline for further work: gosim can run on Go 1.26,
including translated and race-tested behavior. It was not a one-line toolchain
bump, so compatibility with unexported standard-library organization, runtime
linknames, and architecture-specific assembly must remain an explicit,
continuously tested maintenance area.

## How we should improve gosim

The goal is now to deepen one engine, not copy selected gosim ideas into legacy
GoMaD. Gosim already provides checksums, structured logs, metatesting,
race-detector integration, coroutines, machines, faultable networking, and a
crash-aware disk. Work should close semantic and integration gaps around that
foundation.

### Easier

- Provide a Temporal-facing command that owns translation, package selection,
  seed ranges, race mode, and artifact paths without exposing `.gosim`
  internals.
- Emit one copy-pasteable reproduction command for every failure, including
  package, test, seed, step, and trace settings.
- Add reusable fixtures for starting a Temporal service set as gosim machines
  and applying common crash, restart, delay, and partition scenarios.
- Turn unsupported translation, linkname, and syscall boundaries into concise
  diagnostics that identify the missing adapter and source operation.

### Better

- Replace the Go 1.26 compatibility approximations for weak pointers,
  synctest, FIPS state, race metadata, and `internal/sync` hashing where
  Temporal or differential tests demonstrate observable differences.
- Add differential tests that run focused standard-library behaviors natively
  and under gosim, then compare values, errors, logs, and lifecycle effects.
- Audit Temporal's dependency graph for native escape hatches, unsafe code,
  CGO, syscalls, DNS, TLS, and filesystem semantics before widening test
  coverage.
- Model a minimal multi-node Temporal topology with independent globals,
  network identities, crash/restart lifecycles, and persistence state.
- Keep execution checksums and race-detector edges comprehensive as new runtime,
  network, disk, and synchronization operations are added.

### Simpler

- Keep Go-version-specific hooks isolated in one compatibility layer with a
  documented mapping from every unexported Go symbol to its gosim behavior.
- Maintain one upstream provenance record and a reviewable local patch series
  so future gosim refreshes do not require rediscovering the fork's intent.
- Put Temporal lifecycle adaptation behind a small, testable machine-cluster
  interface instead of spreading gosim bookkeeping through functional tests.
- Maintain an auditable inventory of untranslated code and native escape
  hatches, including the nondeterminism each escape can introduce.
- Do not resurrect legacy high-level fakes or keep dual execution paths for
  cases that can be expressed through gosim's standard library and OS model.

### Faster

- Extend the existing translation cache in
  [`internal/translate/cache.go`](tools/gomad/internal/translate/cache.go) with
  measurements and precise invalidation for the Temporal dependency graph.
- Shard seed ranges across workers, retain the first useful failure per shard,
  and make replay independent of worker count.
- Profile translation, compilation, simulation memory, and steps per second on
  a representative Temporal scenario before optimizing the runtime.
- Replace the collision-correct constant hash in the Go 1.26
  `internal/sync.HashTrieMap` adapter before it becomes a workload bottleneck.
- Reuse prepared translated standard-library and dependency artifacts across
  Temporal test packages when their inputs are identical.

The recommended order is: preserve the green Go 1.26 compatibility baseline;
remove correctness-risk approximations that affect Temporal; run one minimal
Temporal service scenario; add reproduction and fault-scenario ergonomics;
then optimize from measured translation and execution profiles.

## Practical conclusion

There is one forward path: improve the Go 1.26 gosim fork in `tools/gomad` and
integrate Temporal with it. `tools/gomad_old` is abandoned, read-only reference
material; it should not influence prioritization through its lower short-term
integration cost.

All new simulation tests should use gosim's scheduler, machine model, network,
and disk. Single-process interleaving tests should run as a one-machine gosim
scenario; distributed failure tests should add machines and explicit faults.
This keeps scheduling, reproduction, tracing, race semantics, and failure
models consistent across the test portfolio.

The next milestone is not another framework comparison. It is one minimal,
reproducible Temporal scenario running under gosim on Go 1.26, followed by an
inventory of the exact translation, syscall, dependency, and lifecycle gaps
blocking a representative multi-node scenario. Once the remaining useful
integration knowledge has been extracted, the legacy source can be removed.
