# Kotlin sample

The same two Models as `lean/`, written the way a Kotlin team would author them: typed builders with
lambdas-with-receiver for the declarations, plain functions with exhaustive `when` for the step
functions, and a kotest spec for the pins. Nothing here has been compiled; the files are written to
be recognizable to a Kotlin expert, not to pass `gradle build` today.

| File | What it holds |
| --- | --- |
| `Umpire.kt` | The framework surface: `Step`, `Finite`, `Machine`, `Refinement`, `Property`, `Scenario`, `Limits`, `Query`, `Set`, `Composition`, and the builders. Bodies sketched or `TODO()`. |
| `Worker.kt` | The `Worker` module both compositions synchronize with, as in `lean/Worker.lean`. |
| `NexusCaller.kt` | Model 1, in the Lean file's section order. |
| `StandaloneActivity.kt` | Model 2, same order. |
| `Pins.kt` | The `#guard` pins as two kotest `FunSpec`s. |

## The DSL mechanism

Every declaration is a top-level `val` built by a function that takes a name and a
lambda-with-receiver:

```kotlin
val handlerReply = action<Reply>("handlerReply") {
    party = handler
    on = operation
    schema = "temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError"
    input("reply")
    examples {
        Reply.HandlerError(retryable = false) realizedAs "BadRequest"
        Reply.HandlerError(retryable = true) realizedAs "Internal"
    }
}

val nexusProtocol = machine<ProtocolState, ProtocolOutcome, ProtocolFact>("nexusProtocol") {
    entity = operation
    refines(nexusProduct) via ::productOf
    starts(ProtocolState(Phase.Unscheduled))
    ends { terminalPhase(phase) }
    timers(backoff, scheduleToClose, scheduleToStart, startToClose)
    unobservable(backoff)
    evidence { fact -> when (fact) { /* one arm per fact */ } }
    steps {
        schedule runs ::scheduleStep
        handlerReply runs ::protocolHandlerReplyStep
        /* ... */
    }
}

val syncSucceeds = property("syncSucceeds", nexusProtocol) {
    handlerReply(Reply.SyncSuccess) holds { step ->
        step.state.phase == Phase.Succeeded && ProtocolFact.NexusOperationCompleted in step.facts
    }
}

val syncCompletion = query("syncCompletion") { find(syncSucceeds) on syncReplied within two }
```

Why this shape, and not annotations or a string DSL:

- **Receivers scope the vocabulary.** Inside `machine { }` the IDE completes `starts`, `ends`,
  `timers`, `steps`; inside `steps { }` it completes `runs` on an action and nothing else. `@DslMarker`
  (`@UmpireDsl` in `Umpire.kt`) stops an inner block from seeing the outer block's members, so
  `steps { timers(...) }` is a compile error rather than a silent write to the wrong scope.
- **Actions are values, so their arity is a type.** `action<Reply>("handlerReply")` is an `Action1<Reply>`
  and `handlerReply runs ::fn` only accepts `(ProtocolState, Reply) -> List<Step<...>>`. Binding
  `schedule` to a two-argument function, or `handlerReply` to a function over `Resolution`, is a type
  error at the `runs` line. Function references (`::scheduleStep`) keep the step functions plain, named,
  and callable from a test.
- **Infix functions read as the Lean keywords.** `handlerReply(SyncSuccess) holds { ... }`,
  `find(syncSucceeds) on syncReplied within two`, `refines(nexusProduct) via ::productOf`,
  `workerStop syncs operation[workerStop] with worker[Worker.workerStop]`. Where the Lean keyword is a
  Kotlin hard keyword (`for`, `in`, `when`), the Kotlin name is the nearest word: `entity =`, `on`, and
  the `holds` receiver.
- **Overloading by lambda arity separates the two claim forms.** `holds { step -> }` after a trigger is a
  same-step claim; a bare `holds { before, after -> }` is a transition claim. No `when:` line is
  needed to tell them apart.
- **`Finite<T>` is resolved from the type.** `machine<S, O, F>` is `reified`, so the state table is
  enumerated from `S` without the author listing anything: `enum class` entries, `sealed interface`
  subtypes (one per assignment of a `data class` subtype's constructor parameters, which is how
  `HandlerError(retryable)` becomes two classes), `data class` states as the cartesian product of their
  fields, and an explicit `companion object : Finite<T>` for the bounded `Attempts`. This uses
  `kotlin-reflect` (`sealedSubclasses`, `primaryConstructor`); a KSP processor could generate the same
  lists at build time, and `Finite.of` already consults a companion first so the swap needs no change
  in a Model file.
- **Compositions project with property references.** `member(NexusCallerState::operation, nexusProtocol)`
  ties the member's state type to the field's type, so `operation[schedule]` is again an
  `Action3<Timeout, Timeout, Timeout>` and its inputs are typed. Which machine owns the action is not
  in the type: `worker[handlerReply]` compiles and fails at init with a message naming the member.

## Where each check runs

| Check | When | How it surfaces |
| --- | --- | --- |
| A step function names an action that does not exist | compile | `Unresolved reference 'timout'.` Actions are `val`s, not strings. |
| A step function's parameters do not match the action's inputs | compile | type mismatch at the `runs` line |
| A `when` over a `sealed`/`enum` domain misses a case | compile | see below |
| A fact is added without deciding its evidence | compile | the `evidence { when (fact) ... }` block stops being exhaustive |
| A composition names an action its member's machine lacks | when the `val` initializes | `IllegalStateException: composition member worker: handlerWorker has no action 'handlerReply'` |
| A Property names an action its machine has no step for | when the `val` initializes | `IllegalStateException: property timesOut: nexusProtocol has no action 'timeout'` |
| A Scenario lists an action its model lacks; a Query pairs a Property with a Scenario of an unrelated machine; a Set of purpose `Functional` lists a `verify` Query | when the `val` initializes | `IllegalStateException` naming the declaration, thrown from `check(...)` in `Umpire.kt` |
| The finite table, `stuck`, `reachable` | on demand: the table is `by lazy`, the other two are computed when asked | pinned in `Pins.kt`, which is the first to ask |
| The refinement (`refines ... via`) | when the machine `val` initializes | `nexusProtocol does not refine nexusProduct: row ...`, or `refinement.rejected == null` in the pins |
| Search results (`find`, `verify`) | test | `Query.run()` from `Pins.kt` |
| A canary naming a Query whose path has a silent step | not here | belongs to Case production, out of this comparison's scope |

"When the `val` initializes" means the first time the file's JVM class (`NexusCallerKt`) is loaded.
In practice that is the first test touching any declaration in the file, and the failure arrives as an
`ExceptionInInitializerError` with the builder's message as its cause. It is not compile time, and the
IDE will not underline it. A Kotlin compiler plugin (a `FirAdditionalCheckersExtension` in K2) could
run the same `check` calls against the call sites, since every argument is a constant reference; that
is the path to token-pinned errors, and it is real engineering, not configuration.

## What an author's mistake looks like

**A missing `when` branch** (a new `Reply` constructor, or a forgotten `HandlerError` arm), from
`kotlinc` 2.x:

```
e: file:///.../NexusCaller.kt:171:12 'when' expression must be exhaustive. Add the 'is HandlerError' branch or an 'else' branch.
```

The IDE shows the same message inline with a quick-fix that inserts the branch. This bites only
where the `when` has no `else`: the arms over input domains (`Reply`, `AttemptResult`, `Control`)
and over facts are written that way, but eight `when`s over `Phase` in `StandaloneActivity.kt` end
in `else -> emptyList()`, so adding a phase there is silent. Spelling out every phase is the price
of the compiler's help, and the sample pays it only where the arms differ.

**A Property on a product-only action, read on the protocol machine** (the Lean pins' `timesOut`
example):

```
java.lang.ExceptionInInitializerError
Caused by: java.lang.IllegalStateException: query timesOutOnProtocol: the Property names the action 'timeout' of 'nexusProduct', and 'nexusProtocol' has no action of that name; a Property on the refined machine is read on the refining one through the values of the same name, and a state through its map
    at umpire.QueryScope$Half.on(Umpire.kt:...)
    at temporal.feature.nexus.caller.NexusCallerKt.<clinit>(NexusCaller.kt:...)
```

**A failed pin**, from kotest under Gradle:

```
NexusCallerPins > the Queries > each functional Query finds its claim on its path FAILED
    io.kotest.assertions.AssertionFailedError: Expected instance of umpire.Outcome$Found but was NotFound
        at Pins.kt:66
```

## Toolchain and loop

- JDK 17+, Kotlin 2.x, Gradle with the Kotlin JVM plugin; `kotlin-reflect` on the runtime classpath
  for `Finite.of`; kotest (`kotest-runner-junit5`, `kotest-assertions-core`) for the pins.
- Edit-to-compile-error in the IDE: immediate, for everything in the "compile" rows above.
- Edit-to-pin-result: `./gradlew test` with the Gradle daemon warm is a few seconds of compilation
  plus the table and search time; the first run of a session pays JVM and daemon start-up.
  CI needs a JDK and a Gradle cache; it is the heaviest toolchain in this comparison after Lean's.
- The Java SDK and gRPC ecosystem are on the same classpath: the `schema` strings in the actions name
  `temporal.api.*` messages that exist as generated Java classes, so a realization layer could take
  `KClass<out com.google.protobuf.Message>` instead of a string, and a driven party could be a
  Temporal Java SDK worker in the same test process.

## What Kotlin made easy, and what it made awkward

Easy:

- The builder DSL reads as configuration and the IDE completes it scoped by receiver. This is the
  most readable mainstream option in this comparison, and an author who has used Gradle's Kotlin DSL
  or kotlinx.html needs no introduction.
- Sealed exhaustiveness covers every `when` in the step functions, the `evidence` blocks and
  `productOf`. Adding a phase or a fact is a list of compiler errors to walk.
- Function references keep the step functions ordinary functions: named, documented, callable from a
  pin (`handlerReplyStep(ProductState(Scheduled), HandlerError(true)) shouldBe emptyList()`).
- `data class` states give `copy(phase = ...)` for the `moves` helper and structural equality for
  `step.state == succeededOnRetry`; default arguments make `ProtocolState(Phase.Unscheduled)` the
  start state without naming the other four fields.
- `value class Attempts` with a companion `Finite` is the bounded integer, checked in `init`.

Awkward, and stated plainly:

- **Semantic checks run when the builder runs, not at compile time.** Everything in the "initializes"
  rows above is a runtime `check`. It is caught by the first test, not by the compiler, and not by
  the IDE.
- **No nested patterns.** Lean writes `| .handlerError true => []`; Kotlin writes
  `is Reply.HandlerError -> if (reply.retryable) ... else ...`. Exhaustiveness covers the constructor,
  not the field.
- **Enum entries cannot carry per-entry fields**, so a domain with one parameterized constructor
  (`Reply`, `AttemptResult`, `ProtocolFact`) has to be a `sealed interface` of `data object`s and a
  `data class`, while a flat domain stays an `enum class`. Two spellings for one concept.
- **Casing.** Kotlin types and `data object`s are UpperCamelCase, so the spec's `syncSuccess`,
  `backingOff`, `nexusOperationCompleted` appear as `SyncSuccess`, `BackingOff`,
  `NexusOperationCompleted`. Declaration names (`nexusProtocol`, `handlerReply`, `retrySucceeds`)
  match the spec exactly.
- **Hard keywords.** `for`, `in` and `when` are unavailable as DSL words; `entity =`, `on` and the
  `holds` receiver stand in.
- **`Set`.** Naming the framework type after the Lean keyword shadows `kotlin.collections.Set` inside
  package `umpire` and in any file that star-imports it. `Umpire.kt` spells the collection type out
  once; the Model files import by name.
- **No package alias.** Both Models declare `three`, `four`, `retry`, `terminalHolds` and the phase
  types, so `Pins.kt` imports the activity's under `as` aliases, and the Worker module is an `object`
  so `Worker.polling` reads as in Lean.
- **Top-level `val` order matters.** A lambda evaluated during a `val`'s initializer that references a
  `val` declared later in the file sees `null` on the JVM. The Models are ordered so every reference
  points upward; nothing enforces it.
- **The Property-to-Scenario check crosses machine types.** A product Property is read on a protocol
  Scenario through the refinement map, so `find(p) on s` takes `Property<*, *, *>` and checks the
  relation at init rather than in the type system. A phantom type parameter for "the machine this
  Property is about or refines" would type it, at the cost of a fourth type argument everywhere.
- **The machine is a parameter of `property(...)`**, not a line in its block, because the `step` in
  `holds { step -> }` is typed by it and Kotlin infers lambda parameter types from the receiver, not
  from an assignment inside the block.
- **Reflection at class load** (`sealedSubclasses`, `primaryConstructor.call`) is where `Finite` comes
  from. It is cached, small, and one KSP processor away from build time, but today it is a runtime
  dependency on `kotlin-reflect`.

## Libraries to leverage

Existing JVM and Kotlin libraries that would cut the effort of building `Umpire.kt` for real.
Maintenance status was checked on 2026-09-29 with
`gh api repos/<owner>/<repo> --jq '{pushed_at, archived, stargazers_count}'`; anything with no push
in the twelve months before that date, or archived, is marked no-go and not recommended.

| Library | Last push | Status | What it would replace |
| --- | --- | --- | --- |
| [kotest](https://github.com/kotest/kotest) (`kotest-framework`, `kotest-assertions`, `kotest-property`) | 2026-09-28 | maintained | The pin runner and assertions in `Pins.kt`; `kotest-property`'s generators and shrinking for the exploratory set's random walks over the table. |
| [jqwik](https://github.com/jqwik-team/jqwik) | 2026-09-28 | maintained | Stateful, action-based property testing: its `ActionChain` with a model state and invariants is the same shape as a Scenario plus a transition claim, so the `verify` mode of `Query.run()` over random paths could be jqwik chains rather than a hand-written walker. |
| [grpc-java](https://github.com/grpc/grpc-java) + [grpc-kotlin](https://github.com/grpc/grpc-kotlin) | 2026-09-29 / 2025-12-19 | maintained | The transport for a driven party talking to a Temporal frontend, and the `Case` protobuf exchange with the Go Testpilot runtime; grpc-kotlin adds coroutine stubs. |
| [protobuf](https://github.com/protocolbuffers/protobuf) (`protobuf-java`, `protobuf-kotlin`) | 2026-09-29 | maintained | The `schema` strings in the action declarations become `KClass<out Message>` references to generated `temporal.api.*` classes; the `Case` format is emitted from generated builders with the Kotlin DSL. |
| [wire](https://github.com/square/wire) | 2026-09-28 | maintained | An alternative protobuf compiler that generates Kotlin `data class`es with `copy`, which would make protobuf messages usable directly as `Finite` states; a choice against `protobuf-kotlin`, not in addition to it. |
| [kotlinx.serialization](https://github.com/Kotlin/kotlinx.serialization) | 2026-09-29 | maintained | Serializing states, tables and coverage targets to the fixture JSON the Lean pins compare against; compile-time serializers for the sealed fact types. |
| [KSP](https://github.com/google/ksp) | 2026-09-25 | maintained | Generating `Finite.entries` for every `enum`, `sealed` and `data class` state at build time, retiring the `kotlin-reflect` dependency in `Finite.of`; also a place to emit the machine's action catalog as constants. |
| [KotlinPoet](https://github.com/square/kotlinpoet) | 2026-09-26 | maintained | The code emitter behind the KSP processor above. |
| [kotlin-compile-testing (ZacSweers fork)](https://github.com/ZacSweers/kotlin-compile-testing) | 2026-09-24 | maintained | Testing the KSP processor or a compiler plugin: assert that a missing `when` branch or an undeclared action produces the expected diagnostic. |
| [Kotlin compiler (K2 plugin API)](https://github.com/JetBrains/kotlin) | 2026-09-29 | maintained | Moving the init-time `check` calls (Property names an action of its machine, Scenario lists actions of its model) to compile time as FIR checkers. The API is stable enough to use but not yet declared stable. |
| [Arrow](https://github.com/arrow-kt/arrow) (`arrow-core`) | 2026-09-26 | maintained | `Either`/`Raise` for the admission path of a Query and typed errors for `Refinement.rejected`, if the framework wants typed failures instead of exceptions. Optional. |
| [Temporal Java SDK](https://github.com/temporalio/sdk-java) | 2026-09-29 | maintained | A driven `caller`, `handler` or `worker` party in the same test process: a Nexus handler worker, an activity worker, and the `WorkflowServiceStubs` for `Describe`/`Poll` observations. |
| [kotlinx.collections.immutable](https://github.com/Kotlin/kotlinx.collections.immutable) | 2026-09-24 | maintained | Persistent sets and maps for the explicit-state search frontier and the visited set in `reachable()` and `Query.run()`. |
| [TLA+ tools (TLC)](https://github.com/tlaplus/tlaplus) | 2026-09-28 | maintained | An explicit-state model checker on the JVM. Realistic only as an external check: a machine's table could be emitted as a TLA+ spec and TLC run on it for the `verify` Queries; embedding TLC as a library is possible (`tla2tools.jar`) but its API is not designed for it. |
| [Apalache](https://github.com/informalsystems/apalache) | 2026-09-24 | maintained | Symbolic model checking of TLA+ on the JVM (Scala). Same external-check role as TLC, for bounded verification over larger tables. Not a library dependency. |
| [Java PathFinder](https://github.com/javapathfinder/jpf-core) | 2026-09-17 | maintained | Model checking of JVM bytecode. Not a fit: the tables here are finite and enumerable directly, so JPF's exploration of JVM state adds cost without insight. Listed to close the question. |
| [Jackson](https://github.com/FasterXML/jackson-databind) + [jackson-module-kotlin](https://github.com/FasterXML/jackson-module-kotlin) | 2026-09-29 / 2026-09-22 | maintained | Canonical JSON for fixtures and the Behavior Fingerprint: `SORT_PROPERTIES_ALPHABETICALLY` and `ORDER_MAP_ENTRIES_BY_KEYS` give a deterministic byte stream; not RFC 8785 number formatting, which the fixtures do not need since every value is a string, enum or small integer. |
| [Guava](https://github.com/google/guava) (`Hashing`) or [commons-codec](https://github.com/apache/commons-codec) | 2026-09-29 | maintained | SHA-256 over the canonical JSON for the fingerprint. `java.security.MessageDigest` also suffices with no dependency. |
| [erdtman/java-json-canonicalization](https://github.com/erdtman/java-json-canonicalization) (RFC 8785) | 2020-10-13 | not maintained, no-go | Would have been the RFC 8785 canonicalizer. Use Jackson's ordering instead. |
| [cyberphone/json-canonicalization](https://github.com/cyberphone/json-canonicalization) (RFC 8785 reference, Java module) | 2024-12-13 | not maintained, no-go | Same role, same replacement. |
| [QuickTheories](https://github.com/quicktheories/QuickTheories) | 2020-10-13 | not maintained, no-go | Property testing; kotest-property or jqwik instead. |
| [junit-quickcheck](https://github.com/pholser/junit-quickcheck) | 2024-11-18 | not maintained, no-go | Property testing; jqwik instead. |
| [tschuchortdev/kotlin-compile-testing](https://github.com/tschuchortdev/kotlin-compile-testing) | 2024-07-15 | not maintained, no-go | The original compile-testing library; the ZacSweers fork above is the maintained line. |
| [Arrow Meta](https://github.com/arrow-kt/arrow-meta) | 2026-06-04 | maintained, but not recommended | A compiler-plugin framework whose K2 support lags the compiler; writing against the K2 FIR API directly is the safer route. |

The shortest realistic stack: kotest for pins and property generators, jqwik for stateful search
over random paths, `protobuf-kotlin` and grpc-kotlin for the Case exchange and driven parties, the
Temporal Java SDK for the parties themselves, kotlinx.serialization or Jackson for fixtures, and KSP
with KotlinPoet to remove reflection. The explicit-state search over a finite table is small enough
that no model-checking library is worth its integration cost; TLC is the only one with a plausible
external role.

## Spec note

Model 2 follows the revised SPEC.md: the refinement is by mapped states (a protocol row is a stutter
or maps onto some product row between the same mapped states, of any action class), `productOf`
reads a requested pause as started, and the product sees the retry as a return to scheduled. Under
that rule every protocol row of `activityProtocol` is a stutter or a product row, and the refinement
pin holds.
