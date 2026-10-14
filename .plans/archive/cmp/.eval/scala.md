# Review: cmp/scala

## Scores (1-5, 5 best)

1. **Spec fidelity: 5.** Both Models are complete with every name matching SPEC.md, Lean section order and comments are kept, the revised Model 2 is applied (`productOf` maps `pauseRequested` to `started` at `StandaloneActivity.scala:391`, visible retry rows at `StandaloneActivity.scala:148` and `:153`), and every listed pin appears in `Pins.scala`. The only deviations are structural: `workerStop` is imported from the worker module instead of declared in the Nexus Model (`NexusCaller.scala:99`), and `AttemptResult` lists `canceled` before `failed` (`StandaloneActivity.scala:38-40`).

2. **Language plausibility: 3.** The Model files are idiomatic Scala 3 and the author clearly knows the toolkit (fewer-braces blocks, `TupledFunction`, `Mirror` derivation, `Tuple1` for one-element examples), but several framework signatures cannot type-check as written: `starts(phases: Phased[S]#Phase*)` at `Umpire.scala:297` uses a projection on an abstract type member, so `starts(ProductPhase.scheduled)` would not check against it; `evidence[S,O,F](lines: F => Evidence)(using ...)` at `Umpire.scala:306` puts the using clause after the lambda, so the pattern-matching lambda has no parameter type; `Entity` at `Umpire.scala:146-148` overloads a `val key` with a `def key`, which Scala rejects; and `Finite.derived` at `Umpire.scala:63-70` summons `Finite` for enum singleton case types that no given provides, so `derives Finite` on any enum would hit the `error` branch.

3. **Authoring readability: 4.** Step functions are near line-for-line Lean and the `property ... when ... holds:` and `query ... find ... in ... limits` lines read like configuration, but every enum case is fully qualified (`ProductFact.nexusOperationCompleted`), every top-level `val` carries a type ascription, and `machine[S, O, F]("name"):` needs three explicit type arguments, which adds visible ceremony over the Lean.

4. **Check story accuracy: 3.** The README is honest about the macro split (`README.md:55-59`) and the exhaustivity and arity claims are correct, but its headline claim that "a Query that pairs a Property of one machine with a Scenario of another does not type-check" (`README.md:31-32`) contradicts the sample's own three verify Queries, which pair a `Property[ProductState]` with a `Scenario[ProtocolState]` (`NexusCaller.scala:551`, `StandaloneActivity.scala:566-567`), and the `Phased` and `derives Finite` compile-time checks rest on code that does not compile as sketched.

5. **Framework realism: 4.** Finite enumeration is fully worked out including a compile-time `sizeOf` that handles singleton cases and `Bounded` (`Umpire.scala:74-90`), the refinement contract is specified precisely by mapped states (`Umpire.scala:309-319`), and the search has a clear signature; but `Table.build`, `Refinement.rejected`, `Query.run`, `restrict` and `Compose.machine` are all `???`, and `Compose.sync`, `starts`, `ends` are no-ops (`Umpire.scala:535-537`) that record nothing.

6. **README honesty and library table: 5.** Costs are stated plainly (JVM weight, macro split, Scala not in Temporal's stack, large maintainer surface at `README.md:155-180`), and all spot-checks match the table.

| Library | README claim | gh api result |
| --- | --- | --- |
| typelevel/scalacheck | 2026-09-29, maintained | pushed 2026-09-29, not archived |
| disneystreaming/weaver-test | 2025-06-02, archived, no-go | pushed 2025-06-02, archived |
| lloydmeta/enumeratum | 2026-06-13, maintained | pushed 2026-06-13, not archived |
| softwaremill/magnolia | 2026-09-24, maintained | pushed 2026-09-24, not archived |
| scalameta/munit | 2026-09-18, maintained | pushed 2026-09-18, not archived |

## Verbatim snippets

The `handlerReply` step function of the Nexus product machine (`NexusCaller.scala:138-148`):

```scala
def handlerReplyStep(state: ProductState, reply: Reply): List[ProductStep] =
  if state.phase != ProductPhase.scheduled then Nil
  else reply match
    case Reply.syncSuccess => productStep(ProductPhase.succeeded, ProductFact.nexusOperationCompleted)
    case Reply.async => productStep(ProductPhase.started, ProductFact.nexusOperationStarted)
    case Reply.operationFailed => productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)
    case Reply.operationCanceled => productStep(ProductPhase.canceled, ProductFact.nexusOperationCanceled)
    case Reply.handlerError(true) => Nil
    case Reply.handlerError(false) => productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)
```

The `syncSucceeds` property (`NexusCaller.scala:425-427`):

```scala
val syncSucceeds: Property[ProtocolState] =
  property("syncSucceeds")(nexusProtocol) when handlerReply(Reply.syncSuccess) holds: step =>
    step.state.phase == Phase.succeeded && step.facts.contains(ProtocolFact.nexusOperationCompleted)
```

The `syncReplied` scenario and the `syncCompletion` query (`NexusCaller.scala:484-486`, `:542`):

```scala
val syncReplied: Scenario[ProtocolState] =
  scenario("syncReplied")(nexusProtocol) starts Phase.unscheduled actions (
    schedule(unset, unset, unset), handlerReply(Reply.syncSuccess))

val syncCompletion = query("syncCompletion") find syncSucceeds in syncReplied limits two
```

The `nexusCaller` compose block (`NexusCaller.scala:613-620`):

```scala
object nexusCaller extends Compose[NexusCallerState]("nexusCaller"):
  val operation = member("operation", nexusProtocol)(_.operation)
  val worker = member("worker", handlerWorker)(_.worker)
  sync(workerStop, operation(workerStop) || worker(workerStop))
  sync(handlerReply, operation(handlerReply) || worker(serve))
  starts(operation at Phase.unscheduled, worker at Worker.Phase.polling)
  ends(operation at Phase.succeeded, operation at Phase.failed, operation at Phase.canceled,
    operation at Phase.timedOut)
```

## Line counts

| File | Lines |
| --- | --- |
| README.md | 227 |
| NexusCaller.scala | 638 |
| StandaloneActivity.scala | 641 |
| Umpire.scala | 540 |
| Pins.scala | 264 |
| Worker.scala | 77 |
| Total | 2387 |
| Two Model files alone | 1279 |

## Red flags

- **The three verify Queries do not type-check under the sample's own framework.** `QueryIn[S].in(s: Scenario[S])` (`Umpire.scala:391`) fixes one state type, yet `terminalHolds` pairs `Property[ProductState]` with `Scenario[ProtocolState]` (`NexusCaller.scala:551`), and `terminalHolds` and `pauseHolds` do the same in `StandaloneActivity.scala:566-567`. The README advertises exactly this rejection as a feature (`README.md:31-32`, `:150-151`). The framework has no path for reading a product Property through the refinement map, which is the whole point of those Queries.
- **A pin contradicts the framework's `starts` semantics.** `Pins.scala:104` asserts `nexusProtocol.starts == List(at(Phase.unscheduled))`, one state. But `starts(Phase.unscheduled)` expands over every other field (`Umpire.scala:297-298`), giving 24 states, which is what the `ends.size == 96` pin two lines earlier relies on. This test would fail.
- **`Phased`-typed APIs cannot accept a phase value as written.** `starts`, `ends`, `ScenarioDecl.starts` and `Member.at` take `Phased[S]#Phase` (`Umpire.scala:297`, `:299`, `:352`, `:516`). The projection is an abstract type member unconnected to the given instance, so `ProductPhase.scheduled` cannot be shown to inhabit it. The `Phased.Aux` alias defined at `Umpire.scala:125` and never used is the shape of the fix. The README's "compile-time `phase` field check" (`README.md:47`) depends on this API.
- **`derives Finite` would fail on every enum as sketched.** `summonAll` (`Umpire.scala:63-70`) summons `Finite[Timeout.unset.type]` for each singleton case, and no given exists for those types, so the `error("case ... carries a field that is not Finite")` branch fires. The README says `derives Finite` "is all the plumbing a new type needs" (`README.md:145-146`).
- **`evidence:` lambdas have no inferable parameter type.** `evidence[S,O,F](lines: F => Evidence)(using m: MachineScope[S,O,F])` (`Umpire.scala:306`) needs `F` before the pattern-matching lambda is typed, but `F` only comes from the trailing using clause. The same shape at `Umpire.scala:283` is fine because the block is a context function; the fix is moving the using clause first.
- **`Entity` overloads a `val` with a `def` of the same name** twice (`Umpire.scala:146-148`), which Scala rejects as a double definition.
- **The README assumes `-Werror`** (`README.md:97`) while the DSL depends on `given Conversion` instances (`Umpire.scala:186`, `:525`) used without `import scala.language.implicitConversions`, which is a feature warning at every use site and therefore an error under that flag.
- **`Compose` records nothing.** `sync`, `starts` and `ends` are `= ()` (`Umpire.scala:535-537`), so the composed `machine` at `Umpire.scala:540` has no inputs to build from. Allowed by SPEC as a sketch, but it means the composition check story is entirely unimplemented, not merely elided.

## Strengths

- **Step functions are the closest to Lean of any non-Lean approach:** `if` guard, exhaustive `match`, `copy`, list literals, and the compiler's space engine decomposes `handlerError(true)` versus `handlerError(false)` for exhaustivity (`NexusCaller.scala:292-305`).
- **The finite-enumeration story is real, not hand-waved.** `Finite.sizeOf` (`Umpire.scala:74-90`) correctly treats singleton enum cases as products of zero fields, `Bounded[N]` as `N+1`, and parametrised cases as products, so `pinStates[ProtocolState](192)` folding to a literal is believable.
- **The macro limitation is stated correctly and designed around.** "A macro cannot evaluate code from its own compilation run" (`README.md:55`) is a true Scala 3 constraint, the two-module sbt layout is the standard workaround, and `Query.pinned` insists on a stable path with a good error message (`Umpire.scala:405-413`).
- **Error examples are realistic.** The E006, E172 and E029 messages in `README.md:65-95` match Scala 3's actual diagnostic format, including the `TupledFunction` shape and the "It would fail on pattern case" wording.
- **Honest cost accounting.** The README names the JVM cold start, the maintainer-facing feature surface, and that Scala is not in Temporal's stack (`README.md:168-180`), and the "awkward" list is longer than the "easy" list.

## One-paragraph verdict

This sample shows that Scala 3 can host the Umpire model layer with step functions that are almost transliterated Lean and with declarations (`property ... when ... holds`, `query ... find ... in ... limits`) that read as configuration while remaining ordinary method calls the IDE understands. Enumeration, exhaustivity, arity and bounded literals are genuinely compile-time; the bounded search is compile-time only across a module boundary, and the README says so plainly. The single biggest reservation is that the framework's central typing claim is contradicted by the sample itself: indexing Property, Scenario and Query by state type makes the three `verify` Queries that read a product Property on the protocol machine ill-typed, and several other surface signatures (`Phased` projections, `evidence` inference, `derives Finite` for enums, the `Entity` field/method clash) would not compile as sketched. None of these is a limit of the language, but together they mean the "what the compiler decides" table in the README overstates what this particular design delivers, and a real `Umpire.scala` would need a maintainer fluent in inline, `Mirror`, match types and quotes, which the README itself identifies as the hiring cost.
