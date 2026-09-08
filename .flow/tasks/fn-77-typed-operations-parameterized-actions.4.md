---
satisfies: [R1, R5, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.4 Integrate parameterized Actions with finite and runtime domains

## Description
Integrate parameterized Actions with finite and runtime domains for the referenced parent requirements.

**Size:** M
**Files:** model/Umpire/Operation/**; model/Umpire/Core.lean; model/Umpire/Target/**; model/Umpire/Query/**; model/Umpire/Planning/**; model/Umpire/Artifact/Planning.lean; model/Umpire/Artifact/Types.lean
**Touches:** [model/Umpire/Operation/**, model/Umpire/Core.lean, model/Umpire/Target/**, model/Umpire/Query/**, model/Umpire/Planning/**, model/Umpire/Artifact/Planning.lean, model/Umpire/Artifact/Types.lean]

### Approach
- Bind checked operation templates to exact admitted argument instances; use generic Target/FiniteMachine semantics so prior state and arguments determine allowed alternatives, never caller-selected outcomes.
- Add the smallest checked versioned bridge needed by ordinary ModelValue consumers. Preserve old Atom/literal bytes when no extension is used; parameter equality is typed canonical equality, not display-string comparison.
- Extend existing finite adapters to enumerate exactly explicit instance domains with soundness/completeness. Record fixed, sampled and abstracted dimensions; reject unsupported stronger abstraction claims.
- Represent separately declared runtime admissibility and resource bounds. A runtime value outside finite samples either satisfies this admission or rejects out-of-scope without changing the finite verification claim.
- Update instance/domain identity and receipt encoding only through named versioned semantics; add exact replay and incomplete-search tests.

### Investigation targets
**Required:**
- model/Shared/SemanticData.lean:8 — current Atom representation.
- model/Umpire/Core.lean:379 — authoritative generic TransitionKernel.
- model/Umpire/Target/Semantics.lean:15 — private checked Target.
- model/Umpire/Target/FiniteMachine.lean — existing finite authority.
- model/Umpire/Planning/Engine.lean — candidate traversal and validity receipts.

### Quick commands
`cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests Umpire.Artifact.Tests.Codecs`

`cd model && mise exec -- lake build Umpire.Target.Tests.Parameterized`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Umpire.Target.Tests.Parameterized into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] Finite enumeration equals authored parameter domains and retains Target-owned nondeterministic/error alternatives.
- [ ] Runtime out-of-sample admission is explicit and cannot broaden finite completeness; unsupported abstractions and exhausted search remain distinct.
- [ ] Parameterized semantic edits change intended identities; absent extensions preserve old canonical bytes, fingerprints and replay behavior.
- [ ] Bridge round-trip/denotation and finite proof obligations compile without new trust; focused domain/planning tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
