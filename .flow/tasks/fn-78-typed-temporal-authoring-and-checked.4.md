---
satisfies: [R3, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.4 Build the checked evidence-projection kernel

## Description
Define D3's generic checked command/event ownership and evidence-projection kernel. Provide one bounded, run-local, transactional admission interface that turns declared source evidence into pending, stutter, emitted semantic-step, or rejected results without importing Temporal or Nexus meaning into Umpire's generic owners.

**Size:** L
**Files:** `model/Umpire/Target/{Language,FiniteMachine,FiniteTable,Tests/**}.lean`, `model/Umpire/Observation/{Declaration,Evaluation/**,Tests/**}.lean`, new focused projection modules under `model/Umpire/Observation/`, `model/Umpire/ARCHITECTURE.md`
**Touches:** [model/Umpire/Target/**, model/Umpire/Observation/**, model/Umpire/ARCHITECTURE.md]

### Approach
- Reuse the kernel's Action/Outcome representation while making controllable submissions and observed confirmed outcomes explicit at the checked boundary; coordinate with fn-75's semantic facade if it has landed.
- Define closed projection declarations for scope keys, stable source-event identities, causal references, semantic outputs, evidence-field policy, and configurable buffer/key/support/work ceilings.
- Stage deduplication, causal closure, transition validation, emitted steps, and exact transitive support before committing an append. A rejection commits none of that append's releases.
- Keep accepted events immutable and preserve prior semantic state, diagnostics, and proved violations across later rejected inputs.
- Admit only declared/authorized evidence fields and redaction dispositions; raw runtime evidence remains outside Property and generic semantic APIs.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/FiniteMachine.lean:420-496` — checked Target seam
- `model/Umpire/Observation/Declaration.lean` — current evidence declarations and bounds
- `model/Umpire/Observation/Evaluation/Admission.lean:516-577` — checked whole-trace admission
- `model/Umpire/Observation/Evaluation/Structure.lean` — causal ordering/closure analysis
- `model/Umpire/Observation/Tests.lean` — focused test root

### Key context
- Equal Target states alone are insufficient for merging projection or monitor progress.
- Missing parents are pending; irrelevant or duplicate admitted evidence stutters; known cycles, identity conflicts, unsupported relevant evidence, and invalid semantic transitions reject.

## Acceptance
- [ ] The checked boundary distinguishes authorized command submission from confirmed Target-owned semantic events without replacing the Target's Action/Outcome authority.
- [ ] Projection exposes a deterministic bounded `admit`/`close` surface with explicit pending, stutter, one-or-more emission, and typed rejection results.
- [ ] Each emitted step retains exact direct and transitive supporting evidence plus declared scope identities; source order and causal references determine order, never cross-source wall time.
- [ ] Rejected appends emit no new steps and leave accepted events, buffers, semantic state, prior diagnostics, and prior violations unchanged.
- [ ] Tests cover duplicates, irrelevant evidence, missing parents, multiple released steps, cycles, conflicts, unsupported evidence, invalid transitions, wrong scope/operation, immutable accepted events, and buffer/key/support/work exhaustion.
- [ ] Fresh projector instances isolate repeated/concurrent Runs, and tenfold evidence/overlapping-key loads either succeed within declared ceilings or fail closed deterministically.
- [ ] Raw evidence cannot enter Property evaluation or bypass checked projection; compile-failure and axiom-audit baselines pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
