# Concurrency testing ideas for Gomad

Research date: 2026-09-30. Shuttle pages observed through `latest` identify version
0.9.4; Loom references use 0.7.2. Primary-source findings for the rewrite of
[`GOMAD_CMP.md`](../../../.plans/GOMAD_CMP.md). This note distinguishes documented
tool behavior from proposed Go adaptations; it does not assert that Gomad implements
the proposals.

## What the existing tools teach

| Tool or technique | Documented behavior | Proposed Gomad adaptation |
| --- | --- | --- |
| Rust Shuttle | Controlled randomized execution, schedule/data replay, multiple search schedulers | Separate the execution mechanism from the search policy; run portfolios over native Go targets |
| Rust Loom | Instrumented small-model exploration, bounds, equivalent-execution elimination | Thorough bounded exploration of small Go concurrency kernels; report the modeled boundaries |
| CHESS and PCT | Systematic controlled testing and low-depth priority scheduling | Explore simple ordering constraints early; keep probabilistic claims conditional on the actual model |
| .NET Coyote | Rewritten concurrency operations, controlled nondeterminism, readable and reproducible traces | Preserve ordinary application code where runtime hooks suffice; reject unsupported escapes |
| MUZZ and RFF | Concurrency-sensitive feedback and schedule-aware mutation | Guide input and execution choices with behavioral novelty, rather than code coverage alone |
| FEST | Adaptive bounded scheduling with semantic timelines and scenario guidance | Reward rare protocol outcomes and concentrate search on relevant concurrent operations |

### Shuttle: scalable search and useful failure evidence

Shuttle substitutes its synchronization and thread primitives for Rust's standard
types, controls their execution, and deliberately prioritizes randomized testing
over exhaustive checking. Passing campaigns do not establish correctness. It also
controls random data and emits a schedule string usable by its replayer. Its
built-in random, PCT, and DFS policies serve different workloads; the documentation
positions exhaustive DFS primarily for small concurrency primitives.
[Shuttle crate documentation](https://docs.rs/shuttle/latest/shuttle/)

The scheduler interface receives the runnable tasks, current task, and whether the
current task is yielding; it also supplies random data. Its state persists across
executions. This provides a boundary between execution and a search policy.
For Gomad, the analogous interface should operate on stable logical identities and
typed choices.
[Shuttle Scheduler](https://docs.rs/shuttle/latest/shuttle/scheduler/trait.Scheduler.html)

`PortfolioRunner` runs several schedulers in parallel and supports stopping at the
first failure or continuing to find several. The transferable parallelism is across
controlled executions. Increasing native parallelism inside one deterministic
execution introduces another control problem.
[Shuttle PortfolioRunner](https://docs.rs/shuttle/latest/shuttle/struct.PortfolioRunner.html)

Shuttle annotations include operation kinds and vector clocks for causal
dependencies; Shuttle Explorer can visualize them. Gomad can borrow the failure
workflow: replay one retained execution, render goroutine timelines, show why a
goroutine became runnable or blocked, and link choices to source sites. Recording
an ordered schedule alone does not establish happens-before edges.
[Shuttle annotations](https://docs.rs/shuttle/latest/shuttle/annotations/index.html)

Shuttle's uncontrolled-nondeterminism checker replays each chosen schedule once,
checking schedule validity and runnable-task sets. Its documentation explicitly
states that a passing check does not prove there is no uncontrolled nondeterminism,
even with an exhaustive underlying scheduler. Gomad should distinguish
observed repeatability, exact artifact replay, and universal guarantees.
[UncontrolledNondeterminismCheckScheduler](https://docs.rs/shuttle/latest/shuttle/scheduler/struct.UncontrolledNondeterminismCheckScheduler.html)

### Loom: small models and explicit limits

Loom controls instrumented synchronization and memory operations. Operations that
use ordinary types remain invisible; uncontrolled randomness and system calls must
be mocked. It reduces equivalent executions, supports preemption bounds, and warns
about exponential growth with threads. Its own caveats say it cannot fully model
Relaxed reordering or some reorderings across atomic variables. Consequently, even
Loom's exhaustive mode is qualified by its instrumentation and supported memory
model. Its value for Gomad is disciplined small-model exploration, not an
unqualified promise to enumerate every possible Go execution.
[Loom documentation and caveats](https://docs.rs/loom/0.7.2/loom/)

Loom checkpoints serialize its execution `Path`; its model runner invokes the test
closure again. These checkpoints preserve exploration position, not a live process
image with heap, threads, and external resources. Gomad's campaign resume,
deterministic prefix replay, and hypothetical live-state snapshots should therefore
be described as distinct capabilities with different performance effects.
[Loom model/checkpoint source](https://docs.rs/loom/0.7.2/src/loom/model.rs.html)

### CHESS and PCT: search rare orderings efficiently

CHESS combines controlled scheduling with systematic exploration and reproducible
failures, checking assertions, deadlocks, livelocks, and data races. Its project
description emphasizes quantified coverage. Gomad should expose the
search bounds and explored space as coverage evidence.
[Microsoft CHESS description](https://www.microsoft.com/en-us/download/details.aspx?id=52619)

PCT defines bug depth as the minimum number of ordering constraints needed to
expose a bug. Its theoretical per-run lower bound is
`1 / (n * k^(d-1))` for the paper's model of `n` threads, `k` steps, and depth `d`.
Random priorities and selected priority decreases target low-depth bugs. Bug depth
is not synonymous with a context-switch budget. A Gomad priority scheduler should
initially be described as inspired by PCT unless its event model, bounds, dynamic
goroutine treatment, and random choices satisfy the proof's assumptions. Adaptive
corpus bias and arbitrary faults do not inherit that probability bound.
[PCT publication](https://www.microsoft.com/en-us/research/publication/a-randomized-scheduler-with-probabilistic-guarantees-of-finding-bugs/),
[original PCT paper](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/asplos277-pct.pdf)

Dynamic partial-order reduction tracks interactions while executing, then finds
backtracking points for alternative dependent orders. Applying it soundly requires
a valid dependence relation. Two runnable choices are not independent simply
because they belong to different goroutines: they may touch the same state,
compete for a timer, or change which operation becomes enabled. Treat unknown
dependence conservatively. Semantic trace hashes can guide sampling without being
proofs that pruned schedules are equivalent.
[Flanagan and Godefroid, DPOR](https://patricegodefroid.github.io/public_psfiles/popl2005.pdf)

### Coyote: control all choices and explain the boundary

Coyote rewrites .NET task-related operations to control execution, including
timeouts and injected failures; it emits a global trace of scheduling and declared
nondeterministic choices for replay. This is evidence that controlled testing can
preserve production APIs through instrumentation. It is not evidence that the same
rewriting technique works unchanged for Go's compiler and runtime.
[How Coyote works](https://microsoft.github.io/coyote/overview/how/)

Coyote reports unsupported concurrency or unrewritten external operations. Its
`--no-repro` escape allows testing to continue while disabling reproducible traces.
Gomad should preserve its stronger reproducible boundary: unsupported behavior or
divergence needs a separate outcome from an application bug. A target watchdog is
also different from a proven deadlock or a failed liveness property.
[Using Coyote: supported scenarios](https://microsoft.github.io/coyote/get-started/using-coyote/)

## What changes when the target language is Go

Go guarantees sequentially consistent outcomes for programs free of data races;
`sync/atomic` operations behave as if placed in a sequentially consistent order.
Racy programs can have outcomes not explained by a sequentially consistent
interleaving. This makes importing Rust's relaxed-atomic memory model the wrong
default. It does not make Go atomics uninteresting: missing an interleaving between
two atomic operations can hide a check-then-act bug. Scheduler control must offer
relevant scheduling points to explore those operation orders.
[Go memory model](https://go.dev/ref/mem)

Go's race detector reports conflicting accesses in executed paths and includes
their goroutine creation stacks. It complements controlled scheduling: an invariant
failure can occur in a race-free program, and a race can exist without causing a
visible invariant failure. Gomad should first investigate whether its runtime
instrumentation and race instrumentation compose without introducing artificial
happens-before edges or nondeterministic callbacks. This compatibility is a
qualification question, not a capability established by the Go documentation.
[Go race detector](https://go.dev/doc/articles/race_detector)

`testing/synctest` provides isolated bubbles and fake time that advances when all
goroutines are durably blocked. Network I/O, system calls, and mutex acquisition
are explicitly outside its durably-blocking set; cleanups/finalizers run outside
bubbles. Its API offers waiting and fake-clock behavior, not explicit schedule
enumeration or replay control. It is useful as a semantic and test-design reference
for time skipping, not a replacement for Gomad's search controller.
[Go 1.27.1 synctest documentation](https://pkg.go.dev/testing/synctest@go1.27.1)

Gomad currently uses a patched Go toolchain and native language constructs, as
documented locally. Compared with Rust primitive substitution, this is an
opportunity to exercise dependencies with fewer source changes. The required audit
still includes compiler-inlined atomics, synchronization fast paths, channel and
`select` choices, time, I/O, randomness, map behavior, and uncontrolled runtime work.
A single processor or a deterministic runnable queue does not alone establish
coverage of all those choices.
[Local Gomad README](../../../tools/gomad3/README.md)

## Working with a guided fuzzer

Native Go fuzzing mutates inputs using code-coverage feedback, minimizes failures,
and replays retained corpus entries as tests. Targets should be deterministic and
self-contained because workers execute in parallel and in nondeterministic order.
This is a plausible first integration: encode workload and bounded schedule-policy
parameters in fuzz input, then invoke a Gomad-qualified target in an isolated
execution. Standard code coverage alone supplies no general custom concurrency
novelty interface; a separate Gomad corpus controller may be needed.
[Go fuzzing documentation](https://go.dev/doc/security/fuzz/)

MUZZ argues that ordinary control-flow coverage misses distinctions among
interleavings. It adds thread-context and scheduling instrumentation and uses their
feedback to preserve valuable inputs. Its evidence is for multithreaded native
targets, not native Go. The transferable hypothesis is to reward concurrent
behavior independently of whether a new branch executed.
[MUZZ paper and abstract](https://www.usenix.org/conference/usenixsecurity20/presentation/chen-hongxu)

RFF mutates abstract positive/negative reads-from constraints and favors rare or
new reads-from pairs. It controls individual shared-memory operations. The authors
report that their early concrete-schedule mutation design produced many infeasible
mutations and costly storage. Borrow structured semantic mutation first: swap a
message/timeout order, delay a particular goroutine, or explore another ready
`select` case. Runnable/select traces alone cannot support RFF's full memory-event
equivalence claims; these require additional instrumentation and modeling.
[RFF, ASPLOS 2024](https://www.comp.nus.edu.sg/~gregory/papers/asplos24.pdf)

FEST combines randomized bounded concurrency testing with feedback-guided schedule
mutation, abstract Lamport timelines, and scenario specifications. Its evaluation
uses distributed-system designs expressed in P. This supports trying semantic
outcome guidance while retaining a bounded scheduler, but provides no established
native-Go performance result. It also observes feedback overhead reducing coverage
for some workloads: compare bugs found per wall-clock budget, not novelty alone.
[FEST, NSDI 2026 prepublication](https://www.usenix.org/system/files/conference/nsdi26/nsdi26spring_li_prepub.pdf)

### Proposed division of responsibility

The fuzzer should generate workloads, operations, parameter values, fault scripts,
and search hints. Gomad should enforce valid execution choices, virtual time,
isolation, observations, oracles, and exact replay. Keep discovery mutation separate
from replay: discovery can choose a fallback for an unavailable hinted actor, but
exact replay must reject a mismatched choice or enabled set.

Start with coarse behavioral feedback: cancellation-versus-completion winner,
timeout-versus-message winner, competing `select` outcome, contention ordering,
retry/duplicate result, and application state-transition labels. These are sampling
features. They must not be called sound independence proofs or complete state
hashes. Prefer stable logical resource and operation identities; exclude addresses,
host times, and incidental goroutine numbers.

Explore both dimensions deliberately. For a novel workload, try several bounded
schedule policies before abandoning it. For a rare execution outcome, mutate
nearby input values and selected ordering constraints. Compare a portfolio of
random schedules, low-depth priority schedules, guided schedules, and bounded
systematic exploration. Preserve an exploration fraction for outcomes that the
current novelty definition misses.

Retain the immutable target/build identity, workload bytes, configuration, complete
typed decision tape and controlled-I/O evidence, plus the failure oracle. Minimize
workload and schedule constraints separately while preserving the same failure
class and verifying exact replay of the minimized artifact. Search progress,
behavioral novelty, and repeatability are separate measurements.

Parallelize isolated runs or exploration prefixes; commit corpus feedback in a
stable order if campaign reproduction matters. Skip idle virtual-time intervals
while preserving timer/message ordering and separately bounding spins or excessive
event counts. Reuse campaign progress and deterministic setup replay before adding
live snapshots. Any future snapshot must include runtime, I/O, random-stream, clock,
coverage, and external-state consequences, and prove restored suffix equivalence
against a cold replay on each qualified platform.
