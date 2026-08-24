# GOMADv3, Loom, and Systematic Concurrency Exploration

**Assessment date:** 2026-08-13

**Local basis:** current working tree, including the in-progress choice-trace implementation

**External-source policy:** official documentation and repositories, plus papers written by the projects' authors

## Executive conclusion

GOMADv3 and Rust Loom overlap in their core premise: make concurrency nondeterminism controllable, rerun a test under different executions, and retain enough evidence to diagnose a failure. They differ in the most important implementation choice. Loom replaces concurrency primitives with model-aware versions and owns a small test execution closely enough to enumerate scheduling, atomic-read, and spurious-wakeup branches. GOMADv3 patches the native Go runtime and surrounds an ordinary Go executable with virtual time, deterministic I/O, process isolation, and durable artifacts. That gives GOMADv3 broader execution fidelity inside its supported Go boundary, but much less information with which to claim exhaustive coverage.

The right lesson is therefore **not to turn GOMADv3 into a Go port of Loom**. The useful path is:

1. finish a choice controller that can validate and force decisions before effects;
2. add durable, bounded prefix exploration as the reference search;
3. add iterative preemption-bounded search and PCT-style sampling for useful coverage at Temporal scale;
4. enrich controlled transitions with stable actor, enabled-set, resource, access, and happens-before metadata before attempting DPOR;
5. pursue exhaustive claims only for an explicitly declared model and bound; and
6. use `World` as a separate explicit-model path for message, timeout, and fault exploration rather than pretending that a `World` hash represents the native Go heap and runtime.

Even after that work, GOMADv3 should not claim memory-model coverage. Its single-P execution can explore sequentially consistent schedules of supported, data-race-free code, but it neither explores racy multi-P executions nor controls which write an ordinary racy read observes. Go itself promises sequential consistency for data-race-free programs and sequentially consistent atomics; racy programs remain erroneous and may have additional behaviors ([Go memory model](https://go.dev/ref/mem)). Race detection and eventual multi-P work must remain separate execution profiles.

## Current GOMADv3 baseline

The present working tree is already beyond plain seeded stress testing:

- A supported run uses one P, disables asynchronous preemption and the system monitor, seeds runtime randomness, and advances virtual time at native-runtime quiescence ([architecture](../tools/gomad3/ARCHITECTURE.md#runtime-choices-and-virtual-time)). Each seed executes in a fresh process with a prepared target and constrained deterministic-I/O envelope.
- The optional choice trace is versioned, bounded, checksummed, and retained in artifacts. Its implementation records three kinds: runnable selection, `select` poll permutation, and the final `select` result ([protocol](../tools/gomad3/protocol/choicewire.json), [runtime hooks](../tools/gomad3/go1.26.4.patch), [validator and projection](../tools/gomad3/internal/choicewire/trace.go)).
- The runnable and `select`-poll records are decisions; the `select` result is an observation. The trace has ordinals, alternative counts, selected indices, and target-relative site fingerprints where available.
- **The trace is observational.** Current `replay` reruns the same seed/profile and compares the resulting trace after execution; it does not feed the recorded choices back into the runtime or reject a mismatch before applying a decision ([README](../tools/gomad3/README.md)).
- Current campaigns choose seeds, optionally retain semantically novel outcomes, and can resume durable batch work. They do not maintain a schedule frontier, force prefixes, or enumerate alternatives. The existing roadmap correctly places those after choice-tape replay ([bug-finding roadmap](GOMAD3_NEXT_BUG_FINDING.md)).
- `World` already has canonical snapshots, semantic digests, explicit event ordering, transition records, and adapter-owned semantics ([World architecture](../tools/gomad3/ARCHITECTURE.md#world)). It is a promising explicit-state substrate, but it cannot establish native-runtime quiescence or summarize arbitrary application state.

Consequently, today's valid description is **seeded schedule sampling with optional choice observation and exact-envelope replay**, not systematic or exhaustive exploration.

## Comparison at a glance

| Dimension | GOMADv3 now | Loom | Most useful lesson |
| --- | --- | --- | --- |
| Program boundary | Native Go executable under a patched runtime and reviewed I/O boundary | Deterministic closure using Loom replacement primitives | GOMAD's transparency and Loom's semantic visibility are a trade-off, not interchangeable advantages. |
| Controlled choices | Seed influences runtime choices; trace currently exposes runnable and `select` activity | Scheduler choices, atomic load/read-from choices, and spurious notifications are explicit path branches ([path source](https://github.com/tokio-rs/loom/blob/948c8cc78b178ede6eeff3afc7d97f2f4ea08559/src/rt/path.rs#L6-L85)) | A search engine needs a complete typed choice interface, not just RNG reproducibility. |
| Search | Seed and guided-seed sampling | DFS over paths, with DPOR and optional preemption bound | Implement forced prefix search first; add reduction only after dependency semantics exist. |
| Replay | Same seed and envelope; final trace comparison when tracing is enabled | Deterministic path/checkpoint isolation and schedule logging | Validate enabled alternatives and the chosen transition before mutation. |
| Reduction | Semantic corpus deduplicates interesting realized outcomes; no schedule reduction | Happens-before/vector-clock dependency analysis plants backtracking points only for interfering operations ([execution source](https://github.com/tokio-rs/loom/blob/948c8cc78b178ede6eeff3afc7d97f2f4ea08559/src/rt/execution.rs#L134-L160)) | Outcome hashes and choice-site novelty do not substitute for a sound independence relation. |
| Memory semantics | Native single-P Go behavior; no memory-read exploration | Partial C11/Rust atomic-memory model | Keep memory-model claims explicitly out of the GOMAD schedule-search claim. |
| External effects | Virtual time, deterministic I/O, explicit `World`, artifact transcripts | Nondeterministic system calls and randomness must be mocked; hidden operations are invisible | GOMAD's I/O/World/artifact work is a real differentiator worth preserving. |
| Scale | Ordinary supported tests, but one fresh process per execution | Small models; exponential growth and a hard small-thread bias | Offer a strategy portfolio and make every cutoff part of the result. |

## What Loom does that GOMADv3 can learn from

### The execution path is the central object

Loom repeatedly executes a deterministic closure. Its path is not merely a list of random numbers: branches distinguish scheduling choices, possible atomic load sources, and spurious notifications, and a depth-first `step` operation backtracks to the next unexplored alternative ([Loom path implementation](https://github.com/tokio-rs/loom/blob/948c8cc78b178ede6eeff3afc7d97f2f4ea08559/src/rt/path.rs#L389-L452)). This enables a precise statement that a path alternative was explored.

GOMADv3's current `kind/site/alternatives/selected` records are a useful start, but a forced-choice protocol needs more:

- a stable logical identity for every enabled alternative, not only its index in a runtime-private array;
- the complete canonical enabled set at the decision;
- whether a switch is forced by blocking/termination or preempts an actor that remains runnable;
- a typed decision phase placed before the target-visible effect;
- an explicit terminal record proving complete tape consumption; and
- divergence on a changed kind, site, enabled set, chosen identity, missing decision, or extra decision.

This should be owned by one deep `ChoiceController` module with `seed`, `record`, `replay`, and `prefix` modes. Search policy belongs outside it. Runtime and adapters present choices; the controller validates/selects; the runner persists results. That boundary is shared by Loom's path, Shuttle's scheduler oracle, and CHESS's replay/record/search phases.

### DPOR depends on semantic causality, not state hashes

Loom's reduction works because its modeled primitives report operations and causality. It tracks vector clocks/happens-before, recognizes dependent accesses, and inserts backtracking points where another ordering may matter ([Loom DPOR implementation](https://docs.rs/loom/latest/src/loom/rt/execution.rs.html#139-160)). Its documentation gives the simple example that reordering two reads of the same atomic cannot change behavior, so that equivalent execution is skipped ([Loom limitations and implementation](https://docs.rs/loom/latest/loom/#combinatorial-explosion-with-many-threads)).

For GOMADv3, the prerequisites for a defensible DPOR are therefore richer than the v1 choice trace:

```text
transition = {
  stable actor/goroutine identity,
  operation kind and phase,
  stable resource identity,
  access mode or semantic effect,
  canonical enabled alternatives,
  happens-before/vector-clock metadata
}
```

The dependency relation must be conservative and owned by the component that knows the semantics: runtime code for channels, mutexes, semaphores, and goroutine lifecycle; timer code for logical deadlines; and adapters for `World` operations and modeled faults. Unknown operations must be treated as dependent, reducing performance rather than silently losing executions.

A repeated `World` digest can safely prune only when the exploration model declares that the digest includes all future-relevant state. It does not include arbitrary goroutine stacks, heap data, channel state, runtime queues, or native timers, so using it as a whole-program visited-state key would be unsound. A `World`-only or explicit-handler explorer can use canonical snapshots and hashes; native-runtime DPOR needs transition dependencies and happens-before information instead.

### Exhaustiveness is always conditional

Loom defaults to checking all modeled interleavings, but its builder also exposes maximum threads, branches, permutations, duration, preemptions, and checkpoint/resume ([Loom `Builder`](https://docs.rs/loom/latest/loom/model/struct.Builder.html)). Its guarantee depends on a deterministic test whose relevant operations all cross Loom primitives. Hidden standard-library synchronization is invisible; loops that require fairness need explicit yields; large models explode combinatorially ([Loom guide](https://docs.rs/loom/latest/loom/#writing-tests), [large-model limitations](https://docs.rs/loom/latest/loom/#large-models)).

Loom also documents important memory-model gaps: sequentially consistent accesses are modeled more weakly as acquire/release, which can produce false alarms, while some load-buffering behaviors are not explored, which can miss bugs ([Loom README](https://github.com/tokio-rs/loom/blob/948c8cc78b178ede6eeff3afc7d97f2f4ea08559/README.md#L68-L80)). GOMADv3 should learn from this candor: the coverage envelope belongs in every artifact and human result, not only in documentation.

Recommended result classes are:

- `model_complete`: every execution of a finite declared model was explored;
- `bound_complete`: every modeled execution within named bounds was explored;
- `sampled`: randomized/PCT/seed executions only;
- `budget_exhausted`: unexplored work remains;
- `cutoff`: a step, fairness, capacity, or liveness limit stopped an execution;
- `replay_divergent`; and
- `invalid_model`: hidden nondeterminism or an unsupported transition invalidated the claim.

The artifact should bind the modeled choice kinds, instrumentation/boundary identity, memory semantics, search and reduction versions, fairness policy, and every run/depth/step/preemption/time/frontier bound. “Exhaustive” without that envelope should never appear.

## Lessons from adjacent projects

### CHESS: the closest architectural ancestor

Microsoft CHESS controlled native program scheduling through concurrency-API wrappers, serialized concurrent activity into tasks, and repeatedly ran an idempotent scenario. Its scheduler separated replay of a known prefix, recording the rest of an execution, and search for a new prefix. At scheduling points it retained both the selected task and enabled tasks. This architecture is closer to GOMADv3 than Loom's replacement-memory model because GOMAD also runs real systems code behind a controlled runtime boundary ([CHESS OSDI paper](https://www.usenix.org/event/osdi08/tech/full_papers/musuvathi/musuvathi_html/index.html)).

CHESS's strongest near-term lesson is iterative preemption bounding. A preemption switches away from an actor that could continue; switches caused by blocking or termination are free. Exploring bounds in increasing order goes deep without a raw schedule-depth cutoff and finds a minimum-preemption counterexample. Once all executions through bound `c` finish, the precise claim is that any missed modeled bug requires more than `c` preemptions—not that the program is correct. The authors show that for fixed `c` the search is polynomial in execution length, though still exponential as `c` rises, and their studied bugs required at most two preemptions ([iterative context-bounding paper](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/chess-pldi07-iterativecontextbounding.pdf)).

For GOMADv3, this is a higher-value first systematic policy than DPOR: it needs stable actor choices and preemption classification, but not a complete dependency model.

### Shuttle and PCT: a scalable strategy portfolio

AWS Shuttle deliberately trades exhaustive assurance for scale. It controls runnable tasks and data nondeterminism, emits a compact schedule that can be forced on replay, and supports random, uniform-random-walk, PCT, exhaustive DFS, replay, annotated schedules, and parallel portfolios ([Shuttle scheduler API](https://docs.rs/shuttle/latest/shuttle/scheduler/), [Shuttle test guide](https://docs.rs/shuttle/latest/src/shuttle/lib.rs.html#160-185)). It can also double-run a schedule and compare steps and runnable sets to detect some uncontrolled nondeterminism, while explicitly warning that a clean result does not prove none exists ([nondeterminism checker](https://docs.rs/shuttle/latest/shuttle/scheduler/struct.UncontrolledNondeterminismCheckScheduler.html)).

PCT is especially relevant to Temporal-sized tests. It assigns randomized priorities and a small number of priority-change points. For a bounded execution with at most `n` threads and `k` steps, the authors prove a per-run lower bound of `1/(n·k^(d-1))` for a bug of depth `d` ([PCT paper](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/asplos277-pct.pdf)). GOMADv3's current seeded RNG does not inherit this result. To make a PCT claim it must implement the actual priority/change-point algorithm, define its step and actor bounds, and retain `n`, `k`, `d`, algorithm version, and seed.

A practical portfolio would run:

- forced-prefix DFS for tiny fixtures and search-engine qualification;
- iterative preemption bounds `0`, `1`, then `2` for small/medium scenarios;
- PCT at several bug depths for large tests;
- ordinary random or uniform-random-walk sampling for diversity; and
- replay/minimization workers for every new failure.

All but completed bounded DFS/preemption searches remain labeled sampled or budget-exhausted.

Shuttle also exposes a boundary GOMAD should avoid copying blindly: all relevant synchronization must use Shuttle replacements, and Shuttle models every atomic as sequentially consistent, so relaxed-memory bugs can be missed ([Shuttle atomic warning](https://docs.rs/shuttle/latest/shuttle/sync/atomic/index.html#warning-about-relaxed-behaviors)). GOMAD's native Go execution avoids the Rust replacement mismatch, but still has no data-race or memory-read explorer.

### Must, TraceForge, and Stateright: reduce domain behavior, not incidental interleavings

Must models distributed processes using a small API for nondeterministic choice plus blocking/nonblocking/selective message receive under explicit FIFO, mailbox, or asynchronous communication semantics. Its optimal DPOR explores each consistent execution graph once; one graph represents many schedule interleavings with the same message reads-from relation ([Must paper](https://www.amazon.science/publications/model-checking-distributed-protocols-in-must), especially Sections 2.1–2.2). The authors' implementation reports assertion failures, deadlocks, and nontermination separately ([Must crate documentation](https://docs.rs/must-mc/latest/must/)). TraceForge follows the same practical pattern for systematic message-delivery exploration and reproducible schedules ([TraceForge repository](https://github.com/awslabs/TraceForge)).

Stateright makes the alternative stateful design explicit: a model provides deterministic initial states, actions, transitions, and properties, while the checker explores and hashes reachable states. Its actor model can use pluggable network semantics, and its breadth-first mode finds shortest single-threaded counterexamples ([model interface](https://github.com/stateright/stateright/blob/ab8c8be9341505e0f71edbe5dd88ed275bd976a4/src/lib.rs#L148-L246), [project features](https://github.com/stateright/stateright/blob/ab8c8be9341505e0f71edbe5dd88ed275bd976a4/README.md#features)). This is the right conceptual model for a finite `World` explorer, but not for pruning native Go executions whose full state GOMAD cannot capture.

This strongly supports a second, explicit-model route for GOMADv3:

- let `World` adapters declare the communication semantics and finite nondeterministic choices;
- explore ready-event/message/fault alternatives independently of incidental goroutine scheduling;
- use canonical snapshots/digests to memoize only the complete declared `World` model state;
- pair modeled execution with safety invariants, terminal-state predicates, or executable reference models; and
- compose it with runtime schedule exploration only under explicit product bounds, because multiplying both spaces will grow quickly.

For Temporal, this route may provide more value than native DPOR for retries, duplicate delivery, cancellation, timeout, failover, and task-state protocols. It can reason about the semantic choices that matter without first teaching the Go runtime whether every pair of low-level operations commutes.

### CDSChecker, GenMC, and Nekara: make the model boundary formal

CDSChecker exhaustively tests fixed-input concurrent data structures under the C/C++ memory model by replacing atomic/thread APIs, using partial-order reduction, and reporting feasible, redundant, buggy, and infeasible executions. Its documentation calls out fairness configuration, finite execution bounds, snapshot-memory exhaustion, and the need to instrument every concurrency API ([CDSChecker project and guide](https://plrg.ics.uci.edu/software_page/42-2/), [authors' paper](https://demsky.eecs.uci.edu/publications/c11modelcheck.pdf)). GenMC similarly attaches sound/complete claims to finite, data-deterministic tests, a supported API subset, and an explicit selected memory model ([GenMC usage](https://github.com/MPI-SWS/genmc/blob/master/doc/manual/usage.md), [features](https://github.com/MPI-SWS/genmc/blob/master/doc/manual/features.md)).

Nekara generalizes concurrency control through a small operation/resource API and makes the key completeness condition explicit: scheduling points must precede every pair of potentially noncommuting actions. Its authors also note that the minimal API lacks enough independence information for partial-order reduction ([Nekara paper](https://www.microsoft.com/en-us/research/wp-content/uploads/2021/09/nekara-ase2021.pdf)). This is direct evidence for separating GOMAD's basic choice oracle from a richer optional dependency protocol.

## Additional projects worth mining

These projects add lessons not already covered by Loom, CHESS, Shuttle, Must, or the weak-memory checkers. Ranked by likely payoff for GOMADv3:

| Priority | Project or family | Most useful lesson for GOMADv3 |
| --- | --- | --- |
| Immediate | Microsoft Coyote | Separate runtime control from search policy; treat uncontrolled nondeterminism and liveness as first-class results |
| Immediate | Vanadium Go concurrency testing | Reuse a Go-native precedent for transitions, resources, clocks, and bounded exploration status |
| Immediate | Lincheck | Separate scenario generation, execution schedules, semantic oracles, and failure minimization |
| `World` | FoundationDB, TigerBeetle, MadSim, Turmoil | Add structured fault campaigns, recovery phases, and coverage feedback without claiming exhaustive search |
| Architectural | Java PathFinder | Use typed, composable choice generators and event listeners; do not copy whole-VM state matching |
| Reference | Déjà Fu and Concuerror | Improve outcome/trace UX and attach DPOR dependencies to semantic operations |

### Coyote: control boundaries and liveness are product features

Microsoft Coyote, the successor to P#, combines a controlled task runtime with pluggable systematic and probabilistic strategies, schedule replay, controlled timers and failures, and explicit detection of unsupported or uncontrolled behavior. Its command-line workflow exposes iteration and step limits, portfolio testing, replay artifacts, and a mode that acknowledges when reproducibility is unavailable ([Coyote testing workflow](https://microsoft.github.io/coyote/get-started/using-coyote/), [architecture overview](https://microsoft.github.io/coyote/overview/how/)). This is a good product-level model for keeping GOMAD's choice controller independent from DFS, preemption-bounded, PCT, and random policies while reporting loss of control as an invalidated result rather than silently weakening the claim.

Coyote also has programmable liveness monitors. Because a finite execution cannot prove an infinite behavior, it detects hot monitor states using explicit thresholds and fair scheduling assumptions; an unfair strategy can transition to a fair random suffix ([Coyote/P# design](https://www.microsoft.com/en-us/research/wp-content/uploads/2023/04/978-3-031-30820-8_26.pdf), Section 3.2). The same work shows why PCT should prioritize logical asynchronous operations rather than treating every transient continuation as a new actor. GOMAD should copy the explicitness, not the exact heuristic: add typed progress monitors, record fairness and thresholds in the coverage envelope, and distinguish a threshold-triggered liveness finding from a mathematical proof of nontermination. Fresh-process isolation remains a GOMAD strength worth preserving over Coyote's repeated in-process execution.

### Vanadium: a direct Go precedent

Vanadium's archived Go concurrency-testing package interposes thread creation, mutex, and channel events as coarse transitions, serializes them through a central scheduler, tracks alternatives, and reruns setup/body/cleanup to enumerate transition sequences. Its API distinguishes state-space exhaustion from time- and execution-count-limited exploration through `Explore`, `ExploreFor`, and `ExploreN` ([package documentation](https://pkg.go.dev/v.io/x/ref/runtime/internal/testing/concurrency)).

The package is old and internal, so it should be mined rather than adopted. Its `choice`, `clock`, `resource`, `state`, and `transition` decomposition is nevertheless the closest Go-specific design precedent for GOMAD's proposed choice oracle and frontier. Compare its stable thread/resource identities and exhaustion tests before inventing new trace semantics, while retaining GOMAD's transparent runtime fork and stronger artifact/replay boundary.

### Lincheck: schedules need semantic oracles and two-axis minimization

Lincheck generates concurrent operation scenarios, explores or stresses their executions, compares observed results with a user-supplied sequential specification, validates final state, and minimizes a failing scenario by removing operations ([result validation](https://kotlinlang.org/docs/lincheck-results-validation.html), [strategy options and minimization](https://kotlinlang.org/docs/lincheck-testing-strategies-options.html)). This suggests an opt-in GOMAD adapter for typed operation histories and executable reference models. A failure reducer should minimize both the semantic input scenario and the forced choice tape; minimizing only scheduler choices can leave a counterexample much larger than necessary.

This is particularly applicable to Temporal queues, registries, task state machines, and idempotency logic where linearizability, serializability, or a purpose-built sequential model is more informative than a terminal assertion. Lincheck's own boundary is also instructive: its model-checking strategy assumes sequential consistency and can miss relaxed-memory or unsupported-library behavior ([strategy limitations](https://kotlinlang.org/docs/lincheck-testing-strategies.html)). GOMAD should make the oracle and modeled-library assumptions equally visible.

### FoundationDB, TigerBeetle, MadSim, and Turmoil: improve `World`, not the exhaustive claim

FoundationDB's deterministic simulation adds semantic fault-injection points, randomized swarm configurations over topology/workload/fault dimensions, named conditional-coverage predicates, and a recovery check after the environment is healed ([FoundationDB testing overview](https://apple.github.io/foundationdb/testing.html), [FoundationDB paper](https://www.foundationdb.org/files/fdb-paper.pdf), Section 4). Too much chaos can collapse executions into a small set of early failure states, so coverage feedback should tune fault intensity rather than merely maximize it.

TigerBeetle's VOPR makes the recovery pattern sharper: use a chaos phase to construct difficult state, then select and heal a viable quorum, stop injecting disruptive faults into it, and require convergence and progress ([VOPR liveness testing](https://tigerbeetle.com/blog/2023-07-06-simulation-testing-for-liveness/), [VOPR documentation](https://github.com/tigerbeetle/tigerbeetle/blob/main/docs/internals/vopr.md)). MadSim and Tokio's Turmoil show how build-time adapters can swap production async/network/filesystem APIs for deterministic simulation facades ([MadSim](https://github.com/madsim-rs/madsim), [Turmoil](https://github.com/tokio-rs/turmoil)).

TigerBeetle also documented a generator blind spot: extensive VOPR runs missed a query bug because the generated queries were too structured, while an independent Jepsen reference model found it ([fuzzer blind-spot analysis](https://tigerbeetle.com/blog/2025-06-06-fuzzer-blind-spots-meet-jepsen/)). This argues for independently reviewing generators and oracles, generating less-constrained semantic inputs, and using model-based or metamorphic checks rather than treating campaign volume as coverage.

These systems are deterministic and replayable but predominantly sample an enormous scenario space; they are not evidence for an exhaustive claim. GOMAD should borrow their named fault plans, swarm dimensions, coverage predicates, and `chaos -> heal -> require progress` campaign shape for `World`, keeping the run seed, scenario configuration, fault choices, and schedule choices separately reproducible. Preserve a real-binary/native profile alongside simulation because mocked networks, disks, clocks, and bindings cannot validate OS and integration contracts.

### Java PathFinder and lower-priority references

Java PathFinder's typed `ChoiceGenerator` interface separates scheduling choices from bounded representatives of data domains and permits cascaded generators when several choices occur at one transition. Its listener/property/publisher interfaces similarly separate execution mechanics, checking, and reporting ([choice generators](https://github.com/javapathfinder/jpf-core/wiki/ChoiceGenerators), [listeners](https://github.com/javapathfinder/jpf-core/wiki/Listeners)). Those extension boundaries are worth copying. Its state matching is not: JPF owns and can snapshot the complete Java VM, while GOMAD cannot safely prune native Go executions using a partial runtime or `World` hash.

Déjà Fu is useful for reporting one representative trace per distinct outcome and providing standard properties such as no deadlock, no uncaught exception, and consistent result ([Déjà Fu](https://github.com/barrucadu/dejafu)). Concuerror reinforces that DPOR dependencies should be defined at language/domain operations such as receive, registry, and ETS actions rather than raw scheduler shuffles ([Concuerror](https://concuerror.com/)). These are useful validation references, but Loom, Shuttle, and Must already cover most of their architectural lessons. Nidhugg and Relacy are likewise valuable weak-memory references, but CDSChecker and GenMC already establish the relevant boundary for GOMAD.

## Prioritized roadmap for GOMADv3

### P0: preserve the current trust boundary

Before building search on it, finish the known replay, resume, publication, network, and qualification-label correctness work. A search engine amplifies every replay and recovery defect. Preserve fresh-process execution, immutable content-addressed artifacts, bounded protocols, deterministic I/O, fail-closed unsupported behavior, and exact identity checks; these are stronger operational foundations than most research checkers provide.

### P1: force and validate the existing choices

Evolve choice trace v1 into a controller without yet widening the instrumented surface:

1. give runnable alternatives stable per-run actor identities, creation lineage, and explicit logical-operation grouping rather than only queue offsets;
2. record the canonical enabled identity set;
3. implement `record`, `replay`, and `prefix` modes;
4. validate before mutating the run queue or `select` poll order;
5. fail on incomplete/extra tape and trace overflow; and
6. add a double-run nondeterminism audit that compares enabled sets, not only final trace hashes.

Qualification tests should deliberately change an enabled set and prove divergence occurs before target-visible mutation.

### P2: add a durable bounded frontier and honest coverage records

Implement raw alternative-prefix DFS as a reference algorithm. Every observed branching decision contributes unexplored alternative prefixes. Persist the frontier transactionally with run/depth/step/frontier-byte/time limits and resume it without rediscovery. This will be redundant but easy to validate against tiny state machines where the outcome count is known.

Artifacts must distinguish frontier exhaustion from capacity or time exhaustion and store the full coverage envelope described above.

### P3: add preemption-bounded and PCT policies

Add stable actor-continuation tracking so a decision records whether switching consumes a preemption. Define whether a spawned goroutine inherits its parent's logical operation or becomes a new PCT actor; assigning fresh priorities to every transient continuation can degrade the intended search. Explore bounds iteratively and publish the highest completely exhausted bound. In parallel, implement PCT as a separate sampled strategy with its theorem inputs retained. Add a portfolio scheduler above the shared choice-controller interface; do not embed strategy into the runtime.

Measure executions and distinct semantic/failure outcomes per compute-hour against current seed sampling on both small fixtures and representative Temporal tests.

### P4: widen choice coverage deliberately

Current traces do not cover enough decisions for systematic claims. Add one domain at a time with pre-apply control and tests:

1. goroutine runnable identity and lifecycle;
2. actual ready-`select` winner semantics, not only poll randomization/result observation;
3. channel send/receive pairing and wake-up choice;
4. mutex/semaphore/condition wake-up choice;
5. equal-deadline timer ordering;
6. controlled application randomness/data choices; and
7. explicit `World` message and fault choices.

Map seeds and other non-scheduling randomness should either become typed data choices or remain outside the schedule-completeness envelope. Do not add compiler checkpoints broadly until a minimized workload demonstrates a missing, relevant preemption point.

### P5: prototype dependency metadata, then DPOR

Keep raw DFS as the oracle. Add semantic transition records and vector clocks for a narrow set—initially goroutine lifecycle, channels, and mutexes. Build conservative independence tests and differential tests showing that reduced and unreduced search reach identical outcomes on bounded fixtures. Only then add DPOR, and call it complete solely for the modeled transition set and declared bounds.

Treat unknown runtime operations, I/O, faults, and application data choices as dependent unless an owning adapter supplies a reviewed commutativity rule. A false dependency costs executions; a false independence can erase the only failing execution.

### P6: develop `World` exploration as a separate product capability

Add a finite `World` scenario API with typed nondeterministic inputs, message delivery, logical timeout, cancellation, and fault choices. Require a canonical full-model snapshot before using hash-based visited-state pruning. Support typed operation histories, executable reference models, state invariants, and minimization of both scenario operations and choice tapes. Start with mailbox and purpose-built Temporal state-machine harnesses rather than transparent TCP.

For sampled campaigns, make topology, workload, fault vocabulary, and fault intensity explicit swarm dimensions with named coverage predicates. Add a two-phase liveness mode that first constructs difficult state under chaos, then heals a viable subset and requires convergence or progress under a recorded fairness policy. Keep these sampled/recovery claims separate from finite `World` frontier exhaustion.

This path can reach meaningful exhaustive results sooner than native-runtime DPOR because the model deliberately owns its full state and dependency semantics. Label it explicit-model verification, not verification of an unmodified multi-service Temporal deployment.

### P7: keep races and true parallelism orthogonal

Run Go's race detector in a separate, non-GOMAD profile and correlate findings by test/failure identity. Do not combine its instrumentation with current schedule reproducibility claims. Multi-P record/replay would require control over parallel synchronization and memory observations, not merely more runnable-choice records; pursue it only as a separate research project with an explicit Go-memory-model statement.

## Claims GOMADv3 should and should not make

After P2, GOMADv3 could truthfully say: “All modeled choice prefixes within these recorded bounds were executed for this exact target and platform bundle.” After P3 it could additionally report a completed preemption bound or a PCT sampling guarantee. After a qualified P5 it could say: “One representative per modeled dependency-equivalence class was explored within this envelope.” For a finite P6 model it could say: “All declared `World` behaviors were explored.”

It still should not say:

- all Go executions were explored;
- the program is race-free;
- the Go memory model was exhaustively checked;
- an unmodified distributed deployment was verified;
- absence of a failure proves correctness outside the recorded inputs, model, platform, and bounds; or
- a semantic-state or choice-trace hash proves schedule equivalence.

## Bottom line

Loom demonstrates what makes exhaustive concurrency testing credible: a finite deterministic model, complete interception of the claimed nondeterminism, explicit path branches, causal dependency semantics, systematic backtracking, and visible bounds. CHESS and Coyote show how much of that can apply to native systems code, Shuttle and PCT show how to scale when exhaustive search is unrealistic, and Must shows that distributed behavior is often better reduced at the message/semantic level than at the thread-interleaving level. Vanadium supplies a direct Go precedent, Lincheck supplies the missing semantic-oracle and minimization layer, and FoundationDB/TigerBeetle supply a stronger campaign model for `World`.

GOMADv3 already has unusually strong runtime transparency, virtual time, process isolation, deterministic-I/O evidence, replay envelopes, and durable campaign machinery. Its highest-leverage next move is to turn the current observational runnable/`select` trace into a small trustworthy choice oracle, then build bounded search policies above it. DPOR should follow semantic transition metadata, not precede it; `World` hashing should power a separate explicit-model explorer; and memory-model coverage should remain an explicit non-goal.
