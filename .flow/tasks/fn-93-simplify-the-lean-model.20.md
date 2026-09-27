---
satisfies: [R9, R10]
---
# fn-93-simplify-the-lean-model.20 Delete the guarded Property forms (B6)

## Description
Lane B6, part two. Delete `guardedEventuallyWithin`/`guardedNeverWithin` (no command can author them): `Evaluate.lean` guarded section (~1198-1386), the guarded arms of `Property/Check.checkClause`, their JSON writers, `PropertyUnless`, `PropertyTemporalClause`, and the arms that reject them by name in `Search/Product/Monitor.lean` (~71-81, 175-176) and `Case/Projection/Lowering.lean` (~104-118). Tests `GuardedTemporal` (471), `GuardedCases` (569). D6 was recorded in task 19.

**Size:** M
**Files:** `model/Umpire/Property.lean`, `model/Umpire/Property/Evaluate.lean`, `model/Umpire/Property/Check.lean`, `model/Umpire/Search/Product/Monitor.lean`, `model/Umpire/Case/Projection/Lowering.lean`, `model/Umpire/Property/Tests/{GuardedTemporal,GuardedCases}.lean`, `model/UmpireTests.lean`, vocabulary gate, `model/Umpire/ARCHITECTURE.md:137-138`, `.plans/UMPIRE4_SPEC.md` Property glossary (~323, review only)
**Touches:** [model/Umpire/Property.lean, model/Umpire/Property/**, model/Umpire/Search/Product/Monitor.lean, model/Umpire/Case/Projection/Lowering.lean, model/UmpireTests.lean, tools/umpire/internal/retiredvocabulary/**, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_SPEC.md]
**Depends on other specs:** `Search/Product/Monitor.lean` (fn-88.4) and `Case/Projection/Lowering.lean` (fn-89.4).

### Approach
- Check canonical bytes first: if any golden or Fingerprint encodes the guarded constructors' indices (`predicateCode`, clause tags), deleting constructors must not shift the surviving indices; if it would, keep the constructor slots or stop and report.
- The removed rejection arms reject a form no command can author; confirm no `#guard_msgs` pins their text, else list it.

### Investigation targets
**Required:**
- `model/Umpire/Property/Evaluate.lean:1190-1390`
- `model/Umpire/Property/Check.lean:709-922` (`checkClause`)
- `model/Umpire/Search/Product/Monitor.lean:60-90,170-180`

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens umpire-check-regression umpire-check-retired-vocabulary
```

## Acceptance
- [ ] Guarded forms, their arms, writers, types, rejection arms and tests gone
- [ ] No surviving canonical byte or Fingerprint changed
- [ ] Every gate green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
