---
satisfies: [R9, R10]
---
# fn-93-simplify-the-lean-model.16 Remove the test-only term elaborators and macros (B4, decision D4)

## Description
Lane B4, part one. Remove `property%`, `scenario%`, `query%` (zero uses) and the `field_compare%`/`correlated_response%` macros; rewrite their tests with the commands or drop tests a command-level test already pins.

### Owner decision
- **D4 — drop the test-only term elaborators and migration tests. Recommended default: taken.** Record first in the Done summary; if declined, close this task and task 17 with no change.

**Size:** M
**Files:** syntax tails `model/Umpire/Property/Elab.lean:140-162`, `model/Umpire/Scenario/Elab.lean:141-164`, `model/Umpire/Query/Elab.lean:157-182` (the rest of each Elab module stays: `Command/Authoring` imports it); macros `model/Umpire/Property.lean:694` (`field_compare%`), `:755` (`correlated_response%`); tests `model/Umpire/Property/Tests/TemporalAuthoring.lean` (16 uses), `model/Umpire/Scenario/Tests/Authoring.lean` (9), `model/Umpire/Property/Tests/Fields.lean` (2); vocabulary gate; `model/README.md:114-151`, `model/ARCHITECTURE.md:111`, `model/Umpire/ARCHITECTURE.md:145-146`
**Touches:** [model/Umpire/Property/Elab.lean, model/Umpire/Scenario/Elab.lean, model/Umpire/Query/Elab.lean, model/Umpire/Property.lean, model/Umpire/Property/Tests/**, model/Umpire/Scenario/Tests/**, tools/umpire/internal/retiredvocabulary/**, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md]

### Approach
- For each test using a term elaborator: find the command-level test that pins the same behavior (name it in the receipt) or rewrite it with the command (`property`, `scenario`, `query`) inside a small machine.
- The `implemented_by` pairs inside the deleted tails go with them.
- Every `#guard_msgs` that survives passes unchanged.

### Investigation targets
**Required:**
- `model/Umpire/Property/Elab.lean`, `model/Umpire/Scenario/Elab.lean`, `model/Umpire/Query/Elab.lean` — tails vs kept parts
- `model/Umpire/Property/Tests/TemporalAuthoring.lean`
- `model/Umpire/Command/Authoring.lean` imports of the Elab modules

### Quick commands
```sh
cd model && lake build
make umpire-check-retired-vocabulary umpire-check-goldens
```

## Acceptance
- [ ] D4 recorded; the five term forms are gone and in the vocabulary gate
- [ ] Each affected test rewritten or dropped with its covering command-level test named
- [ ] Goldens byte-identical; every root builds


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
