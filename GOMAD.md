# GoMaD overview

GoMaD is an experimental deterministic simulation environment for testing
concurrent and distributed Go systems. It runs ordinary Go code under a seeded,
cooperative scheduler so failures caused by a particular interleaving can be
reproduced. Simulated time advances without waiting for wall-clock time, and
the deeper machine model can inject network, process, and disk failures.

This branch contains two implementations:

- [`tools/gomad`](tools/gomad) is the current engine. It is a Go 1.26 port of
  [`jellevandenhooff/gosim`](https://github.com/jellevandenhooff/gosim), imported
  at the commit recorded in [`UPSTREAM.md`](tools/gomad/UPSTREAM.md).
- [`tools/gomad_old`](tools/gomad_old) preserves the original
  Temporal-specific implementation. Existing Temporal entry points still use
  this implementation while the new engine is evaluated for integration.

The two engines share the same central idea—transform normal Go source to take
control of nondeterminism—but place the simulation boundary at different
levels. [`GOSIM.md`](GOSIM.md) contains the detailed comparison and recommended
lessons.

## What GoMaD is for

GoMaD is intended to make concurrency and failure scenarios:

- deterministic: a seed selects the same random values and schedule;
- reproducible: a failed seed can be rerun directly;
- fast: the simulator jumps over idle time instead of sleeping;
- faultable: machines, network links, and disks can fail under test control;
- diagnosable: events carry step, simulated-time, machine, and goroutine
  context; and
- close to production: applications remain ordinary Go rather than being
  rewritten as explicit state machines.

Determinism is bounded by the translated environment. Native calls, unsupported
syscalls, CGO, or code deliberately excluded from translation can reintroduce
uncontrolled scheduling or state.

## Current engine

The current [`tools/gomad`](tools/gomad) engine implements three layers behind
a Go-test-like command.

### Deterministic runtime

The runtime in [`gosimruntime`](tools/gomad/gosimruntime) replaces goroutine,
channel, map, timer, synchronization, and random behavior. Only one simulated
goroutine executes at a time. At cooperative suspension points, a seeded random
source chooses the next runnable goroutine.

Each simulated goroutine is backed by a coroutine, which makes scheduling a
library concern instead of requiring a custom compiler or Go runtime. When all
goroutines are waiting for time, the scheduler advances the simulated clock to
the next timer. A test can therefore exercise hours of timeouts in much less
wall-clock time.

Map iteration and random values vary between seeds but remain stable for a
given seed. The runtime also maintains an execution checksum over scheduling
and simulation events, giving determinism checks a signal independent of log
text.

### Source and standard-library translation

The translator in [`internal/translate`](tools/gomad/internal/translate)
rewrites Go syntax and package state before invoking the normal Go compiler.
Important transformations include:

- `go` statements, channels, `select`, and maps become runtime operations;
- package globals become per-machine state that can be reinitialized;
- runtime and unexported standard-library calls are redirected through
  version-specific hooks; and
- Linux syscall entry points are redirected into the simulated operating
  system.

Most of the standard library is translated as well as the application. This
lets code continue using packages such as `sync`, `time`, `os`, `net`, and
`net/http` above a comparatively small runtime/syscall boundary. The trade-off
is that the hooks must be updated when Go changes unexported runtime or
standard-library details.

Translation results are cached by source and tool inputs. The CLI builds the
translated test binary with the standard toolchain, so normal profiling and
debugging tools remain usable.

### Simulated machines and operating system

The public [`Machine`](tools/gomad/machine.go) API creates isolated simulated
hosts. Each machine owns its package globals, IP address, filesystem, network
state, and lifecycle. A crash stops its goroutines and connections without
running graceful cleanup; restart reinitializes globals and runs its entry
point again.

The simulated Linux layer provides TCP networking and a POSIX-like filesystem.
Tests can control link connectivity and delay, crash individual machines, and
explore which in-flight filesystem writes survive a crash. The
[`nemesis`](tools/gomad/nemesis) package composes these operations into fault
scenarios.

This model is useful for questions that an in-process scheduler alone cannot
answer: connection recovery after partitions, reconstruction after node
restart, and persistence behavior around `fsync` and crashes.

## Test and debugging workflow

From [`tools/gomad`](tools/gomad), build the local runner and execute translated
tests with:

```text
go build -tags=test_dep -o .gosim/gosimtool ./cmd/gosim
.gosim/gosimtool test ./internal/tests/behavior
```

The runner accepts fixed seeds, seed ranges, test filters, race mode, tracing,
and other flags modeled after `go test`. A failure should always be replayed
with its exact seed before changing the test.

Logs include the simulation step, time, machine, and goroutine. Syscall and
stack tracing can expose the boundary crossed by a failing operation. The
debug command can launch Delve and stop at a selected simulation step.
[`metatesting`](tools/gomad/metatesting) can run nested simulations and compare
checksums or captured logs across seeds and reruns.

The Go 1.26 acceptance baseline is:

```text
go test -ldflags=-checklinkname=0 -tags=linkname,test_dep ./gosimruntime
.gosim/gosimtool prepare-selftest
.gosim/gosimtool test ./internal/tests/behavior ./nemesis
.gosim/gosimtool test -race ./internal/tests/behavior ./nemesis
```

## Go 1.26 status

The current engine declares Go 1.26.0 and has been verified with Go 1.26.1 on
Darwin/ARM64. The port handles the Go 1.26 runtime, FIPS, reflection iteration,
generic map, `internal/sync`, race, synctest, weak-pointer, syscall, and
architecture-specific changes exercised by the imported behavior and nemesis
suites.

Some compatibility adapters are intentionally approximate:

- weak pointers retain their referent instead of modeling collection;
- synctest bubble membership is not modeled;
- FIPS indicator and bypass state are deterministic but not a full runtime
  clone;
- internal race adapters preserve ordering but lose some object and caller-PC
  fidelity; and
- the `internal/sync` hash-trie adapter uses a collision-correct constant hash,
  which can degrade to linear behavior.

The simulator remains experimental. Its host model does not cover every Linux
or Go facility; unsupported examples include parts of UDP, DNS/hostnames,
filesystem permissions and links, and code that depends on CGO or external
services. Passing the imported suites establishes a stable porting baseline,
not universal compatibility with the Temporal dependency graph.

## Original Temporal-specific engine

The original implementation in [`tools/gomad_old`](tools/gomad_old) uses a
higher-level, Temporal-driven boundary. Its
[`transformer`](tools/gomad_old/transformer) rewrites language constructs and
selected library calls, copies dependencies into a `gomad.local` namespace,
applies embedded overlays for packages that need special handling, rewrites
failpoints, and compiles a standalone test binary in a generated workspace.

The simulated language layer in
[`api/lang`](tools/gomad_old/api/lang) covers goroutines, channels, selects,
maps, ranges, and function-entry yields. The library layer in
[`api/lib`](tools/gomad_old/api/lib) provides deterministic time, randomness,
contexts, synchronization, atomics, signals, networking, logging, and an
in-memory filesystem. Additional copied or adapted packages provide HTTP, SQL,
and in-process unary gRPC support.

Its [`runtime`](tools/gomad_old/runtime) uses one native goroutine for each
simulated goroutine and scheduler handshakes to ensure only the selected one
runs. Scheduling and timer ordering come from the seed; the clock jumps to the
next event. Native-operation stalls and all-blocked states produce simulated
goroutine locations plus native stack diagnostics.

Temporal tests opt in with the `gomad` build tag through
[`tests/gomad_test.go`](tests/gomad_test.go). A typical run and replay are:

```text
go test -tags='gomad test_dep' ./tests
go test -tags='gomad test_dep' ./tests -gomad.seed=<seed>
```

The controller in [`ctrl`](tools/gomad_old/ctrl) can also run two binaries in
lockstep and compare their logs after every scheduler step. Its experimental
checkpoint support records selected runtime operations, channel buffers,
clock, random state, and goroutine entry functions, then reconstructs and
replays them to continue from a checkpoint. This is not a snapshot of arbitrary
Go heap or external state.

## Direction

The current engine provides the deeper long-term model: independently
crashable machines, per-machine globals, faultable TCP links, crash-consistent
disks, execution checksums, structured traces, metatesting, and race-detector
integration. The original engine provides the shorter existing path into
Temporal's functional-test architecture.

The practical migration strategy is to keep the Go 1.26 port isolated and
well-tested, then validate one small Temporal scenario through the current
engine. That prototype should measure translation coverage, build cost,
runtime cost, dependency escape hatches, and failure-model value before the
existing `gomad` test-tag integration is replaced.
