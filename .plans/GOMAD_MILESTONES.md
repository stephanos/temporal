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
outcomes). The F10 items D13 and D21–D25 are complete as well and were removed on 2026-09-30;
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
| F10 | `fn-105-gomad-follow-ups-deferred-scope` | required D7 CI job and D12/D14 fixes, the D16–D20 investigations, and deferred follow-ups ([decisions and scope](#f10-follow-ups-deferred-scope)) |
| Downstream cell | [fn-107](../.flow/specs/fn-107-gomad-finish-downstream-cell.md) | open; consumer implementation and exact replay on both platforms, including D8/D9/D10 |
| Code-size cleanup | [fn-108](../.flow/specs/fn-108-gomad-reduce-code-size-without-removing.md) | open; local cleanup, then D1/D2 consolidation and equivalence checks ([delivery order](#code-size-cleanup-fn-108)) |
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
  - `TestNexusOTELSuite/TestOperation`: two dedicated clusters in one process are not yet
    deterministic. Investigation is approved as
    [D17](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.17.md); the pool-capacity
    wait is already resolved, and the remaining cause is unidentified.
  - `TestWorkerCommandsTaskSuite/TestDispatchCancelOnWorkflowTimeout`: under the strict tick the
    activity's capped deadline equals the workflow run expiration, and when the activity's timer
    task runs first the server never creates the cancel command (9 of 16 seeds; no timeout budget
    is involved). [D18](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.18.md) identified the
    cause in its [report](../docs/research/gomad/GOMAD_D18_WORKER_CANCEL_DELIVERY.md) and proposes
    `clock_tick: forward` for the suite; the correction is undecided and the skip stays.
  - `TestFairness{,AutoEnable}Suite/Test_Activity_Basic` and
    `TestWorkflowTaskTestSuite/TestWorkflowTaskHeartbeatingWithEmptyResult` assume wall time passes
    during server work. Investigations are approved as
    [D19](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.19.md) for fairness and
    [D20](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.20.md) for heartbeat timeouts.
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

**Status.** Nineteen open items. D7 is required CI work: its darwin job is written and its
commands pass locally on darwin/arm64, and it stays open until a GitHub Actions run passes.
D12 and D14 are required fixes. D16-D20 require investigation before corrections are selected.
D1-D5 and D8-D10 are delivered through fn-108, fn-109, and fn-107. D6 clock policies and D15
larger traces remain explicitly deferred, and D11 dynamic Linux clock auditing stays deferred
on the D21 findings.

### Required work

| Item | Scope and origin | Decision |
| --- | --- | --- |
| D7 | Add a darwin/arm64 job for the existing functional smoke selection (`fn-101.4`) | Required CI addition approved 2026-09-30. R7 in fn-105 requires a standard macOS runner, preserved Linux coverage, platform-specific reports/artifacts, zero unsupported/failed/infrastructure errors, explicitly traced exact replay, and passing GitHub Actions evidence. |
| D12 | Identify and fix the linux/amd64 replay-divergence channel (`fn-106.1`); about one tier-3 seed-run in 26 diverged, and the recorded evidence points to host timing under load | Must be fixed, decided 2026-09-30. Native Linux instrumentation is an execution prerequisite. R12 in fn-105 requires a regression reproducer, repeated exact replay on both seeds under load, and restoration of strict CI expectations; diagnosis alone cannot close the task. |
| D14 | Fix Darwin TestSignalWorkflowTestSuiteChasm replay divergence (F7); two heap-span refills swap order at cluster start, historically about one seed-11 replay in 28 | Fixed in the working tree on darwin/arm64 (2026-09-30, toolchain key `8d28bd44`). Cause: before an M sleeps on a contended runtime lock, `lock2` samples the wait time (`mLockProfile.start`), and that draw came from the process-wide seeded stream when the M held the P. Whether the scheduler lock outlasts the spin is host timing; in the instrumented divergent replay the P's M waited in `findRunnable` while another M parked in `stopm`. The draw shifted every later seeded draw, type-assertion caches grew at other call sites (first on the cluster-start goroutine, in `database/sql` and `fx`), a later goroutine allocated a cache of another size, two heap-span refills swapped order, and the replay diverged at a choice ordinal. Fix: the sample draws from the M's own stream (`gomadLockProfileStart`, one `lock_spinbit.go` hunk). On the unfixed toolchain with scratch instrumentation 5 of 350 seed-11 replays diverged under host load; none of the 345 matching replays slept on a runtime lock while holding the P, and the dumped divergent replay did so once, just before its first differing event. Reproducer: `TestProfileSchedulerLockContentionLeavesSeededStreamInPlace` (fixture `io_handoff_contention`) failed 20 of 20 invocations on the unfixed toolchain and passed 20 of 20 on the fixed one. Fixed toolchain, tracing on, host load: 400 exact replays per seed of a retained success on seeds 11 and 17 with no divergence, 12 fresh repetitions per seed that agreed within each qualification, and four runs of the generated workload `qualified` on both seeds. The generated manifest expects darwin/arm64 `qualified` again. The patched code also runs on linux/amd64, so the same draw may be the D12 channel; that is an unverified hypothesis, linux/amd64 was not run, and D12 stays open. Evidence: `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d14-evidence.json`. |
| D16 | Investigate TestStandaloneActivityTestSuite/TestStartDelay/UpdateWhilePaused_AfterWindow_ExtendsDispatch; the client times out before an empty poll response under the forward clock | Investigated on darwin/arm64 (2026-10-01, toolchain key `8d28bd44`); nothing is fixed and the skip stays. Cause: the forward offset is cumulative and unbounded (`gomadTimeNow` adds 1 to 1024 ns per read and never subtracts), so `time.Now` led the timer clock by about 0.33 s when the subtest polled. `context.WithTimeout` arms its timer for `time.Until(time.Now()+d)`, which is `d` plus the lead, and each gRPC hop re-derives the deadline from `grpc-timeout` and adds the lead again. The frontend hop, the matching hop, and matching's `WithDeadlineBuffer` child add three leads; above 1 s in total the client deadline fires before the empty response. On the unmodified tree seeds 5, 12, 20, 21, 24 of 1 to 24 fail with the same error, and a rerun of each reproduced its stdout and stderr digests; seeds 11 and 17 pass (seed 11 by about 20 ms). Under `strict` the subtest passes on all seven of those seeds with the child deadline 1 s before the client's, and natively 999.8 ms before. A standard-library fixture reproduces the timeout on seeds 11 and 17 after 700,000 reads. gRPC, the 1 s budget, and the test behave as they do natively; the owner is the Gomad runtime clock. Proposed for a decision, not built or verified: add the draws to the virtual clock itself so `time.Now` and timers share one clock, as fn-103 specified; this first needs the simulation time transport failure that made fn-103 move to a separate offset reproduced and resolved, and its feasibility is not established. Partial fallbacks with stated residuals (offset-aware `time.Since`/`time.Until`; absorbing the offset on idle advances) and an accepted-limitation alternative are in the report. The correction, its conformance fixture, removal of the skip, and requalification of the seven forward suites remain open work in fn-105. linux/amd64 was not run. Report: `docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md`; evidence: `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d16-evidence.json`. |
| D17 | Investigate TestNexusOTELSuite/TestOperation with two clusters; configuring two pool slots resolves the initial wait but repeated runs and replay still differ | Investigation approved 2026-09-30. R17 in fn-105 requires reproductions, control comparisons, the first divergent event and missing-terminal-frame diagnosis, and an owned correction proposal for a subsequent decision. Keep the skip until verification supports removal; a shared D12/D14 cause requires evidence. |
| D18 | Investigate worker cancellation delivery and inconsistent test timeout budgets | Investigated 2026-09-30 on darwin/arm64 ([report](../docs/research/gomad/GOMAD_D18_WORKER_CANCEL_DELIVERY.md)). Cause: a strict-tick timestamp tie between the workflow run-timeout timer and the activity's capped ScheduleToClose timer; when the activity timer task runs first, no cancel command is generated, and no test budget or delivery step is at fault. Proposed correction, owned by the Gomad qualification configuration: `clock_tick: forward` for the suite, which qualified seeds 11 and 17 with the skip lifted in scratch manifests. The correction is open work pending a decision; nothing is fixed and the skip stays until verified. |
| D19 | Investigate activity fairness backlog readiness in both fairness suites | Blanket investigation approval 2026-09-30. R19 distinguishes setup bias from a fairness defect and preserves the fairness assertions. |
| D20 | Investigate heartbeat timeout counting under virtual time | Blanket investigation approval 2026-09-30. R20 establishes timeout/reset semantics and proposes a correction preserving timeout/recovery coverage. |

### Architecture consolidation

| Item | Scope and origin | Revival trigger |
| --- | --- | --- |
| D1 | Shared completed-execution assessment across seed, choice, and simulation (`fn-102` R2) | Revived by the 2026-09-30 cleanup request; fn-108 R6 owns delivery |
| D2 | Shared retention and artifact-input composition, with separate strategy transactions (`fn-102` R3) | Revived by the 2026-09-30 cleanup request; fn-108 R7 owns delivery |
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
owns requirements and acceptance criteria for reducing maintained production code while
preserving every shipped feature. Task breakdown must reuse or transfer fn-105 D1/D2
obligations; their original contracts remain in fn-102 R2/R3. D3's public executor
migration remains outside fn-108 and belongs to fn-109.

**Outcome.** A measured net reduction in authored production code, including new helpers,
types, and wrappers, with equivalent public behavior and fixed-input evidence. The roughly
200-line local-cleanup estimate is guidance, not a quota. Moving code between directories,
compressing formatting, or deleting comments does not satisfy the reduction.

### Delivery order

Record the implementation baseline and comparable production, test, generated, and
protocol/template counts before editing. Then deliver the following stages:

| Stage | Scope | Acceptance in fn-108 |
| --- | --- | --- |
| 1. Local cleanup | Recheck and remove unused internal helpers, decimal/decoder code, and test-only minimizer serialization; remove repeated canonical validation; reuse the existing memory-adapter rewrite and dossier-publication owners | R2–R5; retain independent validation, exact adapter bytes/pins, and dossier publication/error behavior |
| 2. Execution assessment | Give common World, coverage, choice-feature, and outcome interpretation one private owner across seed, choice, and simulation completion paths | R6; preserve detached evidence, validation order, failure precedence, and strategy-specific handling |
| 3. Retention and composition | Share novelty predicates, capacity calculations, and artifact inputs while retaining seed ordinal commits, atomic exploration rounds, and replay-verified corpus admission | R7; failed or interrupted publication cannot advance committed policy state |
| 4. Equivalence and size evidence | Compare final code counts and fixed-input canonical projections; run focused tests, generated/architecture checks, both-platform Gomad gates, Temporal integration/smoke, and affected qualification suites | R1, R8, R9; preserve current dispositions and report missing validation as incomplete acceptance |

**Preservation contract.** Keep World, both simulation backends, exploration/minimization,
guidance/corpus, portable campaign distribution, qualification-set sharding, prebuilt target
provenance, caches, and both `gomad3integration` and `gomad3sim`. Keep public Go interfaces,
CLI commands/flags/defaults, compatibility packs and versions, legacy choice inspection,
schemas, replay identities, resource bounds, behavioral assertions, and existing comments.
Runtime/overlay redesign and unrelated defect fixes remain outside fn-108.

D12/D14 replay fixes retain their separate owners and acceptance requirements. Cleanup
cannot weaken their expectations or treat existing divergence as new qualification evidence.

**Status.** Open spec; tasks are broken down and implementation has not started.

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
