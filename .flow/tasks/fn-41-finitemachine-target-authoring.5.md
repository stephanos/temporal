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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
