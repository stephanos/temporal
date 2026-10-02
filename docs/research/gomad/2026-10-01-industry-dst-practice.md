# Industry DST practice: transferable mechanisms for Gomad

Dated research snapshot (2026-10-01) of external sources, not a statement of Gomad support.

Researched 2026-10-01. Every URL below was fetched and read in this session unless it sits in
the "Not verified" section. Fetches went through a summarizing fetch tool, so quoted numbers are
as that tool reported them from the page; the ShardStore paper was read from extracted PDF text
directly. Sources already reviewed by the team (Shuttle, Loom, CHESS/PCT, Coyote, RFF, FEST,
MUZZ, Gosim basics, FDB testing page, TigerBeetle liveness post, Antithesis assertion basics,
synctest, DPOR) are not re-summarized.

## Ranked: 10 most promising ideas for Gomad

1. **Causality analysis: bug probability over tape position.** Take one failing tape. For a
   set of cut points t, replay the tape exactly up to t, then run N continuations with fresh
   seeded choices and record the fraction that still fail. Plot failure probability against t;
   sharp upward jumps mark the choices that "bake in" the bug, and the logs around the jump are
   where to look. Antithesis does this with VM snapshots; Gomad can do it with tape-prefix
   replay, which it already has (cost is one prefix replay per sample, so sample t by bisection
   on the probability, not uniformly). Antithesis reports it localized a MongoDB corruption to
   a 10 ms window, and for etcd it showed which two faults mattered, letting them retune fault
   rates and raise reproduction from once per 19 days to roughly daily. Complements the
   minimizer: the minimizer shrinks, this explains.
   - https://antithesis.com/blog/2026/causality_analysis/
   - https://www.antithesis.com/docs/reports/likelihood
   - https://antithesis.com/blog/mongo_bug/
   - https://antithesis.com/blog/multiverse_debugging/

2. **Schedule bisection to a single transposition (Hermit "analyze").** Treat schedules as
   strings, find the closest passing/failing pair by edit distance, then binary-search between
   them until two adjacent schedules differ by one swapped pair of events; report both stack
   traces as the root cause. Gomad's choice tapes are exactly such strings. Output is a
   two-stack-trace "these two events raced" report, far more consumable than a minimized tape.
   Caveat: needs a passing tape near the failing one; tapes whose length diverges after a swap
   need a re-alignment rule.
   - https://developers.facebook.com/blog/post/2022/11/22/hermit-deterministic-linux-testing/
   - https://github.com/facebookexperimental/hermit

3. **Swarm testing of the fault/workload/knob configuration, including "faults off".** Per
   seed, pick a random subset of features (fault kinds, workload operations, knobs) and give
   only that subset random weights (TigerBeetle: k-combination, then weights 1-100). Uniform
   "everything on" runs keep systems shallow. Two independent warnings that always-on faults
   hide bugs: Turso missed bugs needing >1 GB databases because faults fired aggressively in
   every run; Will Wilson notes frequent node kills mask corruption that only surfaces on
   recovery. For Gomad: each seed derives a swarm profile (which fault classes exist at all,
   their rates, which workload verbs run), recorded in the tape header.
   - https://tigerbeetle.com/blog/2025-04-23-swarm-testing-data-structures/
   - https://antithesis.com/docs/resources/testing_techniques/
   - https://turso.tech/blog/the-wonders-of-ai
   - https://se-radio.net/?p=9188

4. **Starvation-interval scheduling (rr chaos mode).** Choosing uniformly among runnable
   goroutines can never starve the only runnable goroutine, so timeout-vs-progress bugs stay
   hidden. rr's fix: two priorities, each thread 0.1 probability of "low", periodically
   re-randomized; random timeslices; and short random intervals (up to seconds, capped at 20%
   of runtime) in which low-priority threads do not run at all. In Gomad this is a new tape
   choice kind: "victim goroutine set G is unschedulable for virtual interval d", with virtual
   time allowed to advance past the victims' pending work. Antithesis ships the same idea as
   "thread pausing" and "node pause/throttling" faults.
   - https://robert.ocallahan.org/2016/02/introducing-rr-chaos-mode.html
   - https://antithesis.com/docs/environment/fault_injection/

5. **"Brown M&Ms": a known-bug track record with time-to-find as the search metric.** Keep a
   table of real historical bugs, each with a build/patch that reintroduces it and a make
   target that must rediscover it; record seeds-to-find and track it across Gomad changes.
   etcd keeps exactly this (issue, version introduced, how found, repro target, last commit it
   reproduced on) so framework changes cannot silently lose detection power. Antithesis:
   "if the common bugs are found slowly, you'll probably never find the rare bugs at all".
   Aiven validated a clean 2,200-hour result only by pointing the same harness at upstream
   Kafka and finding KAFKA-19880. This is the cheapest honest answer to "is exploration
   getting better", and it is what should judge items 3, 4 and 6.
   - https://antithesis.com/docs/best_practices/is_antithesis_working/
   - https://github.com/etcd-io/etcd/blob/main/tests/robustness/README.md
   - https://etcd.io/blog/2025/autonomus_testing_with_antithesis/
   - https://aiven.io/blog/deterministic-simulation-testing-in-diskless-apache-kafka

6. **BUGGIFY sites with per-run enablement, knob randomization, and wind-down.** Details
   beyond the FDB testing page: each site is decided on or off once per run on first
   evaluation; enabled sites fire 25% of the time (overridable per site); uses are (a) skip
   optional work, (b) `|| BUGGIFY` on rare error conditions, (c) injected delays, (d)
   randomizing tuning knobs at init (748 knobs); and after 300 simulated seconds fault
   injection is scaled back so recovery can be checked. Gomad runs unmodified code, so the
   natural seams are Temporal's dynamic config (randomize per seed, with test-friendly ranges:
   Antithesis recommends running background jobs every minute, tiny split thresholds, small
   caches) and existing fault-injection interceptors. ShardStore's one escaped bug was a
   cache-miss path never reached because test cache sizes were huge.
   - https://transactional.blog/simulation/buggify
   - https://antithesis.com/docs/best_practices/optimizing/
   - https://jamesbornholt.com/papers/shardstore-sosp21.pdf

7. **Metamorphic configuration and cross-version runs.** Pebble runs one seeded operation
   sequence against many internal configurations and requires identical output; a variant
   chains versions, each starting from the previous version's final on-disk state (four old
   versions plus dev). For Gomad: same workload tape under different dynamic-config/shard-count
   settings must produce equivalent workflow histories; and the durable-volume model makes
   "restart node on build B over state written by build A" a natural fault. Jepsen found three
   upgrade crashes in TigerBeetle because VOPR had no cross-version fuzzing; MongoDB runs
   upgrade/downgrade as standing topologies; TigerBeetle has since made VOPR upgrades mirror
   production restarts. Caveat: schedule tapes do not transfer across configs, only workload
   tapes do, so the workload stream and the scheduler stream must be separable.
   - https://www.cockroachlabs.com/blog/metamorphic-testing-the-database/
   - https://jepsen.io/analyses/tigerbeetle-0.16.11
   - https://antithesis.com/case_studies/mongodb_productivity
   - https://tigerbeetle.com/newsletters/2025-09-12-august-in-tigerland

8. **An outside-in honesty harness plus fault-granularity audits.** Every team with an
   in-process simulator ended up building a second, non-deterministic harness that reuses the
   workload and checkers against real binaries and real I/O: TigerBeetle Vortex (found bugs
   in client batching and connection teardown in four months), Dropbox Trinity Native and
   Heirloom (10x and 100x slower, same seeds), Turso (own simulator plus Antithesis, which
   caught a partial-write bug the simulated I/O loop could not). Jepsen's TigerBeetle findings
   show how models go wrong: VOPR corrupted whole sectors, so checksums always caught it and
   a zero-padding assertion reachable only by single-bit flips never fired. For Gomad: run the
   same functional tests and fault schedule (best effort) on the stock runtime with real
   loopback and disk, diff outcomes, and audit each modeled fault for the granularities it
   cannot produce (partial writes, bit flips, fsync loss, short reads, half-open TCP).
   - https://tigerbeetle.com/blog/2025-02-13-a-descent-into-the-vortex/
   - https://dropbox.tech/infrastructure/-testing-our-new-sync-engine
   - https://turso.tech/blog/introducing-limbo-a-complete-rewrite-of-sqlite-in-rust
   - https://jepsen.io/analyses/tigerbeetle-0.16.11

9. **A continuous fuzzing orchestrator with fair scheduling and a seed database.** TigerBeetle's
   CFO runs all fuzzers on about 1,000 cores and pushes failing seeds to a dashboard; two
   lessons were that long-running fuzzers starved VOPR to 1-10% of CPU until they added a fair
   scheduler, and that respawning more eagerly took utilization from 25% to 80%. Dropbox runs
   tens of millions of seeds nightly and auto-files one task per failing seed with the commit
   hash. RisingWave runs 16 seeds per PR inside a 20-minute budget and unbounded runs on main;
   S2 runs per PR, per commit, and nightly. Resonate auto-files GitHub issues with a repro
   command. For Gomad: a scheduler that splits a core budget fairly across (test, swarm
   profile) pairs, keys failures by (commit, seed, failure signature) for dedupe, and files
   issues with the replay command and tape.
   - https://tigerbeetle.com/newsletters/2025-03-05-february-in-tigerland
   - https://dropbox.tech/infrastructure/-testing-our-new-sync-engine
   - https://www.risingwave.com/blog/applying-deterministic-simulation-the-risingwave-story-part-2-of-2/
   - https://s2.dev/blog/dst
   - https://docs.resonatehq.io/evaluate/how-resonate-is-tested

10. **Retroactive instrumentation on replay, with standard failure artifacts.** Because replay
    is exact, search runs should log almost nothing (Antithesis targets under 200 MB per test
    hour) and a failing tape should be re-run with verbose logs, tracing, packet capture, or a
    debugger: "go back in time and decide I was capturing the traffic all along". MongoDB
    resolves about 95% of bugs from the triage report alone, so the artifact bundle matters
    more than interactive time travel. etcd's bundle is a good template: per-node data dirs,
    per-client operation JSON, an interactive Porcupine `history.html`, and the ability to
    re-validate saved reports after a checker fix. Caveat: Gomad must guarantee extra logging
    does not perturb the tape (logging must not consume scheduler choices or virtual time).
    - https://antithesis.com/blog/multiverse_debugging/
    - https://antithesis.com/docs/best_practices/optimizing/
    - https://antithesis.com/case_studies/mongodb_productivity
    - https://github.com/etcd-io/etcd/blob/main/tests/robustness/README.md
    - https://github.com/anishathalye/porcupine

Near misses: best-representative-per-bucket exploration (Metroid/Zelda), test-composer command
taxonomy, and workload-generator hygiene. All covered below.

## Source notes

### Antithesis

- **Deterministic hypervisor** — https://antithesis.com/blog/deterministic_hypervisor/
  One VM per physical core; throughput over latency; every input-consumption point is a branch
  point in an "input tree". Hard-won: PMC instruction counts miscount about 1 in a trillion;
  they needed 50+ GiB of custom kernel logs per run to find residual nondeterminism.
  Transfer: Gomad's tape is the same input tree without snapshots; justify investing in
  trace-diff tooling for nondeterminism hunts. Caveat: snapshots make their branching cheap;
  Gomad pays prefix replay.
- **Zelda post (SDK guidance)** — https://antithesis.com/blog/zelda/
  `SOMETIMES_EACH` distinguishes a tuple of values and gives each distinct value bounded
  exploration energy; numeric assertions bias toward states with higher values;
  `SOMETIMES_ALL` treats terms as sub-goals and explores the frontier. Warning: adding
  dimensions multiplies buckets and dilutes energy.
  Transfer: Gomad's semantic-coverage corpus could accept developer-declared bucket keys
  (e.g. workflow state x pending-task kinds) with a per-key energy cap.
- **Metroid post** — https://antithesis.com/blog/2025/metroid/
  Bucket by a coarse key (position) and keep one best representative per bucket ranked by
  auxiliary resources, replacing it when a better one arrives; adding resources to the key
  itself was too slow. Transfer: corpus entries keyed by semantic bucket, ranked by a
  secondary objective (shorter prefix, more pending timers, more in-flight transfers).
- **Mario / state-space talk** — https://antithesis.com/blog/sdtalk/
  Thin on mechanism; exploration heatmaps as a way to see where search time goes.
- **Test Composer** — https://antithesis.com/docs/test_templates/ and
  https://antithesis.com/docs/test_templates/test_composer_reference/
  Small commands composed by the platform: `first_` (setup, no faults), `parallel_driver_`
  (may run concurrently, including copies of itself), `serial_driver_`, `singleton_driver_`
  (port of an existing integration test), `anytime_` (invariant checks during faults),
  `eventually_` (kills drivers, stops faults, checks recovery), `finally_` (after drivers end
  naturally, faults stopped). "The more granular the test commands, the more effective."
  Transfer: Temporal functional tests are singleton drivers; splitting a few into granular
  parallel drivers plus anytime/eventually checks would give Gomad a composable workload
  instead of fixed scripts. `eventually_` is a liveness check with a mandated quiet period.
- **Fault injector** — https://antithesis.com/docs/environment/fault_injection/
  Network (latency, partition, clog) on by default; node kill/pause/throttle, clock jumps,
  thread pausing, CPU modulation opt-in; a workload can request a quiet period
  (`ANTITHESIS_STOP_FAULTS`). Fault events are first-class in logs. Transfer: workload-
  requested quiet windows; clock-skew and throttling faults per node in the multi-node harness.
- **Assertion catalog** — https://antithesis.com/docs/properties_assertions/assertions/
  Assertions self-declare at startup so never-reached ones are reported; one property per
  message, tracked across runs. Transfer: a static catalog of Gomad semantic-coverage points
  so "never reached in N seeds" is a finding.
- **Sometimes assertions as a coverage metric** —
  https://antithesis.com/docs/best_practices/sometimes_assertions/
  Locations vs situations; unreached sometimes-assertions mean the workload is weak.
- **Triage report** — https://antithesis.com/docs/reports/triage/
  Findings (new/ongoing/resolved across runs), environment, utilization, per-property status
  with history. Transfer: report per-property history across nightly runs, not per-run lists.
- **Customer write-ups**
  - WarpStream — https://www.warpstream.com/blog/deterministic-simulation-testing-for-our-entire-saas
    Whole-SaaS workload; producers embed (producer id, monotonic sequence) in key, value and
    headers so consumers can check ordering, loss and misrouting without an oracle. Found a
    data race in 233 s and a data-loss race about once per wall-clock hour. New behaviors
    stopped appearing after about 160 simulated hours: the plateau says improve the workload,
    not run longer. Transfer: self-validating payloads; a "new coverage per hour" curve as the
    stop signal for a campaign.
  - etcd — https://etcd.io/blog/2025/autonomus_testing_with_antithesis/
    830 wall hours; reproduced five known bugs, found several new ones, and found flaws in
    their own linearizability model. Tested old releases with known bugs deliberately.
  - MongoDB — https://antithesis.com/case_studies/mongodb_productivity (vendor-authored)
    Eight topologies including upgrade/downgrade; about one bug per 2,500 test hours.
  - Aiven — https://aiven.io/blog/deterministic-simulation-testing-in-diskless-apache-kafka
    "A passing test can be the most suspicious kind of test."
  - Turso — https://turso.tech/blog/introducing-limbo-a-complete-rewrite-of-sqlite-in-rust,
    https://turso.tech/blog/the-wonders-of-ai, https://turso.tech/blog/carl-sverre-ruined-my-day
    Own simulator for speed, Antithesis for real I/O; generators bound what can be found
    ("if your fuzzer never generates indexes..."); driver-equivalence as a cheap property.

### TigerBeetle

- **VOPR internals doc** — https://github.com/tigerbeetle/tigerbeetle/blob/main/docs/internals/vopr.md
  Seed plus commit hash replays; storage checker requires byte-identical data files across
  caught-up replicas. Transfer: a cross-node convergence checker over durable volumes at
  quiescence where Temporal semantics allow it.
- **Fault levels** — https://tigerbeetle.com/blog/2023-07-11-we-put-a-distributed-database-in-the-browser
  Up to 8% read and 9% write corruption per replica; a live visualization of the cluster.
- **Vortex** — https://tigerbeetle.com/blog/2025-02-13-a-descent-into-the-vortex/ (idea 8).
- **Random Fuzzy Thoughts** — https://tigerbeetle.com/blog/2023-03-28-random-fuzzy-thoughts/
  Treat fuzzer bytes as a finite PRNG; when entropy runs out the network becomes reliable and
  the system must converge (a built-in liveness phase); seeds are fragile across generator
  changes, so persist the generated structure, not the seed. Transfer: Gomad tapes already
  persist choices; apply "tape exhausted means benign defaults" as the wind-down rule, and
  persist generated workloads separately from scheduler choices.
- **Swarm testing data structures** — https://tigerbeetle.com/blog/2025-04-23-swarm-testing-data-structures/ (idea 3).
  Also: reflection over the public API so a new method fails compilation until the test
  covers it. Transfer: fail CI when a new dynamic-config key or RPC has no generator entry.
- **Fuzzer blind spots meet Jepsen** — https://tigerbeetle.com/blog/2025-06-06-fuzzer-blind-spots-meet-jepsen/
  Four fuzzers missed a query bug because the "clever" workload pre-registered queries whose
  matches were always consecutive in the index. Fix: dumber random generation plus an exact
  model. "Be wary of introducing unintended constraints in fuzzer workloads."
- **Jepsen analysis** — https://jepsen.io/analyses/tigerbeetle-0.16.11
  What simulation missed and why: sector-only corruption, no cross-version fuzzing, clients
  outside the simulator. (The fetch tool's list of "recommendations" looked partly inferred;
  only the bug-specific causes are relied on here.)
- **It takes two to contract** — https://tigerbeetle.com/blog/2023-12-27-it-takes-two-to-contract/
  Assert the same fact on both sides of a boundary (caller/callee, replica/replica,
  before/after restart). Transfer: Gomad-side checkers that pair an in-memory view with the
  durable-volume view after crash/restart.
- **CFO** — https://tigerbeetle.com/newsletters/2025-03-05-february-in-tigerland (idea 9).
- **Newsletters** — https://tigerbeetle.com/newsletters/2025-09-12-august-in-tigerland,
  https://tigerbeetle.com/newsletters/2026-05-12-april-in-tigerland
  Upgrade simulation made to mirror production; liveness mode found repair livelocks fixed
  with jitter.

### FoundationDB

- **BUGGIFY** — https://transactional.blog/simulation/buggify (idea 6).
- **Will Wilson interviews** — https://se-radio.net/?p=9188,
  https://www.complexsystemspodcast.com/software-testing-with-will-wilson/
  Swarm: run some tests with features off (disable deletes to force growth and GC). Over-
  aggressive faults hide recovery bugs. Buggify as "violate unstated performance contracts"
  (answer at the maximum allowed latency). Branch from fruitful saved states.
- **Strange Loop slides** — https://www.slideshare.net/slideshow/deterministic-simulation-testing/39333008
  Slide titles confirm a "Hurst exponent" section and `swizzle` (stop a random subset of
  nodes, restart in a different order); the slide text gives no usable detail on the Hurst
  exponent. The commonly repeated reading (make failures bursty and correlated rather than
  independent) could not be confirmed from a primary source.

### Go-specific

- **Polar Signals, Go** — https://www.polarsignals.com/blog/posts/2024/05/28/mostly-dst-in-go
  wasip1 + `faketime` + a 10-line runtime seed patch; "mostly" deterministic with unexplained
  residual failures; randomization limited to local run queues. Four data-loss/duplication
  fixes in FrostDB.
- **Polar Signals, Rust rewrite** — https://www.polarsignals.com/blog/posts/2025/07/08/dst-rust
  Left the Go approach because fault injection needed a bespoke interface per dependency and
  deeper runtime changes than they would maintain. Lesson for Gomad: a single choke point for
  fault injection (their message bus; Gomad's modeled FS/TCP layer) is what keeps fault
  coverage from costing per-component work.
- **Gosim design** — https://github.com/jellevandenhooff/gosim/blob/main/docs/design.md
  Running hash over scheduling events to detect nondeterminism; JSON logs with machine,
  goroutine and step for "metatesting"; no documented validation against real Linux.
- **Resonate** — https://docs.resonatehq.io/evaluate/how-resonate-is-tested,
  https://dtornow225.substack.com/p/issue-32-deterministic-simulation
  CI runs each seed twice and diffs logs; differential testing of all storage backends
  against an independent in-memory oracle; Porcupine for histories; predicate-triggered
  branching from a shared seed prefix. Transfer: Temporal has multiple persistence backends;
  the same workload tape across backends must produce equivalent results.
- **etcd robustness** — https://github.com/etcd-io/etcd/blob/main/tests/robustness/README.md
  gofail failpoints, traffic profiles, Porcupine, saved reports (ideas 5 and 10).
- **synctest proposal** — https://github.com/golang/go/issues/67434
  Goroutines blocked in real I/O are not "durably blocked"; fake the network. Gomad's modeled
  TCP already sidesteps this.
- **Other Go forks/tools** — https://pkg.go.dev/github.com/glycerine/pont (Go fork with a
  `-onethread` flag and runtime seed; network-heavy code breaks under one thread),
  https://pkg.go.dev/github.com/tmc/fuzztape/sched (tape-driven scheduler over synctest with
  explicit yields), https://arshnah.is-a.dev/blog/building-detsim (AST rewrite; cheap
  decision-limited "peek" trials and hashed decision traces to skip redundant runs; a trace
  viewer that marks which decisions are load-bearing; map iteration and a torn-write test that
  wrote identical bytes as cautionary leaks). Small projects, lightly evidenced.

### Rust ecosystem

- **madsim / RisingWave** — https://www.risingwave.com/blog/deterministic-simulation-a-new-era-of-distributed-system-testing/,
  https://www.risingwave.com/blog/applying-deterministic-simulation-the-risingwave-story-part-2-of-2/
  libc interception plus patched crates; recovery tests kill nodes under end-to-end SQL and
  compare results; scaling tests migrate shards. Their open problems: suite time bloat and
  "smarter scheduling than more iterations".
- **S2** — https://s2.dev/blog/dst
  turmoil plus libc overrides; a CI meta-test reruns a seed and byte-compares TRACE logs.
  Leaks found: timestamps in HTTP headers, randomized hash maps, dependency time/entropy
  calls, CI-vs-laptop differences. 17 bugs pre-production.
- **sled simulation guide** — https://sled.rs/simulation.html
  `receive`/`tick` state machines and a delivery-time priority queue. Architecture advice,
  little that transfers to an unmodified-code tool.
- **Dropbox Nucleus** — https://dropbox.tech/infrastructure/-testing-our-new-sync-engine
  CanopyCheck generates one tree and perturbs it into the other two (correlated inputs make
  interesting cases common) and minimizes by deleting nodes. Trinity could not minimize:
  small input changes reshuffle scheduling, invalidating the seed. That is the argument for
  Gomad's separate workload and schedule tapes. Trinity reruns each seed to check determinism.

### Others

- **AWS ShardStore** — https://jamesbornholt.com/papers/shardstore-sosp21.pdf
  (landing page https://www.amazon.science/publications/using-lightweight-formal-methods-to-validate-a-key-value-storage-node-in-amazon-s3)
  Conformance against executable reference models; tens of millions of sequences before each
  deploy. Argument biasing (reuse previously-put keys, sizes near page size) is probabilistic
  only and added "only where we have quantitative evidence"; mirroring production
  distributions had no effect. Code coverage is used to find harness blind spots as code
  evolves. Minimization: generic reducers suffice (61 ops to 6) if the system is
  deterministic and the operation alphabet is ordered simplest-first. Crash states:
  coarse per-component flush choices plus explicit flush ops found everything; exhaustive
  block-level enumeration found nothing more and was much slower. Transfer: order Gomad's
  choice alternatives so "zero" is the benign default for the minimizer; keep crash models
  coarse until a brown M&M demands finer.
- **Hermit** — see idea 2. Maintenance mode; 3-6x overhead; about 90% of binaries run
  deterministically.
- **rr chaos mode** — see idea 4.
- **Pebble metamorphic** — see idea 7.
- **Porcupine** — https://github.com/anishathalye/porcupine
  Go linearizability checker with partitioned models and HTML visualization; can blow up on
  some models. Fits Temporal histories only where a sequential spec exists (per-workflow
  update/signal ordering is a candidate).
- **Phil Eaton** — https://notes.eatonphil.com/2024-08-20-deterministic-simulation-testing.html
  Criticisms: you test the mocked edges, not the whole; workload tuning is labor-intensive
  and easy to get shallow; mocks encode your beliefs about failure; seeds die when code
  changes; Go needs runtime surgery.
- **Kafka-19880 commentary** — https://labhub.hopto.org/blog/2026-07-16-deterministic-simulation-testing-kafka-19880.en
  Secondary source; value of DST was the 20-line unit test distilled from the finding.

## Cross-cutting lessons

- **Workload quality is the bottleneck, not runtime.** WarpStream plateau, TigerBeetle blind
  spots, Turso generator limits, ShardStore cache-size miss, Wilson and Eaton on tuning.
- **Clean results need validation.** Brown M&Ms, reachability catalogs, coverage as blind-spot
  detector.
- **Faults must sometimes be absent and must eventually stop.** Swarm, BUGGIFY wind-down,
  `eventually_`, finite entropy.
- **Every simulator needs a real-world counterpart.** Vortex, Trinity Native, Heirloom,
  Antithesis-on-top-of-own-simulator, Jepsen.
- **Determinism is tested, not assumed.** Run-twice-and-diff in CI (S2, Resonate, Trinity),
  running event hash (Gosim).
- **Developers consume reports, not replays.** MongoDB's 95% figure, etcd artifacts, auto-
  filed issues with seed and commit.

## Not verified

- Will Wilson's Strange Loop 2014 talk video (https://www.youtube.com/watch?v=4fFDFbi3toc):
  not fetched; only the slide deck, and only partially. No primary detail on the Hurst
  exponent was obtained.
- TigerBeetle CFO source (`src/scripts/cfo.zig`): 404 at the paths tried; CFO details come
  from the newsletter only. Tigerstyle assertion-density doc: not fetched.
- AWS "Systems Correctness Practices" (https://cacm.acm.org/practice/systems-correctness-practices-at-amazon-web-services/): 403.
- Turso simulator README: 404; `--doublecheck`/`--differential`/shrink flags seen only in
  search snippets.
- Antithesis "Finding more bugs" doc: 404. Formance and Ramp case studies: not fetched.
- Rivet, Sourcegraph, Dolt, Materialize DST posts; Ben Sigelman critique; Go race-detector/rr
  discussions; Elle: nothing substantive found or not searched in depth.
