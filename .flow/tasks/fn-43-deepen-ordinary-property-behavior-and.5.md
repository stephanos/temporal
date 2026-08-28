---
satisfies: [R2, R5, R7]
---
# fn-43-deepen-ordinary-property-behavior-and.5 Extract Implementation Link forward simulation

## Description
Extract the proof-bearing mapping core required by R5 from the duplicated witness and checked-link paths. Keep the abstraction closed over the current Umpire kernel types and responsibilities and consume Task 1's transition-result mapping.

**Size:** M
**Files:** `model/Umpire/ImplementationLink/Language.lean`, `model/Umpire/ImplementationLink/Application.lean`, `model/Umpire/ImplementationLink/Tests/Compilation.lean`, `model/Umpire/ImplementationLink/Tests/Application.lean`
**Touches:** [model/Umpire/ImplementationLink/Language.lean, model/Umpire/ImplementationLink/Application.lean, model/Umpire/ImplementationLink/Tests/Compilation.lean, model/Umpire/ImplementationLink/Tests/Application.lean]

### Approach
- Identify the minimum shared `KernelMorphism`/`ForwardSimulation` data and laws already duplicated by witness and checked-link translation.
- Build transition-result translation on `TransitionResult.map`; move value/step/trace mapping and initial/step/trace preservation behind the interface with semantic lemmas callers can use without unfolding.
- Have `ImplementationLinkWitness` add its declaration index and forward obligations around the morphism; have `CheckedImplementationLink` retain the checked morphism plus existing metadata.
- Reuse the same translation path in Application while preserving all validation/application diagnostics and Known Gap behavior.
- Remove any local transition-result mapper made redundant by Task 1 rather than retaining a parallel helper.
- Preserve and adapt existing module, witness, and `traceForward` comments so the generic-vs-Link ownership split is explicit.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:120-124` — shared transition-result representation and Task 1 mapping contract.
- `model/Umpire/ImplementationLink/Language.lean:288-417` — witness mappings and forward laws.
- `model/Umpire/ImplementationLink/Language.lean:480-510` — checked-link duplication and Link-owned metadata.
- `model/Umpire/ImplementationLink/Application.lean:563-615` — second step/trace translation and preservation path.
- `model/Umpire/ImplementationLink/Tests/Compilation.lean:211-289` — witness/index/coverage failure fixtures.
- `model/Umpire/ImplementationLink/Tests/Application.lean:266-449` — checked application and failure matrix.

**Optional** (reference as needed):
- `model/Temporal/System/Nexus/ImplementationLink.lean` — concrete system adapter that must keep compiling after fn-41.

### Key context
- fn-41 is a hard predecessor: consume its FiniteMachine-derived checked target and authority rather than preserving the pre-refactor target proof shape.
- Do not generalize into category-theory infrastructure, type-class discovery, or cross-language mappings not required by the two current paths.

### Quick commands
```bash
cd model && mise exec -- lake build Umpire.ImplementationLink.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus
```
## Acceptance
- [ ] One documented morphism/forward-simulation boundary owns the current value, step, and trace mapping plus initial/step preservation laws, with transition results mapped through Core `TransitionResult.map`.
- [ ] No parallel transition-result mapper remains in Implementation Link after the shared abstraction lands.
- [ ] Witness and checked-link representations reuse that boundary while Link declaration indexing, coverage, Known Gaps, fingerprints, and diagnostics remain Link-owned.
- [ ] Application uses the shared translation path and retains existing behavior for absent/ambiguous mappings, uncovered definitions, impossible traces, limits, and Known Gaps.
- [ ] Existing comments explain the new ownership split, downstream Nexus links compile, and focused Implementation Link suites pass.
- [ ] Public preservation theorems introduce no new unapproved axiom dependency.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
