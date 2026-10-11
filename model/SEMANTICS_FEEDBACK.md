# Feedback on SEMANTICS.md

A review of `model/SEMANTICS.md` as of commit `2f0ba19b13`. Each point is numbered, has a
severity, describes the problem as a reader of the guide meets it, and recommends one fix.

Severities:

- **High**: the guide is wrong, contradicts itself, or leaves a reader unable to write or read a
  Model correctly.
- **Medium**: the guide is right but costs the reader real effort, or the DSL name it documents
  will keep causing mistakes.
- **Low**: polish; worth doing when the section is next touched.

Line numbers refer to the version reviewed.

---

## Part A. Overall shape

### 1. The guide is three documents

**Severity:** High

**Problem.** The introduction promises one guide "in the order you write it", where each section
says what a concept means, how it is written and what is refused. That holds for sections 1 to 7.
Sections 8 to 19 are a reference: longer, less uniform, written for someone who already knows the
language. Sections 20 and 21 are an architecture note on the tooling and the DSL's layering. A
newcomer reading top to bottom hits the change of register at section 8 without warning, and a
maintainer looking up one rule has to know which of the three documents holds it.

**Recommended solution.** Make the split explicit. Either three files (`SEMANTICS.md` for the
tutorial through the table and claims, `REFERENCE.md` for the composite forms and realizations,
`DESIGN.md` for levels, results, core and sugar) or three top-level parts in one file with the
contents list grouped the same way. Keep the "Meaning / Writing it / Refused" template for the
tutorial part only, and say so in the introduction.

### 2. The running example is abandoned

**Severity:** High

**Problem.** Line 13 promises "one small feature throughout", the job. Sections 1 to 7 use it.
Section 8 already uses `Phase.paused`, which the job never declared. Section 10 switches to
`Active` and `Fact.attemptAdmitted` from the activity Model, section 11 to `ServedDispatch`,
section 12 to `Stage.staged`, section 14 to `Lamp`, section 17 to `RecordOverQueue`, section 18
to `Relay`, section 19 back to `JobRun` with members (`attempts`, `described`, `dispatchHold`)
that appear from nowhere. The job's `Fact` enum also changes shape between section 1
(`submitted, started, finished`) and section 2 (`statusQueued, ...`) without a word. A reader
cannot build the promised mental model because the example keeps changing under them.

**Recommended solution.** Pick one of two: either carry the job through every section, declaring
each new case (`paused`, `unscheduled`, `backingOff`) at the point it is first needed, with a
single consolidated listing at the end that a reader could compile; or drop the promise, and open
each section with "this example is from `model/temporal/features/...`" so the reader knows where
to look. The first is better for a guide; the second is cheaper.

### 3. "Refused" paragraphs are inconsistent and duplicate section 20

**Severity:** Medium

**Problem.** A "Refused" paragraph appears in sections 1, 2, 3, 4, 10, 13 and 17, is inlined as
prose in 5 and 6, and is absent from 7, 8, 9, 11, 12, 14, 15, 16, 18 and 19. Section 20's "Who
refuses what" table then restates most of them. The reader never knows whether a section's
silence means "nothing is refused here" or "see the table".

**Recommended solution.** Make section 20's table the single authoritative list of what each
stage refuses, and keep per-section "Refused" text only where it teaches a rule that is not
obvious from the table (for example, that rule overlap is checked over every state of the
catalog). Every remaining per-section paragraph should link to the table.

### 4. Naming conventions are repeated eight times

**Severity:** Low

**Problem.** "Named after its `val`" is said at lines 135, 162, 174, 446, 514, 642, 934 and 1053.
The authoritative statement, "Naming" at line 1053, sits at the very end, after the reader has
already met every declaration. The three-symbol rule at line 1050 is likewise orientation, placed
last.

**Recommended solution.** Add a short "Conventions" section before section 1 holding the naming
rule, the Definition ID rule, the three symbols and the capitalisation rule (point 40). Delete
every later repetition, keeping only the exceptions ("a name is written as a string only where
no `val` declares it").

---

## Part B. Names in the DSL

### 5. `when` means three things

**Severity:** High

**Problem.** `when(Phase.x)` and `when[Role]` are rule headings (section 6). `property when
worker.finish(Result.ok) holds { ... }` is a Property selector (section 8). `whenAction` is a
third selector form. A reader who has learned `when` as "the guard of a rule" then meets it as
"the class a Property reads" and must re-learn it; a search for `when` in a Model finds both.

**Recommended solution.** Give the Property selector its own word that does not appear in rules,
for example `property.of(worker.finish(Result.ok))` and `property.ofAction(worker.finish)`.
Reserve `when` for rule headings.

### 6. `on` means three things

**Severity:** Medium

**Problem.** `on(client.submit) { ... }` opens a rule (section 6). `action(this).on(job)` says
which entity an action targets (section 3). `Observation(on = job)` says which entity a read is
of. The rule form is by far the most common and should own the word.

**Recommended solution.** Rename the entity target to `.targets(job)` or `.of(job)` and the
observation parameter to `of = job`. Keep `on` for rules only.

### 7. `enter`/`enters` and `reject`/`rejects` differ by one letter

**Severity:** High

**Problem.** `enter(state, facts*)` is a value form used in an effect; `enters(Phase.p)` is a
rule form that also records the phase's status fact. `reject(outcome, s)` is a value form;
`rejects(rejection)` is a rule form over shared outcomes. The pair differ in kind, arity and what
they record, but the names differ by a trailing `s`. This is the kind of distinction readers
misread in review and misremember when writing.

**Recommended solution.** Give the rule forms a visibly different shape, for example
`entering(Phase.p)` and `rejecting(Rejection.notFound)`, or make them methods of a single object
(`steps.enter(Phase.p)`, `steps.reject(r)`). Document the four in one table in section 5 with
"used in an effect" versus "used after `~>`" as a column.

### 8. `disabled` is defined twice and documented inconsistently

**Severity:** High

**Problem.** The sugar table (line 276) says `disabled` is `Nil` and "in an effect it is
refused, in a rule use `disabled(a)`". Line 643 then uses `else disabled` inside an effect (the
hole example) and calls it correct. Section 21 lists `disabled` under sugar twice, once as a rule
form and once as an effect form. `framework/Syntax.scala` defines both `val disabled` and
`def disabled(actions*)`. The reader cannot tell whether an effect may return `disabled`.

**Recommended solution.** Decide, then say it once. If effects may return `Nil`, fix line 276 to
"in an effect, gives no step" and keep the example at line 643. If they may not, fix the example.
Either way rename the rule form so it is not an overload: `unbound(process.stop)` or
`never(process.stop)`.

### 9. `Rules`, `Bindings` and `Facets` are three superclasses for one section

**Severity:** Medium

**Problem.** `object rules extends Rules` (section 6), `object rules extends Bindings(...)`
(section 18, the only place `Bindings` is introduced), and `object rules extends Facets(...)`
(section 15). The section is always called `rules` but its superclass varies with how the rules
were built. Section 21 lists `Bindings` as core and `Rules` as sugar without ever saying that one
lowers to the other.

**Recommended solution.** One base, `Rules`, with named constructors: `Rules` for authored cases,
`Rules.bound(a ~> f, ...)` for direct bindings, `Rules.from(facet.rules, ...)` for assembly. Say in
section 6 that the first two lower to the same IR, and move the `Bindings` form there.

### 10. Four spellings of "derived from"

**Severity:** Medium

**Problem.** `Derived(...)` derives a machine (section 14), `DerivedComposition` and
`withMember` derive a composition (section 17), `DerivesFrom(...)` derives a realization (section
19). They do the same thing: one source, one change, everything else kept.

**Recommended solution.** One word, `Derived(source.change(...))`, for all three, with the
change methods (`rebind`, `restrict`, `withMember`, `inserting`) carried by the source's type.
Section 14's table can then cover compositions and realizations as extra rows.

### 11. `Declares[S]` says nothing

**Severity:** Medium

**Problem.** Section 8 introduces `Declares[S]` as "the supertype of a machine and a
composition" and immediately describes its use as "declares the same Property on whichever
*model* it is called with". The prose has the right name; the type does not.

**Recommended solution.** Rename to `Model[S]`. The word "Model" is already used throughout the
guide for exactly this thing.

### 12. `Phased` versus `Phasing`

**Severity:** Medium

**Problem.** The machine mixin is `Phased[S, P]` (section 4). Capabilities take a given
`Phasing[S, P]` (section 16, line 791) that the guide never introduces, and call
`phasing.roleCases` and `phasing.phase` on it. A reader cannot tell whether these are one thing
or two.

**Recommended solution.** If `Phasing` is the typeclass that `Phased` provides, say so in section
4 ("a `Phased` machine gives a `Phasing[S, P]`") and reference it from section 16. Better, use
one name: make `Phased` the typeclass and let the mixin be `Phased.by(_.phase, writer)`.

### 13. `limits` versus `bound`

**Severity:** Low

**Problem.** `query verify p in s limits three` bounds a Query. `capabilities.bound(three,
Closable.terminalStatesAreFinal -> five)` bounds brought Queries. Same concept, two verbs, and
`bound` is also the past tense the guide uses for "an action that is bound to a rule".

**Recommended solution.** `capabilities.limits(three, ...)`.

### 14. `except` and `overriding` are both "waivers"

**Severity:** Low

**Problem.** Line 818 calls both forms waivers and records them in `.waivers.json`, but the
verbs do not say so. `except` reads as a list filter; `overriding` is a participle beside the
imperative `except`.

**Recommended solution.** `waive(P, because = ...)` and `replace(P -> properties.own, because =
...)`. Both are imperatives, both say what they do, and the file name follows from the first.

### 15. `never(to).from(before)` reads backwards

**Severity:** Medium

**Problem.** The core form is `!before(b) || !to(a)`: the precondition is on the state before,
the prohibition on the step after. The sugar states the prohibition first and the precondition
second, so a reader parses "never X" and then has to revise it to "never X, when coming from Y".
`once(over).keeps(x)` has a related issue: "once" reads as "exactly once", where it means
"from the point `over` holds".

**Recommended solution.** Precondition first: `from(before).never(to)`, and `whenever(over)
.keeps(x)` or `after(over).keeps(x)`. Keep the core-form table so the change is checkable.

### 16. "level" means two things

**Severity:** Medium

**Problem.** Section 13 uses *level* for Product versus Protocol ("read at two levels"), and so
do sections 3, 15 and 16 ("every level's machine", "one machine at one level",
"split by level"). Section 20 uses *level* for the five expression strata (value, reading of a
state, step, reading of a step, claim). Section 5 says a misplaced `Step` "is at the wrong
*level*" and links to section 20, but a reader at section 5 has only met the Product/Protocol
meaning.

**Recommended solution.** Rename section 20's concept. "Place" is already the column header in
its table; "stratum" or "tier" would also do. Keep "level" for Product and Protocol.

### 17. "class" collides with Scala

**Severity:** Medium

**Problem.** "Class" means "one assignment of an action's inputs" and is used in nearly every
section, in a document whose examples are full of `case class` and `final case class`. Phrases
like "a `Recorded` case without a status", "one column per class" and "a case class whose every
field is a Property" sit on the same page.

**Recommended solution.** Rename to "variant" or "call" (`finish(Result.ok)` is a call of
`finish`). If the IR key `name-key(i1)` makes renaming expensive, keep "class" but state the
collision once in the Conventions section: "a *class* of an action is not a Scala class".

### 18. `loss = Loss.reliable` and the unnamed order

**Severity:** Low

**Problem.** `channel[Note](capacity = 2, order = Order.fifo, loss = Loss.reliable,
duplicates = 1)` says the loss is reliable. The prose says "ordered-or-not" but names only
`Order.fifo`; the other value is never shown.

**Recommended solution.** `delivery = Delivery.reliable` / `Delivery.lossy`, and name both order
values in the text (`Order.fifo`, `Order.any`, or whatever the second is).

### 19. `Evidences`

**Severity:** Low

**Problem.** "Evidence" is a mass noun; `Evidences` reads as a mistake and is the only plural
of its kind in the realization sections (`Controls`, `Workers`, `ServerSteps` are countable).

**Recommended solution.** `object evidence extends Evidence(...)`.

### 20. "observation" means two things

**Severity:** Medium

**Problem.** Section 3: an observation is "a value no recorded event carries", a derived read
such as an attempt count from a describe call. Section 19, line 943: "observations: a protobuf
message a run records". The first is a read the test makes; the second is data the run keeps.
They are not the same.

**Recommended solution.** Keep "observation" for section 3's meaning (it has a declaration,
`Observation(on = job)`). Call section 19's item "recorded messages" or fold it into "learned
values", whichever the kit actually does.

### 21. "witness" means three things

**Severity:** Medium

**Problem.** A witness is the path a `find` found (section 9), the counterexample a failed
`verify` or refinement reports (section 20), and the `witness(class, ...).records(fact)` sugar
that declares a pinned Scenario, Property and `find` Query in one (section 9, lines 517 to 532).
The sugar's name describes what it hopes to get, not what it declares.

**Recommended solution.** Rename the sugar to what it is: `path(class, ...).records(fact)` or
`trace(...)`. Keep "witness" for the result.

---

## Part C. Structure: consolidations and removals

### 22. Progress sits two sections away from Properties

**Severity:** Medium

**Problem.** A progress claim is written in `object properties` (line 614), but section 11
comes after Monitors and assumptions and before Holes. A reader learning what may go in
`properties` has to read past monitors to find the second thing that may.

**Recommended solution.** Order: 8 Properties, 9 Progress, 10 Monitors and assumptions, 11
Scenarios and Queries. Progress depends on assumptions, so add one sentence forward-referencing
`assume.fair`.

### 23. `.live` and Expectations belong in Realizations

**Severity:** Medium

**Problem.** `.live` is explained three times (lines 497 to 499, 531 to 532, 550 to 551) with
nearly the same words. The Expectations paragraphs (550 to 564) talk about Runs, judges,
dispositions, cleanup and conformance, none of which exist until section 19 defines a
realization. Section 9 is the longest section in the guide because of it.

**Recommended solution.** Move `.live`, `.explore` and both Expectations paragraphs into section
19 under a "Which Queries become Cases" heading. Section 9 keeps one sentence: "A `find` whose
machine declares a realization may be marked `.live`; see Realizations."

### 24. The composite-machine sections are interleaved

**Severity:** Low

**Problem.** Sections 13 to 17 are Refinement, Derivations, Facets, Capabilities, Compositions.
Facets split one machine; Refinement, Derivations and Compositions relate machines to each
other; Capabilities are declared in facets and expand to Properties. Capabilities sitting between
Facets and Compositions breaks both groupings.

**Recommended solution.** Facets, then Capabilities (declared in facets), then Refinement,
Derivations, Compositions. Or Facets, Refinement, Derivations, Compositions, then Capabilities
with a note that a facet declares them.

### 25. Modalities belong with the table

**Severity:** Low

**Problem.** "Modalities: what a row means" (lines 390 to 404) explains rows, disabled pairs and
hole rows, which are what section 7 derives. It sits at the end of section 6 (Rules), so the
reader meets "row" before "The table" has defined it.

**Recommended solution.** Move Modalities into section 7 after the derivation list and before
the lint table, which refers to it.

### 26. Migration notes do not belong in a semantics guide

**Severity:** Medium

**Problem.** Lines 72 to 74 ("Older Models bound a plain `Int`..."), line 121 ("Roles replace
the named phase sets Models used to declare"), line 388 ("The old `in(...)` rule heading is
retired and refused") and line 520 ("No new IR declaration kind is added") describe how the DSL
used to be or how a change was made. Git holds that history. Worse, section 18 (line 914) still
tells the reader to use `Finite.upTo(n)` after section 1 has retired it in favor of `UpTo[N]`.

**Recommended solution.** Delete the four notes. Fix line 914 to `UpTo[N]` or, if channels
genuinely need the old form, say why. If retired forms must be listed for the lint, put them in
one "Retired forms" appendix.

### 27. Lines 557 to 564 are pull-request text

**Severity:** High

**Problem.** The paragraph beginning "Convenience expectations and full `RunExpectation`
values..." reads as the acceptance criteria of the change that introduced it: "This text ... adds
no IR, Case or assessment field and cannot supply or change judge explanations. Existing judge
reason IDs and `expected_run` metadata are still emitted and compared." A reader of a design
guide has no use for "still emitted"; it is a statement about a diff.

**Recommended solution.** Replace with two sentences in section 19: "An expectation whose
Property or Contract outcome is not `satisfied` must say why in `because`. The text documents the
author's expectation and reaches no IR." Drop the rest.

### 28. Temporal-specific conventions are mixed into the DSL guide

**Severity:** Medium

**Problem.** The guide says at line 5 that it "covers the language". Yet it also carries the two
lifter switches `facets-only` and `facet-vocabulary` (lines 767 to 771), the capability package
split into `capabilities/product` and `capabilities/system` (783 to 785), the rule that every
bound is declared in `temporal/Bounds.scala` (535 to 537), and the StepTables equivalence check
(434 to 436). These are conventions of one set of Models and one build, not semantics.

**Recommended solution.** Move them to the README's "Writing a Model" or to a
`model/temporal/README.md`. Leave one pointer sentence in the guide where each was.

### 29. Channels are a full section for a feature no Model uses

**Severity:** Low

**Problem.** Line 914 says channels "are used by the lifter's fixtures today, not by a Temporal
Model". The feature still has section 18 of 21, a line in the Contents, and a bullet in section
1's catalog list, and it is the only place `Bindings` is introduced (point 9).

**Recommended solution.** Move Channels to an appendix headed "Forms no Model uses yet", or into
the fixture directory's own README. Keep the one-line mention in section 1.

### 30. The `CanEqual` workaround is a wart, not a rule

**Severity:** Low

**Problem.** Line 78 instructs the reader to write `val _ = () => summon[CanEqual[Phase,
Phase]]` to keep an import alive. This is a workaround for an unused-import warning. Documenting
it as part of the language makes it permanent.

**Recommended solution.** Fix the cause (suppress the warning for `framework.Finite.given`, or
export the givens from a package object) and delete the sentence.

### 31. Section 21's lists duplicate every section's core-form table

**Severity:** Low

**Problem.** Lines 1036 to 1045 enumerate every core and every sugar form by name. Sections 5,
8, 10 and 14 already have "Core form" tables. The list is a second place to update on every DSL
change and will drift (it already lists `disabled` twice).

**Recommended solution.** Keep the two rules (sugar lives in `Syntax.scala` files; each form
documents its core form and has a fixture) and the `make lint-model` pointer. Delete the
enumeration.

---

## Part D. Gaps

### 32. Facts are never defined

**Severity:** High

**Problem.** Facts appear as an enum (section 1), a `status` on a phase case (section 2), an
`evidence` line (section 4), the third field of `Step` (section 5), something a Property reads
with `records` (section 8) and something a realization confirms (section 19). No paragraph says
what a fact *is*: what a step records it for, whether `List[F]` is ordered or a set, whether one
step may record the same fact twice, whether a fact's position in the list matters for the IR
key, and what a fact case with fields means for the table.

**Recommended solution.** A "Facts" subsection in section 5 (or section 3, beside Outcomes)
answering those five questions, with the running example's `Fact` enum declared there once.

### 33. Two `evidence` declarations, no stated relationship

**Severity:** High

**Problem.** Section 4 declares `val evidence: PartialFunction[Fact, String]` on the machine,
"only the exceptions: a fact is confirmed by evidence of its own name". Section 19 declares
`object evidence extends Evidences(answered(Fact.submitted, startJob), ...)` on the realization,
"kinds of recorded data that confirm facts". The reader cannot tell which one the lowering reads,
whether the machine-level map is a default the realization overrides, or whether they are keyed
to each other by the string.

**Recommended solution.** Say in section 4 that the machine's `evidence` names the *kind* a fact
expects, and in section 19 that the realization declares each kind once and the lowering joins
them by name. If that is not how it works, say how it does.

### 34. One start or many

**Severity:** Medium

**Problem.** Line 236: "`init` is the one start state". Line 414: "the starts (the `init`)".
Line 654: "every start maps to a start". Line 875: "the reader composes every start". Line 507:
`.starts(s)` fixes another start for a Scenario. The guide uses singular and plural
interchangeably and never says what the plural refers to.

**Recommended solution.** One sentence in section 4: "A machine has one start, `init`. A
Scenario may start elsewhere with `.starts`. A composition's starts are the product of its
members' starts, which is why the reader speaks of starts in the plural."

### 35. What `end` means for a check is unsaid

**Severity:** Medium

**Problem.** Section 4 says `end` "says where a path may stop" and section 7 lists "the ends".
Nothing says whether a search stops exploring at an end state, whether an end state may still
have enabled rows (the lint table implies it may, since `silent-rejection` is reported only for
"non-end" states), or whether a `verify` reads steps taken from an end state.

**Recommended solution.** Three sentences in section 4 or 7: whether exploration continues past
an end, whether rows from an end state are part of the table, and what `.readAtEnds` on a monitor
therefore reads.

### 36. Unbound classes of a bound action

**Severity:** Medium

**Problem.** Section 6 binds `on(worker.finish(Result.ok))`, one class of a two-class action.
The guide does not say what the table holds for `finish(Result.failed)`: a disabled pair, a
`never-enabled` lint finding, or a refusal at the `on`.

**Recommended solution.** One sentence after the example, and a row in the lint table if it is a
finding.

### 37. Guards do not read inputs, but the guide never says so

**Severity:** Medium

**Problem.** Line 355's comment says the effect "reads the state and the inputs". Nothing says
whether a `where(g)` guard may read inputs. Line 369's guard reads `_.startToClose`, which is a
state field, but a reader who remembers `startToClose` as an input token from line 175 will think
guards read inputs.

**Recommended solution.** State it in the heading table: "`where(g)`: `g` reads the state only;
an effect that must branch on an input does so in its body." Rename the state field in the
example so it does not share a name with the input token.

### 38. Two errors with no Result

**Severity:** Medium

**Problem.** Line 334: a call that violates a callee's `require` "is an error, not a disabled
row". Line 413: "a result state outside the catalog is an error". Section 20's Results table has
no row either maps to; `declaration-error` is the nearest but does not list them.

**Recommended solution.** Add both to the `declaration-error` row, or to `admission-error` if
the reader finds them before any check, and cross-reference from lines 334 and 413.

### 39. `total` contradicts itself

**Severity:** Medium

**Problem.** Line 541 to 542: an authored total "is a review figure, not a prediction, and
nothing else reads it". Line 1011 lists "a wrong total" as an admission error, so the reader does
read it, and line 540 says "the reader checks" it.

**Recommended solution.** "The IR generator computes every Query's total. An author may state
it as a check of their own reading; the reader refuses an IR whose stated total differs from the
computed one. Nothing else reads it."

### 40. Undefined proper nouns

**Severity:** High

**Problem.** The following are used as defined terms and never defined or linked: Definition ID,
Behavior Fingerprint, Testpilot, Case, Run, Contract, Program, Verdict, Known Gap, judge,
disposition, cleanup, conformance, the `reach` path of a capability (line 823), `runs =` and
`learned =` on `Realizes` (line 954), `origin` (line 780), "exercises" a Property (line 444).
Most come from the Go tooling. A reader of the Scala guide has no way to find out what a
Behavior Fingerprint is or why it is computed.

**Recommended solution.** A glossary at the end of the guide with one line each and a link to
the tool or README section that owns the term. Define `reach`, `runs`, `learned` and `origin` in
place since they are DSL fields.

### 41. Role subtyping is implied, never stated

**Severity:** Low

**Problem.** `Retrying extends Waiting` is shown, and a Model "may declare a role of its own by
extending one". Whether `when[Waiting]` includes the `Retrying` cases, and whether a shared
Property over `Closed` sees a Model-declared `Expired extends TimedOut`, is left to the reader's
knowledge of Scala subtyping.

**Recommended solution.** One sentence: "A role test includes every narrower role: `when[Live]`
holds for a `Retrying` case."

### 42. The `FailureModel` rule is ambiguous

**Severity:** Low

**Problem.** Line 728: a failure model "must bind a fault ... and not every Query of it may
expect violation". This could mean "no Query may expect violation" or "at least one Query must
not expect violation".

**Recommended solution.** Say which. Presumably: "at least one Query of it must expect
satisfaction, since it is the real design".

### 43. `syncs` is missing from the section order; facets have no `states`

**Severity:** Low

**Problem.** Section 4 lists the sections of a machine in order and says the lint refuses any
other order. Compositions add `object syncs` (section 17), which is in no order list. Facets
(section 15) show `effects`, `rules`, `capabilities`, `properties`, `queries`; whether a facet may
hold `states` is unsaid, though line 770 lists what stays machine-wide and omits `states`.

**Recommended solution.** One order list per kind (machine, derived machine, facet, composition,
realization) in a single table in section 4, each row naming the required and optional sections.

### 44. `entity` "written only where ambiguous" with no example

**Severity:** Low

**Problem.** Line 242 says `entity` is derived from the actions and written only where that is
ambiguous. There is no example of writing it and no statement of what counts as ambiguous.

**Recommended solution.** One example line in the header listing, and the rule: "ambiguous when
the bound actions target two entities".

---

## Part E. Consistency

### 45. Capitalisation has no rule

**Severity:** Medium

**Problem.** Property, Scenario, Query, Model, Case, Run, Product and Protocol are capitalised as
terms. Monitor, assumption, hole, facet, capability, composition, channel, realization,
derivation are not. A reader cannot tell whether the capital marks "a declaration with an IR
kind" or is arbitrary.

**Recommended solution.** Pick one rule and state it in the Conventions section. The least
surprising: capitalise a word only when it names a declaration the IR carries (Property, Scenario,
Query, Monitor, Realization, Capability), lower-case everything else.

### 46. "behaviour" and "behavior"

**Severity:** Low

**Problem.** Line 734 has "behaviour"; lines 628, 946 and the term "Behavior Fingerprint" have
"behavior".

**Recommended solution.** American spelling throughout, to match the tooling's identifiers.

### 47. Effects as `def` or `val`

**Severity:** Medium

**Problem.** Line 279: effects are "defs of the state (and the class's inputs)". Line 296 shows
`val start = effect { ... }`. Whether both are allowed, which is preferred, and whether the
`val` block form may take inputs, are not said.

**Recommended solution.** A sentence after the block example: "A `def` is the general form; the
`effect { }` block is for an effect of the state alone, and the two lift to the same IR."

### 48. "step function" versus "effect"

**Severity:** Medium

**Problem.** Section 5 opens with "a *step function* gives a list of steps" and then says
"Effects live in `object effects`". Section 6 says the generator lowers rules and effects into
"one step function". So a step function is sometimes the authored def and sometimes the lowered
result.

**Recommended solution.** Define once: "An *effect* is a function you write. A *step function*
is what the IR binds to an action: the generator lowers an action's rules and their effects into
one." Then use the words consistently.

### 49. The example drifts between sections

**Severity:** Medium

**Problem.** `Phase` has `queued, running, done, timedOut` in section 2. Section 6 uses
`Phase.unscheduled`, `client.pause`, `process.stop`; section 8 uses `Phase.paused`; section 13
uses `Phase.backingOff` and `timers.backoff`. `Fact` is `submitted, started, finished` in
section 1 and `statusQueued, statusRunning, ...` in section 2. None of these additions is
declared. This is the detail behind point 2.

**Recommended solution.** As in point 2. At minimum, every case used in a later example must
be declared in an earlier one.

### 50. The Contents descriptions are in three styles

**Severity:** Low

**Problem.** Entries read "finite domains, catalogs, counters" (a list), "the promises" (a noun
phrase), "one thing changed" (a sentence fragment), "Product and Protocol" (two names), and
section 10 has no description at all.

**Recommended solution.** One style, a noun phrase of what the section defines, for every entry.

---

## Part F. Clarity

### 51. Three ways to write a class, in one sentence

**Severity:** Low

**Problem.** Lines 181 to 183 give the positional form, the named form with a parenthetical
about defaults, and the empty form, all in one sentence with the `:=` note trailing.

**Recommended solution.** A three-row table: form, example, which inputs it leaves at the
domain's first value.

### 52. Five refusals in one sentence

**Severity:** Low

**Problem.** Lines 325 to 327 list what the IR generator refuses about `choose` as one sentence
with four commas and a parenthetical.

**Recommended solution.** A bulleted list, each item naming who refuses it.

### 53. The refinement rule needs a list

**Severity:** Medium

**Problem.** Lines 652 to 656 state the refinement check as one sentence: starts map to starts,
every row result is carried by a product row from mapped source to mapped target with the same
outcome and the visible facts, or source and target map to one state and the step stutters, and
closedness is preserved. This is the one rule in the section that a reader must get exactly
right, and it is the hardest sentence in the guide.

**Recommended solution.** A numbered list of the three obligations (starts, carried steps,
stutters) and the role condition, each with its witness kind, followed by one sentence on what
`visible` narrows in each.

### 54. "A Property that reads two kinds ... Nobody calls it."

**Severity:** Low

**Problem.** Line 780 says "A Property that reads two kinds is brought only where both are
declared. Nobody calls it." Neither "kinds" (of capability) nor "it" (the companion def) is
resolved in the sentence.

**Recommended solution.** "A companion def that takes two capabilities is brought only where the
machine declares both. The author never calls a companion def; declaring the capability does."

### 55. "Its outcome and fact types are therefore `String`"

**Severity:** Low

**Problem.** Line 834's "therefore" does not follow from the preceding sentence, which is about
reachability. The reason is the composed keys (`<field>_<fact>`) two sentences earlier.

**Recommended solution.** Move the sentence up to follow the key definition: "...both members'
facts (`<field>_<fact>`). Its outcome and fact types are therefore `String`, the composed keys."

### 56. The guide reads as a spec by its author

**Severity:** Medium

**Problem.** Sentences such as "`evidence` lists only the exceptions: a fact is confirmed by
evidence of its own name unless a line says otherwise, and a fact case with fields needs a line
that covers all its values" (line 240) are precise and complete and still hard to read on first
pass, because they state the rule before the reader knows there is a question. The guide is dense
with such sentences; the density is uniform, so there is nowhere for a reader to rest.

**Recommended solution.** In the tutorial sections, lead each "Meaning" paragraph with the
question the concept answers ("How does a Run know a fact happened?") before the rule. Keep the
reference sections dense; that is what a reference is for.
