# Learning from concurrency tools to improve Gomad

**Research date:** 2026-09-30

Gomad should find more distinct Go concurrency bugs per unit of compute and give
developers small, understandable reproductions that replay exactly. This assessment
compares tools, separates transferable mechanisms from Go-specific limitations,
and proposes experiments for search, guided fuzzing, debugging, and execution speed.

The [source assessment](../docs/research/gomad/2026-09-30-concurrency-tools.md)
contains additional primary-source findings. These are research candidates;
[milestones](GOMAD_MILESTONES.md), the [roadmap](GOMAD_NEXT.md), and Flow-Next specs
govern delivery. The earlier [simulation assessment](../docs/research/gomad/GOMAD_CMPv2.md)
remains dated background.

## Start from the implemented boundary

Gomad retains ordinary Go source and uses a patched toolchain with reviewed I/O
models. A supported execution starts with one P and asynchronous preemption
disabled. The Runner prepares a target once and launches fresh processes for
parallel exploration. Native virtual time skips to the earliest timer when no
goroutine is runnable; process-backed simulation also arbitrates participant and
modeled-event readiness. Exact replay binds the retained binary, toolchain,
platform, controller, inputs, and model identities.
[Execution contract](../tools/gomad3/README.md#contract),
[architecture](../tools/gomad3/ARCHITECTURE.md)

Existing mechanisms include bounded choice-prefix and combined-simulation
exploration, semantic/choice coverage, a replay-verified guided corpus, campaign
recovery, sharding/merge, and a combined-simulation minimizer. Guidance currently
reuses observed seeds and transcripts. General scenario mutation, general input
shrinking, and live Go process snapshots are extensions.
[Current workflows](../tools/gomad3/CLI.md)

Known workload/platform replay gaps remain in the
[milestones](GOMAD_MILESTONES.md#open-findings). Establish a qualified benchmark
set before judging a new policy. Host-load divergence, unsupported operations,
trace overflow, and watchdog expiration need separate outcomes and cannot become
confirmed target bugs.

## What other tools contribute

| Tool or work | Mechanism | Lesson for Gomad | Transfer limit |
| --- | --- | --- | --- |
| [Shuttle](https://docs.rs/shuttle/latest/shuttle/) | Rust replacement primitives; randomized, PCT, DFS, replay, parallel portfolios | Compare policies and annotate failures | Go operations need runtime/compiler visibility; Rust integration cannot transfer directly |
| [Loom](https://docs.rs/loom/0.7.2/loom/) | Instrumented Rust primitives and atomic behavior; reduced small-model exploration | Expose meaningful transitions, bound search, validate reduction | Uninstrumented behavior is invisible; its relaxed-memory model also has limits |
| [CHESS](https://www.microsoft.com/en-us/download/details.aspx?id=52619) and [PCT](https://www.microsoft.com/en-us/research/publication/a-randomized-scheduler-with-probabilistic-guarantees-of-finding-bugs/) | Controlled schedules, bounded preemptions or randomized priorities | Explore simple disruptive orderings early | Guarantees depend on scheduling steps and algorithm assumptions |
| [Coyote](https://microsoft.github.io/coyote/overview/how/) | Controlled .NET concurrency with exploration and trace replay | Join schedule control and failure diagnosis | Its instrumentation targets .NET; Go needs its own contract |
| [RFF](https://www.comp.nus.edu.sg/~gregory/papers/asplos24.pdf) and [FEST](https://www.usenix.org/system/files/conference/nsdi26/nsdi26spring_li_prepub.pdf) | Guided schedule mutation using reads-from relations or semantic timelines | Reward new interactions as well as code paths | RFF instruments memory events; FEST evaluates P models, not native Go |
| [Gosim](https://github.com/jellevandenhooff/gosim) | Go translation and simulated services; debugging at recorded steps | Link replay ordinals to Go source and debugger stops | Its runtime/translation strategy differs from Gomad's |
| [FoundationDB](https://apple.github.io/foundationdb/testing.html) and [TigerBeetle](https://tigerbeetle.com/blog/2023-07-06-simulation-testing-for-liveness/) | Application-integrated simulation and fault/recovery testing | Generate difficult state and check progress after recovery | Protocol and availability assumptions must come from the Go consumer |
| [Antithesis assertions](https://antithesis.com/docs/product/writing_tests/assertions/) | Declared safety and reachability properties | Check whether interesting situations occurred | Public property semantics do not establish proprietary search internals |

The transfers below are engineering proposals. These sources establish mechanisms,
not their effectiveness or implementation cost for Gomad.

## Go determines which executions are reachable

### Scheduling visibility comes before search sophistication

A policy can select only transitions the runtime offers. With asynchronous
preemption disabled, a segment without a scheduling opportunity can finish while
another goroutine is runnable. Separate atomic loads/stores implementing a
read-modify-write can lose an update without a data race. Enumerating all current
choices still misses that witness if the required switch lies between those
operations. [Runtime contract](../tools/gomad3/README.md#contract)

Build witnesses for atomic check-then-act, uncontended synchronization, channel
close/send, cancellation/completion, lock handoff, and deadline/delivery ties.
Establish whether each failing ordering is legal in Go and reachable under the
current profile. If visibility is missing, prototype narrowly scoped runtime or
compiler scheduling points in a separate qualified profile. Avoid inserting
arbitrary yields into unchanged consumer tests.

Hooks must respect runtime critical sections, GC/write barriers, and operation
semantics. Record stable sites, hook coverage, disabled-mode behavior, overhead,
and replay identity. Full shared-memory instrumentation is a different scope from
adding atomic or synchronization boundaries.

### Rust memory-model machinery has a different purpose

Go promises sequential consistency for data-race-free programs, and atomics behave
as though executed in a sequentially consistent order. Controlled SC interleavings
are therefore the initial target. Loom's Rust relaxed-memory machinery is not a
drop-in requirement. Race-free programs can still have deadlocks, incorrect
protocols, and non-atomic compound operations.
[Go memory model](https://go.dev/ref/mem)

Single-P exploration covers offered scheduling points, not every multi-P or racy
execution. Keep stock Go `-race` and native integration/stress testing as
complementary lanes; Gomad's current profile excludes the race detector. A clean
race run means its executed paths exposed no detected race.
[Race detector](https://go.dev/doc/articles/race_detector),
[supported profile](../tools/gomad3/README.md#contract)

### Reduction needs dependencies

Try preemption bounding after the controller distinguishes continuing, blocking,
yielding, and exiting. Charge a switch away from an actor that could continue;
a necessary blocking switch does not spend that budget. Compare bounds 0, 1,
and 2 with separate step, depth, and execution limits.
[Loom controls](https://docs.rs/loom/0.7.2/loom/model/struct.Builder.html)

PCT needs defined actor identities, enabled sets, priority changes, and step
sampling. Its bug depth concerns ordering constraints, not a preemption count.
An approximate priority scheduler does not inherit the paper's probabilistic
guarantee. [PCT paper](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/asplos277-pct.pdf)

Dynamic partial-order reduction (DPOR) additionally needs resource identities,
dependencies, and causality. Start with an explicit mailbox/network or storage
model, treating unknown interactions as dependent. Compare reduced and unreduced
finite searches on outcomes and deadlocks. Native Go reduction needs dependencies
for the actual operations it prunes; a World hash omits application heap and
stacks and cannot justify native-state equivalence.
[DPOR paper](https://patricegodefroid.github.io/public_psfiles/popl2005.pdf),
[Gomad model boundary](../tools/gomad3/ARCHITECTURE.md#cluster-simulation)

## Couple a guided fuzzer to deterministic execution

Search spans input data, operation mix, concurrency, scheduling, timing, and faults.
Schedule seeds cannot discover a failure requiring an input the workload never
generates. RFF and FEST support trying concurrency-specific feedback, but neither
establishes native-Go performance. [RFF](https://www.comp.nus.edu.sg/~gregory/papers/asplos24.pdf),
[FEST](https://www.usenix.org/system/files/conference/nsdi26/nsdi26spring_li_prepub.pdf)

Start with a typed workload generator and the existing Runner/Guide. A candidate
should carry:

- canonical scenario data, operation arguments, actor count, and resource limits;
- independent workload, scheduling, and fault seeds, plus realized fault plans;
- an optional validated choice prefix or versioned scheduling policy;
- generator, mutator, oracle, feature-schema, target, and execution-profile identities;
- logical-time, wall-time, trace, input, history, and oracle-work budgets.

The generator owns operation validity and dependencies. Mutate key skew, capacity,
batches, retries, cancellation, payload sizes, and overlap. Distinguish invalid
generator candidates from deliberately tested invalid API requests. Record
realized operations and values so replay does not rely on today's generator.

### Execution and feedback loop

1. Select parents from an immutable bounded corpus snapshot. Reserve a declared
   share for unguided candidates and low-fault workloads.
2. Generate a canonical candidate. Initially mutate one dimension at a time,
   then combine mutations after measuring their separate benefits.
3. Execute in a fresh process. Check independent safety properties, required
   witnesses, and recovery progress obligations.
4. Return bounded versioned features. Prefer confirmed failure signatures and
   novel semantic interactions; optional code coverage adds another signal.
5. Replay every authoritative corpus admission using its complete newly recorded
   execution. Quarantine divergence for Gomad diagnosis.
6. Shrink confirmed failures and publish corpus changes in candidate-ordinal order.
   Preserve immutable parent artifacts.

Useful feedback includes cancellation versus completion, timeout versus delivery,
retry after lost acknowledgement, queue occupancy classes, lifecycle transitions,
and combinations of probes. Normalize or bound arbitrary payloads, timestamps,
identities, and counters so they do not manufacture novelty. Executions reaching
the same source lines can have different concurrent behavior.

For schedule mutation, replay a validated parent prefix, change an offered
alternative, and record a fresh suffix. The old suffix can become inapplicable.
A changed scenario may invalidate even the prefix; begin with a fresh schedule
unless compatibility is established. Candidate infeasibility is search feedback;
divergence replaying an unchanged artifact is a confidence failure. Never repair
exact replay silently. [Prefix validation](../tools/gomad3/choice/tape.go)

Later, test abstract constraints such as cancellation before an eligible
completion or an actor waiting until another reaches a known site. Implement only
observable, enforceable constraints. Runnable/select traces cannot support RFF's
general reads-from constraints without additional memory-event instrumentation.

### A stock Go fuzzer needs a feedback bridge

Go's built-in fuzzer supplies input mutation, coverage guidance, and minimization;
its calls should be deterministic and independent of persistent state. Reusing
an activated runtime across inputs needs a demonstrated reset of globals,
goroutines, timers, models, scheduler, and coverage. Fresh executions preserve
Gomad's existing isolation. [Go fuzzing](https://go.dev/doc/security/fuzz/)

A `testing.F` wrapper launching Gomad children does not automatically convert
child semantic features into fuzz-engine feedback. Start by importing generated
inputs or adding typed mutation to Guide. An external engine needs a versioned
evaluate/result protocol exposing child features, validity, budgets, signatures,
and artifact identities. Go supports integration coverage from instrumented
binaries; online guidance and Gomad's I/O boundary require explicit integration.
[Integration coverage](https://go.dev/doc/build-cover)

Qualify coverage instrumentation as a separate execution profile and replay the
same binary. Rebuilding without instrumentation creates a separate experiment.

## Make failures understandable

Shuttle annotates operations and vector clocks for its Explorer; Gosim demonstrates
Delve stops at recorded steps. Borrow the workflow while retaining Gomad's artifact
contract. [Shuttle annotations](https://docs.rs/shuttle/latest/shuttle/annotations/index.html),
[Gosim debugging](https://github.com/jellevandenhooff/gosim)

First extend inspection with a bounded timeline connecting logical actors, choice
ordinals, available source sites, model operations, faults, timers, and oracle
checks. Show the last valid choice and first divergence with expected/observed
alternatives. Add causal edges only when instrumentation establishes them; log
order alone does not establish happens-before.

Later, replay to a choice boundary and inspect stacks/application state in a
debugger. Host pauses must not advance logical time or change controlled choices.
Different debug build flags change target identity and need fresh qualification.

Shrink operations, payloads, actor count, faults, and schedule constraints within
attempt/time limits. Accept only the same normalized property failure with exact
replay of the new artifact. General typed shrinking and durable minimizer resume
extend the current combined-simulation reducer.
[Current minimization](../tools/gomad3/CLI.md)

## Spend compute on new executions

### Parallelize cases and deterministic search rounds

Reuse prepared binaries with bounded local worker pools and disjoint portable
shards. For adaptive guidance, propose deterministic epochs: each reads one corpus
snapshot, assigns candidate ordinals and policy budgets, merges validated results
in ordinal order, then publishes the next snapshot. Completion timing must not
change later candidate selection.
[Current publication and parallelism](../tools/gomad3/ARCHITECTURE.md#system-boundary),
[distribution](../tools/gomad3/CLI.md)

Portable shard/merge currently covers unguided seed campaigns. Distributing
adaptive candidates or dynamically discovered prefixes needs a round coordinator;
existing local parallel exploration does not establish that distributed capability.
[Current shard limits](../tools/gomad3/README.md)

Compare portfolios of seed sampling, bounded prefixes, candidate PCT/preemption
bounds, and guided scenario mutation. Version adaptive allocation rules. Reserve
compute for replay/reduction and bound outstanding trace/artifact bytes. Measure
scaling and replay reliability under load; ten times the workers can expose
determinism leaks or saturate memory/storage.

### Skip idle time and preserve deadline competition

Keep native timers in the runtime and external events in their models. Advance
only at established quiescence; eligible equal-time events must participate in
ordering before further advancement. Generate deadline-adjacent situations through
logical inputs or modeled delays. Bound event/decision counts as well as elapsed
time so repeated immediate events cannot escape the budget. Busy runnable work
consumes real compute and prevents idle-time skipping.
[Timer/arbitration contracts](../tools/gomad3/ARCHITECTURE.md)

`clock_tick=forward` advances observed `time.Now` during busy work independently
of timer/simulation time. It does not model arbitrary CPU execution duration.
Testing deadline expiration during CPU-bound work needs a separately reviewed
time/progress profile. [Clock policies](../tools/gomad3/README.md#contract)

### Checkpoints have four meanings

| Checkpoint | Saved state | Benefit and feasibility |
| --- | --- | --- |
| Campaign/search progress | Completed ordinals, remaining prefixes, corpus epoch, budgets, identities | Existing recovery/frontiers avoid repeating finished work; extend to fuzzer epochs and minimizer progress |
| Explicit model snapshot | Canonical World/network/volume state | Existing snapshots can support model-owned search when the extension captures all future-relevant state |
| Replay prefix | Inputs/decisions reaching a boundary in a fresh process | Preserves isolation and reproducible state; still pays initialization and prefix execution |
| Live execution snapshot | Go heap/stacks, runtime/GC/scheduler, descriptors, coordinator/models, clock and tape cursors | Could amortize prefixes, but needs a new backend and platform-specific qualification |

Loom checkpoints serialize exploration paths and rerun the test closure; they
do not snapshot a native process. Restoring World alone does not restore live Go
goroutines. [Loom checkpoint source](https://docs.rs/loom/0.7.2/src/loom/model.rs.html)

Ordinary fork-and-continue is unsuitable as a default. Go's fork/exec child must
avoid allocation, rescheduling, and lock acquisition before exec because inherited
locks may be held. Evaluate OS/VM snapshots only after profiling proves prefix
execution dominates, including every participant and external state. Measure
restore cost, memory, identity/relocation constraints, isolation, and suffix
equivalence against fresh replay.
[Go fork/exec implementation](https://go.dev/src/syscall/exec_linux.go)

## Oracles decide whether an edge case is a bug

Pair workloads with independent invariants and small sequential models. Candidate
properties include legal workflow/update transitions and preservation of
acknowledged durable effects. Linearizability checks need specifications handling
overlap and uncertain timeout outcomes; equality to one expected history is
insufficient.

Require witnesses that a planned overlap/fault occurred. After declaring recovery
preconditions satisfied, require specific work to complete within a logical-time
or progress bound. Record fairness assumptions and distinguish unmet preconditions,
bounded progress failure, runtime/model deadlock, and wall-watchdog termination.
Finite recovery tests do not prove progress under every schedule.
[TigerBeetle liveness](https://tigerbeetle.com/blog/2023-07-06-simulation-testing-for-liveness/),
[Antithesis properties](https://antithesis.com/docs/product/writing_tests/assertions/)

Validate oracles against known broken and corrected implementations. Compare
modeled contracts with stock Go/native behavior where feasible. Exact replay proves
an observation reproduces; independent oracle/model validation establishes its
meaning.

## Experiments and adoption gates

These stages describe evidence, not a new task queue. Keep policy, workload
generation/shrinking, feature extraction, and oracles in independently testable
deep modules. Runner owns execution, bounds, and publication; domain models own
domain semantics.

| Stage | Experiment | Evidence required to adopt |
| --- | --- | --- |
| Confidence | Qualified known bugs, corrected counterparts, witnesses, loaded replay runs | Separate target, Gomad, model, and oracle failures; record platform limits |
| Visibility | Audit witnesses; instrument only missing scheduling boundaries | Reach a previously inaccessible legal failure, replay exactly, quantify overhead |
| Input search | Typed scenarios and semantic feedback with existing Guide/Runner | Additional confirmed bugs or lower discovery cost against equal-budget seeds/fixed workloads |
| Schedule search | Portfolio, preemption bounds, PCT against the raw frontier | Measured benefit across several bug classes; explain neutral/worse results |
| Diagnosis | Timeline, typed shrinking, resumable minimization, then debugger stops | Smaller same-property failures, exact replay, recoverable interrupted reduction |
| Repeated work | Epoch checkpoints and model snapshots; live snapshots after a prefix-cost study | Accounted savings without isolation/replay regressions |
| Redundant schedules | Narrow DPOR after dependency instrumentation | Reduced/unreduced finite searches agree on outcomes and deadlocks within declared bounds |

Use atomic compound-operation, cancellation/deadline, shutdown/deadlock,
queue-saturation, and fault/recovery bug families. Retain the real Temporal
update-admission ordering regression alongside small witnesses, using its
[task evidence](../.flow/tasks/fn-100-gomad-f6-a-package-level-functional.2.md).
Hold bugs out when
tuning policies. Under equal total compute and wall budgets, report confirmed
signatures, discovery-time distributions, misses, executions, CPU cost, virtual
time advanced, replay success/divergence, reproduction size, and retained bytes.
Include preparation, replay, reduction, and checkpoint costs; retain budget-censored
misses in the results.

Adopt mechanisms that find more distinct replayable target bugs per compute-hour
and shorten diagnosis under explicit support/search limits. Seed counts, coverage,
and bounded search completion support that measurement; they do not establish
that every relevant Go execution was explored.
