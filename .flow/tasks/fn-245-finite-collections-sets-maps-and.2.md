---
satisfies: [R2, R3]
---
# fn-245-finite-collections-sets-maps-and.2 Front-end admission of imbl OrdSet, OrdMap and the extended Vector method table; native value export; interpreter evaluation

## Description
After .1. In `model/umpire/front` (`shape.rs`, `body/vector.rs` and a sibling `body/collections.rs`): admit `imbl::OrdSet<E>` and `imbl::OrdMap<K, V>` fields over fieldless enums and bounded nominal scalars (plus `bool` and checked integers where `Vector` admits them), under their canonical imports; lower exactly the method table in the spec to one node each, including pure `iter().all/any`, `keys()/values().all/any` closures; refuse `HashSet`/`HashMap`, nested collections, non-finite element types, `for` loops, `map/filter/fold`, set algebra, `entry`, mutating or capturing closures, aliases and unchecked unwraps by source location naming the admitted form. In `umpire-dsl`: `IntoValue` for `OrdSet` and `OrdMap` on native targets (wasm projection absent, as for `Vector`), consumed by `derive(State)`'s hidden `Values`. In `umpire-checker` (`model/umpire/backends/native/src/interpret*.rs`): evaluate every new node over `Value::Set`/`Value::Map`; no-op insert/remove emits no state change and falls under the accepted-self-loop rules. Controls: macro `pass`/`ui` fixtures for each admitted and refused form; interpreter unit tests per node; the existing differential check extended to the new fixtures once .4 lands (wire the hook here).

Touches: model/umpire/front/**, model/umpire/author/dsl/**, model/umpire/backends/native/**, model/README.md.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
