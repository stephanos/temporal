# Finite collections: sets, maps and bounded lists in Models, admitted across native, interpreted and TLC checking (model)

## Conversation Evidence

> user (2026-10-24): "do we have a plan to add support for list, set and map?"
> assistant: no; sequences exist as `imbl::Vector<T>` with a method table, sets and maps exist at no layer; fn-211 and fn-240 only name "set-valued state" as the condition they wait for.
> user (2026-10-24): "let's answer the open questions together; one at at time"
> Decisions, each chosen by the owner from a recommendation: (1) consumer: shared fixtures now, the workflow Model later; (2) bounds: finite now, unbounded later; (3) surface: `imbl` persistent types with plain methods, no DSL vocabulary; (4) placement: after fn-237, before fn-240 is revived.

## Goal & Context

Models are plain Rust read once by the shared front end (fn-238) and checked three ways: compiled natively, interpreted over the admitted tree, and through TLC over the emitted TLA+. Today the only collection a Model may hold is `imbl::Vector<T>` of scalars, with `len`, `is_empty`, `push_back` and guarded `get`. The next Temporal Models (workflow, Nexus) will hold set- and map-shaped state: pending timers, signal buffers, per-key status. This spec admits finite sets and maps, and the list operations a queue needs, in every layer at once, against shared fixtures under `model/umpire/dev/fixtures`, so the seams exist when the first Model needs them. Nothing here changes a Temporal Model.

Finite means every element, key and value type is a declared finite type: a fieldless enum or a bounded nominal scalar (fn-227's one range declaration). A set is then a subset of a finite domain, a map a partial function over a finite key domain, and the state space stays finite, so native exploration remains exhaustive, TLC agreement remains exhaustive, and the symbolic back end (fn-240) can encode a set as a fixed-width word. Lists stay bounded the way `Vector` is today, by an authored length guard the front end sees.

## Architecture & Data Models

- **Authoring surface.** State fields may be `imbl::OrdSet<E>`, `imbl::OrdMap<K, V>` and `imbl::Vector<T>`, imported under their canonical names as `Vector` is. `OrdSet`/`OrdMap` rather than `HashSet`/`HashMap`: iteration order is then the element order, so `Debug`, `Display`, `Value` export and goldens are deterministic without sorting, and fieldless enums and bounded scalars already derive `Ord`. `HashSet`/`HashMap` fields refuse with a located diagnostic naming the ordered type. Element, key and value types must be fieldless enums or bounded nominal scalars; `bool` and checked integers are admitted where `Vector` admits them today. No nested collections and no collection-valued map values in this slice.
- **Method table, not Rust interpretation.** Admitted calls, each lowered to one IR node: sets `insert`, `remove`, `contains`, `len`, `is_empty`, `OrdSet::new()`/`default()`, `iter().all(|e| p)` and `iter().any(|e| p)` with a pure predicate closure over the bound element; maps `insert`, `remove`, `contains_key`, `get(k).copied().unwrap_or(d)` (eager default, as `Vector::get`), `len`, `is_empty`, `OrdMap::new()`/`default()`, `keys().all/any` and `values().all/any` with pure closures; lists gain `contains`, `front().copied().unwrap_or(d)` and `pop_front`. Everything else, including `for` loops, `iter().map/filter/fold`, closures that mutate or capture mutably, `retain`, `union`, `intersection`, `difference`, `extend`, `entry`, unchecked `unwrap`/`expect`, aliases and shadowed imports, refuses by source location.
- **IR.** `Type::Set(Box<Type>)` and `Type::Map(Box<Type>, Box<Type>)` beside `Seq`; `Value::Set(Vec<Value>)` in canonical element order and `Value::Map(Vec<(Value, Value)>)` in canonical key order, so equality and `Display` are structural; `ExprKind` gains `SetEmpty`, `Insert`, `Remove`, `Contains`, `MapEmpty`, `MapInsert`, `MapRemove`, `ContainsKey`, `MapGetOr`, `Forall(binder, source, predicate)`, `Exists(binder, source, predicate)`, `Front`/`PopFront` for lists; `Length` extends to sets and maps under fn-242's length rule. The element type is carried on the node, never an enumerated domain, so an unbounded variant later is additive (decision 2).
- **Native and interpreted.** `IntoValue` for `OrdSet<E>`, `OrdMap<K, V>` and the extended `Vector` exports the canonical values on native targets (the wasm projection stays absent, as today); `derive(State)`'s hidden `Values` and typed replay consume them. The interpreter evaluates the new nodes over `Value::Set`/`Value::Map`; its differential check against the compiled checker covers every fixture that admits a collection, which also supplies the "sets" measurement fn-238.6 could not take.
- **TLC.** A set field emits as a subset of its finite element set (`SUBSET E` in the cfg-bounded sense, with the type invariant), a map as a total function from the key domain to the value domain extended by one `Absent` sentinel, so a partial map keeps a fixed TLA+ type and `contains_key` is a sentinel test; `Forall`/`Exists` emit as `\A`/`\E` over the element set; lists keep their sequence emission with `Head`/`Tail`. Typed TLC decoding reads sets, functions and sequences back into `Value` and replays through `checked_step` as today. Nothing changes for Models without collections: byte-identical goldens.
- **Symbolic hand-off.** fn-240's emitter gets a set as an `|E|`-bit word and a map as `|K|` words of value width plus one absent bit; this spec records the encoding in DESIGN.md and does not implement it (fn-240 R7 grows by one item when revived).
- **Fixtures.** Three shared fixtures, each with `model.rs`, `tests.rs`, `expected/**`, selected for lift, interpreter and TLC: `finite_sets` (a membership set over a fieldless enum with a `Forall` invariant and a negative control), `finite_maps` (a per-key status map over bounded keys with absent-aware reads, a postcondition and a negative control) and `bounded_queue` (a `Vector` queue with `push_back`, `pop_front`, `front`, `contains` under an authored length guard). Each has a false claim replayed by TLC. They are the consumers for now; the workflow Model is the consumer later.

## API Contracts

- `umpire-ir`: `Type::Set`, `Type::Map`, `Value::Set`, `Value::Map`, the `ExprKind` variants above; existing variants, discriminants and `Display` unchanged.
- `umpire-dsl`: `IntoValue` for `imbl::OrdSet<E>`, `imbl::OrdMap<K, V>` where `E, K, V: IntoValue + Clone + Ord`; `Vector` unchanged in type.
- `umpire-front`: admission of the fields and the method table, refusals located at the offending node and naming the admitted alternative.
- `umpire-checker` (interpreter): evaluation of every new node; `umpire-tla`/`umpire-tlc`: emission, cfg bounds and typed decoding; no new Cargo dependencies (`imbl` is already admitted).
- No protocol change (`umpire.model.v1` and `umpire.run.v1` untouched), no Go change, no change to Realize, Reset or Propose.

## Edge Cases & Constraints

- Canonical order: `Value::Set` and `Value::Map` are sorted by the element or key's canonical ordinal (enum position, then integer); a duplicate element or key in a decoded backend value is a decoding refusal.
- `insert` of a present element and `remove` of an absent one are no-ops in all three layers and emit no state change; a step whose only effect is such a no-op is a self-loop under the existing accepted-self-loop rules (fn-224).
- `len()` on sets and maps obeys fn-242: an `i32` length, with `+` and `*` on it refused as for `Vector`.
- `get(k).copied().unwrap_or(d)`: `d` is evaluated eagerly, as for `Vector`; `d` must lie in the value type's domain or the front refuses.
- Quantifier closures are pure: they may read the bound element, state, World and inputs; a closure that assigns, calls a non-admitted method, or captures by mutable reference refuses. Nested quantifiers are refused in this slice.
- Catalog limits and R30 budgets hold: the three fixtures are sized so the default gate stays inside the current soft targets; TLC bounds for sets are `SUBSET` of small enums (at most eight elements in fixtures).
- Every Model without collections keeps byte-identical goldens, expansion snapshots and TLA+ output.
- No Temporal detail enters `model/umpire` (owner rule).

## Acceptance Criteria

- **R1:** The IR carries set and map types, canonical values and the listed expression nodes; `Display` and equality are structural; existing variants are unchanged. Errors: a duplicate element or key in a decoded value refuses; nothing else.
- **R2:** The front end admits `imbl::OrdSet<E>`, `imbl::OrdMap<K, V>` and the extended `Vector` with exactly the method table above over finite element, key and value types, lowering each call to one node, and refuses everything else by source location naming the admitted form. Errors: `HashSet`/`HashMap` fields, non-finite element types, nested collections, iteration and mutating closures are located refusals; a refused body leaves the Model checking natively, as today.
- **R3:** Native value export and the interpreter agree with the compiled checker on every fixture that admits a collection (differential check), and the three fixtures' native goldens record their reachable states and Cases. Errors: a disagreement fails the fixture; no clamping or exploration cut may hide one.
- **R4:** TLC checks the three fixtures from their emitted TLA+ with exhaustive agreement, replays each false claim and decodes sets, functions and sequences back into typed values; Models without collections emit byte-identical modules. Errors: an unsupported form refuses emission naming it; a non-replayable trace is witness-rejected.
- **R5:** `finite_sets`, `finite_maps` and `bounded_queue` exist as shared fixtures with lift, interpreter and TLC goldens and one negative control each; the interpreted-versus-compiled timing for the set fixture is measured and recorded, closing fn-238.6's deferred "sets" obligation one way or the other (revert the default if the interpreted checker is more than 25 percent slower on it, per that task's rule). No error surface beyond the gate.
- **R6:** DESIGN.md and `model/README.md` record the collection subset (types, method table, canonical values, TLA+ encodings including the `Absent` sentinel, the symbolic encoding fn-240 will use, and the unbounded checkpoint named as a later owner), with `make model-time` output. No error surface beyond the gate.

## Boundaries

- No unbounded sets, maps or lists; no `String` or open-identifier keys; no nested collections; no collection-valued map values. These are the later checkpoint this spec names for Apalache (fn-211) or SMT arrays (fn-240), not work here.
- No `for` loops, `map`/`filter`/`fold`, set algebra (`union`, `intersection`, `difference`) or `entry` API in this slice.
- No Umpire newtypes and no DSL vocabulary: Models keep writing `imbl` types and plain methods (owner decision 3).
- No Temporal Model change, no protocol or Go change, no symbolic emitter (recorded for fn-240 only).
- No new Cargo dependencies.

## Decision Context

2026-10-24, owner, four decisions taken one at a time:

1. **Consumer:** shared fixtures now, the workflow Model later. Rejected: waiting for the next Temporal Model (the seams would be designed under delivery pressure); adding set-shaped state to the activity Model (contrived); treating Quint parity as the consumer (builds composites no Model uses).
2. **Bounds:** finite now, unbounded later. Finite element domains keep native exploration and TLC exhaustive and let fn-240 encode sets as words; the IR carries element types so an unbounded variant is additive; Apalache or SMT arrays own unbounded forms later. Rejected: finite only with no provision (leaves fn-211 and fn-240 without a target); unbounded from the start (loses exhaustive agreement and blocks on a back end that does not exist).
3. **Surface:** `imbl` persistent types with plain methods, following the `Vector` precedent; the ordered variants (`OrdSet`, `OrdMap`) are the admitted spelling so goldens and value export are deterministic without sorting. Rejected: std collections (second family, O(n) clones); Umpire newtypes (DSL vocabulary the owner has rejected before, see fn-190).
4. **Placement:** after fn-237.5 and before fn-240 is revived. This spec edits the front end, IR, interpreter, TLC emitter and fixtures, the same crates fn-237.1/.2 touch, so it does not overlap them; fn-240 then has a set construct to encode and measure. Spec dependency: fn-237.

Related earlier decisions: Quint parity as the long-term goal (2026-10-21), the shared front end (fn-238), length parity (fn-242), fn-238.6's deferred sets measurement, and fn-240's and fn-211's "set-valued state" conditions, which this spec partially discharges (finite only).
