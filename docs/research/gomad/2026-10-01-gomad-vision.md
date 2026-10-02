# What is next for Gomad: assessment of GOMAD_CMP.md and a proposed vision

Research date: 2026-10-01. Dated research snapshot and proposal. It does not define current
support or task state. [Milestones](../../../MILESTONES.md) and Flow-Next specs
govern delivery.

This note assesses every idea in [GOMAD_CMP.md](../../../.plans/GOMAD_CMP.md) against the code,
adds ideas from other projects and papers, and proposes a direction. Four source notes hold the
evidence, with `path:line` references and source URLs:

- [Feasibility: schedule search](2026-10-01-feasibility-schedule-search.md) (code study)
- [Feasibility: workload, feedback, diagnosis](2026-10-01-feasibility-workload-diagnosis.md) (code study)
- [Industry simulation-testing practice](2026-10-01-industry-dst-practice.md) (external sources)
- [Academic concurrency-testing survey](2026-10-01-academic-concurrency-testing.md) (external sources)

Nothing was built or executed for this assessment. Timings come from retained reports. Claims
about runtime behavior that the source notes mark as inferred stay inferred here.

## The short version

Gomad today is a determinism engine with a qualification apparatus around it. Every Make target
and CI job runs `qualify` or `qualify-set`. No job runs `explore`, `--guide`, or `minimize`. The
project has found one server bug (update-admission ordering) and has no number for bugs found
per compute-hour.

The proposed vision: Gomad becomes the place where Temporal concurrency bugs are found,
explained, and kept found. A nightly hunt runs real suites under varied schedules, inputs, and
faults. Each failure arrives as a report that names the two events whose order matters. A table
of known bugs proves that each Gomad change kept or improved detection.

Four findings from the code decide the order of work:

1. **Nothing measures search.** There is no known-bug set, no discovery-cost metric, and no
   hunting job. Every adoption gate in GOMAD_CMP.md compares against a baseline that does not
   exist.
2. **The decision space has the wrong shape.** It lacks the interleavings that matter (a waker
   always keeps running after `Unlock`, `close`, or `cancel()`; atomics never yield) and it is
   flooded with decisions that cannot change behavior (about 46% of branching decisions in the
   Signal suite are select poll-order shuffles, emitted whether or not two cases are ready).
   Search algorithms come after this is fixed.
3. **A target cannot receive a per-execution input or report an application-level fact.** A
   campaign varies only the seed, a choice tape, and a simulation plan. Semantic probes are
   131 compiler-inserted standard-library probes. Workload generation, swarm testing,
   `sometimes` assertions, and input shrinking all wait on the same two primitives.
4. **Failures are not legible.** `inspect --choices` prints hashes although records carry the
   select caller's text offset and artifacts retain the binary. `minimize` rejects every
   artifact without a simulation profile, which is every Temporal functional-test failure.

## Assessment of the ideas in GOMAD_CMP.md

Size: S is days, M is one to two weeks, L is a month or more.

| CMP idea | Verdict | What the code says | Smallest next step |
| --- | --- | --- | --- |
| Visibility witnesses | Do first | The `preemption` fixture already pins the atomic case as a hang. No fixture covers wake-up ordering. | Six broken/corrected fixtures and a reachability table. S, no toolchain change. |
| Scheduling points at atomics | Feasible, do last | The callee-prologue interception covers `sync` methods and typed atomics. Raw `atomic.AddInt64` calls need a new caller-side IR pass; no SSA change. Temporal has 136 raw calls and 113 typed declarations. | Separate execution profile after the wake-up yield proves its value. M to L. |
| Preemption bounding | Blocked | Every switch is voluntary. A Runnable decision fires in `runqget` after the current goroutine left the P, so bound 0 is the whole offered space. | Wake-up yield first (see new ideas). |
| PCT | Blocked, lower priority | Same block. Thomson et al. also found naive random within 2 bugs of iterative delay bounding on 49 benchmarks; Fray found the control mechanism mattered more than the algorithm. | Revisit after yield points exist. |
| DPOR | Defer general DPOR | Runtime records carry no resource or dependency data. Executions have 8k to 58k decisions. World `EquivalenceClass` and simulation lanes do declare independence. | Two sound reductions instead: skip select-poll flips with fewer than two ready cases; treat different simulation lanes as independent. M each. |
| Typed workload generator and guided loop | Blocked on an input channel | `runJob` carries ordinal, seed, choice plan, simulation plan. Scenarios are Go closures. `--guide` spends up to 75% of a campaign re-running seeds whose result is known. | `WORKLOAD_SEED` through `--env` after the D17 fix, on a tape-driven test such as `FuzzMatcherData`. S. |
| Schedule mutation | Mostly exists | `BuildRankPrefix` flips any ordinal and records a fresh suffix. Missing: a sampling controller, a suffix reseed, non-fatal infeasible prefixes. | Sampling mutator beside the BFS controller. S to M, controller only. |
| Go fuzzer bridge | Offline import only | `-test.fuzz` fails closed at `os.StartProcess`. Seed-corpus mode should run with a read-only mount (inferred). `-cover` has no flag and no export path. | Grow a `FuzzMatcherData` corpus natively, replay it under seeds. S. |
| Timeline and symbolized inspection | Cheap first half | Select sites symbolize from the retained binary today. Runnable records have no site and goroutine identity is a one-way hash. | Print `function file:line` in `inspect --choices`. S. |
| Debugger stops | Later | The darwin re-exec returns early at zero slide, so Delve's launch is compatible (inferred). A Runner-managed target needs its inherited descriptors and a disabled watchdog. | Feasibility probe with `dlv exec` on a retained target. M for a prototype. |
| Typed shrinking, minimizer resume | Resume is cheap; shrinking is blocked | Minimizer state is sealed JSON that is never persisted. There is no input to shrink. | Persist state after each commit. S. Then a prefix-length reducer for seed failures. M. |
| Deterministic epochs, round coordinator | No consumer yet | Both engines expose pure `NextRound`/`CommitRound`/`ReplaySegment`. The work is file formats and CLI verbs. | Wait until a search policy beats seeds on real suites. |
| Live snapshots | Defer | Boot-only probe costs 1.7 s; whole suites cost 1.5 to 3.2 s, so boot is most of an execution. Go is not fork-safe; VM and CRIU options are Linux-only. | Record the choice ordinal where the first test body starts. S. |
| Independent oracles | Start outside the simulation | Four oracle helpers, no linearizability checker. The simulation's "Temporal" scenario is raw TCP plus one `SyncMap`. | Add one invariant to a qualified functional suite; add the goroutine leak profile as an oracle. S. |
| Benchmark set | Does not exist | The update-admission regression survives only as fixed code. It is a clock-tie bug, so any seed likely finds it. | Reconstruct it behind a `gomad` tag and add schedule-dependent fixtures. S to M. |

### Statements in GOMAD_CMP.md to correct

- "Try preemption bounding after the controller distinguishes continuing, blocking, yielding,
  and exiting." The controller has nothing to distinguish. The dependency is new yield points.
- The witness list (cancellation/completion, lock handoff, close/send) reads as an atomics
  problem. Those witnesses depend on wake-up ordering, which the document does not name.
- The schedule-mutation paragraph describes a mechanism to build. The primitive exists with
  stricter validation than the text assumes.
- "The existing two-outcome fixture shows no advantage over seed sampling" (also in
  GOMAD_NEXT.md). The test asserts both strategies saw 2 outcomes in 16 executions on a
  one-goroutine program. It does not measure executions to discovery.
- The raw frontier is proposed as a baseline. BFS flips only the first `--max-choice-depth`
  ordinals, which is cluster bootstrap for every functional suite.
- "Explicit model snapshot" is half true. World restores; the simulation network and volume
  models only export.
- "Evaluate OS/VM snapshots only after profiling proves prefix execution dominates." Retained
  reports already suggest it does for `testcore` suites.
- Two README statements feed the same misreading: equal-deadline timers use a separate,
  unrecorded stream that a tape cannot flip, and exploration enumerates poll-order positions
  for every multi-case select, ready or not.

## New ideas

### From the code

| Idea | Why it matters | Size |
| --- | --- | --- |
| Wake-up yield through the existing cooperative-preempt flag in `ready()` | The one change that makes preemption and delay bounds meaningful and reaches most CMP witnesses. `proc.go` is already allowlisted; no compiler work. Inferred, needs a scratch-toolchain prototype. | S to M |
| Richer choice record | `data` is zero on Runnable records. Add why the previous goroutine left, the running goroutine on select records, the ready-case count, and the resource behind each alternative. One wire bump unlocks timelines, pruning, POS, and conflict analysis. | M |
| Prune no-op decisions | Skip select-poll flips with fewer than two ready cases; schedule system goroutines by a fixed rule. The trivial probe records 26 branching decisions with 2 user goroutines. | S to M |
| Exploration start offset | Expand only decisions at ordinal K or later, with K at the first test body. | S to M |
| Suffix reseed in prefix mode | Several different continuations of one prefix. Required for causality analysis. Overlay only. | S |
| Stable `time.AfterFunc` goroutine identities | Context-deadline goroutines take a creation-order counter, so mutated prefixes will diverge on set digests (inferred). | M |
| Deduplicate the target binary across artifacts | Each artifact embeds a 155 to 179 MB binary; the 1 GiB corpus holds about six `./tests` cases. | M |
| Stop re-running corpus seeds in guided campaigns | Guided budget goes to known results until a mutation operator exists. | S |
| Use Temporal's own seams | `testhooks` for assertion emits, dynamic config for knob randomization, `faultinjection` for persistence faults, `event_generator.go` for workloads. No simulation harness needed. | S each |

### From other projects and papers

| Idea | Source | Transfer to Gomad | Prerequisite |
| --- | --- | --- | --- |
| Known-bug table with seeds-to-find | [etcd robustness](https://github.com/etcd-io/etcd/blob/main/tests/robustness/README.md), [Antithesis](https://antithesis.com/docs/best_practices/is_antithesis_working/) | One row per historical bug with a build that reintroduces it; track detection cost across Gomad changes. | None |
| Goroutine leak profile as an oracle | [Go blog](https://go.dev/blog/goroutine-leak-profiles), [Golf](https://cs.au.dk/~amoeller/papers/golf/paper.pdf) | Check every execution at exit. 58% of Go blocking bugs come from message passing ([Tu et al.](https://songlh.github.io/paper/go-study.pdf)). | Verify it runs under the patched runtime (unchecked) |
| Causality analysis | [Antithesis](https://antithesis.com/blog/2026/causality_analysis/) | Replay a failing tape to cut point t, run N reseeded continuations, plot the failure fraction against t. Jumps mark the decisions that fix the outcome. | Suffix reseed |
| Bisection to one transposition | [Hermit](https://developers.facebook.com/blog/post/2022/11/22/hermit-deterministic-linux-testing/) | Binary-search between a passing and a failing tape to one swapped pair and print both stacks. | Symbolized records, decision fingerprints |
| Delay-bounded tapes | [Emmi et al.](https://microsoft.com/en-us/research/wp-content/uploads/2016/02/popl198ap-emmi.pdf), [Thomson et al.](https://www.doc.ic.ac.uk/~afd/papers/2016/TOPC.pdf) | A deterministic default order plus at most K deviations. Needs no preemption. Tapes are sparse, so they mutate and minimize well. | Deterministic default policy |
| POS and conflict analysis | [POS](https://www.cs.columbia.edu/~junfeng/papers/pos-cav18.pdf), [Morpheus](https://www.cs.columbia.edu/~junfeng/papers/morpheus-asplos20.pdf), [Fray](https://arxiv.org/pdf/2501.12618) | Random priorities reset on same-object events; operations whose site never conflicted run immediately. Fray measured about 25% more failing tests than random walk on real JVM code. | Resource identity per alternative |
| Swarm profiles with faults sometimes off | [TigerBeetle](https://tigerbeetle.com/blog/2025-04-23-swarm-testing-data-structures/), [BUGGIFY](https://transactional.blog/simulation/buggify) | Each seed draws a subset of fault kinds, workload verbs, and dynamic-config knobs, then weights. Faults wind down so recovery is checked. | Per-execution input channel |
| `sometimes` assertions and IJON-style probes | [Antithesis](https://antithesis.com/docs/best_practices/sometimes_assertions/), [IJON](https://www.gwern.net/doc/reinforcement-learning/exploration/2020-aschermann.pdf) | Declared catalog so "never reached in N seeds" is a finding; `set`, `max`, and state-product primitives feed the corpus. | Application probe API |
| DEMi-style minimization | [DEMi](https://cs.nyu.edu/~apanda/assets/papers/nsdi16.pdf) | Remove external events first, keep the schedule near the original, match decisions by fingerprint. Median 1.6x of hand-minimized. | Decision fingerprints |
| Starvation intervals | [rr chaos mode](https://robert.ocallahan.org/2016/02/introducing-rr-chaos-mode.html) | A victim goroutine set is unschedulable for a virtual interval. Uniform choice can never produce this. | New choice kind |
| Metamorphic and cross-version runs | [Pebble](https://www.cockroachlabs.com/blog/metamorphic-testing-the-database/), [Jepsen on TigerBeetle](https://jepsen.io/analyses/tigerbeetle-0.16.11) | One workload tape across dynamic-config settings or persistence backends must give equivalent histories. | Workload tape separate from schedule tape |
| Outside-in honesty lane | [Vortex](https://tigerbeetle.com/blog/2025-02-13-a-descent-into-the-vortex/), [Dropbox](https://dropbox.tech/infrastructure/-testing-our-new-sync-engine) | Same workload and checkers on the stock runtime with real I/O; audit each modeled fault for granularities it cannot produce. | Shared workload and checker code |
| Fault priors | [NEAT](https://usenix.org/system/files/osdi18-alquraan.pdf), [TaxDC](https://ucare.cs.uchicago.edu/pdf/asplos16-TaxDC.pdf) | 88% of partition failures need one isolated node; 29% involve partial partitions; over 60% of distributed concurrency bugs need one untimely message. | Multi-node Temporal subject |
| Read ADVOCATE's code | [ADVOCATE](https://github.com/ErikKassubek/ADVOCATE) | A patched-runtime record/replay and fuzzing tool for Go 1.27 that predicts bugs from one trace and rewrites the trace to trigger them. | None |

Recurring lessons from teams that run simulation testing at scale: workload quality limits
results before runtime sophistication does (WarpStream saw no new behavior after about 160
simulated hours); a clean result needs validation against known bugs; developers act on
reports far more often than on interactive replay (MongoDB resolves about 95% of findings
from the report).

## Proposed vision

**Gomad finds Temporal concurrency bugs, explains them, and proves it still can.**

Three properties define done:

- **Finds:** a standing hunt over real suites reports distinct, replayable target failures per
  compute-hour, separated from test assumptions, model defects, and Gomad divergence.
- **Explains:** a failure report names source sites and the smallest ordering difference
  between a passing and a failing run.
- **Proves:** a known-bug table shows seeds-to-find for each bug and gates Gomad changes.

### Horizon 1: measure and make legible (days each, no toolchain change)

1. Known-bug set: reconstruct the update-admission regression behind a `gomad` tag; add six
   broken/corrected fixtures (atomic check-then-act, unlock/wake order, close versus send,
   cancel versus complete, lock-order deadlock, equal-deadline tie). Record reachable today
   and failures per 1,000 seeds.
2. Discovery-cost metric: report the execution ordinal of each first new outcome or failure.
3. Nightly `explore` over the smoke suites on darwin/arm64, where replay is qualified.
4. Symbolized `inspect --choices`.
5. Goroutine leak profile as a per-execution oracle, after a compatibility probe.
6. Stop re-running corpus seeds under `--guide`; persist minimizer state.
7. Sampling mutator and exploration start offset, then rerun seeds versus BFS versus sampler
   at equal budgets.

Expect the nightly hunt to surface virtual-time test assumptions first (the D16 to D20 class).
That output is still the baseline.

### Horizon 2: fix the decision space and open the input channel (weeks each)

1. **One choice-wire revision** that carries why the previous goroutine left, the running
   goroutine on select records, ready-case count, resource identity per alternative, and a
   perturbation-stable decision fingerprint. Each wire or overlay change is a new toolchain
   identity and a requalification, so batch them.
2. **Wake-up yield** as a separate, identity-bound profile, prototyped in a scratch toolchain
   and judged on the Horizon 1 table.
3. **No-op pruning:** ready-case reduction and a fixed rule for system goroutines.
4. **Per-execution candidate:** an input blob delivered like the simulation exploration plan,
   bound into Campaign, Artifact, corpus, and replay identity, with separate workload,
   schedule, and fault streams.
5. **Application assertion and probe API** (`always`, `sometimes`, `reachable`, `set`, `max`)
   hosted behind Temporal's `testhooks` seam, with a declared catalog and campaign-level
   `must-hit`.
6. **In-process policies** behind a recorded policy identity: deterministic default plus
   delay bounds first, POS second.
7. **Swarm profiles** over dynamic config, fault interceptors, and workload verbs in the
   one-box cluster.

### Horizon 3: explain and scale (a quarter or more)

1. Causality analysis and bisection to one transposition as `gomad explain`.
2. DEMi-style minimizer for seed failures and typed inputs.
3. A fleet: fair scheduling across (test, profile) pairs, failures keyed by commit, seed, and
   signature, issues filed with the replay command.
4. Metamorphic runs across persistence backends and dynamic-config settings; outside-in lane.
5. Multi-node Temporal in the simulation. This needs a shared-store or persistence-node model
   and a membership implementation under the `gomad` tag; neither exists.
6. Atomic call-site instrumentation as its own profile.
7. Trace validation against a TLA+ model where one exists.

### Deliberately not pursued

- General native DPOR: no dependency data, tens of thousands of decisions per execution.
- Learned scheduling: published experiments use about 10K runs per configuration; a
  core-day buys roughly 10^4 functional executions.
- Live process snapshots: Go is not fork-safe and the alternatives are Linux-only.
- The race detector inside Gomad: keep stock `-race` as a separate lane.
- Distributing choice exploration before a policy beats seed sampling on a real suite.

## Tensions with committed work

- **D12 first on linux.** More search amplifies uncontrolled nondeterminism. Hunt on
  darwin/arm64 until linux/amd64 replay is strict.
- **fn-110 shrinks the patch; the wake-up yield adds `proc.go` lines.** The yield needs a
  patch-policy decision. Overlay-only policies and the caller-side atomic pass do not.
- **D15 trace capacity.** Yield points and atomic hooks raise decision counts; the Signal suite
  already uses 8.3 MiB of the 64 MiB cap and eight suites overflow it.
- **Milestones list preemption bounding and DPOR as out of scope.** Horizon 1 fits inside the
  current constraints. Horizon 2 items 1 to 3 need the scope statement changed.
- **Identity sprawl.** Every new input, policy, and probe must enter Campaign, Artifact, plan,
  corpus, and replay identity or it becomes a divergence source. The corpus identity already
  omits environment and clock-tick policy.

## Open decisions

1. Does the vision statement above match the intended purpose, or is the priority broader
   test compatibility (downstream cell, more platforms) over bug finding?
2. May a wake-up yield profile add lines to `proc.go` while fn-110 is open?
3. Is a server-side `gomad`-tagged hook that reintroduces a fixed bug acceptable for the
   known-bug table?
4. Should GOMAD_CMP.md be rewritten around these findings, or stay as dated background with
   this note beside it?

## Limits of this assessment

- No experiment ran. Reachability of the wake-up and cancellation witnesses, seed-corpus fuzz
  mode, Delve compatibility, and the leak profile under the patched runtime are inferred.
- linux/amd64 has no retained per-execution timings.
- Several external papers were read only as abstracts, slides, or READMEs. Each source note
  lists them under "Unverified".
- Effort sizes are judgments from reading code.
