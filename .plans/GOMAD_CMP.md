# Learning from concurrency tools to improve Gomad

**Research date:** 2026-09-30 · **Extended:** 2026-10-01 with a code feasibility
study and a wider source survey

Gomad should find more distinct Go concurrency bugs per unit of compute and give
developers small, understandable reproductions that replay exactly. This assessment
compares tools, separates transferable mechanisms from Go-specific limitations,
and proposes experiments for search, guided fuzzing, debugging, and execution speed.

The [source assessment](../docs/research/gomad/2026-09-30-concurrency-tools.md)
contains additional primary-source findings. These are research candidates;
[milestones](../MILESTONES.md), the [roadmap](GOMAD_NEXT.md), and Flow-Next specs
govern delivery. The earlier [simulation assessment](../docs/research/gomad/GOMAD_CMPv2.md)
remains dated background.

The 2026-10-01 extension draws on four source notes, which hold the `path:line`
references and URLs behind the statements added here:
[schedule-search feasibility](../docs/research/gomad/2026-10-01-feasibility-schedule-search.md),
[workload and diagnosis feasibility](../docs/research/gomad/2026-10-01-feasibility-workload-diagnosis.md),
[industry practice](../docs/research/gomad/2026-10-01-industry-dst-practice.md), and the
[academic survey](../docs/research/gomad/2026-10-01-academic-concurrency-testing.md).
The [vision note](../docs/research/gomad/2026-10-01-gomad-vision.md) orders the work.
No experiment ran for the extension. Statements marked *inferred* come from reading
code and have no execution behind them.

## Direction

Gomad finds Temporal concurrency bugs, explains them, and proves it still can.

- **Finds:** a standing hunt over real suites reports distinct, replayable target
  failures per compute-hour, separated from test assumptions, model defects, and
  Gomad divergence.
- **Explains:** a failure report names source sites and the smallest ordering
  difference between a passing and a failing run.
- **Proves:** a known-bug table records seeds-to-find for each bug and gates Gomad
  changes.

Four findings from the code set the order of work:

1. Nothing measures search. No known-bug set, discovery-cost metric, or hunting job
   exists, so every adoption gate below compares against a missing baseline.
2. The decision space has the wrong shape. It lacks wake-up and atomic
   interleavings and carries thousands of decisions that cannot change behavior.
3. A target cannot receive a per-execution input or report an application-level fact.
4. Failures are not legible. Inspection prints hashes and the minimizer rejects every
   Temporal functional-test failure.

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
[milestones](../MILESTONES.md#open-findings). Establish a qualified benchmark
set before judging a new policy. Host-load divergence, unsupported operations,
trace overflow, and watchdog expiration need separate outcomes and cannot become
confirmed target bugs.

### What the implemented boundary does not yet do

- **No automation hunts bugs.** Every Make target and CI job runs `qualify` or
  `qualify-set`. `explore`, `--guide`, and `minimize` appear in no Makefile or
  workflow. One server bug (update-admission ordering) was found during
  qualification.
- **An execution varies by seed, choice tape, and simulation plan only.** Argv,
  environment, and mounts are fixed per campaign. Go code cannot read its seed.
- **Guidance re-runs known executions.** Corpus seeds take up to 75% of a guided
  campaign against an identity-bound target, so they reproduce retained records.
  The Guide is a retention and regression store until a mutation operator exists.
- **The corpus holds about six `./tests` cases.** Each artifact embeds the 155 to
  179 MB target binary and the corpus cap is 1 GiB.
- **Semantic probes are standard-library only.** 131 compiler-inserted probes
  exist. Application code cannot declare or emit one. World transitions are the
  only application-reachable feature channel, and no in-repo target uses it.
- **The simulation track has no Temporal subject.** Its Temporal scenario is raw
  TCP plus one `collection.SyncMap`. Hosting `testcore` on simulated nodes needs
  a shared-store or persistence-node model and a membership implementation under
  the `gomad` tag (*inferred* from code; no document states the reason).

Retained per-execution wall times on darwin/arm64: about 0.25 s for a trivial
program, 1.7 s for the boot-only cluster probe, and 1.5 to 4.4 s for functional
suites, which make 8.6k to 58k branching decisions each. A core-day buys roughly
10^4 functional executions. linux/amd64 has no retained per-execution timings.

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

Sources added on 2026-10-01:

| Tool or work | Mechanism | Lesson for Gomad | Transfer limit |
| --- | --- | --- | --- |
| [Delay bounding](https://microsoft.com/en-us/research/wp-content/uploads/2016/02/popl198ap-emmi.pdf) and the [Thomson study](https://www.doc.ic.ac.uk/~afd/papers/2016/TOPC.pdf) | A deterministic scheduler deviates from its default at most K times; on 49 benchmarks naive random found 43 bugs, iterative preemption bounding 38, iterative delay bounding 45 | The one bounded sampler that needs no preemption; random is a strong baseline | Needs a deterministic default choice at every decision |
| [POS](https://www.cs.columbia.edu/~junfeng/papers/pos-cav18.pdf), [Morpheus](https://www.cs.columbia.edu/~junfeng/papers/morpheus-asplos20.pdf), [Fray](https://arxiv.org/pdf/2501.12618) | Random priorities reset on same-object events; operations whose site never conflicted run immediately | Fray saw about 25% more failing tests than random walk on 2,664 real JVM tests; control mattered more than the algorithm | Needs the resource behind each alternative |
| [DEMi](https://cs.nyu.edu/~apanda/assets/papers/nsdi16.pdf) | Minimize external events first, keep the schedule near the original, match events by fingerprint | Reproductions within a median 1.6x of hand-minimized | Needs decision fingerprints that survive perturbation |
| [Hermit](https://developers.facebook.com/blog/post/2022/11/22/hermit-deterministic-linux-testing/) | Bisect between a passing and a failing schedule to one swapped event pair | Report two stacks instead of a tape | Needs a nearby passing tape and a re-alignment rule |
| [Antithesis causality analysis](https://antithesis.com/blog/2026/causality_analysis/) | Re-randomize continuations from cut points of a failing run and plot failure probability | Locate the decisions that fix the outcome | Gomad pays one prefix replay per sample |
| [rr chaos mode](https://robert.ocallahan.org/2016/02/introducing-rr-chaos-mode.html) | Victim threads do not run for random intervals | Uniform choice among runnable goroutines never starves one | Needs a new choice kind |
| [etcd robustness tests](https://github.com/etcd-io/etcd/blob/main/tests/robustness/README.md) | A table of historical bugs with repro targets | Detection cost on known bugs judges every search change | Each bug needs a build that reintroduces it |
| [BUGGIFY](https://transactional.blog/simulation/buggify) and [swarm testing](https://tigerbeetle.com/blog/2025-04-23-swarm-testing-data-structures/) | Per-run site enablement, knob randomization, fault wind-down; random feature subsets per run | Faults must sometimes be absent and must eventually stop | Unmodified Temporal code offers dynamic config and existing fault interceptors as seams |
| [IJON](https://www.gwern.net/doc/reinforcement-learning/exploration/2020-aschermann.pdf), [Mallory](https://arxiv.org/pdf/2305.02601), [GFuzz](https://par.nsf.gov/servlets/purl/10321048) | Annotated state, happens-before summaries, and channel-operation pairs as feedback | Feedback coarser than traces and finer than lines | Needs an application probe API or channel identities |
| [Go goroutine leak profile](https://go.dev/blog/goroutine-leak-profiles) | GC-based detection of goroutines that can never unblock | A per-execution oracle for Go's dominant blocking-bug class | Behavior under the patched runtime is unchecked |
| [Pebble metamorphic tests](https://www.cockroachlabs.com/blog/metamorphic-testing-the-database/), [Vortex](https://tigerbeetle.com/blog/2025-02-13-a-descent-into-the-vortex/) | One operation sequence across configurations and versions; a second harness on real I/O | Oracles without a reference model; a check on the simulator's honesty | Needs a workload tape separate from the schedule tape |
| [ADVOCATE](https://github.com/ErikKassubek/ADVOCATE) | Patched-runtime record/replay and fuzzing for Go 1.27; predicts bugs from one trace and rewrites it | The closest existing Go tool; worth a code-level read | README-level evidence only |

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

#### What the runtime offers today

Every goroutine switch is voluntary. A Runnable decision fires in `runqget` after
the current goroutine has left the P, so the running goroutine is never an
alternative. Two decision kinds are recorded and tape-forcible: the runnable pick
and each select poll-order step. Equal-deadline timer ties draw from a separate
unrecorded stream, so a seed changes them and a tape cannot.

| Operation | Scheduling opportunity |
| --- | --- |
| Blocking channel operation, blocking `select`, mutex slow path, `Wait`, sleep, modeled I/O wait, goroutine exit | Yes |
| `runtime.Gosched` | Yes; the yielder goes to the global queue and re-enters once the local queue is empty |
| Allocation | Sometimes, at GC start or an assist without credit; heap-dependent and not tape-targetable |
| `sync/atomic` operations, uncontended `Mutex`/`RWMutex`/`Once`/`WaitGroup` fast paths | Never |
| Non-blocking channel operation, `select` with a ready case, `close` | Never; the caller may ready another goroutine and keeps running |

Two holes follow:

- **Wake-ups.** After `Unlock`, `close`, `cancel()`, `wg.Done()`, or a non-blocking
  send, the waker runs on to its next blocking point. "The woken goroutine runs
  before the waker's next statement" is legal Go that Gomad cannot produce
  (*inferred*). The cancellation/completion, lock-handoff, and close/send
  witnesses depend on this boundary more than on atomics.
- **Atomics.** The `preemption` conformance fixture already pins the atomic case
  as a hang.

The offered space is also noisy:

- A hook inside the poll-order shuffle emits n-1 decisions for every n-case
  `select`, ready or not. In `TestSignalWorkflowTestSuiteChasm` seed 11, 26,865
  of 57,801 decisions are select-poll. Poll order matters only when at least two
  cases are ready.
- System goroutines are explorable alternatives. A trivial probe with two user
  goroutines records 26 branching decisions.
- Goroutines parked in the global queue are absent from the alternative set until
  the local queue drains.

#### Candidate scheduling points

| Candidate | Mechanism | Reach | Size and constraint |
| --- | --- | --- | --- |
| Wake-up yield | A recorded "waker continues / waker yields" decision in `ready()` sets the existing cooperative-preempt flag; the yield lands at the next function prologue, where `newstack` already refuses unsafe points | Wake-up witnesses; makes delay and preemption bounds meaningful | S to M prototype (*inferred*, needs a scratch toolchain). Adds `proc.go` lines while fn-110 shrinks the patch |
| `sync` methods and typed atomics | Existing callee-prologue interception; targets become non-inlinable | Lock and typed-atomic boundaries | M. Shifts timing and heap behavior of every target |
| Raw `atomic.AddInt64`-style calls | A new caller-side IR pass in the pre-inlining slot; no SSA change | Atomic check-then-act | M to L. Temporal has 136 raw calls and 113 typed declarations, so a callee-only hook misses half |

Channel and semaphore code is in the prohibited patch class. Each candidate is a
separate identity-bound profile. Yield points raise decision counts against the
64 MiB trace cap that eight suites already overflow (D15).

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

Preemption bounding is blocked on new yield points. With every switch voluntary,
bound 0 is the whole offered space and the controller has no continuing actor to
charge. After a wake-up yield exists, charge a switch away from an actor that
could continue; a necessary blocking switch does not spend that budget. Compare
bounds 0, 1, and 2 with separate step, depth, and execution limits.
[Loom controls](https://docs.rs/loom/0.7.2/loom/model/struct.Builder.html)

Delay bounding fits the current runtime. A delay is "pick something other than
the default at this dispatch", which is a Gomad decision and needs no preemption.
It requires a deterministic default order in place of the seeded draw. At 58k
decisions, K = 2 is about 3.4e9 schedules, so it is a sampling space and a sparse
tape encoding, never an enumeration.

A choice record holds the alternative count, a digest of the sorted alternative
set, and the selected identity. It omits the alternatives, the running goroutine,
why the previous goroutine left, and any resource. The runtime has all of these
in hand at the decision. A different in-process policy is an overlay-only change
in `gomadChoiceRunqIndex`, selected and identity-bound the way `GOMAD3_CLOCK_TICK`
is. One choice-wire revision should carry the missing fields plus a
perturbation-stable decision fingerprint, because each wire or overlay change is
a new toolchain identity and a requalification. Goroutine identities are stable
lineage hashes, except `time.AfterFunc` goroutines, which take a creation-order
counter and will diverge under mutated schedules (*inferred*).

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

General native DPOR stays out of reach: runtime records carry no dependency data
and executions run to tens of thousands of decisions. Two narrow reductions are
available from facts the runtime can observe:

1. Never expand a select-poll decision when fewer than two cases were ready.
   Record the ready-case count at the select result. This could remove up to
   about 46% of a functional suite's frontier.
2. In the combined frontier, treat model decisions on different simulation causal
   lanes as independent. World `EquivalenceClass` already declares independence.

Validate each by comparing reduced and unreduced searches on a finite fixture.
With resource identities in the record, partial-order sampling and Morpheus-style
conflict analysis become the cheaper way to stop spending randomness on decisions
that never conflict.

### The existing explorer cannot yet serve as a baseline

Breadth-first choice exploration skips every decision past `--max-choice-depth`,
counted from ordinal 0. The boot-only cluster probe alone records 4,562 choice
records, so on a functional suite the explorer permutes bootstrap and never
reaches test logic. An exploration start offset at the first test body removes
that limit.

The "no advantage over seed sampling" result in the roadmap is a conformance pin.
Its test asserts that both strategies saw 2 outcomes in 16 executions on a
one-goroutine fixture and never measures executions to discovery. Neither
explorer summary reports the execution ordinal of a first new outcome.

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

The Runner has no slot for candidate data today, and the Guide emits seeds only.
Two primitives unblock workload generation, swarm profiles, `sometimes`
assertions, and typed shrinking together:

- **A per-execution candidate.** Deliver an input blob the way the simulation
  exploration plan already reaches a target, and bind it into Campaign, Artifact,
  plan, corpus, and replay identity. The corpus identity already omits
  environment and clock-tick policy; close or use that gap deliberately.
- **An application assertion and probe API.** `always`, `sometimes`, and
  `reachable` with a declared catalog, so "never reached in N seeds" is a
  finding, plus IJON-style `set` and `max` primitives as corpus features with a
  cap on distinct values. Temporal's `testhooks` seam (tag `test_dep`) is the
  natural host.

Scenario bodies are Go closures. Canonical scenario data needs a data scenario
and a registered dispatcher. Temporal already has seams that need no simulation
harness: dynamic config for per-seed knob randomization, `faultinjection` for
persistence faults, and `event_generator.go` and `FuzzMatcherData` as operation
generators.

Draw a swarm profile per candidate: a random subset of fault kinds, workload
verbs, and knobs, then weights for that subset. Keep runs with faults off, and
wind faults down so recovery can be checked. Turso missed bugs that needed
databases over 1 GB because faults fired in every run.

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

The single-point mutation primitive exists. `BuildRankPrefix` flips any ordinal
of a validated parent tape and the execution records a fresh suffix. Three parts
are missing:

- a controller that samples flip points, since only the breadth-first frontier
  calls the primitive;
- a suffix reseed, since the suffix today is always the base seed's continuation
  and one prefix yields one execution;
- an infeasible-candidate outcome, since a diverging prefix aborts the choice
  campaign as a host error.

The override is a rank in sorted-identity order because the tape does not hold
the other identities.

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

`-test.fuzz` fails closed under Gomad because the fuzz coordinator starts workers
through the denied `os.StartProcess` boundary (*inferred*). Seed-corpus mode
(`-test.run=FuzzX`) should run in one process with the corpus supplied through
`--io-ro-mount` (*inferred*). No flag requests `-cover`, and counters written to
the in-memory filesystem have no export path. The offline pilot is
`FuzzMatcherData` in `./service/matching`, a package that already qualifies:
grow its corpus with the stock fuzzer, then replay it under schedule seeds.

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

What inspection and reduction offer today:

- Select records carry the caller's text offset and every artifact retains the
  unstripped binary, yet `inspect --choices` prints a hash of the offset.
  Printing `function file:line` needs no toolchain change.
- Runnable records carry no site, and goroutine identity is a one-way lineage
  hash. A per-goroutine timeline needs the richer record.
- Choice ordinals, I/O transcript ordinals, World transitions, virtual time, and
  output lines each have their own ordinal space.
- The minimizer accepts only artifacts with a simulation profile and shrinks
  only forced overrides, so it rejects every Temporal functional-test failure.
  Its state is sealed JSON that is never persisted.
- Delve's ASLR-off launch matches the darwin re-exec short-circuit (*inferred*).
  A debug stop still needs a Runner mode that keeps the target's inherited
  descriptors and disables the wall watchdog.

Explanation techniques that exact replay makes affordable:

- **Causality analysis.** Replay a failing tape to cut point t, run N reseeded
  continuations, and plot the failure fraction against t. Jumps mark the
  decisions that fix the outcome. Bisect on t, since each sample costs a prefix
  replay. Needs the suffix reseed.
- **Bisection to one transposition.** Binary-search between a passing and a
  failing tape to one swapped pair and print both stacks.
- **A reducer for seed failures.** Keep the first k decisions forced and let the
  suffix run free, searching on k. Then DEMi-style: remove external events
  first and match decisions by fingerprint, since deleting one event shifts
  every later positional record.
- **Retroactive instrumentation.** Search with minimal logging, then replay the
  failing artifact with verbose logs or tracing. The extra logging must consume
  no scheduler choice and no virtual time.

Developers act on reports more often than on interactive replay; MongoDB
resolves about 95% of findings from the report alone. Rank the report bundle
above debugger stops.

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

Both exploration engines already expose pure `NextRound`, `CommitRound`, and
`ReplaySegment` transitions, so a round coordinator is three file formats and
CLI verbs. It has no consumer until a search policy beats seed sampling on a
real suite.

A standing hunt needs less: a scheduler that splits a core budget fairly across
(test, swarm profile) pairs, keys failures by commit, seed, and signature, and
files an issue with the replay command. TigerBeetle's fleet starved its
simulator to 1-10% of CPU until it added fair scheduling. Deduplicating the
target binary across artifacts makes retention affordable; a `./tests` artifact
embeds 155 to 179 MB today. Hunt on darwin/arm64 until the linux/amd64 replay
divergence (D12) is fixed, since more search amplifies uncontrolled
nondeterminism.

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
| Explicit model snapshot | Canonical World/network/volume state | World snapshots restore. Simulation network and volume snapshots are export-only; `Spec` has no initial network or volume state |
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

Retained reports already suggest the prefix dominates for `testcore` suites. The
boot-only probe takes 1.7 s and whole suites take 1.5 to 3.2 s (*reading
inferred*; no report splits boot from test body). VM and CRIU snapshots are
Linux-only while darwin/arm64 is a qualified platform. The cheap next step is to
record the choice ordinal at which the first test body starts.

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

The oracle machinery today is four helpers (`StateInvariant`, `ExactHistory`,
`NoDuplicateOrLost`, `EventualConvergence`) and no linearizability checker. No
Temporal service runs in the simulation, so fault and recovery oracles have no
Temporal subject yet. Start with oracles that need no simulation:

- **Goroutine leak profile at exit.** Generally available in Go 1.27. About 58%
  of Go blocking bugs come from message passing. Check it under the patched
  runtime first.
- **One independent invariant in a qualified functional suite**, run across
  seeds with `explore`.
- **Metamorphic runs.** One workload tape across dynamic-config settings or
  persistence backends must produce equivalent histories.
- **Self-validating payloads.** WarpStream embeds producer id and sequence in
  each record so consumers check ordering and loss without a model.
- **Porcupine** where a sequential specification exists, such as per-workflow
  update and signal ordering.

Keep a stock-toolchain, real-I/O lane that reuses the workload and checkers, and
audit each modeled fault for granularities it cannot produce. TigerBeetle's
simulator corrupted whole sectors, so a bug reachable only by single-bit flips
never fired.

When a multi-node Temporal subject exists, use fault priors from bug studies:
88% of partition failures need one isolated node, 29% involve partial
partitions, and over 60% of distributed concurrency bugs need one untimely
message.

## Experiments and adoption gates

These stages describe evidence, not a new task queue. Keep policy, workload
generation/shrinking, feature extraction, and oracles in independently testable
deep modules. Runner owns execution, bounds, and publication; domain models own
domain semantics.

| Stage | Experiment | Evidence required to adopt |
| --- | --- | --- |
| Confidence | Qualified known bugs, corrected counterparts, witnesses, loaded replay runs | Separate target, Gomad, model, and oracle failures; record platform limits |
| Visibility | Audit witnesses; instrument only missing scheduling boundaries | Reach a previously inaccessible legal failure, replay exactly, quantify overhead |
| Input search | Typed scenarios and semantic feedback through a per-execution candidate | Additional confirmed bugs or lower discovery cost against equal-budget seeds/fixed workloads |
| Schedule search | Sampling mutator, delay bounds, then partial-order sampling against seed sampling; preemption bounds and PCT after yield points exist | Measured benefit across several bug classes; explain neutral/worse results |
| Diagnosis | Symbolized sites, seed-failure reduction, causality analysis, transposition bisection, resumable minimization, then debugger stops | Smaller same-property failures, exact replay, recoverable interrupted reduction |
| Repeated work | Epoch checkpoints and model snapshots; live snapshots after a prefix-cost study | Accounted savings without isolation/replay regressions |
| Redundant schedules | Ready-case and lane reductions; narrow DPOR after dependency instrumentation | Reduced/unreduced finite searches agree on outcomes and deadlocks within declared bounds |

### Order of work

**First: measure and make legible.** Days each, no toolchain change.

1. Known-bug set with broken and corrected variants in one binary: atomic
   check-then-act, unlock/wake order, close versus send, cancel versus complete,
   lock-order deadlock, equal-deadline tie. Record reachable today and failures
   per 1,000 seeds.
2. Discovery-cost metric in both explorer summaries.
3. Nightly `explore` over the smoke suites on darwin/arm64. Expect virtual-time
   test assumptions (the D16 to D20 class) first; that output is the baseline.
4. Symbolized `inspect --choices`.
5. Leak-profile oracle after a compatibility probe.
6. Guided campaigns stop re-running corpus seeds; the minimizer persists its state.
7. Sampling mutator and exploration start offset; rerun seeds, breadth-first
   frontier, and sampler at equal budgets.
8. `WORKLOAD_SEED` through `--env` on a tape-driven test, once the D17
   environment fix is qualified, and the offline `FuzzMatcherData` import.

**Second: fix the decision space and open the input channel.** Weeks each.

1. One choice-wire revision: why the previous goroutine left, the running
   goroutine on select records, ready-case count, resource per alternative,
   decision fingerprint.
2. Wake-up yield profile, prototyped in a scratch toolchain and judged on the
   known-bug table.
3. No-op pruning and a fixed rule for system goroutines.
4. Per-execution candidate and the application assertion and probe API.
5. In-process policies behind a recorded policy identity: deterministic default
   with delay bounds, then partial-order sampling.
6. Swarm profiles over dynamic config, fault interceptors, and workload verbs in
   the one-box cluster.

**Third: explain and scale.** A quarter or more.

1. Causality analysis and transposition bisection behind one command.
2. DEMi-style minimizer for seed failures and typed inputs.
3. A hunting fleet with fair scheduling and failure deduplication.
4. Metamorphic runs and the real-I/O lane.
5. Multi-node Temporal in the simulation.
6. Atomic call-site instrumentation as its own profile.
7. Trace validation against a TLA+ model where one exists.

**Not pursued:**

- General native DPOR. Records carry no dependency data and executions run to
  tens of thousands of decisions.
- Learned scheduling. Published experiments use about 10K runs per configuration.
- Live process snapshots. Go is not fork-safe and the alternatives are Linux-only.
- The race detector inside Gomad. Stock `-race` stays a separate lane.
- Distributed choice exploration before a policy beats seed sampling.

### Decisions this order needs

- Whether bug finding outranks broader compatibility work such as the downstream
  cell and more platforms.
- Whether a wake-up yield profile may add `proc.go` lines while fn-110 is open.
- Whether a `gomad`-tagged server hook may reintroduce a fixed bug for the
  known-bug table.
- The milestones list preemption bounding and DPOR as out of scope. The first
  stage fits the current constraints; the wire revision, yield profile, and
  pruning need that scope statement changed.

Use atomic compound-operation, cancellation/deadline, shutdown/deadlock,
queue-saturation, and fault/recovery bug families. Retain the real Temporal
update-admission ordering regression alongside small witnesses. Its
[task evidence](../.flow/tasks/fn-100-gomad-f6-a-package-level-functional.2.md)
is a sentence and a commit reference, and only the corrected code is in the
tree, so the broken variant must be reconstructed. It is a clock-tie and
map-order bug that any seed likely finds (*inferred*), which makes it an oracle
and replay benchmark and a weak schedule-search benchmark. Published wins on
small benchmark suites are weak evidence too: 18 of the 49 benchmarks in the
Thomson study fail more than half the time under random scheduling.
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
