# Gomad v3: Milestones to a Deterministic Temporal Functional Test

**Plan date:** 2026-09-08 · **Consolidated:** 2026-10-03

## Purpose

This document is the delivery ladder for running Temporal's functional tests (`./tests`, built on
the `testcore` one-box cluster with in-memory SQLite and loopback gRPC) under Gomad v3 so that the
same seed produces the same run and a retained artifact replays byte-exactly.
[GOMAD_NEXT.md](.plans/GOMAD_NEXT.md) remains the capability roadmap across all four tracks.

Completed milestones and task outcomes are intentionally absent from this delivery document;
their source-bound summaries and evidence remain in Flow task receipts and artifacts. This file
retains the current blockers, delivery order, constraints, unresolved findings, and rationale
needed by open work.

## Work tracking

Open work is tracked as flow-next specs under `.flow/specs/`; their tasks and acceptance criteria
(R-IDs) are authoritative, and this document keeps rationale and status. When a spec's scope
changes, change the spec and summarize the change here.

| Milestone | Spec | State |
| --- | --- | --- |
| F10 | `fn-105-gomad-follow-ups-deferred-scope` | open; 12 tasks remain: required D12, working-tree D26 awaiting qualification, merged-but-unqualified D27, architecture/downstream D3–D5 and D8–D10, and deferred D6, D11, and D15 ([decisions and scope](#f10-follow-ups-deferred-scope)) |
| Downstream cell | [fn-107](.flow/specs/fn-107-gomad-finish-downstream-cell.md) | open; only task 5 remains, owning both-platform qualification and completion, but this machine lacks both the downstream checkout and qualified hosts |
| Code-size cleanup | [fn-108](.flow/specs/fn-108-gomad-reduce-code-size-without-removing.md) | open; only task 8 remains, owning the linux/amd64 gates for R9 ([status](#code-size-cleanup-fn-108)) |
| Deep modules and tool interfaces | [fn-109](.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md) | open; tasks 2–6 are merged but remain open pending acceptance and native qualification; tasks 7–21 remain ([delivery order](#deep-modules-and-tool-interfaces-fn-109)) |
| Runtime patch minimization | [fn-110](.flow/specs/fn-110-gomad-minimize-the-runtime-patch.md) | open; task 2 is implemented in the working tree but blocked on native qualification; tasks 3–5 remain ([delivery order](#runtime-patch-minimization-fn-110)) |
| Determinism assurance and test strategy | [fn-112](.flow/specs/fn-112-gomad-determinism-assurance-and-test.md) | open; tasks 5, 9, and 16 are merged but remain open pending qualification, and task 10's soak and contract documentation remain unimplemented ([findings and proposed order](#quality-assessment-2026-10-01)) |
| Version-pin maintenance | [fn-113](.flow/specs/fn-113-gomad-reduce-version-pin-maintenance.md) | open; tasks 1–4 are merged but none is complete because R6 still requires qualified darwin/arm64 and linux/amd64 evidence ([cost evidence](#maintenance-cost)) |
| Search-path defects and wasted work | [fn-114](.flow/specs/fn-114-gomad-correct-search-path-defects-and.md) | open; task 13 is merged but remains open pending native qualification, and task 14 owns combined qualification and documentation |

Work a spec with `/flow-next:work <spec>`; list what is ready with `flowctl ready`.

## Immediate delivery order

The 2026-10-03 side-branch commits and the D27 candidate are merged into `gomad`. This linux/arm64
host cannot supply qualified evidence and has no usable CI credentials. D26's reviewed candidate
is integrated with fn-110's runtime and generated-file changes in the uncommitted working tree;
all linux/arm64 implementation evidence remains developmental until the native gates run.

1. Qualify fn-114 task 13 and fn-112 task 5 on darwin/arm64 and linux/amd64, then run fn-114 task 14.
2. Qualify merged fn-112 tasks 16 and 9, fn-113 tasks 1–4, and fn-109 tasks 2–6; do not close them from the development harness.
3. Finish verification of the combined D26/fn-110 working-tree candidate, then run the required native qualification for D26 and D27; continue D12 investigation with the available pre-native instrumentation.
4. Complete fn-110 task 2's native gates, then continue fn-110 tasks 3–5.
5. Run fn-108 task 8 and the remaining platform-only gates whenever a qualified host or CI becomes available.

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
  D12 still accept a seed that diverged. Routine runs do not establish choice-tape replay.
- **No exact replay available for eight suites.** Their choice traces overflow the 64 MiB
  maximum, so they cannot opt into tracing until larger traces exist (deferred as D15).
  `TestTaskQueueStats_Pri_Suite`, `TestVersioning3FunctionalSuite`,
  `TestVersioning3QueryFunctionalSuite`, and `TestWorkerDeploymentSuite` also run with a raised
  I/O transcript; `TestClientMiscTestSuite`, `TestScheduleV1`, and
  `TestScheduleV1WorkflowPauseInteraction` run with the default transcript.
  `TestVersioningFunctionalSuite` is expected `qualified` on the same seed-repeatability basis.
- **Forward-clock correction awaiting qualification.**
  - `TestStandaloneActivityTestSuite/TestStartDelay/UpdateWhilePaused_AfterWindow_ExtendsDispatch`:
    under `clock_tick: forward` the lead of `time.Now` over the timer clock grows with every read
    (about 0.33 s when this subtest polls), and each deadline re-derived from `time.Now` fires
    later by that lead. The frontend hop, the matching hop, and matching's child context add
    three leads, which use up the 1 s empty-response budget, so the client's 3 s deadline fires
    first (seeds 5, 12, 20, 21, 24 of 1 to 24 on toolchain `8d28bd44`; seed 11 now passes with
    about 20 ms to spare). The
    [D16](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.16.md) investigation
    ([report](docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md)) assigns the
    correction to the Gomad forward clock. The integrated D26 working-tree candidate removes
    the skip; its seeded native qualification is still required before acceptance.
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
  fully qualified. Existing diagnostic-localisation tooling still needs to be applied to fresh
  loaded cohorts on native linux/amd64 before either remaining candidate is changed
  ([determinism gaps](#determinism-gaps)).
- **Host-clock escapes (F10 D27).** Reporting and profiling surfaces still expose host-derived
  time or cycle counts, including GC statistics, execution traces, FIPS entropy input, and raw
  clock syscalls admitted by an exact compatibility pack. The Temporal functional closure has
  not been shown to consume these values as control inputs, so D27 does not attribute D12 to
  them. The D27 candidate pins the darwin `gettimeofday` path and `cputicks` in the static
  inventory and publishes the reporting limitation without altering collector-owned fields,
  prohibited collector files, or assembly. R27 remains open until both native toolchain gates
  pass. The source basis and earlier investigation remain in the
  [D21 report](docs/research/gomad/GOMAD_HOST_CLOCK_ESCAPES.md).
- **DTrace clock audit** on darwin needs a root run; CI supplies it on the macOS runner.
- **Downstream cell.** Task 5 cannot qualify the consumer on this machine because the downstream
  checkout and qualified native hosts are unavailable. The in-process cluster test remains a
  capability blocker until the downstream cuts its seams and injects its filesystem and
  membership transport ([GOMAD_CLOUD.md](.plans/GOMAD_CLOUD.md)).

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

**Status.** Twelve items remain. D12 is the required native-linux fix; D26 is integrated with
fn-110 in the uncommitted working tree but awaits qualification; D27 is merged but remains
unqualified. D3–D5 and D8–D10 remain assigned to fn-109 and fn-107. D6 clock policies, D11 dynamic
Linux clock auditing, and D15 larger traces remain explicitly deferred.

### Required work

| Item | Scope and origin | Decision |
| --- | --- | --- |
| D12 | Identify and fix the linux/amd64 replay-divergence channel (`fn-106.1`); about one tier-3 seed-run in 26 diverged, and the recorded evidence points to host timing under load | Must be fixed, decided 2026-09-30. Native Linux instrumentation is an execution prerequisite. R12 in fn-105 requires a regression reproducer, repeated exact replay on both seeds under load, and restoration of strict CI expectations; diagnosis alone cannot close the task. |
| D26 | Correct the forward clock found by D16 ([task](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.31.md)) | Reviewed candidate integrated with fn-110 in the uncommitted working tree. It shares forward draws with timer time, reconciles activation and model-transport epochs without rewind, adds conformance and process regressions, and removes the D16 skip. Combined runtime verification and both native qualification gates remain; linux/arm64 results are developmental only. |
| D27 | Apply the D21 host-clock escape remedies ([task](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.32.md)) | Candidate merged. The contract states the reporting escapes and fixtures pin the darwin `gettimeofday` path and `cputicks`; collector-owned fields, prohibited collector files, and assembly remain unchanged. Native darwin/arm64 and linux/amd64 toolchain gates still block R27. |

### Architecture consolidation

| Item | Scope and origin | Revival trigger |
| --- | --- | --- |
| D3 | Private executor injection instead of public `Executor`/`ReplayExecutor` (`fn-102` R4); a Go API change | Revived by the 2026-09-30 architecture request; fn-109 R5 owns delivery |
| D4 | Architecture fitness checks for ownership, host effects, and public signatures, with negative fixtures (`fn-102` R5) | Revived by the 2026-09-30 architecture request; fn-109 R8 owns delivery |
| D5 | Architecture/platform/determinism documentation reconciliation (`fn-102` R6) | Revived by the 2026-09-30 architecture request; fn-109 R9 owns delivery |

The [architecture assessment](.flow/artifacts/fn-102-gomad-architecture-consolidate/architecture-assessment.md)
retains the evidence. fn-109 owns D3–D5 and must preserve CLI behavior, schemas, canonical
bytes, failure classification and precedence, replay compatibility, and the shared policy
extraction already delivered by fn-108.

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
owns the remaining requirement. Only [task 8](.flow/tasks/fn-108-gomad-reduce-code-size-without-removing.8.md)
is open: run the outstanding linux/amd64 gates for R9. The darwin evidence and exact commands to
repeat are retained in
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
| 1. Common policy | Reuse fn-108's assessment and retention extraction; give the pure seed controller one completion transition | R2, R3, R16; reuse or transfer D1/D2 obligations once, with fn-108 R6/R7 evidence |
| 2. Operation and preparation ownership | Narrow request/private construction, complete preparation, target commands, installation descriptions and capability evaluation | R4–R6, R10, R15, R17; overlapping public migrations follow verified fn-108 extraction |
| 3. Resources and protocols | Separate Artifact references/opened handles, generate simulation-time consumers and hide model-wire slots behind typed commands | R7, R13, R14; preserve ownership, bytes, validation and domain errors |
| 4. Simulation interfaces | Compare progress interfaces, implement the lifecycle owner and select backend-specific network/filesystem handles at creation | R11, R12; preserve concurrent blocking work, stale-incarnation rejection and backend fidelity |
| 5. Architecture and qualification | Enforce ownership/effect/signature checks, reconcile guidance and retain complete finding/evidence coverage | R8, R9, R18–R20; negative fixtures, fixed-identity equivalence and both-platform gates |

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

**Status.** Tasks 2–6 are merged but remain open pending their acceptance
checks and native qualification. Tasks 7–21 remain unimplemented.

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
| 1. Overlay extraction | Move the three scheduler/quiescence implementations, crypto initialization, and syscall linkname declarations; retain upstream hooks and lifecycle points | R2, R3, R6; preserve lock state, ordering, entropy overrides, activation, and negative behavior |
| 2. Canonical regeneration | Emit `-U1`, update exact descriptor source sets and generated consumers, and make the pinned regeneration check follow the descriptor | R4, R5; repeated byte-identical regeneration and zero-fuzz source equivalence between final `-U3` and `-U1` patches on both hosts |
| 3. Qualification and final evidence | Run full Gomad gates on both platforms, affected entropy/process-simulation tests, Temporal integration/smoke and affected qualification; publish final counts and build identities | R7, R8; fresh repeatability and exact replay under existing contracts, with no weakened dispositions |

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

**Status.** Task 2's scheduler/quiescence extraction is implemented in the
working tree and passes local generation, validation, and focused conformance checks, but it
remains blocked on required native darwin/arm64 and linux/amd64 gates. Tasks 3–5 remain.

## Quality assessment (2026-10-01)

**Scope.** This section retains only the unresolved determinism, test-strategy, and maintenance
findings from the 2026-10-01 assessment. Flow tasks and their artifacts own completed outcomes.

**Spec.** [fn-112](.flow/specs/fn-112-gomad-determinism-assurance-and-test.md) has four open tasks.
Tasks 5, 9, and 16 are merged but remain open pending qualification; task 9 also
defers overlapping CLI and Runner work until fn-109 lands. Task 10's soak gate and final contract
documentation remain unimplemented. Q3 and Q4 remain D12 candidates under fn-105, and Q5 is D26.

**Current verdict.** Determinism assurance remains incomplete until the stream-isolation,
retained-success collision, and suite-consolidation candidates pass their native gates; D12 gains
causal native-linux evidence; and the soak and contract work supplies a measured bound and states
the remaining exclusions.

### Determinism gaps

| # | Gap | Evidence | Proposed direction |
| --- | --- | --- | --- |
| Q2 | Stream isolation remains unqualified | Task 5's candidate separates collector and target-visible draws, but its required native gates have not run | Qualify the candidate and its runtime assertion on both supported platforms |
| Q3 | D12 candidate: one-shot syscall wait in `suspendG` | The patch waits for a host syscall to exit once, before the suspend loop. A goroutine that enters a syscall in that window can be scanned mid-syscall, which shifts scan work. Inferred from source, not reproduced | Measure with native diagnostics first; only if a divergence identifies this path, wait inside the loop's `_Gsyscall` case and requalify |
| Q4 | D12 candidate: linux ASLR and `runtime.NumCPU` | Neither is controlled on linux/amd64. Impact is inferred low because the binary is non-PIE | Rerun the D12 reproducer under `setarch -R` before any code change |
| Q5 | Forward shared-clock correction remains unqualified | D16: the original offset made derived deadlines fire late and the subtest fail 5 of 24 seeds. Eight generated workloads use `forward` | Verify the integrated D26/fn-110 candidate and qualify it on both native platforms |
| Q6 | Qualification has too little statistical power | `qualify` repeats twice. Exact replay forces run-queue and select choices, which masks perturbations that would split two fresh runs. A 1-in-26 defect passes most two-repetition checks | A scheduled soak of N fresh repetitions under CPU load with a strict zero-divergence gate |
| Q7 | Channels without a positive control | Netpoll, SIGPROF, enabled block and mutex profiling on linux, and `NumCPU` remain exclusions | Publish each exclusion in task 10 and add a seeded fixture where the final contract claims deterministic behavior |
| Q8 | Closure mode compiles no guards | `-gomadguard` is added only in guarded mode; pack admissions of `syscall` are per package, so admitted code runs live | State the limit in the contract, or sample guarded mode in the soak |

Q3 and Q4 remain hypotheses only. Current one-P reasoning makes Q3 low-likelihood, and the
prepared linux binary is expected to be non-PIE; neither should change production code before a
native diagnostic divergence identifies a causal path. D12 must first compare fresh loaded
cohorts with existing diagnostics, repeat them under `setarch x86_64 -R`, and record CPU topology
and cgroup controls. The remaining Q7 exclusions and Q8 closure-mode limit belong in task 10's
contract documentation.

### Remaining test-strategy work

- Qualify task 5's seeded-stream inventory and runtime check on both native platforms.
- Build task 10's loaded soak gate and publish the final determinism contract and exclusions.
- Qualify task 9's suite consolidation and finish its fn-109-dependent CLI and Runner mapping.
- Qualify task 16's retained-success collision handling.

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

The fn-113 pin-maintenance candidate is merged and exposes impact, adapter regeneration, and
compatibility-pack refresh operations, but none of its four tasks is complete until R6 passes on
qualified darwin/arm64 and linux/amd64 hosts. Until then, the manual repair paths in this table
remain the delivery baseline.

About 20k production lines have no caller in a workflow, Makefile, or qualification manifest:

| Feature | Production lines (approx.) |
| --- | --- |
| `tools/gomad3sim` and simulation exploration | 11,100 |
| `plan`, `execute-shard`, `merge`, `resume`, `recover` | 3,200 |
| `--guide` corpus, choice-exploration strategy, `minimize` | 3,900 |
| `compare-support` | 500 |

Removing or freezing any of these conflicts with fn-108's preservation rule and fn-109's
simulation requirements (R11, R12). It needs an explicit owner decision and has no spec.

### Proposed order

1. Qualify merged fn-112 tasks 5, 9, and 16 on both native platforms.
2. Use the existing diagnostics and task 5's checks with Q3/Q4 controls to close D12 on native
   linux/amd64; do not make a speculative Q3 or `NumCPU` change before a causal divergence.
3. Deliver task 10's loaded soak gate and final contract documentation.
4. Verify the integrated D26/fn-110 candidate and run its native qualification, then decide whether
   any remaining uncalled feature needs a new owner.

## Out of scope

- New fault injection, partition, crash-restart, and multi-node capabilities through
  `tools/gomad3sim`. Those are the simulation track and cannot host `testcore`.
- Multi-P scheduling, deterministic GC, DPOR, and preemption bounding. These are BUG-7 research
  items.
- Revival of gomad1 or gomad2. [GOMAD_CMP.md](.plans/GOMAD_CMP.md) records why.

## Open risks

- **GC timing** is not controlled. Until fn-112 task 5's separated stream passes both native
  gates, allocation-heavy suites may still couple collector timing to target-visible draws.
- **Latent divergence passes qualification.** Routine qualification still has only two fresh
  repetitions, and exact replay forces run-queue and select choices; task 10's loaded soak must
  supply the missing statistical bound ([Q6](#determinism-gaps)).
- **Spin loops** anywhere in the cluster stall virtual time. Pollers with backoff are fine, but a
  single `for {}` with a non-blocking select is fatal under this runtime.
- **Upstream Go releases** invalidate the patch and the boundary manifest each time; every Go bump
  repeats the port and requalification.
- **Compatibility packs pin exact module versions.** Every dependency bump that touches a packed
  or adapted module invalidates the pack or adapter and reopens the capability closure.
