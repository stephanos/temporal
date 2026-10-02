# Gomad remaining work and decisions

[fn-105](fn-105-gomad-follow-ups-deferred-scope.md) owns the tasks and
acceptance criteria for all 27 remaining items. Required fixes stay open until
their acceptance criteria are met. Deferred items record why they are deferred
and what would revive them;
a trigger must be recorded before their implementation. Deferred items may remain
open or close as won't-do.

## Acceptance Criteria

### Required work

- **R7:** Add required macOS functional smoke CI
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.7.md)). Decision on
  2026-09-30: activate D7. Use a standard GitHub-hosted macOS runner qualified
  as darwin/arm64 and run the same selected functional tests as the Linux smoke
  job, under the same relevant workflow triggers. Validate the qualification
  manifest, retain platform-specific reports and uniquely named artifacts, and
  require zero unsupported, failed, and infrastructure-error outcomes. Keep this
  representative replay gate explicitly traced under R13 and require exact
  choice-tape replay. Verify the Darwin job in GitHub Actions and preserve Linux
  coverage. Any affected replay divergence remains required work under D12/D14;
  adding the job alone cannot close its acceptance.

- **R12:** Fix the linux/amd64 replay-divergence channel in D12
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.12.md)). Decision on
  2026-09-30: this must be fixed; host availability is an execution prerequisite,
  not a deferral trigger. Identify and fix the cause, retain a regression reproducer,
  and demonstrate repeated exact replay for the affected F5/F6 suites on seeds 11
  and 17 on native linux/amd64, including host-load runs. Retain commands, platform
  identity, repetition counts, and outcomes. Restore the affected expectations to
  `qualified` and remove CI acceptance of `nondeterministic` and `replay_divergence`
  only after that evidence passes. Diagnosis or classification alone cannot close
  D12; an unavailable host or unresolved cause leaves the task open.

- **R13:** Make runtime choice tracing opt-in for routine full-suite functional
  qualification in D13 ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.13.md)).
  Decision on 2026-09-30: default the full-suite generator to `choice_bytes: 0`
  and `replay_successes: false`, with success-retention limits consistent with
  that policy. Keep the CLI's existing opt-in behavior. Retain explicitly traced
  representative replay/conformance gates and tracing required by choice-based
  exploration. Reports and documentation distinguish same-seed repeatability
  from verified choice-tape replay without claiming tape replay for untraced runs.
  D12 and D14 require tracing-enabled verification; disabling tracing cannot
  satisfy their acceptance criteria. No trace-capacity increase is required to
  deliver this policy change. Larger-trace support remains deferred under R15.

- **R14:** Fix the darwin/arm64 `TestSignalWorkflowTestSuiteChasm` replay
  divergence in D14 ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.14.md)).
  Decision on 2026-09-30: this is a required fix. Identify the allocating goroutine
  and cause of the heap-span refill ordering difference, fix it, and retain a
  regression reproducer. Demonstrate repeated exact replay on seeds 11 and 17,
  including host-load runs, with retained commands, platform identity, repetition
  counts, and outcomes. Restore the suite's Darwin expectation to `qualified` only
  after verification passes. A shared fix with D12 is acceptable, but D14 requires
  its own Darwin evidence. Diagnosis, classification, or an unverified shared-cause
  hypothesis cannot close D14.

- **R22:** Fix parallel Nexus outcome endpoint collisions
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.22.md)). Required fix
  approved on 2026-09-30. Give each independent start/cancel outcome subtest its
  own endpoint identity across both dispatch paths and error-handling variants,
  keeping endpoint assertions consistent with the registered identity. Preserve
  parallel execution and API/outcome coverage. Verify native Go and Gomad on
  seeds 11 and 17, then remove the four corresponding skips and regenerate the
  qualification manifest. Production endpoint uniqueness behavior stays unchanged;
  diagnosis or a still-skipped test cannot close this fix.

- **R23:** Correct the schedule-migration idempotency test while preserving the
  current contract ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.23.md)).
  Decision on 2026-09-30: preserve production behavior and require the test
  correction. Explicitly establish pending migration before checking that a
  repeated request succeeds without duplicate work. Verify the existing
  closed-state response separately after completion. Replace incidental
  side-effect timing with explicit state preconditions using existing controls
  or an isolated contract test. Verify native Go and Gomad on seeds 11 and 17,
  then remove the skip and regenerate the qualification manifest. A broader
  completed-migration retry contract is outside this task.

- **R24:** Correct reset/signal ordering in the Nexus reset-cross-tree test
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.24.md)). Required test
  fix approved on 2026-09-30. Establish an explicit observable completion
  predicate for the first post-reset workflow task before sending the done
  signal. Preserve reset-run identity, pending-operation survival, all four
  HSM/CHASM creation-policy cases, and the intended history ordering. Use bounded
  existing test/history observation patterns; preserve production behavior.
  Verify native Go and Gomad on seeds 11 and 17, then remove the skip and
  regenerate the qualification manifest. A relaxed history assertion or a
  still-skipped test cannot close this correction.

- **R25:** Fix enhanced `DescribeTaskQueue` report-flag caching in production
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.25.md)). Required
  production fix approved on 2026-09-30. Ensure cached information and response
  construction honor requested poller/stat fields without leaking unrequested
  information or omitting subsequently requested fields. Retain regressions for
  both request orders, cache hits, and flag combinations; preserve shared cached
  data, valid reuse/expiry, and build-ID/task-queue-type isolation. Keep the
  existing functional assertions, verify focused/native/Gomad behavior, and obtain
  a dedicated production correctness review. Update this bug's qualification
  finding and expected target-failure dispositions only after verification;
  unrelated findings remain intact. This is an explicitly approved production
  behavior correction; a Gomad-only workaround cannot close it.

- **R26:** Correct the forward clock in D26
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.31.md)). Decision on
  2026-10-01: implement the correction the D16 investigation proposed. Add the
  forward draws to the virtual clock itself so `time.Now`, timers, sleeps, and
  context deadlines share one clock, as fn-103 specified. First reproduce and
  resolve the simulation time transport failure that moved fn-103 to a separate
  offset; feasibility is not established. If the shared clock proves infeasible,
  record the evidence and return the partial fallbacks and the accepted-limitation
  alternative from the D16 report for a decision; a fallback is not selected by
  this requirement. Add a standard-library conformance fixture that fails on
  the current offset and passes with the correction, remove the
  `UpdateWhilePaused_AfterWindow_ExtendsDispatch` skip, and requalify every
  `clock_tick: forward` workload on seeds 11 and 17 with traced exact replay.
  Preserve `strict` identities and behavior. The tick policy stays part of
  execution identity, and the changed runtime produces a new toolchain identity.
  Errors: a remaining lead of `time.Now` over the timer clock, a changed
  `strict` result, or missing linux/amd64 evidence leaves this requirement open.

- **R27:** Apply the host-clock escape remedies in D27
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.32.md)). Decision on
  2026-10-01: act on the D21 proposals. State in the README contract that
  `MemStats.LastGC`, `MemStats.PauseEnd`, the FIPS entropy `monoTime`, the
  execution tracer's clock snapshot, and pack-gated `syscall.Gettimeofday` carry
  host time. Pin the unpinned darwin `gettimeofday` path and `cputicks` in the
  clock inventory with a fixture for each. The overwrite of the collector stamps
  with stored virtual time from `runtime/proc.go` needs the patch-policy owner's
  recorded approval before implementation; without it the stamps stay a stated
  limitation. The collector and assembly patch prohibitions remain in force.
  Errors: an unclassified clock reference, a remedy that edits a prohibited
  file without approval, or missing linux/amd64 evidence leaves this open.

### Approved investigations

Decision on 2026-09-30: the user approves all remaining investigations, including
the worker-command investigation and R19-R21 below. These tasks must establish
causes and propose owned corrections or explicit limitations. Any needed fix
remains tracked until selected and verified; an investigation
receipt does not claim the underlying issue is fixed.

- **R16:** Investigate the standalone-activity forward-clock poll deadline
  mismatch in D16 ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.16.md)).
  Decision on 2026-09-30: investigation is approved; the correction remains to
  be selected from its evidence. Reproduce the seed-11 client timeout and compare
  native Go with Gomad on seeds 11 and 17. Identify the causal clock/deadline and
  cancellation/response ordering, then establish whether the correction belongs
  in Gomad, deadline propagation, or the test's timeout expectation. Retain a
  minimal reproducer, commands, identities, and outcomes. Record the owner and
  proposed correction or accepted limitation for a subsequent decision. Any
  needed fix remains explicit open work in fn-105. Keep the skip until evidence
  supports removal and preserve the activity-delay assertion and forward-clock
  timestamp improvements. Classification alone does not resolve the skipped test.

- **R17:** Investigate `TestNexusOTELSuite/TestOperation` with two dedicated
  clusters in D17 ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.17.md)).
  Decision on 2026-09-30: investigation is approved. The two-slot pool setting
  already resolves the initial wait; identify the cause of the remaining
  same-seed divergence and the seed-11 replay's missing terminal frame. Retain
  reproductions on seeds 11 and 17, native/Gomad comparisons, suitable
  single-cluster/two-cluster controls, and diagnostic choice evidence where useful.
  Preserve the Nexus operation's application-tracing assertions. A connection
  with D12/D14 requires evidence. Record the correction owner and proposed next
  action for a subsequent decision, retaining any needed fix as explicit open
  work in fn-105. Keep the skip until verification supports its removal;
  classification alone does not resolve the operation.
  The recommended explicit-environment correction and skip removal are tracked by
  [task 30](../tasks/fn-105-gomad-follow-ups-deferred-scope.30.md).

- **R18:** Investigate the worker cancellation delivery delay and inconsistent
  timeout budgets ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.18.md)).
  Reproduce seeds 11 and 17, compare native/Gomad behavior, and trace workflow
  timeout through command generation, transfer, and control-queue polling.
  Account for the 90-second context, five-second child polls, two-second server
  minimum, and 120-second await budget. Record the cause, correction owner, and
  next action. Preserve cancellation assertions and keep the skip until verified.
  The recommended forward-clock correction and skip removal are tracked by
  [task 27](../tasks/fn-105-gomad-follow-ups-deferred-scope.27.md).

- **R19:** Investigate activity fairness backlog readiness in both fairness
  suites ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.19.md)). Establish
  whether all intended activity tasks are eligible before dispatch measurement,
  compare native/Gomad runs and unfairness evidence, and distinguish setup bias
  from a product fairness defect. Propose an owned correction that preserves the
  workload distribution and fairness assertion. Keep the skips until verified.
  The backlog wait, trigger retry, and skip removal are tracked by
  [task 28](../tasks/fn-105-gomad-follow-ups-deferred-scope.28.md).

- **R20:** Investigate heartbeat timeout counting under virtual time
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.20.md)). Compare native
  and Gomad histories, elapsed time, and timeout/reset semantics. Establish
  whether the second expected timeout depends on incidental RPC latency, then
  propose an explicit semantic event/deadline condition or a product correction.
  Preserve heartbeat timeout/recovery coverage and keep the skip until verified.
  The explicit heartbeat deadlines and skip removal are tracked by
  [task 29](../tasks/fn-105-gomad-follow-ups-deferred-scope.29.md).

- **R21:** Investigate target exposure and remedies for host-clock reporting
  escapes ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.21.md)), including
  `MemStats.LastGC` and the inventory's other paths. Distinguish target evidence
  from runtime control flow and test suspected links to D12/D14. Assess remedies
  within the current collector/assembly patch prohibitions; record any option
  that needs a separate policy decision. Propose an owned fix or explicit
  limitation for a subsequent decision. D21 is the prerequisite for D11 and must
  record whether its evidence requires a dynamic Linux audit and the audit's
  feasible scope. Classification alone does not resolve D12/D14.

### Conditional Linux clock audit

- **R11:** Make the dynamic linux/amd64 clock audit conditional on D21
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.11.md)). Decision on
  2026-09-30: D11 depends on completion of D21's host-clock investigation.
  Keep implementation deferred until those findings establish the need and
  feasible scope of this audit. D21 must record the recommendation, evidence,
  and next action. If needed, implement a bounded fixture that disables its vDSO
  clock path and detects forbidden host-clock syscalls after activation, with an
  unseeded positive control and a passing seeded run in Linux CI. Account for
  intentionally retained host-clock paths and existing patch restrictions.
  Completion of D21 alone does not claim the audit is implemented; any required
  audit stays open until verified, while a continued deferral retains its reason.

### Deferred clock policies

- **R6:** Keep the `seeded` and `fixed=<d>` virtual-clock policies deferred
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.6.md)). Decision on
  2026-09-30: the extra timing scenarios are optional exploration features with
  no demonstrated workload that needs them. Revive when a specific bug class
  requires deliberate timestamp ties or constant increments. Record that evidence
  before implementation. On revival, implement the policies and manifest settings,
  qualify a runtime fixture and core workload for each policy, and retain the
  execution-identity and COMPAT-5 evidence required by fn-103. Preserve the existing
  `strict` and `forward` behavior. This decision requires no clock implementation.

### Deferred trace-capacity extension

- **R15:** Larger choice traces, split from the original D13 capacity proposal
  on 2026-09-30, remain deferred
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.15.md)). Revive when a named workload needs a retained
  decision tape beyond the current 64 MiB limit for debugging, replay verification,
  exploration, or minimization. Choose a configurable larger bound or streaming
  based on that workload's evidence. Preserve explicit resource bounds, artifact
  identity, fail-visible overflow, replay validation, and compatibility. Qualify
  the named workload with a retained complete trace and verified choice-tape
  replay; measure trace/storage cost and test limits, interruption, and malformed
  input. Finishing the opt-in policy does not claim this extension is implemented.

### Work coordinated with other specs

- **R1:** Complete shared execution assessment through D1
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.1.md)) under
  [fn-108](fn-108-gomad-reduce-code-size-without-removing.md) R6. Preserve
  equivalent captured-evidence projections, classifications, canonical inputs,
  and failure precedence across seed, choice, and simulation strategies.
- **R2:** Complete shared retention policy and artifact-input composition through
  D2 ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.2.md)) under fn-108 R7.
  Preserve separate strategy transactions, bounded retention, failure visibility,
  recovery, and replay evidence.
- **R3:** Complete private execution injection through D3
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.3.md)) under
  [fn-109](fn-109-gomad-deepen-modules-and-tool-interfaces.md) R5. Migrate
  repository consumers, preserve usable public preparation/replay seams, and
  verify private failure-injection coverage and external compilation.
- **R4:** Complete architecture fitness checks through D4
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.4.md)) under fn-109 R8.
  Discover both qualified platform source sets and retain negative fixtures for
  ownership, host effects, and public-signature visibility.
- **R5:** Complete architecture/platform/determinism documentation reconciliation
  through D5 ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.5.md)) under
  fn-109 R9. Document implemented capabilities, current residual findings, and
  intentional interface changes with bounded support and replay claims.
- **R8:** Complete downstream closure-mode support through D8
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.8.md)) as reused by
  [fn-107](fn-107-gomad-finish-downstream-cell.md) R7/R9. Retain supported final-target
  closure analysis on both platforms, with exact reachable adapters and identity
  drift rejection or fresh evidence that the dependency was eliminated.
- **R9:** Complete downstream dual-platform packs and qualification through D9
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.9.md)) as reused by
  fn-107 R8/R10. Retain reproducible identities, supported closure/linked analysis,
  seeds 11 and 17 repeated twice on each qualified host, and exact retained replay.
- **R10:** Complete the reusable downstream-seam guide through D10
  ([task](../tasks/fn-105-gomad-follow-ups-deferred-scope.10.md)) as reused by
  fn-107 R12. Keep generic guidance independent of consumer names and concrete
  consumer commands consistent with retained successful dual-platform evidence.

Reuse or transfer each coordinated obligation exactly once during task breakdown.
The named specs supply their full delivery and qualification criteria. Registration,
cross-references, and this decision record do not claim implementation.

## Architecture consolidation

| Item | Scope and origin | Revival trigger |
| --- | --- | --- |
| D1 | Shared completed-execution assessment across seed, choice, and simulation (`fn-102` R2) | Revived 2026-09-30; fn-108 R6 coordinates delivery under R1 |
| D2 | Shared retention and artifact-input composition, with separate strategy transactions (`fn-102` R3) | Revived 2026-09-30; fn-108 R7 coordinates delivery under R2 |
| D3 | Private executor injection instead of public `Executor`/`ReplayExecutor` (`fn-102` R4); a Go API change | Revived 2026-09-30; fn-109 R5 coordinates delivery under R3 |
| D4 | Architecture fitness checks for ownership, host effects, and public signatures, with negative fixtures (`fn-102` R5) | Revived 2026-09-30; fn-109 R8 coordinates delivery under R4 |
| D5 | Architecture/platform/determinism documentation reconciliation (`fn-102` R6) | Revived 2026-09-30; fn-109 R9 coordinates delivery under R5 |

These were deferred as maintenance without a waiting consumer or behavior change.
The [architecture assessment](../artifacts/fn-102-gomad-architecture-consolidate/architecture-assessment.md)
retains the evidence. Preserve CLI behavior, schemas, canonical bytes, failure
classification and precedence, and replay compatibility when reviving them.

The 2026-09-30 request to reduce code size while preserving every feature brings
D1/D2 into [fn-108](fn-108-gomad-reduce-code-size-without-removing.md), alongside
verified local cleanup. Task breakdown must reuse or transfer those obligations
instead of creating duplicate implementation tasks. D3's public executor migration
remains outside fn-108's preservation-only scope.

The subsequent 2026-09-30 request to address every architecture finding creates
[fn-109](fn-109-gomad-deepen-modules-and-tool-interfaces.md). It reuses fn-108
for D1/D2 and revives D3/D4/D5 through fn-109 R5/R8/R9. The coordinator's lost
simulation bounds and the reviewed interface/ownership gaps provide current
evidence. Task breakdown must reuse or transfer the existing D1-D5 obligations
exactly once. Authoring these specs changes no task status and claims no
implementation.

## Capability and CI follow-ups

The 2026-09-30 request to finish the work in `GOMAD_CLOUD.md` brings D8/D9/D10
into [fn-107](fn-107-gomad-finish-downstream-cell.md). That spec records their
implementation and qualification requirements; task breakdown must reuse or
transfer the existing obligations. This records requested scope without
claiming completion or changing task status.

| Item | Scope and origin | Decision | Prerequisite or revival trigger |
| --- | --- | --- | --- |
| D6 | `seeded` and `fixed=<d>` clock ticks, manifest settings and qualified fixtures (`fn-103`) | Explicitly deferred 2026-09-30 under R6; `forward` addresses known ties and the extra policies are exploration features | A specific bug class needs deliberate ties or constant quanta |
| D8 | Downstream closure-mode adapter for the signal-handling metrics library (`fn-104` C3/R2) | Revived by fn-107 R7/R9 under R8 | Final downstream target and closure evidence as specified by fn-107 |
| D9 | linux/amd64 downstream packs and qualification (`fn-104`) | Revived by fn-107 R8/R10 under R9 | D8 and the final downstream target; qualify both actual hosts |
| D10 | Downstream seam guide (`fn-104` R4) | Revived by fn-107 R12 under R10 | D9's retained qualification and pack evidence |
| D11 | Dynamic Linux clock audit with disabled vDSO, seccomp denial, and positive control (`fn-101.3`, pre-amendment R5) | Conditional on D21, decided 2026-09-30 under R11 | Complete D21 and retain findings establishing audit need and feasible scope |
| D15 | Larger choice traces (`fn-106.3`, split from D13 on 2026-09-30) | Routine qualification uses seed repeatability; the opt-in policy does not require larger tapes | A named workload needs a retained decision tape beyond 64 MiB for debugging, replay verification, exploration, or minimization |

The constraints in [MILESTONES.md](../../MILESTONES.md#constraints) apply throughout.
New tick policies carry execution identity and the
[COMPAT-5 evidence set](../../.plans/GOMAD_NEXT.md#compat-5-targeted-deterministic-adapters-and-io-models).
Downstream packs/adapters bind exact versions; dependency drift reopens their
qualification. On revival, use the origin spec's requirement text as acceptance.
