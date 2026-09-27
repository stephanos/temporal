---
satisfies: [R7]
---
# fn-93-simplify-the-lean-model.27 One Diagnostic type for the five error records (A5)

## Description
Lane A5, part one. `DefinitionError` (`Core.lean` ~439/472), `ScenarioError` (`Scenario.lean` ~11/44), `QueryError` (`Query.lean` ~246/277) and `ImplementationLinkError` (`ImplementationLink/Language.lean` ~635/688) become one `Diagnostic` over a kind parameter with one renderer (spec §API Contracts). `SpaceError` went with B1 (if D1 was declined, include it). The renderer keeps the `implementationLinkId` field name and the canonical ID sort.

**Size:** M
**Files:** `model/Umpire/Core.lean` (or a new `model/Umpire/Diagnostic.lean`), `model/Umpire/Scenario.lean`, `model/Umpire/Query.lean`, `model/Umpire/ImplementationLink/Language.lean`, their `*Check` modules and tests that construct or match the records
**Touches:** [model/Umpire/Core.lean, model/Umpire/Diagnostic.lean, model/Umpire/Scenario.lean, model/Umpire/Scenario/**, model/Umpire/Query.lean, model/Umpire/Query/**, model/Umpire/ImplementationLink/**, model/Umpire/**/Tests/**]
**Depends on other specs:** fn-88.6 adds `QueryErrorKind.unreplayableWitness` in `Query.lean`.

### Approach
- Each record's JSON rendering is golden-pinned; build one renderer and a `#guard` per record kind comparing old and new bytes on representative values before deleting the old renderers.
- Kinds stay per-domain enums (`Diagnostic DefinitionErrorKind`, …).

### Investigation targets
**Required:**
- `model/Umpire/Core.lean:430-480`, `model/Umpire/Scenario.lean:1-50`, `model/Umpire/Query.lean:240-280`, `model/Umpire/ImplementationLink/Language.lean:630-690`

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens umpire-check-regression
```

## Acceptance
- [ ] One `Diagnostic` type and renderer; the old records and renderers gone
- [ ] Every rendered diagnostic and golden byte-identical


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
