# Gomad v3: Minimal Deterministic Go Runtime

## Decision

Build `tools/gomadv3` around a small patch and source overlay for the Go 1.26.4
runtime. Do not translate source, patch the compiler, replace native Go types,
or simulate the standard library in this phase.

The result is a private Go toolchain whose `go run` and `go test` commands work
normally. Deterministic runtime behavior is activated only when `GOMADSEED` is
set. Without `GOMADSEED`, binaries built by the patched toolchain must behave
like upstream Go 1.26.4.

```text
GOMADSEED=1 make gomadv3-test GOMADV3_PACKAGES=./path/to/package
GOMADSEED=1 make gomadv3-run GOMADV3_RUN=./cmd/example
```

The Makefile supplies the required supporting settings:

```text
GODEBUG=asyncpreemptoff=1
GOMAXPROCS=1
GOMADSEED=1
```

`GOMADSEED` is the only feature gate. `GOMAXPROCS=1` and
`asyncpreemptoff=1` are deterministic-mode invariants, not independent ways to
enable the patch.

## Goal

For a fixed Go 1.26.4 patch, target architecture, program, external inputs, and
`GOMADSEED`, runtime-controlled choices must repeat exactly across separate
invocations of `go run` and `go test`.

The seed controls:

- which runnable goroutine executes next at a natural runtime scheduling point;
- which ready `select` case wins;
- map hash seeding and map iteration order; and
- other runtime choices that can affect synchronization or user-visible
  ordering.

Different seeds should explore different choices when more than one valid
choice exists. The same seed must reproduce the same choices.

## Patch-minimization rules

The Go patch is the most expensive artifact to maintain. Build scripts,
black-box tests, and documentation may live in this repository, but the patch
against upstream Go must stay deliberately tiny.

The first prototype has strict production-patch boundaries:

- make no compiler, `cmd/go`, public-package, map-implementation, channel, GC,
  platform, or assembly changes; and
- keep all v3 behavior tests outside the upstream Go patch.

The intended production files are `src/runtime/gomad.go`,
`src/runtime/rand.go`, and `src/runtime/proc.go`. The net-new `gomad.go` lives
in the source overlay; the patch modifies only upstream files. Additional
runtime changes are not a normal implementation detail. Preserve the failing
black-box test and review whether the determinism claim should be narrowed
before expanding this surface.

Apply these rules in order:

1. Reuse an existing Go runtime decision path.
2. Seed or gate that path with the smallest possible conditional.
3. Validate it through an external v3 test.
4. Add a new runtime hook only when a minimized failing test proves the reused
   path cannot satisfy the contract.

Do not add stable IDs, traces, counters, replacement queues, domain-specific
PRNGs, or diagnostic frameworks to the initial patch. They may be attractive,
but none is required to test the runtime-only hypothesis.

## Exact scope

### In scope

- A patch against the exact upstream `go1.26.4` source tree.
- A locally built, cached Go distribution under `tools/gomadv3`.
- Native goroutines, channels, `select`, maps, and runtime synchronization.
- Seeded selection among goroutines already runnable at runtime scheduling
  points.
- Ordinary `go run` and `go test` workflows using the custom `go` binary.
- Tests proving that patched Go is normal when disabled and deterministic when
  enabled.

### Out of scope

- Compiler changes or inserted scheduling checkpoints.
- AST rewriting and replacement channel or map types.
- Replay, time travel, snapshots, or execution logs.
- Virtual time.
- Network, filesystem, process, signal, and other operating-system simulation.
- Making external I/O completion order deterministic.
- Complete exploration of data races or CPU instructions between runtime safe
  points.
- cgo, plugins, and foreign threads in deterministic mode.
- The race detector in deterministic mode.
- Supporting Go versions other than 1.26.4.

Network, file, clock, and other external APIs will be replaced separately. The
runtime patch must not contain those adapters.

## Determinism contract

The initial claim is intentionally narrower than “every Go program is perfectly
deterministic.” It is:

> Given deterministic external inputs, the patched runtime makes its own
> user-visible scheduling, `select`, map, and synchronization choices a pure
> function of the program execution and `GOMADSEED`.

This has two important consequences:

1. A goroutine can switch only where the unmodified compiler already permits
   the runtime to schedule. With async preemption disabled, a CPU-bound loop may
   run forever and no seed can preempt it.
2. An uncontrolled timer, syscall, network poll, signal, or cgo callback may
   make a goroutine runnable at a host-dependent moment. Those sources must be
   absent or replaced by the separate simulation layer before the complete
   process can be called deterministic.

The runtime patch guarantees repeatable choices over a repeatable runnable set;
it does not make an externally varying runnable set repeatable.

## Activation contract

### Disabled

When `GOMADSEED` is absent:

- initialization follows the upstream Go path;
- runtime entropy remains host-seeded;
- scheduler, `select`, and map behavior remain upstream behavior;
- `GOMAXPROCS` and async preemption retain their normal meanings; and
- all upstream runtime and toolchain tests must continue to pass.

Every change to an upstream hot path should have one predictable guard:

```go
if gomadEnabled {
	// Deterministic choice.
} else {
	// Unmodified upstream path.
}
```

Preserve upstream comments and keep the normal branch structurally unchanged.

### Enabled

When `GOMADSEED` is present:

- parse it as an unsigned 64-bit integer; seed `0` is valid;
- reject an empty or malformed value before user `init` functions run;
- force the initial effective `GOMAXPROCS` to one;
- disable async preemption;
- deterministically seed the runtime's existing random state before maps or
  scheduler choices can consume it.

The Makefile sets `GOMAXPROCS=1` and `GODEBUG=asyncpreemptoff=1` explicitly so
the invocation is self-documenting. The runtime also enforces both invariants,
so direct use of the custom binary cannot accidentally enter a partially
deterministic mode.

Calling `runtime.GOMAXPROCS` to raise the value after startup is unsupported in
the first prototype. Do not patch that public function merely to enforce the
restriction; cover it in the documented deterministic-input contract.

The runtime may print the active seed only when explicitly asked through a
diagnostic setting or on a fatal deterministic-mode error. Normal program
output must not change merely because the mode is active.

## Runtime design

### One small activation module

Add one file, provisionally `src/runtime/gomad.go`, that owns:

- environment parsing and the enabled bit;
- the seed; and
- early deterministic-mode initialization.

Its internal interface should remain close to:

```go
var gomadEnabled bool
var gomadSeed uint64

func gomadInit()
```

Do not introduce a Gomad PRNG. Seed Go's existing global and per-M runtime
random state through the existing `randinit` and `mrandinit` paths. This lets
`rand`, `randn`, `cheaprand`, and `maps_rand` keep their implementations and
existing callers.

Call `gomadInit` immediately before the existing `randinit` call. In
deterministic mode, `randinit` initializes the existing global generator from
`gomadSeed` instead of host entropy. Minimized scheduler tests showed that
runtime startup timing otherwise changes per-M state and the initial
scheduler tick before user initialization. In deterministic mode, seed every
M directly from `gomadSeed`, reinitialize M0 and reset its scheduler tick just
before user package initialization, and disable the system monitor. Keep the
upstream `mrandinit` and system-monitor behavior unchanged when Gomad is off.

A shared runtime stream means an additional random-consuming operation can
change later scheduling choices. That is acceptable for the first contract:
the same program and seed must repeat, but schedules need not remain stable
after the program changes. Domain-separated streams would require more patch
surface and are deferred unless this coupling prevents a concrete use case.

### Scheduler

`GOMAXPROCS=1` removes parallel application execution. To make the seed affect
scheduling without adding a scheduler, reuse Go's existing
`randomizeScheduler` behavior, which already perturbs `runnext` and run-queue
placement for race builds. Change its activation from race-only to race-or-
Gomad, and let it consume the deterministically seeded existing `randn` path.
Concretely, replace the current constant with a variable initialized to
`raceenabled`, then set it in `gomadInit`; do not add conditions throughout the
scheduler.

Do not add a new run queue, candidate list, stable goroutine ID, or scheduler
algorithm. If the existing randomized path is repeatable but produces
insufficient schedule diversity, the only permitted first extension is a small
seeded choice within the existing local `runqget` representation in
`src/runtime/proc.go`. That extension requires a failing diversity test first.

Staged channel and mutex waiters provided that failing test: all seeds retained
FIFO arrival order. In deterministic one-P mode, `runqget` therefore chooses a
seeded offset from the existing local queue and fills that slot from the head.
The upstream concurrent-consumer path remains unchanged.

Do not add compiler checkpoints. Scheduling remains cooperative at the runtime
points already present in Go 1.26.4: blocking, channel operations, semaphore
operations, `runtime.Gosched`, goroutine creation/exit, GC safe points, and
other existing scheduler entries.

### `select`

Do not patch `src/runtime/select.go` initially. Go 1.26.4 already constructs
`pollorder` with `cheaprandn`; deterministic seeding of the existing per-M
runtime state should make this choice repeatable.

Channel pointer ordering used only to prevent deadlock may remain unchanged if
it cannot affect the selected case or program-visible wakeup order. Verify that
assumption across fresh processes with address-space randomization before
considering any `select.go` hunk.

### Maps

Go 1.26.4 maps obtain hash seeds and iterator offsets through
`internal/runtime/maps.rand`, which is already backed by `runtime.maps_rand`
and the runtime's `rand` path. Deterministically seed that existing path; do not
patch `src/internal/runtime/maps` or add a map-specific adapter.

This should control:

- per-map hash seeds;
- reset seeds after clear or clone operations; and
- iterator entry and directory offsets.

Audit runtime hashing for direct randomness, notably special keys such as
floating-point NaNs. Those calls should already use the seeded runtime random
state. Add a new hook only if a black-box same-seed test still diverges.

### Channels and synchronization

Native channel wait queues are ordered by runtime arrival. With one P and a
seeded scheduler, that order should already be repeatable. Do not replace
channels or add random wakeup behavior unless tests demonstrate a remaining
nondeterministic choice.

Runtime semaphore and mutex code already consumes `cheaprand` for relevant
internal choices. Test the deterministically seeded existing path. Do not patch
`sema.go` or lock code unless a minimized same-seed test diverges and proves
that the random seeding is bypassed.

### Garbage collection and runtime goroutines

GC, scavenging, finalizers, timers, and system goroutines can change when an
application goroutine becomes runnable. They must be tested under repeated
process runs before the runtime makes a determinism claim.

Keep this phase minimal:

- do not implement a custom collector;
- serialize application execution with one P;
- exclude finalizer-dependent programs;
- exclude host-timer behavior until time APIs are replaced; and
- add targeted fixes only for runtime-internal choices proven to perturb the
  supported scheduling checksum or output.

If concurrent GC timing still changes supported program behavior, the first
fallback is to disable automatic GC for small deterministic tests and document
the heap limit. A deterministic allocation-count GC trigger is a later runtime
extension, not part of the first patch.

## Expected Go source surface

Keep upstream modifications reviewable as a single patch file against
`go1.26.4`, and keep net-new runtime files in the checked-in overlay. The
expected starting surface is:

| Artifact | Go source file | Purpose |
| --- | --- | --- |
| Overlay | `src/runtime/gomad.go` | Activation and seed parsing |
| Patch | `src/runtime/rand.go` | Seed existing global and per-M random state |
| Patch | `src/runtime/proc.go` | Enforce one P and enable existing scheduler randomization |

The expected patch contains no changes to `select.go`, `sema.go`, channel code,
`internal/runtime/maps`, GC, `cmd/compile`, `cmd/go`, public packages, platform
code, or assembly. Keep regression programs in `tools/gomadv3/testdata` so they
test the built toolchain without enlarging or conflicting with the upstream
source patch.

Any additional production hunk requires a concrete minimized failure, an
explanation of why the existing seeded runtime path cannot fix it, and explicit
review of the expanded maintenance cost.

`tools/gomadv3/test.sh` must enforce the prohibited-area rules for both source
inputs before building. CI fails if `go1.26.4.patch` or `overlay` touches a
prohibited area or contains generated output, or if the patch no longer applies
exactly to pristine Go 1.26.4 source.

## `tools/gomadv3` layout

```text
tools/gomadv3/
  README.md
  Makefile
  build.sh
  go.mod
  go1.26.4.patch
  overlay/
    src/runtime/gomad.go
  test.sh
  testdata/
  .toolchain/       # generated and ignored
```

- `go1.26.4.patch` contains modifications to upstream runtime files; `overlay`
  contains net-new runtime source.
- `build.sh` downloads or reuses the official Go 1.26.4 source archive,
  verifies its checksum, rejects overlay collisions, copies the overlay and
  applies the patch to a fresh tree, and runs `src/make.bash` with an installed
  bootstrap Go.
- `tools/gomadv3/Makefile` turns those steps into file dependencies and stamp
  files.
- `.toolchain/bin/go` is the stable path used by the root Makefile.
- `test.sh` runs upstream-focused patch tests plus v3 black-box determinism
  tests, and enforces the prohibited-area rules.
- `testdata` contains small programs that exercise scheduling, `select`, maps,
  and activation behavior without external I/O.

Generated Go source and binaries must not be committed. The build key must
include:

- Go version;
- source archive checksum;
- patch checksum;
- overlay path-and-content checksum;
- host OS and architecture; and
- bootstrap Go version.

A changed input produces a new toolchain or rebuilds the current one. A stamp
must never claim success after an interrupted patch or build.

## Makefile workflow

Keep the root `Makefile` shallow. It should delegate toolchain construction to
`tools/gomadv3/Makefile` and consistently use one variable for the custom Go
binary:

```make
GOMADV3_GO := $(ROOT)/tools/gomadv3/.toolchain/bin/go

.PHONY: gomadv3-go gomadv3-run gomadv3-test

gomadv3-go:
	$(MAKE) -C tools/gomadv3 toolchain

gomadv3-run: gomadv3-go
	GODEBUG=asyncpreemptoff=1 GOMAXPROCS=1 GOMADSEED=$(GOMADSEED) \
		$(GOMADV3_GO) run $(GOMADV3_RUN) $(GOMADV3_ARGS)

gomadv3-test: gomadv3-go
	GODEBUG=asyncpreemptoff=1 GOMAXPROCS=1 GOMADSEED=$(GOMADSEED) \
		$(GOMADV3_GO) test -tags test_dep $(GOMADV3_PACKAGES) $(GOMADV3_ARGS)
```

The eventual Makefile implementation should validate that `GOMADSEED`,
`GOMADV3_RUN`, or `GOMADV3_PACKAGES` is present where required and return an
actionable error before invoking Go.

Direct use remains possible after one build:

```text
make gomadv3-go
GODEBUG=asyncpreemptoff=1 GOMAXPROCS=1 GOMADSEED=7 \
  tools/gomadv3/.toolchain/bin/go test -tags test_dep ./path/to/package
```

Do not replace the repository's normal `go`, `unit-test`, or build targets in
this phase. v3 is opt-in and side-by-side until its disabled-mode and
determinism tests are established.

## Implementation plan

### Phase 1: Reproducible custom toolchain

- Add the `tools/gomadv3` layout and ignore generated toolchain state.
- Pin the official Go 1.26.4 source URL and checksum.
- Implement atomic download, extraction, patch, and build steps.
- Add nested and root Makefile targets.
- Prove that `gomadv3-go` is incremental and rebuilds after a patch change.
- Prove that a failed or interrupted build leaves no valid success stamp.

Exit criterion: `make gomadv3-go` produces a verified custom
`tools/gomadv3/.toolchain/bin/go` from a clean checkout with one command.

### Phase 2: Activation and deterministic randomness

- Add `src/runtime/gomad.go` through the source overlay.
- Parse `GOMADSEED` early enough to affect runtime and map initialization.
- Enforce one P and disabled async preemption only when active.
- Seed the existing global and per-M runtime random state; add no new PRNG.
- Reset M0 scheduler state before user initialization and disable the runtime
  system monitor only when active.
- Test missing, zero, maximum, malformed, and overflowing seeds.
- Keep these new tests in `tools/gomadv3/testdata` and run the unmodified
  upstream runtime suite with `GOMADSEED` absent.

Exit criterion: without the variable, upstream behavior and tests are
unchanged; with it, the existing runtime random functions have exact golden
sequences.

### Phase 3: Runtime scheduling choices

- Reuse the existing `randomizeScheduler` mechanism in deterministic mode.
- Let it consume the seeded existing `randn` path.
- Verify goroutine spawn, yield, block, ready, close, and exit behavior.
- Use the direct `runqget` choice justified by the staged-waiter diversity
  failure, without changing the upstream multi-P path.
- Add a watchdog test proving that a CPU-bound goroutine is an explicit scope
  limitation rather than a claimed explored schedule.

Exit criterion: same-seed scheduling programs produce identical output in at
least 100 fresh processes, while different seeds exercise multiple schedules.

### Phase 4: `select`, maps, and synchronization

- Verify that existing `cheaprandn` makes `select` repeatable without a patch.
- Verify that existing `maps_rand` makes map hashing and iteration repeatable
  without a patch.
- Audit special map-key hashing for bypasses.
- Verify channel wait queues and semaphore choices before changing either.
- Preserve a minimized failing test and review patch scope before touching
  another production file.

Exit criterion: same-seed `select`, map, channel, and synchronization programs
are stable across fresh processes and deliberate address-space variation.

### Phase 5: `go run` and `go test` usability

- Run both commands only through the custom binary in v3 Makefile targets.
- Verify the seed reaches the program and test binary.
- Determine whether applying `GOMADSEED` to the `go` command and its build tools
  causes any build instability; avoid a `cmd/go` patch unless a test requires
  environment scoping.
- Reuse Gomad's existing seed-range and metatesting ideas only after the
  one-seed interface is reliable.
- Document the deterministic-input requirement and unsupported external
  wakeups.

Exit criterion: a developer can build once, then deterministically run or test
a package by supplying only `GOMADSEED` to the corresponding Make target.

## Verification

The checked-in v3 suite implements the following verification details:

- special map-key coverage includes 32-bit and 64-bit integers, strings,
  floating-point and complex values including NaNs, empty and non-empty
  interfaces, arrays, and structs; every family must repeat for one seed and
  show ordering diversity across seeds `0` through `31`;
- disabled `go run` and a stable prefixed `go test` result are compared with a
  local stock Go 1.26.4 toolchain, with no toolchain download during the test;
- a prebuilt map fixture retains bounded allocation padding and proves that
  different reported layouts preserve the same logical output; this isolated
  address check uses `GOGC=off` because crossing an automatic-GC threshold is
  a separate shared-random-stream input; and
- prebuilt map and scheduler fixtures retain exact same-seed output while
  bounded unrelated CPU-burning processes are alive.

### Disabled-mode compatibility

- Build and run the upstream Go runtime tests with `GOMADSEED` absent.
- Compare representative `go run` and `go test` results with stock Go 1.26.4.
- Verify default `GOMAXPROCS`, async preemption, map randomness, and scheduler
  behavior were not forced into deterministic mode.
- Run the repository's focused tests using `-tags test_dep`.

### Enabled-mode repeatability

For each case, launch at least 100 fresh processes for the same seed and compare
complete output and exit status:

| Area | Cases |
| --- | --- |
| Scheduler | spawn tree, repeated `Gosched`, block/unblock, channel close |
| Seeds | `0`, `1`, maximum `uint64`, ranges of ordinary seeds |
| `select` | two ready receives, ready send/receive, default, nil cases |
| Maps | creation, clear, clone, repeated iteration, float keys including NaN |
| Channels | buffered, unbuffered, multiple waiters, close wakeups |
| Sync | mutex and semaphore contention that can change visible ordering |
| Commands | direct binary, `go run`, `go test`, cached and uncached builds |
| Isolation | sequential and parallel child processes with different seeds |

For different seeds, require schedule or choice diversity only in programs that
present multiple choices. A seed is not useful if every test is forced into the
same FIFO execution.

### Host perturbation

Repeat the suite with:

- warm and cold Go build caches;
- unrelated host CPU load;
- fresh processes and address-space layouts; and
- parallel seed processes.

Any same-seed divergence must be reduced to the first runtime choice that
differs before expanding the feature scope.

## Failure modes

### External readiness changes the runnable set

The seeded scheduler cannot compensate for host-dependent I/O or timer
completion. Tests in this phase must not use those sources inside the claimed
deterministic region. Their later replacements must convert them into
deterministic runtime readiness events.

### The `go` tool itself sees `GOMADSEED`

Environment variables supplied to `go run` and `go test` also reach the custom
`go` command and potentially its helper processes. This is acceptable for the
minimal runtime-only prototype if builds remain correct and repeatable. If it
does not, prefer a small external wrapper that scopes `GOMADSEED` to the final
program rather than patching `cmd/go`.

### A CPU-bound goroutine never yields

This is expected with async preemption disabled and no compiler patch. Detect it
with a wall-clock watchdog in the outer test harness and report the limitation.
Wall time must never become a scheduler input.

### GC or a runtime service goroutine perturbs choices

First reproduce the differing runtime choice. If runtime timing changes the
runnable set, either add the smallest targeted runtime fix or exclude that
behavior from the initial contract. Do not introduce domain streams or silently
broaden the patch into a custom runtime.

### A patch applies with fuzz or to the wrong Go version

Require the exact Go version and source archive checksum. Apply the patch with
zero fuzz and validate expected source hashes before building. Delete incomplete
generated trees and publish the success stamp only after toolchain and smoke
tests pass.

## Trade-offs

### Simplicity

This is substantially smaller than compiler instrumentation or AST translation.
Programs retain native Go types and use the normal `go` command. The cost is a
narrower determinism contract and dependence on the unmodified compiler's
existing scheduling points.

### Performance and scale

One P reduces throughput. Reusing existing random and scheduler paths avoids a
second queue or per-choice allocation. This mode is for tests; run different
seeds in separate processes to use multiple cores.

### Maintenance

The patch is pinned to Go 1.26.4 and touches unstable runtime internals.
Prohibited-area checks, unmodified upstream normal paths, and external
black-box tests are adoption requirements, not aspirations. Patch growth must
be justified by a minimized failing test rather than incremental accretion.

### Security

Deterministic map seeds remove a production hash-randomization defense. The
mode is for trusted test programs only. Never set `GOMADSEED` in production.

## Success criteria

The v3 experiment succeeds when:

- one Make target builds and caches the patched Go 1.26.4 toolchain;
- `GOMADSEED` is the only activation switch;
- absence of `GOMADSEED` preserves upstream behavior;
- the Makefile makes `go run` and `go test` easy to invoke with one seed;
- scheduler, `select`, map, and proven synchronization choices repeat for the
  same seed across fresh processes;
- different seeds produce different schedules where alternatives exist;
- external I/O and compiler instrumentation remain outside the patch; and
- every production Go patch hunk is justified by a minimized failing test and
  stays outside the prohibited areas.

Only after these criteria pass should Gomad consider virtual time, external API
replacement integration, seed ranges, replay, or compiler-assisted scheduling.
