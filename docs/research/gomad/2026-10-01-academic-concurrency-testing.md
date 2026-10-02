# Academic concurrency-testing survey: search algorithms and feedback signals for Gomad

Research date: 2026-10-01

This is a dated research snapshot of external sources. It records what the cited papers and
project pages say and how they might apply; it is not a statement of what Gomad supports.

## Scope and method

Excluded as already reviewed: CHESS, PCT, DPOR (Flanagan and Godefroid), Loom, Shuttle, Coyote,
MUZZ, RFF, FEST.

Evidence levels used below:

- **Read**: the paper PDF was downloaded and the relevant passages were read as text.
- **Page**: only an abstract page, project page, README, or slide deck was fetched.
- **Unverified**: nothing primary could be fetched; listed separately at the end and not relied on.

Two facts about Gomad shape every fit assessment:

1. Every goroutine switch is voluntary. The running goroutine is never an alternative at a
   decision, and a waker keeps running after `Unlock`/`close`/cancel. Algorithms whose power comes
   from preempting the running thread (PCT priority change points, preemption bounding) have
   nothing to act on until new yield points exist.
2. A choice record holds the alternative count, a set digest, and the selected goroutine identity.
   It holds no resource or dependency information. Functional suites make 8k to 58k branching
   decisions per execution and cost 1.5 to 4 s per fresh-process execution, so a budget is roughly
   10^4 executions per core-day.

## 1. Randomized schedule samplers beyond PCT

### POS, Partial Order Sampling (Read)

Yuan, Yang, Gu. "Partial Order Aware Concurrency Sampling." CAV 2018.
https://www.cs.columbia.edu/~junfeng/papers/pos-cav18.pdf

- Mechanism: each pending event gets an independent uniform random priority and the enabled event
  with the highest priority runs (BasicPOS). POS adds one rule: after running event `e`, reset the
  priority of every pending event that accesses the same object as `e`. The reset stops priority
  constraints from propagating, which is what made BasicPOS as bad as `1/|V|!` in the worst case.
- Evidence: 134.1x stronger overall guarantee than random walk and PCT on randomly generated
  programs, 2.6x faster error detection than both on SCTBench, best on 20 of 32 non-trivial bugs,
  5.0x over RAPOS on micro-benchmarks.
- Controller needs: for each runnable goroutine, the identity of the object its pending operation
  touches. Gomad's choice record does not carry this.
- Fit: good in principle, because POS only chooses among enabled events and never preempts. The
  missing piece is the object identity. A usable approximation is the resource that made each
  goroutine runnable (channel, mutex, timer, netpoll descriptor), which the runtime knows at
  wake time.

### Morpheus (Read)

Yuan, Yang. "Effective Concurrency Testing for Distributed Systems." ASPLOS 2020.
https://www.cs.columbia.edu/~junfeng/papers/morpheus-asplos20.pdf (landing page:
https://www.cs.columbia.edu/~junfeng/papers/morpheus/)

- Mechanism: POS over Erlang message primitives, plus conflict analysis. After each trial it builds
  happens-before with vector clocks, finds operations that actually conflicted, and stores their
  signature `{process id, static code location}` in a history table. In later trials any pending
  operation whose signature never conflicted is scheduled immediately instead of being sampled.
  The authors report that non-conflicting operations were more than 90% of the total in their
  tests, which is what degraded plain POS.
- Evidence: 11 new protocol-level errors in four Erlang systems (RabbitMQ, Mnesia and others);
  280.77% overall advantage over random walk and PCT; conflict analysis improved detection by
  64.82% on average and up to 241.94%.
- Controller needs: per-operation resource identity and a stable signature per decision site.
- Fit: the most relevant single idea for executions with 8k to 58k decisions. Most of those
  decisions are almost certainly irrelevant, and this is a cheap, history-based way to stop
  spending randomness on them. Same prerequisite as POS.

### SURW, Selectively Uniform Random Walk (Page; primary PDF returned 403)

Zhao, Wolff, Mathur, Roychoudhury. "Selectively Uniform Concurrency Testing." ASPLOS 2025.
https://www.comp.nus.edu.sg/~umathur/publications/2025-asplos-zhao-surw-concurrency-testing/

- Mechanism, as described by the abstract page and by the Fray paper (which reimplemented it):
  take a set of "interesting" events as input, and at each step pick a thread with probability
  proportional to its number of remaining interesting events. This makes the sample uniform over
  interleavings of the selected events rather than uniform over per-step choices.
- Evidence: the abstract claims more bugs, found faster, than comparable randomized algorithms.
  Independent data point from Fray: SURW needed an average of 82 iterations to find failures
  versus 190 for POS on real Kafka, Lucene and Guava tests, and took about one day and 200 lines
  to implement.
- Controller needs: an estimate of remaining event counts per goroutine, which implies a profiling
  run and goroutine identities that are stable across runs.
- Fit: plausible. A naive uniform pick among runnable goroutines is heavily biased when executions
  have tens of thousands of decisions; this corrects the bias without needing preemption. The
  open question is whether count estimates from one run stay accurate after the schedule changes.

### Delay bounding (Read)

Emmi, Qadeer, Rakamaric. "Delay-Bounded Scheduling." POPL 2011.
https://microsoft.com/en-us/research/wp-content/uploads/2016/02/popl198ap-emmi.pdf

- Mechanism: fix a deterministic scheduler and allow it to deviate from its own choice at most `K`
  times per execution ("delays"). With `K = 0` there is exactly one execution. The delay happens
  at dispatch time, when a task would normally be picked. The search space is bounded by `I^K`
  for `I` scheduler invocations, independent of the number of tasks.
- Evidence: see the Thomson study below, where iterative delay bounding found 45 of 49 bugs.
- Controller needs: only the runnable set and a deterministic default order.
- Fit: the best structural match of all the samplers. A delay is "pick something other than the
  default at this dispatch", which is exactly a Gomad decision and needs no preemption. With 58k
  decisions, `K = 2` is already about 3.4e9 schedules, so it is a sampling space and a tape
  encoding (sparse deviations from default), not something to enumerate.

### PCTCP and taPCT (Page; primary PDFs returned 403)

Ozkan, Majumdar, Niksic, Tabaei Befrouei, Weissenbacher. "Randomized Testing of Distributed
Systems with Probabilistic Guarantees." OOPSLA 2018.
https://2018.splashcon.org/event/splash-2018-oopsla-randomized-testing-of-distributed-systems-with-probabilistic-guarantees
Ozkan, Majumdar, Oraee. "Trace Aware Random Testing for Distributed Systems." OOPSLA 2019 (not
fetched beyond search metadata). Author lecture slides covering both:
https://soft.vub.ac.be/dare23/assets/talks/ozkan-testingDS-2.pdf

- Mechanism (PCTCP): partition the partial order of message events into causally dependent chains
  online, assign random priorities to chains, and pick `d-1` random priority change points. Chains
  replace threads, so the guarantee depends on the width `w` of the partial order, not on the
  number of messages.
- Evidence: for width `w` and `n` events, a bug of depth `d` is hit with probability at least
  `1/(w^2 n^(d-1))`. The abstract reports bugs found in Zookeeper and Cassandra and better results
  than naive random exploration. taPCT adds partial-order awareness; its numbers were not read.
- Controller needs: causal dependency between events (which message or wake-up caused which), to
  build chains.
- Fit: relevant to the multi-node simulation at the message-delivery level, where "delay this
  chain" is a legal action without preemption. Not applicable to goroutine scheduling as-is.

### Thomson, Donaldson, Betts empirical study (Read)

"Concurrency Testing Using Controlled Schedulers: An Empirical Study." TOPC 2016.
https://www.doc.ic.ac.uk/~afd/papers/2016/TOPC.pdf

- Findings on 49 SCTBench benchmarks with a 100,000 schedule limit: naive controlled random
  scheduling found 43 bugs, more than iterative preemption bounding (38) and DFS (33), and all
  but 2 of those found by iterative delay bounding (45). PCT with `d=3` was best overall: it found
  every bug the others found plus three more and missed one.
- Most bugs needed a schedule bound of 1 or 2. 18 benchmarks were exposed more than 50% of the time
  by random scheduling; the authors say such trivial benchmarks should not be used to justify new
  techniques.
- Takeaway for Gomad: published wins on SCTBench-class benchmarks are weak evidence. Random is a
  strong baseline and delay bounding beats preemption bounding.

### Fray (Read)

Li, Kang, Vikram, Laybourn, Dharanikota, Tiwari, Padhye. "Fray: An Efficient General-Purpose
Concurrency Testing Platform for the JVM." OOPSLA 2025. https://arxiv.org/pdf/2501.12618
(HTML: https://arxiv.org/html/2501.12618v1)

- The largest real-software evaluation found: 2,664 existing tests from Kafka Streams, Lucene and
  Guava, 10 minutes each. Random walk found failing interleavings in 291 tests. POS and SURW each
  found 360+, needing 190 and 82 iterations on average. rr chaos mode found 5; JPF ran none.
- 18 distinct bugs: six atomicity violations, five order violations, five thread leaks, one
  spurious wakeup, one unclassified.
- On the SCTBench port, random walk alone found all 28 bugs.
- What it says about simple algorithms: the control mechanism and applicability mattered far more
  than the search algorithm. Among algorithms, partial-order-aware sampling gave a real but modest
  gain (about 25% more failing tests than random) on real code.

### QL, learning-based scheduling (Read)

Mukherjee, Deligiannis, Biswas, Lal. "Learning-Based Controlled Concurrency Testing." OOPSLA 2020.
https://www.microsoft.com/en-us/research/wp-content/uploads/2019/12/QL-OOPSLA-2020.pdf

- Mechanism: Q-learning where the state is a hash of a partial program observation and the action
  is the worker to schedule. The scheduler only sees the hash. The default observation hashes each
  machine's inbox; richer variants add user-chosen state such as each node's Raft role.
- Evidence: on a buggy Raft implementation, variants with richer state covered more states, more
  leader elections, and more elections with multiple candidates than random and PCT over 10K
  runs, and found the bug more often. The authors state that with no state observation QL
  degrades to random, and that higher coverage does not always imply better bug finding.
- Controller needs: a cheap, deterministic abstract-state hash at each decision.
- Fit: Gomad's semantic probes could supply the hash. The cost is the concern: the paper's
  experiments use about 10K runs, which is roughly a core-day at 1.5 to 4 s per execution.

### PERIOD (Page)

Wen, He, Wu, Xu, Qin. "Controlled Concurrency Testing via Periodical Scheduling." ICSE 2022.
https://sites.google.com/view/period-cct/ and https://github.com/wcventure/PERIOD

- Mechanism as far as the project pages show: a schedule is a sequence of periods, each listing
  which thread executes its key points in that period; exploration is by increasing period depth.
  Key points come from static analysis.
- Evidence claimed on the project page: 10 real CVEs, 36 benchmarks, 5 new bugs. The paper itself
  is closed access and was not read.
- Fit: it serializes threads at instrumented memory and sync operations, which Gomad does not
  have. Low priority.

### Actor bug taxonomy (Page)

Torres Lopez, Marr, Mossenbock, Gonzalez Boix. "A Study of Concurrency Bugs and Advanced
Development Support for Actor-based Programs." https://arxiv.org/abs/1706.07372

- Categories: message protocol violations, message order violations, bad message interleavings,
  plus communication deadlocks, behavioral deadlocks and livelocks. The authors find tooling
  targets deadlocks and protocol violations and leaves livelocks and behavioral deadlocks
  uncovered. Useful as an oracle checklist, not as an algorithm.

### LLM-guided and learned schedule exploration, 2024 to 2026

- No verified primary source was found that uses an LLM to pick scheduling decisions in a
  controlled-concurrency tester and evaluates it against PCT or POS.
- Closest item read: Koorma, Sharma, Edwards, Eslamimehr. "Directed Neuro-Symbolic Stochastic
  Execution for Verification of Distributed Parallel AI Programs."
  https://arxiv.org/pdf/2608.07947. It uses an LLM as a ranking prior over schedules for
  PyTorch/Ray programs and claims 2.9x more concurrency bugs than its strongest baseline. It is a
  preprint on a narrow domain; treat as weak evidence.
- Black-box alternative read: Weiss et al. "Black-Box Bug-Amplification for Multithreaded
  Software." https://arxiv.org/pdf/2507.21318. It trains regression models on repeated trials to
  predict which input configurations trigger a rare failure, reporting often an order of magnitude
  more occurrences than random sampling on 17 bugs. This maps to tuning Gomad seed-campaign
  parameters, not to schedule choice.

## 2. Go-specific concurrency bug finding

### Tu et al. bug study (Read)

"Understanding Real-World Concurrency Bugs in Go." ASPLOS 2019.
https://songlh.github.io/paper/go-study.pdf

- 171 bugs from Docker, Kubernetes, etcd, gRPC, CockroachDB, BoltDB, classified by cause (shared
  memory or message passing) and behavior (blocking or non-blocking).
- About 58% of blocking bugs come from message passing. Typical pattern: an unbuffered channel
  whose receiver leaves through a timeout or another `select` case, leaving the sender blocked
  forever.
- About two thirds of shared-memory non-blocking bugs have traditional causes; the rest come from
  Go features such as anonymous-function capture and `WaitGroup` misuse. Blocking-bug fixes average
  6.8 lines.
- Takeaway: `select` case choice and timer-versus-message ordering are the decisions that matter,
  and Gomad already controls both.

### GFuzz (Read)

Liu, Xia, Liang, Song, Hu. "Who Goes First? Detecting Go Concurrency Bugs via Message
Reordering." ASPLOS 2022. https://par.nsf.gov/servlets/purl/10321048

- Mechanism: rewrite each `select` to prefer one case, with a timeout fallback to avoid false
  deadlocks; mutate the vector of preferred cases; keep orders that are "interesting".
- Feedback table: count of each pair of consecutive channel operations (new pair, or count bucket
  change), new channel created, new channel closed, channel left open, and maximum buffer
  fullness. Priority score: `sum(log2 CountChOpPair) + 10*#CreateCh + 10*#CloseCh +
  10*sum(MaxChBufFull)`.
- A sanitizer tracks which goroutines hold references to a channel to decide when a blocked
  goroutine can never be unblocked.
- Evidence: 184 new bugs in seven systems (170 blocking, 14 non-blocking), 12 false positives,
  124 confirmed, 67 fixed.
- Fit: Gomad already has the control. The reusable part is the feedback signal, which is
  cheap to collect in a patched runtime and is concurrency-specific where code coverage is not.

### GoPie (Page) and ADVOCATE (Page)

Jiang, Wen, Yang, Peng, Yang, Jin. "Effective Concurrency Testing for Go via Directional
Primitive-constrained Interleaving Exploration." ASE 2023. https://github.com/CGCL-codes/GoPie
ADVOCATE: https://github.com/ErikKassubek/ADVOCATE and
https://github.com/ErikKassubek/ADVOCATE/blob/main/doc/fuzzing.md

- GoPie per its README: patches the Go runtime (Go 1.19.1), instruments the target, and mutates
  short chains of primitive operations guided by execution feedback. The paper reports 11 new
  bugs, 9 confirmed (from search metadata; paper not read).
- ADVOCATE per its README: a patched-runtime record and replay tool for Go that requires Go 1.27,
  detects about 20 classes of actual and predicted bugs and leaks from traces, rewrites traces to
  trigger predicted bugs, and integrates three fuzzing modes (GFuzz, a happens-before-improved
  GoPie, and its own "Flow") in an energy-based queue loop.
- Fit: ADVOCATE is the closest existing Go tool to Gomad that is not Gosim, and it targets the
  same Go version. Its trace-based prediction (analyze one trace, then rewrite it to force the
  predicted bug) is an approach Gomad does not have. Worth a code-level read.

### GoAT (Read)

Taheri, Gopalakrishnan. "Automated Dynamic Concurrency Analysis for Go."
https://arxiv.org/pdf/2105.11064

- Uses Go's execution tracer to capture concurrency events, inserts yields at "critical points"
  around concurrency primitives, and proposes measuring sync-pair, blocking-blocked and
  blocked-pair coverage from traces. It reports detecting all GoKer blocking bugs from traces.
  Much of it is described as work in progress.

### GoBench (Page)

Yuan, Li, Lu, Liu, Li, Xue. "GoBench: A Benchmark Suite of Real-World Go Concurrency Bugs."
CGO 2021. https://github.com/timmyyuan/gobench

- 82 real bugs from 9 projects (GoReal) and 103 kernels (GoKer). The GoKer kernels are also
  vendored in the Go tree as test data for the goroutine leak profile. A direct regression suite
  for any Gomad search algorithm: run each kernel under each sampler and compare schedules to
  failure.

### Uber data race study (Read)

Chabbi, Ramanathan. "A Study of Real-World Data Races in Golang." PLDI 2022.
https://arxiv.org/pdf/2204.00764

- Over 2,000 races found and over 1,000 fixed in a 46M-line monorepo. Causes named: transparent
  capture by reference in closures, named return variables, deferred functions, mixing shared
  memory with message passing, value-versus-pointer confusion, the unsafe built-in map, slices,
  and `WaitGroup` placement. Median goroutine count is 2048 per process against 256 threads for
  Java.
- Fit: these are memory-level races. Under Gomad's voluntary-switch model they cannot manifest
  unless the racing accesses are separated by a blocking operation. They need the race detector
  or added yield points, not better scheduling.

### Goroutine leaks: goleak, LeakProf, Golf and the Go 1.27 profile (Read and Page)

Saioc, Shirchenko, Chabbi. "Unveiling and Vanquishing Goroutine Leaks in Enterprise
Microservices." CGO 2024. https://arxiv.org/pdf/2312.12002
Saioc, Lee, Moller, Chabbi. "Dynamic Partial Deadlock Detection and Recovery via Garbage
Collection." ASPLOS 2025. https://cs.au.dk/~amoeller/papers/golf/paper.pdf
Go blog: https://go.dev/blog/goroutine-leak-profiles. Release notes: https://go.dev/doc/go1.27

- goleak (test-time) found 857 existing leaks and blocked about 260 new ones in a year at Uber;
  LeakProf (production sampling) found 24.
- Golf: use GC marking as a sound over-approximation of liveness. Only unblocked goroutines are
  roots; a blocked goroutine becomes a root once the primitive it waits on is marked; whatever is
  never rooted is leaked. It detected 94% of partial deadlocks in microbenchmarks, 50% in a large
  industrial test suite, and 252 in 24 hours on a production service.
- Go status: the `goroutineleak` profile was experimental in Go 1.26 and is generally available
  in Go 1.27, with the `GOEXPERIMENT` removed. It misses leaks on primitives reachable from
  globals or from runnable goroutines, and goroutines blocked in I/O or syscalls.
- Fit: Gomad is pinned to go1.27.1, so this is available now as a per-execution oracle with no
  false positives by construction. Under Gomad the detection point is deterministic, and modeled
  I/O may shrink the I/O blind spot.

### testing/synctest (Page)

https://go.dev/blog/synctest

- The upstream notion of "durably blocked": every goroutine in a bubble is blocked on something
  only another bubble goroutine can release. Mutex waits and external I/O are deliberately not
  durable. This is the same quiescence predicate Gomad needs for virtual-time advance, so the
  upstream definition is a useful reference point.

### Other Go items

- Go-Oracle (Read): Tsimpourlas et al., https://arxiv.org/pdf/2412.08061. A learned pass/fail
  classifier over execution traces for GoBench programs. Not relevant to search.
- Shi, Moldrup, Mathur, Pavlogiannis. "The Complexity of Testing Message-Passing Concurrency."
  POPL 2026. https://arxiv.org/pdf/2505.05162 (Read, abstract and introduction). Complexity
  bounds for checking whether a per-thread channel trace is consistent, parameterized by threads,
  channels and capacities, evaluated on 103 instances from Go projects. Relevant if Gomad ever
  does predictive analysis over recorded traces.
- No other controlled scheduler for unmodified Go was found beyond Gosim, GoPie, GFuzz and
  ADVOCATE.

## 3. Distributed-system model checking and fuzzing of implementations

### SAMC (Read)

Leesatapornwongsa, Hao, Joshi, Lukman, Gunawi. OSDI 2014.
https://ilyasergey.net/CS6213/_static/papers/samc.pdf

- Four reduction policies driven by short protocol-specific rules: local-message independence
  (two messages to one node commute if the handler provably touches disjoint state),
  crash-message independence, crash-recovery symmetry, and reboot-synchronization symmetry.
- Evidence: 12 old deep bugs requiring up to 3 crashes and 3 reboots reproduced one to three orders
  of magnitude faster than black-box DPOR, random plus DPOR, and pure random; some were unreachable
  for the others after 2 days. 2 new bugs.
- Controller needs: message-level and fault-level decisions with semantic identity, and
  user-written predicates.
- Fit: applicable to the multi-node simulation as pruning rules for the bounded BFS over choice
  prefixes, for example "crashing node X before or after delivering a message it would drop
  anyway is the same".

### FlyMC (Page)

Lukman et al. EuroSys 2019.
https://collaborate.princeton.edu/en/publications/flymc-highly-scalable-testing-of-complex-interleavings-in-distrib/

- Abstract only: state symmetry, event independence and parallel flips give 16x on average and up
  to 78x over prior checkers; 12 old bugs reproduced and 10 new ones across 8 systems, without
  random walks or manual checkpoints.

### MoDist and DEMETER (Read and Page)

MoDist: Yang et al. NSDI 2009.
https://www.usenix.org/legacy/event/nsdi09/tech/full_papers/yang/yang.pdf
DEMETER: Guo et al. SOSP 2011.
https://www.microsoft.com/en-us/research/publication/practical-software-model-checking-via-dynamic-interface-reduction/

- MoDist: an interposition layer exposes actions of unmodified systems to a central checker. Its
  virtual clock explores timeouts systematically using static symbolic analysis of time
  comparisons, based on the observation that programs check timeouts soon after reading the clock
  and use time values in simple arithmetic. 35 bugs in Berkeley DB, a production Paxos and
  PacificA, 10 of them protocol-level.
- DEMETER (abstract): dynamic interface reduction explores each node's local interleavings
  separately and only combines distinct interface behaviors, cutting state space by 5x up to
  five orders of magnitude.
- Fit: MoDist's treatment of a timeout as an explicit branch ("fire or not") rather than a
  consequence of clock value is the relevant idea for virtual time. DEMETER's per-node
  decomposition would need per-node replay isolation that a single Go process does not give.

### DEMi (Read)

Scott, Panda, Brajkovic, Necula, Krishnamurthy, Shenker. "Minimizing Faulty Executions of
Distributed Systems." NSDI 2016. https://cs.nyu.edu/~apanda/assets/papers/nsdi16.pdf

- Mechanism: minimize in three stages. First delta-debug the external events (faults, client
  requests). For each candidate subsequence, search for some schedule of internal events that
  still reproduces the violation, using DPOR with prioritized backtracking instead of depth-first
  order. The main heuristic is to stay close to the original schedule. Then minimize internal
  events, then message contents. Events are matched across runs by content fingerprints, because
  removing an external event changes which internal events exist.
- Evidence: across 10 bugs in akka-raft and Spark, outputs were within 1x to 4.6x (median 1.6x)
  of the smallest manually found reproduction.
- Controller needs: a separation of external from internal decisions, and a way to identify "the
  same decision" in a perturbed run.
- Fit: directly relevant to the tape minimizer. Gomad's records are positional (count, digest,
  goroutine identity), so deleting a fault shifts every later decision. A fingerprint for each
  decision, plus a replay mode that follows the original choice when the fingerprint matches and
  falls back to default otherwise, is the prerequisite.

### Mallory (Read)

Meng, Pirlea, Roychoudhury, Sergey. "Greybox Fuzzing of Distributed Systems." CCS 2023.
https://arxiv.org/pdf/2305.02601

- Mechanism: instrument a small set of significant events, build a Lamport timeline per window,
  abstract it into a happens-before summary, and use that as the abstract state. Raw timelines
  are useless as feedback because every run is new; the summary is what makes novelty meaningful.
  A Q-learning policy picks the next fault given the current abstract state and is rewarded for
  new states.
- Evidence: 54.27% more distinct states than Jepsen in 24 hours, same coverage 2.24x faster,
  known bugs 1.87x faster, 22 new bugs in Braft, Dqlite, Redis and others, 18 confirmed.
- Fit: Gomad's semantic probes are the same kind of annotated event. The transferable idea is the
  feedback function: which ordered pairs of probe event types across nodes have been seen in a
  happens-before relation, rather than which probes fired.

### ModelFuzz (Read)

Gulcan, Ozkan, Majumdar, Nagendra. "Model-Guided Fuzzing of Distributed Systems." OOPSLA 2025.
https://arxiv.org/pdf/2410.02307

- Mechanism: map each implementation execution to a path in a TLA+ model using TLC, and keep a
  schedule if it reaches a new abstract model state. Mutations swap the receipt order of two
  messages or change which process crashes.
- Evidence: higher coverage than pure random, line-coverage-guided and trace-coverage-guided
  fuzzing on Two-Phase Commit, etcd-raft and RedisRaft. 12 new bugs, four found only by model
  guidance. The paper argues line coverage ignores message order and trace coverage marks every
  run as new.
- Fit: needs a model. Without one, the result still says what granularity of feedback works: an
  abstract protocol state, coarser than traces and finer than lines.

### Mocket, SandTable, Remix (Page, Page, Read)

Mocket (EuroSys 2023): https://github.com/tcse-iscas/Mocket
SandTable (EuroSys 2024): https://github.com/tangruize/SandTable
Remix: Ouyang et al. "Multi-Grained Specifications for Distributed System Model Checking and
Verification." https://arxiv.org/pdf/2409.14301

- Mocket turns paths of the TLC state graph into tests and forces the implementation along them
  via annotated variables and actions. SandTable explores at specification level and confirms at
  implementation level; its README reports 23 bugs (18 new) in 8 Raft and Zab systems. Remix
  composes fine and coarse specifications per module and found six severe ZooKeeper bugs.
- Fit: all three need a deterministic way to drive the implementation along a prescribed event
  order. Gomad provides that, so it could serve as the execution substrate if a TLA+ model of a
  Temporal protocol exists.

### Trace validation against TLA+ (Read)

Cirstea, Kuppe, Loillier, Merz. "Validating Traces of Distributed Programs Against TLA+
Specifications." https://arxiv.org/pdf/2404.16075
Howard, Kuppe, Ashton, Chamayou, Crooks. "Smart Casual Verification of the Confidential
Consortium Framework." NSDI 2025. https://arxiv.org/pdf/2406.17455
Hackett et al. "Trace Validation of Unmodified Concurrent Systems with OmniLink."
https://arxiv.org/pdf/2601.11836

- Log partial variable updates from the implementation and ask TLC whether some behavior of the
  spec matches the trace; traces may be incomplete. The first paper found spec and
  implementation discrepancies in every system it tried. CCF runs trace validation in CI and
  credits it with six subtle bugs. OmniLink removes the need for a total order by treating events
  as timeboxes.
- Fit: Gomad produces a total order and exact replay, which removes the hardest part of trace
  validation. It is an oracle, not a search algorithm.

### Stateright and PObserve (Page)

https://github.com/stateright/stateright and https://p-org.github.io/P/advanced/pobserve/pobserve/

- Stateright model-checks actor code that also runs on a real network, with symmetry reduction and
  built-in linearizability testers. PObserve checks P monitors against service logs. Both are
  oracle and specification approaches.

### Fault-placement studies and tools

- CrashTuner (Page, SOSP 2019 slides): https://www.sigops.org/s/conferences/sosp/2019/slides/lu.pdf.
  Crash a node just before a read or just after a write of "meta-info" variables (node, container,
  attempt identifiers). 21 new crash-recovery bugs, 10 critical, across 5 systems in 35 hours.
  Most listed bugs are "pre-read" cases: another node uses an identifier of something just removed.
- NEAT study (Read): Alquraan, Takruri, Alfatafta, Al-Kiswany. OSDI 2018.
  https://usenix.org/system/files/osdi18-alquraan.pdf. 136 partition failures in 25 systems. 88%
  can be triggered by isolating a single node, most need three or fewer common events, 62% are
  deterministic, most reproduce on three nodes, and 29% involve partial partitions.
- TaxDC (Read): Leesatapornwongsa, Lukman, Lu, Gunawi. ASPLOS 2016.
  https://ucare.cs.uchicago.edu/pdf/asplos16-TaxDC.pdf. 104 distributed concurrency bugs. 63%
  surface only with faults, more than 60% are triggered by a single untimely message, and order
  violations outnumber atomicity violations about 2:1.
- Timeout bugs (Page, IC2E 2018 slides, Dai et al.):
  https://conferences.computer.org/IC2E/2018/pdf/ic2e18_slides_tingdai.pdf. Root causes: misused
  timeout value 47%, missing timeout 31%, improper handling 12%, unnecessary timeout 5%, clock
  drift 5%.
- Fit: these are priors for the fault tape. Isolate one node, include partial and one-way
  partitions, keep clusters at three nodes, and place crashes next to reads and writes of
  membership and ownership identifiers.

### Antithesis (Page)

https://antithesis.com/docs/introduction/how_antithesis_works.md

- The public documentation says a deterministic hypervisor branches timelines from interesting
  states and that guidance uses reinforcement learning. It gives no algorithmic detail.

## 4. Reduction and diagnosis

- Source DPOR and optimal DPOR (Read): Abdulla, Aronis, Jonsson, Sagonas. POPL 2014.
  https://user.it.uu.se/~parosha/publications/papers/popl2014.pdf. Source sets replace persistent
  sets; wakeup trees make exploration optimal and avoid sleep-set-blocked executions. Implemented
  in Concuerror for Erlang. The paper notes that sleep sets alone prevent fully exploring two
  equivalent interleavings but still start and then abandon them.
- TruSt (Read): Kokologiannakis, Marmanis, Gladstein, Vafeiadis. POPL 2022.
  https://plv.mpi-sws.org/wmc/popl2022-trust-full.pdf. Optimal DPOR with linear memory and no
  shared state between exploration branches, so it parallelizes.
- Must (Read): Enea, Giannakopoulou, Kokologiannakis, Majumdar. OOPSLA 2024.
  https://cdn.amazon.science/b1/79/e8aa270849979221a37c55ebdd1b/model-checking-distributed-protocols-in-must.pdf.
  Optimal DPOR for several message-passing communication models behind a Rust API, used on AWS
  protocol models. It relies on all communication going through a few primitives.
- Parsimonious optimal DPOR (downloaded, not analyzed): https://arxiv.org/pdf/2405.11128.
- Lazy happens-before (Read): Thomson, Donaldson. PPoPP 2015.
  https://www.doc.ic.ac.uk/~afd/papers/2015/PPoPP.pdf. Dropping mutex-induced edges from the
  happens-before relation gives a coarser equivalence, so more schedules count as the same. The
  catch: not every linearization of a lazy relation is feasible, so it is a caching and pruning
  heuristic rather than a sound reduction.
- Fit for all DPOR variants: they need a dependency relation between operations, which the choice
  record lacks, and they are designed for executions orders of magnitude shorter than 8k to 58k
  decisions. Exhaustive use is realistic only on small harnesses. The useful by-product is the
  happens-before relation itself: hashing it (plain or lazy) gives a schedule-equivalence
  fingerprint that deduplicates seeds and corpus entries without capturing program state.
- Time-travel debugging: Delve replays Mozilla rr traces
  (https://github.com/go-delve/delve/blob/master/Documentation/usage/dlv_replay.md). Gomad's exact
  replay already gives re-execution; reverse stepping would need rr, which is Linux-only.

## 5. Snapshot-based exploration

- Nyx-Net (Read): Schumilo et al. https://arxiv.org/pdf/2111.03013. Whole-VM snapshots with a
  cheap incremental second-level snapshot, so a fuzzer can skip a shared test prefix. Up to 300x
  throughput and up to 70% more coverage than AFLNet on ProFuzzBench.
- Go and fork: the Go runtime is multi-threaded and `fork` copies one thread. An open Go proposal
  to detect `fork` from C code describes child processes hanging for exactly this reason
  (https://github.com/golang/go/issues/67174). No primary source on CRIU with Go programs was
  fetched.
- Fit: the payoff is real, since each execution pays 1.5 to 4 s and BFS prefix exploration and
  minimization replay long shared prefixes. But a fork server inside the Go runtime is not
  supported upstream, and VM or CRIU snapshots are Linux-only while darwin/arm64 is a qualified
  platform. A cheaper first step is to measure how much of the per-execution cost is process
  start and setup versus the test body.

## 6. Feedback signals

- Go native fuzzing (Page): https://go.dev/doc/security/fuzz/. Coverage-guided, keeps inputs that
  expand coverage, runs workers in separate processes, minimizes failing inputs, instrumented on
  amd64 and arm64. Code coverage says little about interleavings; ModelFuzz and Mallory both
  argue this.
- IJON (Read): Aschermann, Schumilo, Abbasi, Holz. S&P 2020.
  https://www.gwern.net/doc/reinforcement-learning/exploration/2020-aschermann.pdf. One-line
  annotations feed program state into the coverage map: `IJON_SET(v)` marks a value as seen,
  `IJON_INC`, `IJON_STATE(v)` multiplies edge coverage by a virtual state, and a max primitive
  rewards progress on a numeric quantity. The paper warns that too many virtual states swamp the
  fuzzer.
- AFLNet and StateAFL (Read): https://arxiv.org/pdf/2412.20324 and
  https://arxiv.org/pdf/2110.06253. AFLNet adds a state machine inferred from response codes to
  code coverage. StateAFL infers state by locality-sensitive hashing of long-lived memory.
- Krace alias coverage (Read): Xu, Kashyap, Zhao, Kim. S&P 2020.
  https://gts3.org/assets/papers/2020/xu:krace.pdf. Coverage is the set of instruction pairs
  where a write in one thread is followed by an access to the same address in another. The fuzzer
  stops exploring interleavings for a seed when this stalls. 23 races in ext4 and btrfs.
- SegFuzz (Read): Jeong et al. S&P 2023. https://lifeasageek.github.io/papers/jeong-segfuzz.pdf.
  Decompose an interleaving into segments of at most four shared-memory accesses, use the set of
  seen segments as coverage, and mutate the order within a segment. 21 new kernel bugs.
- SECT (downloaded, abstract only): https://arxiv.org/pdf/2504.21394. Kernel CCT via eBPF.
- Fit: Krace and SegFuzz need memory-access instrumentation. Their idea transfers at a coarser
  grain: record, per shared primitive, short windows of which goroutine or call site operated on
  it consecutively. GFuzz's channel-operation pairs are already that. IJON maps one to one onto
  semantic probes.

## What a Go runtime controller must expose, by technique

| Technique | Runnable set only | Resource identity per alternative | Stable decision fingerprint | Abstract state or probes | New yield points |
| --- | --- | --- | --- | --- | --- |
| Delay bounding | yes | | | | |
| SURW | yes, plus per-goroutine counts | | goroutine identity across runs | | |
| POS | | yes | | | |
| Morpheus conflict analysis | | yes | yes | | |
| PCTCP on messages | | causal parent of each event | | | |
| PCT, preemption bounding | | | | | yes |
| QL, Mallory, ModelFuzz, IJON | yes | | | yes | |
| GFuzz feedback | yes | channel identity | | | |
| DEMi minimization | | | yes | | |
| Happens-before fingerprint | | yes | | | |
| DPOR family | | yes, plus independence | | | |
| Leak profile oracle | none (upstream Go 1.27) | | | | |

Two prerequisites unlock most of the list: record the resource identity behind each alternative,
and give each decision a fingerprint that survives schedule perturbation.

## Ranked: ten most promising ideas not already reviewed

1. **Resource-aware sampling: POS with priority reset.** Sources: POS paper, Fray. Benefit:
   about 25% more failing tests than random walk on real JVM code, 2.6x on SCTBench. Prerequisite:
   resource identity per alternative.
2. **Conflict analysis to skip irrelevant decisions.** Source: Morpheus. Benefit: 65% average
   improvement over plain POS there, and it attacks the 8k to 58k decision count directly.
   Prerequisite: resource identity and decision-site signatures.
3. **Delay-bounded tapes over the default scheduler.** Sources: Emmi et al., Thomson et al.
   Benefit: needs no preemption, gives sparse tapes that mutate and minimize well; 45 of 49
   SCTBench bugs. Prerequisite: a deterministic default choice at every decision.
4. **Goroutine leak profile as an oracle on every execution.** Sources: Go blog, Go 1.27 notes,
   Golf, Tu et al. Benefit: catches the dominant Go blocking-bug class with no false positives.
   Prerequisite: none beyond go1.27.1; check interaction with modeled I/O.
5. **DEMi-style minimization.** Source: DEMi. Benefit: reproductions within a median 1.6x of
   hand-minimized. Prerequisite: decision fingerprints and an external/internal split of the tape.
6. **SURW-style count-weighted choice.** Sources: SURW page, Fray. Benefit: 82 versus 190
   iterations against POS in Fray's evaluation. Prerequisite: a profiling run and stable goroutine
   identity. Primary paper unread.
7. **Happens-before summary feedback over probes.** Sources: Mallory, ModelFuzz. Benefit: 54%
   more states and 1.87x faster bug finding than Jepsen. Prerequisite: vector clocks across
   simulated nodes, or resource identity to derive them.
8. **IJON-style probe primitives (set, max, state product).** Source: IJON. Benefit: lets the
   corpus climb toward deep states that coverage cannot see. Prerequisite: probe API extension and
   a cap on virtual states.
9. **GFuzz channel feedback as a schedule fitness signal.** Source: GFuzz. Benefit: a
   concurrency-specific signal that found 184 bugs when paired with select-order mutation.
   Prerequisite: channel identity and counters in the patched runtime.
10. **Study-driven fault priors and semantic pruning.** Sources: NEAT, TaxDC, CrashTuner slides,
    SAMC. Benefit: concentrates the fault budget (single-node isolation, partial partitions,
    crashes adjacent to membership metadata access) and prunes equivalent crash and message
    orders. Prerequisite: semantic tags on simulation messages and metadata access probes.

Deferred: PCT and preemption bounding (need new yield points first), QL (sample cost against a
10^4 executions per core-day budget), exhaustive DPOR (needs dependency information and short
executions), snapshot branching (Go is not fork-safe; Linux-only options), TLA+-guided testing
(needs a model).

## Unverified or partially verified

- SURW paper: PDF and DOI returned 403. Mechanism taken from the abstract page and Fray's
  description.
- PCTCP and taPCT papers: ACM and MPG copies returned 403. Content from the conference abstract
  page and the author's lecture slides; taPCT evaluation numbers not read.
- PERIOD paper: closed access; project site and README only.
- GoPie and GoBench papers: READMEs only. GoPie's "11 bugs, 9 confirmed" comes from search
  metadata.
- FlyMC, DEMETER: abstracts only. Mocket, SandTable: READMEs only. CrashTuner and the timeout
  study: slide decks only.
- DEMi "up to 97% reduction" and "16x smaller than prior blackbox technique": from search
  metadata of the USENIX abstract, not from the text read.
- Not fetched at all: FCatch (ASPLOS 2018), the CREB crash-recovery study (FSE 2018), Choi and
  Zeller schedule delta debugging (ISSTA 2002), Conzzer, Ankou, FIRM, the AFL++ snapshot LKM,
  any CRIU-with-Go source, Concuerror documentation.
- Whether the Go race detector and the leak profile run correctly under the patched Gomad
  runtime was not checked; both statements above are assumptions about upstream behavior.
