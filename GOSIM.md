# GoMaD compared with gosim

This document compares the GoMaD implementation on this branch with
[`jellevandenhooff/gosim`](https://github.com/jellevandenhooff/gosim) at commit
[`ffd3a613`](https://github.com/jellevandenhooff/gosim/tree/ffd3a613542675755e4cbf8186b5edaf404ed95c).
Both projects are experimental deterministic simulation systems for ordinary Go
code, but they choose different simulation boundaries.

The shortest useful summary is:

> GoMaD replaces selected Go language and library operations so Temporal's
> in-process test cluster can run under a deterministic scheduler. Gosim
> replaces the Go runtime/standard-library boundary and Linux syscalls so one
> process can contain multiple simulated machines with realistic network,
> filesystem, and crash behavior.

Neither boundary is universally better. GoMaD is closer to the current
Temporal repository and cheaper to specialize. Gosim provides a deeper and
more coherent model of distributed hosts, but would require substantial
version and dependency integration before it could run this Temporal tree.

## Side-by-side

| Dimension | GoMaD | gosim |
| --- | --- | --- |
| Primary scope | Temporal-specific, in-tree functional and concurrency testing | General-purpose distributed-systems simulation for Go |
| Source integration | `go test` with a `gomad` build-tagged `TestMain` | A dedicated `gosim test` CLI modeled after `go test` |
| Translation boundary | Test package and non-stdlib dependencies; selected calls are rewritten to GoMaD APIs | Nearly all program, dependency, and standard-library code is translated |
| Runtime integration | Higher-level replacements for language constructs and library APIs | A lightweight runtime plus Go-version-specific hooks for unexported standard-library/runtime entry points |
| Simulated unit | One GoMaD simulator running an in-process Temporal test cluster | Multiple simulated machines, each with its own globals, disk, network stack, and lifecycle |
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

The main architectural disagreement is where to make that boundary manageable.

## GoMaD: virtualize the APIs Temporal uses

GoMaD does not translate the Go standard library as a whole. It rewrites
language operations and selected package references, supplies cooperative
implementations of common APIs, copies a few library packages where necessary,
and leaves other dependency families on their real implementations. Its
[transformer configuration](tools/gomad/transformer/transform.go) contains
Temporal-specific gRPC/HTTP rules and a growing skip list for dependency
boundaries that otherwise produce incompatible Go types.

This makes GoMaD comparatively adaptable to the source tree in front of it.
When Temporal needs a particular behavior, the project can add a focused shim,
fake, or overlay without first implementing a complete host. It also lets the
framework follow the repository's current Go toolchain and dependency versions.

The trade-off is a broad semantic surface. Reimplementing `context`, `sync`,
timers, networking, HTTP, SQL, gRPC, OS calls, and their interactions at a high
level is difficult. A real API may compile against a fake but behave
differently around cancellation, buffering, connection lifecycle, errors, or
resource cleanup. Package skips can also let native behavior back into the
simulation. GoMaD mitigates these problems with stuck detection and an optional
two-process determinism check, but those mechanisms detect symptoms rather
than prove semantic equivalence.

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
bindings. The compared revision declares Go 1.23.2 and contains `go123` hooks;
this Temporal branch declares Go 1.26.3. It is therefore reasonable to expect a
Go-version port before evaluating Temporal itself. Translating the full
standard library and a dependency graph as large and type-sensitive as
Temporal's would also be a substantial compatibility exercise.

Gosim's deeper boundary is not complete emulation. Its own
[`README`](https://github.com/jellevandenhooff/gosim/blob/ffd3a613542675755e4cbf8186b5edaf404ed95c/README.md)
calls the project experimental and lists gaps such as UDP, hostnames, file
permissions, and links. Those gaps matter for a server with broad platform,
database, telemetry, and networking dependencies.

## Different notions of a distributed system

The current GoMaD world is effectively one simulated host. Temporal frontend,
history, matching, worker, persistence, and clients can run concurrently, but
they share the same simulation state and do not acquire independent process
globals or host lifecycles. In-process TCP and fake gRPC are primarily
coordination mechanisms; they are not faultable network links.

Gosim makes machines first-class. Package globals are rewritten into
per-machine storage and reinitialized on restart. Machines communicate through
the simulated OS, which can delay or disconnect network links. This model can
ask questions GoMaD currently cannot express directly:

- What happens when one Temporal node crashes without running deferred cleanup?
- Does a connection fail and recover correctly across a partition?
- Which writes survive a crash around `fsync`?
- Does a restarted process reconstruct correct state from disk and peers?

Conversely, the extra fidelity is unnecessary when the bug is an ordering
inside one in-process test cluster. For that class of problem, GoMaD's direct
integration has a much shorter path to useful coverage.

## Determinism and debugging

GoMaD's verification mode runs two OS processes in lockstep with the same seed
and compares their output after every scheduler step. This is simple and useful
for the Temporal code already emitting diagnostic logs. Its blind spot is
unlogged state: two runs can diverge internally and converge on the same output,
or only reveal the difference much later.

Gosim computes a running checksum over scheduling decisions and selected
runtime events. Its metatesting layer can rerun a seed and compare the checksum
and logs, providing a more direct signal that execution diverged. Gosim also
associates logs with machine, goroutine, and simulated time; numbers checksum
events by step; can trace syscalls; and can launch Delve stopped at a chosen
step. This is a stronger debugging product around the simulator, not only a
stronger simulation model.

GoMaD has one feature that gosim does not expose at the compared revision:
checkpoint-and-restore. GoMaD can record selected simulation operations and
replay to a checkpoint before exploring a continuation. That can eventually
reduce the cost of exploring deep Temporal scenarios. Today it is narrower than
a machine or process snapshot, so it should not be used to infer that arbitrary
Temporal heap state and side effects have been rolled back.

## Fit for Temporal

### Where GoMaD has the advantage

- It already lives in the Temporal module and targets its current tests,
  dependencies, generated code, and toolchain.
- Its integration can be improved incrementally as specific Temporal tests
  expose unsupported operations.
- It is well aligned with schedule, timer, channel, and in-process RPC bugs.
- Seed replay and accelerated time can add value before full distributed-host
  simulation exists.
- The checkpoint prototype provides a path toward branching long-running
  scenarios.

### Where gosim has the advantage

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

### Adoption risks for gosim in this repository

Before gosim could replace GoMaD for Temporal, an engineering spike would need
to answer at least:

1. Can its Go 1.23 runtime hooks be ported to the repository's Go version and
   kept current at an acceptable cost?
2. Can the complete Temporal dependency graph be translated without breaking
   generated protobufs, reflection, unsafe code, gRPC/telemetry type identity,
   or build constraints?
3. Which persistence backends can run against the simulated filesystem, and
   what happens to CGO-backed or external databases?
4. Which required Linux syscalls, socket behaviors, DNS features, TLS paths,
   and file semantics are absent?
5. What is the transform, compile, memory, and execution cost for a Temporal
   functional test at this scale?
6. Can existing test-cluster lifecycle code be mapped cleanly onto gosim
   machines without maintaining a second architecture solely for tests?

These are integration questions, not evidence that the gosim architecture is
wrong. They explain why its greater modeled fidelity does not translate into a
drop-in replacement.

## Go 1.26 import experiment

The pinned gosim source is imported as a nested module under
[`tools/gomad`](tools/gomad), with provenance recorded in
[`UPSTREAM.md`](tools/gomad/UPSTREAM.md). The experiment used Go 1.26.3 on
Darwin/ARM64. Gosim's `go.mod` still declares Go 1.23.2 because changing that
line would overstate compatibility: ordinary runtime code and translated
standard-library code have different results.

The coroutine/runtime unit target passes when invoked with gosim's required
linkname settings:

```text
go test -ldflags=-checklinkname=0 -tags=linkname,test_dep ./gosimruntime
```

The translated self-test build does not yet pass:

```text
.gosim/gosimtool build-tests ./internal/tests/behavior ./nemesis
```

The first failures were mechanical Go standard-library moves and additions.
The local experiment added adapters for:

- `internal/runtime/syscall/linux.Syscall6`, moved from
  `internal/runtime/syscall`;
- the ARM64 CPU `getpfr0` probe and DIT helpers;
- the new `internal/runtime/sys` caller intrinsics;
- FIPS `subtle.xorBytes`;
- `sync.runtime_SemacquireWaitGroup`; and
- `syscall.runtimeClearenv`.

After those adapters, translation reaches new FIPS SHA-256 and SHA-512 assembly
entry points (`blockSHA2` and `blockSHA512`) and stops, depending on which
package is translated first. It also reports new runtime-linkname surfaces in
`crypto/subtle`, `weak`, `internal/synctest`, `internal/sync`,
`internal/runtime/maps`, `internal/syscall/unix`, and `time`. Some are aliases
to existing gosim behavior, but others need an explicit decision about weak
pointers, synctest bubbles, runtime map internals, FIPS state, assembly
fallbacks, or simulated time semantics.

The result is clear enough for planning: gosim's core runtime can compile and
run on Go 1.26, but its source translator is not an easy Go-version bump. Its
low-level boundary buys a coherent simulation model at the cost of tracking
unexported standard-library organization and runtime linknames. A real port
should be treated as a dedicated compatibility project with translated
behavior and race suites as its acceptance tests.

## What GoMaD should take from gosim

GoMaD should copy gosim's high-leverage runtime and developer-experience ideas,
not its entire standard-library translation boundary. That preserves GoMaD's
main advantage—being easy to evolve with Temporal—while improving the parts
where gosim is observably stronger.

### Easier

- Add metatest helpers that rerun a seed and automatically compare checksums
  and logs, following
  [`metatesting/metatest.go`](tools/gomad/metatesting/metatest.go). This turns
  determinism verification into a normal test assertion instead of a special
  manual mode.
- Make seed ranges, exact-seed replay, and test selection first-class in one
  runner. Gosim's [`cmd/gosim`](tools/gomad/cmd/gosim/main.go) is a useful UX
  reference, but GoMaD can keep its simpler `go test` integration.
- Emit one copy-pasteable reproduction command whenever a run fails, including
  the seed and any step or trace filters.

### Better

- Hash scheduler decisions and externally visible simulation events using a
  running execution checksum like
  [`gosimruntime/checksum.go`](tools/gomad/gosimruntime/checksum.go). Logs should
  remain diagnostic output, not the definition of determinism.
- Give every event stable simulated-time, step, goroutine, and eventual-machine
  fields, borrowing the structured logging model in
  [`gosimruntime/log.go`](tools/gomad/gosimruntime/log.go). The same event stream
  should drive traces, checksums, and deadlock reports.
- Add explicit race-detector acquire/release edges for simulated channels,
  semaphores, timers, and network delivery, using
  [`gosimruntime/raceutil_race.go`](tools/gomad/gosimruntime/raceutil_race.go)
  as the reference.
- Define faults as composable scenarios—partition, delay, crash, restart—rather
  than accumulating call-site failpoints. Gosim's
  [`Machine`](tools/gomad/machine.go) and [`nemesis`](tools/gomad/nemesis)
  packages show the shape of that API.

### Simpler

- Use a single typed event record as the contract between scheduling,
  checksumming, tracing, replay, and debugging. This removes parallel ad hoc
  logging protocols.
- Maintain an auditable inventory of native escape hatches and skipped
  packages. Every escape should say which nondeterminism or blocking behavior
  remains possible.
- Keep per-machine state behind one deep lifecycle interface if multi-process
  simulation is added. Do not expose machine bookkeeping throughout Temporal
  tests.
- Do not copy gosim's `go123` hook table or full standard-library translation
  yet. The Go 1.26 experiment demonstrates that this would make GoMaD harder,
  not simpler, before Temporal needs disk-crash and host-lifecycle fidelity.

### Faster

- Use a checksum in ordinary runs and reserve dual-process lockstep for focused
  validation. That avoids paying for two processes on every exploratory seed
  while retaining a stronger diagnostic mode.
- Cache transformations by tool version, source, imports, build tags, and
  architecture, as gosim does in
  [`internal/translate/cache.go`](tools/gomad/internal/translate/cache.go).
- Shard seed ranges across test workers and stop each shard on its first useful
  failure.
- Benchmark gosim's coroutine implementation in
  [`internal/coro`](tools/gomad/internal/coro) behind GoMaD's scheduler
  abstraction. Adopt it only if native-goroutine overhead is material; its
  runtime linknames add version risk.

The recommended order is: execution checksum and typed traces first,
metatesting and reproduction UX second, race annotations third, then a small
multi-machine fault prototype. Coroutine replacement and full syscall/disk
simulation should remain evidence-driven follow-ups.

## Practical conclusion

For near-term work on the current Temporal tree, GoMaD is the more direct path
to deterministic scheduling and simulated-time coverage. Gosim is the stronger
reference architecture for the longer-term goal of testing Temporal as a set
of independently crashable machines with faultable networks and durable disks.

A useful strategy is therefore not to treat the projects as mutually
exclusive. GoMaD can continue validating whether deterministic simulation
finds valuable Temporal bugs, while adopting ideas that do not require gosim's
entire runtime boundary:

- Record an execution checksum instead of relying only on log equality.
- Add structured machine/goroutine/step metadata and step-oriented debugging.
- Make the boundary between simulated and native code explicit and auditable.
- Integrate race-detector happens-before annotations.
- Define first-class fault scenarios before accumulating ad hoc failpoints.
- Treat a multi-node/multi-machine model as a distinct future layer rather than
  implying it through in-process RPC fakes.

If the primary requirement becomes crash consistency, network partitions, or
node lifecycle fidelity, gosim deserves a focused porting prototype. If the
primary requirement remains reproducible interleavings in Temporal's existing
functional tests, improving GoMaD's coverage and determinism diagnostics is the
lower-complexity investment.
