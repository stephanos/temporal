---
satisfies: [R1]
---
# fn-245-finite-collections-sets-maps-and.1 IR types, canonical values and expression nodes for sets, maps and list queue operations

## Description
Add `Type::Set`, `Type::Map`, `Value::Set` (canonical element order), `Value::Map` (canonical key order) and the `ExprKind` variants `SetEmpty`, `Insert`, `Remove`, `Contains`, `MapEmpty`, `MapInsert`, `MapRemove`, `ContainsKey`, `MapGetOr`, `Forall`, `Exists`, `Front`, `PopFront` to `umpire-ir` (`model/umpire/ir/src/{lib,value,expr}.rs`), extending `Length` to sets and maps. Structural `Display` and equality; duplicate elements or keys refuse at construction from decoded values. Element types are carried on the node, never an enumerated domain, so an unbounded variant is additive later. Each new variant extends `ExprKind::children()`/`children_mut()` so recursion-only walks need no new arm. Existing variants keep their meaning and Models without collections keep the same checking results; goldens may be re-accepted after review. Unit tests for canonical ordering, duplicate refusal and `Display`.

Starts after fn-237.5 (cross-spec prerequisite through this spec's dependency on fn-237); no other lane edits `model/umpire/ir` meanwhile. Also starts after fn-248.3 (cross-spec prerequisite, prose only: flowctl has no cross-spec task deps; fn-248 itself lands after fn-247), which adds `ExprKind::children()`/`children_mut()`; this task extends both for every new variant.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
