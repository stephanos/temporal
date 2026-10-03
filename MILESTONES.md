# Gomad v3: Milestones to a Deterministic Temporal Functional Test

**Plan date:** 2026-09-08 · **Consolidated:** 2026-09-30

## Purpose

This document is the delivery ladder for running Temporal's functional tests (`./tests`, built on
the `testcore` one-box cluster with in-memory SQLite and loopback gRPC) under Gomad v3 so that the
same seed produces the same run and a retained artifact replays byte-exactly.
[GOMAD_NEXT.md](.plans/GOMAD_NEXT.md) remains the capability roadmap across all four tracks.

Initial functional-test delivery is complete: milestones F0–F9 (removed on 2026-09-30; see revision `3e4303807`
of this file, named `GOMAD_MILESTONES.md` until 2026-10-01, which manifest findings citing
`MILESTONES.md#f3-…` through `#f7-…` refer to),
the clock-tick spec `fn-103`, and the gap spec `fn-106` (see revision `3a7deb99d` for their
outcomes). The F10 items D1, D2, D13, D14, and D21–D25 are complete as well and were removed by 2026-10-01;
their outcomes are in the done summaries of the fn-105 tasks under `.flow/tasks/`. What remains
is the F10 backlog, the downstream implementation and qualification
in fn-107, feature-preserving code-size reduction in fn-108, deep modules and tool
interfaces in fn-109, runtime patch minimization in fn-110, the open findings, the
remaining test dispositions, and the rules that still apply. Vocabulary consolidation
and documentation acceptance are tracked in fn-111. The 2026-10-01
[quality assessment](#quality-assessment-2026-10-01) defines determinism-assurance and
test-strategy work, tracked in fn-112, and pin-maintenance work, tracked in fn-113.

## Work tracking

Open work is tracked as flow-next specs under `.flow/specs/`; their tasks and acceptance criteria
(R-IDs) are authoritative, and this document keeps rationale and status. When a spec's scope
changes, change the spec and summarize the change here.

| Milestone | Spec | State |
| --- | --- | --- |
| F10 | `fn-105-gomad-follow-ups-deferred-scope` | required D12 fix, the D26 forward-clock correction, the D27 host-clock remedies, and deferred follow-ups ([decisions and scope](#f10-follow-ups-deferred-scope)) |
| Downstream cell | [fn-107](.flow/specs/fn-107-gomad-finish-downstream-cell.md) | open; consumer implementation and exact replay on both platforms, including D8/D9/D10; blocked on 2026-10-01 because the `../downstream` checkout its tasks edit is absent from this machine |
| Code-size cleanup | [fn-108](.flow/specs/fn-108-gomad-reduce-code-size-without-removing.md) | open; tasks 1–7 of 8 done and verified on darwin/arm64: production Go down 286 code lines, D1/D2 delivered; task 8 owns the outstanding linux/amd64 gates for R9 ([status](#code-size-cleanup-fn-108)) |
| Deep modules and tool interfaces | [fn-109](.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md) | open; tasks 1 and 22 of 22 done: isolated simulation bounds preserved and real simulation exploration repaired; twenty tasks remain for the architecture findings, reusing D1/D2 and reviving D3/D4/D5 ([delivery order](#deep-modules-and-tool-interfaces-fn-109)) |
| Runtime patch minimization | [fn-110](.flow/specs/fn-110-gomad-minimize-the-runtime-patch.md) | open; task 1 of 5 done with a dated baseline; tasks 2–5 retain overlay extraction, canonical one-context-line regeneration, and both-platform qualification ([delivery order](#runtime-patch-minimization-fn-110)) |
| Vocabulary and documentation | [fn-111](.flow/specs/fn-111-gomad-consolidate-vocabulary-and-update.md) | done; all three tasks and current spec completion review passed on 2026-10-02; canonical vocabulary and current guide/audit acceptance retained ([scope](#vocabulary-and-documentation-fn-111)) |
| Determinism assurance and test strategy | [fn-112](.flow/specs/fn-112-gomad-determinism-assurance-and-test.md) | open; tasks 1–4, 6–8, and 11–15 done; baseline CI green on `8789deab0`, simulation/replay gates repaired, diagnostics and timer/shuffle fixtures verified, generated filesystem/TCP conformance added with model fixes, and built-CLI recovery tests reviewed on darwin/arm64; watchdog classification and diagnostic replay repaired; exploration rounds keep the parent's cancellation classification (task 14, darwin only; its native Linux CI rerun is open) and the watchdog replay test no longer races its wall clock (task 15). Task 16 (two retained successes with one outcome signature) is queued; stream isolation, soak gate, and suite reshaping remain ([findings and proposed order](#quality-assessment-2026-10-01)) |
| Version-pin maintenance | [fn-113](.flow/specs/fn-113-gomad-reduce-version-pin-maintenance.md) | open; tasks 1–4 implemented on linux/arm64 with a development harness only: `gomadtool pin-impact`, `adapter-regenerate`, and `compatibility-pack refresh`, with no pin loosened, and the bump procedure documented; a walked gRPC plus klauspost/compress bump takes about 11 steps (1 hand edit) instead of about 20 (2+ hand edits) before the shared gates, or about 41 when all 11 rewritten gRPC files change upstream. R6 (`validate`, `test`, pack qualification, and the core set on darwin/arm64 and linux/amd64) is not run ([cost evidence](#maintenance-cost), [task 4 evidence](.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-4/evidence.md)) |
| Search-path defects and wasted work | [fn-114](.flow/specs/fn-114-gomad-correct-search-path-defects-and.md) | open; tasks 1–4 and 7 done and reviewed; guided selection now skips answered seeds, with frozen regression/resume/shard selection ([task 7 evidence](.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-7/final-completion-summary.md)); C3's error path, E4's historical probe premise, and C2's predicted prefix divergence corrected. Callback identities swap but valid same-seed prefixes succeed; all seven select shapes have an exhausted unreduced baseline. Corpus identity now binds environment and tick policy, and coverage-instrumented targets are rejected by provenance and replay validation ([task 3 evidence](.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-3/final-completion-summary.md)). Host gates pass on darwin/arm64; Linux remains unverified; forced-prefix divergences and completed siblings now survive commit/resume ([task 4 evidence](.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-4/final-completion-summary.md)). On 2026-10-02 tasks 5, 6, 8–12, 15, and 16 were also done and reviewed on darwin/arm64: schedule-independent timer-callback identities with a checked parentless-creation inventory, the choice-exploration start ordinal, persisted minimizer state with per-parent `minimize --resume`, one shared prepared target per store with count-once byte accounting (unpruned representative set 1.17 GB on disk versus 11.44 GB counted without sharing), and select readiness recorded in the Choice Trace with no-op select-poll decisions no longer expanded after a soundness check. The toolchain build key is now `2008ea81`. Linux remains unverified for all of them; task 13 and the qualification task 14 remain |

Work a spec with `/flow-next:work <spec>`; list what is ready with `flowctl ready`.

## Immediate delivery order

Approved on 2026-10-02: fn-114.4 is reviewed, complete, and pushed through `67dbe0266`.
Fn-111 documentation acceptance and completion review are also done. Later on 2026-10-02,
fn-114 tasks 5, 6, 8–12, 15, and 16 and fn-112 tasks 14 and 15 were completed and reviewed
on darwin/arm64. Milestone execution is paused again at the user's request after fn-114.12.
On resumption, fn-114.13 (runtime-owned goroutine ordering) precedes the fn-114.14
qualification, then fn-112.16. Every task completed on 2026-10-02 still needs a native Linux
run, which a push to the fork's CI provides. Reviews now use the `claude` backend
(claude-opus-5-5, the same model as the implementer), so they are not cross-family
verdicts. Runtime extraction and pin maintenance retain their existing qualification requirements.

## Verification instructions for agents

Apply this workflow to Gomad milestone work. Optimize elapsed time by choosing checks that
cover the changed behavior and reusing valid results from the same source revision.

1. **Check cheap boundaries first.** Run focused regressions during implementation. After
   import or package-boundary changes, run `TestPackageArchitecture` in the nested module's
   root package. Run `make -C tools/gomad3 validate` before broad tests when changing files
   that may affect generated code, protocol identities, or toolchain inputs. Check the
   generator's input list before editing a shared host/runtime file.
2. **Run one full host gate per frozen batch.** Use
   `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` with the pinned native Go
   on `PATH` and the documented toolchain setup. This includes Runner and CLI packages;
   count it as covering their overlapping task test commands instead of running both broad
   suites. Retain focused regression evidence and identify the packages covered. Runtime,
   overlay, integration, race, and platform-specific requirements still need their own gates.
3. **Scope reruns to the new change.** After a small review fix, run the affected regression,
   package, and relevant boundary checks. Repeat the full gate when a shared dependency,
   runtime or protocol change, broader regression, or unresolved coverage concern warrants
   it; record the reason. Documentation-only edits need document/diff checks. Keep source
   stable during each test command and identify the revision each result covers.
4. **Keep handoffs small.** Retain the meaningful failing regression, final passing commands,
   exit codes, elapsed times, source revision, and review verdict in one task handover.
   Reference existing evidence instead of copying it into successive manifests and reports.
   Keep generated binaries, bulk traces, and scratch snapshots local unless delivery requires
   them. Use one independent review for a completed batch; re-review actionable fixes.
5. **Advance after the gate.** Once the required checks and review pass, complete the Flow task
   and perform the authorized delivery action. Broaden testing only for a concrete remaining
   risk. Report unavailable checks honestly; retry an unchanged environment failure only
   when its cause or relevant inputs have changed.

On 2026-10-02, focused checks took roughly 2–15 seconds, Runner/CLI suites 111–126 seconds,
and full host suites 128–150 seconds on darwin/arm64. Removing the overlapping broad suite
saves about two minutes per cycle. Use current command timings to guide further optimization.

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
    [D16](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.16.md) investigation
    ([report](docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md)) assigns the
    correction to the Gomad forward clock. The correction is proposed and not implemented, and
    the skip stays.
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
  fully qualified. The 2026-10-01 assessment adds two untested candidates and the missing
  localisation tooling ([determinism gaps](#determinism-gaps)).
- **Baseline CI restored (fn-112 task 1).** All six jobs passed in
  [run 36968858553](https://github.com/stephanos/temporal/actions/runs/36968858553) on
  `8789deab055d1b72ac6bc86711d74f3fd7313fa2`, including both platform core and smoke gates.
  The previous Linux failures came from a stale modernc compatibility-pack profile and
  retention fixtures assuming platform-independent candidate rank. The pack was rebound and
  fixture ranks now follow the platform's candidate identities; D12 allowances are unchanged.
  [Retained evidence](.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-1/summary.md)
  binds the logs and inspected source. This run predates task 2's new gates; their Linux
  execution remains unverified.
- **Host-clock escapes** recorded by the static inventory (`toolchain/clock_inventory_test.go`):
  `gcMarkTermination` stamps `MemStats.LastGC` and `MemStats.PauseEnd` with host wall time (also
  visible through `debug.GCStats`, the text heap profile, `expvar`, and the Prometheus
  `go_memstats_last_gc_time_seconds` gauge), the FIPS entropy source's `monoTime` (host time on
  linux/amd64 only), and the execution tracer's clock snapshot; on linux also
  `syscall.Gettimeofday` behind the `syscall` pack gate. Investigated, not fixed
  ([D21 report](docs/research/gomad/GOMAD_HOST_CLOCK_ESCAPES.md), darwin/arm64 2026-09-30;
  linux/amd64 by source reasoning only): none of these is read by runtime control flow; each
  hands a host value to the target as reporting state, trace output, FIPS entropy input, or the
  result of a pack-gated call, so a target that prints or branches on one is not repeatable. A
  fixture that prints `LastGC` is `nondeterministic` at stdout while its choice tape replays
  exactly. By source inspection the Temporal functional closure links the readers but neither
  emits nor consumes the values, and these four escapes are not a cause of D12 or D14. Proposed
  and selected on 2026-10-01 as [D27](#required-work): state the limitation in the contract, pin the unpinned darwin
  `gettimeofday` path and `cputicks` (on linux/amd64 host cycle counts steer block and mutex
  profile sampling when a rate is set and reach text profiles and sampled protobuf profiles; not
  shown unrelated to D12), and overwrite the stamps with the stored virtual time from `runtime/proc.go`. That
  overwrite changes no prohibited file, leaves the host read in place, and writes
  collector-owned state, so it needs the patch-policy owner's judgement, as does guarding the
  read in `mgc.go` or the linux assembly. The collector patch prohibition remains in force.
- **DTrace clock audit** on darwin needs a root run; CI supplies it on the macOS runner.
- **Downstream cell.** Gomad-side work for a module that embeds the server is complete; its
  in-process cluster test classifies as a capability blocker until the downstream cuts its own
  seams and injects its filesystem and membership transport ([GOMAD_CLOUD.md](.plans/GOMAD_CLOUD.md)).

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
  [the compatibility roadmap](.plans/GOMAD_NEXT.md#compat-5-targeted-deterministic-adapters-and-io-models).
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

**Spec.** [fn-105-gomad-follow-ups-deferred-scope](.flow/specs/fn-105-gomad-follow-ups-deferred-scope.md)
owns tasks and acceptance criteria.

**Outcome.** Fix work explicitly selected during the issue walkthrough must meet its
acceptance criteria. Other items retain their original deferral reasons and revival
triggers from the 2026-09-29 scope cut. A trigger must be recorded here before a
deferred item is implemented; deferred items may remain open or close as won't-do.

**Status.** Thirteen open items. D7's macOS smoke job passed in GitHub Actions;
its report and source-bound job evidence are retained with the task.
D12 is a required fix that needs a linux/amd64 host; the D14 lock-profile fix is an unverified
candidate for it. D17–D20's recommended corrections are applied and qualified on darwin/arm64.
D16 retains its skip; its correction is D26, and the D21 remedies are D27, both selected on
2026-10-01 and not started.
D3-D5 and D8-D10 are delivered through fn-109 and fn-107. D6 clock policies and D15
larger traces remain explicitly deferred, and D11 dynamic Linux clock auditing stays deferred
on the D21 findings.

### Required work

| Item | Scope and origin | Decision |
| --- | --- | --- |
| D7 | Add a darwin/arm64 job for the existing functional smoke selection (`fn-101.4`) | Complete: the [macOS smoke job](https://github.com/stephanos/temporal/actions/runs/36945812832/job/110647570973) passed on `macos-15` at `38957053f`. All four workloads qualified on seed 11 with exact replay and zero unsupported, failed, infrastructure-error, or replay-diverged outcomes. The workflow and smoke manifest match the current checkout. [Retained evidence](.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d7-github-evidence.json) binds the job, report, and source hashes. The overall workflow failed in Linux jobs, which remain separate required work. |
| D12 | Identify and fix the linux/amd64 replay-divergence channel (`fn-106.1`); about one tier-3 seed-run in 26 diverged, and the recorded evidence points to host timing under load | Must be fixed, decided 2026-09-30. Native Linux instrumentation is an execution prerequisite. R12 in fn-105 requires a regression reproducer, repeated exact replay on both seeds under load, and restoration of strict CI expectations; diagnosis alone cannot close the task. |
| D16 | Investigate TestStandaloneActivityTestSuite/TestStartDelay/UpdateWhilePaused_AfterWindow_ExtendsDispatch; the client times out before an empty poll response under the forward clock | Investigated on darwin/arm64 (2026-10-01, toolchain key `8d28bd44`); nothing is fixed and the skip stays. Cause: the forward offset is cumulative and unbounded (`gomadTimeNow` adds 1 to 1024 ns per read and never subtracts), so `time.Now` led the timer clock by about 0.33 s when the subtest polled. `context.WithTimeout` arms its timer for `time.Until(time.Now()+d)`, which is `d` plus the lead, and each gRPC hop re-derives the deadline from `grpc-timeout` and adds the lead again. The frontend hop, the matching hop, and matching's `WithDeadlineBuffer` child add three leads; above 1 s in total the client deadline fires before the empty response. On the unmodified tree seeds 5, 12, 20, 21, 24 of 1 to 24 fail with the same error, and a rerun of each reproduced its stdout and stderr digests; seeds 11 and 17 pass (seed 11 by about 20 ms). Under `strict` the subtest passes on all seven of those seeds with the child deadline 1 s before the client's, and natively 999.8 ms before. A standard-library fixture reproduces the timeout on seeds 11 and 17 after 700,000 reads. gRPC, the 1 s budget, and the test behave as they do natively; the owner is the Gomad runtime clock. Proposed for a decision, not built or verified: add the draws to the virtual clock itself so `time.Now` and timers share one clock, as fn-103 specified; this first needs the simulation time transport failure that made fn-103 move to a separate offset reproduced and resolved, and its feasibility is not established. Partial fallbacks with stated residuals (offset-aware `time.Since`/`time.Until`; absorbing the offset on idle advances) and an accepted-limitation alternative are in the report. The correction, its conformance fixture, removal of the skip, and requalification of the seven forward suites remain open work in fn-105. linux/amd64 was not run. Report: `docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md`; evidence: `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d16-evidence.json`. |
| D17 | Deliver explicit environment entries and enable the two-cluster Nexus operation | The [investigation](docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md) found that the runtime discarded the supplied pool-size setting, causing the test to wait on its own single cluster slot. Runner-managed targets now retain explicit environment values before package initialization; controls stay hidden, direct seeded launches retain their TZ-only environment, and disabled behavior is preserved. The suite receives `TEMPORAL_TEST_DEDICATED_CLUSTERS=2`, and its skip is removed. On darwin/arm64, the new regression passes after reproducing the defect; both leaf seeds create two clusters, all four suite subtests pass on seeds 1–17, and seeds 11/17 qualify with exact replay. The tracked manifest and native suite pass; the one-slot control still times out at the same pool receive. The full `make -C tools/gomad3 test` gate passes, including runtime and upstream conformance. [Correction task](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.30.md) and [evidence](.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d17-correction-evidence.json) record source and build identities. linux/amd64 was not run. |
| D18 | Correct worker cancellation qualification after the timeout investigation | The [investigation](docs/research/gomad/GOMAD_D18_WORKER_CANCEL_DELIVERY.md) found a strict-tick timer tie, not a delivery or budget defect. The recommended `clock_tick: forward` is applied to `TestWorkerCommandsTaskSuite` and the skip is removed. On darwin/arm64, the tracked suite qualifies on seeds 11 and 17; traced exact replay matches on both seeds; the leaf passes on seeds 1–64, while the strict reproducer still fails on 9 of 16. The native suite passes. [Correction task](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.27.md) and its retained evidence record the checks. linux/amd64 was not run. |
| D19 | Correct fairness backlog readiness and auto-enable trigger polling | The [investigation](docs/research/gomad/GOMAD_D19_FAIRNESS_BACKLOG_READINESS.md) found test setup bias and a normal empty poll during queue reload, with no matcher fairness defect. The test now waits for all 225 activity tasks before measuring and retries an empty trigger poll; both skips are removed. On darwin/arm64 both tracked suites qualify on seeds 11 and 17, exact choice-tape replay matches for all four results, and strict-tick seeds 1–17 pass for each suite at unfairness 0.45. The native suites pass once; a 30-run auto-enable leaf check had two failures at the previously observed workflow-task wait, unrelated to the new readiness check. [Correction task](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.28.md) and [evidence](.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d19-correction-sweep.json) record the checks. linux/amd64 was not run. |
| D20 | Correct heartbeat timeout counting under virtual time | The [investigation](docs/research/gomad/GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md) found that the test relied on incidental latency to cross a strict heartbeat deadline. The test now pins the 5 s timeout, sends both rejected heartbeats at least 100 ms past a local upper bound on each chain deadline, and asserts every heartbeat outcome; the expected 47-event history and both timeout/recovery cycles are preserved. The skip is removed. On darwin/arm64 the leaf passes seeds 1–64 under strict and forward clocks and 20 native repetitions; after strengthening the new assertion to Require, the native suite and tracked qualification pass, with separate traced exact replay on seeds 11 and 17. [Correction task](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.29.md) and [source-bound evidence](.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d20-correction-evidence.json) record the verification scope. linux/amd64 was not run. |
| D26 | Correct the forward clock found by D16 ([task](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.31.md)) | Selected 2026-10-01, not started. Put the forward draws on the virtual clock so `time.Now` and timers share one clock, as fn-103 specified. The fn-103 simulation time transport failure must be reproduced and resolved first, and feasibility is not established; an infeasible result returns the D16 report's fallbacks for a decision. Acceptance in fn-105 R26 adds a conformance fixture, removes the D16 skip, and requalifies every forward workload with traced exact replay. |
| D27 | Apply the D21 host-clock escape remedies ([task](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.32.md)) | Selected 2026-10-01, not started. State the escapes in the contract and pin the darwin `gettimeofday` path and `cputicks` with fixtures. The stamp overwrite from `runtime/proc.go` waits for the patch-policy owner's recorded approval; without it the stamps stay a stated limitation. Acceptance is fn-105 R27. |

### Architecture consolidation

| Item | Scope and origin | Revival trigger |
| --- | --- | --- |
| D3 | Private executor injection instead of public `Executor`/`ReplayExecutor` (`fn-102` R4); a Go API change | Revived by the 2026-09-30 architecture request; fn-109 R5 owns delivery |
| D4 | Architecture fitness checks for ownership, host effects, and public signatures, with negative fixtures (`fn-102` R5) | Revived by the 2026-09-30 architecture request; fn-109 R8 owns delivery |
| D5 | Architecture/platform/determinism documentation reconciliation (`fn-102` R6) | Revived by the 2026-09-30 architecture request; fn-109 R9 owns delivery |

These were deferred as maintenance without a waiting consumer or behavior change.
The [architecture assessment](.flow/artifacts/fn-102-gomad-architecture-consolidate/architecture-assessment.md)
retains the evidence. Preserve CLI behavior, schemas, canonical bytes, failure
classification and precedence, and replay compatibility when reviving them.

The 2026-09-30 request for feature-preserving code-size reduction revives D1/D2
under [fn-108](.flow/specs/fn-108-gomad-reduce-code-size-without-removing.md).
Reuse or transfer the existing obligations during task breakdown. The subsequent
request to address every architecture finding creates
[fn-109](.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md),
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
| D11 | Dynamic Linux clock audit with disabled vDSO, seccomp denial, and positive control (`fn-101.3`, pre-amendment R5) | Stays deferred: D21 (2026-09-30) found no evidence that requires it, and it is not implemented. The static inventory remains the linux escape gate | A selected fix removes or guards a linux clock read (the `proc.go` overwrite does not), D12 attributes a divergent event to a host-clock read, an upgrade adds an unclassifiable reference, or a pack admits a raw clock syscall. Feasible scope is in the [D21 report](docs/research/gomad/GOMAD_HOST_CLOCK_ESCAPES.md#d11-determination) |
| D15 | Larger choice traces (`fn-106.3`, split from D13 on 2026-09-30) | Routine qualification uses seed repeatability; the opt-in policy does not require larger tapes | A named workload needs a retained decision tape beyond 64 MiB for debugging, replay verification, exploration, or minimization |

The [constraints](#constraints) apply throughout.
New tick policies carry execution identity and the
[COMPAT-5 evidence set](.plans/GOMAD_NEXT.md#compat-5-targeted-deterministic-adapters-and-io-models).
Downstream packs/adapters bind exact versions; dependency drift reopens their
qualification. On revival, use the origin spec's requirement text as acceptance.

## Code-size cleanup (fn-108)

**Spec.** [fn-108](.flow/specs/fn-108-gomad-reduce-code-size-without-removing.md)
owns the requirements. Tasks 1–7 of eight are done; the delivery order and preservation contract
were removed from this document on 2026-10-01 and remain in the spec. Measurements, the comment
audit, and gate reports are in
[final.md](.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final.md).

**Status.** Implemented and qualified on darwin/arm64 on 2026-10-01. Authored
production Go across `tools/gomad3`, `tools/gomad3sim`, and `tools/gomad3integration` fell from
58624 to 58338 code lines (-286) and by 10691 code bytes under one counting script, with the two
new owner files `runner/completion.go` and `runner/retention.go` counted. The runtime overlay,
generated files, the patch, schemas, and templates are unchanged; tests grew by 1929 code lines.
Public `go doc` and CLI help captures equal the baseline, and compatibility packs, qualification
manifests, and schemas are untouched. Six comment lines were removed: three copies of one comment
whose code was merged, and the comment remains above the merged code. On darwin/arm64,
`make -C tools/gomad3 validate` and `test`, the `gomad3sim` and integration tests, the smoke set
(4/4), the core set (7/7), and the representative Temporal set (28/28 on seeds 11 and 17, 56
exact replays) passed, and all 1140 baseline tests keep their result. R9 is incomplete, with
[task 8](.flow/tasks/fn-108-gomad-reduce-code-size-without-removing.8.md) owning the linux run: no
linux/amd64 gate ran because no host was available. `make lint-code-fast` exits 2 as it did
before fn-108 and cannot analyse the nested module. Minimization has no before-and-after
projection and rests on its existing tests. Evidence and the linux commands are in
[final.md](.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final.md).

## Deep modules and tool interfaces (fn-109)

**Spec.** [fn-109](.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md)
owns all eleven ranked findings and five secondary opportunities from the
[architecture assessment](.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/architecture-assessment.md).
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

**Status.** Open spec with 22 tasks; tasks 1 and 22 are done. The isolated coordinator now
carries simulation bounds, and a real simulation-exploration campaign completes locally and
through the coordinator with exact replay on darwin/arm64. Their task summaries retain the
evidence. Tasks 2–21 remain; the next options-owner task can reuse the repaired execution path.

## Runtime patch minimization (fn-110)

**Spec.** [fn-110](.flow/specs/fn-110-gomad-minimize-the-runtime-patch.md)
owns both approaches from the
[patch-size investigation](docs/research/gomad/GOMAD_PATCH_SIZE.md).
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

**Status.** Open spec with five tasks; task 1 recorded the patch, overlay, and qualification
baseline. Tasks 2–5 remain. That baseline used toolchain key `8d28bd44`; re-anchor it to the
current inputs before extraction. Its simulation failures were subsequently repaired under
fn-109 task 22 and fn-112 task 2, whose evidence supersedes those historical failures.

## Vocabulary and documentation (fn-111)

**Spec.** [fn-111](.flow/specs/fn-111-gomad-consolidate-vocabulary-and-update.md)
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

**Status.** Done: all three tasks and the current spec completion review passed on 2026-10-02. Vocabulary and
guide reconciliation evidence is retained under
`.flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update/` and is bound to
the recorded source hashes. Re-run affected acceptance checks when those contents change;
committing unchanged contents does not invalidate the evidence. Current acceptance and review are
retained in [task 3](.flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update/task-3/acceptance-summary.md).
Closure changes only the milestone status text after that audited snapshot.

## Quality assessment (2026-10-01)

**Scope.** A read-only assessment of determinism soundness, the test suite, and maintenance
cost. Three audits read the source, manifests, workflows, and CI history. Nothing was built or
run, so every runtime finding below comes from source reading and is unverified by execution
unless it cites an existing report.

**Spec.** [fn-112](.flow/specs/fn-112-gomad-determinism-assurance-and-test.md) owns Q1, Q2, Q6,
Q7, Q8, the suite defects, and the test layers, with acceptance criteria R1 to R11. Q3 and Q4 are
recorded as candidates on the fn-105 D12 task, and Q5 is delivered by D26. fn-112 has thirteen
tasks; its original ten-task plan review returned `SHIP` on 2026-10-01. The CLI tests exposed
three additional follow-ups: [task 12](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.12.md)
has repaired stock-compiler selection from the normal host-test entrypoint, and
[task 11](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.11.md) now preserves watchdog
classification when a killed target leaves no I/O terminal.
[Task 13](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.13.md) now executes diagnostic
replay of those artifacts. Tasks 1–4, 6–8, and 11–13 are done: baseline CI fixes,
simulation and choice-replay gates, runtime diagnostics and host trace comparison, timer/shuffle
fixtures with explicit host-channel limits, generated filesystem/TCP conformance with model fixes,
built-CLI explore/replay and exact-boundary kill/resume tests, stock-compiler selection, and
watchdog classification with valid incomplete evidence and diagnostic replay.
Task 4 passed implementation review after repairing interrupted-run reports and saved-report
evidence bindings. Its full host and runtime gates passed on darwin/arm64; Linux remains
unverified. [Task 4 evidence](.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-4/completion-summary.md)
records the checks, unchanged diagnostics-off identities, and root-lint discovery limitation.
Its open questions (soak target, ordering of the runtime edits shared with fn-105, fn-109, fn-110, and
fn-114, patch-policy approval, and the fate of `gomad3sim` and `resume`) are in the spec.

**Assessment verdict (2026-10-01).** Determinism rests on point fixes found after each divergence. No invariant states
that host timing cannot reach the seeded stream or the heap, and no tool locates where two
same-seed runs first differ. The suite has about 1,270 tests in 42k lines against 57k production
lines, and most of that mass covers Runner bookkeeping. "Never fails" is not provable for a design
that keeps the collector, the allocator, and real threads inside the boundary. The achievable
target is a checked invariant plus a measured bound, such as zero divergences in 10,000 loaded
seed-runs per platform.

### Determinism gaps

| # | Gap | Evidence | Proposed direction |
| --- | --- | --- | --- |
| Q1 | No divergence localiser | Choice records carry kind, site, alternatives, and identities, with no stream position, allocation count, or GC cycle. Replay reports a diverging ordinal; two fresh runs report only the differing evidence field name. D12 surfaces at ordinals 8 to ~85k, far from its cause | Add a state digest to each choice record (draw counters, malloc count, GC cycle and phase, virtual time) and a differ for two fresh traces |
| Q2 | No structural barrier against host timing | Any M holding the P draws from the one process stream. Four host-timed draw sites were rerouted after incidents; the remaining sites are unaudited | Per-purpose random streams, and a runtime assertion that host-timed paths never draw from the seeded stream |
| Q3 | D12 candidate: one-shot syscall wait in `suspendG` | The patch waits for a host syscall to exit once, before the suspend loop. A goroutine that enters a syscall in that window can be scanned mid-syscall, which shifts scan work. Inferred from source, not reproduced | Wait inside the loop's `_Gsyscall` case, then measure D12 on a native linux host |
| Q4 | D12 candidate: linux ASLR and `runtime.NumCPU` | Neither is controlled on linux/amd64. Impact is inferred low because the binary is non-PIE | Rerun the D12 reproducer under `setarch -R` before any code change |
| Q5 | Forward clock runs `time.Now` and timers on different clocks | D16: the offset only grows, so derived deadlines fire late and the subtest fails 5 of 24 seeds. Eight generated workloads use `forward` | D26: one shared clock as fn-103 specified, feasibility-gated |
| Q6 | Qualification has too little statistical power | `qualify` repeats twice. Exact replay forces run-queue and select choices, which masks perturbations that would split two fresh runs. A 1-in-26 defect passes most two-repetition checks | A scheduled soak of N fresh repetitions under CPU load with a strict zero-divergence gate |
| Q7 | Channels with no fixture | netpoll, SIGPROF, block and mutex profilers on linux, `NumCPU`, timer-tie and run-queue shuffle draws (seeded, never taped) | One seeded fixture with a positive control per channel |
| Q8 | Closure mode compiles no guards | `-gomadguard` is added only in guarded mode; pack admissions of `syscall` are per package, so admitted code runs live | State the limit in the contract, or sample guarded mode in the soak |

Q1 is implemented: `explore --diagnostics` and `qualify --diagnostics` retain fresh traces,
and `gomadtool diagnostic-diff` reports the first ordinal and differing fields. Interrupted
traces are explicitly unavailable; complete pairs are fully validated before comparison.
The injected-draw fixture localizes ordinal 5. Applying this tool to D12 still needs a native
Linux host. Q3
and Q4 are candidates for D12 and stay under fn-105 R12 acceptance if they are pursued.

Task 6 adds dedicated timer-tie and run-queue-shuffle fixtures to repeatability, seed-diversity,
and host-load checks. Both produced 32 distinct orders across 32 seeds, and the full Darwin
runtime gate passed; implementation review found no issues. Host netpoll readiness, SIGPROF,
enabled block/mutex profiling, and unvirtualized `NumCPU` have explicit exclusion sentences,
along with the closure-mode guard limit, retained for task 10 to publish in README/SPEC.
[Task 6 evidence](.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-6/completion-summary.md)
records the source basis and limits; Linux remains unverified.

Task 7 compares 64-operation filesystem and loopback TCP sequences across five fixed seeds
against stock Go 1.27.1. It fixes self-rename data loss and closed-handle error differences,
including libc errno preservation and stock directory-read wrappers and sentinels. Three
repetitions matched 1,920 operations per side, with only the declared directory-size difference.
Generation, host, runtime and overlay gates passed on darwin/arm64 build `56e4a2f0`; independent
review returned `SHIP`. Native Linux remains unverified, and root lint retains its documented
reference and nested-module discovery limitations.
[Task 7 evidence](.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-7/final-completion-summary.md)
binds the final sources and results and retains the intermediate failures and corrections.

Task 8 now drives the built CLI through explore/replay, published-campaign resume rejection,
and coordinator kills after exactly two or zero journaled executions. Twenty consecutive final
runs passed on darwin/arm64, including forty kills and 120 execution comparisons. Review caught
and corrected an excluded journal-capacity limit; a demonstrated red/green negative control now
protects it. The full host gate passes with an explicit stock compiler, and independent review
returned `SHIP`. Tasks 12 and 11 below fix the default compiler lookup and watchdog-terminal
defects. Native Linux remains unverified; root-lint limitations are unchanged.
[Task 8 evidence](.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-8/final-completion-summary.md)
binds the final source, gate logs and review receipts.

Task 12 resolves the pinned stock compiler before the patched test driver changes PATH. The
standard host gate now passes without an override: all 45 packages ran fresh on darwin/arm64.
Explicit and empty overrides pass; patched and wrong-version compilers remain rejected.
Independent review returned `SHIP`. The change is confined to the Make entrypoint; direct
patched-driver invocation still needs an explicit stock override. Linux execution remains
unverified, and root-lint limitations are unchanged.
[Task 12 evidence](.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-12/final-completion-summary.md)
binds the red/green gates, unchanged toolchain identity, and review receipt.

Task 11 distinguishes an absent I/O terminal from corrupt terminal data, tolerating absence
only after a verified watchdog or cancellation. The reproduced CLI runner error now becomes
status 1 with a watchdog observation and an inspectable artifact; no incomplete transcript is
published as complete. Four coverage modes, existing recovery tests and all 45 fresh host
packages pass on Darwin; independent review returned `SHIP`. Inspection and verify-only replay
pass. Task 13 below repairs the separately reproduced executable replay rejection. Linux and
root-lint limitations remain.
[Task 11 evidence](.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-11/final-completion-summary.md)
binds the reproduction, corrected artifacts, final gates and review receipt.

Task 13 replays supported watchdog artifacts without a complete I/O transcript as diagnostic
observations, using only retained captured inputs. Both the task 11 artifact and a fresh artifact
return status 1 with a matching diagnostic observation and no exact choice claim. Exact replay
still requires complete evidence; missing-I/O artifacts with complete recorded choices fail
explicitly. Captured-file replay succeeds after deleting the host source, while uncaptured inputs
fail. All 45 fresh host packages and CLI regressions pass on Darwin; independent review returned
`SHIP` and confirmed the task 11 replay finding is addressed. Linux and root-lint limitations remain.
[Task 13 evidence](.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-13/final-completion-summary.md)
binds the final sources, retained artifacts, negative controls and review.

### Testing strategy

The suite should have four layers. New tests belong to one of them.

| Layer | Purpose | State on 2026-10-01 |
| --- | --- | --- |
| 1. Runtime conformance fixtures | One seeded black-box fixture per nondeterminism channel, each with a positive control | Strong: 44 fixture directories; 10 fixtures run 100 times on 3 seeds, 32 seeds must diverge for 6 fixtures and 20 map families. Missing the Q7 channels |
| 2. Determinism soak | N fresh repetitions of real workloads under host load, strict gate | Missing. Qualification runs two repetitions |
| 3. Model conformance | Generated operation sequences compared between the in-memory filesystem or TCP model and the host OS | Missing. Expected results are hand-coded, no test file imports `math/rand` or `testing/quick`, and the 3 fuzz functions are panic-only decoder fuzzers |
| 4. End-to-end CLI | `explore`, `replay`, and kill-then-resume through real processes | Thin. `explore` and `replay` are never driven end to end, and no test kills a coordinator and compares the resumed campaign with an uninterrupted one |

Suite defects from the assessment, updated after fn-112 task 2:

- `test-simulation` now runs all six tagged `tools/gomad3sim/*_toolchain_test.go` files through
  the seeded exec wrapper and ten Runner transport cases, including seven process-node cases.
  The gate is included in `test` and both platform CI jobs.
- `overlay-test` now includes all five omitted packages: `internal/gomadsim`,
  `internal/gomadio`, `internal/gomadmodelwire`, `os`, and `cmd/internal/gomadcap`.
  The simulation tests use an external package to avoid the `testing`/`os` import cycle;
  the model-response test supplies runtime control environment at subprocess startup.
- The `choice_replay` fixture now records seed 1, finds an unforced seed with different output,
  and requires its tape to reproduce both the output and projected decisions under that seed.
- `./toolchain` runs under patched `test-toolchain` and stock `test-builder`; `test-host` and
  the stock Linux host-tools job no longer duplicate it.

Only `TestProcessBackendSynchronizesNodeClockWithModelDelay` remains outside
`test-simulation`, with its watchdog failure retained as a finding.
`TestProcessBackendResetsGlobalsDescriptorsAndGoroutines` is restored after fixing premature
cancellation of node model/time services during graceful cleanup. A deterministic regression
fails before that fix and passes afterward; the transport context is canceled on process exit.
[Task 2 evidence](.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-2/completion-summary.md)
retains the failures and the passing full darwin/arm64 `make test` log. The all-green baseline
CI run predates these gate additions, so the new Linux gate's status is unverified.

Candidates to shrink into table-driven or generated form, because they detect change more than
defects:

- `cmd/gomad/internal/cli`: 53 of 62 tests assert fields forwarded to injected dependency structs.
- `runner`: 62 of 166 tests drive a fake executor.
- `deterministicio`: 25 pinned-string or digest tests and 55 rejection tables; eight adapter test
  files share one template, and two of them differ in 9 of 191 lines.
- `architecture_test.go`: the import and export rules have value; the deleted-file list, banned
  words, and required filenames pin past refactors.

Thin coverage of shipped behavior: the minimizer (4 unit tests, no real-subprocess run), corpus
replacement and eviction, merge with more than two shards, and `world/process` `Open`, `Finish`,
and `FinishError`.

### Maintenance cost

Recurring costs are exact-version pins. Upstream `go.mod` changed in 73 commits over six months,
including 4 `go` directive bumps.

| Pin | Size | Repair |
| --- | --- | --- |
| Runtime patch and overlay | 1045 patch lines in 20 files; 61 overlay files, 18,423 lines | Manual rebase, `make generate`, `make upgrade-dossier` |
| Interception fingerprints and boundary manifest | 131 intercepts, 132 fingerprinted entries | Generated; a diff needs an approved SHA |
| Toolchain inventories | 25 host-clock references, 10 goroutine creation sites | Hand edits after `make test-toolchain` on each platform |
| Dependency adapters | 15 adapters, 135 SHA-256 literals (125 distinct; 30 are per-platform prepared source sets); 6 target modules absent from the root `go.mod` (downstream cell, fn-107) | `gomadtool adapter-regenerate` dry run, then apply with the approval digest; 14 of 15 adapters (not `modernc.org/libc`), and a moved anchor still needs a person |
| Compatibility packs | 11 packs, 49 rules, 18 module-version pins (the unselected `modernc-libc-xsys-v041` was removed) | `gomadtool compatibility-pack refresh` per platform, then one `generate --approve-review` per request |

Counts were re-measured at `6be0755fe` (fn-113 task 1); the patch and overlay grew after the
2026-10-01 measure, and fn-113 task 3 removed one pack. `gomadtool pin-impact` reports which of
these pins a candidate `go.mod` invalidates before the build rejects them. The bump procedure is
in the [README](tools/gomad3/README.md) and [CLI guide](tools/gomad3/CLI.md#bump-a-dependency).

About 20k production lines have no caller in a workflow, Makefile, or qualification manifest:

| Feature | Production lines (approx.) |
| --- | --- |
| `tools/gomad3sim` and simulation exploration | 11,100 |
| `plan`, `execute-shard`, `merge`, `resume`, `recover` | 3,200 |
| `--guide` corpus, choice-exploration strategy, `minimize` | 3,900 |
| `compare-support` | 500 |

Removing or freezing any of these conflicts with fn-108's rule of reducing size without removing
features and with fn-109's simulation requirements (R11, R12). It needs an explicit owner
decision and has no spec. [fn-113](.flow/specs/fn-113-gomad-reduce-version-pin-maintenance.md)
owns the pin-repair tooling, including one-command adapter regeneration, which cuts the
second-largest recurring cost without removing anything.

### Proposed order

1. Restore a green `gomad3.yml`, starting with `make validate`.
2. Correct the four suite defects above so existing tests run.
3. Build Q1, then use it with Q3 and Q4 on a native linux host to close D12 and restore strict
   linux expectations.
4. Add the soak gate (Q6) and the Q7 fixtures.
5. Deliver Q5 through D26.
6. Decide the fate of the uncalled features, then add layer 3 and layer 4 tests for what stays.

## Out of scope

- New fault injection, partition, crash-restart, and multi-node capabilities through
  `tools/gomad3sim`. Those are the simulation track and cannot host `testcore`.
- Multi-P scheduling, deterministic GC, DPOR, and preemption bounding. These are BUG-7 research
  items.
- Revival of gomad1 or gomad2. [GOMAD_CMP.md](.plans/GOMAD_CMP.md) records why.

## Open risks

- **GC timing** is not controlled and shares the seeded runtime stream. Allocation-heavy suites
  may diverge between repetitions; the open linux and darwin replay findings may be such cases.
- **Latent divergence passes qualification.** The evidence digest excludes heap layout, GC count,
  and stream position, and exact replay forces only run-queue and select choices, so two runs
  can differ internally until a target emits the difference
  ([Q1, Q6](#determinism-gaps)).
- **Spin loops** anywhere in the cluster stall virtual time. Pollers with backoff are fine, but a
  single `for {}` with a non-blocking select is fatal under this runtime.
- **Upstream Go releases** invalidate the patch and the boundary manifest each time; every Go bump
  repeats the port and requalification.
- **Compatibility packs pin exact module versions.** Every dependency bump that touches a packed
  or adapted module invalidates the pack or adapter and reopens the capability closure.
