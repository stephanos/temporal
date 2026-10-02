---
satisfies: [R4]
---
# fn-93-simplify-the-lean-model.5 WireName for the semantic-root enums (A2)

## Description
Lane A2, second task. Convert the hand-written name functions in the semantic roots, using the form task 4 settled: `Umpire.Query` (5), `Scenario` (3), `Property` (4), `Search.lean` (3), `Search/Product/Monitor` (2), `Search/Product/Scenario` (1), `Model/Types` (1), `Artifact/Types` (2). If task 4 concluded the semantic roots cannot use the derivation at all (DG4 second branch), this task instead records that in its receipt and closes with the list of kept functions.

**Size:** M
**Files:** `model/Umpire/Query.lean`, `model/Umpire/Scenario.lean`, `model/Umpire/Property.lean`, `model/Umpire/Search.lean`, `model/Umpire/Search/Product/Monitor.lean`, `model/Umpire/Search/Product/Scenario.lean`, `model/Umpire/Model/Types.lean`, `model/Umpire/Artifact/Types.lean`, `model/Umpire/Search/VisibilityTests.lean` (fn-88 R12 line/surface pin)
**Touches:** [model/Umpire/Query.lean, model/Umpire/Scenario.lean, model/Umpire/Property.lean, model/Umpire/Search.lean, model/Umpire/Search/Product/**, model/Umpire/Search/VisibilityTests.lean, model/Umpire/Model/Types.lean, model/Umpire/Artifact/Types.lean, model/Umpire/Tests/**]
**Depends on other specs:** fn-88 (closed first) — `Search.lean` and `Search/Product/**` are fn-88 surfaces; fn-88 R12 pins `Search.lean`'s line count and `#check` surface: lower the count in the same commit, never raise it.

### Approach
- Same per-type loop as task 4 (equivalence check, then delete old function in one commit).
- `PlanningOutcome.constructorClassifiers_exactlyOne` (`model/Umpire/Search.lean:582,631`) is proved by per-constructor `rfl`; keep it green by generating structural defs.
- Re-scan with a strict pattern first (`def .*\.name : .* → String` whose arms are all literals) to fix the exact list; the planning count (73 strict total) may have moved after fn-88.

### Investigation targets
**Required:**
- `model/Umpire/Search.lean:560-640` — outcome names and classifier proofs
- `model/Umpire/Query.lean` name functions; `model/Umpire/Property.lean` name functions
- Task 4's receipt — the settled derivation form

### Quick commands
```sh
cd model && lake build Umpire UmpireTests
LEAN_NUM_THREADS=1 make lint-model
make umpire-check-goldens && make umpire-check-regression
```

## Acceptance
- [ ] Every listed hand-written name function derived (or listed as kept with the DG4 reason)
- [ ] Equivalence checks passed before deletion; `constructorClassifiers_exactlyOne` proofs unchanged in statement
- [ ] fn-88's Search pins updated downward only; goldens, regression, Fingerprints byte-identical; `lint-model` green


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
