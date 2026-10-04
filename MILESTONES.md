# Gomad v3: Milestones to a Deterministic Temporal Functional Test

**Plan date:** 2026-09-08 · **Consolidated:** 2026-10-03

## Purpose

This document is the delivery ladder for running Temporal's functional tests (`./tests`, built on
the `testcore` one-box cluster with in-memory SQLite and loopback gRPC) under Gomad v3 so that the
same seed produces the same run and a retained artifact replays byte-exactly.
[GOMAD_NEXT.md](.plans/GOMAD_NEXT.md) remains the capability roadmap across all four tracks.

This document keeps only remaining work, delivery order, constraints, and unresolved findings.
Completed work and its source-bound acceptance evidence remain in `.flow/tasks/`,
`.flow/artifacts/`, and Git history. Open scope spans F10, downstream delivery,
code-size qualification, deep modules, runtime patch minimization, determinism
assurance, pin-maintenance qualification, and search-path qualification.

## Work tracking

Open work is tracked as flow-next specs under `.flow/specs/`; their tasks and acceptance criteria
(R-IDs) are authoritative, and this document keeps rationale and status. When a spec's scope
changes, change the spec and summarize the change here.

| Milestone | Spec | Remaining work |
| --- | --- | --- |
| F10 | [fn-105](.flow/specs/fn-105-gomad-follow-ups-deferred-scope.md) | D12 native Linux replay fix; D26 combined runtime candidate and D27 host-clock candidate awaiting native qualification; D3–D5 architecture, downstream D8–D10, and deferred D6/D11/D15 |
| Downstream cell | [fn-107](.flow/specs/fn-107-gomad-finish-downstream-cell.md) | Task 5: consumer and both-platform exact replay; blocked by the absent `../downstream` checkout and qualified hosts |
| Code-size cleanup | [fn-108](.flow/specs/fn-108-gomad-reduce-code-size-without-removing.md) | Task 8: native linux/amd64 qualification for R9 |
| Deep modules and tool interfaces | [fn-109](.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md) | Tasks 2–12 have merged candidates awaiting acceptance; tasks 13–18 have reviewed source candidates awaiting native gates; task 19's committed architecture candidate awaits formal review after a predispatch tool failure; task 20's earlier guidance has formal SHIP and its nine-flag CLI correction is source-reviewed and committed, while inherited D5 acceptance remains open; task 21 retains matched developmental 10/100 campaigns and finding/preservation evidence, with R18 reconciliation and both native gates incomplete; tasks 23–26 retain source-reviewed lint-routing, path-base, controller and Runner-owned CLI semantic repairs; root ordinary and integration lint/vet pass, while the retained 419 nested Gomad findings and 54 inherited CLI findings keep qualification red |
| Runtime patch minimization | [fn-110](.flow/specs/fn-110-gomad-minimize-the-runtime-patch.md) | Tasks 2–4 have merged candidates awaiting native gates; the current final `-U3` exceeds the original baseline, leaving R8's extraction reduction unmet; task 5 retains final evidence and both-platform qualification |
| Determinism assurance and test strategy | [fn-112](.flow/specs/fn-112-gomad-determinism-assurance-and-test.md) | Tasks 5, 9, and 16 have merged candidates awaiting qualification; task 10's soak gate and contract documentation are delivered and await one retained scheduled or dispatched soak run per platform |
| Version-pin maintenance | [fn-113](.flow/specs/fn-113-gomad-reduce-version-pin-maintenance.md) | Tasks 1–4 have merged candidates; final native gates and both-host acceptance remain |
| Search-path qualification | [fn-114](.flow/specs/fn-114-gomad-correct-search-path-defects-and.md) | Task 13 candidate awaits native qualification; task 14 and R12 retain combined-candidate qualification on native linux/amd64 |

Work a spec with `/flow-next:work <spec>`; list what is ready with `flowctl ready`.

## Immediate delivery order

The 2026-10-03 side-branch candidates are integrated. D26's reviewed source candidate is
combined with fn-110's runtime and generated-file changes. This development host cannot supply
qualified `darwin/arm64` or `linux/amd64` evidence; qualification reports bind exact source and
toolchain identities, so pre-merge or pre-integration results do not qualify the combined candidate.
The collector-file patch prohibition and absent downstream checkout remain separate constraints.

1. Verify the combined D26/fn-110 candidate, then qualify fn-114 task 13, fn-112 task 5, D26, D27, and fn-110 task 2 on both native platforms; run fn-114 task 14 against that same candidate.
2. Qualify the merged fn-112 tasks 16 and 9, fn-113 tasks 1–4, and fn-109 tasks 2–6 against the integrated source; retain each task's acceptance checks.
3. Investigate D12 with loaded native linux/amd64 cohorts and the existing diagnostics, then restore strict replay expectations when a causal fix qualifies.
4. Qualify the merged fn-109 tasks 7–12 and fn-110 tasks 3–4, and continue fn-109 tasks 13–21 and fn-110 task 5 in their delivery order. Source implementation may advance after its predecessor candidate is integrated and reviewed; keep acceptance open until its required native gates pass. Complete fn-108 task 8 and other platform-only gates when native hosts or CI are available.
5. Resume fn-107's downstream cell when its checkout is available, and run the final both-platform consumer gates.

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
5. **Commit each task separately.** After its source checks and review pass, commit the task's
   implementation, tests, documentation, and Flow records together before starting the next
   task. This supersedes older
   task instructions reserving commits for the user. If required native gates are unavailable,
   commit the verified progress with those gates recorded as incomplete and keep acceptance
   open. Complete the Flow task only when all its required gates pass. Preserve unrelated
   changes; push only when authorized. Broaden testing only for a concrete remaining risk,
   and retry an unchanged environment failure only when its cause or relevant inputs change.

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
  A fixture that prints `LastGC` can have nondeterministic stdout even when its
  choice tape replays exactly. Linux `cputicks` can steer block and mutex profile
  sampling. Overwriting collector-owned GC stamps still needs the patch-policy
  owner's decision; the collector-file prohibition remains in force.
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

**Remaining scope.** D12 is the required native-linux fix. D26 is integrated with
fn-110 but awaits qualification; D27 is merged but unqualified. D3–D5 and
D8–D10 remain assigned to fn-109 and fn-107. D6 clock policies, D11 dynamic
Linux clock auditing, and D15 larger traces retain their revival conditions.

### Required work

| Item | Scope and origin | Decision |
| --- | --- | --- |
| D12 | Identify and fix the linux/amd64 replay-divergence channel (`fn-106.1`); about one tier-3 seed-run in 26 diverged, and the recorded evidence points to host timing under load | Must be fixed, decided 2026-09-30. Native Linux instrumentation is an execution prerequisite. R12 in fn-105 requires a regression reproducer, repeated exact replay on both seeds under load, and restoration of strict CI expectations; diagnosis alone cannot close the task. |
| D26 | Correct the forward clock found by D16 ([task](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.31.md)) | Reviewed source candidate integrated with fn-110. It shares forward draws with timer time, reconciles activation and model-transport epochs without rewind, adds conformance and process regressions, and removes the D16 skip. Combined runtime verification and both native qualification gates remain; linux/arm64 results are developmental. R26 still requires every forward workload with traced exact replay. |
| D27 | Apply the D21 host-clock escape remedies ([task](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.32.md)) | Candidate merged. The contract states the reporting escapes and fixtures pin the darwin `gettimeofday` path and `cputicks`; collector-owned fields, prohibited collector files, and assembly remain unchanged. Their overwrite requires a separate patch-policy decision. Native darwin/arm64 and linux/amd64 toolchain gates still block R27. |

### Architecture consolidation

| Item | Scope and origin | Revival trigger |
| --- | --- | --- |
| D4 | Architecture fitness checks for ownership, host effects, and public signatures, with negative fixtures (`fn-102` R5) | Revived by the 2026-09-30 architecture request; fn-109 R8 owns delivery |
| D5 | Architecture/platform/determinism documentation reconciliation (`fn-102` R6) | Revived by the 2026-09-30 architecture request; fn-109 R9 owns delivery |

The [architecture assessment](.flow/artifacts/fn-102-gomad-architecture-consolidate/architecture-assessment.md)
retains the evidence. fn-109 owns D3–D5, reuses accepted evidence where it
still matches the integrated source, and must preserve CLI behavior, schemas,
canonical bytes, failure classification and precedence, replay compatibility,
and the shared policy extraction delivered by fn-108.

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
Do not widen qualification based on Darwin evidence. Root `make lint-code-fast`
now routes nested modules through task 23's module-aware selector. Task 25's
controller repair lets ordinary root and tagged integration lint/vet complete;
the automatic nested Gomad scope still fails on 419 findings across 31 owners
([current routing and qualification](.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-25/acceptance-open.md)).
Nested vet and later scopes remain unreached. The historical fn-108 report
retains its original discovery failure; current full/native qualification stays open.

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
| 1. Common policy and operation ownership | Accept the merged fn-108 extraction, seed completion, request construction, and target command work against R2–R5, R10, and R16 | Preserve the exact public and recorded contracts; qualify merged tasks 2–6 before closing them |
| 2. Installation and capability ownership | Validated installation descriptions; capability collection, pure evaluation and linked projection | R6, R15, R17 |
| 3. Resources and protocols | Separate Artifact references/opened handles, generate simulation-time consumers and hide model-wire slots behind typed commands | R7, R13, R14; preserve ownership, bytes, validation and domain errors |
| 4. Simulation interfaces | Compare progress interfaces, implement the lifecycle owner and select backend-specific network/filesystem handles at creation | R11, R12; preserve concurrent blocking work and backend fidelity |
| 5. Architecture and qualification | Enforce ownership/effect/signature checks, reconcile guidance and retain complete finding/evidence coverage | R8, R9, R18–R20; negative fixtures, fixed-identity equivalence and both-platform gates |

D4/D5 remain owned by R8/R9. Coordinate shared runtime-overlay and
source-inventory edits with fn-110; patch minimization retains its own scope.

**Preservation contract.** Keep CLI grammar/defaults, recorded formats, canonical
bytes for fixed identities, error precedence, existing comments, fresh processes,
native timer ownership and separate seed/round/corpus transactions. Migrate the
Artifact-handle interface with explicit ownership and lifetime. D12 retains its
separate owner; unavailable hosts or unexplained regressions leave affected
acceptance incomplete.

**Status.** Tasks 2–12 are merged but remain open pending their acceptance
checks and native qualification. The preparation owner, host-command seam,
installation description, capability ownership, and detached Artifact/opened-handle
migration are implemented; their task artifacts retain developmental checks and
review evidence. Task 13's generated host/runtime time-wire source candidate
passes feasible generation, validation, focused host checks and two fresh source
audits; actual patched-runtime and process acceptance remains open on both native
platforms. Its developmental stock-runtime vector run is not native qualification.
Task 14's typed network/volume command candidate passes literal byte vectors for
all 41 operations, feasible validation and two fresh source audits; real patched-runtime
overlay/process acceptance remains open on both native platforms. Task 15's
eleven behavioral characterizations and two-interface design comparison pass
feasible focused/race/repeat checks and fresh source audits; native gates and
R11 remain open. Task 16's atomic typed lifecycle source candidate passes feasible
focused/race/repeat, preservation, validation and fresh source audits; native/process
acceptance and R11 remain open. Task 17's creation-time network handles pass feasible
ownership, preservation, validation and fresh source review, including canonical
selection of its process regressions; native/process acceptance and R12 remain open.
Task 18's creation-time filesystem handles and mappings pass feasible ownership,
preservation, validation and fresh source review, including canonical selection
of its process regressions; native/process acceptance and R12 remain open.
Task 19's architecture candidate is committed with passing developmental checks
and a bounded independent corrective source review. Formal fan-out failed before
any reviewer started; its round was refunded and no verdict exists. The failure
remains unreproduced after isolated metadata diagnostics; acceptance stays open
([review blocker](.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-19/review-blocked.md)).
Task 20's guidance is committed and its formal three-draw review returned SHIP
([review and acceptance](.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-20/acceptance-open.md)).
Its subsequent nine-flag CLI inventory correction is committed with fresh
independent source review; the earlier formal receipt does not cover that
correction ([correction evidence](.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-20/cli-inventory-correction/acceptance-open.md)).
Inherited D5 native and bounded-control qualification, task 20 acceptance, and
fn-105 D5 closure remain open; task 19's formal review and native gates are still
required. Task 21 is admitted for source/evidence verification and retains matched
developmental 10/100 campaigns, a sixteen-finding matrix and preservation audit.
R18 inventory/provenance reconciliation, final formal review and both native
qualification gates remain incomplete ([qualification evidence](.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/qualification-evidence.md)).
The [2026-10-04 R18 disclosure supplement](.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/preservation-disclosure-2026-10-04.md)
records Choice Trace v2 refusal, controller-v2 journal refusal and retirement
of a previously selected v041 fixture. Original R18/R19 availability,
workload/default, fixed-identity and both native qualification requirements remain open.
Task 23's lint-routing correction and reproducible nested host gates are
independently source-reviewed and committed as ce80d2425c. Actual lint remains
red. Task 24's one-line path-base repair is independently source-reviewed,
preserves working regexes and intended scopes without new suppressions, and
refutes the historical literal-escape hypothesis. Task 25's independently reviewed
controller repair preserves literal before/after lifecycle behavior and clears
the actual package exhaustive finding. Root fast completes ordinary root and
tagged integration lint/vet, then its automatic nested scope fails on the same
419 Gomad findings across 31 owners; nested vet and later scopes remain unreached
([controller acceptance](.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-25/acceptance-open.md)).
Task 26's independently reviewed caller correction restores Runner semantic
ownership while preserving CLI error order, presence/zero checks and writer
routing. Current and saved base CLI pass the same 33 behavior tests, with the
ownership regression adding the 34th final test. Runner production and public
signatures are unchanged; external compilation passes. The 54 affected CLI lint
findings remain inherited, complete CLI/portable-plan gates retain environment
failures, and Darwin identity proof remains skipped. Task 26 stays blocked on
qualification ([caller acceptance](.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-26/acceptance-open.md)).
The remaining nested findings retain bounded source owners
([path-policy acceptance](.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-24/acceptance-open.md)). Task 21 remains the final
verifier, and original R18/R19 and native acceptance stay open
([routing acceptance](.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-23/acceptance-open.md)).
Task 19 includes repairs for the confirmed record and pack-governance
timezone effects and inaccessible public reporting/pack-loading types. Its
World terminal correction moves arbitrary-error normalization to process reporting
and adds detached terminal input to the pure recorder; direct custom-error and
sentinel-rebinding behavior changes are explicitly inventoried. Preserve known
typed failures and fixed-identity recorded bytes. Native Linux qualification
is part of task 21's final gate. Reuse accepted task evidence where its source
identity still matches, without repeating implementations.

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
| 1. Overlay extraction | Qualify the integrated scheduler/quiescence, crypto-initialization, and syscall-linkname extraction while retaining upstream hooks and lifecycle points | R2, R3, R6; preserve lock state, ordering, entropy overrides, activation, and negative behavior |
| 2. Canonical regeneration | Emit `-U1`, update exact descriptor source sets and generated consumers, and make the pinned regeneration check follow the descriptor | R4, R5; repeated byte-identical regeneration and zero-fuzz source equivalence between final `-U3` and `-U1` patches on both hosts |
| 3. Qualification and final evidence | Run full Gomad gates on both platforms, affected entropy/process-simulation tests, Temporal integration/smoke and affected qualification; publish final counts and build identities | R7, R8; fresh repeatability and exact replay under existing contracts, with no weakened dispositions |

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

**Status.** Task 2's scheduler/quiescence extraction is integrated with D26;
task 3's crypto/syscall overlay relocation and task 4's descriptor-bound canonical
`-U1` regeneration are also merged. Local generation, validation, and focused
conformance evidence belongs to the source candidate; required native
darwin/arm64 and linux/amd64 gates remain for all three tasks.
Task 5 remains. Fresh isolated source measurements reproduce task 1's original
baseline (`8d28bd44`) at 32,652 bytes / 1,007 lines and the current extracted
`-U3` at 38,362 bytes / 1,112 lines. R8's extraction reduction is therefore unmet:
the current `-U3` is 5,710 bytes larger, although its canonical `-U1` is smaller
at 29,015 bytes / 778 lines. Keep the original comparator; context reduction
does not substitute for extraction reduction. The measurement inventory also
reports overlay growth separately, including integrated work from other specs.
Source-text equivalence on the unsupported development host is not either native
R4 gate or R7 qualification. The source measurements and input identities are
retained under [task 5 artifacts](.flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task-5/source-size-evidence.json).

## Quality assessment (2026-10-01)

**Scope.** This section retains only the unresolved determinism, test-strategy, and maintenance
findings from the 2026-10-01 assessment. Flow tasks and their artifacts own completed outcomes.

**Spec.** [fn-112](.flow/specs/fn-112-gomad-determinism-assurance-and-test.md) has four open tasks.
Tasks 5, 9, and 16 are merged but remain open pending qualification; task 9's
CLI and Runner consolidation is integrated with fn-109. Task 10 delivered the soak gate
(`gomadtool soak`, `make gomad3-soak`, `tools/gomad3integration/qualification/soak.json`, and the
scheduled `determinism-soak-darwin` and `determinism-soak-linux` jobs of `gomad3.yml`) and the
contract documentation in `tools/gomad3/README.md`, SPEC, CLI, ARCHITECTURE, and TUTORIAL; it stays
open until one completed scheduled or dispatched soak run per platform is retained with its report.
Q3 and Q4 remain D12 candidates under fn-105, and Q5 is D26.

**Current verdict.** Determinism assurance remains incomplete until the stream-isolation,
retained-success collision, and suite-consolidation candidates pass their native gates; D12 gains
causal native-linux evidence; and the first retained soak run on each platform supplies the measured
bound. The contract now states the remaining exclusions and the closure-mode limit. Existing diagnostic traces can localize a fresh-run
difference, but D12 still needs a native Linux reproducer. Collector and
real-thread limitations prevent an unconditional guarantee.

### Determinism gaps

| # | Gap | Evidence | Proposed direction |
| --- | --- | --- | --- |
| Q2 | Stream isolation remains unqualified | Task 5's candidate separates collector and target-visible draws, but its required native gates have not run | Qualify the candidate and its runtime assertion on both supported platforms |
| Q3 | D12 candidate: one-shot syscall wait in `suspendG` | The patch waits for a host syscall to exit once, before the suspend loop. A goroutine that enters a syscall in that window can be scanned mid-syscall, which shifts scan work. Inferred from source, not reproduced | Measure with native diagnostics first; only if a divergence identifies this path, wait inside the loop's `_Gsyscall` case and requalify |
| Q4 | D12 candidate: linux ASLR and `runtime.NumCPU` | Neither is controlled on linux/amd64. Impact is inferred low because the binary is non-PIE | Rerun the D12 reproducer under `setarch -R` before any code change |
| Q5 | Forward shared-clock correction remains unqualified | D16: the original offset made derived deadlines fire late and the subtest fail 5 of 24 seeds. Eight generated workloads use `forward` | Verify the integrated D26/fn-110 candidate and qualify it on both native platforms |
| Q6 | No measured bound yet | The scheduled soak runs 64 to 128 fresh repetitions per workload and seed per run (the measured round cost decides how many fit the 120-minute per-job budget) under two busy host threads, with diagnostics on, and accumulates per-cohort counts across runs. No run is retained: the local linux/arm64 exercise used a stand-in `gomad qualify`, because the patched toolchain does not build there, and measured no Gomad bound | Retain the first scheduled or dispatched run per platform and quote its per-cohort counts; re-check the round sizing from its `execution_wall_nanos` |
| Q7 | Channels without a positive control | Netpoll readiness, SIGPROF and CPU profiling, enabled block and mutex profiling, and `NumCPU` are declared outside the contract in README and SPEC (`RUNTIME.TRUST.EXCLUSIONS`); timer ties and run-queue shuffles have seeded fixtures | Add a fixture before any exclusion is brought inside the contract |
| Q8 | Closure mode compiles no guards | `-gomadguard` is added only in guarded mode; pack admissions of `syscall` are per package, so admitted code runs live. README and SPEC (`TARGET.CAPABILITY`) state the limit, and the soak includes the guarded-mode `frontend-system-info` probe | Read soak results for that probe separately from the closure-mode suites |

Q3 and Q4 remain hypotheses only. Current one-P reasoning makes Q3 low-likelihood, and the
prepared linux binary is expected to be non-PIE; neither should change production code before a
native diagnostic divergence identifies a causal path. D12 must first compare fresh loaded
cohorts with existing diagnostics, repeat them under `setarch x86_64 -R`, and record CPU topology
and cgroup controls. The informational linux/amd64 soak jobs retain such cohorts and their traces.

### Remaining test-strategy work

- Qualify task 5's seeded-stream inventory and runtime check on both native platforms.
- Retain one completed scheduled or dispatched soak run per platform with its report (task 10).
- Qualify task 9's suite consolidation, including its integrated CLI and Runner mapping.
- Qualify task 16's retained-success collision handling.

Retain runtime/model conformance and real CLI recovery coverage while
consolidating structural tests. The minimizer, corpus replacement and eviction,
multi-shard merge, and `world/process` lifecycle remain thinly exercised.
`TestProcessBackendSynchronizesNodeClockWithModelDelay` remains outside
`test-simulation`, with its watchdog finding retained; native Linux gates
still need execution evidence.

### Maintenance cost

Exact-version pins remain recurring maintenance costs. The current inventory
has 15 dependency adapters and 135 SHA-256 anchor literals, plus 11 compatibility
packs, 49 rules and 18 distinct module-version pins. These counts are source-bound;
re-measure them after pin or adapter changes.

| Pin | Size | Repair |
| --- | --- | --- |
| Runtime patch and overlay | 20 upstream files in the combined patch; 73 overlay files in the task 13–14 source candidate | Manual rebase, `make generate`, `make upgrade-dossier`; remeasure lines after the final candidate is frozen |
| Interception fingerprints and boundary manifest | 131 intercepts, 132 fingerprinted entries | Generated; a diff needs an approved SHA |
| Toolchain inventories | 25 host-clock references, 10 goroutine creation sites | Hand edits after `make test-toolchain` on each platform |
| Dependency adapters | 15 adapters, 135 SHA-256 literals (125 distinct; 30 are per-platform prepared source sets); 6 target modules absent from the root `go.mod` (downstream cell, fn-107) | `gomadtool adapter-regenerate` dry run, then apply with the approval digest for any registered adapter, including `modernc.org/libc`; a changed or ambiguous rewrite still needs a person |
| Compatibility packs | 11 packs, 49 rules, 18 module-version pins (the previously selected `modernc-libc-xsys-v041` fixture and pack were retired after fixture coverage moved to v047) | `gomadtool compatibility-pack refresh` per platform, then one `generate --approve-review` per request |

The fn-113 pin-maintenance candidate is merged and exposes impact, adapter regeneration, and
compatibility-pack refresh operations, but none of its four tasks is complete until R6 passes on
qualified darwin/arm64 and linux/amd64 hosts. Exact review, manual resolution of
unsupported rewrites, and both-platform gates remain part of delivery.
The procedure, measured walks, historical baseline and approval evidence
remain under `.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/`.
Do not infer a Linux result from Darwin walks or substitute projected savings
for measured acceptance.

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

1. Qualify merged fn-112 tasks 5, 9, and 16 on both native platforms.
2. Use the existing diagnostics and task 5's checks with Q3/Q4 controls to close D12 on native
   linux/amd64; do not make a speculative Q3 or `NumCPU` change before a causal divergence.
3. Retain the first soak run on each platform, quote its per-cohort bound, and make the linux soak strict when fn-105 R12 closes D12.
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
  The Linux replay finding needs a reproduced cause; collector-file remedies
  remain outside the permitted patch policy.
- **Latent divergence passes qualification.** Routine qualification still has only two fresh
  repetitions, and exact replay forces run-queue and select choices; the scheduled soak supplies
  the statistical bound once its first runs are retained ([Q6](#determinism-gaps)).
- **Spin loops** anywhere in the cluster stall virtual time. Pollers with backoff are fine, but a
  single `for {}` with a non-blocking select is fatal under this runtime.
- **Upstream Go releases** invalidate the patch and the boundary manifest each time; every Go bump
  repeats the port and requalification.
- **Compatibility packs pin exact module versions.** Every dependency bump that touches a packed
  or adapted module invalidates the pack or adapter and reopens the capability closure.
