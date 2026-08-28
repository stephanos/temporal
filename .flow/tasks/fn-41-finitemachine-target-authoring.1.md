---
satisfies: [R1, R2, R3, R7]
---
# fn-41-finitemachine-target-authoring.1 Add the FiniteMachine Target adapter

## Description
Create the focused public Target-layer adapter and its contract tests (R1, R2, R3, R7). This task owns the deep-module boundary only; family migrations remain separate so the API is proven before it receives production callers.

**Size:** M
**Files:** `model/Umpire/Target/FiniteMachine.lean`, `model/Umpire/Target.lean`, `model/Umpire/Target/ImportTests.lean`, `model/Umpire/Target/Tests/FiniteMachine.lean`, `model/Umpire/TargetTests.lean`
**Touches:** [model/Umpire/Target/FiniteMachine.lean, model/Umpire/Target.lean, model/Umpire/Target/ImportTests.lean, model/Umpire/Target/Tests/FiniteMachine.lean, model/Umpire/TargetTests.lean]

### Approach
- Add `Umpire.Target.FiniteMachine` below Query/Planning/Artifact, using the existing Target language and kernel types rather than changing `Umpire.Core` or creating a Shared dependency.
- Make the descriptor own ordered finite lists and encoders for all five domains, initial/step enumerators, the seven emitted-value coverage obligations, and one executable witness for every listed planning action.
- Derive membership domain predicates, membership authoritative relations, their soundness/completeness fields, the complete behavior domain, the checked kernel availability, and the dependent authored-planning value from the descriptor.
- Provide stable membership rewrite theorems while keeping the derived relations reducible enough for local `change`/`simp` proofs; do not require downstream unfolding of private adapter internals.
- Pass authored lists and transition results through without normalization. Support empty proof-valid machines and multiple transition results, and reuse the existing `checkTarget` diagnostic for colliding encodings.
- Re-export the adapter through `Umpire.Target`; extend the focused import contract and aggregate Target tests. Keep direct `TransitionKernel` construction unchanged.
- Preserve every existing comment in touched files and add concise module/declaration documentation for the new public seam.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:126-243` — behavior-domain and transition-kernel proof contract to derive.
- `model/Umpire/Target/Language.lean:21-44` — exact finite-planning contract and executable-action requirement.
- `model/Umpire/Target/Language.lean:192-229` — dependent `AuthoredTarget.make` planning seam.
- `model/Umpire/Target/Language.lean:747-905` — existing validation, checked composition, and kernel re-ascription behavior.
- `model/Umpire/Target/Tests/Validation.lean:145-190` — canonical-encoding failure pattern to retain.

**Optional** (reference as needed):
- `model/Umpire/Target/Tests/Fixtures.lean:1-100` — small checked-target fixture conventions.
- `model/Umpire/Examples/Switch.lean:113-230` — independent relation that must remain on the direct expert path.

## Acceptance
- [ ] `FiniteMachine` is public through `Umpire.Target` and constructs the exact kernel/planning inputs consumed by ordinary `AuthoredTarget` authoring.
- [ ] Domain predicates and authoritative relations are derived from membership, routine soundness/completeness and action-completeness plumbing is absent from caller code, and stable rewrite theorems are available.
- [ ] Contract tests cover successful construction, empty proof-valid domains, multiple results for one state/action, preserved action order, out-of-domain proof obligations, executable-action evidence, and existing colliding-encoding diagnostics.
- [ ] Existing direct `TransitionKernel`, incomplete-kernel, and checked Target APIs remain source-compatible.
- [ ] Existing comments in touched files are preserved; no `sorry` or `admit` is introduced.
- [ ] `cd model && mise exec -- lake build Umpire.Target.ImportTests Umpire.Target.Tests.FiniteMachine Umpire.TargetTests` passes.

## Done summary
Added the public Umpire Target `FiniteMachine` adapter, deriving complete membership-based kernels and exact dependent planning inputs from one proof-carrying finite descriptor. Added import and contract coverage for successful and empty machines, ordered nondeterminism, compile-time closure/executability obligations, collision diagnostics, and unchanged expert Target APIs.

baseline: red (`cd model && mise exec -- lake build ...` failed pre-edit because the task-owned `Umpire.Target.Tests.FiniteMachine` module did not yet exist); pre-edit `make umpire-check-regression` and `make lint-model` were green.
stage: impl-review - ran [2026-08-28T19:30Z..2026-08-28T19:37Z]
## Evidence
- Commits: da7c593e9e29b82d8051ca5f4d496c25af476e55, a329435dfe75498a533a62b31174b4537bc94dcd
- Tests: cd model && mise exec -- lake build Umpire.Target.ImportTests Umpire.Target.Tests.FiniteMachine Umpire.TargetTests, cd model && mise exec -- lake build Umpire.Target.Tests.FiniteMachine Umpire.TargetTests Temporal.Feature.Nexus.LifecycleTests Temporal.System.Nexus.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests, make umpire-check-regression, make lint-model, rg -n '(sorry|admit)' model/Umpire/Target/FiniteMachine.lean model/Umpire/Target/Tests/FiniteMachine.lean
- PRs: