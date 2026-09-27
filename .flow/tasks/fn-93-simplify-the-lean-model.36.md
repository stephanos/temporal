---
satisfies: [R19]
---
# fn-93-simplify-the-lean-model.36 Table-driven tests and one Nexus product declaration (D-tables, D-dups)

## Description
Lanes D-tables and D-dups. Table-drive the 10 `runCheck … isNone` theorems in `Nexus/Success/Tests.lean` (~290-316, 465-477), the Known Gap rejections through every entry point, and `ImportGraphTests`' 10 isolation tests (one direct and one transitive violation per rule). Shrink positive `#check` import lists to one representative `example` per import surface; keep every negative pin. `Nexus/Tests/Machines.lean` stops redeclaring the product machine (~55-124) and its Caller pins (~131-167), machine rejections move onto a minimal machine, triple-pinned messages keep one pin.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Success/Tests.lean`, `model/ModelLint/ImportGraphTests.lean`, `model/Temporal/Feature/Nexus/Tests/Machines.lean`, Known Gap test files (`model/Umpire/Inventory/Tests/KnownGaps.lean`, `model/Umpire/KnownGap*` tests), `*ImportTests.lean` files with positive `#check` lists
**Touches:** [model/Temporal/Feature/Nexus/Success/Tests.lean, model/Temporal/Feature/Nexus/Tests/**, model/ModelLint/ImportGraphTests.lean, model/Umpire/Inventory/Tests/**, model/Umpire/**/*ImportTests.lean, model/Umpire/**/Tests/**]

### Approach
- Count negative pins (`#guard_msgs` with `error`, failing `example`s, rejection tables) before and after; the counts of distinct pinned behaviors must not drop.
- Machines.lean: import the production product from `Caller/Model.lean` instead of redeclaring; Nexus action docstrings shared (part of G5).

### Investigation targets
**Required:**
- `model/Temporal/Feature/Nexus/Success/Tests.lean:280-320,460-480`
- `model/ModelLint/ImportGraphTests.lean:85-700`
- `model/Temporal/Feature/Nexus/Tests/Machines.lean:50-170`

### Quick commands
```sh
cd model && lake build TemporalModelTests umpire-lint-tests
LEAN_NUM_THREADS=1 make lint-model
```

## Acceptance
- [ ] Listed tests are table-driven; no redeclared product machine
- [ ] Every negative pin still exists (before/after inventory in the receipt)
- [ ] `lint-model` and every root green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
