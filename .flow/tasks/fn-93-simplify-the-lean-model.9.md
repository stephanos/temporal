---
satisfies: [R16]
---
# fn-93-simplify-the-lean-model.9 Remove dead declarations and move the cited theorems into tests (A1)

## Description
Lane A1. Delete the zero-reference declarations the spec lists and move design-guarantee theorems into tests. First task of the deletion phase; needs no owner decision.

**Size:** M
**Files:** `model/Umpire/Command/Records.lean` (122-137, 209-291, 301-310; keep `Action.example?` at 297), `model/Umpire/Command/Finite.lean` (`enumerate` 99-104, `EnumerationRefusal` 132-147, `enumerateBounded` 149-159; keep `enumerateOver`), `model/Umpire/Command/Tests/Finite.lean` (107, 232, 235), `model/Umpire/Command/Authoring.lean` (`NoFact` 87), `model/Umpire/Case/Producer.lean` (`factAt` 78), `model/Umpire/Command/Syntax.lean` (`ignoreEnumFields` 175), `model/Umpire/Exploration/Promotion.lean` (`promotionSpec` 37), `model/Umpire/ImplementationLink/Language.lean` (`translateStep` 594, `canonicalImplementationLinkErrorJson` 1183), `model/Umpire/ImplementationLink/Refinement.lean` (`toStutteringSimulation` 186), `model/Umpire/Inventory/Types.lean` (`HasUniqueIds` 75), `model/Umpire/Json.lean` (`semanticallyEqual` 114), `model/Umpire/Scenario.lean` (`exactlyOneAction` 173); theorems `Operation/Canonical.lean:181` `rpcSchema_inj`, `Value/Field.lean:43` `references_complete`, `Replay.lean:136,144` `settle_*_sublist` move to the matching test modules
**Touches:** [model/Umpire/Command/**, model/Umpire/Case/Producer.lean, model/Umpire/Exploration/Promotion.lean, model/Umpire/ImplementationLink/**, model/Umpire/Inventory/Types.lean, model/Umpire/Json.lean, model/Umpire/Scenario.lean, model/Umpire/Operation/**, model/Umpire/Value/**, model/Umpire/Replay.lean, model/Umpire/**/Tests/**, model/Umpire/ARCHITECTURE.md]
**Depends on other specs:** `Case/Producer.lean` (fn-89.4, fn-88.4) and `Exploration/**`, `Replay.lean` (fn-88.10); line numbers above are planning-time.

### Approach
- Re-verify zero references for each name across `model/` and `tools/` (Go may embed names in strings) before deleting.
- Unreferenced non-`@[simp]` theorems beyond the listed ones: list candidates with a grep, delete only those nothing cites (docs included). A removed `@[simp]` lemma goes only if every root builds without it.
- Moved theorems keep their statements; `model/Umpire/ARCHITECTURE.md:310` cites `Canonical.rpcSchema_inj` — update the citation to the test location.
- Moved theorems keep E2 checker entries if they had pins.

### Investigation targets
**Required:**
- `model/Umpire/Command/Records.lean` (whole, 312 lines)
- `model/Umpire/Command/Finite.lean:80-160`
- `model/Umpire/Command/Claims.lean:28` — keeps `Action.example?`

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens && make umpire-check-retired-vocabulary
```

## Acceptance
- [ ] Every listed declaration deleted or, if a reference turned up, kept and listed
- [ ] Cited theorems live in tests with unchanged statements; ARCHITECTURE citation updated
- [ ] Every root builds; goldens and regression byte-identical


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
