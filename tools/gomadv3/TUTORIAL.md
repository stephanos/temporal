# Gomad v3: from a flaky schedule to a replayable artifact

Concurrency bugs have an annoying habit: the interesting execution happens once,
the test fails, and then the same test passes for the rest of the afternoon.

Gomad v3 is an attempt to make that execution something you can keep.

At a high level, Gomad runs an ordinary Go program with a specially built Go
runtime. That runtime makes a useful set of normally unpredictable decisions
from a seed: which runnable goroutine goes next, how a `select` polls its cases,
how maps are randomized, and how equal-deadline timers are ordered. It also gives
the process a virtual clock and routes a reviewed set of external operations
through deterministic implementations.

Run the same target with the same seed and inputs, and those controlled decisions
repeat. Run it with many seeds, and you sample different executions. When one
fails, Gomad stores the exact binary and the evidence needed to inspect and
replay it.

That is the basic idea. The rest of this tutorial follows one Go test all the way
through the system so the individual pieces make sense together.

## The short version

A normal Gomad session looks like this:

```sh
make gomadv3

tools/gomadv3/.bin/gomad doctor

tools/gomadv3/.bin/gomad analyze \
  --build-tag test_dep \
  go-test ./path/to/package -- '-test.run=^TestSomethingConcurrent$'

tools/gomadv3/.bin/gomad explore \
  --choices \
  --seeds 0-99 \
  --build-tag test_dep \
  go-test ./path/to/package -- '-test.run=^TestSomethingConcurrent$'
```

If a seed fails, Gomad prints an artifact path and a replay command:

```sh
tools/gomadv3/.bin/gomad inspect --choices .gomad/artifacts/v1/campaign-.../failures/sha256-...
tools/gomadv3/.bin/gomad replay .gomad/artifacts/v1/campaign-.../failures/sha256-...
```

Here is what happened behind those commands:

```text
gomad CLI
   |
   +-- checks the installation and target's capability boundary
   |
   +-- builds one immutable test binary with the pinned Go toolchain
   |
   +-- creates a durable Campaign plan and journal
   |
   +-- launches one fresh, supervised process per seed
           |
           +-- patched Go runtime controls scheduling and virtual time
           +-- deterministic I/O handles reviewed external operations
           +-- optional World code models explicit external events
           |
           +-- stdout, stderr, I/O, World, and choice evidence flow back
   |
   +-- classifies results and publishes content-addressed artifacts
   |
   +-- replay validates the artifact, runs its stored binary, forces recorded
       runtime choices when available, and compares the new observation
```

The important theme is ownership. Gomad is not one giant simulation loop. The
runtime controls runtime decisions, the Runner controls processes, deterministic
I/O controls transparent external effects, World controls explicitly modeled
events, and artifacts tie their evidence together.

## First, Gomad builds its own Go

Gomad is not a test wrapper around the stock `go` command. Its core behavior
lives inside a pinned Go 1.26.4 toolchain containing a small runtime patch and a
source overlay.

Building Gomad from the repository root creates both that toolchain and the CLI:

```sh
make gomadv3
```

The generated files and toolchain stay under `tools/gomadv3/.toolchain`; the CLI
is written to `tools/gomadv3/.bin/gomad`.

Why patch Go itself? Because the Go runtime already owns goroutines, channels,
`select`, timers, synchronization, maps, package initialization, and the test
harness. Controlling those mechanisms below the application lets Gomad run
ordinary code without asking it to swap every primitive for a Gomad-specific
version. Your test still uses `go`, `time.Sleep`, channels, contexts, and
`testing.T`.

The custom toolchain is opt-in. A program built with it follows normal upstream
runtime paths unless Gomad activation is present, so merely using the compiler
does not turn every binary into a deterministic Execution.

You can activate the runtime directly with `GOMADSEED=7 ./binary`, which is
handy for low-level experiments. The CLI is the normal user path because it also
provides target review, deterministic I/O, process containment, artifacts, and
replay.

The complete Runner contract is currently qualified on `darwin/arm64`. `doctor`
checks that contract before you spend time on a campaign:

```sh
tools/gomadv3/.bin/gomad doctor
```

It verifies the host, pinned toolchain, Runner identity, deterministic-I/O
boundary, adapters, and artifact directory. Think of it as a preflight check,
not a target test.

## Before running, Gomad asks whether the target fits

Determinism is only honest if Gomad knows which sources of nondeterminism it
controls. The `analyze` command performs that review without compiling or
executing the target:

```sh
tools/gomadv3/.bin/gomad analyze \
  --build-tag test_dep \
  go-test ./path/to/package -- '-test.run=^TestSomethingConcurrent$'
```

Gomad asks the pinned `go` command for the complete package closure, including
test-only packages. It records package and module identities, source hashes,
foreign source files, build tags, `linkname` directives, and the generated test
main. It then applies a fail-closed policy to that evidence.

"Fail closed" means an unknown escape hatch is a rejection, not a best effort.
A target with an unapproved native dependency, forbidden import, foreign source,
or unsupported boundary does not quietly run against the host and call the
result deterministic. Exact compatibility packs can approve known versions and
source identities, but they are specific evidence rather than broad exceptions.

This is also why `gomad analyze` is useful even when you are not ready to run a
campaign: it tells you whether the target is supported and shows the shortest
dependency path to each blocker.

## Preparation happens once

When `explore` starts, the Runner repeats the capability review and prepares the
target before any deterministic execution begins.

For `go-test`, it builds one test executable. For `go-run`, it builds one program
executable. A prebuilt `exec` target is also supported, but it must arrive with
trusted v3 provenance describing the same reviewed closure and binding that
claim to the exact binary bytes.

The build itself is deliberately outside deterministic mode. Gomad is trying to
control the program under test, not make the Go compiler part of the schedule.
Once built, the target is made read-only, hashed, and recorded with its arguments,
build information, selected compatibility packs, adapters, toolchain build key,
Go version, operating system, and architecture.

Every seed in the campaign uses that same prepared binary. This matters more
than it may seem: rebuilding per seed could introduce a second source of
difference that has nothing to do with concurrency.

## A seed is a repeatable set of runtime decisions

Now the interesting part starts.

Gomad accepts a `uint64` seed; zero is valid. `--seeds 0-99` selects an explicit
range, while `--count 100` is shorthand for seeds 0 through 99.

The seed is not application input and it is not a promise that seed 42 maps to
one timeless, universal schedule. It initializes the decision machinery inside
this exact runtime and target. If you change the program or toolchain, earlier
choices can disappear or new ones can be introduced, changing everything that
comes later.

For an unchanged target and execution identity, the seed controls runtime-owned
nondeterminism such as:

- which runnable goroutine is selected when alternatives exist;
- scheduler shuffles and related tie-breaking;
- the polling order of `select` cases;
- map hashing and iteration randomness; and
- ties between timers with the same deadline.

Different seeds may produce different executions, but they do not have to. If a
test reaches no meaningful branch in the controlled choices, many seeds can
look identical. Committed Gomad v3 explores by sampling seeds; it does not claim
to enumerate every possible schedule.

## Each seed gets a clean process

The Runner prepares once, but executes each seed in a fresh private working
directory and process. Globals, leaked goroutines, file descriptors, allocator
state, and runtime random state cannot carry from seed 7 into seed 8.

Inside a target, Gomad forces the initial `GOMAXPROCS` to one. Campaign
parallelism still works, but it happens across independent processes rather
than by giving one target multiple Ps. This makes the controlled schedule much
smaller and easier to repeat.

There are a few layers around that target process:

- the CLI starts an isolated coordinator for the campaign;
- the Runner selects work and handles results in deterministic selection order;
- a supervisor owns each target's process group and wall-clock deadline; and
- a small bootstrap installs the seed and inherited capabilities before user
  package initialization.

The target starts with an empty environment. Gomad adds its private activation
data, `TZ=UTC`, and only values explicitly supplied with `--env`. Ambient
credentials, proxy settings, and other shell state therefore do not accidentally
become replay inputs.

The supervisor is there because deterministic programs can still go wrong in
very ordinary ways. It drains stdout and stderr, enforces a wall-time watchdog,
and cleans up the complete process group with `SIGTERM` followed by `SIGKILL` if
necessary. Output is bounded for storage, but Gomad continues reading and hashes
the complete streams.

This is containment for trusted tests, not a security sandbox. Code that
deliberately bypasses the reviewed boundary with raw syscalls is outside the
contract.

## Time moves when the program cannot

On activation, the process clock starts at midnight UTC on January 1, 2000.
Standard wall and monotonic time, sleeps, timers, tickers, callbacks, context
deadlines, and the Go test timeout all observe that virtual clock.

Gomad keeps Go's native timer heaps. It does not replace every timer with a
parallel simulation data structure. Instead, the patched runtime waits until
its normal deadlock accounting shows that no goroutine is runnable. At that
quiescent point, it jumps the clock straight to the earliest timer deadline and
makes every timer at that instant eligible before scheduling resumes.

So this:

```go
time.Sleep(24 * time.Hour)
```

can finish almost immediately in wall time when nothing else is runnable. The
logical day still passed from the program's point of view.

Virtual time does not bulldoze runnable work. A busy loop or a goroutine
repeatedly polling a `select` remains runnable, so the clock cannot advance.
That is when the supervisor's real wall-time watchdog steps in. A watchdog
observation is kept separate from a deterministic target failure because host
elapsed time is a safety bound, not part of the virtual schedule.

Gomad also disables asynchronous preemption and the runtime system monitor while
active. Those otherwise introduce host-timing decisions that the seeded runtime
does not control. The trade-off is the same one just described: code that never
reaches a cooperative scheduling point can run until the watchdog stops it.

## External effects need their own boundary

Controlling goroutines and time is not enough if a test can still read a changing
file, wait on a real socket, ask the kernel for entropy, or inherit a hostname.

Every Runner-managed target therefore uses one versioned deterministic-I/O
profile. The pinned compiler inserts small prologues into reviewed `os` and
`net` functions. Their public names, signatures, interfaces, and call sites stay
the same, but Gomad can route calls to its in-memory implementations before they
reach the host.

The committed profile covers supported filesystem operations, loopback TCP,
hostname, and entropy. It also has a version-pinned adapter for supported
`modernc.org/libc` operations, which is what lets qualified pure-Go SQLite code
reach the same deterministic filesystem, time, and entropy boundary.

Every modeled operation is appended to a bounded transcript. During replay,
the target receives the recorded transcript through read-only shared memory.
The first different operation, argument, result, missing call, or extra call
becomes a replay divergence. There is no fallback to the live host.

Entropy has its own deterministic stream and is intentionally independent of
the scheduling seed. Changing schedule exploration should not silently change
the test's random input.

Sometimes a test genuinely needs fixture files from the host. A lazy read-only
mount makes that dependency explicit:

```sh
tools/gomadv3/.bin/gomad explore \
  --io-ro-mount ./fixtures=/fixtures \
  --seeds 0-9 \
  --build-tag test_dep \
  go-test ./path/to/package -- '-test.run=^TestWithFixtures$'
```

The Runner captures entries when the target first observes them. A retained
artifact stores those captured bytes, and replay serves them from the artifact
instead of reopening `./fixtures`. Writes fail as read-only; symlinks, special
files, unstable captures, and accesses outside the declared mounts fail rather
than weakening replay.

## World is for explicit event models

Deterministic I/O handles transparent, synchronous operations. Some tests need
something richer: external events that can become ready at different logical
times, compete with one another, be cancelled, or change modeled state.

That is what `World` is for.

World is an opt-in, in-memory event engine under `tools/gomadv3/world`. A target
connects through `world/process`, registers requests and readiness, and explicitly
calls `Quiesce` when application work cannot proceed. World then advances to the
earliest event time and delivers all events at that instant in canonical order.
Seed-derived ranks can choose between events that an adapter has declared
semantically equivalent.

World does not inspect goroutines, own application state, or perform host I/O.
It owns event identity, ordering, snapshots, transitions, terminal states, and
replay validation. Each adapter owns its own domain semantics. The initial
mailbox adapter is a small example of that split.

This distinction keeps the simple path simple: opening an in-memory file does
not need an event scheduler. An adapter reaches for World only when readiness,
competition, cancellation, or logical-time coordination is actually part of
the behavior being tested.

## What `--choices` adds

Seed replay is useful, but a seed alone only says how the runtime's random
streams started. For stronger evidence, run exploration with `--choices`:

```sh
tools/gomadv3/.bin/gomad explore \
  --choices \
  --choice-bytes 8MiB \
  --seeds 0-99 \
  --build-tag test_dep \
  go-test ./path/to/package -- '-test.run=^TestSomethingConcurrent$'
```

The runtime writes a bounded v2 choice trace containing stable logical
decisions and observations. The important decision records are runnable
goroutine selection and `select` polling; the trace also observes the final
`select` result. Alternatives use logical identities rather than physical queue
positions, pointers, or goroutine IDs.

When replay opens an artifact with a complete v2 trace, it projects the decision
records into a read-only choice tape. Before applying each recorded decision,
the runtime checks the decision kind, call site, available alternatives,
alternative identities, and selected value. A mismatch stops the target at the
first divergent ordinal. An unconsumed or exhausted tape is a divergence too.

The tape is bound to the exact target hash, toolchain build key, platform, and
choice-controller implementation. It is meant to reproduce this execution, not
to be portable across source changes.

Trace storage is explicitly bounded. If the trace overflows, Gomad reports a
Runner failure; it does not pretend that partial choice evidence supports exact
choice replay.

## Results become durable evidence

As each seed finishes, the Runner classifies the outcome. The major categories
stay deliberately separate:

- a target failure, such as a test assertion, exit, signal, runtime fatal, or
  logical test timeout;
- a watchdog observation, where host time had to bound the process;
- a replay divergence;
- invalid input or a modeled capacity failure; and
- a Runner or host failure in preparation, launch, containment, capture,
  integrity checking, or publication.

The Campaign journal records every completed selection ordinal. Results may finish
in parallel, but publication happens in selection order, so host timing cannot
change the journal or the guided corpus.

Failures are published as immutable, content-addressed artifacts. An artifact
can include:

- the exact prepared executable and its build identity;
- the seed, arguments, explicit environment, limits, and Runner identity;
- the outcome and a stable failure signature;
- bounded stdout and stderr plus hashes of the complete streams;
- the deterministic-I/O transcript;
- captured read-only mount data;
- World snapshots and transitions; and
- the choice trace and tape identity when choice recording was enabled.

Failure signatures exclude the seed, so two seeds that produce the same
semantic observation can be grouped as one distinct failure. Host paths and
diagnostic timestamps are kept out of stable record identities.

Publication uses a private staging area and writes the manifest last. A crash
can leave explicit partial state, but it cannot make an incomplete directory
look like a valid replay artifact.

Successful Executions are discarded by default. You can retain all successes, or only
ones that add semantic or choice coverage, but you must provide explicit count
and byte limits. That prevents an exploratory campaign from quietly turning
into unbounded artifact storage.

## Inspection answers “what did I actually get?”

`inspect` validates before it explains. Point it at either a Campaign or an
artifact:

```sh
tools/gomadv3/.bin/gomad inspect .gomad/artifacts/v1/campaign-...
tools/gomadv3/.bin/gomad inspect --choices .gomad/artifacts/v1/campaign-.../failures/sha256-...
```

For a Campaign, it shows the selected and attempted Executions, failures, retained
successes, artifact paths, truncation, and replay commands. For an artifact, it
shows the target and toolchain identity, outcome, transcripts, mounts, output
hashes, and replay command. `--choices` adds record counts, branching decisions,
choice kinds, trace and tape hashes, and exact-replay availability.

This is a useful habit: inspect the artifact instead of treating its directory
name as proof.

## Replay uses the stored execution, not today's source tree

Replay begins with a full preflight. Gomad validates the manifest and every
payload, verifies compatibility packs and adapters, checks the host and pinned
toolchain identity, confirms the World and I/O records, and reads the stored
binary's build information.

Only then does it copy the verified target into a fresh private directory and
run it:

```sh
tools/gomadv3/.bin/gomad replay .gomad/artifacts/v1/campaign-.../failures/sha256-...
```

Replay never rebuilds from the current checkout, swaps in a convenient local
binary, rereads a captured mount from the host, or silently upgrades an old
schema. It supplies the recorded I/O transcript and World plan, uses the same
seed, and forces the choice tape when the artifact contains replayable v2 choice
evidence. Finally, it compares the new outcome, outputs, transcripts, World
state, and choice evidence with the artifact.

If you only want to audit the artifact and its compatibility without executing
the target, use:

```sh
tools/gomadv3/.bin/gomad replay --verify-only ARTIFACT_DIR
```

A replayed watchdog remains diagnostic: Gomad can confirm the recorded evidence,
but it cannot make host elapsed time deterministic.

## Campaigns can stop, resume, and learn

An ordinary `explore` campaign can run seeds in parallel and stop after the
first failure, after a distinct-failure budget, or after all selected seeds.
Per-Execution and overall wall deadlines keep the work bounded.

The Campaign plan records the target, identities, limits, environment, mounts, and
exact seed selection before execution. If a campaign is interrupted, resume it
with:

```sh
tools/gomadv3/.bin/gomad resume .gomad/artifacts/v1/campaign-INTERRUPTED
```

Resume locks and validates the original Campaign, prepared binary, completed Execution
records, Runner, toolchain, I/O profile, and referenced artifacts. It schedules
only unfinished ordinals. It does not reinterpret the command against the
current checkout.

For larger searches, Gomad can collect stable semantic probes, choice features,
or both:

```sh
tools/gomadv3/.bin/gomad explore \
  --choices \
  --coverage=semantic+choice \
  --keep-successes=novel \
  --success-limit=32 \
  --success-bytes=1GiB \
  --count=1000 \
  --build-tag test_dep \
  go-test ./path/to/package -- '-test.run=^TestSomethingConcurrent$'
```

Semantic coverage is not source-code coverage. It records stable probes at the
reviewed I/O boundary. Choice coverage summarizes the runtime branch points
Gomad observed, while guidance separately considers abstract World outcomes and
transitions. Novel-success retention keeps examples that add semantic probes or
choice features.

Guided exploration stores replay-verified interesting cases in a bounded private
corpus, then mixes useful prior seeds with fresh requested seeds in later
campaigns:

```sh
tools/gomadv3/.bin/gomad explore \
  --guide \
  --corpus .gomad/corpus \
  --count=1000 \
  --build-tag test_dep \
  go-test ./path/to/package -- '-test.run=^TestSomethingConcurrent$'
```

Guidance still runs realized seeds and transcripts. It prioritizes known-useful
areas of the search; it does not turn seed sampling into exhaustive exploration.

## Qualification checks the checker

Finding a failure is only useful if the deterministic boundary itself is
trustworthy. Gomad has a separate qualification workflow for that.

`qualify` prepares and executes one target independently at least twice with the
same seed, then compares canonical evidence:

```sh
tools/gomadv3/.bin/gomad qualify \
  --seed 7 \
  --repeat 2 \
  --choices \
  --replay-successes \
  --success-limit=2 \
  --success-bytes=256MiB \
  --build-tag test_dep \
  go-test ./path/to/package -- '-test.run=^TestSomethingConcurrent$'
```

The comparison covers the exact target and arguments, toolchain and Runner,
complete output hashes, I/O transcript, captured mounts, World state, outcome,
semantic probes, and optional choice evidence. A repeat that merely exits zero
but produces different evidence does not qualify.

`qualify-set` applies the same idea to a versioned manifest of workloads. It
analyzes every workload first, runs only supported ones, and publishes explicit
supported and unsupported counts. `compare-support` compares two validated
reports so a dependency or boundary change cannot quietly reduce the supported
corpus.

These commands are mainly for maintaining Gomad and its supported platform
bundle, but they explain an important design choice: support is something Gomad
records as evidence, not something it assumes because a test happened to pass.

## What determinism means here

Gomad's promise is deliberately local:

> For the same pinned toolchain, architecture, target bytes, deterministic
> inputs, and seed, supported runtime-controlled choices are repeatable across
> fresh processes.

That promise does not mean every Go program is deterministic, every possible
schedule has been searched, or an artifact survives arbitrary source and
toolchain changes. It also does not model data races or weak-memory outcomes;
race detection remains a separate test profile.

The current supported target is an internally linked, pure-Go binary on the
qualified `darwin/arm64` platform. Cgo, external linking, multiple Ps, signals,
finalizers, subprocesses, non-loopback networking, DNS, plugins, and
unrecognized host I/O are outside the committed deterministic contract.

The main feature gaps are similarly straightforward: bounded choice-exploration
exploration currently has neutral benchmark efficiency, combined
schedule-plus-fault exploration and failure minimization are not yet complete,
and there is no qualified Linux Runner bundle. Multi-node distributed-system
simulation is available through explicit in-process and process-backed fidelity
tiers. Those limits are roadmap items, not hidden assumptions in current
results.

## A practical way to start

For a first useful target, choose a small assertion-based test with real
concurrency and little external integration. Then:

1. Run `make gomadv3` and `gomad doctor`.
2. Run `gomad analyze` and deal with capability blockers before exploring.
3. Start with a small seed range, `--choices`, and one focused `-test.run`.
4. Inspect any retained artifact before replaying it.
5. Increase the seed count or add semantic coverage only after the small run is
   understandable.

That workflow plays to Gomad's strength. It does not try to prove that concurrent
code is correct. It turns controlled executions into named, bounded, inspectable
evidence—and turns the rare bad one from “it failed once” into something another
developer can run again.

For exact command behavior and limits, see the [Gomad v3 README](README.md).
For the ownership boundaries and design rationale, see the
[architecture document](ARCHITECTURE.md). The brief status of
planned work lives in [GOMAD3_NEXT.md](../../GOMAD3_NEXT.md).
