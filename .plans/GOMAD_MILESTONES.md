# Gomad v3: Milestones to a Deterministic Temporal Functional Test

**Plan date:** 2026-09-08 · **Consolidated:** 2026-09-30

## Purpose

This document is the delivery ladder for running Temporal's functional tests (`./tests`, built on
the `testcore` one-box cluster with in-memory SQLite and loopback gRPC) under Gomad v3 so that the
same seed produces the same run and a retained artifact replays byte-exactly.
[GOMAD_NEXT.md](GOMAD_NEXT.md) remains the capability roadmap across all four tracks.

Initial functional-test delivery is complete: milestones F0–F9 (removed on 2026-09-30; see revision `3e4303807`
of this file, which manifest findings citing `GOMAD_MILESTONES.md#f3-…` through `#f7-…` refer to),
the clock-tick spec `fn-103`, and the gap spec `fn-106` (see revision `3a7deb99d` for their
outcomes). The F10 items D1, D2, D13, D14, and D21–D25 are complete as well and were removed by 2026-10-01;
their outcomes are in the done summaries of the fn-105 tasks under `.flow/tasks/`. What remains
is the F10 backlog, the downstream implementation and qualification
in fn-107, feature-preserving code-size reduction in fn-108, deep modules and tool
interfaces in fn-109, runtime patch minimization in fn-110, the open findings, the
remaining test dispositions, and the rules that still apply. Vocabulary consolidation
and documentation acceptance are tracked in fn-111.

## Work tracking

Open work is tracked as flow-next specs under `.flow/specs/`; their tasks and acceptance criteria
(R-IDs) are authoritative, and this document keeps rationale and status. When a spec's scope
changes, change the spec and summarize the change here.

| Milestone | Spec | State |
| --- | --- | --- |
| F10 | `fn-105-gomad-follow-ups-deferred-scope` | required D7 CI job and D12 fix, the corrections proposed by the D16–D20 investigations, and deferred follow-ups ([decisions and scope](#f10-follow-ups-deferred-scope)) |
| Downstream cell | [fn-107](../.flow/specs/fn-107-gomad-finish-downstream-cell.md) | open; consumer implementation and exact replay on both platforms, including D8/D9/D10 |
| Code-size cleanup | [fn-108](../.flow/specs/fn-108-gomad-reduce-code-size-without-removing.md) | implemented and verified on darwin/arm64: production Go down 286 code lines, D1/D2 delivered; R9 stays incomplete until the linux/amd64 gates run ([status](#code-size-cleanup-fn-108)) |
| Deep modules and tool interfaces | [fn-109](../.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md) | open; all sixteen architecture findings, reusing D1/D2 and reviving D3/D4/D5 ([delivery order](#deep-modules-and-tool-interfaces-fn-109)) |
| Runtime patch minimization | [fn-110](../.flow/specs/fn-110-gomad-minimize-the-runtime-patch.md) | open; overlay extraction and canonical one-context-line regeneration, with both-platform qualification ([delivery order](#runtime-patch-minimization-fn-110)) |
| Vocabulary and documentation | [fn-111](../.flow/specs/fn-111-gomad-consolidate-vocabulary-and-update.md) | open; glossary merged into SPEC, guides reconciled with parsers and source, and acceptance evidence retained; spec completion review pending ([scope](#vocabulary-and-documentation-fn-111)) |

Work a spec with `/flow-next:work <spec>`; list what is ready with `flowctl ready`.

## Remaining `./tests` dispositions

Every `./tests` test has a disposition in the generated manifest
(`tools/gomad3integration/qualification/tests.json`, from `tests.generator.json`); none is
excluded. What still keeps the package short of "every functional test replays exactly":

- **Routine qualification is untraced (F10 D13).** The generated manifest defaults to no choice
  trace and no success replay, so a `qualified` workload there establishes same-seed
  repeatability: two fresh repetitions per seed with equal evidence. Its report makes no
  choice-tape replay claim (`replayed` and `choice.available` are false), and a mismatch between
  repetitions is still `nondeterministic`. Tracing and success replay stay on in the replay
  gates (`temporal.json`, `smoke.json`, and `qualification/core.json`) and for the one test the
  generator spec opts in, `TestSignalWorkflowTestSuiteChasm`, which carried the darwin replay
  divergence fixed under D14 and stays `intermittent` on linux under D12. A seed there verifies
  choice-tape replay when it reports `choice_replay_exact`; the `intermittent` expectations of
  D12 still accept a seed that diverged. The exact-replay results recorded below for individual
  suites are the dated traced runs they name, not a property of routine runs.
- **No exact replay available for eight suites.** Their choice traces overflow the 64 MiB
  maximum, so they cannot opt into tracing until larger traces exist (deferred as D15).
  `TestTaskQueueStats_Pri_Suite`, `TestVersioning3FunctionalSuite`,
  `TestVersioning3QueryFunctionalSuite`, and `TestWorkerDeploymentSuite` also run with a raised
  I/O transcript; `TestClientMiscTestSuite`, `TestScheduleV1`, and
  `TestScheduleV1WorkflowPauseInteraction` run with the default transcript.
  `TestVersioningFunctionalSuite` is expected `qualified` on the same seed-repeatability basis.
- **Skips with an identified cause.**
  - `TestStandaloneActivityTestSuite/TestStartDelay/UpdateWhilePaused_AfterWindow_ExtendsDispatch`:
    under `clock_tick: forward` the lead of `time.Now` over the timer clock grows with every read
    (about 0.33 s when this subtest polls), and each deadline re-derived from `time.Now` fires
    later by that lead. The frontend hop, the matching hop, and matching's child context add
    three leads, which use up the 1 s empty-response budget, so the client's 3 s deadline fires
    first (seeds 5, 12, 20, 21, 24 of 1 to 24 on toolchain `8d28bd44`; seed 11 now passes with
    about 20 ms to spare). The
    [D16](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.16.md) investigation
    ([report](../docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md)) assigns the
    correction to the Gomad forward clock. The correction is proposed and not implemented, and
    the skip stays.
  - `TestNexusOTELSuite/TestOperation`: the test needs two dedicated cluster slots and the pool
    has one under Gomad. `TEMPORAL_TEST_DEDICATED_CLUSTERS=2` is recorded but never reaches the
    test, because the patched runtime shows Go code an environment of `TZ=UTC` only. The test
    waits on itself with no virtual deadline until the wall watchdog kills the target, and the
    host-timed kill is what was recorded as nondeterminism. With the pool forced to two slots in
    a throwaway copy the suite qualifies on seeds 11 and 17 with exact choice-tape replay. The
    [D17](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.17.md) investigation
    ([report](../docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md)) assigns the
    correction to the Gomad runtime patch and Runner. The correction is proposed and not
    implemented, and the skip stays.
  - `TestWorkerCommandsTaskSuite/TestDispatchCancelOnWorkflowTimeout`: under the strict tick the
    activity's capped deadline equals the workflow run expiration, and when the activity's timer
    task runs first the server never creates the cancel command (9 of 16 seeds; no timeout budget
    is involved). [D18](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.18.md) identified the
    cause in its [report](../docs/research/gomad/GOMAD_D18_WORKER_CANCEL_DELIVERY.md) and proposes
    `clock_tick: forward` for the suite; the correction is undecided and the skip stays.
  - `TestFairness{,AutoEnable}Suite/Test_Activity_Basic`: the test measures dispatch fairness without
    waiting for its 225 activity tasks to reach matching. In the whole suite the shard transfer
    readers exhaust their 20-token rate-limiter burst in the first virtual instant, the next read
    waits on a 50 ms timer, and the test drains a partial backlog first (7 of 17 seeds fail under
    either tick; with a complete backlog the metric is the ideal 0.45 in every run). The auto-enable
    suite also fails in `triggerAutoEnable` when the queue reload ends its single poll empty (3 of 17
    seeds under the strict tick). [D19](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.19.md)
    identified both causes in its
    [report](../docs/research/gomad/GOMAD_D19_FAIRNESS_BACKLOG_READINESS.md) and proposes a test
    precondition and a repeated trigger poll; no matcher fairness defect was found, the correction is
    undecided, and both skips stay.
  - `TestWorkflowTaskTestSuite/TestWorkflowTaskHeartbeatingWithEmptyResult`: the test sends a
    heartbeat every second against a 5 s heartbeat timeout, and the server rejects a heartbeat only
    strictly after the deadline. Natively the heartbeat due at 5 s is 10 to 92 ms late and is
    rejected; under the strict tick it arrives at exactly 5 s and is accepted, so the test counts
    one rejection instead of two on every seed tried. No server or runtime defect was found.
    [D20](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.20.md) identified the cause in its
    [report](../docs/research/gomad/GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md) and proposes a test
    correction that sends the two rejected heartbeats explicitly past the deadline; the correction
    is undecided and the skip stays.
- **Intermittent suites.** The F5/F6 suites and `TestSignalWorkflowTestSuiteChasm` on linux
  (F10 D12). The darwin divergence of `TestSignalWorkflowTestSuiteChasm` is fixed (F10 D14,
  2026-09-30): the generated manifest expects it `qualified` on darwin/arm64 and `intermittent`
  on linux/amd64. The linux `intermittent` expectation for the F5/F6 suites is
  stated in the representative and smoke manifests (`temporal.json`, `smoke.json`); the generated
  manifest still expects those suites `qualified` on linux, where the full set is not run as a gate.

## Open findings

- **Linux replay divergence (F10 D12).** About one tier-3 seed-run in 26 on linux/amd64 is
  nondeterministic or diverges on replay, on either seed and a different suite each run, at choice
  ordinals from 8 to ~85k. A fork bisect showed it is not caused by the FIPS DRBG
  draw (`6bc11ef7d`) or the mark-start greying (`440552d2c`): runs with either reverted still
  diverged. A Rosetta linux/amd64 container reproduced it once in 32 seed-runs and never in 100
  sequential replays of the same artifact, so it depends on host timing under load. The F5 and F6
  suites are `intermittent` on linux, and both the dispatch-only linux gate and the required smoke
  gate accept `nondeterministic` and `replay_divergence` for them while failing on target
  failures, unsupported targets, and infrastructure errors. The darwin representative set stays
  fully qualified.
- **Host-clock escapes** recorded by the static inventory (`toolchain/clock_inventory_test.go`):
  `gcMarkTermination` stamps `MemStats.LastGC` and `MemStats.PauseEnd` with host wall time (also
  visible through `debug.GCStats`, the text heap profile, `expvar`, and the Prometheus
  `go_memstats_last_gc_time_seconds` gauge), the FIPS entropy source's `monoTime` (host time on
  linux/amd64 only), and the execution tracer's clock snapshot; on linux also
  `syscall.Gettimeofday` behind the `syscall` pack gate. Investigated, not fixed
  ([D21 report](../docs/research/gomad/GOMAD_HOST_CLOCK_ESCAPES.md), darwin/arm64 2026-09-30;
  linux/amd64 by source reasoning only): none of these is read by runtime control flow; each
  hands a host value to the target as reporting state, trace output, FIPS entropy input, or the
  result of a pack-gated call, so a target that prints or branches on one is not repeatable. A
  fixture that prints `LastGC` is `nondeterministic` at stdout while its choice tape replays
  exactly. By source inspection the Temporal functional closure links the readers but neither
  emits nor consumes the values, and these four escapes are not a cause of D12 or D14. Proposed
  for a subsequent decision: state the limitation in the contract, pin the unpinned darwin
  `gettimeofday` path and `cputicks` (on linux/amd64 host cycle counts steer block and mutex
  profile sampling when a rate is set and reach text profiles and sampled protobuf profiles; not
  shown unrelated to D12), and overwrite the stamps with the stored virtual time from `runtime/proc.go`. That
  overwrite changes no prohibited file, leaves the host read in place, and writes
  collector-owned state, so it needs the patch-policy owner's judgement, as does guarding the
  read in `mgc.go` or the linux assembly. The collector patch prohibition remains in force.
- **DTrace clock audit** on darwin needs a root run; CI supplies it on the macOS runner.
- **Downstream cell.** Gomad-side work for a module that embeds the server is complete; its
  in-process cluster test classifies as a capability blocker until the downstream cuts its own
  seams and injects its filesystem and membership transport ([GOMAD_CLOUD.md](GOMAD_CLOUD.md)).

## Constraints

- **No policy widening.** Gomad never grants `syscall`, `os/exec`, `os/signal`, or
  `golang.org/x/sys` generically. Every exception is an exact compatibility pack bound to a
  module version, go.sum hash, per-file SHA-256, owner, and workload, reviewed under the
  existing `discover`, `review`, `generate --approve-review`, `check`, `qualify` flow.
- **No source translation and no test rewriting.** Determinism comes from the patched
  toolchain and the reviewed boundary. A Temporal test that needs a Gomad-specific overlay of
  its own source, as gomad1 required, is a blocker to record, never a fix to ship.
- **Fail-closed stays.** An unmodeled boundary operation terminates the process. Work that needs
  a new modeled operation adds it with a semantic contract, a resource bound, transcript
  coverage, exact replay, and a negative test, per COMPAT-5 in
  [the compatibility roadmap](GOMAD_NEXT.md#compat-5-targeted-deterministic-adapters-and-io-models).
- **Evidence over narration.** Work is done when its command produces the stated report on a
  clean checkout. A passing local run that depends on untracked state does not count.
- **Platform.** The boundary manifest qualifies `darwin/arm64` and `linux/amd64`. Each platform
  is its own qualification and artifacts replay only where they were produced. The macOS sandbox
  test and the DTrace clock audit remain `darwin/arm64` only. Each platform's compatibility packs
  are its own; `compatibility-pack-qualification` qualifies the requests that name the host.
- **Server source changes are allowed but bounded.** A change under `common`, `service`,
  `temporal`, or `tests/testcore` is acceptable when it isolates an optional provider behind a
  build tag or an injection seam and the default build is unchanged. A change that alters
  runtime behavior for production builds requires dedicated production review and
  regression evidence.
- **Validation scope.** The full `./tests` set is not run as a gate; a change is validated on the
  smoke selection plus the suites it affects, with `make gomad3-tests-qualification` as the
  on-demand local run.

## F10: follow-ups (deferred scope)

**Spec.** [fn-105-gomad-follow-ups-deferred-scope](../.flow/specs/fn-105-gomad-follow-ups-deferred-scope.md)
owns tasks and acceptance criteria.

**Outcome.** Fix work explicitly selected during the issue walkthrough must meet its
acceptance criteria. Other items retain their original deferral reasons and revival
triggers from the 2026-09-29 scope cut. A trigger must be recorded here before a
deferred item is implemented; deferred items may remain open or close as won't-do.

**Status.** Sixteen open items. D7 is required CI work: its darwin job is written and its
commands pass locally on darwin/arm64, and it stays open until a GitHub Actions run passes.
D12 is a required fix that needs a linux/amd64 host; the D14 lock-profile fix is an unverified
candidate for it. D16-D20 are investigated; each row records a proposed correction that is not
applied and awaits a decision, and the skips stay until a correction is verified.
D3-D5 and D8-D10 are delivered through fn-109 and fn-107. D6 clock policies and D15
larger traces remain explicitly deferred, and D11 dynamic Linux clock auditing stays deferred
on the D21 findings.

### Required work

| Item | Scope and origin | Decision |
| --- | --- | --- |
| D7 | Add a darwin/arm64 job for the existing functional smoke selection (`fn-101.4`) | Required CI addition approved 2026-09-30. R7 in fn-105 requires a standard macOS runner, preserved Linux coverage, platform-specific reports/artifacts, zero unsupported/failed/infrastructure errors, explicitly traced exact replay, and passing GitHub Actions evidence. |
| D12 | Identify and fix the linux/amd64 replay-divergence channel (`fn-106.1`); about one tier-3 seed-run in 26 diverged, and the recorded evidence points to host timing under load | Must be fixed, decided 2026-09-30. Native Linux instrumentation is an execution prerequisite. R12 in fn-105 requires a regression reproducer, repeated exact replay on both seeds under load, and restoration of strict CI expectations; diagnosis alone cannot close the task. |
| D16 | Investigate TestStandaloneActivityTestSuite/TestStartDelay/UpdateWhilePaused_AfterWindow_ExtendsDispatch; the client times out before an empty poll response under the forward clock | Investigated on darwin/arm64 (2026-10-01, toolchain key `8d28bd44`); nothing is fixed and the skip stays. Cause: the forward offset is cumulative and unbounded (`gomadTimeNow` adds 1 to 1024 ns per read and never subtracts), so `time.Now` led the timer clock by about 0.33 s when the subtest polled. `context.WithTimeout` arms its timer for `time.Until(time.Now()+d)`, which is `d` plus the lead, and each gRPC hop re-derives the deadline from `grpc-timeout` and adds the lead again. The frontend hop, the matching hop, and matching's `WithDeadlineBuffer` child add three leads; above 1 s in total the client deadline fires before the empty response. On the unmodified tree seeds 5, 12, 20, 21, 24 of 1 to 24 fail with the same error, and a rerun of each reproduced its stdout and stderr digests; seeds 11 and 17 pass (seed 11 by about 20 ms). Under `strict` the subtest passes on all seven of those seeds with the child deadline 1 s before the client's, and natively 999.8 ms before. A standard-library fixture reproduces the timeout on seeds 11 and 17 after 700,000 reads. gRPC, the 1 s budget, and the test behave as they do natively; the owner is the Gomad runtime clock. Proposed for a decision, not built or verified: add the draws to the virtual clock itself so `time.Now` and timers share one clock, as fn-103 specified; this first needs the simulation time transport failure that made fn-103 move to a separate offset reproduced and resolved, and its feasibility is not established. Partial fallbacks with stated residuals (offset-aware `time.Since`/`time.Until`; absorbing the offset on idle advances) and an accepted-limitation alternative are in the report. The correction, its conformance fixture, removal of the skip, and requalification of the seven forward suites remain open work in fn-105. linux/amd64 was not run. Report: `docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md`; evidence: `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d16-evidence.json`. |
| D17 | Investigate TestNexusOTELSuite/TestOperation with two clusters; the two-slot pool setting never reached the test, which still waits on itself | Investigated on darwin/arm64 (2026-10-01, toolchain key `8d28bd44`, which contains the D14 fix); nothing is fixed and the skip stays. Cause: the Runner records a supplied `--env` entry and passes it to the process, and the patched `syscall.copyenv` then replaces the environment Go code reads with `TZ=UTC` in deterministic mode, so `TEMPORAL_TEST_DEDICATED_CLUSTERS=2` has no effect and the dedicated pool keeps the one slot it derives from `GOMAXPROCS`. The test takes that slot and waits for a second on a channel receive with no deadline (`test_cluster_pool.go:109`, from `nexus_otel_test.go:143`), the held cluster's timers keep virtual time advancing, and the wall watchdog kills the target at a host-timed virtual instant. The first evidence field that differs between repetitions is then `virtual_time_elapsed_nanos`, and the workload is reported `nondeterministic`. A killed target writes no terminal choice frame, so its artifact is replayed without tracing, and the seed-11 replay's stderr differs from the traced recording in one `%p` address; a fresh untraced run equals the replay. No evidence of a cause shared with D14 was found, and its fix does not change the outcome: stderr is byte-identical across repetitions up to the kill, and with a virtual `-test.timeout` the failing run repeats byte for byte. The killed runs keep no Choice Trace, and D12 is unassessed because no linux run exists. With the pool forced to two slots in a throwaway copy the suite with the skip lifted is `qualified` on seeds 11 and 17 with exact choice-tape replay (12 executions) and passes on seeds 1 to 17; natively it passes with eight and with two slots and waits the same way with one. Proposed for a decision, not built or verified: deliver supplied environment entries to the target (Gomad runtime patch and Runner), or reject the flag and size the pool under the `gomad` build tag in `tests/testcore`. The correction, its conformance fixture, and removal of the skip remain open work in fn-105. linux/amd64 was not run. Report: `docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md`; evidence: `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d17-evidence.json`. |
| D18 | Investigate worker cancellation delivery and inconsistent test timeout budgets | Investigated 2026-09-30 on darwin/arm64 ([report](../docs/research/gomad/GOMAD_D18_WORKER_CANCEL_DELIVERY.md)). Cause: a strict-tick timestamp tie between the workflow run-timeout timer and the activity's capped ScheduleToClose timer; when the activity timer task runs first, no cancel command is generated, and no test budget or delivery step is at fault. Proposed correction, owned by the Gomad qualification configuration: `clock_tick: forward` for the suite, which qualified seeds 11 and 17 with the skip lifted in scratch manifests. The correction is open work pending a decision; nothing is fixed and the skip stays until verified. |
| D19 | Investigate activity fairness backlog readiness in both fairness suites | Investigated 2026-10-01 ([report](../docs/research/gomad/GOMAD_D19_FAIRNESS_BACKLOG_READINESS.md)); nothing is fixed and both skips stay. Cause: setup bias. The test polls before its 225 activity tasks reach matching; under Gomad the shard transfer readers' rate limiter (20 per second, burst 20) cannot refill while the test keeps the process busy, so a partial backlog is drained first. A complete backlog gives the ideal metric in every run, and no matcher fairness defect was found. The auto-enable suite also fails when its trigger poll returns empty during the queue reload. Proposed correction, owned by the test's owner: wait for the full backlog through the existing `DescribeTaskQueuePartition` count and repeat the trigger poll; it passes seeds 1 to 17, `qualify-set` on seeds 11 and 17 with the skips lifted, and natively. The correction is undecided open work under R19. |
| D20 | Investigate heartbeat timeout counting under virtual time | Investigated 2026-10-01 on darwin/arm64 ([report](../docs/research/gomad/GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md)); nothing is fixed and the skip stays. Cause: the test's timing arithmetic. Five 1 s sleeps put a heartbeat exactly on the 5 s heartbeat deadline of each chain, and the server rejects only strictly after it (`Now().After(OriginalScheduledTime + timeout)`). Natively both expected rejections rest on 10 to 92 ms of round-trip latency and sleep overshoot (a native run with 995 ms sleeps fails 2 of 3 times); under the strict tick the heartbeat is accepted at exactly 5 s, the rejection comes one heartbeat later, and the loop ends with one rejection on every leaf seed tried (1 to 17). Under `forward` the test passes on the tick offset, 1.4 ms at the smallest. No server or runtime defect was found. Proposed correction, owned by the test's owner: send the two heartbeats that must be rejected 100 ms after a deadline the test computes and assert every heartbeat's outcome, which removes the need for latency in the rejections and keeps the existing latency ceiling of about 0.93 s for the accepted heartbeats; in a throwaway copy it passes strict leaf seeds 1 to 64, `qualify-set` on seeds 11 and 17 with the skip lifted under both ticks with exact replay, and natively, with the expected history unchanged. Alternatives: `clock_tick: forward` for the suite (Gomad qualification configuration) and an inclusive server comparison (history service, production review). The correction is undecided open work under R20. linux/amd64 was not run. |

### Architecture consolidation

| Item | Scope and origin | Revival trigger |
| --- | --- | --- |
| D3 | Private executor injection instead of public `Executor`/`ReplayExecutor` (`fn-102` R4); a Go API change | Revived by the 2026-09-30 architecture request; fn-109 R5 owns delivery |
| D4 | Architecture fitness checks for ownership, host effects, and public signatures, with negative fixtures (`fn-102` R5) | Revived by the 2026-09-30 architecture request; fn-109 R8 owns delivery |
| D5 | Architecture/platform/determinism documentation reconciliation (`fn-102` R6) | Revived by the 2026-09-30 architecture request; fn-109 R9 owns delivery |

These were deferred as maintenance without a waiting consumer or behavior change.
The [architecture assessment](../.flow/artifacts/fn-102-gomad-architecture-consolidate/architecture-assessment.md)
retains the evidence. Preserve CLI behavior, schemas, canonical bytes, failure
classification and precedence, and replay compatibility when reviving them.

The 2026-09-30 request for feature-preserving code-size reduction revives D1/D2
under [fn-108](../.flow/specs/fn-108-gomad-reduce-code-size-without-removing.md).
Reuse or transfer the existing obligations during task breakdown. The subsequent
request to address every architecture finding creates
[fn-109](../.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md),
which reuses fn-108 for D1/D2 and revives D3/D4/D5 through R5/R8/R9. The current
coordinator option loss and reviewed interface/ownership gaps supply the revival
evidence. Public migrations remain outside fn-108 and follow its shared policy
extraction where they overlap. Authoring these specs starts no tasks and claims
no implementation.

### Capability and CI follow-ups

| Item | Scope and origin | Why deferred | Revival trigger |
| --- | --- | --- | --- |
| D6 | `seeded` and `fixed=<d>` clock ticks, manifest settings and qualified fixtures (`fn-103`) | Explicitly deferred 2026-09-30 under fn-105 R6; `forward` addresses known ties and the extra policies are exploration features | A specific bug class needs deliberate ties or constant quanta |
| D8 | Downstream closure-mode adapter for the signal-handling metrics library (`fn-104` C3/R2) | Revived by fn-107 R7/R9 under fn-105 R8 | Final downstream target and closure evidence as specified by fn-107 |
| D9 | linux/amd64 downstream packs and qualification (`fn-104`) | Revived by fn-107 R8/R10 under fn-105 R9 | D8 and the final downstream target; qualify both actual hosts |
| D10 | Downstream seam guide (`fn-104` R4) | Revived by fn-107 R12 under fn-105 R10 | D9's retained qualification and pack evidence |
| D11 | Dynamic Linux clock audit with disabled vDSO, seccomp denial, and positive control (`fn-101.3`, pre-amendment R5) | Stays deferred: D21 (2026-09-30) found no evidence that requires it, and it is not implemented. The static inventory remains the linux escape gate | A selected fix removes or guards a linux clock read (the `proc.go` overwrite does not), D12 attributes a divergent event to a host-clock read, an upgrade adds an unclassifiable reference, or a pack admits a raw clock syscall. Feasible scope is in the [D21 report](../docs/research/gomad/GOMAD_HOST_CLOCK_ESCAPES.md#d11-determination) |
| D15 | Larger choice traces (`fn-106.3`, split from D13 on 2026-09-30) | Routine qualification uses seed repeatability; the opt-in policy does not require larger tapes | A named workload needs a retained decision tape beyond 64 MiB for debugging, replay verification, exploration, or minimization |

The [constraints](#constraints) apply throughout.
New tick policies carry execution identity and the
[COMPAT-5 evidence set](GOMAD_NEXT.md#compat-5-targeted-deterministic-adapters-and-io-models).
Downstream packs/adapters bind exact versions; dependency drift reopens their
qualification. On revival, use the origin spec's requirement text as acceptance.

## Code-size cleanup (fn-108)

**Spec.** [fn-108](../.flow/specs/fn-108-gomad-reduce-code-size-without-removing.md)
owns the requirements. Its seven tasks are done; the delivery order and preservation contract
were removed from this document on 2026-10-01 and remain in the spec. Measurements, the comment
audit, and gate reports are in
[final.md](../.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final.md).

**Status.** Implemented on 2026-10-01 in the working tree; nothing is committed. Authored
production Go across `tools/gomad3`, `tools/gomad3sim`, and `tools/gomad3integration` fell from
58624 to 58338 code lines (-286) and by 10691 code bytes under one counting script, with the two
new owner files `runner/completion.go` and `runner/retention.go` counted. The runtime overlay,
generated files, the patch, schemas, and templates are unchanged; tests grew by 1929 code lines.
Public `go doc` and CLI help captures equal the baseline, and compatibility packs, qualification
manifests, and schemas are untouched. Six comment lines were removed: three copies of one comment
whose code was merged, and the comment remains above the merged code. On darwin/arm64,
`make -C tools/gomad3 validate` and `test`, the `gomad3sim` and integration tests, the smoke set
(4/4), the core set (7/7), and the representative Temporal set (28/28 on seeds 11 and 17, 56
exact replays) passed, and all 1140 baseline tests keep their result. R9 is incomplete: no
linux/amd64 gate ran because no host was available. `make lint-code-fast` exits 2 as it did
before fn-108 and cannot analyse the nested module. Minimization has no before-and-after
projection and rests on its existing tests. Evidence and the linux commands are in
[final.md](../.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final.md).

## Deep modules and tool interfaces (fn-109)

**Spec.** [fn-109](../.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md)
owns all eleven ranked findings and five secondary opportunities from the
[architecture assessment](../.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/architecture-assessment.md).
Its twenty acceptance criteria cover campaign options, common execution policy,
complete preparation, public construction, protocols, simulation progress,
backend handles, Artifact lifetime, installation knowledge, controller completion
and capability ownership.

**Outcome.** Smaller, testable interfaces hide sequencing, validation and resource
lifetime from callers while preserving shipped capabilities and replay contracts.
The selected simulation and handle changes require implementation and conformance
evidence; design notes or renewed deferral alone cannot close them. Re-anchor
findings and reuse verified work that lands after the assessment.

### Delivery order

| Stage | Scope | Acceptance in fn-109 |
| --- | --- | --- |
| 1. Coordinator parity | Correct lost simulation bounds and characterize local/isolated options, defaults and failure behavior | R1; exercise every strategy through the actual coordinator seam |
| 2. Common policy | Reuse fn-108's assessment and retention extraction; give the pure seed controller one completion transition | R2, R3, R16; reuse or transfer D1/D2 obligations once, with fn-108 R6/R7 evidence |
| 3. Operation and preparation ownership | Narrow request/private construction, complete preparation, target commands, installation descriptions and capability evaluation | R4–R6, R10, R15, R17; overlapping public migrations follow verified fn-108 extraction |
| 4. Resources and protocols | Separate Artifact references/opened handles, generate simulation-time consumers and hide model-wire slots behind typed commands | R7, R13, R14; preserve ownership, bytes, validation and domain errors |
| 5. Simulation interfaces | Compare progress interfaces, implement the lifecycle owner and select backend-specific network/filesystem handles at creation | R11, R12; preserve concurrent blocking work, stale-incarnation rejection and backend fidelity |
| 6. Architecture and qualification | Enforce ownership/effect/signature checks, reconcile guidance and retain complete finding/evidence coverage | R8, R9, R18–R20; negative fixtures, fixed-identity equivalence and both-platform gates |

D3/D4/D5 are revived through R5/R8/R9. Task breakdown must reuse or transfer
their existing obligations rather than create duplicate owners. Coordinate
shared runtime-overlay and source-inventory edits with fn-110; patch minimization
retains its own scope and qualification requirements.

**Preservation contract.** Keep CLI grammar/defaults, recorded formats, canonical
bytes for fixed identities, error precedence, existing comments, fresh processes,
native timer ownership and separate seed/round/corpus transactions. Inventory
and migrate the intended executor-injection and Artifact-handle Go interface
changes. D12/D14 retain separate ownership; unavailable hosts or unexplained
regressions leave affected acceptance incomplete.

**Status.** Open spec; tasks are broken down and implementation has not started.

## Runtime patch minimization (fn-110)

**Spec.** [fn-110](../.flow/specs/fn-110-gomad-minimize-the-runtime-patch.md)
owns both approaches from the
[patch-size investigation](../docs/research/gomad/GOMAD_PATCH_SIZE.md).
Move existing Gomad-only implementation into additive overlays and emit the canonical
Go 1.27.1 patch with one context line. This work reduces the upstream patch; fn-108
separately owns net production-code reduction.

**Outcome.** A smaller upstream source diff and a smaller canonical patch, with every
supported behavior, existing comment, boundary guard, and qualification assertion
preserved. The measured combined prototype was 20,352 bytes and 581 lines versus the
32,275-byte, 998-line baseline, a 36.9% byte reduction. This is guidance, not a quota
or completed qualification; moved overlay code remains part of the reported cost.

### Delivery order

| Stage | Scope | Acceptance in fn-110 |
| --- | --- | --- |
| 1. Baseline | Record pinned inputs, source sets, patch and overlay counts, and current workload dispositions | R1; comparable measurements and qualified inputs |
| 2. Overlay extraction | Move the three scheduler/quiescence implementations, crypto initialization, and syscall linkname declarations; retain upstream hooks and lifecycle points | R2, R3, R6; preserve lock state, ordering, entropy overrides, activation, and negative behavior |
| 3. Canonical regeneration | Emit `-U1`, update exact descriptor source sets and generated consumers, and make the pinned regeneration check follow the descriptor | R4, R5; repeated byte-identical regeneration and zero-fuzz source equivalence between final `-U3` and `-U1` patches on both hosts |
| 4. Qualification and final evidence | Run full Gomad gates on both platforms, affected entropy/process-simulation tests, Temporal integration/smoke and affected qualification; publish final counts and build identities | R7, R8; fresh repeatability and exact replay under existing contracts, with no weakened dispositions |

**Preservation contract.** Keep GC stabilization, host/target random streams, arrival
ordering, timer-presence detection, synctest, caller-site and goroutine identities,
traceback normalization, panic/testing completion, disabled behavior, and resource
bounds. Existing artifacts retain their original toolchain identity; the final candidate
records fresh artifacts under its new identity. Missing native-host evidence leaves
acceptance incomplete.

Goroutine-state embedding, compiler/linker hook consolidation, early environment
initialization, and interception redesign remain outside this spec. D12/D14 fixes
retain their separate owners; patch minimization cannot change their dispositions
to obtain a passing gate.

**Status.** Open spec; tasks are broken down and implementation has not started.

## Vocabulary and documentation (fn-111)

**Spec.** [fn-111](../.flow/specs/fn-111-gomad-consolidate-vocabulary-and-update.md)
captures the vocabulary update plan and its assessment of all 25 original glossary
terms. SPEC becomes the canonical vocabulary source; GLOSSARY.md is deleted.
CLI, ARCHITECTURE, TUTORIAL, and README use those definitions and describe
implemented behavior without changing APIs, flags, defaults, schemas, or replay identities.

**Outcome.** One vocabulary, working navigation, and consistent distinctions between
Target and Prepared Target, Choice Trace and Decision Tape, Backend and Fidelity,
seed repeatability and exact replay, and platform support and workload qualification.
The 24 current glossary concepts survive as definitions or aliases; Parity Case
remains historical README material.

| Stage | Scope | Acceptance in fn-111 |
| --- | --- | --- |
| 1. Vocabulary consolidation | Retain per-term assessments, merge definitions and aliases into SPEC, delete GLOSSARY, and preserve existing requirement identifiers | R1–R3 |
| 2. Guide reconciliation | Check architecture, CLI examples/defaults, and tutorial against implemented platforms, clocks, replay, exploration, and simulation guarantees | R4–R6 |
| 3. Navigation and acceptance | Repair active references, validate local links/fences, compare identifiers and command/flag inventories, and retain scoped evidence | R7, R8 |

Reuse verified documentation evidence for fn-109 R9 and fn-105 D5; the broader
architecture work keeps its existing owner and future interface-documentation
obligations. D12/D14 replay fixes and D15 trace capacity retain
their separate owners and acceptance. Documentation reconciliation cannot close
those items or widen qualification claims.

**Status.** Open spec; `flowctl show` is authoritative for its two tasks. Vocabulary and
guide reconciliation evidence is retained under
`.flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update/` and is bound to
file hashes of an uncommitted working tree, so it must be regenerated after the
edits are committed. Spec completion review remains pending.

## Out of scope

- New fault injection, partition, crash-restart, and multi-node capabilities through
  `tools/gomad3sim`. Those are the simulation track and cannot host `testcore`.
- Multi-P scheduling, deterministic GC, DPOR, and preemption bounding. These are BUG-7 research
  items.
- Revival of gomad1 or gomad2. [GOMAD_CMP.md](GOMAD_CMP.md) records why.

## Open risks

- **GC timing** is not controlled and shares the seeded runtime stream. Allocation-heavy suites
  may diverge between repetitions; the open linux and darwin replay findings may be such cases.
- **Spin loops** anywhere in the cluster stall virtual time. Pollers with backoff are fine, but a
  single `for {}` with a non-blocking select is fatal under this runtime.
- **Upstream Go releases** invalidate the patch and the boundary manifest each time; every Go bump
  repeats the port and requalification.
- **Compatibility packs pin exact module versions.** Every dependency bump that touches a packed
  or adapted module invalidates the pack or adapter and reopens the capability closure.
