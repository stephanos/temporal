---
satisfies: [R1, R2, R3, R7]
---
# fn-43-deepen-ordinary-property-behavior-and.2 Deepen Property and Behavior authoring

## Description
Implement the Property/Behavior half of R1-R3 and prove the approach in the smallest complete Switch example. This task owns the first learner-visible checked facade and exact-constructor migration.

**Size:** M
**Files:** `model/Umpire/Property/Language.lean`, `model/Umpire/Behavior/Language.lean`, `model/Umpire/Examples/Switch.lean`, `model/Umpire/Examples/SwitchTests.lean`
**Touches:** [model/Umpire/Property/Language.lean, model/Umpire/Behavior/Language.lean, model/Umpire/Examples/Switch.lean, model/Umpire/Examples/SwitchTests.lean]

### Approach
- Follow the language-owned placement and extraction-hiding shape of `checkedTarget`, but require explicit validity proofs and retain `checkProperty`/`checkBehavior` as the typed diagnostic path.
- Add `PropertyPattern.exact`, `SetupConstraint.roleEquals`, `BehaviorTrace.singleStep`, and the narrow exactly-one-action declaration constructor only where it returns the existing data type and encodes the repeated Switch/Nexus invariant.
- Replace Property/Behavior-local canonical ID, duplicate, and source-path mechanics with Task 1 primitives; adapt results to unchanged domain errors.
- Migrate the Property and Behavior happy paths in Switch while retaining public diagnostic results used by tests and preserving authored documentation/comments.
- Add equivalence and compatibility assertions to the existing Switch tests rather than duplicating fixtures.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/Language.lean:887-901` — checked authoring precedent and dependent extraction boundary.
- `model/Umpire/Property/Language.lean:80-141` — Property pattern surface and logical helper candidates.
- `model/Umpire/Property/Language.lean:643-674` — typed Property checker to keep authoritative.
- `model/Umpire/Behavior/Language.lean:75-149` — setup, occurrence, trace, and declaration constructor surface.
- `model/Umpire/Behavior/Language.lean:843-914` — typed Behavior checker and canonicalization path.
- `model/Umpire/Examples/Switch.lean:422-518` — smallest complete extraction ceremony to remove.

**Optional** (reference as needed):
- `model/Umpire/Examples/SwitchTests.lean:7-60` — semantic and exact-byte compatibility assertions.

### Key context
- Do not copy `checkedTarget`'s default `by native_decide`; the validity proof stays visible at each authored declaration's trust boundary.
- Preserve raw result declarations where they intentionally demonstrate typed failure handling.

### Quick commands
```bash
cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Examples.Switch
```

## Acceptance
- [ ] Property and Behavior expose documented proof-taking checked facades through their existing public imports, with raw typed checkers unchanged.
- [ ] Exact Property, role-equality setup, one-step trace, and evidenced one-action constructors are equivalent to the record values they replace and add no normalization or runtime failure behavior.
- [ ] Switch Property/Behavior happy paths no longer use `Except.toOption.get`; typed diagnostic results, teaching comments, and authored `documentation` text remain available and unchanged.
- [ ] Property/Behavior diagnostics retain their previous kind, offending value, related IDs, source-path fallback, and canonical ordering for invalid fixtures.
- [ ] Focused Property, Behavior, and Switch builds/tests pass with no canonical metadata, fingerprint, or artifact-byte drift and no new unapproved axiom dependency.

## Done summary
Added proof-taking Property/Behavior checked facades, narrow semantic constructors, and shared identity-primitive adapters, then migrated the Switch Property/Behavior happy paths without changing typed diagnostics, fingerprints, or artifact bytes. Focused builds, repository lint, the public trust audit, and independent Codex review are green.

stage: impl-review - ran (SHIP)
## Evidence
- Commits: 19159cb404530c9e025a6edc7215e22244f80ee7
- Tests: baseline: green (cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Examples.Switch), TDD RED: cd model && mise exec -- lake build Umpire.Examples.SwitchTests (missing checked facades and semantic constructors; exit 1), cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Examples.Switch, cd model && mise exec -- lake build Umpire.Examples.SwitchTests, make lint-model, cd model && mise exec -- lake env lean ../.flow/tmp/Fn43Task2Trust.lean (public axiom inventory; no compiler-trust or custom axioms), git diff --check 64099bd40e2eb49c4b4e52b171be0fcfd144ad84..HEAD
- PRs: