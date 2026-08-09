# Gosim gaps

[`GOSIM.md`](GOSIM.md) explains gosim's intended boundary well, but it does not
fully describe the implementation risk inside that boundary. The missing point
is that gosim is not simply a mocked Go runtime. It is a hybrid of AST
translation, Go-version-specific standard-library hooks, a custom deterministic
runtime, and a simulated Linux syscall layer.

That distinction matters because all four layers must agree on Go semantics.
The public standard-library implementation is reused more often than in an API
mocking system, but the simulator is only as complete as its translator, hook
table, runtime primitives, and syscall model together.

This document focuses only on gosim at the imported `ffd3a613` snapshot.

## The actual interception stack

| Layer | What gosim replaces | Main correctness risk |
| --- | --- | --- |
| AST translator | Goroutines, channels, maps, `select`, package globals, imports, initialization, and selected linknames | Missed syntax, incorrect type rewriting, or an escape into untranslated code |
| Runtime hooks | Unexported functions used by `sync`, `time`, `net`, `os`, atomics, polling, and other standard-library packages | Every Go release may change an unstable internal signature or invariant |
| Deterministic runtime | Scheduling, channel/map behavior, semaphores, timers, randomness, globals, logging, checksums, and race edges | The replacement must reproduce language/runtime semantics closely enough for application code |
| Simulated Linux OS | File, socket, polling, memory-map, random, and machine-control syscalls | Unsupported or approximate kernel behavior changes what higher-level standard-library code observes |

The translator explicitly rewrites map and channel types and operations,
goroutine starts, package globals, and package initialization
([`internal/translate/translate.go`](tools/gomad/internal/translate/translate.go#L512-L574),
[`internal/translate/globals.go`](tools/gomad/internal/translate/globals.go#L78-L169)).
The standard-library boundary is then connected to versioned hooks and the
syscall ABI
([`docs/design.md`](tools/gomad/docs/design.md#L173-L200),
[`docs/design.md`](tools/gomad/docs/design.md#L258-L310)).

The architecture is therefore deeper than high-level API mocks, but it is not a
small or singular mock boundary.

## Confirmed gaps

### 1. The Go-version port is a semantic port, not a dependency update

The snapshot declares Go 1.23.2
([`go.mod`](tools/gomad/go.mod#L1-L4)) and names its hook package `go123`. The
hooks target unexported standard-library/runtime symbols, whose signatures have
no compatibility guarantee
([`docs/design.md`](tools/gomad/docs/design.md#L173-L188)). The translator also
contains Go-1.23-specific hook, accepted-linkname, assembly, and package-skip
tables
([`internal/translate/main.go`](tools/gomad/internal/translate/main.go#L27-L109)).

A port must validate more than compilation:

- every hooked symbol still has the expected signature and calling convention;
- linkname targets and empty assembly declarations still resolve correctly;
- synchronization and timer invariants still match the new standard library;
- architecture-specific generated syscall wrappers remain ABI-correct; and
- race-detector annotations still describe the correct happens-before edges.

Without a generated compatibility inventory and conformance tests, a successful
build can still hide semantic drift.

### 2. "Translate the standard library" has explicit escape boundaries

The normal translation path skips `runtime`, `errors`, `reflect`, `unsafe`,
`testing`, runtime profiling/metrics/coverage packages, and other selected
packages. It also preserves assembly for a package allowlist
([`internal/translate/main.go`](tools/gomad/internal/translate/main.go#L27-L80)).
Imports can opt out through `//gosim:notranslate`
([`internal/translate/translate.go`](tools/gomad/internal/translate/translate.go#L577-L609)).

Some escapes are necessary, but the snapshot does not produce a single artifact
that answers:

- which packages were translated, skipped, wrapped, or kept native;
- which unexported functions were hooked or intentionally left alone;
- which assembly and linkname edges remain native; and
- which executed calls crossed into non-simulated code.

This is the largest trust gap in the runtime-mock strategy. The boundary is
distributed across translation tables, hook files, source annotations, and
generated syscall code rather than exposed as a fail-closed manifest.

### 3. Built-in maps and channels are still replaced types

Gosim keeps the standard-library implementation above many runtime hooks, but it
does not keep native Go maps and channels. The translator changes their types and
operations to gosim runtime types
([`docs/design.md`](tools/gomad/docs/design.md#L113-L152),
[`internal/translate/translate.go`](tools/gomad/internal/translate/translate.go#L536-L564)).
It consequently needs a custom `reflect` layer to hide those differences
([`docs/design.md`](tools/gomad/docs/design.md#L193-L200)).

That leaves a difficult conformance surface around:

- reflection over maps, channels, interfaces, and method values;
- `unsafe` code that depends on native representations;
- generic constraints and implicit conversions involving rewritten types;
- map equality, hashing, iteration, and allocation behavior; and
- channel/select panic, close, fairness, and race semantics.

The source contains unimplemented or panic paths in the reflection wrapper, so
reflection compatibility is not complete
([`internal/reflect/value.go`](tools/gomad/internal/reflect/value.go#L80-L120),
[`internal/reflect/type.go`](tools/gomad/internal/reflect/type.go#L370-L405)).

### 4. Cooperative scheduling cannot control code between yield points

The scheduler chooses a runnable simulated goroutine and lets it run until it
yields, blocks through a modeled primitive, finishes, or panics. The runtime
itself notes that uninstrumented synchronization or a spin can hang a step and
that a per-step wall-clock timeout is missing
([`gosimruntime/runtime.go`](tools/gomad/gosimruntime/runtime.go#L234-L275)).

Runtime hooks reduce this risk by moving standard synchronization and polling
onto the simulated scheduler, but they cannot automatically preempt:

- CPU-bound loops without a modeled synchronization point;
- native code reached through a translation escape;
- unsupported assembly or `unsafe` synchronization; or
- a missed runtime/standard-library hook.

This failure mode should be detected as an explicit boundary violation with the
responsible goroutine and native stack, not left to a whole-test timeout.

### 5. Per-machine global isolation is conditional

Gosim rewrites package globals into per-machine containers and re-runs package
initializers when a machine starts. This is essential to its process model, not a
minor runtime feature
([`docs/design.md`](tools/gomad/docs/design.md#L202-L242),
[`internal/translate/globals.go`](tools/gomad/internal/translate/globals.go#L216-L276)).

The implementation deliberately shares globals classified as immutable to avoid
reinitialization cost. The design also warns that sharing objects between
machines can break identity assumptions, including sentinel errors
([`docs/design.md`](tools/gomad/docs/design.md#L238-L252)). Controlled
nondeterminism similarly relies on a promise that shared caching cannot affect
observable execution
([`docs/design.md`](tools/gomad/docs/design.md#L418-L434)).

Missing safeguards include:

- runtime rejection of direct cross-machine pointer/channel sharing;
- validation that a global classified as shared is actually immutable;
- a report of shared versus machine-local globals; and
- tests for sentinel identity, registries, caches, and package initialization
  across repeated crash/restart cycles.

### 6. Runtime and standard-library hook coverage is incomplete

Several hook packages intentionally panic for unsupported operations, including
parts of `runtime/debug`, `runtime/trace`, internal ABI/CPU helpers, `os`,
`syscall`, polling, and `x/sys/unix`
([`internal/hooks/go123/runtime_debug.go`](tools/gomad/internal/hooks/go123/runtime_debug.go#L1-L42),
[`internal/hooks/go123/runtime_trace.go`](tools/gomad/internal/hooks/go123/runtime_trace.go#L1-L16),
[`internal/hooks/go123/syscall.go`](tools/gomad/internal/hooks/go123/syscall.go#L1-L88)).

At the OS boundary, unknown raw syscalls are logged and return `ENOSYS`
([`internal/simulation/os_linux.go`](tools/gomad/internal/simulation/os_linux.go#L162-L175)).
Package loading forces Linux, the host architecture, the `sim` build tag, and
`CGO_ENABLED=0`
([`cmd/gosim/main.go`](tools/gomad/cmd/gosim/main.go#L155-L159),
[`internal/translate/main.go`](tools/gomad/internal/translate/main.go#L120-L136)).

These are hard compatibility limits. Every target workload needs an executed
inventory of hooks and syscalls classified as simulated, native, `ENOSYS`,
panic, or unreachable.

### 7. The simulated clock is global

The runtime uses one clock for all machines. The source explicitly leaves
per-machine clock offsets and broken clocks as future work
([`gosimruntime/time.go`](tools/gomad/gosimruntime/time.go#L1-L35)). The current
model can accelerate time and order timers deterministically, but it cannot
model:

- clock skew or drift between machines;
- wall-clock jumps;
- a clock that pauses across a machine crash; or
- different wall and monotonic clock failures.

This is a runtime-model gap rather than a network or filesystem omission.

### 8. The Linux mock is intentionally narrow

The network model provides IPv4 TCP-like streams, symmetric connectivity, and
constant symmetric delay. IPv4-only addressing is explicit in the machine API
([`machine.go`](tools/gomad/machine.go#L46-L57)); connectivity and delay are
applied in both directions
([`internal/simulation/network/net.go`](tools/gomad/internal/simulation/network/net.go#L130-L153)).
The README records that UDP and hostname resolution are absent
([`README.md`](tools/gomad/README.md#L19-L24)).

The filesystem models valuable write ordering and `fsync` behavior, but it is
still a subset of Linux file semantics. The public crash API applies partial
disk persistence during restart and notes that the choice should move to the
crash operation
([`machine.go`](tools/gomad/machine.go#L95-L116)).

Reusing the real `net`, `os`, and `net/http` source improves consistency above
the boundary. It does not make the syscall results kernel-equivalent. Network,
filesystem, and crash behavior still require differential and scenario-specific
validation.

### 9. Determinism checking does not cover complete simulation state

The checksum records scheduler picks and results, log writes, goroutine creation,
run boundaries, and clock advancement
([`gosimruntime/checksum.go`](tools/gomad/gosimruntime/checksum.go#L23-L35)). It
does not directly hash arbitrary heap state, syscall results, filesystem state,
network packets, global values, or race edges.

The active metatesting API compares checksum and log output; its result has no
separate trace field
([`metatesting/metatest.go`](tools/gomad/metatesting/metatest.go#L28-L43),
[`metatesting/metatest.go`](tools/gomad/metatesting/metatest.go#L259-L290)). Equal
checksums are therefore a useful execution fingerprint, not proof that the whole
runtime and OS state were equal.

A complete determinism contract needs a versioned event schema and explicit
coverage of all modeled nondeterministic choices and externally observable
results.

### 10. Schedule exploration is reproducible but not systematic

The scheduler uses a seeded random choice among runnable goroutines
([`gosimruntime/runtime.go`](tools/gomad/gosimruntime/runtime.go#L257-L275)). Seed
ranges replay or vary that choice, but there is no coverage metric, state-space
reduction, schedule bounding, shrinking, or PCT-style guarantee. The design
mentions PCT only as related work
([`docs/design.md`](tools/gomad/docs/design.md#L502-L514)).

Deterministic replay and exploration quality are separate properties. Gosim has
the first; the snapshot does not define the second beyond randomized seeds.

### 11. The replacement `testing` runtime is incomplete

Because translated tests need simulator-owned execution, gosim carries a copied
`testing` implementation. The runner supports only a small subset of standard
test flags and rejects other supplied `-test.*` flags
([`gosimruntime/testmain.go`](tools/gomad/gosimruntime/testmain.go#L59-L71),
[`gosimruntime/testmain.go`](tools/gomad/gosimruntime/testmain.go#L111-L122)).
Coverage, benchmarks, examples, and several entry points are unimplemented
([`internal/testing/missing.go`](tools/gomad/internal/testing/missing.go#L1-L45));
`T.TempDir` and `T.Setenv` panic
([`internal/testing/testing.go`](tools/gomad/internal/testing/testing.go#L642-L706)).

Metatesting requires prebuilt test binaries for cached runs and lists fuzzing and
parallelism as future work
([`metatesting/doc.go`](tools/gomad/metatesting/doc.go#L14-L44)). This expands the
Go-version maintenance surface beyond runtime hooks and affects normal CI use.

### 12. There is no compatibility or fidelity report

The snapshot has substantial behavior tests, race tests, an embedded etcd
example, and metatests. What is missing is one generated report that ties those
tests to the claimed boundary:

- supported Go version, OS, and architectures;
- translated and native packages;
- hook coverage by standard-library package;
- syscall coverage and known semantic deviations;
- native-versus-simulated differential test results;
- race-detector conformance;
- transform, initialization, runtime, and memory costs; and
- unsupported `testing` features and CLI flags.

Without that report, users discover compatibility by translation failure,
`ENOSYS`, panic, hang, or behavior drift.

## Unknowns that require an experiment

Source inspection cannot answer:

1. How much work is required to port all hooks and translator rules to the
   target Go version?
2. Which target dependency is the first to fail because of CGO, `unsafe`,
   reflection, assembly, linkname, or a missing syscall?
3. Does a large dependency graph complete a scheduler step without native
   blocking or excessive uninterrupted CPU work?
4. What are translation time, incremental rebuild time, package-global
   initialization time, peak RSS, and execution slowdown?
5. Do repeated crash/restart cycles correctly reset all mutable globals while
   retaining intended shared immutable data?
6. Does `-race` remain sound across channel, semaphore, timer, syscall, and
   machine boundaries after the Go-version port?

## Minimum acceptance bar

Before treating gosim as a usable runtime-mock platform for a new codebase, the
following should pass:

1. **Exact-version port:** all hook and linkname entries compile and have direct
   signature/invariant tests on the target Go version.
2. **Boundary manifest:** the build emits translated, skipped, native, hooked,
   assembly, CGO, and syscall classifications and fails on unknown entries.
3. **Differential conformance:** channels, maps, `select`, `sync`, timers,
   reflection, TCP, files, and crash-visible persistence are compared with native
   Go wherever the model claims equivalent behavior.
4. **Escape detection:** uninstrumented blocking, native synchronization, and
   cross-machine object sharing fail with actionable diagnostics.
5. **Determinism coverage:** every modeled random choice and observable runtime,
   network, filesystem, and machine event enters a versioned fingerprint.
6. **Race validation:** modeled synchronization passes copied Go race tests and
   workload-specific race tests.
7. **Tooling compatibility:** required `testing` methods, flags, cached runs, and
   CI artifacts work without bespoke manual steps.
8. **Measured scale:** full and incremental translation, global initialization,
   runtime, and memory costs are recorded for a representative workload.

The core conclusion is narrower than "mock the runtime and inherit standard
library fidelity." Gosim moves much of the semantic burden below public APIs,
which is a strong design, but it then assumes responsibility for the language
rewrite, per-machine global model, unstable runtime hooks, and Linux syscall
semantics. Those are the gaps that must be closed or made explicit.
