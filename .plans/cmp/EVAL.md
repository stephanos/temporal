# Evaluation of the ten samples

Ten independent reviewers, none of whom wrote a sample, scored each directory against one rubric
(`.eval-rubric.md`): spec fidelity, language plausibility, authoring readability, accuracy of the
compile-time versus test-time story, framework realism, and README honesty with two or more
library claims re-checked against GitHub. Their full reports are in `.eval/<lang>.md`. This file
is the synthesis. Scores are 1 to 5, 5 best. Nothing was compiled; "plausibility" means a fluent
reader's judgement, and several reviewers found code that could not compile in principle.

> **Update after review.** While writing their walkthroughs, most writers edited their samples to
> address findings from their reviews. By file timestamps, seven samples changed after they were
> reviewed: Julia, Kotlin, Quint, Racket, Rust, Scala and TypeScript. Go and Nim are unchanged;
> Nim's walkthrough notes its findings beside the quoted code instead. Each walkthrough's last
> section says what was fixed and what remains. The scores and red flags below describe the samples
> as reviewed, not as they are now. The Lean fixes are described under its verdict. Nothing was
> recompiled or re-reviewed.

## Scores

| Sample | Fidelity | Plausibility | Readability | Check story | Framework | README | Mean |
|---|---|---|---|---|---|---|---|
| Lean | 3 | 3 | 5 | 4 | 5 | 4 | 4.0 |
| Go | 4 | 5 | 3 | 5 | 4 | 5 | 4.3 |
| Kotlin | 5 | 4 | 4 | 3 | 3 | 5 | 4.0 |
| Scala 3 | 5 | 3 | 4 | 3 | 4 | 5 | 4.0 |
| Rust | 4 | 4 | 4 | 4 | 4 | 5 | 4.2 |
| Nim | 5 | 4 | 4 | 3 | 4 | 4 | 4.0 |
| Racket | 4 | 4 | 3 | 4 | 4 | 5 | 4.0 |
| TypeScript | 5 | 4 | 3 | 3 | 4 | 5 | 4.0 |
| Julia | 4 | 4 | 4 | 4 | 4 | 5 | 4.2 |
| Quint | 4 | 3 | 3 | 4 | 4 | 4 | 3.7 |

The means are close and should not be read as a ranking. The spread inside each row is the
signal: Lean scores highest on readability and framework because the framework is the real 3,900
line elaborator, and lowest on fidelity and plausibility because its new file would not elaborate.
Go is the mirror image: nothing in it is implausible and every claim about checks is accurate, and
it is the least pleasant to read.

## Sizes

Two Model files per sample, Nexus plus standalone activity, counted by the reviewers.

| Sample | Model lines | Code share of Model files | Framework sketch lines |
|---|---|---|---|
| Lean | 1,505 | 77% | 0 in directory, 3,920 real |
| Racket | 1,237 | 46% | 737 |
| Scala 3 | 1,279 | 66% | 540 |
| Kotlin | 1,491 | 72% | 747 |
| Nim | 1,496 | 76% | 676 |
| Quint | 1,514 | 75% | 294 |
| Julia | 1,559 | 83% | 784 |
| Rust | 1,678 | 75% | 1,632 |
| TypeScript | 1,683 | 82% | 619 |
| Go | 1,967 | 78% | 557 |

Code share is code lines over code plus comment lines, so a lower number means more prose per
line of model; Racket's is low because Lisp is dense, not because it is over-commented. Every
sample kept the Lean comments, so the differences in line count are ceremony. Go's Models are
roughly twice the Lean reference, Scala's and Racket's are shorter than it.

## The same four declarations, side by side

The step function is ordinary code everywhere and differs only in match syntax. The declarative
lines are where the languages separate. The `syncSucceeds` property and the `syncCompletion`
query, trimmed:

Lean
```lean
property syncSucceeds
  machine: nexusProtocol
  when: handlerReply (syncSuccess)
  holds: fun step => step.state.phase == .succeeded && step.facts.contains .nexusOperationCompleted

query syncCompletion
  find: syncSucceeds
  in: syncReplied
  limits: two
```

Nim
```nim
property syncSucceeds:
  machine: nexusProtocol
  `when`: handlerReply(syncSuccess)
  holds: step => step.state.phase == succeeded and nexusOperationCompleted in step.facts

query syncCompletion:
  find: syncSucceeds
  `in`: syncReplied
  limits: two
```

Rust
```rust
property! { syncSucceeds
    machine: nexusProtocol
    when: handlerReply(SyncSuccess)
    holds: |step| step.state.phase == Phase::Succeeded && step.facts.contains(&ProtocolFact::NexusOperationCompleted)
}
query! { syncCompletion find: syncSucceeds in: syncReplied limits: two }
```

Scala 3
```scala
val syncSucceeds: Property[ProtocolState] =
  property("syncSucceeds")(nexusProtocol) when handlerReply(Reply.syncSuccess) holds: step =>
    step.state.phase == Phase.succeeded && step.facts.contains(ProtocolFact.nexusOperationCompleted)

val syncCompletion = query("syncCompletion") find syncSucceeds in syncReplied limits two
```

Kotlin
```kotlin
val syncSucceeds = property("syncSucceeds", nexusProtocol) {
    handlerReply(Reply.SyncSuccess) holds { step ->
        step.state.phase == Phase.Succeeded && ProtocolFact.NexusOperationCompleted in step.facts
    }
}
val syncCompletion = query("syncCompletion") { find(syncSucceeds) on syncReplied within two }
```

Julia
```julia
@property syncSucceeds begin
    machine = nexusProtocol
    when = handlerReply(syncSuccess)
    holds = step -> step.state.phase == Phase.succeeded && ProtocolFact.nexusOperationCompleted in step.facts
end
@query syncCompletion begin
    find = syncSucceeds
    var"in" = syncReplied
    limits = two
end
```

Racket
```racket
(property syncSucceeds
  #:machine nexusProtocol
  #:when [handlerReply 'syncSuccess]
  #:holds (λ (s) (and (eq? (ProtocolState-phase (step-state s)) 'succeeded)
                      (member 'nexusOperationCompleted (step-facts s)))))
(query syncCompletion #:find syncSucceeds #:in syncReplied #:limits two)
```

TypeScript
```ts
export const syncSucceeds = property({
  name: "syncSucceeds",
  machine: nexusProtocol,
  when: handlerReply.of({ reply: { kind: "syncSuccess" } }),
  holds: (step) => step.state.phase === "succeeded" && step.facts.includes("nexusOperationCompleted"),
});
export const syncCompletion = query({ name: "syncCompletion", find: syncSucceeds, in: syncReplied, limits: two });
```

Go
```go
var syncSucceeds = &protocolProperty{
	Name:    "syncSucceeds",
	Machine: NexusProtocol,
	When:    handlerReply.With(SyncSuccess{}),
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == Succeeded && step.Records(NexusOperationCompleted{})
	},
}
syncCompletion = &umpire.Query{Name: "syncCompletion", Find: syncSucceeds, In: syncReplied, Limits: two}
```

Quint
```quint
val syncSucceeds = claim(fired(HandlerReply(SyncSuccess)),
  state.phase == Succeeded and recorded(NexusOperationCompleted))

run syncCompletion = syncReplied.expect(found(syncSucceeds))
```

Reading these cold: Lean, Nim, Rust and Racket keep the keyword-per-line shape. Scala and Kotlin
read as fluent sentences. Julia is Lean with `=` and two stropped keywords. TypeScript and Go
repeat every name as a string and wrap every classed action in a constructor. Quint is the
shortest because it is a different model: a property is a predicate over the last recorded step
and a query is a run with an expectation, so the machine, scenario and limits vocabulary
partly dissolves.

## Cross-cutting findings

**Every sample is semantically faithful.** All ten reviewers walked the step functions row by row
against the spec and found no drift in enablement, targets or facts, including the revised Model 2
arms. Names match the spec exactly in Lean, Kotlin, Scala, Nim, TypeScript; Go, Rust, Julia,
Racket and Quint deviate only by forced casing or one collision (Racket's `handlerError` query,
Julia's `nexusCaller.machine`). This is the part the samples were for, and it held.

**Every framework sketch has compile errors a fluent expert would catch.** None of the nine
non-Lean samples would build as written. The failures cluster in the same place: the typing of
the refinement and of the product-property-on-protocol-scenario queries.

- Scala, Nim and Kotlin index Property, Scenario and Query by state type, which makes the spec's
  `verify` queries that read a product claim on a protocol path ill-typed. Scala's README
  advertises exactly that rejection as a feature.
- TypeScript's `checkRefinement` fails under strict function types, and outcome and fact
  inference through key-remapped mapped types degrades to `string`.
- Kotlin's public `inline` functions reach `internal` declarations, and its `Refinement` is
  immutable but built empty.
- Racket's step-function contracts use a rest arity that rejects every fixed-arity step.
- Julia's `ending` and `starting` compare enum values to symbols.
- Rust's derive references a trait that does not exist and calls `.span()` without the import.
- Quint has three QNT101 name collisions, one being the module-global constructor rule its own
  README explains.
- Go is the exception: the reviewer found no construct that would not compile, and the only
  missing piece is a code generator the `go:generate` lines point at.

**The refinement rule itself was the hardest thing to get right.** The spec's first Model 2 had
three rows with no product counterpart; four writers caught it. The Lean reviewer then found that
the real Lean checker is stricter than the spec's stated rule: a matching product row must also
have the same outcome and its facts must appear among the protocol row's facts. Under that rule
the Lean sample needed one more fact on the visible-retry row. SPEC.md carries both notes. The
lesson is not about any language. It is that the refinement check is the one mechanism that
found real mistakes twice in one afternoon, in a spec written carefully by someone who had just
read the original, and it should survive any rewrite.

**Search is unwritten everywhere except Lean.** Every sample elides bounded search with a comment
or a `todo`. Nim's `search` walks only the scenario path. Quint has no search at all and says so;
its `run` blocks are the scenario, and `quint run` is the random exploration. Composition tables
are unwritten in Kotlin, Scala, Racket, Julia, Rust and Nim. This is consistent with the brief,
which asked for surface not engine, but it means the pins that assert query answers are
assertions about code that does not exist. Only Lean's `Pins.lean` runs against a real engine,
and only for Model 1.

**The compile-time stories are overstated in about half the READMEs.** Kotlin claims member
binding is a type error when it is a runtime check. Scala claims cross-machine query pairing is
rejected when its own verify queries would be rejected instead. Nim claims bare-identifier
classes are validated when the macro discards the lookup. Racket lists an expansion-time check
that is stubbed to true. TypeScript's inference claims depend on mapped-type inference TypeScript
does not do. Go, Rust, Julia and Quint READMEs were found accurate, with one wrong rustc error
code and one stale backend default. The reviewers' spot checks of library maintenance dates
matched GitHub in every case across all ten samples, with one network failure.

**Name collisions are the recurring cost of the flat vocabulary.** The spec reuses `scheduled`,
`completed`, `canceled`, `timeout`, `polling` across enums, facts, actions and machines. Lean's
namespaces absorb this. Nim's reviewer counted about twenty same-module collisions where the
README admitted four. Quint fails to parse on three. Racket, Scala and Julia each renamed or
nested something to escape one. Any non-Lean choice will need a naming convention the Lean file
never needed.

## Per-language verdicts

Each is the reviewer's one-paragraph verdict, compressed to its claim and its reservation.

- **Lean.** The cleanest declarative surface and the only real engine, with finiteness,
  exhaustiveness, refinement and search all decided at compile time. Reservation: "reads like the
  original" is not "elaborates". The new file names a realization that does not exist, uses
  evidence names the catalog would reject, and shipped without Activity pins to catch either,
  which is the failure mode a three-minute loop makes expensive. Fixed after review: the extra
  fact, nine status observations, a note on the missing realization, and `ActivityPins.lean`.
- **Go.** Zero new toolchain, instant compiles, typed protobuf schemas that exceed Lean, and
  action-to-step binding checked by generic inference. Reservation: ceremony. Twice the lines of
  Lean, sum types cost an interface plus structs plus markers plus a linter directive each, and
  every semantic check is a `go test`.
- **Kotlin.** The most readable builder DSL, with configuration-shaped blocks, sealed
  exhaustiveness and typed action arity. Reservation: the framework is thinner than it looks. The
  table, the refinement walk, the search and the composition product are `TODO` or structurally
  impossible as sketched, and two compile-time claims are runtime checks.
- **Scala 3.** Step functions are near-transliterated Lean and the declarations read as sentences;
  enumeration, exhaustivity, arity and bounded literals are genuinely compile time. Reservation:
  the framework's central typing idea contradicts the sample's own verify queries, and several
  signatures do not compile as sketched. A real `Umpire.scala` needs a maintainer fluent in
  `inline`, `Mirror`, match types and quotes.
- **Rust.** The strongest compile-time name-check story outside Lean: every DSL name becomes a
  Rust path emitted at the author's token, so rustc and rust-analyzer are the checker. State
  space sizes pin at compile time via associated consts. Reservation: the proc-macro crate is the
  largest file and the least written, and the README itself estimates six hundred more lines of
  parser. A second toolchain and `syn`-literate maintainers in a Go monorepo.
- **Nim.** The Lean shape almost for free: colon blocks, plain procs, compile-time VM, errors on
  the author's line. Enumeration and refinement are real code. Reservation: the two verify
  queries have no implementation and would not type-check, and same-module name collisions are
  several times more numerous than admitted.
- **Racket.** The framework literally is a language, with binding-based expansion-time checks and
  an honest `#lang umpire`. Reservation: everything below the macros is dynamic typing plus
  hand-written contracts, and the sample's own contracts would reject every step function at load.
  Lisp accessor chains in properties are the second cost.
- **TypeScript.** No macros, no codegen, under-a-second loop, and object-literal builders whose
  `const` generics type each declaration against the ones before it. Reservation: the type-level
  DSL is fragile exactly where it is most ambitious; refinement typing fails on contravariance
  and inference degrades silently. In TypeScript the guarantees are only as good as someone
  running `tsc`, and nobody did.
- **Julia.** A declarative surface within a few characters of Lean using nothing exotic, with a
  fully written DSL parser and line-pinned errors. Reservation: the "compile time" is cached
  execution at precompile, not analysis; the sketch would not survive that execution; and a team
  would write the whole runtime, pay tens of seconds of cold start per CI job, and carry two
  sum-type packages.
- **Quint.** Fits unusually well where it matters: machines, tables, refinement and scenarios are
  short, real, and checkable with existing tooling; the refinement predicate doubles as a
  `quint verify` invariant. Reservation: sets, limits, schemas, parties and realizations are
  strings for a Go adapter, there is no search, and the files would not pass `quint parse`.

## What the exercise says about the decision

Three things stand out once the ten reports are read together.

1. **Readability is not where the languages differ most.** Eight of ten samples produce a
   declarative block a reader would accept as configuration. The ones that do not, Go and
   TypeScript, are the two most familiar to the team. The choice is between a familiar language
   with visible ceremony and an unfamiliar one with less.
2. **Static semantic checking is mostly a promise.** Outside Lean, the samples that claim
   compile-time semantics (Scala, Nim, Racket, Rust, Kotlin) each have the claim undercut by their
   own code. That is not evidence the languages cannot do it. It is evidence that doing it is
   framework work that needs a specialist, which is the same staffing problem Lean has in a
   milder form. Go and TypeScript make no such claim and are honest about running everything in
   tests.
3. **The refinement check earned its place.** It found a real spec bug through four writers, and
   then a second, subtler one through the Lean reviewer reading the real checker. Whatever language
   is chosen, the feature-to-integration layering the vision describes should be checked by this
   mechanism, run as a test if not at compile time, and pinned by row as the Lean tests do.

## Methodology and limits

Samples and reviews were produced by agents from the same spec. Reviewers had the spec, the real
Lean files and the sample; they did not run toolchains. Their library spot checks used the
GitHub API on 2026-09-29. Scores are one reviewer's judgement per sample and were not calibrated
across reviewers, so a 4 in one row is not a 4 in another. Line counts include comments carried
over from the Lean file, so ceremony is better read from the snippets than from the totals.
