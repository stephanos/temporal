---
satisfies: [R2, R6, R9]
---
# fn-78-typed-temporal-authoring-and-checked.3 Expose typed authoring for existing Behavior constraints

## Description
Implement D2 and R2 by exposing typed authoring forms for the Behavior constraints already enforced by the checked Behavior language. The new surface must elaborate to existing canonical declarations and must not authorize Target transitions.

**Size:** M
**Files:** `model/Umpire/Behavior/{Language,Authoring,Tests/**}.lean`, `model/Temporal/Feature/Nexus3/{Authoring,Tests}.lean`, `model/Umpire/ARCHITECTURE.md`
**Touches:** [model/Umpire/Behavior/**, model/Temporal/Feature/Nexus3/Authoring.lean, model/Temporal/Feature/Nexus3/Tests.lean, model/Umpire/ARCHITECTURE.md]

### Approach
- Generalize the existing `ExactSequenceSpec` authoring owner with typed forms for allowed/forbidden actions, required named occurrences, occurrence bounds, ordering, and adjacency.
- Lower every surface form into the existing checked Behavior declaration before validation, canonicalization, and evaluation.
- Keep exact action-sequence and exact-trace fixtures as explicit constructs; do not silently broaden them into ordering constraints.
- Use Lean elaboration/source diagnostics where context is needed and compile-failure guards at the author expression.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Behavior/Authoring.lean:9-67` — current typed exact-sequence lowering
- `model/Umpire/Behavior/Language.lean` — canonical checked declarations
- `model/Umpire/Behavior/Tests/Canonicalization.lean` — fingerprint equivalence tests
- `model/Temporal/Feature/Nexus3/Authoring.lean:332-368` — current feature Behavior construction
- `model/Temporal/Feature/Nexus2/AuthoringTests.lean:112-160` — authoring equivalence/diagnostic pattern

### Key context
- Ordering permits intervening Behavior-allowed occurrences; adjacency requires consecutive semantic occurrences.
- True union, general interleaving composition, and repetition remain outside this delivery.

## Acceptance
- [ ] Typed forms cover allowed/forbidden actions, required named occurrences, occurrence bounds, ordering, and adjacency by lowering to existing checked Behavior declarations.
- [ ] Equivalent surface and constructor forms have identical canonical declarations and fingerprints.
- [ ] Tests distinguish ordering from adjacency, exactness from permitted interleaving, and declared constraints from model-impossible occurrences.
- [ ] Existing exact sequence/trace regression fixtures and their bytes/fingerprints remain unchanged.
- [ ] Wrong references, contexts, and malformed bounds fail at the authored expression with stable focused diagnostics.
- [ ] No new form authorizes a transition absent from the Target, and no union/interleaving/repetition semantics are introduced.
- [ ] Focused Behavior/Nexus3 tests and `make umpire-build-model` pass; `model/Umpire/ARCHITECTURE.md` documents the surface and canonical lowering.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
