---
satisfies: [R2, R5, R7]
---
# fn-41-finitemachine-target-authoring.3 Migrate System Nexus lifecycle

## Description
Move the independent System-side Nexus lifecycle target onto the same adapter (R2, R5, R7). This is parallel to the Feature migration because it owns a separate model family and file surface.

**Size:** M
**Files:** `model/Temporal/System/Nexus/Core.lean`, `model/Temporal/System/Nexus/Tests.lean`
**Touches:** [model/Temporal/System/Nexus/Core.lean, model/Temporal/System/Nexus/Tests.lean]

### Approach
- Build the System family `finiteMachine` from its existing setup/state/action/outcome/observation lists, encoders, `initialStates`, `stepResults`, closure proofs, and executable-action witnesses.
- Delegate the existing authority, sound/complete, kernel, and planning declarations to the adapter while retaining their qualified names and types.
- Keep the System lifecycle `step`, value decoders, authoritative case theorems, provider/composition, target metadata, and named initial/transition lemmas as family-owned semantic proof seams.
- Replace only routine domain and assembly proofs; preserve case analysis that proves emitted-value closure and action executability.
- Expand focused checks to pin checked metadata/fingerprint, list-valued kernel behavior, planning order, and the named lemmas consumed by the Implementation Link.
- Preserve all existing comments and do not change the System-to-Feature mapping or its direction of dependency.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/System/Nexus/Core.lean:194-243` — repeated membership authority and identity proofs.
- `model/Temporal/System/Nexus/Core.lean:245-498` — case theorems, kernel domain assembly, planning, and Target authoring.
- `model/Temporal/System/Nexus/Tests.lean:1-26` — current System target coverage.
- `model/Temporal/System/Nexus/ImplementationLink.lean:312-365` — downstream family proof seams and mapping witness.

**Optional** (reference as needed):
- `model/Temporal/System/Nexus/ImplementationLinkTests.lean:1-18` — focused imported-kernel compatibility check.
- `model/Temporal/ImplementationLinkTests/Nexus.lean:18-40` — cross-family initial and step correspondence checks.

## Acceptance
- [ ] System Nexus constructs its kernel and planning through `FiniteMachine`, removing routine membership/record boilerplate while retaining genuine closure and executable-action evidence.
- [ ] Existing public declarations, theorem types, imports, source paths, IDs, canonical metadata, Behavior Fingerprint, planning behavior, and comments remain unchanged.
- [ ] The three supported transitions, unsupported cases, initial setups, authoritative case theorems, and named target lemmas retain their current meaning.
- [ ] The System target still satisfies the existing Implementation Link contract without Feature imports or representation-level consumer changes.
- [ ] No generated fixture or unrelated worktree file changes.
- [ ] `cd model && mise exec -- lake build Temporal.System.Nexus.Tests` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
