---
satisfies: [R5]
---
# fn-93-simplify-the-lean-model.8 DefinitionFamily.id takes a closed IdKind (A3)

## Description
Lane A3, part two. `DefinitionFamily.id (kind key : String)` becomes `DefinitionFamily.id (family) (kind : IdKind) (key)` with identical rendering; about 50 production callers move to `IdKind` constructors.

**Size:** M
**Files:** `model/Umpire/Id.lean`, callers (grep `DefinitionFamily.id` / `.id "`): heaviest are `model/Umpire/Property/Correlated.lean` (11), `model/Umpire/Examples/Switch.lean` (8), `model/Umpire/Command/Authoring.lean` (6), `model/Umpire/Scenario.lean` (6), plus Temporal realization callers
**Touches:** [model/Umpire/Id.lean, model/Umpire/**, model/Temporal/**]
**Depends on other specs:** callers in `Case/Producer.lean` and `Case/Projection/**` are fn-89 surfaces; re-read at start.

### Approach
- `IdKind` spells via the WireName derivation (kebab default gives today's strings; overrides where a kind string is not the kebab form). Enumerate the distinct kind strings passed today first (`grep -rhoE 'DefinitionFamily.id [^ ]+ "[a-z-]+"'` style), so every existing string has a constructor.
- A `#guard` over every constructor pins `IdKind.name` to the old literal; Definition IDs across goldens are the end-to-end oracle.
- Callers that pass a computed string (not a literal) are listed; each maps through a total function or stays on a `String` escape only if it is an open vocabulary (none expected).

### Investigation targets
**Required:**
- `model/Umpire/Id.lean:1-30`
- `model/Umpire/Property/Correlated.lean` — heaviest caller
- `model/Umpire/Examples/Switch.lean` — test-facing caller with goldens

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens && make umpire-check-regression
```

## Acceptance
- [ ] `DefinitionFamily.id` takes `IdKind`; no caller passes a kind string
- [ ] Every Definition ID, golden, Case fixture and Fingerprint byte-identical
- [ ] `lake build` of every root green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
