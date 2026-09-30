# Gomad deferred follow-ups

[fn-105](fn-105-gomad-follow-ups-deferred-scope.md) owns the tasks and
acceptance criteria. Required fixes stay open until their acceptance criteria are
met. Deferred items record why they are deferred and what would revive them;
a trigger must be recorded before their implementation. Deferred items may remain
open or close as won't-do.

## Required work

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

## Deferred trace-capacity extension

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

## Architecture consolidation

| Item | Scope and origin | Revival trigger |
| --- | --- | --- |
| D1 | Shared completed-execution assessment across seed, choice, and simulation (`fn-102` R2) | A second consumer or strategy hits the duplication |
| D2 | Shared retention and artifact-input composition, with separate strategy transactions (`fn-102` R3) | A second consumer or strategy hits the duplication |
| D3 | Private executor injection instead of public `Executor`/`ReplayExecutor` (`fn-102` R4); a Go API change | A second consumer or strategy needs the seam |
| D4 | Architecture fitness checks for ownership, host effects, and public signatures, with negative fixtures (`fn-102` R5) | A second consumer or strategy exposes the boundary problem |
| D5 | Architecture/platform/determinism documentation reconciliation (`fn-102` R6) | A second consumer or strategy requires the evidence |

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

| Item | Scope and origin | Why deferred | Revival trigger |
| --- | --- | --- | --- |
| D6 | `seeded` and `fixed=<d>` clock ticks, manifest settings and qualified fixtures (`fn-103`) | `forward` addresses known ties; the extra policies are exploration features | A bug class needs deliberate ties or constant quanta |
| D7 | macOS functional smoke job (`fn-101.4`) | Linux supplies the smoke gate and macOS already runs Temporal integration | A darwin-only regression escapes to main |
| D8 | Downstream closure-mode adapter for the signal-handling metrics library (`fn-104` C3/R2) | Linked mode removes the import | A downstream module needs closure-mode preparation or manifests |
| D9 | linux/amd64 downstream packs and qualification (`fn-104`) | The downstream measurement is darwin/arm64 | A downstream gate must run in Linux CI |
| D10 | Downstream seam guide (`fn-104` R4) | Analyzer findings already name the sites | A second downstream module adopts Gomad |
| D11 | Dynamic Linux clock audit with disabled vDSO, seccomp denial, and positive control (`fn-101.3`, pre-amendment R5) | Static inventories cover both platforms; darwin DTrace exercises interception | A linux-only host-clock escape is observed |
| D15 | Larger choice traces (`fn-106.3`, split from D13 on 2026-09-30) | Routine qualification uses seed repeatability; the opt-in policy does not require larger tapes | A named workload needs a retained decision tape beyond 64 MiB for debugging, replay verification, exploration, or minimization |

The constraints in [GOMAD_MILESTONES.md](../../.plans/GOMAD_MILESTONES.md#constraints) apply throughout.
New tick policies carry execution identity and the
[COMPAT-5 evidence set](../../.plans/GOMAD_NEXT.md#compat-5-targeted-deterministic-adapters-and-io-models).
Downstream packs/adapters bind exact versions; dependency drift reopens their
qualification. On revival, use the origin spec's requirement text as acceptance.
