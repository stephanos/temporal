---
satisfies: [R8]
---
# fn-93-simplify-the-lean-model.29 Validate a table once and build checked Properties directly (A6)

## Description
Lane A6. Pass `CheckedTable` from the first validation onward instead of re-validating (`FiniteTable.validate` ~`Table.lean:137`; call sites `Command/Authoring.lean:135,445`, `Model/Table.lean:711,713,736`); replace `CheckedFieldProperty` (`Property/Evaluate.lean` ~715-725; 29 uses, 15 in `Case/Projection/Lowering`) with `CheckedProperty`; turn the Producer's raw-Property round-trip (~`Case/Producer.lean:1100-1107`) into a checked constructor. Add the counting test, including a machine from `compose`, `restrict:` and `extend:`.

**Size:** M
**Files:** `model/Umpire/Model/Table.lean`, `model/Umpire/Command/Authoring.lean`, `model/Umpire/Property/Evaluate.lean`, `model/Umpire/Case/Projection/Lowering.lean`, `model/Umpire/Case/Relation.lean`, `model/Umpire/Case/Producer.lean`, tests (`model/Umpire/Model/Tests/**`, `model/Umpire/Case/Tests/FieldLowering.lean`, Nexus Pair and Start tests using `CheckedFieldProperty`)
**Touches:** [model/Umpire/Model/**, model/Umpire/Command/Authoring.lean, model/Umpire/Property/Evaluate.lean, model/Umpire/Case/**, model/Temporal/Feature/Nexus/Pair/Tests.lean, model/Temporal/Feature/Workflow/Start/Tests.lean]
**Depends on other specs:** fn-92 (`compose`, `restrict:`, `extend:`; `Model/Table.lean`), fn-89 and fn-88.4 (`Case/Producer.lean`, `Case/Projection/**`).

### Approach
- Counting test: instrument via a test-only counter (e.g. a `dbg`-free `IO.Ref` in test support or counting through a wrapper the test installs); it must not add production state.
- Pin which diagnostic wins when a table is both non-canonical and invalid before changing order-sensitive code.

### Investigation targets
**Required:**
- `model/Umpire/Model/Table.lean:90-140,700-740`
- `model/Umpire/Property/Evaluate.lean:710-730`
- `model/Umpire/Case/Producer.lean:1090-1110`

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens umpire-check-regression
```

## Acceptance
- [ ] Counting test pins one validation per Query elaboration, including composed and derived machines
- [ ] `CheckedFieldProperty` gone; Producer builds checked Properties directly
- [ ] Diagnostic precedence pinned; everything byte-identical


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
