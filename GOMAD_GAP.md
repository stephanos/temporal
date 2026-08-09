# GOMAD runtime-simulation gaps

[`GOMAD.md`](GOMAD.md) explains gomad's intended boundary well, but it does not
fully describe the implementation risk inside that boundary. The missing point
is that gomad is not simply a mocked Go runtime. It is a hybrid of AST
translation, Go-version-specific standard-library hooks, a custom deterministic
runtime, and a simulated Linux syscall layer.

That distinction matters because all four layers must agree on Go semantics.
The public standard-library implementation is reused more often than in an API
mocking system, but the simulator is only as complete as its translator, hook
table, runtime primitives, and syscall model together.

This document focuses on the runtime-simulation strategy in `tools/gomad`; the
previous AST-rewrite implementation lives in
`tools/gomad_old` and appears below only as a source of reusable test ideas.

## The actual interception stack

| Layer | What gomad replaces | Main correctness risk |
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

Severity is based on correctness impact, likelihood in representative server
workloads, detectability, and whether the failure is silent:

- **Critical:** can silently invalidate core language semantics or machine
  isolation; do not trust simulation results until bounded.
- **High:** can block common workloads, produce materially wrong results, hang,
  or prevent the simulator's fidelity claims from being audited.
- **Medium:** limits bug-finding power, scale, or tooling, but is scoped or
  usually detectable and has a practical workaround.
- **Low:** localized usability or polish issue with little effect on result
  validity. No confirmed gap currently falls in this tier.

| Priority | Gap | Severity | Why it ranks here |
| ---: | --- | --- | --- |
| 1 | 2. Native escape boundaries | **Critical** | Unmodeled code can silently bypass scheduling and OS interception. |
| 2 | 3. Rewritten maps and channels | **Critical** | These are core language primitives; semantic drift contaminates most workloads. |
| 3 | 5. Conditional global isolation | **Critical** | Cross-machine shared state invalidates the simulated process boundary. |
| 4 | 20. Packets crossing restart generations | **Critical** | Pre-crash packets can silently affect a post-restart connection that reuses the same address and ports. |
| 5 | 21. Crash closes TCP gracefully | **High** | The crash path systematically masks half-open connection and timeout behavior. |
| 6 | 1. Go-version semantic port | **High** | Internal hooks and ABIs can drift despite a successful build; critical while upgrading to an unvalidated Go release. |
| 7 | 4. Cooperative scheduling | **High** | Native blocking or CPU spins can hang the entire run without an actionable boundary error. |
| 8 | 6. Incomplete runtime/hooks | **High** | Common dependency paths can panic, return `ENOSYS`, or retain native behavior. |
| 9 | 8. Narrow Linux model | **High** | Network, filesystem, and crash deviations directly affect distributed-system conclusions. |
| 10 | 22. Completion hides work/durability bugs | **High** | Normal return reports success while aborting background work and persisting every write. |
| 11 | 9. Partial determinism fingerprint | **High** | Equal checksums can overstate replay equivalence because modeled state is omitted. |
| 12 | 12. No compatibility/fidelity report | **High** | Unsupported boundaries remain undiscoverable until a workload fails or drifts. |
| 13 | 13. Uneven, unmeasured test coverage | **High** | Core claims lack executable coverage evidence and native differential oracles. |
| 14 | 19. Missing atomic scheduling points | **High** | Lock-free algorithms can execute without the interleavings the scheduler is meant to explore. |
| 15 | 16. Unbounded crash-state exploration | **High** | Exhaustive subsets and retained machines can make the crash feature infeasible or unstable. |
| 16 | 17. Fault-control correctness | **High** | Control-plane inputs can be ignored or silently changed, so the executed scenario differs from the requested one. |
| 17 | 7. Global clock | **Medium** | Important clock faults are absent, but ordinary single-clock timer semantics remain useful. |
| 18 | 10. Random-only schedule exploration | **Medium** | Reduces bug-finding power without by itself changing a replayed schedule's semantics. |
| 19 | 11. Incomplete `testing` runtime | **Medium** | Restricts CI and diagnostics; most failures are explicit rather than silent. |
| 20 | 14. Process-global simulator | **Medium** | Prevents safe in-process parallelism and limits CI throughput. |
| 21 | 15. Missing GC/finalizer semantics | **Medium** | Can change cleanup and lifetime behavior, especially in long simulations. |
| 22 | 18. No resource/throughput model | **Medium** | Load- and latency-dependent failures remain outside the model. |
| 23 | 23. No per-machine environment | **Medium** | Environment-driven heterogeneous node configuration cannot be represented. |
| 24 | 24. Trace payload exposure | **Medium** | Opt-in syscall traces can leak application and storage secrets into CI artifacts. |

### 1. The Go-version port is a semantic port, not a dependency update — High

The nested module declares Go 1.26.0
([`go.mod`](tools/gomad/go.mod#L1-L4)) and routes hooks through the stable
`internal/stdlib/hooks` package. Release-specific translator policy lives in
[`policy_go126.go`](tools/gomad/internal/translate/policy_go126.go), but the
hooks still target unexported standard-library/runtime symbols, whose signatures
have no compatibility guarantee
([`docs/design.md`](tools/gomad/docs/design.md#L173-L188)). The translator also
contains Go-1.26-specific hook, accepted-linkname, assembly, and package-skip
tables.

A port must validate more than compilation:

- every hooked symbol still has the expected signature and calling convention;
- linkname targets and empty assembly declarations still resolve correctly;
- synchronization and timer invariants still match the new standard library;
- architecture-specific generated syscall wrappers remain ABI-correct; and
- race-detector annotations still describe the correct happens-before edges.

Without a generated compatibility inventory and conformance tests, a successful
build can still hide semantic drift.

### 2. "Translate the standard library" has explicit escape boundaries — Critical

The normal translation path skips `runtime`, `errors`, `reflect`, `unsafe`,
`testing`, runtime profiling/metrics/coverage packages, and other selected
packages. It also preserves assembly for a package allowlist
([`internal/translate/main.go`](tools/gomad/internal/translate/main.go#L27-L80)).
Imports can opt out through `//gomad:notranslate`
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

There is also a security implication: gomad is not a sandbox. `unsafe` and
several runtime packages remain native, and individual imports can deliberately
opt out of translation
([`internal/translate/main.go`](tools/gomad/internal/translate/main.go#L27-L57),
[`internal/translate/translate.go`](tools/gomad/internal/translate/translate.go#L577-L609)).
It is therefore a correctness harness for trusted code, not an isolation
boundary for untrusted workloads; native escapes run with the host process's
permissions.

### 3. Built-in maps and channels are still replaced types — Critical

Gomad keeps the standard-library implementation above many runtime hooks, but it
does not keep native Go maps and channels. The translator changes their types and
operations to gomad runtime types
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

### 4. Cooperative scheduling cannot control code between yield points — High

The scheduler chooses a runnable simulated goroutine and lets it run until it
yields, blocks through a modeled primitive, finishes, or panics. The runtime
itself notes that uninstrumented synchronization or a spin can hang a step and
that a per-step wall-clock timeout is missing
([`gomadruntime/runtime.go`](tools/gomad/gomadruntime/runtime.go#L234-L275)).

Runtime hooks reduce this risk by moving standard synchronization and polling
onto the simulated scheduler, but they cannot automatically preempt:

- CPU-bound loops without a modeled synchronization point;
- native code reached through a translation escape;
- unsupported assembly or `unsafe` synchronization; or
- a missed runtime/standard-library hook.

This failure mode should be detected as an explicit boundary violation with the
responsible goroutine and native stack, not left to a whole-test timeout.

### 5. Per-machine global isolation is conditional — Critical

Gomad rewrites package globals into per-machine containers and re-runs package
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

### 6. Runtime and standard-library hook coverage is incomplete — High

Several hook packages intentionally panic for unsupported operations, including
parts of `runtime/debug`, `runtime/trace`, internal ABI/CPU helpers, `os`,
`syscall`, polling, and `x/sys/unix`
([`internal/stdlib/hooks/runtime_debug.go`](tools/gomad/internal/stdlib/hooks/runtime_debug.go#L1-L42),
[`internal/stdlib/hooks/runtime_trace.go`](tools/gomad/internal/stdlib/hooks/runtime_trace.go#L1-L16),
[`internal/stdlib/hooks/syscall.go`](tools/gomad/internal/stdlib/hooks/syscall.go#L1-L88)).

At the OS boundary, unknown raw syscalls are logged and return `ENOSYS`
([`internal/simulation/os_linux.go`](tools/gomad/internal/simulation/os_linux.go#L162-L175)).
Package loading forces Linux, the host architecture, the `gomad` build tag, and
`CGO_ENABLED=0`
([`gomadmain/main.go`](tools/gomad/gomadmain/main.go#L155-L159),
[`internal/translate/main.go`](tools/gomad/internal/translate/main.go#L120-L136)).

These are hard compatibility limits. Every target workload needs an executed
inventory of hooks and syscalls classified as simulated, native, `ENOSYS`,
panic, or unreachable.

### 7. The simulated clock is global — Medium

The runtime uses one clock for all machines. The source explicitly leaves
per-machine clock offsets and broken clocks as future work
([`gomadruntime/time.go`](tools/gomad/gomadruntime/time.go#L1-L35)). The current
model can accelerate time and order timers deterministically, but it cannot
model:

- clock skew or drift between machines;
- wall-clock jumps;
- a clock that pauses across a machine crash; or
- different wall and monotonic clock failures.

This is a runtime-model gap rather than a network or filesystem omission.

### 8. The Linux mock is intentionally narrow — High

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

### 9. Determinism checking does not cover complete simulation state — High

The checksum records scheduler picks and results, log writes, goroutine creation,
run boundaries, and clock advancement
([`gomadruntime/checksum.go`](tools/gomad/gomadruntime/checksum.go#L23-L35)). It
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

### 10. Schedule exploration is reproducible but not systematic — Medium

The scheduler uses a seeded random choice among runnable goroutines
([`gomadruntime/runtime.go`](tools/gomad/gomadruntime/runtime.go#L257-L275)). Seed
ranges replay or vary that choice, but there is no coverage metric, state-space
reduction, schedule bounding, shrinking, or PCT-style guarantee. The design
mentions PCT only as related work
([`docs/design.md`](tools/gomad/docs/design.md#L502-L514)).

Deterministic replay and exploration quality are separate properties. Gomad has
the first; the snapshot does not define the second beyond randomized seeds.

### 11. The replacement `testing` runtime is incomplete — Medium

Because translated tests need simulator-owned execution, gomad carries a copied
`testing` implementation. The runner supports only a small subset of standard
test flags and rejects other supplied `-test.*` flags
([`gomadruntime/testmain.go`](tools/gomad/gomadruntime/testmain.go#L59-L71),
[`gomadruntime/testmain.go`](tools/gomad/gomadruntime/testmain.go#L111-L122)).
Coverage, benchmarks, examples, and several entry points are unimplemented
([`internal/testing/missing.go`](tools/gomad/internal/testing/missing.go#L1-L45));
`T.TempDir` and `T.Setenv` panic
([`internal/testing/testing.go`](tools/gomad/internal/testing/testing.go#L642-L706)).

Metatesting requires prebuilt test binaries for cached runs and lists fuzzing and
parallelism as future work
([`metatesting/doc.go`](tools/gomad/metatesting/doc.go#L14-L44)). This expands the
Go-version maintenance surface beyond runtime hooks and affects normal CI use.

### 12. There is no compatibility or fidelity report — High

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

### 13. Test breadth is high, but coverage evidence is incomplete — High

The runtime-simulation implementation has a large test corpus, but raw counts
are misleading: many cases are subtests, some sources are selected by build
constraints, the race corpus dominates the number of top-level tests, and a
test run generates another translated tree under `.gomad`. The useful inventory
is by responsibility:

| Suite | Relative breadth | What it establishes |
| --- | --- | --- |
| `internal/tests/behavior` | Broad | Language, globals, time, network, filesystem, crash, logging, and library behavior |
| `internal/tests/race` | Broad, dominated by copied race cases | Race-detector behavior across channels, maps, synchronization, time, I/O, and atomics |
| `gomadruntime` | Narrow direct unit suite | Map, timer-heap, random, and scheduler cleanup properties |
| `internal/translate` | Golden and compatibility suite | Translation output, filename handling, caching, and Go-version-specific behavior |
| Script, metatesting, nemesis, and examples | Small end-to-end suite | CLI behavior, repeated execution, fault injection, and representative workloads |

The scenario coverage is strongest for rewritten maps and channels, timers,
TCP/HTTP/gRPC, disk/crash behavior, and race edges. The behavior suite is much
thinner for synchronization: its only direct functional test combines a mutex
and WaitGroup, logs the schedule, and contains no behavioral assertion
([`sync_test.go`](tools/gomad/internal/tests/behavior/sync_test.go#L10-L30)). The
OS behavior tests cover only PID and hostname
([`os_test.go`](tools/gomad/internal/tests/behavior/os_test.go#L8-L20)), and the
network suite still records missing nonexistent-destination and buffer-overflow
cases
([`net_test.go`](tools/gomad/internal/tests/behavior/net_test.go#L26-L28)).

Every behavior test is replayed twice with seed 1 and compared by checksum and
log output, then exercised with five seeds
([`meta_test.go`](tools/gomad/internal/tests/behavior/meta_test.go#L11-L18),
[`metatest.go`](tools/gomad/metatesting/metatest.go#L259-L317)). This is useful
determinism and scenario exploration, but it is not a code-coverage metric or a
native-versus-simulated oracle. The replacement `testing` package leaves
`CoverMode`, `Coverage`, and `RegisterCover` unimplemented
([`missing.go`](tools/gomad/internal/testing/missing.go#L7-L25)), and there is no
checked-in coverage profile or boundary-coverage report. A large test count
therefore does not show which translator, hook, runtime, or syscall paths remain
untouched.

The seed harness also has a concrete correctness bug: `CheckSeeds` accepts a
`numSeeds` argument but ignores it and always runs five seeds
([`metatest.go`](tools/gomad/metatesting/metatest.go#L295-L317)). A caller can
therefore request broader exploration and receive less coverage without an
error.

The copied race corpus is also run under only the default seed, despite a TODO
calling for multiple seeds
([`race_test.go`](tools/gomad/internal/tests/race/race_test.go#L42-L49),
[`testmain.go`](tools/gomad/gomadruntime/testmain.go#L99-L102)). The largest
concurrency corpus therefore checks many primitives but explores only one
simulated schedule per test.

### 14. The simulator and metatest protocol are process-global — Medium

The runtime rejects a second active simulator in the same process
([`gomadruntime/runtime.go`](tools/gomad/gomadruntime/runtime.go#L52-L73)). The
metatest layer reuses one child process and one JSON encoder/decoder per package,
but explicitly lacks synchronization for concurrent `Run` calls and leaves
runner shutdown commented out
([`metatesting/metatest.go`](tools/gomad/metatesting/metatest.go#L127-L162)).

This prevents safe `t.Parallel` execution, concurrent seed workers, and
in-process embedding. Accidental concurrency can panic the runtime or interleave
the request/response stream rather than returning a structured error.

**Opportunity:** either make scheduler/global state instance-owned, or declare
process isolation as the contract and implement a bounded worker-process pool
with serialized requests, restart-on-protocol-error, explicit `Close`, and
same-seed isolation tests.

### 15. GC, finalizer, and object-lifetime semantics are not modeled — Medium

Calls to `runtime.SetFinalizer` are rewritten to gomad, where the implementation
is an unconditional no-op
([`internal/translate/main.go`](tools/gomad/internal/translate/main.go#L96-L104),
[`gomadruntime/runtime.go`](tools/gomad/gomadruntime/runtime.go#L937-L942)). The
host GC still manages physical memory, but its timing is outside the simulated
schedule and cannot create deterministic cleanup or memory-pressure events.

Libraries that use finalizers as a resource-cleanup fallback can retain file,
poller, or network state for an entire run. Heap pressure, GC pauses, cleanup
ordering, and lifetime-sensitive failures are outside the model.

**Opportunity:** publish this as an explicit compatibility limit and fail or
trace finalizer registration. If lifetime behavior matters, introduce explicit
deterministic GC checkpoints and resource-leak assertions rather than trying to
mirror the host collector implicitly.

### 16. Crash-state exploration is exponential and resource lifecycle is incomplete — High

`IterDiskCrashStates` recursively explores the include/exclude choice for each
pending filesystem operation, with dependency pruning but no state budget,
sampling policy, deduplication, or shrinker
([`pendingops.go`](tools/gomad/internal/simulation/fs/pendingops.go#L306-L373)).
The iterator also reads the live pending-operation graph without cloning it
([`pendingops.go`](tools/gomad/internal/simulation/fs/pendingops.go#L306-L310)).

Each yielded crash state creates and starts another machine; machines are added
to `machinesById` and are never removed, while filesystem `Release` is empty
([`gomad.go`](tools/gomad/internal/simulation/gomad.go#L62-L92),
[`simulation.go`](tools/gomad/internal/simulation/simulation.go#L54-L80),
[`filesystem.go`](tools/gomad/internal/simulation/fs/filesystem.go#L504-L506)).
The iterator itself runs in a goroutine with its stop method commented out, so
breaking out of iteration early can leave it blocked indefinitely
([`filesystem.go`](tools/gomad/internal/simulation/fs/filesystem.go#L471-L502),
[`filesystem.go`](tools/gomad/internal/simulation/fs/filesystem.go#L545-L568)).
For a realistic write set, enumeration can grow exponentially while retaining
machines, disks, goroutines, and associated runtime state.

**Opportunity:** snapshot the pending-op graph before iteration; add maximum
states/time/memory budgets, deterministic sampling and shrinking, state hashes
for deduplication, and explicit disposal of yielded machines and filesystems.
Expose explored/pruned/retained state counts in test output.

### 17. Fault-control inputs can silently change the requested scenario — High

The machine-creation path discards `netip.ParseAddr` errors; an invalid supplied
address becomes an automatically allocated address
([`gomad.go`](tools/gomad/internal/simulation/gomad.go#L45-L59),
[`simulation.go`](tools/gomad/internal/simulation/simulation.go#L62-L64)). More
directly, `MachineSetSometimesCrashOnSync` ignores its Boolean argument and
always enables crashing, so passing `false` cannot disable the fault
([`gomad.go`](tools/gomad/internal/simulation/gomad.go#L180-L189)). Machine
lookups also assume every ID exists instead of returning a typed stale/unknown
handle error.

These are control-plane failures: the simulator may run a different topology or
fault policy than the test requested while still producing a normal-looking
trace. That makes resulting success and failure conclusions unreliable.

**Opportunity:** validate every control input, return typed errors for invalid
addresses and machine state transitions, honor disable operations, and record
the effective topology/fault configuration in the determinism trace. Add
round-trip tests for enable/disable, invalid handles, duplicate addresses, and
restart transitions.

### 18. Simulated time has no CPU, disk, or bandwidth cost model — Medium

The clock advances only when no simulated goroutine is runnable and a timer is
waiting
([`gomadruntime/runtime.go`](tools/gomad/gomadruntime/runtime.go#L245-L259)).
Terminating CPU work therefore consumes zero simulated time, filesystem calls
apply synchronously without a configurable latency, and the network exposes a
constant delay rather than bandwidth or queue-service costs. This is separate
from the infinite-loop problem in gap 4: even cooperative workloads cannot
model slow execution under load.

Timeout, retry, election, and backpressure bugs that depend on CPU saturation,
disk latency, bandwidth, queueing, or memory pressure will not emerge from the
current model.

**Opportunity:** define the intended performance boundary explicitly. If these
failures are in scope, add deterministic operation-cost hooks, disk latency,
bandwidth/queue limits, and resource budgets; otherwise require complementary
load and fault-injection tests outside gomad.

### 19. Atomic operations are not scheduling points — High

Every `sync/atomic` hook calls `maybeAtomicYield`, but yielding is disabled by
the compile-time `AtomicYield = false` constant
([`sync_atomic.go`](tools/gomad/internal/stdlib/hooks/sync_atomic.go#L11-L24),
[`sema.go`](tools/gomad/gomadruntime/sema.go#L153-L159)). Runtime acquire/release
helpers and `procPin`/`procUnpin` similarly avoid a scheduling choice
([`sync.go`](tools/gomad/internal/stdlib/hooks/sync.go#L31-L75)).

As a result, a lock-free loop or atomic state machine can run from one unrelated
yield point to the next as one scheduler step. The execution may be a legal Go
schedule, but gomad will not explore the interleavings at the atomic operations
where lock-free bugs usually occur; a CAS retry loop can also become the
uninterruptible-spin failure from gap 4.

**Opportunity:** make atomic preemption configurable and seed-controlled, then
add bounded pre/post-operation scheduling points for atomic loads, stores, swaps,
and CAS failures. Validate with small lock-free litmus tests and report atomic
scheduling-point coverage separately from runnable-goroutine seed coverage.

### 20. Delayed packets can cross restart and connection generations — Critical

Network connection identity contains only source/destination addresses and
ports. The implementation explicitly notes that it lacks an extra connection
generation or TCP sequence mechanism for delayed packets across machine restart
([`stack.go`](tools/gomad/internal/simulation/network/stack.go#L24-L34)). Each
restart creates a fresh network stack whose ephemeral port counter starts again
at 10000
([`simulation.go`](tools/gomad/internal/simulation/simulation.go#L83-L90),
[`stack.go`](tools/gomad/internal/simulation/network/stack.go#L114-L127)), while
queued packets are delivered to whichever stack currently owns the destination
address
([`net.go`](tools/gomad/internal/simulation/network/net.go#L200-L226)).

A delayed pre-crash packet can therefore reach the post-restart stack and, if
the address/port tuple has been reused, be interpreted as part of a new
connection. This silently violates the machine crash boundary and can create
application behavior that real TCP sequence/epoch handling would reject.

**Opportunity:** include endpoint and connection generation IDs in every packet,
invalidate old generations when a stack detaches, and model enough sequence or
TIME_WAIT behavior to reject stale traffic. Add a regression that delays data,
crashes and restarts the destination, reuses the same ports, and proves the old
data cannot enter the new stream.

### 21. A machine crash closes TCP streams gracefully — High

The public API promises that `Crash` does not properly close open network
connections, and the lifecycle correctly passes `graceful=false` into network
shutdown
([`machine.go`](tools/gomad/machine.go#L69-L78),
[`simulation.go`](tools/gomad/internal/simulation/simulation.go#L124-L130)).
However, `Stack.Shutdown` ignores that argument and sends a stream-close packet
for every connection regardless
([`stack.go`](tools/gomad/internal/simulation/network/stack.go#L74-L103)).

Peers therefore observe a clean close after a crash instead of a half-open
connection that fails through retransmission, keepalive, or an application
deadline. This systematically hides reconnect, failure-detector, and retry bugs
in the workload class the simulator is intended to exercise.

**Opportunity:** make harsh shutdown detach and discard local network state
without sending close packets. Define the fate of already queued packets using
the endpoint generations from gap 20, then test an open connection across crash
for peer timeouts, keepalive failure, and retry behavior.

### 22. Normal machine return can hide abandoned work and durability bugs — High

When a machine's main function returns, gomad performs a graceful stop: it
first aborts every remaining goroutine and timer, then closes network streams
and copies the entire in-memory filesystem to persisted state
([`machine.go`](tools/gomad/machine.go#L19-L25),
[`simulation.go`](tools/gomad/internal/simulation/simulation.go#L96-L135),
[`filesystem.go`](tools/gomad/internal/simulation/fs/filesystem.go#L508-L519)).
The top-level runtime also converts `ErrMainReturned` to success and only leaves
an `AssertAllDone`-style quiescence check as a TODO
([`runtime.go`](tools/gomad/gomadruntime/runtime.go#L390-L393),
[`runtime.go`](tools/gomad/gomadruntime/runtime.go#L1046-L1047)).

A server test can consequently report success while background work is still
blocked or runnable, defers are skipped, resources remain open, and writes that
were never explicitly synced become durable. Clean shutdown is a useful mode,
but coupling it to ordinary function return makes both lifecycle and crash
results look stronger than the application earned.

**Opportunity:** represent termination explicitly as clean return, requested
stop, crash, or test abort. By default, diagnose unexpected goroutines, timers,
syscalls, descriptors, and connections before teardown; only force persistence
for an explicitly selected clean-stop policy. Include leftover work and the
chosen durability action in the trace.

### 23. Environment configuration is simulator-wide, not per-machine — Medium

Environment variables live on the scheduler and every machine reads the same
slice
([`runtime.go`](tools/gomad/gomadruntime/runtime.go#L92-L143),
[`runtime.go`](tools/gomad/gomadruntime/runtime.go#L996-L999)). `MachineConfig`
has no environment field, and runtime `setenv`, `unsetenv`, and `clearenv` hooks
panic as unimplemented
([`machine.go`](tools/gomad/machine.go#L46-L57),
[`syscall.go`](tools/gomad/internal/stdlib/hooks/syscall.go#L47-L57)).

Tests cannot model a cluster whose nodes have different feature flags,
credentials, regions, or rolling configuration. Code that mutates its
environment at runtime also fails rather than retaining process-local state.

**Opportunity:** add environment, arguments, and working directory to
`MachineConfig`; define whether mutations survive restart; isolate mutations by
machine; and fingerprint the effective environment without recording secret
values. Test heterogeneous nodes and restart behavior.

### 24. Syscall traces can expose raw application data — Medium

The syscall logger encodes complete read and write buffers as base64 attributes,
including positional I/O
([`os_linux.go`](tools/gomad/internal/simulation/os_linux.go#L96-L98),
[`os_linux.go`](tools/gomad/internal/simulation/os_linux.go#L618-L624),
[`os_linux.go`](tools/gomad/internal/simulation/os_linux.go#L673-L678),
[`os_linux.go`](tools/gomad/internal/simulation/os_linux.go#L812-L851)). The
JSON trace file is created with mode `0644`
([`testmain.go`](tools/gomad/gomadruntime/testmain.go#L141-L166)).

Enabling a diagnostic trace can therefore copy credentials, tokens, request
bodies, and stored records into world-readable local files or retained CI
artifacts. The trace is opt-in, which limits likelihood, but the exposure is
silent once enabled.

**Opportunity:** record payload lengths and hashes by default; require an
explicit option for raw payloads; support field- and buffer-level redaction;
create local trace files as `0600`; and document artifact retention and secret
scanning expectations.

## Reusing tests from `tools/gomad_old`

The legacy suite is useful as a list of semantic contracts, not as a harness to
copy. Runtime-simulation tests should use ordinary Go syntax so the current
translator and standard-library hooks are exercised. The legacy `SIMLANG`,
`SIMLIB`, `SIM.Join`, and `StressRun` calls should be removed; the current
metatesting harness already handles same-seed replay and seed variation. The
legacy transform-output goldens should not be ported because they assert the old
rewrite strategy rather than runtime-simulation behavior.

| Priority | Area | What can be borrowed | Required adaptation |
| --- | --- | --- | --- |
| P0 | Channels and `select` | Duplicate receive cases for one channel, multiple send cases, close waking multiple waiters, receive-until-close, and dynamically setting a channel to nil ([`select_test.go`](tools/gomad_old/api/lang/select_test.go#L113-L199), [`select_test.go`](tools/gomad_old/api/lang/select_test.go#L255-L337), [`channel_test.go`](tools/gomad_old/api/lang/channel_test.go#L191-L231)) | Express with native channels, `select`, and completion channels or WaitGroups. These cover TODOs still present in the current channel suite. |
| P0 | `sync` functional semantics | `TryLock`, reader/writer exclusion, `Cond.Signal`, `Cond.Broadcast`, `Once`, `OnceFunc`, `OnceValue`, and `OnceValues` ([`sync_test.go`](tools/gomad_old/api/lib/sync_test.go#L145-L382)) | Use stock `sync` types and make the Once cases concurrent. Repair the donor defects described below. |
| P0 | Network failure and gRPC cancellation | Dialing without a listener and unary interceptor/context scenarios ([`net_test.go`](tools/gomad_old/api/lib/net_test.go#L80-L84), [`fakegrpc_test.go`](tools/gomad_old/api/ext-lib/fakegprc/fakegrpc_test.go#L26-L57)) | Use native `net` and the existing real gRPC-over-simulated-TCP fixture. Assert error class rather than the old fake-network string, and delay the handler so the deadline actually expires. |
| P1 | Timer scheduling | Concurrent timers completing in deadline order, a timer firing without a receiver, and `time.Tick` ([`timer_test.go`](tools/gomad_old/api/lib/timer_test.go#L39-L73), [`ticker_test.go`](tools/gomad_old/api/lib/ticker_test.go#L124-L139)) | Keep only behavior guaranteed by Go. Revalidate `Timer.Stop` and `Timer.Reset` return values against the target Go release instead of copying old mock assumptions. |
| P1 | WaitGroup and closed-channel oracles | Empty WaitGroup, fanout, send-on-closed, and double-close cases ([`waitgroup_test.go`](tools/gomad_old/api/lib/waitgroup_test.go#L39-L123), [`channel_test.go`](tools/gomad_old/api/lang/channel_test.go#L243-L263)) | Prefer the fanout composition; negative-counter behavior already appears in the race corpus. Assert panic type/occurrence, not simulator-specific message text or channel IDs. |
| P2 | Maps, randomness, context, reflection, and basic goroutines | Little incremental coverage | Do not port wholesale. Current map, random, context, reflection, and goroutine coverage is already broader; add only a same-seed/different-seed map-iteration meta-test if raw iteration order needs an explicit contract. |
| Conditional | Concurrent simulations | Same-seed isolation across concurrent simulator instances ([`concurrent_test.go`](tools/gomad_old/runtime/concurrent_test.go#L12-L47)) | This is an acceptance test only if in-process concurrency becomes a goal. The current runtime explicitly permits only one active simulator per process ([`runtime.go`](tools/gomad/gomadruntime/runtime.go#L52-L73)); otherwise add a test that documents that restriction. |
| Not applicable | Checkpoint/restore | Execution replay, branching, channel-buffer restore, and DRNG restore tests | Runtime simulation exposes machine crash/restart, not execution snapshots. These tests require a new checkpoint feature and should not be presented as crash fidelity tests. |

The donor tests must not be copied verbatim. In particular:

- its nil-channel send and receive tests expect panics, while Go and the current
  runtime correctly require blocking
  ([`channel_test.go`](tools/gomad_old/api/lang/channel_test.go#L233-L241),
  [`channel_test.go`](tools/gomad_old/api/lang/channel_test.go#L275-L283));
- its basic locker test creates a different mutex in each goroutine, so it does
  not test mutual exclusion
  ([`sync_test.go`](tools/gomad_old/api/lib/sync_test.go#L46-L66));
- its `Cond.Broadcast` condition is `counter == counter`, which is always true,
  so no goroutine waits
  ([`sync_test.go`](tools/gomad_old/api/lib/sync_test.go#L293-L319)); and
- several cases append to shared slices from multiple goroutines or assert a
  scheduler order that the Go specification does not guarantee.

The highest-value first tranche is therefore small: native-syntax functional
tests for `TryLock`, the Once family, repaired condition variables, the missing
channel/select cases, dial-without-listener, and an expiring gRPC deadline. Run
each case both natively and under runtime simulation where the model claims Go
equivalence; leave seed reproducibility to metatesting rather than embedding a
second stress harness in every test.

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

Before treating gomad as a usable runtime-mock platform for a new codebase, the
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
9. **Crash/network isolation:** harsh crashes do not send graceful closes,
   packets cannot cross machine or connection generations, and persistence
   follows the selected termination mode.
10. **Fault-control correctness:** invalid inputs and state transitions return
    explicit errors, enable/disable controls are symmetric, and the effective
    scenario is fingerprinted.
11. **Bounded exploration:** crash-state enumeration has deterministic budgets,
    cleanup, deduplication, and safe early termination with explored/pruned
    counts.
12. **Completion invariants:** unexpected goroutines, timers, syscalls,
    descriptors, and connections fail with actionable diagnostics instead of
    being silently aborted.
13. **Atomic exploration:** atomic scheduling points are configurable and
    bounded, and the race corpus runs across multiple schedule seeds.

The core conclusion is narrower than "mock the runtime and inherit standard
library fidelity." Gomad moves much of the semantic burden below public APIs,
which is a strong design, but it then assumes responsibility for the language
rewrite, per-machine global model, unstable runtime hooks, and Linux syscall
semantics. Those are the gaps that must be closed or made explicit.
