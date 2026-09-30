# Scala 3 sample

Two Models, `NexusCaller.scala` and `StandaloneActivity.scala`, written against the framework
surface in `Umpire.scala`, with the shared worker entity in `Worker.scala` and the pins in
`Pins.scala`. Nothing here has been compiled; the files are what an author would write, with the
imports and signatures real code would need.

## The DSL mechanism

Scala 3 is the mainstream language with the deepest kit for an embedded DSL, and the sample uses
four layers of it, each where it pays off:

1. **Plain language for the semantics.** Domains are `enum`s (`case handlerError(retryable:
   Boolean)` is one constructor and two classes), states are `case class`es, step functions are
   `def`s returning `List[Step[S, O, F]]` with an exhaustive `match`. There is no row language to
   learn; a Scala reader reads a step function as code.
2. **Context-function blocks for declarations.** `machine[S, O, F]("nexusProduct") { ... }` takes a
   `MachineScope ?=> Unit`, so `starts(...)`, `timers(...)`, `evidence { ... }` and `steps(...)` are
   bare calls inside the block that resolve against the scope in context. `set(...)` works the
   same way. There is no builder object to thread and no `this` to reference.
3. **`infix` methods for the one-line declarations.** A Property is `property("syncSucceeds")
   (nexusProtocol) when handlerReply(Reply.syncSuccess) holds: step => ...`, a Scenario is
   `scenario(...)(m) starts Phase.unscheduled actions (...)`, a Query is `query("retry") find
   retrySucceeds in retriedThenSucceeded limits four`. These read like the Lean lines and are
   ordinary method calls, so the IDE completes and navigates them.
4. **Type-level facts where the compiler can decide something.** Each `.input[Timeout]("...")`
   line extends the action's tuple type, so `schedule` is `Action[(Timeout, Timeout, Timeout)]`
   and `schedule ~> scheduleStep` requires a four-parameter step function through
   `TupledFunction`. `Finite` is derived with `derives Finite` through `Mirror`, and `Phased`
   finds the `phase` field by label. `Bounded[2]` is `0..2` with the bound in the type.
   Property and Scenario are indexed by the machine's state type, and `in` asks for a
   `Refines[S, P]` witness, so a Query pairs a Property with a Scenario of the same machine or of
   one that declares it refines the Property's machine; anything else does not type-check.

`Compose` is an `object` extending an abstract class, so the member handles (`nexusCaller.
operation`) are named fields a Scenario outside the object can reach; `operation(schedule)(unset,
expires, unset)` and `a || b` for a `sync` pair are extension methods on `MemberAction`. The
union type `Classed | Action[EmptyTuple]` lets a Scenario list a timer bare and a classed action
applied, without an implicit conversion.

## Where each check runs

| Check | When | How |
| --- | --- | --- |
| Step function names an undeclared action, wrong arity, wrong state type | compile | plain typing; `~>` needs a `TupledFunction` for the action's tuple |
| Non-exhaustive `match` over a domain or a fact | compile (warning, error under `-Werror`) | the compiler's space engine; `Boolean` literals and enum cases with fields are decomposed |
| A fact with no `evidence` line | compile | `evidence` takes a total `F => Evidence`; a missing case is a non-exhaustive match |
| State type has no `phase` field | compile | `Phased.derived` uses `compiletime.error` with the type's label |
| Attempt count literal out of bound, `limits` with actions below steps | compile | `inline` parameters and `compiletime.error` |
| State counts (`pinStates[ProtocolState](192)`) | compile | `Finite.sizeOf` is `transparent inline` and folds to a literal |
| Every find-Query is found, every verify-Query verifies | compile of the **test module**, and again as a test | `Query.pinned` macro runs the search in `Pins.scala`; the same searches run in munit |
| Table build, refinement (`rejected == None`), `stuck`, reachability | test | first access to the machine `lazy val`; munit asserts |
| Set well-formedness (a canary with a silent step, an exploratory set with Queries) | test | checked in `set(...)` when the object initialises |
| Action names bound twice, timers unbound, `unobservable` not a timer | test | same |

The macro restriction matters: **a macro cannot evaluate code from its own compilation run**, so
`Query.pinned` only works on a Query that lives in an already-compiled module. This sample keeps
the Models in one sbt project and the pins in a downstream test project, which is the ordinary
shape of a Scala build; but it means the search result cannot appear as an error inside
`NexusCaller.scala` itself. Lean's `#guard` has no such split.

## What an author's mistake looks like

**A step function bound to an action that does not exist** (typo `handlerRepy`):

```
-- [E006] Not Found Error: NexusCaller.scala:198:6
198 |      handlerRepy ~> handlerReplyStep,
    |      ^^^^^^^^^^^
    |      Not found: handlerRepy - did you mean handlerReply?
```

**A step function with the wrong arity for its action** (binding `scheduleStep` as a two-argument
function):

```
-- [E172] Type Error: NexusCaller.scala:396:16
396 |      schedule ~> scheduleStep,
    |                  ^
    |No given instance of type scala.util.TupledFunction[
    |  (ProtocolState, Timeout) => List[ProtocolStep],
    |  ProtocolState *: (Timeout, Timeout, Timeout) => List[ProtocolStep]] was found
```

**A non-exhaustive match** (dropping the `handlerError(false)` arm in `handlerReplyStep`):

```
-- [E029] Pattern Match Exhaustivity Warning: NexusCaller.scala:171:7
171 |  else reply match
    |       ^^^^^
    |       match may not be exhaustive.
    |
    |       It would fail on pattern case: Reply.handlerError(false)
    |
    | longer explanation available when compiling with `-explain`
```

With `-Werror` (the sample assumes it, and has no implicit conversions that would trip the
feature warning) that is an error, and the build stops.

**A failed Query** (say `retrySucceeds` fixed `attempts = 2`), reported by `Query.pinned` in the
test module:

```
-- Error: Pins.scala:37:28
37 |  val retry = Query.pinned(Nexus.retry)
   |                           ^^^^^^^^^^^
   |retry: `retrySucceeds` is not reached on `retriedThenSucceeded`
   |within four (10431 candidate traces searched)
```

and the same failure as a munit test:

```
==> X temporal.feature.pins.NexusCallerPins.every find-Query's Scenario reaches its Property
  retry: NotFound(10431)
```

**A wrong state-count pin**:

```
-- Error: Pins.scala:22:2
22 |  pinStates[Nexus.ProtocolState](8 * 3 * 2 * 2)
   |  ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
   |  state count is 192, pin says 96
```

## Toolchain and loop

- Scala 3.3 LTS or later; `scala-cli` for a single-directory experiment, `sbt` for the two-module
  layout the macro needs (`model` and `pins` with `pins dependsOn model`). munit for the tests.
- Warm incremental compile of a Model file: one to three seconds. The macro module adds the
  search time itself (the `four` and `six` limits are tens of thousands of traces, sub-second in
  JVM code) plus macro-expansion overhead. A cold `sbt` start with a JVM is fifteen to thirty
  seconds; the loop after that is edit, save, see the diagnostic in the IDE (Metals) without
  running anything.
- The exhaustivity, arity and `Finite` errors arrive in the editor as you type. The Query errors
  arrive when the test module compiles, which Metals does on save.

## What the language made easy, and what it made awkward

Easy:

- The step functions are the closest to the Lean originals of any sample that is not Lean: `if`
  guards, `match`, `copy`, list literals, and exhaustivity checked by the compiler. The comments
  carried over without rewording.
- `derives Finite, CanEqual` on a state and `derives Finite` on an enum is all the plumbing a
  new type needs; the table order falls out of declaration order.
- `infix` and context functions give declarations that read as the Lean ones do, with no
  quoting, and everything is a value the IDE understands: rename, find-usages and go-to-definition
  work across Models, Properties and Queries.
- The Property/Scenario/Query chain is typed by state type with a `Refines` witness, so
  cross-machine mistakes do not compile while a product claim can still be verified on the
  protocol machine.
- Enumerated action classes come for free: `Action[I]` carries `Finite[I]`, so `schedule` has its
  eight classes without a declaration.

Awkward:

- **The macro split.** Compile-time search only across a module boundary, and `Query.pinned` has
  to reach the Query by reflection over a stable path. It works and is a known idiom, but it is a
  build-structure requirement, not a language feature, and the failure sits in `Pins.scala`
  rather than next to the Query.
- **Naming by phase.** Lean's `starts: [scheduled]` expands over the other fields by convention;
  Scala needs the `Phased` type class to find the field, and a composed Scenario's start needs
  a second `starts` overload (`operation at Phase.unscheduled`).
- **Shadowing.** The entity `operation` and the composition member `operation` cannot both be
  top-level in one file; the member lives inside the `object nexusCaller`, and Scenarios write
  `nexusCaller.operation(schedule)(...)`. `worker` the party and `worker` the member collide the
  same way inside the object.
- **The feature surface is large.** Context functions, `inline`/`transparent inline`,
  `Mirror`-based derivation, match types on tuples, `TupledFunction`, opaque types, quotes and
  splices. An author of a Model touches none of them, but the maintainer of `Umpire.scala`
  touches all of them, and the error messages from a failing derivation or a mis-typed
  `TupledFunction` are the compiler's, not the DSL's, unless every path is wrapped in
  `compiletime.error`.
- **Toolchain weight.** A JVM, sbt or scala-cli, Metals, and compile times measured in seconds
  rather than milliseconds. Fine for a team that already has a JVM build; a new dependency for one
  that does not.
- **Team fit.** Temporal's server is Go and its Umpire runtime is Go. Scala is not in the stack,
  so a Scala model layer adds a language, a build and a hiring profile for the sake of the DSL.
  The nearest JVM alternative the team already touches is Java (the Java SDK), which has none of
  the compile-time machinery above.

## Libraries to leverage

Existing libraries that would cut the effort of building `Umpire.scala` for real. Maintenance was
checked on 2026-09-29 with `gh api repos/<owner>/<repo>` (last push, archived flag, stars); anything
archived or without a push in the last twelve months is marked no-go.

| Library | Area | Last push | Status | What it would replace |
| --- | --- | --- | --- | --- |
| [typelevel/scalacheck](https://github.com/typelevel/scalacheck) | stateful property testing | 2026-09-29 | maintained | `Commands` gives sequences of actions against a model state with shrinking; it replaces the exploratory set's driver and is a second, randomized reading of the table next to the exhaustive search. Not the search itself: `Commands` samples, it does not enumerate. |
| [hedgehogqa/scala-hedgehog](https://github.com/hedgehogqa/scala-hedgehog) | stateful property testing | 2026-09-17 | maintained | Same role as ScalaCheck `Commands` with integrated shrinking; pick one of the two, ScalaCheck has the larger user base. |
| [typelevel/scalacheck-effect](https://github.com/typelevel/scalacheck-effect) | property testing over effects | 2026-09-14 | maintained | Only if the driver runs Cases against a live server inside an effect type; otherwise not needed. |
| [tlaplus/tlaplus](https://github.com/tlaplus/tlaplus) | explicit-state model checking | 2026-09-28 | maintained | TLC is a Java jar and can be driven from the JVM, but it checks TLA+ specs, not Scala step functions; using it means translating the table to TLA+ and would replace the bounded search and `verify` Queries with a full checker. Realistic as an export target, not as an embedded engine. |
| [javapathfinder/jpf-core](https://github.com/javapathfinder/jpf-core) | JVM model checking | 2026-09-17 | maintained | Checks JVM bytecode for concurrency schedules; it does not fit a finite table over pure step functions. Listed because it is the JVM model checker people name; not recommended here. |
| [scalapb/ScalaPB](https://github.com/scalapb/ScalaPB) | protobuf | 2026-09-25 | maintained | Generates case classes and descriptors from `temporal.api.*` protos; replaces hand-written `schema` strings with typed references and gives the realization layer its message types. |
| [grpc/grpc-java](https://github.com/grpc/grpc-java) | gRPC | 2026-09-29 | maintained | The transport under ScalaPB's gRPC codegen for the driver that talks to a Temporal server. |
| [typelevel/fs2-grpc](https://github.com/typelevel/fs2-grpc) | gRPC, effectful | 2026-09-22 | maintained | Only if the driver is written on Cats Effect; otherwise plain grpc-java through ScalaPB. |
| [scalameta/munit](https://github.com/scalameta/munit) | tests | 2026-09-18 | maintained | The test runner `Pins.scala` uses; small, Scala 3 native, `assertEquals` diffs on case classes. |
| [plokhotnyuk/jsoniter-scala](https://github.com/plokhotnyuk/jsoniter-scala) | canonical JSON | 2026-09-29 | maintained | Macro-derived codecs with deterministic field order and no reflection; replaces a hand-rolled canonical JSON writer for fixtures and the Behavior Fingerprint. |
| [circe/circe](https://github.com/circe/circe) | JSON | 2026-09-14 | maintained | The alternative to jsoniter-scala; an ADT `Json` you can sort keys on before printing. Slower and heavier, but the fixture files are small. Pick one. |
| [google/guava](https://github.com/google/guava) | hashing | 2026-09-29 | maintained | `Hashing.sha256()` and `HashCode` for the fingerprint; `java.security.MessageDigest` in the JDK does the same with more ceremony, so Guava is optional. |
| [softwaremill/magnolia](https://github.com/softwaremill/magnolia) | type-class derivation | 2026-09-24 | maintained | Replaces the hand-written `Mirror` recursion in `Finite.derived` and `Phased.derived` with a derivation that names the field in its errors. |
| [typelevel/kittens](https://github.com/typelevel/kittens) | type-class derivation | 2026-09-20 | maintained | Same role as Magnolia, on Cats type classes; only if Cats is already in. |
| [softwaremill/quicklens](https://github.com/softwaremill/quicklens) | nested `copy` | 2026-09-24 | maintained | `state.modify(_.attempts).using(_.saturatingSucc)`; a convenience for step functions on composed states, not essential. |
| [typelevel/cats](https://github.com/typelevel/cats) | functional core | 2026-09-22 | maintained | `Eq`, `Order` and `Show` for states and facts, and `Foldable` over the table; useful if the team already speaks Cats, otherwise the standard library covers this sample. |
| [typelevel/cats-effect](https://github.com/typelevel/cats-effect) | effects | 2026-09-28 | maintained | Only for a driver that runs Cases concurrently against a server; the model layer is pure and needs none of it. |
| [epfl-lara/stainless](https://github.com/epfl-lara/stainless) | verification | 2026-09-18 | maintained | Verifies a Scala subset with pre/postconditions; a `verify` Query could become a proved lemma about the step functions. Real, but a separate toolchain and a research-grade loop. Not for the first version. |
| [scalameta/scalameta](https://github.com/scalameta/scalameta) | syntax trees | 2026-09-23 | maintained | For the drift test that reads `// authoring:` regions out of the Model files; the quotes API inside a macro covers everything else. |
| [lloydmeta/enumeratum](https://github.com/lloydmeta/enumeratum) | enums | 2026-06-13 | maintained | Not needed: Scala 3 `enum` with `values`, `ordinal` and `Mirror` covers everything Enumeratum added to Scala 2, including parametrised cases. |
| [disneystreaming/weaver-test](https://github.com/disneystreaming/weaver-test) | tests | 2025-06-02 | archived, no-go | Was the Cats Effect test framework; archived. munit instead. |

Recommended set for a real build: ScalaCheck for the exploratory driver, ScalaPB with grpc-java
for schemas and the driver, munit for the pins, jsoniter-scala for canonical JSON, the JDK's
`MessageDigest` for hashing, Magnolia for the derivations. Everything else above is either
optional or a later step.

## Notes on fidelity to the spec

- Enums get `CanEqual` automatically in Scala 3, so only the `case class` states spell it out.
- `Timeout` is imported into the activity Model from the Nexus Model, as the spec says "reuse";
  a real layout would move it to a shared domains file.
- The activity protocol's `control` step reads every control on a finished activity as
  `notFound`, including `pause` and `unpause`, which is the spec's terminal-to-`notFound` line
  applied uniformly and matches the product step.
- The refinement check is by mapped states, not by action class: a protocol row is accounted for
  when its two states map to one product state (a stutter) or to two the product has any row
  between. `Refinement.rejected` in `Umpire.scala` is specified that way.
