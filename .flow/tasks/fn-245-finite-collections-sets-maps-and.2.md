---
satisfies: [R2, R3]
---
# fn-245-finite-collections-sets-maps-and.2 Front-end admission of imbl OrdSet, OrdMap and the extended Vector method table; native value export; interpreter evaluation

## Description

After .1. In `model/umpire/front/src` extend canonical type/import recognition and the existing vector/body admission with an ordinary collection method module. Admit exactly the spec's `imbl::OrdSet`, `OrdMap` and extended `Vector` table over fieldless enums, bounded nominal scalars, bools and the existing checked integer subset. Recognize real borrowed element/key arguments and ignored-return mutating effect statements. Lower the complete `get(k).copied().unwrap_or(d)` chain directly to MapGetOr, and the front safe-read chain to Front with its eagerly evaluated fallback; no general Option admission or new collection return-value surface. Extend existing Vector nominal element admission only as required by this same finite type list. Extend `bounded_vector_growth` through the sequence operand of `PopFront`; wrapping appends in a pop must not bypass the authored length-bound check.

Quantifiers bind pure all/any predicates using Elements, Keys or Values. Admit read-only captures of state, World and inputs; refuse mutation and mutable captures. Preserve canonical iteration order, short-circuit checked failures and empty all/any identities. Refuse aliases/shadowed imports, HashSet/HashMap, nested collections, non-finite types, for loops, map/filter/fold, set algebra, entry, nested quantifiers, unchecked unwrap/expect and consumed mutation return values by location, naming the admitted form.

In the native DSL export layer extend Value, `IntoValue` for OrdSet/OrdMap and `From<Value> for umpire_ir::Literal`; `derive(State)` hidden Values and generated interpreted accessors must round-trip typed collection state. Follow Vector's native-only export boundary. Coordinate actual exhaustive readers of the export enum if needed, without new wasm projection or protocol/Go surface. Interpreter imports retain the one Literal::admit; extend evaluation and generated read/write reconstruction for every new node. Map replacement, absent remove, set no-op insert, empty pop and eager defaults match compiled Rust. Preserve unchanged-handler enablement: an otherwise unchanged, non-rejecting handler remains disabled after no-op mutations without `stay!()` and becomes an accepted self-loop when it executes the marker. Wire the existing collection differential hook; .4 supplies the shared fixtures.

Touches: model/umpire/front/**, model/umpire/author/dsl/** (including macros/generated accessors), model/umpire/backends/native/**; actual existing exhaustive export readers only if necessary.

## Acceptance

- [ ] Macro pass/ui controls cover every admitted method and finite scalar family, each read-only capture source and each located refusal; no new author vocabulary or general Option support.
- [ ] Bounded-growth controls admit guarded queue operations and refuse unguarded `push_back; push_back; pop_front`, including through locals and conditional lowering; `PopFront` cannot hide appends from the bound analysis.
- [ ] Native DSL export to Literal::admit and generated state read/write accessors round-trip canonical set/map/queue states, preserving nominal identity and checked failures.
- [ ] Interpreter controls cover same-type key/value projections, duplicate values in key order, empty all/any, skipped division/overflow versus reached failures, map replacement/no-ops, empty queue and eager fallback failures; otherwise unchanged, non-rejecting handlers are disabled after no-op mutations without `stay!()` and accepted self-loops with it, matching compiled checking.
- [ ] The existing differential hook checks admitted collections against compiled checking with complete exploration and matching failures; compiled remains default.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
