# Builtins: one table of operators, a generic call in the IR, semantics in the back ends

## Execution status (2026-10-21 backlog audit)

Deferred by the owner. The Rust decision has been made and the active workspace is `model/`; the former Scala tree is frozen under `model/archived/` and is not expected to build (AGENTS.md and MILESTONES.md). The earlier “until the Rust verdict” note is historical, not an automatic release of this spec. Existing task states and old SHIP reviews describe historical checkpoints, not current implementation readiness.

Resume only after an explicit owner decision and a fresh scope/dependency review against the Rust architecture. Scala paths and commands below describe the archived layout; do not run them as current gates or mechanically port the plan. Preserve unfinished requirements and completion evidence; successor Rust work does not discharge this spec without an explicit requirement mapping. Taskless proposals remain deferred captures, not executable plans.


## Conversation Evidence

> user (2026-10-17): "are there any sacrifices we can make in its support to reduce that number dramatically? or reduce it without sacrifices? or find a smarter way that might impact the DSL in minor ways? or find a different approach altogether?"
> user: "how does Quint deal with the explosion of these?" (of `Inbox.send`, `isFull`, `Some`/`None`, `UpTo`, `min`/`max`, `+:`, `List.contains`, `Hole.reached` and `Step`, each recognised, lowered and refused in Scala, evaluated in Go and translated once more for Quint)
> user: "I'm okay with that simplification as long as we keep a tight list of builtins (can we also call them that?)"

## Goal & Context

Every operator a Model may use on a value is written three times today: the reify plugin's body
lifter recognises the Scala call shape, lowers it to a dedicated IR form and refuses what it does not
know; the Go interpreter evaluates that form; the Quint exporter translates it again. The IR has a
dedicated kind or enum value for each (`Inbox` with three ops, `OP_CONTAINS`, `OP_CONCAT`, a `min`
lowered to an `If`, an `Option` synthesised as an enum type), and fn-173.4 is about to add a
`Collection` node with a dozen more ops for native `Set` and `Map` state. The lifter's size tracks
that vocabulary, not the Scala expression language: its generic core (literals, references, field
reads, `if`, `match`, `let`, lambda, construction, `copy`) is about 350 of its 4,000 lines.

Quint does not have this growth because it owns its syntax: every operator is one row in a single
builtin table with a name and a type signature; the parser, resolver and type checker treat all
builtins alike through that table, and only the two back ends (the simulator and the Apalache
translation) implement meaning. This spec gives Umpire the same shape. The operators are called
**builtins**. There is one table of them, deliberately short; the lifter emits a generic builtin
call for any of them; the Go interpreter and the Quint exporter are the two places that know what a
builtin does; a conformance test holds the three to the one list.

Owner's standing decisions carried: byte identity is not required; the Go checks' answers are the
oracle; Temporal sources stay excluded; the framework samples are the proving ground; this is
phase-2 IR work (owner, 2026-10-17, phasing), batched with fn-173.4 and the plugin diet.

## Architecture & Data Models

**The builtin table.** One checked-in data file, `model/builtins.json`, is the list. Each row:

```json
{ "name": "inbox.send", "scala": "framework.dsl.Inbox.send", "arity": 2,
  "params": ["channel", "message"], "result": "channel", "doc": "the contents with the message added" }
```

`name` is the builtin's IR name, `<family>.<op>`, families being `int`, `bool`, `list`, `set`,
`map`, `option`, `inbox`, `hole`. `scala` is the fully qualified Scala method the lifter recognises
(an owner and a method name; overloads resolve by `arity`). `params`/`result` are the IR type words
the validator checks (`int`, `bool`, `list`, `set`, `map`, `option`, `channel`, `any`, `same`).
The table is the contract: a builtin not in it does not exist. Adding a row is a spec change to this
spec (an R-ID), never an implementation convenience.

**The initial table, complete.** Everything the lifter special-cases today that is a method on a
value, plus fn-173.4's collection operations, which land as builtins instead of a `Collection` node:

| Family | Builtins |
| --- | --- |
| `int` | `min`, `max` |
| `list` | `contains`, `prepend`, `concat`, `size`, `isEmpty` |
| `set` | `contains`, `incl`, `excl`, `size`, `isEmpty` |
| `map` | `contains`, `get`, `getOrElse`, `updated`, `removed`, `size`, `isEmpty` |
| `option` | `isDefined`, `isEmpty`, `getOrElse` |
| `inbox` | `send`, `isEmpty`, `isFull` |

`hole.reached` keeps the IR's `hole` kind (it names a declared hole by id, not an operator).
`Some`/`None` stay constructions and literals of the synthesised `Option` enum type; `UpTo` is an
erasing alias under fn-190 R9 and has no operator; `Step` is a record construction. Arithmetic,
comparison and Boolean connectives stay `Binary`/`Unary`, and the existing `OP_CONTAINS` and
`OP_CONCAT` are retired in favour of `list.contains` and `list.concat` so that one mechanism covers
every value operator. `because` keeps its form (it edits a step's explanation, not a value).

**The IR.** `Expr.kind` gains one message:

```proto
// A call of a builtin operator by its name in model/builtins.json.
message BuiltinCall {
  string name = 1;
  repeated Expr args = 2;
}
```

`Call` stays what it is, a call of a Model function, so the validator's "function exists" check is
untouched. The `Inbox` message and the `OP_CONTAINS`/`OP_CONCAT` enum values are removed; the
`Collection` node proposed in `model/design/reachable-exploration.md` §6.1 is not added.

**The lifter.** `LiftableSubset.permitted` becomes a lookup in the table by the call's resolved
symbol and arity: a match lifts to `BuiltinCall(name, args)` with each argument lifted against the
row's parameter words; a library call with no row is refused at compile time with today's text,
"outside the liftable subset: <call>", so the compile-time refusal the owner relies on is kept. The
`lift` arms for `Inbox` operations, `min`/`max`, `+:`, `toList`, `contains`, `++` on lists and the
`inboxChannel` resolution go; the channel a `BuiltinCall` of the `inbox` family operates on is its
first argument's state field, which the validator resolves through the field's `channel` type.
`Types.typeRef` keeps describing `Option`, `List`, `Inbox`, `Set` and `Map` as types.

**The back ends.** The Go interpreter evaluates `BuiltinCall` through one map from name to
implementation; the IR validator checks every name against the table and every call's arity and
parameter words; the IR lint reports a name the table does not have; the Quint exporter maps each
name to its Quint operator through one map. A conformance test in Go reads `model/builtins.json`
and fails if any name lacks an interpreter case or an exporter case; a Scala test fails if any row's
`scala` method does not resolve against the DSL.

## API Contracts

Author-facing: nothing changes. A Model writes `inbox.send(m)`, `xs.contains(x)`, `a.min(b)`,
`seen.incl(x)`, `m.getOrElse(k, 0)` as plain Scala, and the strict flags type-check them as today.

Compiler refusals, exact texts are the contract:

- a library call outside the table: today's `outside the liftable subset: <call>` at the call.
- a table row whose `scala` method does not resolve against the DSL or the standard library the
  Models compile against: the Scala conformance test fails naming the row; no author surface.

IR validator refusals, exact texts:

- `builtin <name> is not in model/builtins.json`
- `builtin <name> takes <n> arguments, <m> given`
- `builtin <name>: argument <i> is <found>, <expected> expected`

## Edge Cases & Constraints

- Overloads: a `scala` method with several arities has one row per arity; the lifter picks by the
  applied argument count after the normalization pass (fn-189.10) has removed default getters.
- Receiver as first argument: a method call `recv.op(args)` lifts as `BuiltinCall(name, recv :: args)`;
  an extension method the same way; the table's `params` list includes the receiver.
- `inbox.send` on a channel the state does not hold is a validator error on the first argument's
  type, not a lifter special case.
- `list.contains` on a list of steps or facts is the same builtin; the validator's `any` element word
  admits it.
- Collection builtins over unbounded collections are out of scope here exactly as in fn-173.4: the
  schema bounds them, the builtins do not.
- Backward compatibility: none owed. Retained oracles under `export/testdata` are regenerated once
  by fn-190.10 together with the rest of the phase-2 batch; the equivalence claim is on Query answers
  through the Go checks and on the reachable graph otherwise.
- Position of a `BuiltinCall`: none on the expression (fn-189 R5); refusals report at the tree.

## Acceptance Criteria

- **R1:** `model/builtins.json` exists with exactly the rows of the table in Architecture, each with
  `name`, `scala`, `arity`, `params`, `result` and `doc`; a Go conformance test and a Scala
  conformance test both read it. Errors: a row with an unknown type word, a duplicate `name`, or a
  `scala` method that does not resolve fails the respective conformance test naming the row.
- **R2:** `expression.proto` has `BuiltinCall` in `Expr.kind`; the `Inbox` message, `OP_CONTAINS` and
  `OP_CONCAT` are removed; no `Collection` node exists; the generated Scala and Go classes rebuild.
  Errors: no error surface beyond proto compilation.
- **R3:** The body lifter lifts any call whose resolved symbol and arity match a row to
  `BuiltinCall(name, args)` and has no other code that recognises a builtin; `LiftableSubset.permitted`
  is the table lookup. Errors: a library call with no row is refused with `outside the liftable
  subset: <call>` at the call, as today.
- **R4:** *(2026-10-18, one-language series: recut by fn-201.6: the Go interpreter row is deleted; R4 becomes emitter coverage, every builtin of the table has a translation in each Scala emitter (TLA in fn-206/fn-207; Quint in deferred fn-205) and the checker executes it as Scala)* The Go interpreter evaluates every builtin of the table; the validator and lint refuse an
  unknown name, a wrong arity and a wrong argument type with the exact texts in API Contracts; the
  Quint exporter maps every builtin. Errors: the conformance test fails for any name lacking an
  interpreter or exporter case.
- **R5:** *(2026-10-18, one-language series: recut: the answers are the Scala checker's, not the Go checks'; the retained oracles are the native sample suites)* fn-173.4's native `Set` and `Map` operations are the `set` and `map` families of the table
  and nothing else; the six samples (Bank's sparse maps, TwoPhaseCommit's and Voting's sets) lift,
  validate, interpret and export through builtins, and their Query answers through the Go checks are
  unchanged from the retained oracles. Errors: a changed answer is a finding, not a regeneration.
- **R6:** The lifter's line count after R3 is recorded in fn-189's counts receipt beside the fn-189
  R10 figure, with the command. No error surface.

## Boundaries

- Arithmetic, comparison and Boolean connectives stay structural `Binary`/`Unary` kinds; they are
  not builtins.
- `Call` of a Model function, `construct`, `copy`, `field`, `match`, `let`, `lambda`, `hole` and the
  `Option` enum synthesis are unchanged.
- No new builtin beyond the table. A Model that needs one proposes a row in this spec.
- No change to how declarations (actions, transitions, claims, scenarios) reach the IR; this spec is
  about expressions only.
- Temporal sources stay disabled; the samples and the framework fixtures are the proving ground.
- Not a Quint front end: the Scala DSL remains the authoring language and the type checker is Scala's.

## Decision Context

Three shapes were weighed. Overloading the existing `Call` with builtin names was rejected because
the validator's "every called function exists in the Model" check is worth keeping exact. The
`Collection` node of reachable-exploration §6.1 was rejected because it is one more per-family IR
form with its own op enum, which is the pattern this spec ends. Moving `Binary`/`Unary` into the
table too was rejected as schema churn with no simplification: those kinds have no recognition cost
in the lifter beyond one map of operator names.

The owner's condition is a tight list. The table is therefore closed, checked in as data, read by
both conformance tests, and extended only through this spec's R-IDs. The initial table is the set
the lifter already special-cases plus the collection operations fn-173.4 already enumerates; it adds
no operator a sample or a Temporal Model does not use today.

Sequencing: phase 2 of the owner's phasing (2026-10-17), batched with fn-173.4 (whose `Set`/`Map`
operations land here as builtins), after fn-189.10 (the normalization pass the arity match relies on)
and alongside fn-190.9/.10. Nothing in phase 1 depends on it.

2026-10-18, one-language series: the Go interpreter and the Quint exporter no longer hold the semantics; fn-201.6 recuts this spec so the builtins table, the checker's execution and each Scala emitter's translation are held to the list by one Scala conformance test. The standing decision "the Go checks' answers are the oracle" (Goal & Context, Edge Cases) is withdrawn: the native checker is the oracle (fn-196). Whether `model/builtins.json` stays checked-in data or becomes a Scala table is decided in fn-201.6.

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.

## Requirement coverage

| Req | Task(s) | Gap justification |
| --- | --- | --- |
| R1 | — | not planned yet |
| R2 | — | not planned yet |
| R3 | — | not planned yet |
| R4 | — | not planned yet |
| R5 | — | not planned yet; lands with fn-173.4 |
| R6 | — | not planned yet |
