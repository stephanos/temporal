# Review: cmp/kotlin/

## Scores (1-5, 5 best) with one sentence of justification each

1. **Spec fidelity: 5.** Both Models are complete with every declaration name exactly as SPEC.md, the Lean section order is kept (entities, domains, actions, observation, product, protocol, properties, scenarios+limits, queries, three sets, a Cases placeholder, composition), the revised Model 2 is applied (`Phase.PauseRequested -> ProductPhase.Started` at `StandaloneActivity.kt:452`, visible retry rows at `StandaloneActivity.kt:169-170`), and every pin from the spec is present in `Pins.kt`; the only deviation is UpperCamelCase enum entries, which the README discloses at `README.md:181-184`.
2. **Language plausibility: 4.** The DSL is textbook Kotlin (lambdas-with-receiver, `@DslMarker`, infix members, reified builders, function references), but `Umpire.kt` has a handful of real compile errors a fluent expert would catch: public `inline` functions reaching into `internal`/`private` declarations (`Umpire.kt:237-247`, `416-419`, `726-733`), `Action.name` missing `override` (`Umpire.kt:158`), and `lateinit` on a value-class typed property (`Umpire.kt:193`), all fixable without redesign.
3. **Authoring readability: 4.** The declarative blocks read as configuration (`NexusCaller.kt:460-489`, `665-672`, `680-688`), step functions are plain functions with exhaustive `when`, and comments are carried over; the cost is ceremony around 25-line import blocks, fully qualified `ProtocolFact.X`/`Phase.X` everywhere, and `listOf(Step(...))` constructions, pushing each Model near the Lean file's length.
4. **Check story accuracy: 3.** The README is unusually candid that the semantic checks are init-time, not compile-time (`README.md:104-109`, `172-174`), but it overclaims twice: `operation[Worker.serve]`/`worker[handlerReply]` is said to be a type error (`README.md:86`, `96`) when `Member.get` accepts any `Action0`/`Action1` and only fails via a runtime `check` (`Umpire.kt:659-666`); and "there is no `else` to hide behind" (`README.md:121`) is untrue for Model 2, where eight step-function `when`s use `else -> emptyList()` (`StandaloneActivity.kt:187,195,211,311,378,391,396,441`).
5. **Framework realism: 3.** Finite enumeration is implemented for real (`Umpire.kt:45-82`), builder validations are concrete and named, but the three things that matter most are `TODO()`: `computeTransitions` (`Umpire.kt:319`), `checkRefinement` (`Umpire.kt:387-392`), and `Query.run` (`Umpire.kt:519`); worse, `Refinement` is immutable and constructed with `emptyMap(), null` at `Umpire.kt:368` so `build()` cannot fill `rows` as claimed, and `Composition` passes `emptyMap()` steps (`Umpire.kt:682`) with no sketch of how a product-with-syncs table would be computed, so the two composition `verify` Queries have nothing to search.
6. **README honesty and library table: 5.** Costs are stated plainly (runtime checks, `kotlin-reflect`, JVM startup, `Set` shadowing, top-level `val` ordering hazard); all four spot-checked claims match the GitHub API exactly (see below).

Spot-check results (2026-09-29):

| Library | README claims | `gh api` `pushed_at` | archived | Match |
| --- | --- | --- | --- | --- |
| jqwik-team/jqwik | 2026-09-28, maintained | 2026-09-28T07:44:55Z | false | yes |
| erdtman/java-json-canonicalization | 2020-10-13, no-go | 2020-10-13T08:10:07Z | false | yes |
| ZacSweers/kotlin-compile-testing | 2026-09-24, maintained | 2026-09-24T03:02:34Z | false | yes |
| kotest/kotest | 2026-09-28, maintained | 2026-09-28T08:21:26Z | false | yes |

## Verbatim snippets (for side-by-side comparison; trim comments)

`handlerReply` step of the Nexus product machine (`NexusCaller.kt:167-180`):

```kotlin
fun handlerReplyStep(state: ProductState, reply: Reply): ProductSteps {
    if (state.phase != ProductPhase.Scheduled) return emptyList()
    return when (reply) {
        Reply.SyncSuccess -> productStep(ProductPhase.Succeeded, ProductFact.NexusOperationCompleted)
        Reply.Async -> productStep(ProductPhase.Started, ProductFact.NexusOperationStarted)
        Reply.OperationFailed -> productStep(ProductPhase.Failed, ProductFact.NexusOperationFailed)
        Reply.OperationCanceled -> productStep(ProductPhase.Canceled, ProductFact.NexusOperationCanceled)
        is Reply.HandlerError ->
            if (reply.retryable) emptyList()
            else productStep(ProductPhase.Failed, ProductFact.NexusOperationFailed)
    }
}
```

`syncSucceeds` (`NexusCaller.kt:509-513`):

```kotlin
val syncSucceeds = property("syncSucceeds", nexusProtocol) {
    handlerReply(Reply.SyncSuccess) holds { step ->
        step.state.phase == Phase.Succeeded && ProtocolFact.NexusOperationCompleted in step.facts
    }
}
```

`syncReplied` and `syncCompletion` (`NexusCaller.kt:593-596`, `665`):

```kotlin
val syncReplied = scenario("syncReplied", nexusProtocol) {
    starts(ProtocolState(Phase.Unscheduled))
    actions(schedule(unset, unset, unset), handlerReply(Reply.SyncSuccess))
}

val syncCompletion = query("syncCompletion") { find(syncSucceeds) on syncReplied within two }
```

`nexusCaller` compose block (`NexusCaller.kt:739-749`):

```kotlin
data class NexusCallerState(val operation: ProtocolState, val worker: Worker.WorkerState)

val nexusCaller = compose<NexusCallerState>("nexusCaller") {
    val operation = member(NexusCallerState::operation, nexusProtocol)
    val worker = member(NexusCallerState::worker, handlerWorker)
    workerStop syncs operation[workerStop] with worker[Worker.workerStop]
    handlerReply syncs operation[handlerReply] with worker[Worker.serve]
    starts(NexusCallerState(ProtocolState(Phase.Unscheduled), Worker.WorkerState(Worker.Phase.Polling)))
    ends { terminalPhase(this.operation.phase) }
}
```

## Line counts

| File | Lines |
| --- | --- |
| `NexusCaller.kt` | 777 |
| `StandaloneActivity.kt` | 714 |
| `Umpire.kt` | 747 |
| `README.md` | 254 |
| `Pins.kt` | 159 |
| `Worker.kt` | 91 |
| Total | 2742 |

Two Model files alone (nexus + standalone): 1491.

## Red flags

- **Composition member binding is claimed compile-time but is runtime.** `README.md:86` and `README.md:96` say `worker[handlerReply]` and `operation[Worker.serve]` do not type-check. `Member.get` at `Umpire.kt:663-666` takes any `Action0`/`Action1<A>`/`Action3<A,B,C>`/`Timer` regardless of which machine owns it, and the only guard is `checked()` at `Umpire.kt:659-661`, a runtime `check`. A decision maker reading the "compile" row would be misled.
- **Public inline functions access non-public API.** `action<A>`, `action<A,B,C>`, `machine`, and `compose` are public `inline` and call `internal` constructors (`ActionSpec`, `Action1`, `MachineScope`, `ComposeScope`, `Composition`), the `private` `validated()`, and `internal` members (`inputDomains`, `build()`, `scope.members`). Kotlin rejects this with "Public-API inline function cannot access non-public-API"; the fix is `@PublishedApi internal`, but as written the framework file does not compile.
- **`Action.name` lacks `override`.** `Umpire.kt:158` declares `val name: String` on a class implementing `Trigger` which has abstract `val name` at `Umpire.kt:136`; that is a hard compile error. `Timer` gets it right at `Umpire.kt:187`.
- **`lateinit var party: Party` on a value class.** `Umpire.kt:193` uses `lateinit` on a `@JvmInline value class` typed property. Kotlin 1.x rejects `lateinit` on inline-class types; I am not certain K2 lifted this, so treat as likely rejected.
- **Refinement rows cannot be derived as documented.** `Refinement` is immutable (`Umpire.kt:275-280`) and `via` constructs it with `emptyMap(), null` (`Umpire.kt:368`) with a comment "derived in build()". `build()` at `Umpire.kt:382-384` cannot mutate it, so the pin `refinement.rows shouldHaveSize nexusProtocol.transitions.size` (`Pins.kt:99`) would fail even after `checkRefinement` were implemented, unless `build()` is restructured.
- **Composition table is not sketched at all.** `Composition` passes `emptyMap()` as `steps` (`Umpire.kt:682`), so `table`, `transitions`, `stuck` and any search over `nexusCaller` or `standaloneActivity` operate on an empty machine. The two cross-entity `verify` Queries (`NexusCaller.kt:775`, `StandaloneActivity.kt:712`) therefore bind to nothing that can run. `Composition.actions` at `Umpire.kt:690-691` also contradicts its own comment ("every member action no sync names") by including all member actions.
- **"No `else` to hide behind" is false for Model 2.** `README.md:121` claims step functions have no `else`, so a new constructor is a list of compiler errors. `StandaloneActivity.kt` uses `else -> emptyList()` in eight `when`s over `Phase`/`ProductPhase`; adding a phase there is silent.
- **`README.md:99`** lists "finite table, stuck, reachable" as computed "when the machine `val` initializes", but `table` is `by lazy` and `stuck`/`reachable()` are on-demand; they run only from the pins.

## Strengths

- **Faithful semantics, including the revised Model 2.** Every step function matches SPEC.md row for row; I checked each protocol transition of `activityProtocol` against `productOf` and the product table under the mapped-state rule, and all are stutters or product rows, so the refinement pin would pass. `pausedIsNotDispatched` holds over the whole product table.
- **Typed action arity is real.** `Action0`/`Action1<A>`/`Action3<A,B,C>` plus `StepsScope.runs` overloads (`Umpire.kt:403-413`) genuinely make `schedule runs ::twoArgFn` or `handlerReply runs ::fnOverResolution` a compile error. Unresolved-reference for a mistyped action name is likewise real because actions are `val`s.
- **Exhaustive `when` for facts and replies is used as a design lever.** `evidence { when (fact) ... }` as an expression (`NexusCaller.kt:467-477`) forces a decision per fact, and `Reply`/`AttemptResult` as sealed interfaces make the reply arms exhaustive; the `HandlerError(retryable)` two-class encoding via `Finite.of` (`Umpire.kt:57-59`) is a neat, working answer to Lean's `Fin`/inductive granularity.
- **The `this.operation` shadowing comment** at `NexusCaller.kt:747` shows the author knows Kotlin resolution order (locals beat implicit receiver members); this is the kind of detail a fluent expert writes.
- **The README's cost accounting is the most honest part.** It separates compile-time from `ExceptionInInitializerError`-at-first-test, names `kotlin-reflect` as a runtime dependency, explains the `Set` shadow and the top-level `val` ordering hazard, and its 25-row library table checks out on every spot-check.

## One-paragraph verdict

This sample shows Kotlin can express the Umpire model layer as a readable builder DSL where the declarative parts look like Gradle-style configuration and the step functions stay plain, exhaustive, unit-testable functions; action arity, undeclared action names, and missing `when` arms are caught by the compiler, and everything else (Property/Scenario/Query admission, refinement) is a well-messaged runtime `check` at class load, which the README says plainly. The single biggest reservation is that the framework surface is thinner than it looks: enumeration is real, but the table, the refinement walk, the search, and above all the composition product are `TODO()` or structurally impossible as sketched (immutable `Refinement` built empty, `Composition` with no steps), and the README's two compile-time claims about member binding are overstated, so a decision maker should read this as "Kotlin gives good ergonomics and modest static checking" rather than "Kotlin catches the model-level errors early."
