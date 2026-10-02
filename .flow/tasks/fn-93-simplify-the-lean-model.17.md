---
satisfies: [R9, R16]
---
# fn-93-simplify-the-lean-model.17 Fold the migration tests and move test-only APIs to test support (B4)

## Description
Lane B4, part two. Remove `Umpire/Tests/MigrationCompatibility.lean` (383) after moving its two distinct behaviors (occurrence moves keep the checked product and the planner result; diagnostics follow the occurrence) into `Model/Tests/Composition.lean`. Move test-only APIs into test support. D4 was recorded in task 16.

**Size:** M
**Files:** `model/Umpire/Tests/MigrationCompatibility.lean`, `model/Umpire/Model/Tests/Composition.lean`, `model/UmpireTests.lean`, `model/Umpire/Model/Table.lean` (test-only parts of `CheckedTableModel` at ~570-698), `model/Umpire/Property.lean` (step sugar), `model/Umpire/Model/Check.lean` (`withoutPlanning` ~97, `withEquivalentMachine` ~515, `canonical*Json` ~542), `model/Umpire/Property/Check.lean:1111`, `model/Umpire/Scenario/Check.lean:555`, `model/Umpire/Model/Canonical.lean:146-187`, `model/Umpire/Search.lean:683`, `model/Umpire/Artifact/Codecs.lean:207`, a test-support module (e.g. `model/Umpire/Shared/Test/Model.lean`), importing tests, `model/Umpire/ARCHITECTURE.md:86`
**Touches:** [model/Umpire/Tests/**, model/Umpire/Model/**, model/UmpireTests.lean, model/Umpire/Property.lean, model/Umpire/Property/Check.lean, model/Umpire/Scenario/Check.lean, model/Umpire/Search.lean, model/Umpire/Search/VisibilityTests.lean, model/Umpire/Artifact/Codecs.lean, model/Umpire/Shared/**, model/Umpire/**/Tests/**, model/Umpire/ARCHITECTURE.md]
**Depends on other specs:** `Search.lean` (fn-88 R12 pins: lower only); `Model/Table.lean` is an fn-92.3/.4 surface.

### Approach
- Keep in production what `Command/Authoring.modelVocabulary` (`:440-452`) uses (`validate`, `stateValue`, `actionValue`, `outcomeValue`, `factValue`) and `FiniteTable.checkModel` (production callers `Command/Authoring.lean:139`, `Temporal/System/Nexus/ImplementationLink.lean:81`).
- Before moving each API, grep for production users; any hit keeps it. Test support stays inside `testSupportNamespaces`.
- `DraftModel.make` stays (Nexus System side).

### Investigation targets
**Required:**
- `model/Umpire/Tests/MigrationCompatibility.lean`
- `model/Umpire/Command/Authoring.lean:130-140,440-452`
- `model/Umpire/Model/Table.lean:560-740`

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens
LEAN_NUM_THREADS=1 make lint-model
```

## Acceptance
- [ ] MigrationCompatibility gone; its two behaviors pinned in `Composition.lean`
- [ ] Listed test-only APIs live in test support; production-used parts stay; Search pins lowered if touched
- [ ] Goldens byte-identical; `lint-model` green


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
