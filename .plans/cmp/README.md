# Umpire model layer: the same two Models in ten languages

Each directory holds the Nexus caller Model and a new standalone activity Model written the way a
fluent author would write them in that language, against the shared [SPEC.md](SPEC.md). The Lean
directory holds the real files from `temporal/model` plus a new standalone activity Model in the
same DSL. Everything else is illustrative: written to show the authoring surface, not compiled.
Each README says where its checks run, what an author's mistake looks like, and which maintained
libraries would cut the work. [QUINT_VS_FIZZBEE.md](QUINT_VS_FIZZBEE.md) compares the two
off-the-shelf specification tools.

Each directory also has a `WALKTHROUGH.md` that explains the standalone activity Model from
scratch with inlined code, in the same seventeen sections for every language, so they can be read
side by side.

| Directory | Lines (without README) | DSL mechanism |
|---|---|---|
| `lean/` | 2,068 | custom syntax categories and command elaborators |
| `go/` | 2,857 | typed struct literals, generics, `Bind` helpers, linters for exhaustiveness |
| `kotlin/` | 2,484 | type-safe builders, lambdas with receiver, `infix`, sealed classes |
| `scala/` | 2,143 | builder blocks with context functions, `inline` + `compiletime.error`, `derives`, a `query` macro |
| `rust/` | 3,710 | `machine! { }` proc macros over `syn`/`quote`, `#[derive(Finite)]`, enums with payloads |
| `nim/` | 2,383 | macros over Nim's own colon-block syntax, `static:` compile-time evaluation |
| `racket/` | 2,235 | `syntax-parse` macros, optionally a `#lang umpire` |
| `typescript/` | 2,589 | object literals with `satisfies`, literal types tying declarations together |
| `julia/` | 2,658 | hygienic `@machine` block macros, Moshi sum types, multiple dispatch |
| `quint/` | 2,081 | the language itself: modules, `var`, `action`, `run`, invariants |

Line counts are not effort estimates. Each sample includes a sketched framework file of different
depth. The Lean count is the smallest because its framework is not in the directory.

## The product machine, side by side

The same declaration in each language. Step functions are ordinary functions in every sample;
what differs is how the declarative part is written and what the compiler can say about it.

Lean (real):

```lean
machine nexusProduct
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  timers: [timeout]
  evidence:
    nexusOperationStarted: nexusOperationStarted
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
```

Nim:

```nim
machine nexusProduct:
  `for`: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  timers: [timeout]
  evidence:
    nexusOperationStarted: nexusOperationStarted
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
```

Rust:

```rust
machine! { nexusProduct
    for: operation
    state: ProductState
    outcome: ProductOutcome
    facts: ProductFact
    starts: [Scheduled]
    ends: [Succeeded, Failed, Canceled, TimedOut]
    timers: [timeout]
    steps: { handlerReply: handler_reply_step, complete: complete_step }
}
```

Racket:

```racket
(machine nexusProduct
  #:for operation
  #:state ProductState
  #:starts (scheduled)
  #:ends (succeeded failed canceled timedOut)
  #:timers (timeout)
  #:steps ([handlerReply handlerReplyStep] [complete completeStep]))
```

Scala 3:

```scala
val nexusProduct = machine[ProductState, ProductOutcome, ProductFact]("nexusProduct"):
  forEntity(operation)
  starts(ProductPhase.scheduled)
  ends(ProductPhase.succeeded, ProductPhase.failed, ProductPhase.canceled, ProductPhase.timedOut)
  timers(timeout)
  steps(handlerReply ~> handlerReplyStep, complete ~> completeStep)
```

Kotlin:

```kotlin
val nexusProduct = machine<ProductState, ProductOutcome, ProductFact>("nexusProduct") {
    entity = operation
    starts(ProductState(ProductPhase.Scheduled))
    ends { productTerminal(this) }
    timers(timeout)
    steps {
        handlerReply runs ::handlerReplyStep
        complete runs ::completeStep
    }
}
```

Julia:

```julia
@machine nexusProduct begin
    var"for" = operation
    state = ProductState
    starts = [scheduled]
    ends = [succeeded, failed, canceled, timedOut]
    timers = [timeout]
    steps = (handlerReply = handlerReplyStep, complete = completeStep)
end
```

TypeScript:

```ts
export const nexusProduct = defineMachine({
  name: "nexusProduct",
  for: operation,
  state: ProductState,
  starts: ["scheduled"],
  ends: ["succeeded", "failed", "canceled", "timedOut"],
  actions: [handlerReply, complete, transportFault, workerStop],
  timers: ["timeout"],
  steps: { handlerReply: handlerReplyStep, complete: completeStep },
});
```

Go:

```go
var NexusProduct = &umpire.Machine[ProductState, ProductOutcome, ProductFact]{
	Name:   "nexusProduct",
	For:    operation,
	States: umpire.Fields[ProductState](),
	Starts: []ProductState{{Phase: ProductScheduled}},
	Ends:   productTerminal,
	Timers: []umpire.Action{timeout},
	Steps: umpire.Steps(
		umpire.Bind1(handlerReply, handlerReplyStep),
		umpire.Bind1(complete, completeStep),
	),
}
```

Quint has no machine declaration. A module holds `var phase`, one `action` per action class, and a
`step` that nondeterministically picks among them; the table is what the simulator explores.

## What each buys and costs

| | Compile-time checks | Error locality | Edit loop | Team fit | Ecosystem to lean on | Risk |
|---|---|---|---|---|---|---|
| Lean | everything, including refinement proofs and query results | excellent, curated | 3 to 5 minutes per edit today | poor | Batteries, Veil, one protobuf port | maintainer scarcity, slow loop |
| Go | types only; exhaustiveness via `exhaustive` and `gochecksumtype` linters | good | seconds | native | rapid (stateful PBT), protobuf-go, existing Testpilot runtime | verbose declarations, all semantics at test time |
| Kotlin | types, sealed exhaustiveness; semantics when builders run, or via KSP/K2 plugin later | good | seconds to a minute | good | kotest, jqwik `ActionChain`, grpc-kotlin, Java SDK, KSP | JVM and Gradle in CI |
| Scala 3 | types plus semantic checks via `inline`/macros, including a compile-time query | excellent | tens of seconds; macros slow it | medium | ScalaCheck `Commands`, ScalaPB, munit, jsoniter | language breadth, compile times |
| Rust | types, exhaustive match, macro diagnostics with spans | excellent | tens of seconds to minutes | medium | proptest-state-machine, prost, syn/quote, trybuild, insta | proc-macro upkeep; stateright is unmaintained (last push 2025-07) |
| Nim | types, exhaustive `case`, `static:` table checks | good | seconds | poor | stdlib macros, status-im protobuf, one young gRPC | niche, small ecosystem |
| Racket | whatever the macros check at expansion | excellent | seconds | very poor | syntax-parse, Rosette, Redex; PBT libs unmaintained | Lisp for a Go team |
| TypeScript | literal-type wiring between declarations, `never` exhaustiveness | good | seconds | very good | fast-check `modelRun`, protobuf-es, TS SDK | no runtime sum types, type-level upkeep |
| Julia | none static without JET; macro-time errors | fair | seconds after precompile | poor | Moshi sum types, JET, ProtoBuf.jl; MLStyle unmaintained | dynamic typing, thin gRPC |
| Quint | types and effect system; invariants by simulator or Apalache | good | seconds | medium | Apalache, ITF traces, quint-connect (Rust) | small team, Go ITF replayer is yours to write |

## What the system-level vision changes

The vision as restated in the discussion is larger than the two Models here: a model of the whole
system that grows gradually like gradual typing, never complete, with explicit gaps; tasks as the
connective tissue between components; two levels, feature behavior and integration path; scoping
to a subgraph such as "everything that reaches matching" across workflow, activity and Nexus task
origins; fault injection points derived from the path; agents as the main authors and humans as
the main readers; and a source of truth that can be checked natively or exported to another
checker. That puts five requirements on the DSL that the two-Model comparison does not exercise:

1. **Partiality as a first-class value.** A component, edge or transition can be declared
   `unspecified` and the checker reports it as a gap rather than an error, while anything that
   depends on it is marked unverified. Known Gaps in the current model are the seed of this.
2. **A graph with nested graphs.** Components and the task edges between them, with a feature
   Model attaching to nodes and edges. Scoping is a subgraph selection. Fault points are edges.
3. **Two-level refinement.** The feature machine refines onto the integration path the way the
   protocol machine refines onto the product machine today, so the same mechanism carries over.
4. **Compilation to backends.** The same model must feed a native bounded search for small scopes
   and an export to Quint, TLA+ or FizzBee for larger ones. That argues for a model represented as
   data with a small typed core, not as arbitrary host-language functions.
5. **Agent authorship.** Regular, minimal syntax, exhaustive diagnostics, and machine-readable
   error output matter more than human conveniences.

Against those, the field narrows:

- **Go** can hold the graph and the data model and already owns the runtime, but every semantic
  check lands at test time and declarations stay verbose. It is the right home for the core data
  model and the compilers to backends regardless of what authors write in.
- **Kotlin** gives the most readable authored surface and the graph fits builders well, but
  requirement 1 and 3 need semantic checks the builder runs only when executed.
- **Scala 3** is the one mainstream language where requirements 1, 3 and 5 can be checked at
  compile time with token-pinned errors without writing a parser. It costs the team a harder
  language.
- **Rust** matches Scala on checks through proc macros and beats it on tooling health, at the
  price of ownership noise in declarative code and macro maintenance.
- **Quint** already is a typed, regular, agent-friendly source of truth with export paths, but
  has no notion of a component graph, realizations or gaps; those would be conventions in modules
  plus tooling around ITF traces.
- **Lean** satisfies every check requirement and fails the loop and staffing requirements.

The shape that fits the vision best is a small external model format owned by Go, with the graph,
machines, gaps and scopes as data, one authoring front end chosen for the team, and compilers from
that data to the native search and to Quint or TLA+. The front end is then a thin layer and can be
changed; the data model is the source of truth. Of the front ends here, Kotlin is the pragmatic
pick for humans and agents alike, Scala 3 if compile-time semantic errors are non-negotiable, and
Quint directly if the team is willing to give up realizations in the spec.

## Answers to questions raised during the comparison

**Apalache.** A symbolic bounded model checker for TLA+ and Quint, JVM plus Z3, actively released
(v0.62 in 2026, needs Java 21). It earns its keep on unbounded or large data domains: integers,
sets, maps, many instances. The Models here are finite enums with one to seven reachable product
states per query, where explicit enumeration is instant and complete. Apalache is not worth
adopting on its own. It becomes relevant only if you export to Quint and want `quint verify` for
inductive invariants over a system-level model that stops being tiny.

**Scala 3 compile times.** Roughly on par with Scala 2.13 and much better than 2.12, per the
Scala Center's own reporting; some projects compile a little faster on 3, some a little slower.
Pipelined builds since 3.5 cut multi-module builds by a reported 10 to 30 percent. Heavy `inline`
and macro use adds time. Expect several times slower than Kotlin and an order of magnitude slower
than Go for the same amount of code, and nothing like Lean's minutes per file.

**GraalVM native image.** It changes the runtime story, not the developer loop. A Kotlin or
Scala CLI compiled with native-image starts in milliseconds and uses tens of megabytes, which makes
tools like `umpire-inspect` feel like Go binaries. The build itself takes minutes, reflection and
resources need configuration, and `kotlinx.serialization` or ScalaPB work with it while
reflection-heavy libraries need care. Day-to-day development still runs on the JVM with JIT
warm-up, and Gradle or sbt are still in the loop.

**Stateright.** Last push July 2025, so over a year idle. Treat it as unmaintained and do not
build on it. In Rust, proptest-state-machine is the maintained alternative for randomized
stateful testing, and the exhaustive table stays hand-written in every language.

## Caveats

The first version of the Model 2 spec had a refinement that could not hold: three protocol rows
mapped to product transitions the product did not have. Four of the writers caught it
independently while implementing it, which is a small argument for the value of a refinement
check in whatever language wins. SPEC.md carries the fix and a revision note; each sample was
asked to apply it, and its README says whether it did.

The nine non-Lean samples were written by parallel agents from the same spec and reviewed for
shape, not compiled. Several READMEs record where the spec's names fought the language: Nim's
scoping of `for` and `set`, Kotlin's and Scala's fidelity notes, Julia's `var"for"`. Treat the
snippets as what the authoring surface would look like, and the READMEs as the honest part.
