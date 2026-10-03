# Gomad v3: Milestones to a Deterministic Temporal Functional Test

**Plan date:** 2026-09-08 · **Consolidated:** 2026-09-30

## Purpose

This document is the delivery ladder for running Temporal's functional tests (`./tests`, built on
the `testcore` one-box cluster with in-memory SQLite and loopback gRPC) under Gomad v3 so that the
same seed produces the same run and a retained artifact replays byte-exactly.
[GOMAD_NEXT.md](.plans/GOMAD_NEXT.md) remains the capability roadmap across all four tracks.

This document keeps only remaining work and the constraints needed to deliver it.
Completed work and its acceptance evidence remain in `.flow/tasks/`,
`.flow/artifacts/`, and Git history. Open scope spans F10, downstream delivery,
code-size qualification, deep modules, runtime patch minimization, determinism
assurance, pin-maintenance qualification, and search-path qualification.

## Work tracking

Open work is tracked as flow-next specs under `.flow/specs/`; their tasks and acceptance criteria
(R-IDs) are authoritative, and this document keeps rationale and status. When a spec's scope
changes, change the spec and summarize the change here.

| Milestone | Spec | Remaining work |
| --- | --- | --- |
| F10 | [fn-105](.flow/specs/fn-105-gomad-follow-ups-deferred-scope.md) | D12 Linux replay fix, D26 forward clock, D27 host-clock remedies, D4/D5 architecture, downstream D8–D10, and deferred D6/D11/D15 |
| Downstream cell | [fn-107](.flow/specs/fn-107-gomad-finish-downstream-cell.md) | Consumer implementation and exact replay on both platforms; blocked because the `../downstream` checkout is absent |
| Code-size cleanup | [fn-108](.flow/specs/fn-108-gomad-reduce-code-size-without-removing.md) | Task 8: native linux/amd64 qualification for R9 |
| Deep modules and tool interfaces | [fn-109](.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md) | Tasks 10–21 retain installation/capability ownership, resources/protocols, simulation interfaces, architecture and final qualification |
| Runtime patch minimization | [fn-110](.flow/specs/fn-110-gomad-minimize-the-runtime-patch.md) | Tasks 2–5: overlay extraction, canonical one-context-line regeneration, and both-platform qualification |
| Determinism assurance and test strategy | [fn-112](.flow/specs/fn-112-gomad-determinism-assurance-and-test.md) | Tasks 9–10: suite consolidation, scheduled soak and final documentation; native Linux evidence remains outstanding |
| Version-pin maintenance | [fn-113](.flow/specs/fn-113-gomad-reduce-version-pin-maintenance.md) | Task 4: native Linux gates and final both-host acceptance |
| Search-path qualification | [fn-114](.flow/specs/fn-114-gomad-correct-search-path-defects-and.md) | Task 14 and R12: qualify the combined candidate on native linux/amd64 |

Work a spec with `/flow-next:work <spec>`; list what is ready with `flowctl ready`.

## Immediate delivery order

Milestone execution pauses here at the user's request.
On resumption, continue with fn-109 task 10 and the delivery order below; do not
reimplement work already recorded done in Flow.

Native linux/amd64 evidence remains required for fn-105 D12, fn-108.8,
fn-109.21, fn-113.4 and fn-114.14. Darwin results cannot substitute for it.
The combined runtime candidate is `2ecdbd33`; qualification reports bind exact
source and toolchain identities, so later changes require affected checks.
The collector-file patch prohibition remains a blocker for the unapproved
collector remedies. The downstream checkout is a separate prerequisite.

Reported `gomad-next`, `gomad-next-b` and `gomad-next-c` work is unavailable in
this checkout or the fetched fork. It supplies no integration or completion
proof; use the current sources and open task acceptance. Existing fork CI does
not qualify the current uncommitted sources. Reviews use the configured
`codex:gpt-6-sol:high` pin in fresh contexts, with the same-family limitation.

## Verification instructions for agents

Apply this workflow to Gomad milestone work. Optimize elapsed time by choosing checks that
cover the changed behavior and reusing valid results from the same source revision.

Be wary of overthinking: follow the existing acceptance criteria, choose a grounded
recommendation, and move to implementation. Revisit a decision only when new evidence warrants it.

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

## Search-path findings (fn-114)

[Task 14](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.14.md) and
R12 retain combined-candidate qualification on native linux/amd64. Preserve the
current workload dispositions and self-contained retained artifacts. Darwin
qualification does not close this gate. The collector-file blocker remains
recorded under fn-112.5. Typed scenario shrinking remains open on the roadmap.

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
  fully qualified. The two untested candidates remain in
  [determinism gaps](#determinism-gaps). Use the existing diagnostic tooling
  on a native Linux reproducer.
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
- **Downstream cell.** The consumer must cut its own seams and inject its filesystem
  and membership transport before its in-process cluster test can clear the capability blocker ([GOMAD_CLOUD.md](.plans/GOMAD_CLOUD.md)).

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

**Remaining scope.** Eleven Flow tasks remain: D4–D6, D8–D12, D15, D26 and D27.
D12 requires a native linux/amd64 host. D26 owns the forward-clock correction and
D27 the host-clock remedies. D4/D5 are owned by fn-109; D8–D10 by fn-107.
D6/D15 remain deferred, and D11 retains its recorded revival conditions.

### Required work

| Item | Scope and origin | Decision |
| --- | --- | --- |
| D12 | Identify and fix the linux/amd64 replay-divergence channel (`fn-106.1`); about one tier-3 seed-run in 26 diverged, and the recorded evidence points to host timing under load | Must be fixed, decided 2026-09-30. Native Linux instrumentation is an execution prerequisite. R12 in fn-105 requires a regression reproducer, repeated exact replay on both seeds under load, and restoration of strict CI expectations; diagnosis alone cannot close the task. |
| D26 | Correct the forward clock found by D16 ([task](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.31.md)) | Selected 2026-10-01, not started. Put the forward draws on the virtual clock so `time.Now` and timers share one clock, as fn-103 specified. The fn-103 simulation time transport failure must be reproduced and resolved first, and feasibility is not established; an infeasible result returns the D16 report's fallbacks for a decision. Acceptance in fn-105 R26 adds a conformance fixture, removes the D16 skip, and requalifies every forward workload with traced exact replay. |
| D27 | Apply the D21 host-clock escape remedies ([task](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.32.md)) | Selected 2026-10-01, not started. State the escapes in the contract and pin the darwin `gettimeofday` path and `cputicks` with fixtures. The stamp overwrite from `runtime/proc.go` waits for the patch-policy owner's recorded approval; without it the stamps stay a stated limitation. Acceptance is fn-105 R27. |

### Architecture consolidation

| Item | Scope and origin | Revival trigger |
| --- | --- | --- |
| D4 | Architecture fitness checks for ownership, host effects, and public signatures, with negative fixtures (`fn-102` R5) | Revived by the 2026-09-30 architecture request; fn-109 R8 owns delivery |
| D5 | Architecture/platform/determinism documentation reconciliation (`fn-102` R6) | Revived by the 2026-09-30 architecture request; fn-109 R9 owns delivery |

These were deferred as maintenance without a waiting consumer or behavior change.
The [architecture assessment](.flow/artifacts/fn-102-gomad-architecture-consolidate/architecture-assessment.md)
retains the evidence. Preserve CLI behavior, schemas, canonical bytes, failure
classification and precedence, and replay compatibility when reviving them.

D4/D5 reuse existing accepted evidence where it still matches the source, then
verify the new owners and interfaces delivered by fn-109. Preserve the CLI,
schemas, canonical bytes and failure/replay contracts during these migrations.

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

[Task 8](.flow/tasks/fn-108-gomad-reduce-code-size-without-removing.8.md) owns
native linux/amd64 gates for R9. Its required commands and source-bound baseline
are in [final.md](.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final.md).
Do not widen qualification based on Darwin evidence. Root `make lint-code-fast`
retains the nested-module discovery limitation; use the scoped nested-module
linter for code changed there.

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
| 3. Installation and capability ownership | Validated installation descriptions; capability collection, pure evaluation and linked projection | R6, R15, R17 |
| 4. Resources and protocols | Separate Artifact references/opened handles, generate simulation-time consumers and hide model-wire slots behind typed commands | R7, R13, R14 |
| 5. Simulation interfaces | Compare progress interfaces, implement the lifecycle owner and select backend-specific network/filesystem handles at creation | R11, R12 |
| 6. Architecture and qualification | Enforce ownership/effect/signature checks, reconcile guidance and retain complete finding/evidence coverage | R8, R9, R18–R20 |

D4/D5 remain owned by R8/R9. Coordinate shared runtime-overlay and
source-inventory edits with fn-110; patch minimization retains its own scope.

**Preservation contract.** Keep CLI grammar/defaults, recorded formats, canonical
bytes for fixed identities, error precedence, existing comments, fresh processes,
native timer ownership and separate seed/round/corpus transactions. Migrate the
Artifact-handle interface with explicit ownership and lifetime. D12 retains its
separate owner; unavailable hosts or unexplained regressions leave affected
acceptance incomplete.

**Remaining scope.** Tasks 10–21 remain. Native Linux
qualification is part of task 21's final gate. Reuse the accepted task records
for dependencies; do not repeat their implementations.

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
initialization, and interception redesign remain outside this spec. D12 fixes
retain their separate owner; patch minimization cannot change their dispositions
to obtain a passing gate.

**Remaining scope.** Tasks 2–5. Re-anchor the task 1 baseline (`8d28bd44`)
to the current inputs before extraction; retain original measurements and
workload dispositions in the task evidence.

## Quality assessment (2026-10-01)

**Scope.** A read-only assessment of determinism soundness, the test suite, and maintenance
cost. Three audits read the source, manifests, workflows, and CI history. Nothing was built or
run, so every runtime finding below comes from source reading and is unverified by execution
unless it cites an existing report.

**Remaining scope.** [fn-112](.flow/specs/fn-112-gomad-determinism-assurance-and-test.md)
tasks 9–10 own suite consolidation, the scheduled soak and documentation of the
remaining host-channel and closure-mode limits. Q3/Q4 remain candidates under
fn-105 D12; Q5 is D26. Native Linux qualification remains outstanding.

The target is a checked invariant plus a measured divergence bound under load.
Existing diagnostic traces can localize a fresh-run difference; applying them
to D12 still requires a native Linux reproducer. Collector and real-thread
limitations prevent an unconditional guarantee.

### Determinism gaps

| # | Gap | Evidence | Proposed direction |
| --- | --- | --- | --- |
| Q3 | D12 candidate: one-shot syscall wait in `suspendG` | The patch waits for a host syscall to exit once, before the suspend loop. A goroutine that enters a syscall in that window can be scanned mid-syscall, which shifts scan work. Inferred from source, not reproduced | Wait inside the loop's `_Gsyscall` case, then measure D12 on a native linux host |
| Q4 | D12 candidate: linux ASLR and `runtime.NumCPU` | Neither is controlled on linux/amd64. Impact is inferred low because the binary is non-PIE | Rerun the D12 reproducer under `setarch -R` before any code change |
| Q5 | Forward clock runs `time.Now` and timers on different clocks | D16: the offset only grows, so derived deadlines fire late and the subtest fails 5 of 24 seeds. Eight generated workloads use `forward` | D26: one shared clock as fn-103 specified, feasibility-gated |
| Q6 | Qualification has too little statistical power | `qualify` repeats twice. Exact replay forces run-queue and select choices, which masks perturbations that would split two fresh runs. A 1-in-26 defect passes most two-repetition checks | A scheduled soak of N fresh repetitions under CPU load with a strict zero-divergence gate |
| Q7 | Remaining host-channel limits | Host netpoll readiness, SIGPROF, enabled block/mutex profiling, and unvirtualized `NumCPU` | Publish the explicit limits in README/SPEC under task 10; any added support needs a seeded fixture and positive control |
| Q8 | Closure mode compiles no guards | `-gomadguard` is added only in guarded mode; pack admissions of `syscall` are per package, so admitted code runs live | State the limit in the contract, or sample guarded mode in the soak |

Q3/Q4 are inferred candidates, not reproduced causes. Pursuing them must satisfy
fn-105 R12, including repeated exact replay on both seeds under native Linux
load and strict CI expectations. The collector-file prohibition stays in force.

### Testing strategy

The remaining work is a scheduled determinism soak and consolidation of tests
that mainly detect structural changes. Retain runtime/model conformance and
real CLI recovery coverage while reshaping them. New tests should cover a
concrete behavior or failure, with a positive control when relevant.

`TestProcessBackendSynchronizesNodeClockWithModelDelay` remains outside
`test-simulation`, with its watchdog failure retained as a finding. The expanded
Linux gates still need native execution evidence.

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

Exact-version pins remain recurring maintenance costs. The current inventory
has 15 dependency adapters and 135 SHA-256 anchor literals, plus 11 compatibility
packs, 49 rules and 18 distinct module-version pins. These counts are source-bound;
re-measure them after pin or adapter changes.

[fn-113 task 4](.flow/tasks/fn-113-gomad-reduce-version-pin-maintenance.4.md)
retains native Linux qualification and final both-host acceptance. The procedure,
measured walks, historical baseline and approval evidence remain under
`.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/`. Do not infer a
Linux result from the Darwin walks or substitute projected command savings for
measured acceptance.

About 20k production lines have no caller in a workflow, Makefile, or qualification manifest:

| Feature | Production lines (approx.) |
| --- | --- |
| `tools/gomad3sim` and simulation exploration | 11,100 |
| `plan`, `execute-shard`, `merge`, `resume`, `recover` | 3,200 |
| `--guide` corpus, choice-exploration strategy, `minimize` | 3,900 |
| `compare-support` | 500 |

Removing or freezing any of these conflicts with fn-108's rule of reducing size without removing
features and with fn-109's simulation requirements (R11, R12). It needs an explicit owner
decision and has no implementation spec.

### Proposed order

1. Use the existing diagnostic tooling with Q3/Q4 on native linux/amd64 to close
   D12 and restore strict Linux expectations.
2. Consolidate the suite and add the loaded fresh-run soak through fn-112.9–10;
   publish Q7/Q8's remaining limits.
3. Deliver Q5 through D26 after resolving its simulation-time prerequisite.
4. Complete native Linux qualification for fn-108, fn-113 and fn-114.

Removing or freezing shipped features needs an explicit owner decision; the
current preservation specs authorize no such removal.

## Out of scope

- New fault injection, partition, crash-restart, and multi-node capabilities through
  `tools/gomad3sim`. Those are the simulation track and cannot host `testcore`.
- Multi-P scheduling, deterministic GC, DPOR, and preemption bounding. These are BUG-7 research
  items.
- Revival of gomad1 or gomad2. [GOMAD_CMP.md](.plans/GOMAD_CMP.md) records why.

## Open risks

- **GC timing** is not controlled. Allocation-heavy suites may diverge between
  repetitions; the remaining Linux replay finding needs a reproduced cause.
  Collector-file remedies remain outside the permitted patch policy.
- **Latent divergence passes qualification.** Routine qualification has only two
  repetitions, and exact replay forces choices that can mask fresh-run differences.
  Diagnostic traces help localization; the loaded soak remains open
  ([Q6](#determinism-gaps)).
- **Spin loops** anywhere in the cluster stall virtual time. Pollers with backoff are fine, but a
  single `for {}` with a non-blocking select is fatal under this runtime.
- **Upstream Go releases** invalidate the patch and the boundary manifest each time; every Go bump
  repeats the port and requalification.
- **Compatibility packs pin exact module versions.** Every dependency bump that touches a packed
  or adapted module invalidates the pack or adapter and reopens the capability closure.
