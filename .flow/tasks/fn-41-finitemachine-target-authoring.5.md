---
satisfies: [R6]
---
# fn-41-finitemachine-target-authoring.5 Document FiniteMachine authoring

## Description
Document the settled authoring boundary (R6). This task follows compatibility coverage so the guides describe the proven API and remain compatible with the later fn-39 Nexus layout work.

**Size:** M
**Files:** `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `model/README.md`, `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_SPEC_MODEL_ARCH.md`, `.plans/UMPIRE4_SPEC_COMPS.md`
**Touches:** [model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md, model/README.md, .plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_SPEC_MODEL_ARCH.md, .plans/UMPIRE4_SPEC_COMPS.md]

## Approach
- Teach `FiniteMachine` as the ordinary proof-carrying Target authoring adapter: authors declare finite semantic data and residual evidence once, then consume the same checked Target path as every other model.
- Document direct `TransitionKernel` construction as the expert route for independently specified authority, using Switch as the reference without recasting it as an ordinary adapter client.
- Place the module below Query/Planning/Artifact and outside Shared, Temporal family, runtime, and optional verification ownership; keep the existing isolation rules and import direction explicit.
- Update the Umpire 4 authoring and component contracts to say the adapter is typed convenience, not another Behavior/Scenario language or general DSL, and that semantic choices and checked authority remain explicit.
- Align the README learning path with the concise Lifecycle use while leaving fn-39 responsible for its later physical facade split. Do not edit generator-owned views or fixtures.
- Preserve existing documentation and code comments while integrating the new explanation at the appropriate semantic altitude.

## Investigation targets
**Required** (read before coding):
- `model/Umpire/ARCHITECTURE.md:18-36` — public Umpire module responsibilities and import surface.
- `model/ARCHITECTURE.md:79-88` — Target position in the model dependency DAG.
- `model/README.md:68-130` — current ordinary Target authoring and learning sequence.
- `.plans/UMPIRE4_SPEC.md:152-172` — explicit semantic authoring rules.
- `.plans/UMPIRE4_SPEC_MODEL_ARCH.md:118-198` — target maintainer role and current handwritten-kernel guidance.
- `.plans/UMPIRE4_SPEC_COMPS.md:194-215` — Target interface and component ownership contract.

**Optional** (reference as needed):
- `model/Umpire/ARCHITECTURE.md:419-433` — Switch/reference sequence.
- `.plans/UMPIRE4_SPEC_COMPS.md:370-416` — logical family template to update consistently.

## Acceptance
- [ ] Architecture and authoring guides consistently describe the ordinary `FiniteMachine` route, the direct expert kernel route, and the unchanged checked Target authority.
- [ ] Documentation does not introduce a macro language, second semantic IR, Shared dependency, optional-checker dependency, or abandoned-code migration path.
- [ ] The README distinguishes the reusable authoring API from fn-39's later physical Nexus layout work.
- [ ] Generator-owned regression views and compatibility fixtures remain unchanged.
- [ ] Existing comments and unrelated worktree changes are preserved.

## Done summary
Documented `FiniteMachine` as the ordinary proof-carrying Target adapter, kept direct `TransitionKernel` construction as the independent-authority expert route, and separated reusable authoring from the later Nexus browsing layout. All six architecture/authoring guides now converge on the existing `AuthoredTarget` / `checkTarget` authority; code, comments, generated views, and compatibility fixtures are unchanged.

baseline: red (`make lint-model` failed pre-edit on a transient Lake/virtiofs ENOENT while writing `Temporal/DynamicConfig.olean`; isolated `Temporal.DynamicConfig` warmup and the exact lint retry passed). The exact focused acceptance build and `make umpire-check-regression` baseline commands passed.

verification: green (73-job focused acceptance build, 186-job canonical regression, and 158-job model lint passed). The regression gate caught and removed a reusable-Umpire-to-Temporal documentation reference before final verification; gate receipts were not warrantable only because the inherited false symlink stat at `config/development.yaml` keeps the worktree dirty.

stage: impl-review - ran [2026-08-28T20:46:13Z..2026-08-28T20:51:29Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5a503a02798c80badf1aedd06513d4b21153ae1e, 610faf244e8519ccca56ebf203a2f0274c866f34
- Tests: cd model && mise exec -- lake build Umpire.Target.Tests.FiniteMachine Umpire.TargetTests Temporal.Feature.Nexus.LifecycleTests Temporal.System.Nexus.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests, make umpire-check-regression, make lint-model, cd model && mise exec -- lake build Temporal.DynamicConfig (pre-edit tooling warmup after transient ENOENT), git diff --check 30da5a8bbb0cfa1283e914cf1d46cde2fb7abb93..HEAD, generated regression views and compatibility fixtures unchanged; only the six task-owned documentation files differ from base, gate receipt storage skipped: inherited config/development.yaml false symlink stat kept worktree dirty; all final gate commands exited 0
- PRs:
