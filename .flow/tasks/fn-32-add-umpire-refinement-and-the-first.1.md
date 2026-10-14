---
satisfies: [R1, R2, R6]
---

# fn-32-add-umpire-refinement-and-the-first.1 Define and check the domain-neutral Implementation Link language

## Description
Create the authored-to-checked Implementation Link facade and exhaustive domain-neutral validation fixtures for R1 and R2.

### Review reconciliation (normative)

The prototype is an exact bounded forward simulation. The inert declaration contains finite setup/state/action/outcome/observation/relation/capability tables, a complete support/Known Gap partition, and a positive semantic-transition Limit. A separate proof witness indexed by the exact declaration and checked targets supplies `initialForward`, `stepForward`, and `requiredCoverage`; the trace theorem is derived. There is no reverse/bisimulation obligation, named Behavior occurrence mapping, or serialized proof term.

**Size:** M
**Files:** `model/Umpire/ImplementationLink.lean`, `model/Umpire/ImplementationLink/**`, `model/Umpire.lean`
**Touches:** [model/Umpire/ImplementationLink.lean, model/Umpire/ImplementationLink/**, model/Umpire.lean]

### Approach
- Mirror the checked-language lifecycle used by Target and Observation.
- Keep source/destination targets, mapping tables, support/Known Gap partition, application Limit, and every forward proof obligation explicit.
- Canonicalize before exposing the checked value.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/Language.lean:8-28,354-390` — checked target lifecycle
- `model/Umpire/Observation/Language.lean` — checked mapping/error pattern
- `model/Umpire/Observation/Tests/Compilation.lean` — exhaustive negative fixtures
- `model/Umpire/Core.lean` — semantic trace vocabulary

### Acceptance
- [ ] Complete valid declarations plus exact indexed witnesses check deterministically, with proof terms excluded from identity bytes.
- [ ] Stale, partial, ambiguous, wrong-kind, incomplete support/Known Gap, invalid-limit, and witness-index mismatches fail without a partial value.
- [ ] The public facade is Temporal-independent.
## Acceptance
- [ ] R1/R2 positive and negative matrices pass.
- [ ] Reordered equivalent declarations have identical checked identity.
- [ ] Umpire import purity is preserved.

## Done summary
Added the domain-neutral Umpire Implementation Link authored-to-checked lifecycle, exact indexed forward-simulation witnesses, deterministic identity, strict target/semantic/support/limit validation, and exhaustive positive and negative fixtures. Capability mappings bind complete contracts and reject conflicting providers; the public facade remains Temporal-independent and uses the current Definition vocabulary.

Verification passed the focused Implementation Link build, aggregate Umpire/Temporal build, and full regression gate. `Temporal.System.Nexus.ImplementationLinkTests` is an inherited sequencing gap: Task `.3` owns that absent target, and it was absent both before and after Task `.1`; memory capture was attempted after the non-trivial review fix but memory is not initialized.

stage: impl-review - ran [Codex NEEDS_WORK -> SHIP; 2026-08-27T21:13:58Z..2026-08-27T21:25:23Z; session 01a0450e-49a3-7851-a2a9-a8710eb3e098]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ade871d3396bcd0693b45dd95a21d53f3a3b31db, 88a8b3836071f39cdd75e6cbfb0ec89f5d74b5e0, 76036fcd96c4e4baa1088120092d6efa1994e823
- Tests: baseline: red (cd model && mise exec -- lake build Umpire.ImplementationLink.Tests failed pre-edit because the task-owned target was absent), baseline: red inherited sequencing gap (cd model && mise exec -- lake build Temporal.System.Nexus.ImplementationLinkTests failed pre-edit because Task .3 owns the absent target), baseline: green (cd model && mise exec -- lake build UmpireTests TemporalModelTests), baseline: red tooling (make umpire-check-regression reached a corrupt downloaded Go toolchain cache; the missing archive-identical file was restored before final verification), cd model && mise exec -- lake build Umpire.ImplementationLink.Tests, GATE_SKIPPED:temporal-implementation-link:inherited-sequencing-gap - Task fn-32-add-umpire-refinement-and-the-first.3 creates Temporal.System.Nexus.ImplementationLinkTests; the target is absent before and after Task .1, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, GATE_SKIPPED:unittest:green-receipt 76036fcd - final aggregate pass recorded for downstream reuse, GATE_SKIPPED:smoke:green-receipt 76036fcd - final regression pass recorded for downstream reuse
- PRs:
