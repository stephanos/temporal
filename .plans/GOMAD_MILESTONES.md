# Gomad v3: Milestones to a Deterministic Temporal Functional Test

**Plan date:** 2026-09-08 · **Trimmed:** 2026-09-30

## Purpose

This document is the delivery ladder for running Temporal's functional tests (`./tests`, built on
the `testcore` one-box cluster with in-memory SQLite and loopback gRPC) under Gomad v3 so that the
same seed produces the same run and a retained artifact replays byte-exactly.
[GOMAD3_NEXT.md](GOMAD3_NEXT.md) remains the capability roadmap across all four tracks.

Milestones F0–F9 are complete and were removed from this document on 2026-09-30. Their outcomes,
details, and status history are in revision `3e4303807` of this file; manifests and exclusions
that cite a finding as `GOMAD_MILESTONES.md#f3-…` through `#f7-…` refer to the sections of that
revision. What remains here is the open work, the open findings, and the rules that still apply.

## Work tracking

Open work is tracked as flow-next specs under `.flow/specs/`; their tasks and acceptance criteria
(R-IDs) are authoritative, and this document keeps rationale and status. When a spec's scope
changes, change the spec and summarize the change here.

| Milestone | Spec | State |
| --- | --- | --- |
| F10 | `fn-105-gomad-follow-ups-deferred-scope` | backlog; each item with a revival trigger ([GOMAD_FOLLOWUPS.md](GOMAD_FOLLOWUPS.md)) |
| Gaps | `fn-106-gomad-close-the-remaining-tests-gaps` | done 2026-09-30: no exclusions left, the traceback leak fixed, the remaining gaps classified ([GOMAD_GAPS.md](GOMAD_GAPS.md)) |

Work a spec with `/flow-next:work <spec>`; list what is ready with `flowctl ready`.

## Remaining `./tests` gaps

The generated manifest (`tools/gomad3integration/qualification/tests.json`, from
`tests.generator.json`) gives every `./tests` test a disposition. What still keeps the package
short of "any functional test runs deterministically":

- **Transcript-heavy suites (fn-106 `.3`).** The I/O transcript bound is configurable
  (`--io-transcript-bytes`, `io_transcript_bytes`, up to 1 GiB), and no `./tests` test is excluded
  any more. `TestTaskQueueStats_Pri_Suite`, `TestVersioning3QueryFunctionalSuite`, and
  `TestWorkerDeploymentSuite` qualify on seeds 11 and 17 with a 512 MiB transcript,
  `TestVersioning3FunctionalSuite` with the 1 GiB maximum. All four also exceed the 64 MiB
  choice-trace maximum, so they run without a choice trace: that proves same-seed repeatability but
  retains no exact-replay artifact. A larger or streamed choice trace would restore replay for
  them.
- **One forward-tick skip.** The seven suites that had timestamp-tie skips run under
  `clock_tick: forward` (fn-103, closed 2026-09-30) and qualify on both seeds with those eighteen
  skips removed. `TestStandaloneActivityTestSuite/TestStartDelay/UpdateWhilePaused_AfterWindow_ExtendsDispatch`
  is skipped instead: under `forward` a propagated gRPC deadline lands microseconds later on the
  server than on the client, so its 3 s long poll hits the client deadline first on seed 11. The
  default tick policy stays `strict`.
- **Classified skips (fn-106 `.4`, `.5`).**
  - `TestNexusOTELSuite/TestOperation` runs with `TEMPORAL_TEST_DEDICATED_CLUSTERS=2` instead of
    waiting on itself, but two dedicated clusters in one process are not yet deterministic (seed 17
    nondeterministic, a seed 11 replay differed in stderr), so it stays skipped with that finding.
  - `TestWorkerCommandsTaskSuite/TestDispatchCancelOnWorkflowTimeout`: on seed 11 the cancel
    command does not arrive before the test's 90 s default context runs below the 2 s long-poll
    minimum, after which every poll is refused; its 120 s wait exceeds its own context.
  - The SDK panic-traceback address leak is fixed (the runtime prints a possibly-dead argument slot
    as `?` while Gomad is enabled) and `TestWFTFailureReportedProblemsTestSuite` runs in full.
- **Tests that assume wall time passes during server work.**
  `TestFairness{,AutoEnable}Suite/Test_Activity_Basic` (backlog written before polling) and
  `TestWorkflowTaskTestSuite/TestWorkflowTaskHeartbeatingWithEmptyResult` (timeouts driven by RPC
  latency). Under virtual time server work takes no time; `forward` does not change timers, so
  these stay skipped unless the tests stop relying on wall-clock latency.
- **Test bugs, kept as owned skips.** The Nexus API `…Operation_Outcomes` subtests register one
  endpoint name from parallel subtests (four skips); `TestScheduleMigrationV2ToV1Idempotent`
  expects idempotency after the migration closed; `TestNexusOperationSurvivesResetCrossTree`
  signals before the post-reset workflow task completes.
- **Intermittent suite.** `TestSignalWorkflowTestSuiteChasm` is `intermittent` on darwin for a
  residual replay difference (about 1 in 28 seed-11 replays, two heap-span refills swapping order
  at cluster start); its allocating goroutine is not identified.

## Open findings

- **Linux replay divergence (F10 D12).** About one tier-3 seed-run in 26 on linux/amd64 is
  nondeterministic or diverges on replay, on either seed and a different suite each run, at choice
  ordinals from 8 to ~85k. A fork bisect (fn-106 `.1`) showed it is not caused by the FIPS DRBG
  draw (`6bc11ef7d`) or the mark-start greying (`440552d2c`): runs with either reverted still
  diverged. A Rosetta linux/amd64 container reproduced it once in 32 seed-runs and never in 100
  sequential replays of the same artifact, so it depends on host timing under load. The F5 and F6
  suites are `intermittent` on linux, and both the dispatch-only linux gate and the required smoke
  gate accept `nondeterministic` and `replay_divergence` for them while failing on target
  failures, unsupported targets, and infrastructure errors. The darwin representative set stays
  fully qualified.
- **Host-clock escapes** recorded by the static inventory (`toolchain/clock_inventory_test.go`):
  `gcMarkTermination` stamps `MemStats.LastGC` with host wall time (also visible through
  `debug.GCStats` and the Prometheus `go_memstats_last_gc_time_seconds` gauge), the FIPS entropy
  source's `monoTime`, and the execution tracer's clock snapshot; on linux also
  `syscall.Gettimeofday` behind the `syscall` pack gate. `LastGC` is the one ordinary targets
  reach. Classified, not fixed (fn-106 `.2`): the patch policy keeps every `mgc*` and `mstats*`
  runtime file out of the patch, `time_now` is platform assembly on linux/amd64, and the field is
  written but never read by runtime control flow, so it reaches evidence only if a target prints
  it.
- **`TestDescribeTaskQueueEnhanced_ReportFlags`** (versioning suite) keeps a deterministic failure
  ("poller info should not be reported") that is not yet shown to be a test bug.
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
  [GOMAD3_NEXT_COMPATIBILITY.md](GOMAD3_NEXT_COMPATIBILITY.md).
- **Evidence over narration.** Work is done when its command produces the stated report on a
  clean checkout. A passing local run that depends on untracked state does not count.
- **Platform.** The boundary manifest qualifies `darwin/arm64` and `linux/amd64`. Each platform
  is its own qualification and artifacts replay only where they were produced. The macOS sandbox
  test and the DTrace clock audit remain `darwin/arm64` only. Each platform's compatibility packs
  are its own; `compatibility-pack-qualification` qualifies the requests that name the host.
- **Server source changes are allowed but bounded.** A change under `common`, `service`,
  `temporal`, or `tests/testcore` is acceptable when it isolates an optional provider behind a
  build tag or an injection seam and the default build is unchanged. A change that alters
  runtime behavior for production builds needs its own review outside this plan.
- **Validation scope.** The full `./tests` set is not run as a gate; a change is validated on the
  smoke selection plus the suites it affects, with `make gomad3-tests-qualification` as the
  on-demand local run.

## F10: follow-ups (deferred scope)

**Spec.** [fn-105-gomad-follow-ups-deferred-scope](../.flow/specs/fn-105-gomad-follow-ups-deferred-scope.md);
items and revival triggers in [GOMAD_FOLLOWUPS.md](GOMAD_FOLLOWUPS.md).

**Outcome.** None required. F10 is a backlog, not a milestone to finish: it keeps the scope cut on
2026-09-29 in one place, each item with its origin, why it was deferred, and the observation that
would revive it.

| Item | Origin | Revive when |
| --- | --- | --- |
| D1–D5 architecture consolidation | F8 R2–R6 | a second consumer or exploration strategy hits the duplication |
| D6 `seeded` and `fixed` tick policies | `fn-103` | a bug class needs deliberate ties or constant quanta |
| D7 macOS smoke job | F7 `.4` | a darwin-only regression escapes to main |
| D8 closure-mode downstream support | F9 C3/R2 | a downstream module needs closure-mode preparation |
| D9 linux/amd64 downstream packs | F9 | a downstream gate must run in linux CI |
| D10 downstream-seam guide | F9 R4 | a second downstream module adopts Gomad |
| D11 dynamic linux clock audit | F7 R5 (pre-amendment) | a linux-only clock escape is observed |
| D12 linux replay-divergence channel | fn-106 `.1` | a linux/amd64 host is available for runtime instrumentation, or the rate rises |

**Constraints.** An item is worked only after its trigger is recorded here; its acceptance is the
origin spec's requirement text. Items may be closed as won't-do.

**Status.** Created on 2026-09-29 with eleven tasks; nothing started.

## Out of scope

- New fault injection, partition, crash-restart, and multi-node capabilities through
  `tools/gomad3sim`. Those are the simulation track and cannot host `testcore`.
- Multi-P scheduling, deterministic GC, DPOR, and preemption bounding. These are BUG-7 research
  items.
- Revival of gomad1 or gomad2. [GOMAD_CMP.md](GOMAD_CMP.md) records why.

## Open risks

- **GC timing** is not controlled and shares the seeded runtime stream. Allocation-heavy suites
  may diverge between repetitions; the open linux seed-17 finding may be one.
- **Spin loops** anywhere in the cluster stall virtual time. Pollers with backoff are fine, but a
  single `for {}` with a non-blocking select is fatal under this runtime.
- **Upstream Go releases** invalidate the patch and the boundary manifest each time; every Go bump
  repeats the port and requalification.
- **Compatibility packs pin exact module versions.** Every dependency bump that touches a packed
  or adapted module invalidates the pack or adapter and reopens the capability closure.
