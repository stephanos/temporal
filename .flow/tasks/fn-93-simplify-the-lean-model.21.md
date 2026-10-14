---
satisfies: [R9, R10]
---
# fn-93-simplify-the-lean-model.21 Delete the Bool/Prop denotation copies (B7, decision D7)

## Description
Lane B7. Delete the ~29 Prop `denote` copies and their `_agrees` theorems in `Property/Evaluate.lean`, and `Property.lean`'s `denotes`/`matches_agrees` (~173/188); keep the evaluators, including `PropertyFieldOperator.matches` (~158), which `PropertyAtom.evaluate` calls. Kept bridges (audit list): `evaluatePropertyClause_correlated_positions`, `Correlated/Reference.checked_eventuallyWithin_agrees`, `Shared.CorrelatedObligation.closed_agrees`, `Case/CorrelatedProofs`, `Case/Correlated`'s `window_property`/`observed_property`.

### Owner decision
- **D7 — delete the Bool/Prop denotation copies. Recommended default: taken.** Alternative: write an independent trace semantics (out of scope here). Record first; if declined, close with no change.

**Size:** M
**Files:** `model/Umpire/Property/Evaluate.lean`, `model/Umpire/Property.lean`, `model/Umpire/Property/ImportTests.lean` (7 pins), `model/Umpire/Property/Tests/{Fields,Boolean}.lean` (axiom entries), any doc citing a deleted theorem
**Touches:** [model/Umpire/Property/Evaluate.lean, model/Umpire/Property.lean, model/Umpire/Property/ImportTests.lean, model/Umpire/Property/Tests/**, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_SPEC.md]

### Approach
- A theorem goes only if no surviving proof uses it and no rule text or architecture document cites it (grep `.plans/` and `model/**/*.md`).
- Each kept bridge keeps its exact axiom inventory under E2's checker; a widened inventory reverts the change.

### Investigation targets
**Required:**
- `model/Umpire/Property/Evaluate.lean` (`denote`, `_agrees` declarations; bridge at ~1678)
- `model/Umpire/Property.lean:150-195`
- `model/Umpire/Case/Correlated.lean:386-434`

### Quick commands
```sh
cd model && lake build
grep -rn '_agrees' model/Umpire/Property --include='*.lean' | wc -l
make umpire-check-goldens
```
## Acceptance
- [ ] D7 recorded; denote copies and their `_agrees` theorems gone; evaluators, `PropertyFieldOperator.matches` included, unchanged
- [ ] Every listed bridge still builds with its prior axiom inventory
- [ ] Goldens byte-identical
## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
