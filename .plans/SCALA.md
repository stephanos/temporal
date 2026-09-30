# Scala 3, CEL and the Umpire IR

Notes distilled from a ChatGPT conversation titled "Compare Specification Languages" (shared link
`chatgpt.com/share/6abd1e86-…`), scoped to its Scala 3, CEL and model-IR parts, and checked against
what the `temporal/.plans/archive/cmp` comparison measured on 2026-09-30. Where the two disagree, both
positions are stated. Library facts were re-checked with the GitHub API on 2026-09-30; claims about
tool features (P, Veil, Stainless, Quint) are the conversation's and were not re-verified.

## The core idea

> Scala is syntax. The IR is the specification. Go interprets the specification.

Scala 3 becomes a typed authoring front end that runs only at build time. It emits a semantic model
IR. Go owns everything else: the explorer, the conformance checker, the test drivers, the
deterministic runtime, and the real Temporal integration. The JVM never sits in the model-checking
loop, so a Go `fork()` checkpoint contains the runtime, Temporal, the explorer, the IR interpreter
and the abstract model state in one process, with no Scala process to snapshot and no cross-language
RPC during exploration.

```text
        Temporal .proto  (structure + buf.validate / CEL constraints)
                │
          Scala 3 front end  (types, metadata, transitions, properties, assumptions)
                │  build time only
                ▼
            Umpire IR  (protobuf, canonical, content-addressed)
                │
     ┌──────────┼───────────────┬──────────────┐
     ▼          ▼               ▼              ▼
 Go explorer  Go checker   Quint → Apalache/TLC   docs
     │                     (later: P, Veil/Lean)
 deterministic Go runtime
     │
 real Temporal → observations → IR conformance
```

Strictly, the Temporal specification is written in none of the languages. It is the IR. Scala is
its ergonomic authoring language, Go executes it, and other tools reason about it. This agrees with
the architecture the `cmp/` work arrived at independently.

## The IR

What the IR must be:

- **A semantic transition-system IR, not serialized Scala.** Sections: types, state, inputs,
  events, transitions, properties, observations, assumptions. Transitions are declarative guarded
  commands: a guard, effects on state fields, and a result.
- **Protobuf**, fitting Temporal's ecosystem, with expressions as a `oneof` message (constant, field
  reference, unary, binary, quantifier, if-then-else). Front ends can emit ProtoJSON, which Go
  validates against the schema.
- **Deterministic and content-addressed.** Canonicalize, hash with SHA-256, and name every
  counterexample and replay recipe by the model hash (`model: 8fa31c`, runtime, seed, schedule,
  faults).
- **Defined by documented semantics independent of Scala and Go.** Write the evaluation rules down
  (`Eval(Add(a,b),S) = Eval(a,S)+Eval(b,S)`, `Apply(Assign(x,e),S) = S[x ↦ Eval(e,S)]`). The Scala
  front end is then only a way to construct valid IR, and the Go interpreter only an efficient
  evaluator of it. Never let "whatever the front end emits" become the definition.
- **Split into verification profiles** rather than the least common denominator: finite/executable,
  symbolic, temporal, proof. Each property reports which backends can check it, for example
  "terminal is final" by the Go explorer, TLC, Apalache and Veil, and "eventual delivery" by TLC and
  a P liveness monitor but not by the Go invariant checker.

Added by the `cmp/` work:

- **Step bodies are ordered clauses.** Each clause is pattern, guard and body. The first match wins,
  bodies are `disabled`, `stay`, `not_found` or `moves(phase, facts, updates)`, and a state that no
  clause covers is *unspecified*, a hole rather than a rejection. The Elixir sample exports exactly
  this shape.
- **The interpreter is the semantics.** Generated Go, if ever wanted for speed or for a typed Go
  library, is derived from the IR and pinned against the interpreter on every row.
- **Keep derived tables out of the IR.** The Elixir sample embedded full tables and exported
  1.5 to 2.1 MB per model; compiling those literals was what made its model edits take five
  seconds. Ship guarded commands and let Go derive tables.
- **Expressions-as-data is what enables SMT.** Refinement, invariant preservation and bounded traces
  become solver queries only because guards and updates are trees, not closures.

## How Scala produces the IR

The conversation distinguished two ways, and the `cmp/` work found a third.

**A. Scala builds an expression tree (the conversation's preference).** Operators on a typed
expression GADT construct IR instead of evaluating:

```scala
sealed trait Expr[A]
case class Const[A](value: A) extends Expr[A]
case class Eq[A](l: Expr[A], r: Expr[A]) extends Expr[Boolean]
case class And(l: Expr[Boolean], r: Expr[Boolean]) extends Expr[Boolean]
case class Add(l: Expr[BigInt], r: Expr[BigInt]) extends Expr[BigInt]

extension [A](x: Expr[A]) def ===(y: Expr[A]): Expr[Boolean] = Eq(x, y)
extension (x: Expr[Boolean]) def &&(y: Expr[Boolean]): Expr[Boolean] = And(x, y)

transition(FailRetryable) {
  when((status === Started) && event.retryable && (attempt < maxAttempts))
  set(status := Scheduled, attempt := attempt + 1, dispatchability := BackoffPending)
}
```

`attempt + status` cannot be constructed and `when(attempt)` does not compile, so the IR is
type-safe by construction. No macros are needed. The cost is that authors write operators such as
`===` and `:=` and cannot use native `match`.

**B. Arbitrary Scala functions (rejected by the conversation).** Exporting the semantics of
arbitrary Scala would need bytecode, WASM or generated Go, and the clean semantic boundary
disappears.

**C. Macro lifting of a restricted subset (from the `cmp/` work).** An `inline` step body is handed
to a macro as a *typed* tree; the macro walks `Match`, `CaseDef` and constructor calls and emits IR
clauses, rejecting anything outside the subset. Authors get native pattern matching over tuples,
and names and types are already resolved before the macro sees them. Two caveats: the body must be
passed inline, since a macro sees only a reference to a separately defined `def` (reading its body
needs `-Yretain-trees` or TASTy), and a Scala macro cannot evaluate code compiled in the same run,
so tables cannot be built at compile time within one module.

A practical combination is C for step functions and A for guards and property predicates.

## Scala 3 features, and who uses them

The conversation's rule: make nonsensical specifications fail compilation where practical, make
semantic omissions fail model validation, and leave behavioural correctness to verification.

Model authors should need about six concepts:

1. Enums, case classes and pattern matching, for states and events.
2. Opaque domain types (`WorkflowId`, `RunId`, `UpdateId`, `Attempt`).
3. The extension-method DSL (`when`, `transition`, `invariant`).
4. Exhaustive matching, so the compiler lists every match a new case breaks.
5. Basic generics.
6. Ordinary immutable expressions.

The framework uses the rest:

| Feature | Use in Umpire |
|---|---|
| ADTs over option-heavy records | `Completed` has no retry deadline; illegal combinations cannot be written |
| Opaque types | IDs that are all strings underneath cannot be mixed |
| Refined values (Iron or smart constructors) | `Attempt >= 1`, `Probability ∈ [0,1]`; many come from `buf.validate` |
| Phantom type parameters | `Model[Product]` versus `Model[System]`, so a product transition cannot name a persistence fault |
| GADTs | the typed expression language above |
| Extension methods | make the expression tree read like Scala |
| `given` / `using` | a model context (types, metadata, descriptors, source location) and capabilities, e.g. `fault(...)` only compiles where a `FaultModel` is in scope |
| Context functions | scoped DSL blocks: `productModel { ... }` versus `systemModel { ... }` |
| Type classes | attach observation, identity and IR encoding to Proto-generated classes you do not own |
| `derives` + `Mirror` | derive IR schema, serializer, state hash, pretty printer, diff and generators from one definition |
| Match types | compute the IR type of a Scala type at compile time; hidden in the library |
| Intersection types | backend capabilities: `Property[Executable & SMTCompatible]`, so `verifyWith[Apalache](p)` fails to compile for unsupported constructs |
| Singleton types | names carried in types; internal only |
| Dependent method types | `read(state, field): field.Value` without casts |
| `inline` | move checks such as unknown Proto field or duplicate transition into compilation |
| Quotes and reflection | record source position, expression text and referenced fields into the IR for diagnostics |
| `CanEqual` with strict equality | `workflowId == activityId` does not compile |

Division of labour the conversation proposed, which the `cmp/` work would keep:

- **Scala's compiler:** wrong types, missing ADT cases, mixed IDs, illegal DSL context, unsupported
  backend capability, invalid expression construction.
- **Umpire model validation:** states with no outgoing behaviour, unreachable or overlapping
  transitions, events with no behaviour, properties with no applicable states, unobserved fields,
  new Proto enum values with no behaviour.
- **Model checkers:** an execution that violates an invariant.
- **SMT, Veil or Lean:** that every transition preserves an invariant for all states.
- **The deterministic Go runtime:** that real Temporal can execute a schedule or fault the model
  forbids.

What the `cmp/` compile trial showed about this toolbox: the model files compiled unchanged once the
framework built, and model edits compiled in about a second on a warm server (Scala 3.9.0,
scala-cli 1.17.1). But the framework needed 41 fixes first, 40 of them from one type-level
derivation, and the design leaned on the still-experimental `TupledFunction`. The sophistication
belongs in the framework, and whoever maintains it must be fluent in given resolution and match-type
reduction.

## Libraries worth leveraging

All maintained as of 2026-09-30.

| Library | Use | Note |
|---|---|---|
| **ZIO Blocks Schema** (`zio/zio-blocks`) | reified, inspectable model structure with strongly typed metadata modifiers; derive IR, docs, hashes, serializers, diffs | closest fit to "a model Umpire can inspect"; adopt the schema module, not ZIO effects |
| **Iron** (`Iltotore/iron`) | refined primitives checked at compile time where literal | use selectively |
| **Scala 3 `Mirror`**, then **Magnolia** | generic derivation | start native; add Magnolia only if derivation gets painful |
| **Cats `ValidatedNec`** | accumulate every definition error in one report | "Model invalid: 7 errors" with file and line per error |
| **ScalaCheck Commands** | stateful tests of the front end and interpreter: Scala → IR → Go agreement, and mutation testing of properties | tests Umpire itself, not Temporal |
| **Monocle / Quicklens** | immutable updates | if the front end builds IR, a small typed `Field` abstraction that compiles to `FieldRef` may suit better |
| **Stainless** (`epfl-lara/stainless`) | SMT-backed proofs over a pure Scala subset | optional; see below |

Suggested first cut: Scala 3 with native ADTs, opaque types and `Mirror`, ZIO Blocks Schema, and
Cats `ValidatedNec`. Add Iron where invalid primitives are a real problem and Stainless only for
properties you want proved. Do not assemble all of them at once.

## Stainless

A formal verification tool for Scala from EPFL: write ordinary Scala with `require`/`ensuring`
contracts and lemmas, and Stainless proves them for all inputs through Inox and SMT solvers,
returning valid, a counterexample, or unknown.

It can prove, for a pure model:

- every transition preserves an invariant (`attempt >= 1`, terminal consistency);
- terminal states stay terminal for every state and event;
- invariants over arbitrary-length traces, by structural induction;
- relations between fields, determinism of chosen events, and properties of Umpire's own
  transformations such as normalization being idempotent;
- termination of recursive functions.

It cannot:

- find requirements you forgot to state;
- explore Go goroutine schedules or distributed interleavings (no DPOR, symmetry reduction or
  state storage);
- show the real Go implementation conforms;
- prove temporal liveness in general;
- verify arbitrary Scala. It covers a verification-friendly subset, and quantifier-heavy models
  leave its sweet spot.

The IR design raises a subtlety: if Scala only builds an expression tree, Stainless proves things
about the tree-building program, not the model. Either give the expression types an `eval` so
Stainless reasons about that interpreter, or verify the IR separately through Lean, Veil or SMT.
The conversation preferred the second, which makes Stainless optional.

## Lean versus Scala

The gap is narrower than it first looks. Scala has ADTs, GADTs and dependent method types, and with
Stainless it can prove invariant preservation and arbitrary-length trace properties. So "Scala can
only test and Lean can prove" is wrong. Lean's real advantages:

1. **The trust base.** Lean's kernel checks proof terms, so an agent can generate a complicated
   proof and a small checker validates it. Stainless trusts its pipeline and solvers.
2. **Verifying Umpire itself.** Claims that quantify over models, traces and algorithms, such as
   "the trace checker accepts exactly the traces the model permits" or "this state-space reduction
   preserves every safety violation", are Lean's home ground.
3. **Non-executable properties.** `Prop` states fairness and refinement claims naturally without
   pretending they are Boolean functions.
4. **Proof-carrying structures.** A model can package its invariant together with the proof that
   each step preserves it.

Added by the `cmp/` work: Lean can also lift the logic of any definition by name after elaboration,
supports genuinely custom syntax with typed elaboration, and can show tables and counterexamples in
the editor. It still loses on loop time and staffing, and none of this helps with forgotten
requirements, over-permissive models, or observation mapping.

A way to use Lean without authoring in it: generate Lean from the IR and prove things about the IR's
semantics, or about the Go interpreter against a Lean semantics, with differential testing bridging
the two.

## CEL and buf.validate

Facts: CEL has a Java implementation (`google/cel-java`, published as `dev.cel:cel`) that parses,
type-checks and evaluates against Protobuf values, and Protovalidate has a Java implementation
(`bufbuild/protovalidate-java`) that evaluates `buf.validate` annotations including custom CEL
rules. Buf's lint also checks that CEL rules compile and match their field types.

Three kinds of constraint, from the conversation, and where each should live:

| Kind | Example | Source |
|---|---|---|
| Data validity | `attempt >= 1`, `timeout > 0`, `workflow_id != ""` | import from `.proto` and `buf.validate` |
| State validity | `Completed ⇒ !retryPending`, `BackingOff ⇒ retryDeadline exists` | Umpire model |
| Behavioural properties | terminal is final, a retry increments the attempt, a stale task cannot complete the current attempt | Umpire transitions and properties |

Insights that stand regardless of the expression language:

- **Import data constraints instead of restating them.** A JVM front end can load Proto descriptors,
  read fields, enums, oneofs, presence and `buf.validate` options, and generate Umpire's types, then
  add model metadata on top (`.observable`, `.identity`, `.internal`). The model extends the real
  Temporal data model rather than rebuilding it.
- **Check that transitions preserve imported constraints.** If `attempt >= 1` comes from the proto
  and a transition assigns `attempt - 1`, translate both to SMT and report that the constraint
  cannot be established. Define an explicitly SMT-verifiable subset rather than promise arbitrary
  CEL.
- **Protovalidate is an input, not the specification system.** It answers whether a message is
  valid; Umpire answers which behaviours are permitted from a valid state.

The conversation went further and proposed CEL as the IR's whole expression language, evaluated by
`cel-java` in the front end and `cel-go` in the checker, with CEL's conformance suite keeping them
aligned. That is a real option, but it was declined in the `cmp/` discussion in favour of the IR's
own small expression tree (lit, var, field, comparisons, boolean operators, `in`, named pure
helpers), which also carries patterns and step bodies that CEL does not express. Importing
`buf.validate` rules does not require choosing CEL for the IR: the front end can parse imported CEL
rules and lower them into the IR's own expressions.

## Scala and the other tools

Scala does not replace P, Quint, Veil or Loom, and should not reimplement them. The conversation
recommended treating them as backends of the IR:

- **Quint first.** It already has a typed transition-system language, a JSON IR, a simulator, and
  both Apalache (symbolic, SMT) and TLC (explicit, temporal properties). Engineers never write
  Quint; `umpire verify --backend apalache|tlc` generates it. Apalache is written in Scala, so the
  JVM is familiar ground, though the Go-owned checker means the front end never needs to call it
  in-process.
- **P next**, if the communicating-machine system model becomes central. Its split of
  specification, model and test scenario matches Umpire's, and PObserve feeds a running system's
  events into the same monitors used for checking, which is worth studying before building
  conformance from scratch.
- **Veil** if machine-checked unbounded proofs, or proofs about Umpire's own reductions such as
  DPOR soundness, become central.
- **Loom** for algorithms, not integration: happens-before tracking, dependency detection,
  preemption bounding and DPOR belong in the Go execution engine.

The main danger the conversation named: turning Umpire into a universal formal-methods framework.
Umpire should own the typed semantic model, the common IR, Temporal observations, Go conformance and
deterministic Go execution, and leave solvers, theorem provers, symbolic and temporal checkers to
backends.

## Where this leaves Scala

In the `cmp/` ranking for this architecture, Scala 3 is third, behind Elixir and an own DSL parsed
in Go. It is strongest on lifting guards into IR from typed trees and on compile-time expressiveness,
and weakest on framework fragility, experimental features and the need for a specialist maintainer.
The conversation's architecture and library stack make the Scala option concrete enough to spike:
the matching slice from `UMPIRE_OUTSIDE_THE_BOX.md` section 12, authored with approach C for steps
and approach A for predicates, emitting the proto IR the Go checker reads.

Open questions:

- Approach A, C, or both, judged on how the slice reads and how much framework each needs.
- Whether ZIO Blocks Schema carries enough of the metadata and derivation work to justify the
  dependency.
- Whether Proto descriptor import and `buf.validate` lowering belong in the front end or in a Go
  pre-pass shared by every front end.
- Whether Quint export or a direct Go SMT backend is the better first symbolic backend.
