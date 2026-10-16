# Finding bugs with Gomad

**Date:** 2026-10-16. **Scope:** the WASM backend. Native execution is frozen under
[fn-184](../.flow/specs/fn-184-gomad-native-backend-isolation.md) and deferred under
fn-128 and fn-149.

This plan turns Gomad from a determinism engine into a bug-finding loop. It records
what the code supports today, what the existing research recommends, and five
investments in dependency order with the smallest version of each. It does not
track tasks; Flow-Next specs own those. [GOMAD_NEXT.md](GOMAD_NEXT.md) holds the
broader roadmap and [GOMAD_CMP.md](GOMAD_CMP.md) the research this draws on.

## 1. Where we stand

### What Gomad does today

Every Make target and CI job runs `qualify` or `qualify-set`, which prove same-seed
repeatability and exact replay. No Make target, workflow or job runs `explore`,
`--guide` or `minimize`. Gomad has found one server bug (the update-admission
ordering regression in `compareAdmission`, fixed in `97d9925ac`) and diagnosed five
virtual-time test assumptions (D16 to D20), three resolved by test changes. There
is no known-bug table and no number for bugs found per compute-hour.

What a Temporal engineer gets: a CLI that runs one functional suite under N
scheduling seeds with virtual time, keeps a failing execution as an artifact, and
replays it exactly. Functional suites run in 1.5 to 4 s each after a one-time
build. Retained timing data (native, darwin/arm64, traced):

| Workload | Wall per execution | Virtual time | Choice records |
| --- | --- | --- | --- |
| `TestFrontendSystemInfo` (boot plus one RPC) | 1.7 s | 0 s | 4,562 |
| `TestUserTimersTestSuite` | 1.5 to 2.4 s | 4.0 s | 8,123 |
| `TestActivityTestSuite` | 1.7 to 3.5 s | 13.6 s | 42,165 |
| `TestSignalWorkflowTestSuiteChasm` | 2.2 to 2.3 s | 57.0 s | 86,243 |
| `TestWorkflowUpdateSuite` | 2.9 to 3.2 s | 22.4 s | 207,900 |

Cluster boot is most of every functional execution: the boot-only probe costs as
much wall time as whole suites and records the first 4,500 decisions.

### Constraints that shape the plan

- **A target is one Go process.** Subprocess launch and external databases fail
  closed. The target runs with loopback networking, in-memory SQLite and virtual
  time. Coverage-instrumented binaries are rejected.
- **The only per-execution input is the scheduling seed.** A campaign fixes argv,
  environment and mounts once; the per-execution job carries seed, choice tape and
  simulation plan. Faults and workload variation must be driven from inside the
  target until an input channel exists. `--env` entries are recorded inputs and
  can carry static knobs.
- **Native semantic probes do not exist on WASM.** The native backend had 131
  compiler-inserted stdlib probes behind `--coverage=semantic` and
  `--require-probe`. The cooperative WASM operator rejects semantic coverage and
  guidance. Application code cannot declare a probe on either backend.
- **Choice exploration is systematic but shallow by default.** It is breadth-first
  over recorded decisions within `--max-choice-depth`. `--choice-start-ordinal=N`
  keeps decisions before N in every forced prefix and expands only later ones.
  Select polls of the seven proven no-op shapes no longer expand the frontier.
  Combined exploration over network, storage, fault and crash dimensions exists
  only for simulation targets, not for `testcore` clusters.
- **Failures are not yet legible.** `minimize` accepts only combined-simulation
  failures. The guided corpus caps at 1 GiB and each artifact embeds a 155 to
  179 MB target binary, so it cannot hold functional cases.
- **Linux replay still diverges** on about one tier-3 seed-run in 26 (D12), so a
  failing artifact there may not reproduce. Hunt where replay is qualified.

### Why Omes and the mixed-brain test cannot be targets

`tests/mixedbrain` starts two server processes built from two git refs, Postgres,
an Omes binary launched with `exec`, and a frontend proxy that splits RPCs between
versions. Each of those is outside the deterministic boundary, and a campaign
executes one immutable prepared binary, so version skew cannot be expressed at
all. The test stays on the native lane for what it checks: upgrade compatibility.

What does transfer: Omes scenario bodies are Go code driving the Go SDK, and Gomad
already carries an exact adapter for `go.temporal.io/sdk@v1.48.0`. A test that
boots a `testcore` cluster, runs an in-process SDK worker and executes the
throughput_stress or scheduler_stress scenario body for a bounded iteration count
is a valid target. Its oracles are the scenario's own checks plus the log
scanners from `tests/mixedbrain/log_scan_util.go` run over the in-process server
log. Omes's `math/rand` workload choices are then coupled to the scheduling seed,
which is fine for hunting but means the workload cannot be shrunk independently
of the schedule.

## 2. What the research already says

The research is done and sits unexecuted. Four notes dated 2026-10-01 under
[docs/research/gomad](../docs/research/gomad/README.md) plus [GOMAD_CMP.md](GOMAD_CMP.md)
and its reassessment in the vision note cover Loom, Shuttle, CHESS, PCT, Coyote,
FoundationDB, TigerBeetle, Antithesis, Hermit, rr chaos mode, DEMi, POS, Fray,
ADVOCATE, GFuzz, etcd robustness and about thirty papers.

### Findings that decide the order of work

1. **Nothing measures search.** No known-bug set, no discovery-cost metric, no
   hunting job. Every adoption gate in GOMAD_CMP.md compares against a baseline
   that does not exist.
2. **The decision space has the wrong shape.** About 46% of branching decisions in
   the Signal suite are select poll-order shuffles that cannot change behavior
   (now partly pruned), and a woken goroutine never yields after `Unlock`, `close`
   or `cancel()`, so the interleavings that cause real bugs are unreachable
   regardless of search policy. Thomson et al. found naive random within two bugs
   of delay bounding on 49 benchmarks; Fray found the control mechanism mattered
   more than the algorithm. Search algorithms come after the decision space is
   fixed.
3. **A target cannot receive a per-execution input or report an application-level
   fact.** Workload generation, swarm testing, `sometimes` assertions and input
   shrinking all wait on the same two primitives.
4. **Failures are not legible.** Symbolized sites, causality analysis and
   transposition bisection are all cheap once records carry site identity.

### Mechanisms ranked as most transferable

From the industry note, in its order: causality analysis over tape position
(Antithesis); bisection to one transposition (Hermit); swarm profiles over fault
kinds, workload verbs and knobs with faults sometimes off (TigerBeetle, BUGGIFY);
starvation intervals as a new choice kind (rr chaos mode); a known-bug table with
seeds-to-find as the metric (etcd, Antithesis); BUGGIFY-style per-run enablement
with wind-down; metamorphic and cross-version runs (Pebble); an outside-in real-I/O
lane (Vortex, Dropbox); a fuzzing orchestrator with fair scheduling and a seed
database; retroactive instrumentation on replay.

Recurring lessons: workload quality limits results before runtime sophistication
(WarpStream saw nothing new after about 160 simulated hours); a clean result needs
validation against known bugs; developers act on reports far more than on
interactive replay (MongoDB resolves about 95% from the report).

### Temporal seams the research names and Gomad does not use yet

`common/testing/testhooks` (tag `test_dep`) for assertion emits and targeted
yields; dynamic config for knob randomization; `common/persistence/faultinjection`
and `grpcfaultstest` for fault inputs; `common/testing/event_generator.go` as a
model-based history generator.

Umpire's plans contain the complementary argument (explore in the cheap model,
execute selected witnesses in the implementation; keep deterministic simulation as
a second search mechanism) but neither repo names the other. Umpire integration
is deferred by owner decision on 2026-10-16.

## 3. Hunt with what exists

Days. No toolchain change. Needs no spec.

### The workload

One test under `tests/gomadfunctional` beside the existing boot probe: start a
`testcore` cluster on in-memory SQLite, start a workflow, run one workflow task
through the in-process task poller, complete it. Oracles that need no model: the
history equals the expected event sequence, describe reports closed, no soft
assertion in the server log, no leaked goroutines at exit.

### Dimension 1: schedule permutations

1. Run one seed with `--choices`, then `inspect --choices` to find the ordinal
   where the test body begins.
2. Run `explore --strategy=choice-exploration --choice-start-ordinal=N
   --max-choice-depth=N+K --max-executions=...`. Boot stays fixed in every forced
   prefix and the frontier is only the start-to-complete path. Start with K around
   64 and count distinct outcomes.

Also run plain seeds, `explore --count 200 --on-failure=budget`, over the test and
over the four smoke suites. That nightly baseline is the one the research says is
missing.

### Dimension 2: persistence faults, enumerated

Temporal's fault injection wraps every store method. The YAML form is rate-based
with its own seed, which keeps the fault stream independent of the scheduling
seed. The runtime `Injector` callback sees store, method and request on every
call, so "fail call k of method M with error E" is exact. `testcore` exposes it
through `WithFaultInjectionConfig`.

- Run once clean and count persistence calls per method. For start-and-complete
  that is tens, not thousands.
- For each (method, k, error) with error drawn from ShardOwnershipLost, timeout,
  conditional-failed and unavailable, run the test. Each cell is one campaign,
  driven by `--env` values the test reads.
- Require a witness that the fault fired. An unreached fault is "unreached", not
  "passed".
- Oracle per cell: the workflow still completes through retry or fails with a
  clean error, and the history has no duplicated or missing events.

Cross the two dimensions only for interesting cells, for example a fault on
`UpdateWorkflowExecution` during task completion combined with 64-deep schedule
exploration around it. The persistence seam is the one real fault surface in the
one-box cluster; everything else is in-process gRPC over loopback.

### Calibration

Reintroduce the update-admission regression on a scratch branch and record how
many seeds the matrix needs to catch it. If most seeds catch it, the loop works.
If none do, the problem is the decision space, and the wake-up yield work in the
vision note comes before anything else here. Either way it is the first row of
the known-bug table, and later rows are the six small fixtures the research lists:
atomic check-then-act, unlock/wake order, close versus send, cancel versus
complete, lock-order deadlock, equal-deadline tie.

Expect virtual-time test assumptions before server bugs. Check
`--clock-tick=forward` before blaming the server. A small driver that enumerates
the fault matrix and tabulates witnessed, passed, failed and unreached is the only
new code.

## 4. Give the target a voice: `Sometimes` and a catalog

One to two weeks. Filed as
[fn-191](../.flow/specs/fn-191-gomad-sometimes-assertions-and-a-site.md).
Steps 5 and 6 depend on it.

### softassert is already `Always`

`common/softassert` has `That`, `Fail`, `UnexpectedInternalErr` and
`UnexpectedDataLoss`, with 66 call sites in production code. Each failure logs
"failed assertion: " plus a static message with the failed-assertion tag. The
runner's `internal/evidence/record/softassert` observer scans stdout and stderr for
the prefix and tag, deduplicates messages, and treats a hit as a failure with
identity. Both backends consume it. So the oracle half of the Antithesis model
exists. The coverage half does not.

### The Antithesis pattern

Their SDK exposes `Always`, `Sometimes`, `Reachable`, `Unreachable` and numeric
guidance variants, keyed by a message. Their build-time instrumentor does two
jobs: edge-coverage rewriting, and a source scan that generates a registration
file emitting every assertion's id, message, type and location at init. The
catalog is what makes "this `Sometimes` never fired" reportable. Gomad already has
both halves of that catalog, unconnected: `common/gomaddebug` has the
declare-before-main registry with stable IDs and poison-on-late-registration, and
the native toolchain's `gomadintercept` pass shows the compiler binding a call
shape to a stable site identity.

### Design

**API.** Add `softassert.Sometimes(staticMessage)` and `Unreachable` as an alias of
`Fail`. `Sometimes` is a no-op in ordinary builds. Keep one vocabulary in one
package so authors do not learn a second API. Keep `Always` as an oracle and
`Sometimes` as a coverage obligation; mixing them is how assertions pass
vacuously. Numeric guidance variants can follow later.

**Catalog.** A `go:generate` scanner writes a checked-in file such as
`common/softassert/zz_catalog_gomad.go` under the `gomad && wasip1 && wasm` tag,
declaring every assertion site and every static OTEL span name with a stable ID,
package, file, line and message. `make -C tools/gomad validate` fails when it is
stale, matching how the persistence telemetry wrappers are generated with gowrap.
Non-literal messages are refused because they cannot be catalogued. The scanner
lives in the `gomadtool` command set under the WASM backend. `cmd/gomad` is a thin
launcher that selects a backend from the installation header and is not the place
for this. The alternative, injecting the catalog through the install-time runtime
overlay, is automatic but invisible to reviewers and ordinary builds; not chosen.
A compiler pass is not available on WASM, which builds guests with a stock Go
compiler plus a runtime source overlay.

**Channel.** `Sometimes` is evidence, not a replay diagnostic. The fn-180 probes in
`common/gomaddebug` are dormant in every recording and enabled only on replay, so a
`Sometimes` hit cannot ride on them. It needs the semantic-probe channel: a
first-hit bitset in the guest, flushed through one host import at exit, counted in
the artifact's coverage summary and corpus features, with the catalog hash bound
into target identity. Cost is one load, one branch and one store, identical in
recording and replay, which is what the compulsory runtime witness requires. This
channel is the kill switch for the spec: without it `Sometimes` is only a log
line.

**Limits.** The gomaddebug registry caps at 256 entries and 128-byte names. Sixty-
six assertions plus new sites plus 92 span names exceed that; raise the limits and
add a shared string table.

**Runner.** `--coverage=semantic` and `--require-probe` accept catalogued names on
WASM. A campaign summary lists catalogued-but-never-hit entries. `--must-hit`
fails the campaign. Hits feed guided-exploration features so the Guide prefers
executions that reach new ones.

### The spec

fn-191, "Gomad: Sometimes assertions and a site catalog on WASM". R-criteria:
`Sometimes` exists and is a no-op in ordinary builds; the generated catalog is
checked in, validated fresh and refuses non-literal messages; the guest channel
records first hits with no change to recording or replay evidence; campaigns
report never-hit entries and honor `--must-hit`; one Temporal `Sometimes` site is
demonstrated as reached under exploration and reported missing when its workload
is removed.

## 5. Reuse OTEL spans as the trace

Days, after step 4.

### Why

The server has 92 span starts and 289 attribute and event calls on exactly the
seams a hand-written probe catalog would target: the persistence telemetry
wrappers around every store method, the queue executable, the workflow context,
the update registry, the matching engine, the frontend Nexus handler. The tracer
provider is injected through fx and defaults to the OTEL noop provider. Call sites
only touch the OTEL API interfaces.

### Why not the OTEL SDK

The SDK batches, samples and exports on a background goroutine with its own
timers, which consumes scheduling decisions and virtual time and changes the tape.
Gomad already carries an exact adapter for `go.opentelemetry.io/otel/sdk@v1.44.0`
whose purpose is to keep it off the host. OTEL has no declaration step, no
invariant and no dump concept. What the SDK is fine for is retroactive
instrumentation on a retained failure during replay, read-only.

### Design

A provider swap under the build tag, not an SDK change. The implementation embeds
the noop tracer and span types and overrides `Start`, `SetAttributes`, `AddEvent`,
`RecordError` and `End` to write gomaddebug records: span start, span end, event,
error, with the span name as a string field because names are dynamic and
registration closes before `main`. Rules:

- Do identical work whether or not the host enabled collection; gate only the
  flush on `Enabled`, so allocations match in recording and replay.
- Never read the clock. The host attaches virtual time and tape position.
- Keep `telemetry.IsEnabled` false for this tracer. It is a type check for noop
  today, and when true the persistence wrappers JSON-serialize every request and
  response payload and call `time.Until` on deadlines. Change the check to an
  interface method so the gomad tracer opts out of the payload path.
- Assign span ids from a counter, deterministic on the single-threaded guest, and
  emit the parent id on start. gRPC propagation already crosses the loopback, so
  the host can rebuild one request's tree across frontend, history, matching and
  persistence, positioned on the choice tape.
- Fit the 16 KiB record limit and the raised registry limits.

Tracing always on in the gomad build changes allocation and, where a site reads
time, the schedule relative to a noop build. That is a declared profile, since
recording and replay share the binary, but failures found under it should be
confirmed under the plain profile before being called server bugs.

Gomaddebug stays for what OTEL cannot express: `RegisterDump` for frozen-state
snapshots (Temporal already registers `temporal.shards` and
`temporal.mutable_states`) and `RegisterInvariant` with a monotonic flag for
bisection. Spans say where the system is; dumps and invariants say whether its
state is right.

## 6. Put assertions on the schedule

Small, with step 4.

Today the runner never sees where a soft assertion fired. The zap logger attaches
`logging-call-at` with an absolute path and line from `runtime.Caller`, which works
on wasm through pclntab, but the observer keeps only the message. Three ways to
relate location, by channel:

- **`That` and `Fail` over the log channel.** Read `logging-call-at`, normalize
  against the module root in the prepared target (needs `-trimpath` or a known-root
  rewrite), and store a site next to each message. Keep identity keyed by message,
  since lines churn. On the failing path emit raw PCs from `runtime.Callers` and
  let the runner symbolize them from the retained module, as fn-180 does for
  stacks.
- **`Sometimes` over the catalog channel.** Location is static, captured by the
  generator. The guest sets a bit; the runner maps ID to site through the catalog
  bound into target identity and verifies the guest's catalog hash at startup. A
  changed catalog is an identity mismatch, not a misattribution.
- **Position on the schedule.** The host holds the choice ordinal and virtual time
  when a debug or semantic import arrives, so every hit through the import carries
  them. `inspect` can then name the two decisions an assertion fired between, and
  that ordinal is a ready stop point for stop-and-inspect or a checkpoint. Stdout
  is only approximately positioned inside the WASI transcript.

Practical shape: `Fail` keeps the log line for ordinary builds and additionally
emits ID plus PCs through the semantic import under the gomad tag; `Sometimes`
emits only through the import; the log scanner remains the fallback for artifacts
that predate the channel.

## 7. Checkpoints

Weeks, with a kill switch. Filed as
[fn-190](../.flow/specs/fn-190-gomad-wasm-checkpoint-restore-experiment.md).

### What exists

- Forced-prefix exploration keeps decisions 0 to k and branches at k, which is the
  checkpoint logically, but every candidate re-executes from process start.
- Stop-and-inspect from fn-180 runs an artifact to a chosen decision, validates the
  pending decision without applying it, parks every goroutine and every other
  cluster guest, runs dumps on a debug goroutine, then ends. The freeze half is
  built; the resume half is out of its scope.
- World has snapshot and restore. The helper's network, volume and namespace
  models export canonical inspection material but cannot start from it.

### Why the earlier verdict changed

The 2026-10-01 notes rated live snapshots a quarter-plus of platform-specific
work in tension with ASLR and GC determinism. That was the native backend, where
live state is a Go heap, native stacks, descriptors and signal state. Under WASM
the guest is linear memory, globals and tables with no host pointers; Go's wasm
port keeps goroutine stacks in linear memory and unwinds the native wasm stack to
its run loop on every goroutine switch. That is the property a memory-image
restore needs, inferred from the Go wasm design rather than tested here.

### What a checkpoint needs

1. Capture at an inspectable boundary: linear memory, globals, tables, fuel.
2. Capture host state at the same boundary: clock, tape cursor, entropy cursor,
   WASI descriptors for read-only mounts, helper model snapshots (restore is the
   gap fn-178 already notes).
3. Restore into a fresh instance from the image and re-enter the run loop, then
   continue with a different tape suffix or fault plan.
4. Keep evidence honest: a failure found from a checkpoint is retained only after
   a fresh replay reproduces it. The checkpoint is a search accelerator, never
   evidence.

The central risk is re-entry: at the boundary the guest is suspended inside a host
import with a live wasm stack that cannot be rebuilt for a fresh instance. The
experiment must show the fn-180 handoff can reach the stop with the native stack
unwound and deliver the pending decision through a reserved linear-memory slot on
resume. That is a dormant guest runtime change that must leave recording and replay
evidence unchanged.

### Payoff and experiment

Restoring a 100 to 200 MiB image should take tens of milliseconds against seconds
of boot, so a 128-candidate exploration drops from minutes to seconds and the
fault matrix of step 3 becomes interactive. fn-190 proves recorded-suffix and
fresh-suffix equivalence on a tiny fixture first, then measures the Temporal boot
cell on linux/arm64 per the 2026-10-14 performance host decision, then writes a
go/no-go report. If the fixture cannot resume, the spec stops and names the state
outside the image. If it can, a checkpoint-backed forced-prefix frontier follows as
its own spec.

## 8. Deferred, and why

- **Search policies** (delay bounds, POS, PCT, DPOR, learned scheduling) until the
  known-bug table exists to judge them and the decision space has no-op pruning
  and a wake-up yield. GOMAD_NEXT.md keeps the required evidence per policy.
- **Multi-node Temporal in the simulation.** The persistence and membership seams
  are the cost; the `gomad` build tag drops ringpop and no shared-store model
  exists.
- **Omes as a separate process and the mixed-brain upgrade test.** Native lane.
- **Edge-coverage instrumentation.** The choice trace is the richer signal and
  coverage-instrumented binaries are rejected by design.
- **Live snapshots on native, fork-and-continue, VM or CRIU snapshots.** Go is not
  fork-safe and the alternatives are Linux-only; superseded by fn-190 on WASM.
- **The race detector inside Gomad.** Stock `-race` stays a separate lane.
- **Umpire integration.** Owner decision 2026-10-16. The bridge, when revisited, is
  an Umpire Case as Gomad's per-execution input and the Judge as its oracle, with
  activity admission as the first joint experiment both plans already single out.

## 9. Decision context

Owner conversation on 2026-10-16. Questions asked and answered in order: how to
use Gomad to find bugs rather than qualify determinism; whether the mixed-brain
test or Omes can be the target; whether prior research on learning from other
tools exists (it does, in both repos); whether a small workload with systematic
fault and schedule permutations is possible now; whether `gomaddebug`-style hints
are needed or OTEL can carry them; whether checkpoints can eliminate boot;
whether OTEL can be swapped under a build tag; whether `softassert` should grow a
`Sometimes` with an Antithesis-style catalog; how the runner relates to an
assertion's location on WASM. The sections above are what the code supports today
plus the research already on disk. fn-190 and fn-191 were filed during the conversation.
