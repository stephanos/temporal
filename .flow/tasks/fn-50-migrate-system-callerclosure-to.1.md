---
satisfies: [R1, R2, R5]
---
# fn-50-migrate-system-callerclosure-to.1 Express CallerClosure as a FiniteMachine

## Description
Replace the manual finite kernel and planning proof with the existing adapter while retaining public authority declarations and their definitional proof shapes (R1, R2).

**Size:** M
**Files:** `model/Temporal/System/Nexus/CallerClosure.lean`, `model/Temporal/System/Nexus/Tests.lean`, `model/Temporal/System/Nexus/ImplementationLink.lean`, `model/Temporal/System/Nexus/ImplementationLinkTests.lean`
**Touches:** [model/Temporal/System/Nexus/CallerClosure.lean, model/Temporal/System/Nexus/Tests.lean, model/Temporal/System/Nexus/ImplementationLink.lean, model/Temporal/System/Nexus/ImplementationLinkTests.lean]

### Approach
- Follow the finite-domain, enumerator, coverage, and executable-action pattern in the ordinary System Core and Feature Lifecycle targets.
- Keep the existing ordered values and encoders exact; expose a compatibility kernel whose domains reduce to the current equalities and whose authority predicates reduce to the current conjunctions instead of delegating those fields directly to list membership.
- Transport kernel soundness, completeness, and finite-planning proofs from the machine across those equivalent views.
- Preserve every existing comment and public name, adding focused compile-time assertions for the existing `change` and `rcases` proof forms.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/System/Nexus/CallerClosure.lean:82-179,279-305` — manual kernel and repeated finite planning
- `model/Umpire/Target/FiniteMachine.lean:12-150` — adapter contract and projections
- `model/Temporal/System/Nexus/Core.lean:297-490` — compatibility-kernel System migration pattern
- `model/Temporal/Feature/Nexus/Lifecycle/Target.lean:206-346` — direct delegation pattern and its representation trade-off
- `model/Temporal/System/Nexus/ImplementationLink.lean:424-674` — equality/conjunction `change` and `rcases` consumers that must remain definitionally valid
- `model/Umpire/Target/Tests/FiniteMachine.lean:5-150` — adapter edge and order tests
## Acceptance
- [ ] The machine's domains, encoders, initial states, steps, coverage, and executable-action proof match the current kernel exactly.
- [ ] Existing authority, kernel, and planning names retain their signatures, values, and definitional equality/conjunction proof shapes.
- [ ] Kernel soundness, completeness, and planning are transported from the machine through the compatibility wrapper without duplicating the finite semantic authority.
- [ ] Compile-time compatibility tests exercise the existing Implementation Link `change`/`rcases` forms without list-membership rewrites.
- [ ] Invalid setup/state/action combinations remain empty or false.
- [ ] Focused CallerClosure and FiniteMachine compatibility tests pass with all comments preserved.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
