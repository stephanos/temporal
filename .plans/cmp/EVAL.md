# Evaluation of the nine samples

Independent reviewers, none of whom wrote a sample, scored each directory against one rubric
(`.eval-rubric.md`): spec fidelity, language plausibility, authoring readability, accuracy of the
compile-time versus test-time story, framework realism, and README honesty with two or more
library claims re-checked against GitHub. Their full reports are in `.eval/<lang>.md`. This file
is the synthesis. Scores are 1 to 5, 5 best. Reviewers compiled nothing; "plausibility" means a
fluent reader's judgement, and several reviewers found code that could not compile in principle.
Two samples were compiled afterwards; see "Compile trials" below.

> **Update after review.** While writing their walkthroughs, most writers edited their samples to
> address findings from their reviews. By file timestamps, six samples changed after they were
> reviewed: Julia, Kotlin, Quint, Racket, Rust and Scala. Go is unchanged. Each walkthrough's last
> section says what was fixed and what remains. The scores and red flags below describe the samples
> as reviewed, not as they are now. The Lean fixes are described under its verdict. Nothing was
> recompiled or re-reviewed.

> **Added.** Elixir joined on 2026-09-30, written against the spec with all three revision notes
> and reviewed with the same rubric. It was then compiled and run; see "Compile trials".

> **Dropped.** Nim and TypeScript were removed from the comparison on 2026-09-30. Their reviews
> are in git history.

## Scores

| Sample | Fidelity | Plausibility | Readability | Check story | Framework | README | Mean |
|---|---|---|---|---|---|---|---|
| Lean | 3 | 3 | 5 | 4 | 5 | 4 | 4.0 |
| Go | 4 | 5 | 3 | 5 | 4 | 5 | 4.3 |
| Kotlin | 5 | 4 | 4 | 3 | 3 | 5 | 4.0 |
| Scala 3 | 5 | 3 | 4 | 3 | 4 | 5 | 4.0 |
| Rust | 4 | 4 | 4 | 4 | 4 | 5 | 4.2 |
| Racket | 4 | 4 | 3 | 4 | 4 | 5 | 4.0 |
| Julia | 4 | 4 | 4 | 4 | 4 | 5 | 4.2 |
| Quint | 4 | 3 | 3 | 4 | 4 | 4 | 3.7 |
| Elixir | 5 | 4 | 4 | 4 | 5 | 5 | 4.5 |

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
| Elixir | 1,467 | 70% | 2,829 |
| Kotlin | 1,491 | 72% | 747 |
| Quint | 1,514 | 75% | 294 |
| Julia | 1,559 | 83% | 784 |
| Rust | 1,678 | 75% | 1,632 |
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

Elixir
```elixir
defproperty :syncSucceeds,
  machine: :nexusProtocol,
  when: handlerReply(:syncSuccess),
  holds: fn step -> step.state.phase == :succeeded and :nexusOperationCompleted in step.facts end

defquery :syncCompletion, find: :syncSucceeds, in: :syncReplied, limits: :two
```

Quint
```quint
val syncSucceeds = claim(fired(HandlerReply(SyncSuccess)),
  state.phase == Succeeded and recorded(NexusOperationCompleted))

run syncCompletion = syncReplied.expect(found(syncSucceeds))
```

Reading these cold: Lean, Rust and Racket keep the keyword-per-line shape. Scala and Kotlin
read as fluent sentences. Julia is Lean with `=` and two stropped keywords. Elixir reads as keyword-list configuration in
the Ecto style, with spec names kept verbatim as atoms. Go repeats every name as a string and
wraps every classed action in a constructor. Quint is the
shortest because it is a different model: a property is a predicate over the last recorded step
and a query is a run with an expectation, so the machine, scenario and limits vocabulary
partly dissolves.

## Cross-cutting findings

**Every sample is semantically faithful.** Every reviewer walked the step functions row by row
against the spec and found no drift in enablement, targets or facts, including the revised Model 2
arms. Names match the spec exactly in Lean, Kotlin and Scala; Go, Rust, Julia,
Racket and Quint deviate only by forced casing or one collision (Racket's `handlerError` query,
Julia's `nexusCaller.machine`). This is the part the samples were for, and it held.

**Every framework sketch has compile errors a fluent expert would catch.** None of the seven
non-Lean samples would build as written. The failures cluster in the same place: the typing of
the refinement and of the product-property-on-protocol-scenario queries.

- Scala and Kotlin index Property, Scenario and Query by state type, which makes the spec's
  `verify` queries that read a product claim on a protocol path ill-typed. Scala's README
  advertises exactly that rejection as a feature.
- Kotlin's public `inline` functions reach `internal` declarations, and its `Refinement` is
  immutable but built empty.
- Racket's step-function contracts use a rest arity that rejects every fixed-arity step.
- Julia's `ending` and `starting` compare enum values to symbols.
- Rust's derive references a trait that does not exist and calls `.span()` without the import.
- Quint has three QNT101 name collisions, one being the module-global constructor rule its own
  README explains.
- Go is the exception: the reviewer found no construct that would not compile, and the only
  missing piece is a code generator the `go:generate` lines point at.
- Elixir, written after the others with their reviews in hand, is the second exception. Its
  reviewer found no construct that could not compile, and it did compile on the first attempt
  (see below).

**The refinement rule itself was the hardest thing to get right.** The spec's first Model 2 had
three rows with no product counterpart; four writers caught it. The Lean reviewer then found that
the real Lean checker is stricter than the spec's stated rule: a matching product row must also
have the same outcome and its facts must appear among the protocol row's facts. Under that rule
the Lean sample needed one more fact on the visible-retry row. SPEC.md carries both notes. The
lesson is not about any language. It is that the refinement check is the one mechanism that
found real mistakes twice in one afternoon, in a spec written carefully by someone who had just
read the original, and it should survive any rewrite.

**Search is unwritten everywhere except Lean.** Every sample elides bounded search with a comment
or a `todo`. Quint has no search at all and says so;
its `run` blocks are the scenario, and `quint run` is the random exploration. Composition tables
are unwritten in Kotlin, Scala, Racket, Julia and Rust. This is consistent with the brief,
which asked for surface not engine, but it means the pins that assert query answers are
assertions about code that does not exist. Lean's `Pins.lean` runs against a real engine for Model 1. Elixir is the one other sample whose
search is written: a scenario-guided walk with a node budget. Its pins, including every query,
pass when run.

**The compile-time stories are overstated in about half the READMEs.** Kotlin claims member
binding is a type error when it is a runtime check. Scala claims cross-machine query pairing is
rejected when its own verify queries would be rejected instead. Racket lists an expansion-time check
that is stubbed to true. Go, Rust, Julia, Quint and Elixir READMEs were found accurate, with one wrong rustc error
code, one stale backend default, and Elixir's claim that the expression subset rejects
rebinding, which nothing implements. The reviewers' spot checks of library maintenance dates
matched GitHub in every case across all samples, with one network failure.

**Name collisions are the recurring cost of the flat vocabulary.** The spec reuses `scheduled`,
`completed`, `canceled`, `timeout`, `polling` across enums, facts, actions and machines. Lean's
namespaces absorb this. Quint fails to parse on three. Elixir sidesteps the problem with a separate registry per kind of
declaration, so `:completed` can be a phase, a result and a scenario at once. Racket, Scala and Julia each renamed or
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
- **Racket.** The framework literally is a language, with binding-based expansion-time checks and
  an honest `#lang umpire`. Reservation: everything below the macros is dynamic typing plus
  hand-written contracts, and the sample's own contracts would reject every step function at load.
  Lisp accessor chains in properties are the second cost.
- **Julia.** A declarative surface within a few characters of Lean using nothing exotic, with a
  fully written DSL parser and line-pinned errors. Reservation: the "compile time" is cached
  execution at precompile, not analysis; the sketch would not survive that execution; and a team
  would write the whole runtime, pay tens of seconds of cold start per CI job, and carry two
  sum-type packages.
- **Quint.** Fits unusually well where it matters: machines, tables, refinement and scenarios are
  short, real, and checkable with existing tooling; the refinement predicate doubles as a
  `quint verify` invariant. Reservation: sets, limits, schemas, parties and realizations are
  strings for a Go adapter, there is no search, and the files would not pass `quint parse`.
- **Elixir.** Declarations read as configuration and step functions stay ordinary `case`
  expressions over tuples with guards. One `defmachine` emits both an IR, where guards and updates
  are data, and real Elixir functions the 1.20 type checker reads; a pin compares the two on every
  row. The macros, not the type checker, enforce finite domains, exhaustiveness, dead arms, the
  strict Lean refinement and non-vacuous verify queries, and the README credits them correctly.
  Reservation: all of that rests on about 2,800 lines of hand-written macro framework with two
  implementations of pattern matching, one the BEAM's and one the evaluator's, which a Go team
  would own. The rebinding gap already shows where they can drift.

## What the exercise says about the decision

Three things stand out once the reports are read together.

1. **Readability is not where the languages differ most.** Nearly every sample produces a
   declarative block a reader would accept as configuration. The one that does not, Go, is the
   most familiar to the team. The choice is between a familiar language
   with visible ceremony and an unfamiliar one with less.
2. **Static semantic checking is mostly a promise.** Outside Lean, the samples that claim
   compile-time semantics (Scala, Racket, Rust, Kotlin) each have the claim undercut by their
   own code. That is not evidence the languages cannot do it. It is evidence that doing it is
   framework work that needs a specialist, which is the same staffing problem Lean has in a
   milder form. Go makes no such claim and is honest about running everything in tests. Elixir
   is the one counterexample: its compile-time checks were demonstrated by planted mistakes
   (below). It shows the tier is buildable outside Lean, and also what it costs: the checks live
   in a framework the team writes and maintains.
3. **The refinement check earned its place.** It found a real spec bug through four writers, and
   then a second, subtler one through the Lean reviewer reading the real checker. Whatever language
   is chosen, the feature-to-integration layering the vision describes should be checked by this
   mechanism, run as a test if not at compile time, and pinned by row as the Lean tests do.

## Compile trials

After review, two samples were compiled in scratch copies under `/tmp`, with toolchains installed
through mise. The committed samples were not edited by these trials.

**Scala 3.** Scala 3.9.0 with scala-cli 1.17.1. The main sources failed with 41 errors from three
framework bugs: a missing `scala.compiletime.ops.int` import, a `Phased` derivation that computed
the phase field's index as a value so no given ever matched (40 of the 41 errors), and
`TupledFunction` needing `-experimental`. With those fixed, the main sources built. The test
scope still fails with 21 errors: the `Query.pinned` macro rejects top-level queries, and the
compile-time search it would run is unwritten, so no search was timed.

| Scala | Time |
|---|---|
| Cold build, compile server stopped | 9.7 s |
| Clean build, warm server | 1.6 to 2.0 s |
| No-op recompile | 0.3 to 0.4 s |
| One-line model edit | about 1.0 s |
| Semantic model edit | 1.4 to 1.6 s |

**Elixir.** Elixir 1.20.4 on OTP 28, installed as precompiled binaries in 19 seconds. It compiled
on the first attempt in a Mix project with a one-line `mix.exs`, with one type warning from the
1.20 checker (`min` over possible structs in `check.ex`). All 21 pins passed, in 0.4 seconds,
which independently confirms the spec's counts, both refinements under the strict rule, every
query, and the vacuity rule. The IR export task crashed on tuple action classes such as
`{:handlerError, false}`.

Planted mistakes and where they were caught:

| Mistake | Caught | Report |
|---|---|---|
| One match arm dropped | `mix compile`, at the step | names the missing `{:pauseRequested, :completed}` |
| Fact misspelled | `mix compile`, at the line | "records `:statusCompletd`, which is not a member of the machine's facts" |
| Strict refinement broken | `mix compile`, at the machine | names the protocol row, its mapped states, and why no product row matches |
| Retry claim unfindable | `mix test` | the full six-step trace searched, with states |

| Elixir | Time |
|---|---|
| Clean build | 7 to 11 s |
| No-op recompile | 0.5 s |
| One model file edited | 5.1 to 5.7 s |
| Warm test run | about 1 s |

After the trials the writer fixed what they and the review found, and a fresh copy of the committed
sample was verified: `mix compile --warnings-as-errors` is clean, all 21 pins pass, `mix umpire.ir`
writes one valid JSON IR file per Model, and a planted `{p, p}` pattern is now rejected at compile
time ("p is bound twice in one pattern; use a guard to compare"). The sample ships its `mix.exs`
and builds as-is. The review's rebinding finding and the IR export crash are therefore fixed.

The Elixir edit costs five seconds because the macros build the table and check coverage and
refinement at compile time; Scala's second costs no such work, since its search and table build
are sketched. For comparison, a one-line edit to the Lean caller Model re-elaborates in 3 min 23 s.

## Methodology and limits

Samples and reviews were produced by agents from the same spec. Reviewers had the spec, the real
Lean files and the sample; they did not run toolchains. The compile trials above were run
separately, after review. Their library spot checks used the
GitHub API on 2026-09-29. Scores are one reviewer's judgement per sample and were not calibrated
across reviewers, so a 4 in one row is not a 4 in another. Line counts include comments carried
over from the Lean file, so ceremony is better read from the snippets than from the totals.
