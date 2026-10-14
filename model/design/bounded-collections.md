# Bounded collections in the Umpire DSL

Status: proposal, 2026-10-13. No source has been changed. Paths are relative to the repo root.
Line numbers are from the tree this was written against.

**Location.** This file sits in `model/design/`, the location the task names. I looked for an
existing convention first. `docs/superpowers/specs/` holds twelve dated `*-design.md` files, the
newest from 2026-08-26, which is before the Scala DSL. `.plans/` holds the current Umpire design
notes (for example `.plans/DSL_OPERATORS.md`). `.flow/specs/` holds Flow specs, which `flowctl`
creates and tracks. None of these is a clear home for a design proposal about `model/`. If this
proposal is accepted, a Flow spec can cite this file.

## Summary

Add three bounded collection types to the DSL and the IR:

- `Set[K]`, Scala's immutable set over a finite `K`;
- `Total[K, V]`, a total function from a finite `K` to a finite `V`;
- `ListUpTo[A, N]`, a list of at most `N` values of a finite `A`, by analogy with `UpTo[N]`.

A partial map is written `Total[K, Option[V]]`. Its catalog and its catalog order are exactly
those of a partial map.

**The semantics are defined by an encoding.** Each collection means one existing finite type:

- a set is a record with one `Boolean` per key;
- a total map is a record with one `V` per key;
- a bounded list is a FIFO channel's contents with no redeliveries.

That encoding fixes the catalog order and equality, so §1 of SEMANTICS.md gains no new ordering
rule. The IR stores collections natively, so that keys stay readable, every catalog is exact, and
access costs no match ladder per key. Go does all evaluation, ceiling checks and canonicalization
in one new interpreter module.

**A smaller change, independent of collections.** Let a rule's guard read the action's inputs:
`where((s, rm) => ...)`. This alone removes TwoPhaseCommit's twelve per-class rules and its twelve
per-RM effects.

Neither sample changes state-space size: every collection has the catalog size of the hand
encoding it replaces (§7).

---

## 1. What exists today, and why the samples hand-encode

### 1.1 Every state type is a finite catalog, and every catalog is walked in full

- SEMANTICS §1 (`model/SEMANTICS.md:51-95`): catalog order is semantic. Values are keyed by
  spelling: a case is `C-v1-v2`, a record `v1-v2`, a list `[v1,v2]` (line 64-65). A list or an
  unbounded `Int` field is refused (line 90-92).
- `Finite[T]` is derived from a `Mirror`. Its `@implicitNotFound` text names "a plain Int or a List"
  as having no catalog (`model/framework/dsl/Domain.scala:14-20, 53-66`). `UpTo[N]` is an opaque
  `Int` with a `Finite` of `0..N` (`Domain.scala:85-96`).
- The lifter refuses list-typed fields (`model/framework/export/Types.scala:110-117`). Any type that
  is neither an enum nor a case class is refused with "has no finite catalog"
  (`Types.scala:136-141`). A Scala `Set` field fails there today.
- The IR's `TypeRef` (`proto/internal/temporal/server/api/umpire/v1/value.proto:45-58`) has `list`,
  but only "for a step's facts and a function's results; not a state field". It has one bounded
  collection: `channel` (line 55-56).
- Admission refuses `Int` and `List` state fields (`tools/umpire/ir/validate.go:433-441`). Monitor
  states and channel messages go through `catalog` (`validate.go:906-915`), which also refuses
  `Channel`.
- **The Go reader evaluates every state in the catalog, not only the reachable ones.** `rows` loops
  over all states times all classes (`tools/umpire/interp/machine.go:529-568`). `Preflight` refuses
  the work against the ceilings before listing anything (`machine.go:378-393`). The ceilings are
  `Members: 1<<16` per catalog and `Evaluations: 1<<20` per machine
  (`tools/umpire/interp/limits.go:21`). So the catalog product, not the reachable set, is what a
  collection costs.

### 1.2 The one collection precedent: channels

A channel is a native IR collection:

- it has its own `TypeRef` (`value.proto:55-56`) and catalog (`interp/channels.go:42-72`, mirrored by
  `model/framework/dsl/Channel.scala:38-50`);
- its size has a closed form that is checked before listing (`limits.go:116-131, 170-188`);
- its operations are one `Expr` kind with an op enum (`expression.proto:181-197`; `channels.go:96-120`);
- its state field is bound through a hand-written `given Finite[S]` block
  (`Types.scala:187-202`; SEMANTICS §18, `model/SEMANTICS.md:1012-1019`).

An unordered channel keeps its contents in catalog order "so that sending the same messages in
another order reaches the same state" (`Channel.scala:97-108`; `channels.go:74-94`). That is the
canonical-representation idea this proposal reuses.

The second precedent is a desugaring. `Option[A]` is lifted as a synthetic IR enum
`scala.Option[A]` with cases `None` and `Some(value)` (`Types.scala:144-169`).

### 1.3 Why Bank and TwoPhaseCommit look the way they do

**Bank.** There is no map, so `Balances` has one field per address and `Coins` one field per
denomination (`model/framework/samples/Bank.scala:24-27`). Every dynamic access is a `match` ladder:

- `coinsOf`, `amountOf`, `withAmount`, `withBalance` (`Bank.scala:29-52`);
- `sumForDenom` unrolls the sum by hand (`Bank.scala:55-57`).

**TwoPhaseCommit** has two problems.

1. *No collections.* `States(rm1, rm2, rm3)` is the map and `Members(rm1, rm2, rm3)` the set
   (`model/framework/samples/TwoPhaseCommit.scala:23-33`). Each needs its ladder: `stateOf`,
   `withState`, `contains`, `withMember`, plus the hand-unrolled `all`, `any`, `someAre` and
   `allCommitted` (`TwoPhaseCommit.scala:43-80`).
2. *Guards cannot read inputs.* A rule's guard is `S => Boolean`:
   - the `guard` field of `Case` (`model/framework/dsl/Syntax.scala:509-517`);
   - the top-level `where` (`Syntax.scala:583-589`);
   - `Rule.guard` (`model/framework/dsl/Machine.scala:96-106`).

   An effect may not give `Nil`, because "the rules say where it fires"
   (`model/framework/export/Declarations.scala:1217-1238`). So a guard that depends on `rm` must be
   written once per class, `on(rm.prepare(ResourceManager.rm1))`, and an effect fired for one class
   reads the state alone (`Syntax.scala:759-766`). That forces 12 per-class rules
   (`TwoPhaseCommit.scala:203-239`) and 12 per-RM effects (`TwoPhaseCommit.scala:172-183`).

The rule lowering already has the input names in scope. The step function's parameters are the
state followed by the action's inputs (`Declarations.scala:1202-1208`), and an effect lambda of
arity `1 + inputs` already binds them (`Declarations.scala:1158-1168`). Only `condition` binds the
state alone (`Declarations.scala:1109-1118`).

---

## 2. Types

### 2.1 The three types

| Type | Admitted when | Catalog (the semantic definition) | Size |
| --- | --- | --- | --- |
| `Set[K]` (`scala.collection.immutable.Set`) | `K` has a `Finite` | the record of one `Boolean` per member of `K`'s catalog, first member slowest | `2^|K|` |
| `Total[K, V]` (new, `framework.dsl`) | `K` and `V` have a `Finite` | the record of one `V` per member of `K`'s catalog, first key slowest | `|V|^|K|` |
| `ListUpTo[A, N]` (new, `framework.dsl`) | `A` has a `Finite`, `N` a constant ≥ 0 | a FIFO channel's contents of capacity `N` with no duplicates: empty, then every list of length 1, …, `N`, last item fastest | `Σ_{i=0..N} |A|^i` |

Each collection is admitted wherever the types it is built from have a `Finite`, so nesting works:
`Total[Addr, Total[Denom, UpTo[2]]]` and `Option[Set[K]]` are both admitted. A collection is not a
`Named` type, so a machine's state type is still a case class or an enum that holds the collection
in a field. `Machine` reads its state as `Named` (`machine.go:335`).

**Why the catalog order is the encoding's order.** It is the order of the records the samples write
today. `Members(rm1: Boolean, rm2: Boolean, rm3: Boolean)` lists `{}`, `{rm3}`, `{rm2}`,
`{rm2,rm3}`, … by the product rule. `Set[ResourceManager]` lists the same subsets in the same order.
Migrating a hand encoding therefore keeps the catalog in the same row order, and only the keys
change. Nothing new enters §1: a set is listed the way the record it stands for is listed.

**Partial maps.** `Total[K, Option[V]]` has exactly the catalog of the partial functions `K ⇀ V`.
`Option`'s catalog puts `None` first (`Domain.scala:42-43`), so the empty map comes first. A
separate partial-map type would add a second spelling of one meaning. Scala's own `Map` is
therefore not admitted in the first phases (§9, Phase 6). §8.4 compares this mix with native
Scala `Map` and `List` and explains why it is kept.

### 2.2 Scala surface

```scala
// framework/dsl/Domain.scala (new members)

// Every subset of K: the record of one Boolean per member of K, the first member slowest.
given [K](using k: Finite[K]): Finite[Set[K]]

// A total function from K to V: one V per key, in K's catalog order.
opaque type Total[K, V] = Map[K, V]
object Total:
  def fill[K, V](v: V)(using Finite[K]): Total[K, V]
  def by[K, V](f: K => V)(using Finite[K]): Total[K, V]          // Quint's mapBy
  extension [K, V](t: Total[K, V])
    def apply(k: K): V
    def updated(k: K, v: V): Total[K, V]
    def forall(p: (K, V) => Boolean): Boolean
    def exists(p: (K, V) => Boolean): Boolean
    def count(p: (K, V) => Boolean): Int
    def sumOf(f: (K, V) => Int): Int
    def keysWhere(p: V => Boolean): Set[K]
    def map[W](f: V => W): Total[K, W]
  given [K, V](using Finite[K], Finite[V]): Finite[Total[K, V]]

// At most N items of A. It reads as a List; write one with ListUpTo(xs), as UpTo(n).
opaque type ListUpTo[A, N <: Int] <: List[A] = List[A]
object ListUpTo:
  def apply[A, N <: Int](xs: List[A])(using bound: ValueOf[N]): ListUpTo[A, N]  // require(xs.size <= N)
  def empty[A, N <: Int]: ListUpTo[A, N]
  given [A, N <: Int](using Finite[A], ValueOf[N]): Finite[ListUpTo[A, N]]

// The whole catalog of K as a set, for quantifying over a type that is not an enum.
def catalog[K](using Finite[K]): Set[K]
```

Design notes:

- `Total` is opaque with **no** upper bound. With `<: Map[K, V]`, Scala's member `updated` (which
  returns `Map`) and `removed` would win over the extensions, and an author could leave the total
  catalog.
- `ListUpTo` has the bound `<: List[A]`, which mirrors `UpTo[N] <: Int` (`Domain.scala:88`). Reading
  operations come from `List`. Every write goes back through `ListUpTo(...)`. As with `UpTo`, the
  Scala `require` and the Go domain check both refuse a list that is too long. The lifter lifts
  `ListUpTo(xs)` as `xs`, as it lifts `UpTo(n)` as `n` (`Expressions.scala:583-588`).
- `Set` is Scala's own type. Its equality is set equality, which is the semantic equality, and its
  member API is mostly order-insensitive. The lifter admits a closed subset of that API (§3).
- The `Finite` givens list lazily, and they check their size against the same ceiling Go uses
  before listing anything (§5.3).

### 2.3 Value spelling in the IR

SEMANTICS §1 keys values by spelling, and table identity is computed from those keys
(`model/SEMANTICS.md:443`). The spelling is therefore a compatibility commitment, chosen once:

| Value | Spelling | Example |
| --- | --- | --- |
| set | members in **`K`-catalog order**, in braces | `{rm1,rm3}`, `{}` |
| total map | `key:value` per key in `K`-catalog order, in braces | `{rm1:working,rm2:prepared,rm3:working}` |
| bounded list | the existing list spelling | `[a,b]` |

- **Canonical form.** A set's members, and a total map's entries, are stored in strictly ascending
  catalog index of `K`. Structural `Value.Equal` (`interp/eval.go:103-120`) then is semantic
  equality, as it is today for unordered channel contents.
- **Punctuation.** None of `{`, `}`, `,`, `:` is `-`. Go splits keys only on `-`
  (`machine.go:732`; `internal/engine/compose.go:520`; `internal/engine/refine.go:277`;
  `internal/engine/canonical.go:200`), and I found no split on `,` or `:` under `tools/umpire`.
  Element keys can still contain `-` (for example `{abort,prepared-rm1}`). Hyphen counts then differ
  between states, and `distinctRowKeys` falls back to its exact check (`machine.go:495-522`). This is
  already the case for `Option` fields (`None` against `Some-x`).
- **Distinct keys.** Distinct sets have distinct spellings because distinct members have distinct
  keys, which `distinctKeys` already requires of every catalog (`machine.go:61-72`).

### 2.4 Equality and CanEqual

- `Set[K]` and `Total[K, V]` receive `CanEqual` from `Finite`'s blanket
  `given [T](using Finite[T]): CanEqual[T, T]` (`Domain.scala:23`), as every finite type does under
  `-language:strictEquality` (SEMANTICS §1, line 85-88). `ListUpTo[A, N]` does too.
- `==` lifts to structural `OP_EQ` (`Expressions.scala:18-30`), and Go compares canonical forms.
- The Scala runtime compares `Set` and `Map` by content, so the Scala-side checks agree: the
  overlap check (`Machine.scala:332-347`) and the step tables.

---

## 3. Operations in step functions, guards and Properties

### 3.1 What lifts, and how

All of these may appear at every level that reads values: step functions, guards, `end`,
refinement maps, Properties, monitor `next`/`violated`, starts. The levels rule of SEMANTICS §20
(`model/SEMANTICS.md:1142-1156`) is unchanged, because no collection operation makes a step.

| Authored | Receiver | IR lowering |
| --- | --- | --- |
| `Set(a, b)`, `Set.empty` | – | `Collection{OP_SET_OF, args}` (Go canonicalizes); a constant set is lifted as a canonical `SetValue` literal |
| `s.contains(x)` | `Set`, `ListUpTo` | existing `OP_CONTAINS`; Go's evaluator already iterates `Items` (`eval.go:642-648`) |
| `s.incl(x)`, `s.excl(x)` | `Set` | `OP_INCL`, `OP_EXCL` |
| `s.union(t)`, `s.intersect(t)`, `s.diff(t)`, `s.subsetOf(t)` | `Set` | `OP_UNION`, `OP_INTERSECT`, `OP_DIFF`, `OP_SUBSET` |
| `s.size`, `s.isEmpty`, `s.nonEmpty` | `Set`, `ListUpTo` | `OP_SIZE`; `isEmpty` is `OP_EQ(size, 0)` and `nonEmpty` is `OP_NE` |
| `s.forall(p)`, `s.exists(p)`, `s.count(p)` | `Set`, `ListUpTo` | `OP_FORALL` / `OP_EXISTS` / `OP_COUNT` with `body` a one-parameter `Lambda` |
| `s.sumOf(f)` | `Set`, `ListUpTo`, enum `values` | `OP_SUM` |
| `s.filter(p)`, `s.map(f)` | `Set` | `OP_FILTER`, `OP_MAP` (`type` is the result set's type) |
| `K.values.forall(p)` / `exists` / `count` / `sumOf`, for an enum `K` | – | the same op with no `receiver` and `over = K` |
| `catalog[K].forall(p)` … | – | the same, for any finite `K` |
| `Total.fill(v)`, `Total.by(f)` | – | `OP_TOTAL_BY` over `K` (`fill` lowers to a constant body) |
| `t(k)` | `Total` | `OP_GET` |
| `t.updated(k, v)` | `Total` | `OP_UPDATED` |
| `t.forall((k, v) => …)`, `exists`, `count`, `sumOf` | `Total` | the quantifier ops with a two-parameter `body` |
| `t.keysWhere(p)` | `Total` | `OP_KEYS_WHERE` |
| `t.map(f)` | `Total` | `OP_MAP_VALUES` |
| `xs :+ x`, `x +: xs`, `xs ++ ys` | `ListUpTo` | existing `OP_CONCAT` and list prepend (`Expressions.scala:599-603, 639-640`); `:+` is a new lifter case |
| `xs.headOption` | `ListUpTo` | `OP_HEAD_OPTION`, giving the synthetic `Option` enum |
| `xs.drop(n)` | `ListUpTo` | `OP_DROP` |
| `ListUpTo(xs)` | – | `xs`; Go's domain check bounds the length |

**Refused, with the reason in the refusal:**

- Set `+`, `-`, `|`, `&`, `++`, `--`. `.plans/DSL_OPERATORS.md` rule 2 is "one symbol, one meaning",
  and `+`/`-` already mean integer arithmetic in the liftable table. Write `incl`, `excl`, `union`,
  `diff`.
- `toList`, `head`, `iterator`, `zipWithIndex` and any operation whose result depends on the order
  Scala happens to iterate a `HashSet` in.
- `List.head`, `tail` and `apply(i)` on a `ListUpTo`. They are partial, and Scala throws where the
  IR would hit a hole. Write `headOption` and `drop(1)`.
- `s.map(f).sum` on a `Set`. Scala collapses equal results before summing, which is correct for a
  set but almost never what the author meant. Write `s.sumOf(f)`.
- Any Scala `Map` operation (§2.1).

Today `forall` is admitted only over a capability's fact family. SEMANTICS says it has "no dynamic
collection lowering" (`model/SEMANTICS.md:1207`; `model/framework/inspect/LiftableSubset.scala:63`).
This proposal adds that lowering deliberately and adds rows to the liftable-subset table, each with
a fixture, as that section requires (`model/SEMANTICS.md:1166-1169`).

### 3.2 Quantifier semantics: defined by desugaring

`s.forall(p)` means `p(m1) && p(m2) && …` over `s`'s members in canonical order.
`K.values.forall(p)` means the same over `K`'s catalog. `exists` is the `||` chain, `count` is
`(if p(m1) then 1 else 0) + …`, and `sumOf` is `f(m1) + …`. Because the chain short-circuits
(`eval.go:604-616`), a hole in `p(m3)` is not reached when `p(m2)` already decided the answer, as in
any `&&` chain (SEMANTICS §12). The IR node is a compact form of this chain, not a new meaning.

### 3.3 Why native nodes rather than unrolled chains

The lifter could unroll `K.values.forall(p)` into an `OP_AND` chain itself, since it knows `K`'s
catalog (`Declarations.scala:1062-1078`). It could unroll `t(k)` with a dynamic `k` into a `Match`
on `k` with one arm per key, the way rules match inputs today (`Declarations.scala:1191-1201`). I
reject unrolling as the IR form for three reasons:

- every dynamic access then costs `O(|K|)` IR, and an `updated` inside a quantifier costs
  `O(|K|²)`;
- Quint would lose its own `forall`/`mapBy`;
- it fixes no catalog problem for lists (§8.1).

The unrolled chain stays the semantic definition and the test oracle (§9).

**Tension with `.plans/DSL_OPERATORS.md` rule 8** ("No operator whose lowering needs a new IR
node"). That rule governs *sugar*. These collections are core in the sense of SEMANTICS §21: "what
the IR needs declared" (`model/SEMANTICS.md:1269-1278`). The precedent is `Inbox`, a core node with
an op enum. The proposal adds one `Expr` kind, `Collection`, with one op enum, and no sugar.

### 3.4 Guards that read inputs (independent of collections)

```scala
on(rm.prepare) {
  where((s, r) => mayPrepare(s, r)) ~> effects.prepare
}
```

- **DSL.** `Case` carries `guard: (S, List[Any]) => Boolean`, adapted from the authored arity the
  way `effectOf` adapts effects (`Machine.scala:265-280`). `Rule.guard` changes the same way
  (`Machine.scala:102`). The overlap check becomes
  `earlier.guard(s, inputs) && added.guard(s, inputs)` inside its existing loop over classes and
  states (`Machine.scala:338-342`). That loop already enumerates both, so disjointness keeps its
  meaning: no state *and class* is covered by two rules.
- **Lifter.** `condition` binds a guard lambda of arity `1 + inputs` to the state name followed by
  the input names, as `effectCall` does (`Declarations.scala:1109-1118`, `1158-1168`). The step
  function already takes the inputs as parameters (`Declarations.scala:1202-1208`), so the lowering
  stays `if g(s, i) then e(s, i) else …`. SEMANTICS §6 already writes this core form with `c1(i)`
  (`model/SEMANTICS.md:360-363`; `Syntax.scala:660-662`).
- **Composition with `when`.** A phase heading followed by an input-reading guard,
  `when(Phase.a).where((s, i) => …)`, is `Heading.And` with a two-parameter guard
  (`Declarations.scala:1139-1140`).
- **Derivations.** `rebind(on(actor.a) { where((s, i) => …) ~> e })` uses the same `Case`
  (`model/SEMANTICS.md:801-809`).

---

## 4. IR and Go reader changes

### 4.1 Schema (additive, `proto/internal/temporal/server/api/umpire/v1/`)

```proto
// value.proto
message TypeRef {
  oneof ref {
    // ... named = 1 … channel = 6 unchanged
    // Every subset of a finite type; its catalog is the record of one Boolean per member.
    TypeRef set = 7;
    // A total function from a finite key type to a finite value type.
    TotalOf total = 8;
    // A list of at most `max` values of a finite type: a state field's bounded list.
    SeqOf seq = 9;
  }
}
message TotalOf { TypeRef key = 1; TypeRef value = 2; }
message SeqOf   { TypeRef element = 1; int64 max = 2; }

message Value {
  oneof kind {
    // ... bool = 1 … text = 6 unchanged
    SetValue set = 7;      // canonical: members in ascending catalog index, no repeats
    TotalValue total = 8;  // canonical: one entry per key, in key catalog order
  }
}
message SetValue   { TypeRef element = 1; repeated Value items = 2; }
message TotalValue { TypeRef key = 1; TypeRef value = 2; repeated Entry entries = 3; }
message Entry      { Value key = 1; Value value = 2; }
// A bounded list's value is a ListValue.

// expression.proto
message Expr { oneof kind { /* ... inbox = 16 */ Collection collection = 17; } }

// An operation on a bounded collection. `type` is the collection type the operation reads or
// gives; it fixes the catalog that canonical order and the result's domain come from.
message Collection {
  enum Op {
    OP_UNSPECIFIED = 0;
    OP_SET_OF = 1; OP_TOTAL_BY = 2;
    OP_INCL = 3; OP_EXCL = 4; OP_UNION = 5; OP_INTERSECT = 6; OP_DIFF = 7; OP_SUBSET = 8;
    OP_SIZE = 9;
    OP_GET = 10; OP_UPDATED = 11; OP_KEYS_WHERE = 12; OP_MAP_VALUES = 13;
    OP_FORALL = 14; OP_EXISTS = 15; OP_COUNT = 16; OP_SUM = 17; OP_FILTER = 18; OP_MAP = 19;
    OP_HEAD_OPTION = 20; OP_DROP = 21;
  }
  Op op = 1;
  TypeRef type = 2;
  Expr receiver = 3;   // absent for a constructor or a quantifier over a whole catalog
  repeated Expr args = 4;
  Lambda body = 5;     // quantifiers, folds, filters, maps, OP_TOTAL_BY
  TypeRef over = 6;    // the catalog quantified over when there is no receiver
}
```

The IR is "an internal wire with no compatibility promise"
(`proto/internal/temporal/server/api/umpire/v1/ir.proto:3-5`). Every existing IR stays valid, so
`version` stays 0 (`ir.proto:33-35`; `validate.go:27-30`). This is an open question (§10).

### 4.2 Admission (`tools/umpire/ir/validate.go`)

- **`stateField`** (`validate.go:433-441`) admits `set`, `total` and `seq` when their component
  types are themselves admissible state fields, checked recursively, so no `Int`, `List` or
  `Channel` hides inside. `seq.max` must be ≥ 0. This mirrors the `IntRange` emptiness check at
  `validate.go:451-454`.
- **`typeRef`** (`validate.go:443-463`) recurses into the new references, and `Spell`
  (`validate.go:352-370`) spells them `Set[K]`, `Total[K, V]`, `ListUpTo[A, N]`.
- **`catalogs`/`containsItself`** (`validate.go:1083-1137`) follow the new references as they follow
  `list`, so a type holding a `Set` of itself is refused by name rather than looping in `Size`.
- **`value`** (`validate.go:620`) refuses a non-canonical `SetValue` or `TotalValue` literal: a
  repeated or out-of-order member, or a total map missing a key. Keys are spellings, so a
  non-canonical literal would name a state outside the catalog.
- **`expr`** (`validate.go:487-550`) gains a `Collection` case. It checks the op is known (as
  `operator` does, `validate.go:552-556`), that the receiver and `over` are mutually exclusive as the
  op requires, and the lambda's arity: 1, or 2 for a total map's `(k, v)`.
- **`action`** (`validate.go:213-217`) refuses a collection-typed input in Phases 1-5. This keeps
  class keys, `ClassKey` (`machine.go:736-742`), lowering, realization and conformance untouched. A
  collection as an outcome or a fact field is refused for the same reason.
- **`catalog`** (`validate.go:906-915`) admits collections as monitor states and channel messages
  in Phase 6, once the interpreter's checks of monitor-state domain membership cover them.

### 4.3 Interpreter (`tools/umpire/interp`)

Add one file, `interp/collections.go`, as a deep module behind three entry points:
`Members`/`Size` for the new `TypeRef`s, `collection(x, c, env)` for evaluation, and
`canonical(t, items)` for normalization. Inside it, a per-interpreter **catalog index**
(`TypeRef` spelling → `map[key]int`) is built once from `Members` and therefore stays under the
Members ceiling. It is the only place canonical order is decided. Changes elsewhere are one case
each:

- **`eval.go`**
  - new `Kind`s `SetValue`, `TotalValue` (`eval.go:16-27`);
  - `Key` cases spelling §2.3 (`eval.go:50-79`); `spelled` cases (`eval.go:83-101`);
  - `Literal` cases (`eval.go:391-420`);
  - `Conforms` cases (`eval.go:425-461`): members conform and are canonical; a total map has every
    key; a seq's length is at most `max`;
  - `eval` dispatches `Expr_Collection` (`eval.go:297-354`).

  A total map's `Items` are `framework.Entry{key, value}` records. This is the same device as the
  channel's `framework.Delivery` (`channels.go:22-24`; `eval.go:128-132`), so `Equal` needs no change.
- **`machine.go`.** `Members` gains three cases (`machine.go:19-59`):
  - a set lists the record-of-Booleans product, mapping each assignment to its member list;
  - a total map lists the product of `V` per key;
  - a seq reuses `contents`' FIFO enumeration with no redeliveries (`channels.go:42-72`).
- **`limits.go`.** `Size` gains three closed forms (`limits.go:100-135`): `2^|K|` and `|V|^|K|`
  computed with `big.Int.Exp` and saturated by `counted`, and `channelSize(|A|, N, false)` for a seq
  (`limits.go:170-188`). `SizeOfProduct` and `Preflight` then charge them with no further change.
- **`decisions.go`.** Decisions taken inside a quantifier body are marked `Nested`. The `Nested`
  field exists for "decisions taken while another was being decided" (`decisions.go:26-29`). This
  keeps "why is this class disabled" (`decisions.go:57-60`) pointing at the guard's `if`, not at the
  third iteration of a lambda.

`OP_CONTAINS` and `OP_CONCAT` need no change: sets and seqs keep their members in `Items`
(`eval.go:640-648`). A seq that `OP_CONCAT` grows past `max` lands outside the state domain and is
reported at its row, "lands in …, which is outside the state domain" (`machine.go:582-585`). The
same happens today when a step sends to a full channel (SEMANTICS §18, line 1001-1002).

### 4.4 Lint over the table (`tools/umpire/lint/kinds.go`)

`unreachable-value` reads each state field down to its leaves. It never reports a combination,
which is how a channel's contents are treated today (`lint/kinds.go:408-446`). Proposed leaves:

- for a set, one Boolean leaf per member (`tmPrepared[rm1]=true`);
- for a total map, one leaf per key (`rmState[rm2]=committed`);
- a seq is a combination and is never reported.

This reports exactly what the hand encodings report today. Acceptance entries in
`generated/ir.lint.json` keyed by field path change spelling for migrated models.

---

## 5. State-space size: computed, bounded, never silent

### 5.1 The count

Catalog sizes multiply through records (`limits.go:190-200`). With the formulas in §2.1:

| Field | Size |
| --- | --- |
| `Set[K]`, `|K| = 10` | 1,024 |
| `Total[K, V]`, `|K| = |V| = 4` | 256 |
| `Total[K, Option[V]]`, `|K| = |V| = 4` | 625 |
| `ListUpTo[A, 3]`, `|A| = 2` | 15 |

### 5.2 Where it is refused

1. **Go, before listing.** `Preflight` charges the state catalog against `Members` (65,536) and the
   states × classes product against `Evaluations` (1,048,576) (`machine.go:378-393`;
   `limits.go:21`). It returns `LimitError` with the needed count (`limits.go:23-49`). An element
   catalog is charged too, because the catalog index lists it through `Members`.
2. **Query totals.** Every Query's total includes the state catalog (`tools/umpire/ir/totals.go:13-27`).
   The lifter computes the same number (`model/framework/export/Capabilities.scala:1248-1265`).
   The lifter's `size` must gain the three closed forms (`Capabilities.scala:1294-1308`), or it
   refuses to count a state with a collection, as it refuses channels today.
3. **Scala, before listing.** This one is new. The framework lists catalogs eagerly: in the rule
   overlap check (`Machine.scala:338-342`) and in the step tables
   (`model/framework/dsl/StepTables.scala`). `Finite.derived` has no ceiling today
   (`Domain.scala:59-66`). The collection givens must check their closed-form size against the same
   65,536 *before* listing. Otherwise a `Set` over a 20-case enum hangs the Scala build long before
   Go could refuse it. One constant, documented in SEMANTICS §1 and referenced from both sides,
   keeps the two ceilings equal.

### 5.3 Better refusals

`LimitError` says "the catalog needs N members" (`limits.go:40-49`) and nothing more. The proposal
adds a per-field factorization, computed by the same `Size` walk:

```
samples.TwoPhaseCommit.Transaction needs 786432 members, above the ceiling of 65536:
  rmState: Total[ResourceManager, ResourceManagerState] = 4^4 = 256
  × tmState: TransactionManagerState = 3
  × tmPrepared: Set[ResourceManager] = 2^4 = 16
  × msgs: Set[Message] = 2^6 = 64
```

That is TwoPhaseCommit after adding a fourth resource manager: a one-line change with collections,
refused at once. The same change to Bank, a fifth address, gives 3^10 = 59,049 states (under the
Members ceiling). With 5·5·2·4 = 200 classes that is 11,809,800 evaluations, which `Preflight`
refuses as above 1,048,576. Both refusals exist today for the hand encodings; collections only make
the change that triggers them shorter to write.

---

## 6. Interactions

| With | Effect |
| --- | --- |
| **Rules and guards** | §3.4. One rule per action, not per class. The disjointness check keeps its meaning per class and state. Rules that fire one class, `on(rm.prepare(ResourceManager.rm1))`, keep working. |
| **Refinement maps** | Generic: the engine compares mapped state keys (`internal/engine/refine.go`), and canonical spelling makes keys of equal images equal. A Protocol's `Total[RM, RMState]` can map to a Product's `committed: Set[RM]` with `s.rmState.keysWhere(_ == ResourceManagerState.committed)`. `sameNamedKey` cuts fact keys at `-` (`refine.go:271-281`); facts cannot hold collections (§4.2), so it is unaffected. |
| **Monitors** | A monitor state such as "the RMs that have committed so far" is a `Set[RM]`. This needs `catalog` (`validate.go:906-915`) to admit collections, which is Phase 6. Monitor `next`/`violated` may read collections from Phase 3. |
| **Channels** | `Inbox` stays. It carries *derived* delivery, loss and redelivery rows (`channels.go:140-205`), which a `ListUpTo` does not. A `ListUpTo` is a plain bounded buffer whose every step the Model writes. A `Set` message type (`channel[Set[K]]`) needs only `catalog` to admit it (Phase 6). |
| **Compositions** | None. A composed state is a record of member states (SEMANTICS §17), and keys compose by spelling. |
| **Quint export** (`tools/umpire/export/quint.go`) | Phase 5. See the mapping below. Until then the export reports `unsupported` at the node, as it does for channel operations (`quint.go:419-420`), so nothing is skipped silently. |
| **ITF read-back** (`export/itf.go:339-356`) | Decodes `{"#set": [...]}` and `{"#map": [[k, v], ...]}` and **re-canonicalizes** through the catalog index. Quint sets are unordered, and a dump compared key by key with Go must spell members in catalog order. |
| **Step tables and equivalence** | `StepTables.spell` falls back to `toString` for unknown values (`StepTables.scala:51-62`), and `MachineTablePins` spells catalogs through it (`model/temporal/MachineTablePins.test.scala:25, 31-36`). Scala's `HashSet` iteration order is not catalog order. So spelling must go through the catalog. Proposal: `Finite[T]` gains `spell(t: T): String`, and the collection givens spell in their element catalog's order. `StepTables` and the pins call `machine.fs.spell` instead of `spell(Any)`. The rename map is word-based (`model/framework/check/equivalence/Equivalence.scala:31-45`) and cannot map `true-false-true` to `{rm1,rm3}`, so a migration from a hand encoding is sealed by a table-isomorphism test (Phase 4), not by a rename map. |
| **Lowering, realization, conformance** | Untouched while collections are refused as inputs, outcomes and facts. |
| **Explorer and `Why`** | §4.3, `Nested` decisions. |
| **Fingerprints** | State keys feed Definition IDs and the Behavior Fingerprint (SEMANTICS §7). Migrating a model changes them, as any state-type change does. `SetupKey` takes the start state's first field by cutting at the first `-` (`canonical.go:196-201`). A collection in the first field whose members carry fields would cut inside the braces (§10). |

**Quint mapping (Phase 5).**

| IR | Quint |
| --- | --- |
| `Set[K]` | `Set[K]`; literal `Set(a, b)` |
| `Total[K, V]` | `K -> V`; literal `Map(a -> x, …)` |
| `ListUpTo[A, N]` | `List[A]` |
| `OP_INCL` / `OP_EXCL` | `s.union(Set(x))` / `s.exclude(Set(x))` |
| `OP_CONTAINS` on a set | `s.contains(x)`, not the list encoding at `quint.go:480-482` |
| `OP_SIZE` | `s.size()` / `l.length()` |
| `OP_FORALL` / `OP_EXISTS` | `s.forall(x => …)` / `s.exists(x => …)`; over a catalog, `members_K.forall(…)` using `membersOf` (`quint.go:227-276`) |
| `OP_COUNT` / `OP_SUM` | `s.filter(p).size()` / `s.fold(0, (a, x) => a + f(x))` |
| `OP_TOTAL_BY` | `members_K.mapBy(k => …)` |
| `OP_GET` / `OP_UPDATED` | `t.get(k)` / `t.set(k, v)` |
| `OP_HEAD_OPTION` / `OP_DROP` | `if (l.length() > 0) Some(l.head()) else None`, written in the synthetic Option encoding / `l.slice(n, l.length())` |

`membersOf` is needed only for inputs (`quint.go:227-229`), which collections are not.

---

## 7. Show it working

Neither rewrite changes a catalog size, so neither changes a Query total or a ceiling:

- **Bank**: `Coins` is 3² = 9, `Balances` 9⁴ = 6,561 states; with 4·4·2·4 = 128 classes, 839,808
  evaluations. Before and after.
- **TwoPhaseCommit**: `States` 4³ = 64 × `tmState` 3 × `Members` 2³ = 8 × `Messages` 2·2·8 = 32,
  giving 49,152 states. After: `Total` 4³ × 3 × `Set[RM]` 2³ × `Set[Message]` 2⁵ = 49,152. 14
  classes, 688,128 evaluations.

### 7.1 TwoPhaseCommit

**Before** (`TwoPhaseCommit.scala:23-80, 172-183, 203-239`, abridged):

```scala
final case class States(rm1: ResourceManagerState, rm2: ResourceManagerState, rm3: ResourceManagerState)
final case class Members(rm1: Boolean, rm2: Boolean, rm3: Boolean)
final case class Messages(abort: Boolean, commit: Boolean, prepared: Members)
final case class Transaction(rmStates: States, tmState: TransactionManagerState,
    tmPrepared: Members, messages: Messages)

def stateOf(states: States, rm: ResourceManager): ResourceManagerState = rm match
  case ResourceManager.rm1 => states.rm1
  case ResourceManager.rm2 => states.rm2
  case ResourceManager.rm3 => states.rm3
def withState(...)   // three arms
def contains(...)    // three arms
def withMember(...)  // three arms
def all(members: Members): Boolean = members.rm1 && members.rm2 && members.rm3

object effects:
  def prepareRm1(state: Transaction) = prepared(state, ResourceManager.rm1)
  def prepareRm2(state: Transaction) = prepared(state, ResourceManager.rm2)
  def prepareRm3(state: Transaction) = prepared(state, ResourceManager.rm3)
  // ... nine more

object rules extends Rules:
  on(rm.prepare(ResourceManager.rm1))(
    where(s => mayPrepare(s, ResourceManager.rm1)) ~> effects.prepareRm1
  )
  on(rm.prepare(ResourceManager.rm2))(
    where(s => mayPrepare(s, ResourceManager.rm2)) ~> effects.prepareRm2
  )
  // ... ten more
  on(tm.decideOnCommit)(
    where(s => s.tmState == TransactionManagerState.init && all(s.tmPrepared)) ~>
      effects.decideOnCommit
  )
```

**After:**

```scala
// the messages of TwoPhase.tla: Abort, Commit and RMPrepared(rm)
enum Message:
  case abort, commit
  case prepared(rm: ResourceManager)

final case class Transaction(
    rmState: Total[ResourceManager, ResourceManagerState],
    tmState: TransactionManagerState,
    // the resource managers from which the transaction manager has received RMPrepared
    tmPrepared: Set[ResourceManager],
    // the messages sent so far: never lost and never consumed
    msgs: Set[Message]
)

// A resource manager prepares while it works.
def mayPrepare(s: Transaction, rm: ResourceManager): Boolean =
  s.rmState(rm) == ResourceManagerState.working

// A resource manager aborts spontaneously while it works, or because of an Abort message.
def mayAbort(s: Transaction, rm: ResourceManager): Boolean =
  s.rmState(rm) == ResourceManagerState.working || s.msgs.contains(Message.abort)

// A resource manager commits because of a Commit message, once.
def mayCommit(s: Transaction, rm: ResourceManager): Boolean =
  s.msgs.contains(Message.commit) && s.rmState(rm) != ResourceManagerState.committed

// The transaction manager receives an RMPrepared message.
def mayReceive(s: Transaction, rm: ResourceManager): Boolean =
  s.tmState == TransactionManagerState.init && s.msgs.contains(Message.prepared(rm))

def allPrepared(s: Transaction): Boolean =
  ResourceManager.values.forall(rm => s.tmPrepared.contains(rm))

def allCommitted(s: Transaction): Boolean =
  s.rmState.forall((_, st) => st == ResourceManagerState.committed)

// two resource managers have not arrived at conflicting decisions
def consistent(s: Transaction): Boolean =
  !(s.rmState.exists((_, st) => st == ResourceManagerState.committed) &&
    s.rmState.exists((_, st) => st == ResourceManagerState.aborted))

object Model extends Machine[Transaction, Outcome, Nothing]:
  val init = Transaction(
    rmState = Total.fill(ResourceManagerState.working),
    tmState = TransactionManagerState.init,
    tmPrepared = Set.empty,
    msgs = Set.empty
  )
  // the transaction is over once it is decided and every resource manager has followed
  def end(s: Transaction) =
    s.tmState != TransactionManagerState.init && s.rmState.forall((_, st) => done(st))

  object states:
    // the transaction manager has heard from any one resource manager
    def eagerlyCommits(s: Transaction): Boolean =
      s.tmState == TransactionManagerState.init && s.tmPrepared.nonEmpty

  object effects:
    // a resource manager prepares and tells the transaction manager
    def prepare(s: Transaction, rm: ResourceManager) = enter(s.copy(
      rmState = s.rmState.updated(rm, ResourceManagerState.prepared),
      msgs = s.msgs.incl(Message.prepared(rm))))
    // there is no need to let others know that a resource manager aborted
    def abort(s: Transaction, rm: ResourceManager) =
      enter(s.copy(rmState = s.rmState.updated(rm, ResourceManagerState.aborted)))
    def commit(s: Transaction, rm: ResourceManager) =
      enter(s.copy(rmState = s.rmState.updated(rm, ResourceManagerState.committed)))
    def receive(s: Transaction, rm: ResourceManager) =
      enter(s.copy(tmPrepared = s.tmPrepared.incl(rm)))
    def decideOnCommit(s: Transaction) = enter(s.copy(
      tmState = TransactionManagerState.committed, msgs = s.msgs.incl(Message.commit)))
    def decideOnAbort(s: Transaction) = enter(s.copy(
      tmState = TransactionManagerState.aborted, msgs = s.msgs.incl(Message.abort)))

  object rules extends Rules:
    on(rm.prepare) {
      where((s, r) => mayPrepare(s, r)) ~> effects.prepare
    }
    on(rm.abort) {
      where((s, r) => mayAbort(s, r)) ~> effects.abort
    }
    on(rm.commit) {
      where((s, r) => mayCommit(s, r)) ~> effects.commit
    }
    on(tm.addPrepared) {
      where((s, r) => mayReceive(s, r)) ~> effects.receive
    }
    on(tm.decideOnCommit) {
      where(s => s.tmState == TransactionManagerState.init && allPrepared(s)) ~> effects.decideOnCommit
    }
    on(tm.decideOnAbort) {
      where(_.tmState == TransactionManagerState.init) ~> effects.decideOnAbort
    }
```

The rules shrink from 14 blocks to 6, and the effects from 14 defs to 6. The seven ladder helpers
(`stateOf`, `withState`, `contains`, `withMember`, `all`, `any`, `someAre`) disappear. The guard
lambda's parameter is `r` because `rm` names the actor object `rm` (`TwoPhaseCommit.scala:103-106`).
The `Message` enum is the Quint model's message type, so `msgs: Set[Message]` reads as the original
`msgs` variable. `properties`, `EagerCommit` and `queries` are unchanged apart from calling these
helpers. `everyoneCommits` keeps its class selector `rm.commit(ResourceManager.rm3)`.

### 7.2 Bank

**Before** (`Bank.scala:24-57`, abridged):

```scala
final case class Coins(banana: UpTo[2], burger: UpTo[2])
final case class Balances(king: Coins, donkeykong: Coins, peaches: Coins, mario: Coins)

def coinsOf(balances: Balances, addr: Addr): Coins = addr match
  case Addr.king       => balances.king
  case Addr.donkeykong => balances.donkeykong
  case Addr.peaches    => balances.peaches
  case Addr.mario      => balances.mario
def amountOf(coins: Coins, denom: Denom): Int = ...        // two arms
def withAmount(coins: Coins, denom: Denom, amount: Int)    // two arms
def withBalance(balances: Balances, addr: Addr, ...)       // four arms
def sumForDenom(balances: Balances, denom: Denom): Int =
  balanceOf(balances, Addr.king, denom) + balanceOf(balances, Addr.donkeykong, denom) +
    balanceOf(balances, Addr.peaches, denom) + balanceOf(balances, Addr.mario, denom)

val init = Balances(
  king = Coins(banana = UpTo(2), burger = UpTo(2)),
  donkeykong = Coins(banana = UpTo(0), burger = UpTo(0)),
  peaches = Coins(banana = UpTo(0), burger = UpTo(0)),
  mario = Coins(banana = UpTo(0), burger = UpTo(0))
)
```

**After:**

```scala
// the coins of one account, per denomination; zero coins is no balance at all
type Coins = Total[Denom, UpTo[2]]

// the balances of every address
final case class Balances(of: Total[Addr, Coins])

def balanceOf(balances: Balances, addr: Addr, denom: Denom): Int = balances.of(addr)(denom)

def withBalance(balances: Balances, addr: Addr, denom: Denom, amount: Int): Balances =
  balances.copy(of = balances.of.updated(addr, balances.of(addr).updated(denom, UpTo(amount))))

// sum up amounts over all balances for a given denomination
def sumForDenom(balances: Balances, denom: Denom): Int =
  balances.of.sumOf((_, coins) => coins(denom))

// the total supply of the network is equal to the sum of all coins from the accounts
def totalSupply(balances: Balances): Boolean =
  Denom.values.forall(d => sumForDenom(balances, d) == supply)

object Model extends Machine[Balances, Outcome, Nothing]:
  // the king has it all
  val init = Balances(Total.by[Addr, Coins](a =>
    Total.fill[Denom, UpTo[2]](UpTo(if a == Addr.king then supply else 0))))
```

`effects.send`, `sendFromNowhere`, the rules, `Leaky`, the Properties and the Queries are unchanged
(`Bank.scala:84-158`). They call `balanceOf` and `withBalance` only. `peachesGetsBananas` still
reads `balanceOf(after.state, Addr.king, Denom.banana)`. The explicit type arguments in `init` avoid
asking Scala to infer `N` of `UpTo` through two generic calls (§10).

---

## 8. Alternatives considered and rejected

### 8.1 Desugar at lift time into generated case classes, with no IR change

The lifter would rewrite each collection into a record:

- a set becomes a record of `Boolean` fields;
- a total map, a record with one field per key;
- dynamic access, a `Match` on the key;
- a quantifier, an unrolled chain.

`Option`'s synthetic enum (`Types.scala:144-169`) shows the lifter can do this.

*For:* no Go, proto or Quint change, and the existing `unreachable-value` leaves work as they are.

*Against:*

1. **Lists cannot be desugared without junk states.** A record of `N` optional slots plus a length
   admits non-canonical fillings, and the interpreter evaluates every catalog state
   (`machine.go:529-568`). Channels became a native `TypeRef` with an exact catalog for exactly this
   reason (`channels.go:42-72`).
2. **IR size grows with every access.** Each dynamic get or update is a `|K|`-arm match, and nested
   updates multiply. The rule lowering's input ladder (`Declarations.scala:1191-1201`) shows the
   shape.
3. **Keys become `true-false-true`.** They would appear in tables, witnesses, lint findings and
   acceptance files, and the spelling would then change a second time if the IR ever went native.
4. **Quint would see records**, so its `Set`, `mapBy`, `forall` and `fold` would go unused.
5. **Field names come from key spellings**, which are not always valid identifiers (`Some-x`, `3`).
   They would need an escaping scheme that every tool has to agree on.

*What is kept:* the desugaring is the **semantic definition** (§2.1, §3.2) and the **test oracle**
(§9).

### 8.2 Native IR collections in full generality

General `List`, `Map`, `Set` values as in Quint, with unbounded fold and recursion. Rejected:
SEMANTICS §1 requires every state type to have a finite, ordered catalog, and §20 keeps step
functions non-recursive (`model/SEMANTICS.md:356`; `validate.go:180-211`). This proposal is native
only in *representation*. Every collection type still has an exact finite catalog, quantifiers
range over finite members, and there is no recursion.

### 8.3 Symmetry reduction, or a parametric N

- *Symmetry.* Treating `rm1..rm3` as interchangeable would shrink the explored graph. But the
  reader derives its tables by walking the **whole catalog** (`machine.go:529-568`), so symmetry
  would reduce search but not table derivation. `.plans/UMPIRE4_INSPIRE.md:907-908` says any
  symmetry "should be refused for progress claims until its soundness is argued", and
  TwoPhaseCommit has one (`TwoPhaseCommit.scala:250-259`). The same note says symmetry "is not
  needed yet" (`.plans/UMPIRE4_INSPIRE.md:1018`). Collections make a later symmetry declaration
  easier to state, because the key type `K` of a `Set[K]` is explicit, but they do not depend on one.
- *Parametric N*, proving for every number of resource managers, needs cut-off theorems or
  parameterized verification. That is a different checker from a bounded, finite one. Out of scope.

### 8.4 Alternatives: native Scala collections (`Set`, `Map`, `List`/`Vector`)

The question is whether the DSL should use Scala's own `Set[K]`, `Map[K, V]` and `List[A]` (or
`Vector[A]`) for all three collections, in place of the new `Total[K, V]` and `ListUpTo[A, N]`.
This section compares three options:

- **(a) native everywhere.** Each Scala type means what its type admits: `Set[K]` is every subset,
  `Map[K, V]` every *partial* map, and `List[A]` a list bounded by a library-wide rule.
- **(b) native plus explicit `given` bounds.** The types are native. A `given Finite[...]` beside the
  state chooses each catalog: `Map` total (or partial), and `List` bounded by `n`.
- **(c) the current mix.** Native `Set[K]`, `Total[K, V]` and `ListUpTo[A, N]`, so totality and the
  bound live in the type.

For lists, (a) and (b) coincide. A `List[A]` has no finite catalog without a bound, and a
library-wide default bound would be a hidden constant in every catalog. Under (a), the bound must
come from somewhere outside the type too, which is (b).

#### 8.4.1 Mechanics shared by every option

- **The `Finite` instance.** `Set`, `Map`, `List` and `Vector` are not `Mirror` types, so a given in
  `object Finite`, beside `Option`'s (`model/framework/dsl/Domain.scala:42-43`), does not compete
  with the derived given (`Domain.scala:53-54`). A Model's own `given Finite[S]` block wins over the
  derived one. That is how a state binds its `Inbox` catalog today (`Domain.scala:45-58`;
  `model/SEMANTICS.md:1014-1019`).
- **CanEqual.** Every option gets `CanEqual[T, T]` from the blanket given wherever a `Finite[T]`
  exists (`Domain.scala:23`). Under `-language:strictEquality` (`model/SEMANTICS.md:85-88`), `==` on
  a collection compiles exactly where a catalog exists, as for every finite type today. Scala's
  `Set`/`Map`/`List` equality is by content, and `Total`/`ListUpTo` are opaque over `Map`/`List`,
  so runtime equality agrees with Go's canonical-form equality (`tools/umpire/interp/eval.go:103-120`)
  in every option.
- **Spelling.** Scala's hash collections iterate in no catalog order, and `StepTables.spell` falls
  back to `toString` (`model/framework/dsl/StepTables.scala:51-62`). Every option therefore needs
  catalog-aware spelling (`Finite.spell`, §6). The IR spellings of §2.3 (`{a,b}`, `{k:v}`, `[a,b]`)
  serve every option.
- **The lifter recognizes types in one place.** `typeRef` (`model/framework/export/Types.scala:40-84`):
  - immutable `List`/`Seq` are "lists" (`Types.scala:21-23, 67`) and refused as fields
    (`Types.scala:110-117`);
  - `Map`, `Set` and `Vector` reach `declareType` and are refused as "neither an enum nor a case
    class" (`Types.scala:136-141`);
  - opaque types are read before dealiasing (`Types.scala:43-55`).
- **Library operations are allowlisted.** The lifter refuses a `scala.`/`java.` operation unless
  `LiftableSubset.permitted` admits it (`model/framework/export/Expressions.scala:496-499`;
  `model/framework/inspect/LiftableSubset.scala:30-75`). Native types mean allowlisting part of a
  large Scala API. Framework types mean matching a few framework extension methods.
- **Scala evaluates Model code over whole catalogs before any lift.** This matters for partial
  operations:
  - the rule overlap check evaluates every guard on every catalog state
    (`model/framework/dsl/Machine.scala:338-342`);
  - `Refinement.unclosed` evaluates `toProduct` on every catalog state
    (`model/framework/dsl/Refine.scala:21-29`);
  - the step tables run every effect on every catalog state (`StepTables.scala:14-21`).

  An expression that throws on *some* catalog state, even an unreachable one, therefore fails the
  build in Scala, not in Go.

#### 8.4.2 `Set[K]`

| | (a) native | (b) native + given | (c) current mix |
| --- | --- | --- | --- |
| Authoring | `tmPrepared: Set[ResourceManager]` | same | same |
| Catalog rule | the record of one `Boolean` per key | same (a given could restrict it, but nothing needs to) | same |
| IR value | `{rm1,rm3}` | same | same |
| Equality / CanEqual | Scala content equality; `Finite` blanket `CanEqual` | same | same |
| Lifting operations | §3.1 (`contains`, `incl`, `excl`, `union`, `forall`, …) | same | same |
| Where the bound comes from | `K`'s catalog (`2^|K|`) | same | same |

All three options agree on `Set`. Over a finite `K` the powerset is the only sensible catalog, so
there is no bound or totality to choose. That is why `Set` is native in (c) too.

#### 8.4.3 `Map[K, V]`

| | (a) native, partial | (b) native, total by `given` | (c) `Total[K, V]` |
| --- | --- | --- | --- |
| Authoring | `rmState: Map[RM, RMState]` | `rmState: Map[RM, RMState]` plus `given Finite[Map[RM, RMState]] = Finite.total` in the state's `given` block | `rmState: Total[RM, RMState]` |
| Catalog rule | the record of one `Option[V]` per key: `(|V|+1)^|K|` | the record of one `V` per key: `|V|^|K|` | the record of one `V` per key: `|V|^|K|` |
| IR value | `{rm1:working}`, present keys only | `{rm1:working,rm2:…,rm3:…}` | same as (b) |
| IR type | `TypeRef.total{K, Option[V]}` (§4.1), so no new type is needed | `TypeRef.total{K, V}` | same as (b) |
| Equality / CanEqual | content equality; blanket `CanEqual` | same | same |
| `m(k)` | throws `NoSuchElementException` in Scala on an absent key: must be refused | safe on catalog states | total by type |
| `m.get(k)` | lifts, giving the synthetic `Option` enum (`Types.scala:144-169`) | lifts (always `Some`) | not offered (use `t(k)`) |
| `updated` / `removed` | lift (`OP_UPDATED`, a new `OP_REMOVED`) | `updated` lifts; `removed` must be refused or it leaves the catalog | `updated` lifts; no `removed` exists |
| `forall` / `exists` / `count` | take a tuple `((K, V)) => Boolean`, and tuples are no IR type (open question below) | same | `(k, v) => …`, no tuple |
| Where totality comes from | nowhere (partial) | a `given` keyed by type | the type |

**(a) is refused by the ceilings in both samples.** The reader evaluates every catalog state
(`tools/umpire/interp/machine.go:529-568`) within 65,536 members and 1,048,576 evaluations
(`tools/umpire/interp/limits.go:21`; `machine.go:378-393`):

| Sample | Field | Catalog (b)/(c) | Catalog (a) | Outcome under (a) |
| --- | --- | --- | --- | --- |
| Bank | `Map[Addr, Map[Denom, UpTo[2]]]` | 9⁴ = 6,561 | (4²+1)⁴ = 83,521 | refused: above 65,536 members |
| TwoPhaseCommit | `Map[RM, RMState]` beside `tmState`, `tmPrepared`, `msgs` | 49,152 | 125 × 3 × 8 × 32 = 96,000 | refused: above 65,536 members and 1,048,576 evaluations (1,344,000) |

The partial catalog also gives Bank two spellings of "no coins", an absent key and a zero. Bank's
own comment says "zero coins is no balance at all" (`model/framework/samples/Bank.scala:23`). Such
a model must normalize after every step, or `failedSendChangesNothing`'s `after.state == before`
(`Bank.scala:139-141`) compares two representations of one balance.

**(b) needs the lifter to read catalogs from `given` blocks.** There is precedent: `readFinites`
reads `Finite.upTo(n)` for opaque types and `<channel>.contents` inside state `given` blocks
(`Types.scala:187-202`). But that mechanism is keyed by (state type, element type)
(`Types.scala:195-198`). Two `Map[K, V]` fields of one state cannot differ, one total and one
partial, without a new keying, for example by field name. That keying is not available to the
Scala `given`, which implicit search resolves by type.

**Variant of (c).** `opaque type Total[K, V] <: Map[K, V]` would inherit `Map`'s reading API. But
`updated` would return `Map`, so every write needs re-wrapping, `Total(s.rmState.updated(rm, x))`,
plus a runtime totality check. Rejected: the commonest operation becomes the noisiest.

#### 8.4.4 `List[A]` / `Vector[A]`

| | (a)/(b) native + `given` bound | (b′) native + field annotation | (c) `ListUpTo[A, N]` |
| --- | --- | --- | --- |
| Authoring | `history: List[Op]` plus `given Finite[List[Op]] = Finite.listUpTo(3)` in the state's `given` block | `@bound(3) history: List[Op]` | `history: ListUpTo[Op, 3]` |
| Catalog rule | FIFO channel contents of capacity `n`, no duplicates | same | same (`Channel.scala:38-50`; `channels.go:42-72`) |
| IR value | `[write-0,read-Some-0]` | same | same |
| IR type | `TypeRef.seq{A, n}`, with `n` read from the given | `n` read from the annotation | `n` read from the type argument, as `upToBound` does (`Types.scala:89-102`) |
| Equality / CanEqual | list equality; blanket `CanEqual` | same | same |
| Lifting operations | `contains`, `:+`, `+:`, `++`, `size`, `isEmpty`, `forall`/`exists`/`count`, `headOption`, `drop` (§3.1); `head`, `tail`, `apply(i)` refused | same | same |
| Where the bound comes from | a given, keyed by `List[Op]` | an annotation the lifter reads | the type |

- **(b′) cannot work in Scala.** The framework derives catalogs by implicit search through
  `Mirror`s (`Domain.scala:53-83`), and a `Mirror` carries no field annotations. Scala could not
  list the catalog for the overlap check or the step tables without a macro, and
  `.plans/DSL_OPERATORS.md` rule 5 forbids macros on the author surface. Annotations are read only
  by the lifter today (`@gap`, `model/framework/dsl/Gaps.scala:8`; `model/SEMANTICS.md:457-467`).
- **(a)/(b) loses coherence and position.** The bound belongs to a given, not to a field:
  - every `List[Op]` in scope has the same bound;
  - a stray given in scope silently changes a catalog;
  - `List` is also the type of a step's facts (`Machine.scala:10`) and of the unbounded IR list
    (`proto/internal/temporal/server/api/umpire/v1/value.proto:51-52`). The lifter would have to
    tell the two apart by position alone.
- **The repo has decided this question twice already, for `Int`.** "A plain `Int` field is refused
  at its declaration, even with a local `Finite[Int]`: write `UpTo[N]`" (`model/SEMANTICS.md:81-82`;
  `Types.scala:60-66`). "A plain `Int` message is refused … even with `Finite.upTo(n)`"
  (`model/SEMANTICS.md:1022-1023`).
- **The IR, totals and admission lose nothing** under (a)/(b) if the lifter finds the given: it emits
  the same `SeqOf{n}`. The lifter's counter needs `n` either way (`Capabilities.scala:1294-1308`).
  The loss is in authoring and Scala-side checking.
- `Vector` behaves exactly like `List` here and needs its own lifter recognition and allowlist.

#### 8.4.5 The same fragments in each option

**TwoPhaseCommit: state, one helper, one effect, one rule.**

```scala
// (a) native everywhere: refused, 96,000 states
final case class Transaction(rmState: Map[ResourceManager, ResourceManagerState],
    tmState: TransactionManagerState, tmPrepared: Set[ResourceManager], msgs: Set[Message])
def mayPrepare(s: Transaction, rm: ResourceManager): Boolean =
  s.rmState.get(rm) == Some(ResourceManagerState.working)        // apply would throw in the overlap check
def allCommitted(s: Transaction): Boolean =
  ResourceManager.values.forall(rm => s.rmState.get(rm) == Some(ResourceManagerState.committed))
val init = Transaction(rmState = Map(ResourceManager.rm1 -> ResourceManagerState.working,
    ResourceManager.rm2 -> ResourceManagerState.working, ResourceManager.rm3 -> ResourceManagerState.working),
  tmState = TransactionManagerState.init, tmPrepared = Set.empty, msgs = Set.empty)
def prepare(s: Transaction, rm: ResourceManager) = enter(s.copy(
  rmState = s.rmState.updated(rm, ResourceManagerState.prepared), msgs = s.msgs.incl(Message.prepared(rm))))
on(rm.prepare) {
  where((s, r) => mayPrepare(s, r)) ~> effects.prepare
}
```

```scala
// (b) native + given: 49,152 states
final case class Transaction(rmState: Map[ResourceManager, ResourceManagerState],
    tmState: TransactionManagerState, tmPrepared: Set[ResourceManager], msgs: Set[Message])
given Finite[Transaction] =
  given Finite[Map[ResourceManager, ResourceManagerState]] = Finite.total
  Finite.derived
def mayPrepare(s: Transaction, rm: ResourceManager): Boolean =
  s.rmState(rm) == ResourceManagerState.working                  // safe on catalog states only
def allCommitted(s: Transaction): Boolean =
  s.rmState.forall((_, st) => st == ResourceManagerState.committed)  // untupled; lift shape unverified
val init = Transaction(rmState = Finite.tabulate[ResourceManager](_ => ResourceManagerState.working), ...)
def prepare(s: Transaction, rm: ResourceManager) = ... // as (a); `s.rmState.removed(rm)` compiles, Go refuses its row
```

```scala
// (c) current mix: 49,152 states (§7.1)
final case class Transaction(rmState: Total[ResourceManager, ResourceManagerState],
    tmState: TransactionManagerState, tmPrepared: Set[ResourceManager], msgs: Set[Message])
def mayPrepare(s: Transaction, rm: ResourceManager): Boolean =
  s.rmState(rm) == ResourceManagerState.working
def allCommitted(s: Transaction): Boolean =
  s.rmState.forall((_, st) => st == ResourceManagerState.committed)
val init = Transaction(rmState = Total.fill(ResourceManagerState.working), ...)
def prepare(s: Transaction, rm: ResourceManager) = ... // as (a); no `removed` exists
```

The rule is the same in all three. The guard reading the input (§3.4) is independent of the choice.

**Bank: balances and `send`'s helpers.** `effects.send` itself (`Bank.scala:88-109`) is unchanged
in every option, because it calls only `balanceOf` and `withBalance`.

```scala
// (a) native everywhere: refused, 83,521 states; absent and zero both mean "no coins"
final case class Balances(of: Map[Addr, Map[Denom, UpTo[2]]])
def balanceOf(b: Balances, addr: Addr, denom: Denom): Int = b.of.get(addr) match
  case Some(coins) => coins.getOrElse(denom, UpTo[2](0))
  case None        => 0
def withBalance(b: Balances, addr: Addr, denom: Denom, amount: Int): Balances =
  b.copy(of = b.of.updated(addr,
    b.of.getOrElse(addr, Map.empty[Denom, UpTo[2]]).updated(denom, UpTo(amount))))
```

```scala
// (b) native + given: 6,561 states
final case class Balances(of: Map[Addr, Map[Denom, UpTo[2]]])
given Finite[Balances] =
  given Finite[Map[Denom, UpTo[2]]] = Finite.total
  given Finite[Map[Addr, Map[Denom, UpTo[2]]]] = Finite.total
  Finite.derived
def balanceOf(b: Balances, addr: Addr, denom: Denom): Int = b.of(addr)(denom)
def withBalance(b: Balances, addr: Addr, denom: Denom, amount: Int): Balances =
  b.copy(of = b.of.updated(addr, b.of(addr).updated(denom, UpTo(amount))))
```

```scala
// (c) current mix: 6,561 states (§7.2)
type Coins = Total[Denom, UpTo[2]]
final case class Balances(of: Total[Addr, Coins])
def balanceOf(b: Balances, addr: Addr, denom: Denom): Int = b.of(addr)(denom)
def withBalance(b: Balances, addr: Addr, denom: Denom, amount: Int): Balances =
  b.copy(of = b.of.updated(addr, b.of(addr).updated(denom, UpTo(amount))))
```

**TwoLayeredCache, if it kept Quint's history list.** The sample replaces the log with `Event` facts
and two monitors (`model/framework/samples/TwoLayeredCache.scala:16-18, 56-78`). Kept as a list, the
history is:

```scala
enum Op:
  case write(value: UpTo[2])
  case read(value: Option[UpTo[2]])
```

`Op` has 3 + 4 = 7 values. The catalog of `Cache` is `l1` 4 × `l2` 4 × `num` 4 = 64
(`TwoLayeredCache.scala:14`), times the history catalog. The history catalog is 1 + 7 + 7² + 7³ =
400 for a bound of 3, giving 25,600 states. A bound of 4 gives 2,801 and 179,264 states, refused.
The size tells the author why the sample uses monitors.

```scala
// (a)/(b) native + given
final case class Cache(l1: Option[UpTo[2]], l2: Option[UpTo[2]], num: UpTo[3], history: List[Op])
given Finite[Cache] =
  given Finite[List[Op]] = Finite.listUpTo(3)
  Finite.derived
def write(cache: Cache) = enter(cache.copy(l1 = Some(UpTo(cache.num)), l2 = Some(UpTo(cache.num)),
  num = UpTo(cache.num + 1), history = cache.history :+ Op.write(UpTo(cache.num))), Event.wrote)
on(client.write)(where(c => c.num < 3 && c.history.size < 3) ~> effects.write)
// Forgetting `c.history.size < 3`: Scala appends a 4th entry silently; Go refuses the row.
```

```scala
// (c) ListUpTo
final case class Cache(l1: Option[UpTo[2]], l2: Option[UpTo[2]], num: UpTo[3], history: ListUpTo[Op, 3])
def write(cache: Cache) = enter(cache.copy(l1 = Some(UpTo(cache.num)), l2 = Some(UpTo(cache.num)),
  num = UpTo(cache.num + 1), history = ListUpTo(cache.history :+ Op.write(UpTo(cache.num)))), Event.wrote)
on(client.write)(where(c => c.num < 3 && c.history.size < 3) ~> effects.write)
// Forgetting the guard: `ListUpTo(...)` fails its `require` in Scala at this line; Go refuses the row too.
```

#### 8.4.6 Failure modes for authors, and where each is caught

| Mistake | (a) native | (b) native + given | (c) current mix |
| --- | --- | --- | --- |
| **Reading an absent `Map` key** with `m(k)` | Scala throws during framework construction, because the overlap check evaluates guards on every catalog state, absent keys included (`Machine.scala:338-342`), or later in `Refinement.unclosed` (`Refine.scala:27-29`) or the step tables. The lifter must also refuse `apply`. | Cannot happen on catalog states. A step that first leaves the catalog (`removed`) is refused by Go at its row (`machine.go:582-585`). | Cannot be written: `Total.apply` is total. **Compile time.** |
| **Out-of-bound append** | as (b) | Not caught in Scala. Go refuses the row, "lands in …, which is outside the state domain" (`machine.go:582-585`), at the rule binding's position (`machine.go:584`, `c.at`). | `ListUpTo(...)` `require`s the bound where the value is built, in Scala, at the write's line. That is the `UpTo` mechanism (`Domain.scala:92-94`). Go refuses the row as well. |
| **Accidental unbounded `List` field** | No `Finite[List[A]]`: a **compile** error at the first summon (`Domain.scala:14-18`). If some `Finite.listUpTo` given is in scope for another field of the same type, it applies **silently**. | same | A `List` field has no `Finite`: **compile** error. The lifter refuses list fields (`Types.scala:110-117`), and admission refuses `List` state fields (`tools/umpire/ir/validate.go:433-441`). |
| **`Int` element** (`Set[Int]`, `Map[Int, V]`, `List[Int]`) | No `Finite[Int]` given exists (`Finite.upTo` is a method, `Domain.scala:32-34`): **compile** error. A local `given Finite[Int]` compiles, but the lifter refuses an `Int` field (`Types.scala:60-66`, extended to collection elements) and admission refuses `TypeRef.int` in a state field (`validate.go:433-441`). | same | same |
| **Partial value under a "total" rule** (`Map.empty`, a short literal) | not applicable | Go refuses a start outside the state domain (`machine.go:674-676`) or a row (`machine.go:582-585`). | Not expressible: `Total` has no `empty`, and `fill`/`by` cover every key. |
| **Order-dependent read** (`s.toList.head`) | Refused at **lift** by the allowlist (§3.1). Scala alone would be nondeterministic across set sizes. | same | same for `Set`; `Total` exposes no order |
| **Catalog explosion** (a 5th address, a 4th RM) | Go refuses before listing (`machine.go:378-393`). With the proposed Scala ceiling (§5.2), the framework refuses at **construction**. | same | same |
| **Hidden second representation** (absent and zero) | possible; caught only by Properties the author writes | not possible in a total catalog | not possible |

#### 8.4.7 Interop

| With | (a) native | (b) native + given | (c) current mix |
| --- | --- | --- | --- |
| **Monitors** | A monitor's state needs a `Finite[M]` (`model/framework/dsl/Monitor.scala:33-44`) and admission's `catalog` check (`validate.go:906-915`, to be relaxed in Phase 6). A partial `Map` monitor state has the larger catalog. | The bounding given must be in scope where the monitor is declared, not only where the state is. A monitor's `Seen`-like record (`TwoLayeredCache.scala:35`) needs its own `given` block. | The type carries its catalog wherever it goes. |
| **Channels** | `Inbox` is already a bounded list with *derived* delivery rows (`interp/channels.go:140-205`). A `List` message type needs a bound in scope at the channel declaration, which takes its `Finite[M]` there (`model/framework/dsl/Channel.scala:84-86`). | same | A `ListUpTo` message carries its bound. `Inbox` stays the derived-delivery form. A `ListUpTo` field is a buffer whose every step the Model writes, with the same catalog as FIFO contents. |
| **Refinement maps** | The engine compares mapped keys (`internal/engine/refine.go`). Scala evaluates `toProduct` on every catalog state (`Refine.scala:27-29`), so a map using `m(k)` throws on absent keys. | safe | safe |
| **Quint export** | Partial `Map` is Quint's native map (`keys()`, `get`). The most natural fit. | `K -> V` with `keys() == K`. | same as (b). Lists are `List[A]` in every option (§6). |
| **Step tables, equivalence** | Spelling through `Finite.spell` in all options (§6). `MachineTablePins` spells `machine.fs.values` (`model/temporal/MachineTablePins.test.scala:31-36`). | The given that `machine.fs` resolved is the one tabled, so a stray given changes the pins. | The catalog follows from the type alone. |
| **Totals counter** | The lifter's `size` (`model/framework/export/Capabilities.scala:1294-1308`) needs `(|V|+1)^|K|`. | It needs the bound read out of the given (`readFinites`-style, `Types.scala:191-202`), or it refuses to count the Query, as it does for channels (`Capabilities.scala:1227-1243`). | It reads the bound from type arguments, like `upToBound` (`Types.scala:89-102`). |

#### 8.4.8 Migration cost, and keeping the door open

**Common to all options:** the Go and IR work of §4 and the samples' rewrite. Option-specific Scala
work:

- **(a)**: allowlist `Map`'s and `List`'s APIs, recognize Scala 3's untupled lambda tree, refuse
  `apply`/`head`. Both samples then exceed the ceilings, so they cannot migrate.
- **(b)**: everything in (a), minus partiality, plus:
  - `Finite.total`, `Finite.listUpTo`, `Finite.tabulate`;
  - lifter discovery of those givens, with new per-field keying (`Types.scala:191-202` keys by type);
  - a `given` block in every state type that holds a map or a list.
- **(c)**: two small framework APIs (`Total`, `ListUpTo`), matched by name like `UpTo`
  (`Expressions.scala:583-588`), and the `Set` allowlist.

**Keeping the door open.** The IR and Go are the same for (b) and (c): `TypeRef.total`,
`TypeRef.seq`, the same values and the same spellings. Switching the Scala surface between them
later changes no catalog, key or fingerprint. Only (a)'s partial catalog differs, and even it is
`TypeRef.total{K, Option[V]}`. So:

- **(c) first, native `Map` later** is possible. A later phase can admit Scala `Map[K, V]` as
  surface sugar for `Total[K, Option[V]]`, with `get`, `updated` and `removed` lowering through the
  `Option` values. **Open question:** the spelling of `Total[K, Option[V]]` would then be
  `{k1:None,k2:Some-x}`, while a native partial map reads more naturally as `{k2:x}`. Decide before
  Phase 2 freezes §2.3 whether a total map's `None` entries are omitted from the spelling.
- **(b) first, then `Total`/`ListUpTo`** cannot be done by aliases. A transparent
  `type ListUpTo[A, N] = List[A]` is just `List[A]` to implicit search, so `Finite[ListUpTo[A, 3]]`
  and `Finite[ListUpTo[A, 4]]` would be the same given. Only an opaque type carries a separate
  catalog, as `UpTo` does (`Domain.scala:88`). Moving from (b) to (c) therefore touches every write
  (`ListUpTo(...)`) and every `removed`, `empty` or tuple `forall`. The edit is mechanical, but it is
  a source migration.

#### 8.4.9 Recommendation

**Choose (c), the current mix, by default.** It is the only option in which:

- both samples keep their catalogs and stay under the ceilings;
- totality and bounds are checked at compile time, or at the write in Scala, rather than only at the
  Go row;
- the catalog is a function of the type alone, which keeps step tables, pins and totals coherent;
- the `UpTo`-over-`Int` decision of SEMANTICS §1 is followed.

Native `Set` stays, because nothing needs choosing for it.

Conditions under which another option is right:

- **(b)** if familiarity with plain `Map`/`List` matters more than compile-time checks, and every
  model has at most one bounded type per element type. Accept that out-of-domain writes are found by
  Go at the rule binding, not at the line.
- **(a)** only if partial maps are the natural state and `(|V|+1)^|K|` fits the ceilings. Or if the
  reader stops deriving tables over the whole catalog (`machine.go:529-568`) and derives them over
  reachable states, which would remove most of the size argument against partial catalogs.
- **(c) plus later native `Map` as sugar for `Total[K, Option[V]]`** if Quint-style partial maps are
  wanted later. This costs one spelling decision now (above).

**Open questions specific to this comparison:**

- the exact tree Scala 3 produces for an untupled `Map.forall((k, v) => …)`, needed by (a) and (b);
- whether `Finite.listUpTo` givens can be keyed per field without a macro;
- whether the step tables check results against the catalog, or table an out-of-catalog result
  silently, before Go sees it. This decides whether (b) catches an overflow in Scala at all.

### 8.5 `Bounded[List[A], N]` rather than `ListUpTo[A, N]`

A generic `Bounded[C, N]` would need a separate meaning for each `C`. `ListUpTo` follows the
`UpTo[N]` precedent: an opaque subtype that reads as its underlying type and is written through
`apply` (`Domain.scala:85-96`).

---

## 9. Phased implementation plan

Every phase leaves the gate (`make umpire-check-model`) green. Phases 1 and 2 are independent.

**Phase 1: input-reading guards** (no collection; fixes the rule ladder).

- Files to change:
  - `model/framework/dsl/Syntax.scala`: `Case.guard`, `Case.where`, top-level `where`;
  - `model/framework/dsl/Machine.scala`: `Rule.guard`, `overlap`, a guard adapter beside `effectOf`;
  - `model/framework/export/Declarations.scala`: `condition`; the heading capture for
    `Heading.When`/`And`;
  - `model/SEMANTICS.md` §6: headings table and core form.
- Tests:
  - `model/framework/dsl/Rules.test.scala`: two rules of one action whose input-reading guards
    overlap on one class are refused, naming the class; disjoint ones are accepted;
  - an export fixture in `model/framework/export/testdata/lifts/`, checked by
    `model/framework/export/Fixtures.test.scala`: `where((s, i) => g(s, i))` lifts to the same step
    function as the per-class ladder (compare the evaluated tables, not the trees);
  - `model/framework/dsl/Effects.test.scala` and `Facets.test.scala` for derivations and facets.

**Phase 2: IR and Go reader.**

- Files to change:
  - `proto/internal/temporal/server/api/umpire/v1/value.proto` and `expression.proto`, then
    `make proto` and the ScalaPB jar regeneration (`make gen-model-api`, which
    `model/temporal/check/MakeFeatures.test.scala:180` names);
  - `tools/umpire/interp/{eval.go,machine.go,limits.go,decisions.go}`, a new
    `tools/umpire/interp/collections.go`;
  - `tools/umpire/ir/validate.go`;
  - `tools/umpire/lint/kinds.go`.
- Tests, all with hand-written IR so they need no lifter:
  - `interp/collections_test.go`, the **oracle**: for small `K`, `V`, `A`, `N`, `Members(Set[K])`
    mapped to Boolean assignments equals `Members` of the record encoding, in order. The same for
    `Total` against its record, and for `ListUpTo` against channel contents. `Size` equals
    `len(Members)` wherever listing is within the ceiling. Every op equals its desugared chain on
    every catalog value.
  - `ir/admission_test.go`: non-canonical literals, collection inputs, a set of a type holding
    itself.
  - `ir/bounds_test.go`: `LimitError` for `Set` over 17 keys, before listing.
  - `interp/rowkeys_test.go`: brace keys with hyphenated members.

**Phase 3: DSL and lifter.**

- Files to change:
  - `model/framework/dsl/Domain.scala`: givens, `Total`, `ListUpTo`, `catalog`, `sumOf`, the
    ceiling, `Finite.spell`;
  - `model/framework/dsl/StepTables.scala` (spelling);
  - `model/framework/export/Types.scala`: `typeRef` before its opaque branch at line 46, as
    `upToType` is handled at line 45; `optionType` names at line 146-169;
  - `model/framework/export/Expressions.scala`: `lift` cases beside the channel operations at
    line 605-613; `valueKey` at line 198-207 spelling §2.3 exactly as Go does;
  - `model/framework/export/Capabilities.scala`: `size`;
  - `model/framework/inspect/LiftableSubset.scala`: `permitted` for `Set`, `ArrayOps` over enum
    `values`, `ListUpTo`'s list operations;
  - `model/SEMANTICS.md`: §1 types, §5 "What a step function may contain", §20 liftable-subset rows
    with fixtures, §21 core list.
- Tests:
  - `model/framework/export/testdata/lifts/Collections.scala` with a `Fixtures.test.scala` case per
    new liftable-subset row, as §20 requires;
  - refusal controls in `model/framework/lint/LiftableSubset.test.scala` for `+` on a set,
    `toList`, `head`, `map(...).sum`;
  - a Scala test that `Finite[Set[K]].values.map(spell)` equals the Go catalog keys for the same
    type, the cross-language spelling pin.

**Phase 4: samples.**

- Files to change:
  - rewrite `model/framework/samples/Bank.scala` and `TwoPhaseCommit.scala` as in §7;
  - `model/README.md:640-646` (the samples paragraph).
- Tests:
  - the samples check (`model/framework/check/Samples.scala`) already demands identical answers:
    verify → Verified, find → Found, `refutes*` → Counterexample, progress Verified (lines 37-64);
  - add a table-isomorphism test in `model/framework/check/Samples.test.scala`. It keeps the old
    encodings as test fixtures, maps each old state to the new one through an explicit function,
    and requires equal step tables. The rename map cannot express this (§6).

**Phase 5: Quint.**

- Files to change: `tools/umpire/export/{quint.go,itf.go}`, `tools/umpire/export/README.md`.
- Tests: `quint_test.go` and `itf_test.go` cases. The agreement gate
  (`make umpire-check-backends`) covers a feature only once a feature uses collections.

**Phase 6 (optional, on demand).** Admit collections in monitor states and channel messages
(`validate.go:906-915`) and as action inputs. Inputs need class-key spelling in
`machine.go:446-450`, lowering and realization review. Consider Scala `Map` once §10's untupling
question is answered.

---

## 10. Open questions

1. **`where` overloads.** Can Scala 3 resolve `where[S, O, F](using Firing[…])(S => Boolean)`
   against a two-parameter overload by lambda arity when the leading parameter list is a `using`
   clause? If not, alternatives are a match-type parameter keyed on the `Firing`'s input tuple, or a
   distinct word. Settle with a compile fixture before Phase 1.
2. **CanEqual.** Scala 3's `CanEqual` companion has givens for sets and maps, and `Finite.given`
   adds `CanEqual[T, T]`. I expect the lexical import to win over the implicit-scope given without
   ambiguity, but have not compiled it.
3. **Opaque types in the lifter.** `Types.scala:43-55` reads an opaque type's range from
   `opaqueRanges` before dealiasing. `Total` and `ListUpTo` must be recognized before that branch,
   and the list-field refusal at `Types.scala:112` must not see through `ListUpTo`. Verify how
   `dealias` treats them outside their defining scope.
4. **Type inference** of `UpTo(n)` through `Total.fill`/`Total.by` (§7.2). Explicit type arguments
   are the fallback.
5. **Extension `apply`.** `t(k)` on an opaque `Total` should resolve to the extension `apply`.
   Verify, and fall back to `t.get(k)` if not.
6. **Quint builtins.** The exact names and signatures in the pinned Quint version (`mapBy`, `set`,
   `exclude`, `fold`, `slice`).
7. **IR version.** Keep 0 for an additive change, or bump it so an old reader names the cause rather
   than "a type reference names no type" (`validate.go:459-460`).
8. **`SetupKey`** (`canonical.go:196-201`) cuts the start state at the first `-` to name "the start
   state's first field". A first field that is a collection with field-bearing members would be cut
   inside the braces. Is `SetupKey` still read anywhere it matters?
9. **Naming.** `Total[K, V]` against `PerKey[K, V]` or Quint's `K -> V`. `ListUpTo[A, N]` follows
   `UpTo[N]`.
10. **Collections as inputs** (for example "prepare this subset"). Deferred to Phase 6. Their class
    keys would carry braces into class keys and Case names.
11. **Spelling of `Total[K, Option[V]]`.** Should `None` entries be omitted (`{k2:x}` rather than
    `{k1:None,k2:Some-x}`), so that native Scala `Map` can later be admitted as sugar with no
    fingerprint churn (§8.4.8)? This must be decided before Phase 2 freezes §2.3.
