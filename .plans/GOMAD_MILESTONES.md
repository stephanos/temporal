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
outcomes). What remains is the F10 backlog, the downstream implementation and qualification
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
| F10 | `fn-105-gomad-follow-ups-deferred-scope` | required D12/D14 fixes, D13 opt-in tracing policy, and deferred follow-ups ([decisions and scope](#f10-follow-ups-deferred-scope)) |
| Downstream cell | [fn-107](../.flow/specs/fn-107-gomad-finish-downstream-cell.md) | open; consumer implementation and exact replay on both platforms, including D8/D9/D10 |
| Code-size cleanup | [fn-108](../.flow/specs/fn-108-gomad-reduce-code-size-without-removing.md) | open; local cleanup, then D1/D2 consolidation and equivalence checks ([delivery order](#code-size-cleanup-fn-108)) |
| Deep modules and tool interfaces | [fn-109](../.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md) | open; all sixteen architecture findings, reusing D1/D2 and reviving D3/D4/D5 ([delivery order](#deep-modules-and-tool-interfaces-fn-109)) |
| Runtime patch minimization | [fn-110](../.flow/specs/fn-110-gomad-minimize-the-runtime-patch.md) | open; overlay extraction and canonical one-context-line regeneration, with both-platform qualification ([delivery order](#runtime-patch-minimization-fn-110)) |
| Vocabulary and documentation | [fn-111](../.flow/specs/fn-111-gomad-consolidate-vocabulary-and-update.md) | open; glossary merged into SPEC and guide edits applied in the working tree; reconciliation and acceptance pending ([scope](#vocabulary-and-documentation-fn-111)) |

Work a spec with `/flow-next:work <spec>`; list what is ready with `flowctl ready`.

## Completed functional-test gap work

[fn-106](../.flow/specs/fn-106-gomad-close-the-remaining-tests-gaps.md) owns the completed
tasks and acceptance criteria for Linux seed-17 replay divergence, configurable I/O
transcript capacity, the single-P dedicated-cluster pool, traceback address leakage,
and the worker-command seed-11 hang. Use `flowctl brief` for current task state.
The remaining limitations and findings are recorded below; qualification manifests
own dispositions. Tests that rely on wall-clock latency and owned test bugs retain
their skips until their tests are fixed.

The [LastGC task](../.flow/tasks/fn-106-gomad-close-the-remaining-tests-gaps.2.md)
classifies the host-clock escape without changing the collector patch prohibition.
Classification is not a claim that the escape is fixed.

## Remaining `./tests` dispositions

Every `./tests` test has a disposition in the generated manifest
(`tools/gomad3integration/qualification/tests.json`, from `tests.generator.json`); none is
excluded. What still keeps the package short of "every functional test replays exactly":

- **No exact replay for four suites.** `TestTaskQueueStats_Pri_Suite`,
  `TestVersioning3FunctionalSuite`, `TestVersioning3QueryFunctionalSuite`, and
  `TestWorkerDeploymentSuite` run with a raised I/O transcript and no choice trace; repeatability
  is proven, exact replay is not (F10 D13).
- **Skips with an identified cause.**
  - `TestStandaloneActivityTestSuite/TestStartDelay/UpdateWhilePaused_AfterWindow_ExtendsDispatch`:
    under `clock_tick: forward` a propagated gRPC deadline lands later on the server than on the
    client (seed 11). Investigation is approved as
    [D16](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.16.md); the correction
    remains to be selected from its evidence.
  - `TestNexusOTELSuite/TestOperation`: two dedicated clusters in one process are not yet
    deterministic. Investigation is approved as
    [D17](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.17.md); the pool-capacity
    wait is already resolved, and the remaining cause is unidentified.
  - `TestWorkerCommandsTaskSuite/TestDispatchCancelOnWorkflowTimeout`: on seed 11 the cancel
    command arrives after the test's 90 s context has run below the long-poll minimum.
    Investigation is approved as [D18](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.18.md).
  - `TestFairness{,AutoEnable}Suite/Test_Activity_Basic` and
    `TestWorkflowTaskTestSuite/TestWorkflowTaskHeartbeatingWithEmptyResult` assume wall time passes
    during server work. Investigations are approved as
    [D19](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.19.md) for fairness and
    [D20](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.20.md) for heartbeat timeouts.
  - Test bugs: the Nexus API `…Operation_Outcomes` subtests (four skips) register one endpoint name
    from parallel subtests. This is an approved required fix in
    [D22](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.22.md).
    `TestScheduleMigrationV2ToV1Idempotent` expects idempotency after the
    migration closed. [D23](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.23.md)
    requires correcting the test with explicit pending/closed state checks while
    preserving production behavior. `TestNexusOperationSurvivesResetCrossTree`
    signals before the post-reset workflow task completes.
    [D24](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.24.md) requires
    correcting this ordering with an explicit completion predicate before signalling.
- **Intermittent suites.** `TestSignalWorkflowTestSuiteChasm` on darwin (F10 D14) and the F5/F6
  suites on linux (F10 D12).

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
  `gcMarkTermination` stamps `MemStats.LastGC` with host wall time (also visible through
  `debug.GCStats` and the Prometheus `go_memstats_last_gc_time_seconds` gauge), the FIPS entropy
  source's `monoTime`, and the execution tracer's clock snapshot; on linux also
  `syscall.Gettimeofday` behind the `syscall` pack gate. `LastGC` is the one ordinary targets
  reach. Classified, not fixed: the patch policy keeps every `mgc*` and `mstats*`
  runtime file out of the patch, `time_now` is platform assembly on linux/amd64, and the field is
  written but never read by runtime control flow, so it reaches evidence only if a target prints
  it. Exposure and policy-compatible remedies are an approved investigation in
  [D21](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.21.md); the collector
  patch prohibition remains in force.
- **`TestDescribeTaskQueueEnhanced_ReportFlags`** (versioning suite) fails deterministically ("poller
  info should not be reported") because of a server bug: the enhanced-mode `DescribeTaskQueue`
  response cache in `service/matching/matching_engine.go` is keyed by build ID and task-queue type
  only (`dtq_enhanced:<buildId>.<type>`), so a reachability-only request inside the cache TTL is
  served the previous response's pollers. Virtual time puts the second call in the same instant
  as the first, so the hit is certain. The user approved a required production
  correction in [D25](../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.25.md)
  on 2026-09-30, with regression evidence and a dedicated production review.
  The suite's `target_failure` expectation cites this finding until verification
  supports updating the disposition.
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
  regression evidence; D25 is explicitly approved as such work.
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

**Status.** Twenty-five items after splitting D13's policy and capacity obligations
and adding the approved investigations. The 2026-09-30 blanket approval covers
all remaining investigations, including D18-D21.
D12, D14, and D22-D25 are required fixes; D13 requires opt-in tracing for routine
qualification. Larger-trace support is deferred as D15; D16-D21 require
investigation before corrections are selected. D25 includes dedicated production
review. D7 macOS smoke CI is required. D6 clock policies remain explicitly
deferred, and D11 dynamic Linux clock auditing retains its pending decision in fn-105.
Implementation has not started.

### Required work

| Item | Scope and origin | Decision |
| --- | --- | --- |
| D7 | Add a darwin/arm64 job for the existing functional smoke selection (`fn-101.4`) | Required CI addition approved 2026-09-30. R7 in fn-105 requires a standard macOS runner, preserved Linux coverage, platform-specific reports/artifacts, zero unsupported/failed/infrastructure errors, explicitly traced exact replay, and passing GitHub Actions evidence. |
| D12 | Identify and fix the linux/amd64 replay-divergence channel (`fn-106.1`); about one tier-3 seed-run in 26 diverged, and the recorded evidence points to host timing under load | Must be fixed, decided 2026-09-30. Native Linux instrumentation is an execution prerequisite. R12 in fn-105 requires a regression reproducer, repeated exact replay on both seeds under load, and restoration of strict CI expectations; diagnosis alone cannot close the task. |
| D13 | Make runtime choice tracing opt-in for routine full-suite qualification (`fn-106.3`) | Decided 2026-09-30. R13 in fn-105 requires untraced full-suite defaults, explicit tracing in representative replay/conformance gates and choice-based exploration, and separate reporting of seed repeatability and verified choice-tape replay. D12/D14 verification remains traced. Larger-trace support moves to deferred D15. |
| D14 | Fix Darwin TestSignalWorkflowTestSuiteChasm replay divergence (F7); two heap-span refills swap order at cluster start, historically about one seed-11 replay in 28 | Required fix, decided 2026-09-30. R14 in fn-105 requires identifying and fixing the cause, a regression reproducer, and repeated Darwin exact replay on both seeds under load. A shared D12 fix requires separate Darwin verification before restoring the suite to qualified. |
| D16 | Investigate TestStandaloneActivityTestSuite/TestStartDelay/UpdateWhilePaused_AfterWindow_ExtendsDispatch; the client times out before an empty poll response under the forward clock | Investigation approved 2026-09-30. R16 in fn-105 requires a reproducer, native/Gomad comparison, causal deadline evidence, and a proposed correction with an owner for a subsequent decision. Keep the skip until evidence supports removal; retain any needed fix as explicit open work. |
| D17 | Investigate TestNexusOTELSuite/TestOperation with two clusters; configuring two pool slots resolves the initial wait but repeated runs and replay still differ | Investigation approved 2026-09-30. R17 in fn-105 requires reproductions, control comparisons, the first divergent event and missing-terminal-frame diagnosis, and an owned correction proposal for a subsequent decision. Keep the skip until verification supports removal; a shared D12/D14 cause requires evidence. |
| D18 | Investigate worker cancellation delivery and inconsistent test timeout budgets | Approved 2026-09-30. R18 requires causal delivery/deadline evidence and an owned correction proposal; keep the skip until verified. |
| D19 | Investigate activity fairness backlog readiness in both fairness suites | Blanket investigation approval 2026-09-30. R19 distinguishes setup bias from a fairness defect and preserves the fairness assertions. |
| D20 | Investigate heartbeat timeout counting under virtual time | Blanket investigation approval 2026-09-30. R20 establishes timeout/reset semantics and proposes a correction preserving timeout/recovery coverage. |
| D21 | Investigate host-clock reporting exposure and policy-compatible remedies | Blanket investigation approval 2026-09-30. R21 assesses remedies under existing collector/assembly prohibitions; any changed policy or accepted limitation needs a subsequent decision. |
| D22 | Fix parallel Nexus start/cancel outcome subtests that reuse endpoint names | Required fix approved 2026-09-30. R22 requires independent endpoint identities, preserved parallel/API coverage, native and Gomad verification, and removal of the four skips after verification. |
| D23 | Correct schedule-migration idempotency coverage with explicit pending and closed states | Required test correction approved 2026-09-30. R23 preserves current production semantics, verifies pending-state retry and closed-state behavior separately, and removes the skip after native/Gomad verification. |
| D24 | Correct the Nexus reset-cross-tree test's signal ordering | Required test fix approved 2026-09-30. R24 requires an explicit post-reset task-completion predicate, preserved operation-survival/HSM/CHASM/history assertions, native/Gomad verification, and skip removal after verification. |
| D25 | Fix enhanced DescribeTaskQueue report-flag caching in production | Required production fix approved 2026-09-30. R25 requires request-shape correctness across cache hits/orderings, preserved assertions, focused/native/Gomad verification, dedicated production review, and disposition updates after verification. |

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
| D8 | Downstream closure-mode adapter for the signal-handling metrics library (`fn-104` C3/R2) | Linked mode removes the import | A downstream module needs closure-mode preparation or manifests |
| D9 | linux/amd64 downstream packs and qualification (`fn-104`) | The downstream measurement is darwin/arm64 | A downstream gate must run in Linux CI |
| D10 | Downstream seam guide (`fn-104` R4) | Analyzer findings already name the sites | A second downstream module adopts Gomad |
| D11 | Dynamic Linux clock audit with disabled vDSO, seccomp denial, and positive control (`fn-101.3`, pre-amendment R5) | Static inventories cover both platforms; darwin DTrace exercises interception | A linux-only host-clock escape is observed |
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

**Status.** Open spec; task breakdown and implementation have not started.

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

**Status.** Open spec; task breakdown and implementation have not started.

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

**Status.** Open spec; task breakdown and implementation have not started.

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
obligations. D12/D14 replay fixes and D13/D15 tracing policy/capacity retain
their separate owners and acceptance. Documentation reconciliation cannot close
those items or widen qualification claims.

**Status.** Open spec; documentation edits are already applied in the working tree.
Final reconciliation and recorded acceptance remain pending. Spec registration
starts no implementation tasks.

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
