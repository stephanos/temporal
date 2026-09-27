---
satisfies: [R14]
---
# fn-93-simplify-the-lean-model.32 Short forms, safe deriving, core list and string functions, shared lemmas (A8)

## Description
Lane A8, part two. Rewrite the surviving `match x with | some d => throw … | none => pure ()` sites as `if let` (pattern at `Umpire/Evaluation.lean:143`); derive `LawfulBEq` where the type is neither nested nor mutual and it removes `decideMem` (`Case/Projection/Lowering.lean` ~246-262) or `exactListMember?`; derived `Ord` only where no proof unfolds it and field order reproduces the old order (`ModelValue`, `RoleBinding`, `Edge`, `Scenario.Order`); `T.ctorIdx` for `searchOutcomeConstructorIndex`; replace the hand-rolled helpers in the spec's table; `Scenario.Order` becomes `DefinitionGraph.Edge` if imports allow; `mapOutcome` becomes `@[simp] def`; one `FiniteMachine.domain_mem_iff` replaces the ten domain theorems.

**Size:** M
**Files:** `model/Umpire/Scenario/Check.lean` (`isSubsequence` ~335, `isPrefix` ~344), `model/Temporal/System/Callback/Configuration.lean` (`lastString` ~133), `model/Umpire/Property/Evaluate.lean` (`collectPositions` ~867), `model/Temporal/Case/Catalog.lean:44`, `model/Temporal/Case/EventKind.lean:43`, `model/Umpire/Command/Syntax.lean:80`, `model/Umpire/Scenario.lean` (`adjacentOrders` ~243, `Order` ~115), `model/Tools/LeanSourceInventory.lean:46-73`, `model/Tools/LeanImportGraph.lean:30-33`, `model/ModelLint/ModuleIndex.lean:173-179`, `model/Umpire/Case/Projection/Lowering.lean`, `model/Umpire/Search.lean:571`, `model/Umpire/Examples/Switch.lean:319-352`, `model/Temporal/System/Nexus/Core.lean:400-431`, `model/Temporal/System/Nexus/ImplementationLink.lean:154,182-186`, `model/Umpire/Model/**` (`FiniteMachine`), match-then-throw sites across survivors
**Touches:** [model/Umpire/**, model/Temporal/**, model/Tools/**, model/ModelLint/ModuleIndex.lean]
**Depends on other specs:** fn-88 (Search pins), fn-89 (`Lowering.lean`).

### Approach
- Core names were verified present in Lean v4.32.0 at planning; re-check only if `model/lean-toolchain` moved.
- `PropertyPredicate` keeps its hand-written `decEq` (nested type).
- `String.capitalize` uppercases only the first char — confirm it matches `capitalizeFirst` on the inputs used; `List.isSublist` is order-preserving non-contiguous — confirm it matches `isSubsequence`; add `#guard`s for both before swapping.
- Leave `fieldRootData`, `predicateCode` and every canonical-byte index alone; SHA-256/hex walks and the two monad stacks stay.

### Investigation targets
**Required:**
- `model/Umpire/Scenario/Check.lean:330-350`
- `model/Umpire/Case/Projection/Lowering.lean:240-265`
- `model/Umpire/Examples/Switch.lean:315-355`

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens umpire-check-regression
LEAN_NUM_THREADS=1 make lint-model
```

## Acceptance
- [ ] Each replacement from the spec's table done or listed with the reason it could not be
- [ ] Deriving used only where the handler accepts the type and no proof or byte depends on the old form
- [ ] Everything byte-identical; no axiom inventory widened; `lint-model` green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
