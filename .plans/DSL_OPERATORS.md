# Operators in the Umpire Scala DSL

Research note, 2026-10-03, for fn-112 (`.2`–`.5`, `.9`) and fn-120. Question: where would custom
Scala operators (symbolic, infix-alphanumeric, extension methods, Scala 3 features) make Temporal
Models read better, and where would they hurt? Audience: Temporal feature developers. Test for every
candidate: a reader must guess what it means without a legend. Constraint: the lifter reads TASTy and
matches calls by method name (`Ident("~>")`, `Select(b, "holds")`), so an operator is a plain `def`
or `extension` whose tree shape the lifter recognises; no runtime semantics, no macros, no library
(fn-113 R25). Nothing here is edited into the spec; proposed edits are listed at the end.

## Principles, with the evidence behind them

1. **Words for logic, symbols only where programmers already own them.** Quint replaced TLA+'s
   `/\ \/ ~ => <=> [] <>` with `and or not implies iff always eventually`, but kept `== != & |`, dot
   chaining and `x' = e`: "well-known operators are written like in most programming languages",
   "keep the set of ASCII control characters to a minimum" ([design principles](https://quint.sh/docs/design-principles),
   [language](https://quint.sh/docs/lang)). Alloy offers `=>`/`implies`, `&&`/`and` side by side and its
   users report the English forms read better. P is C-like: `&& || ! == in`, keywords `send goto
   assert choose`. The pattern across all four: symbols that C/Java/Go taught everybody survive;
   symbols a formalism invented get replaced by words.
2. **One symbol, one meaning, in one position.** sbt kept `:=`, `+=`, `++=` (C's "set/append") and
   removed `<<=`, `<+=`, `<++=` in 1.0 because they "have been sources of confusion for many users"
   ([release notes](https://www.scala-sbt.org/1.x/docs/sbt-1.0-Release-Notes.html), [#2716](https://github.com/sbt/sbt/pull/2716)).
   The lesson is not "no symbols" but "no invented symbol families": the survivors were already idioms.
3. **A symbol must be frequent enough to be worth remembering.** Underscore's "Keeping Scala Simple"
   and the Typelevel discussion of cats' `|+| === <*> *>`: acceptable only when there are few of
   them and they recur constantly; the stdlib's `:\` and `/:` were removed because they add nothing
   over `foldLeft`/`foldRight` and only cost the reader. Akka's `!`/`?` survived because the
   vocabulary is two symbols and every tutorial opens with the legend "tell/ask"; a Model has no such
   tutorial, so the budget is smaller still.
4. **The Scala style guide's rule is the one to quote in review.** "Avoid!" symbolic names, except
   in a DSL "so long as the syntax is actually beneficial" and for operations that are "mathematically
   well-defined"; "if you need to explain what the method does, then it should have a real,
   descriptive name" ([naming conventions](https://docs.scala-lang.org/style/naming-conventions.html)).
5. **Scala 3 mechanics.** An alphanumeric method may be used infix only with the `infix` modifier
   (the project compiles with `-Werror`); symbolic names are always infix; give symbols a
   `@targetName` for stack traces and docs ([operators](https://docs.scala-lang.org/scala3/reference/changed-features/operators.html)).
   Alphabetic operators have the lowest precedence, so `a == b && c implies d` parses as
   `((a == b) && c) implies d`, which is the reading an author wants. The lifter matches the source
   name, so `@targetName` is free. Avoid `inline`/`transparent inline` on the author surface: the
   lifter unwraps only `Inlined(_, Nil, e)`.
6. **Infix English works when the call is one clause with one right operand** (ScalaTest's
   `x shouldBe y`, our `query find p in s limits l`). Multi-argument calls stay dotted
   (`phase.in(a, b)`), as munit keeps `assertEquals(a, b)`.
7. **Lifter cost is per operator, not per symbol.** Each operator is one match case in
   `model/lifter` plus a lifting fixture and a refusal fixture (fn-112 R16). Symbolic and alphanumeric
   cost the same. What raises cost is type-directed inference (a root type the operator must take
   from context) or a new IR node; what lowers it is a direct lowering to an expression SEMANTICS.md
   already defines.

## Inventory of what exists today

| Operator | Where | Meaning | Verdict |
| --- | --- | --- | --- |
| `action ~> stepFunction` | `steps(...)`, fn-112 `rebind`/`extend` | bind an action to its step function | **Keep.** The one symbol that earns its place: it names a relation no English word does better, appears in every machine, and is typed per arity. Add `@targetName("binds")`. Never reuse it for anything else, in particular not for leads-to (TLA+'s `~>`); see Do-not-do. |
| `"member" -> machine`, `"member" -> action`, fn-112 `_.member -> action`, `perform(action -> command)`, fn-120 `name -> steps` | compositions, syncs, scripts, `choose` | key paired with its value | **Keep.** Scala's own tuple arrow; readers know it from `Map(...)`. Rule: left is always the key, right its value; never "transition to". |
| `token := value` (fn-112.5) | named inputs `start(scheduleToStart := expires)` | give a named slot a value | **Keep (settled).** The sbt survivor; universal "set". Add `@targetName("set")`. The only symbol fn-112 adds. |
| `m.property(...) when c holds f`, `holdsAcross` | Properties | restrict, then state the predicate | **Keep.** Reads as a sentence. `whenAction("...")` is replaced by the typed form fn-112.4 settles; `holdsAcross` stays (renaming churns frozen goldens for no gain). |
| `query find p in s limits l`, `verify` | Queries | the question, its path, its bound | **Keep.** Spec adds `limits depth(3)` and `.total(n)`; both are words. |
| `on`, `creates`, `results`, `schema[T]`, `input` | action chains | | **Keep.** |
| `s.facts.contains(f)` | predicates | fact recorded | **Rename** to `s.records(f)` (candidate 3). |
| `a == b \|\| a == c \|\| ...` | step guards | phase membership | **Replace** with `phase.in(a, b, c)` (candidate 2, spec R5). |
| `!a \|\| b` | transition Properties | implication | **Replace** with `a implies b` (candidate 1, spec R5). |
| `&&`, `\|\|`, `!`, `==`, `!=`, `+`, `-`, `++`, `<`, `<=` | everywhere | | **Keep as symbols.** Quint keeps `& \| == !=` too; these are universal. Do not add `and`/`or`/`not` words beside them: two spellings of one thing is a legend. |
| `Inbox.send`, `isEmpty`, `isFull`, `hole.reached` | channels, holes | | **Keep.** Words; Akka-style `!` rejected (Do-not-do). |
| `leadsTo(name)(from, to, within, under*)`, `monitor(...)(next)(violated)` | progress, monitors | | **Keep as words.** fn-120 records that any temporal operator takes its meaning from TLA; when one comes it is spelled `always`/`eventually`, as Quint does, never `[]`/`<>`/`~>`. |

## Candidates, ranked

Lifter cost: **low** = one match case lowering to an existing expression node plus two fixtures;
**medium** = also needs a type read from context; **high** = new IR. Risk = chance a reader misreads.

### 1. `a implies b` (accept; already in R5)

```scala
// model/temporal/standaloneactivity/Claims.scala, terminalIsFinal — before
!productTerminal(before) || after.state.phase == before.phase
// after
productTerminal(before) implies after.state.phase == before.phase
```

Definition: `extension (a: Boolean) infix def implies(b: => Boolean): Boolean = !a || b`.
Lowering: `or(not a, b)` with the short-circuit SEMANTICS.md already gives `or`, so a hole in `b` is
not reached when `a` is false. Cost **low**. Risk **low**: it is the word Quint, Alloy and every
logic course use; precedence (lowest) matches intent. Refusal fixture: none needed beyond the
compiler. Do not add `iff`: no Model needs it.

### 2. `phase.in(a, b, c)` (accept; already in R5)

```scala
// standaloneactivity/Model.scala, controlStep requestCancel — before
if s.phase == ProductPhase.scheduled || s.phase == ProductPhase.started ||
  s.phase == ProductPhase.paused || s.phase == ProductPhase.cancelRequested
// after
if s.phase.in(scheduled, started, paused, cancelRequested)
```

Definition: `extension [A](a: A) def in(first: A, rest: A*): Boolean`; one-arity forbids `x.in()`
by the signature. Lowering: `OP_CONTAINS(a, list(first, rest...))`, which SEMANTICS.md spells
`a in b`. Cost **low**. Risk **low**; P uses `in` for membership. Write it dotted, not
`s.phase in (a, b)`: multi-argument infix reads as a tuple. It shares the word `in` with
`find p in s`; the receivers differ (a value vs a Query) and nobody has confused "in a set" with
"in a scenario". Also `terminalPhase(p)`-style helpers become `p.in(completed, failed, ...)`.

### 3. `step.records(fact)` (accept; extend R5 to Step, align with R3's `records`)

```scala
// standaloneactivity/Claims.scala, completes — before
s.state.phase == Phase.completed && s.facts.contains(ProtocolFact.statusCompleted)
// after
s.state.phase == Phase.completed && s.records(ProtocolFact.statusCompleted)
```

The spec already gives compositions `after.records(_.activity, fact)`; a plain `Step` should say the
same word so a Property reads the same on a machine and on a composition. Lowering:
`OP_CONTAINS(fact, field(step, facts))`, exactly what `contains` lifts to today. Cost **low**.
Risk **low**. Not infix (`after records fact` is defensible but then `records` would be the only
infix predicate; keep it dotted like `contains`).

### 4. `scheduleToStart := expires` (accept; settled in R9)

```scala
start(unset, expires, unset)            // before
start(scheduleToStart := expires)       // after
```

Keep as settled. Two rules to add: `@targetName("set")`, and `:=` means "this named slot gets this
value" and nothing else, so that candidate 5 may reuse it with the same meaning and no other
operator may. Cost as planned by fn-112.5.

### 5. `_.field := operand` inside a typed `rpc`/`readUntil` (accept for fn-112.9, owner's call)

```scala
// standaloneactivity/Realization.scala, startBinding — before
Assignment.typed(Field[StartActivityExecutionRequest, String](_.namespace),
  Operand.environment[String](workerNamespaceBinding))
// after, inside rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) { ... } whose request type is in scope
_.namespace := environment(workerNamespace)
```

fn-117's API Contracts table promised `_.taskQueue.name := Environment(b)`; the implementation
shipped `Assignment.typed(Field[Req, V](...), operand)`, which repeats the request type on every
line (about 25 times in the activity realization). fn-112.9's script helpers are where the promise
can be kept: `rpc(role, method)` opens a scope (a context function with `Req` fixed) in which
`:=` takes a `Req => V` selector on the left. Same operator, same meaning as candidate 4. Cost
**medium**: the lifter must read `Req` from the enclosing `rpc` call rather than from the `Field`
type argument, and the lambda arrives through an extension rather than `Field.apply`; needs one
positive and one refusal fixture (selector of another request type does not compile). Risk
**low**. If the context-function form proves awkward in TASTy, the fallback is `field(_.namespace)
:= ...` with `Req` still inferred from the scope; do not fall back to a symbol with another meaning.

### 6. `enter`, `stay`, `disabled`, `.because(...)`, `perform`, `onPath`, `everyCase`, `choose` (accept as words; already specified)

No operator. `stay(s).recording(fact)` (fn-120 sketch) and `enter(state, facts*)` read as
sentences; they lower to the `Step` construct the lifter already builds. Keep `disabled` a `val`
(not `disabled()`), as the spec writes it. fn-127 renamed two of these words; see Words renamed.

### Words renamed (fn-127, 2026-10-05)

A word the DSL shares with Temporal's own vocabulary, or with an operator this note reserves, is
read with the wrong meaning by the people the Models are written for. fn-127 renamed each such
word; nothing it lowers to changed, so the IR, the Cases and the Contracts are the same.

| Was | Is | Why |
| --- | --- | --- |
| `accept` | `enter(state, facts*)` | Temporal says an Update or a Nexus operation is *accepted* (`WorkflowExecutionUpdateAccepted`); a step function's verb read as that event. A step enters its state. |
| `Accepted`, the given `enter` and `stay` read | `given Ok[O] = Ok(o)` | The same collision. "Ok" is the gRPC word every reader owns for the answer that is not an error. Each Model's `Outcome.accepted` stays: it is Model vocabulary, in the IR type catalogs, the fingerprints and the Case bytes. |
| the realization's `poll` and `Instruction.poll` | `readUntil(evidence, role, …) { … }`, `Instruction.readUntil` | The worker's long poll (`PollActivityTaskQueue`) is something the Models model; the instruction reads evidence until a condition holds. The IR record keeps its name, Poll. |
| `.setting` on a call | `call.withFields { field := … }` | fn-125's `setting[T]` declares dynamic configuration, a *setting*; the method appends request fields. |
| `always` | `everyCase(command)` | `always` is the temporal operator Do-not-do 7 reserves with its TLA meaning; the item is a command every Case carries. |
| `umpire.realize.Outcome` | `umpire.realize.PropertyOutcome` | Each Model declares its own `enum Outcome`, so a Queries file imported the realization's one renamed (`Outcome as RunOutcome`). It is the outcome a Run gives a Property. |

### Rejected candidates (real temptations, each fails the legend test)

- **Scenario sequencing** `start() >> attemptStart >> attemptResult(completed)` or `andThen`: a
  comma list already reads as a sequence, and fn-120 defers Scenario combinators (`anyOrder`,
  `repeat`) to a later spec. Adding `>>` now would pre-empt that design.
- **Leads-to** `pending ~> settled` or `|->`: collides with the binding `~>`; TLA+ readers would
  read it as leads-to and Temporal readers as "binds". Keep `leadsTo`.
- **Typed equality** `===`/`=!=` (cats): `==` on enums and case classes is already lifted to
  `OP_EQ`, and the scalafix config leaves universal equality on. An `Eq` typeclass is a library and a
  second spelling of equality.
- **Predicate combinators** `holds (lands(Phase.completed) and records(f))`: `s => ...` lambdas are
  ordinary Scala; a combinator vocabulary is a legend.
- **`and`/`or`/`not` words** beside `&&`/`||`/`!`: Alloy's dual syntax only works because every reader
  learns both; we would have two spellings in one file.
- **Channel send/ask** `inbox ! m`, `inbox ? m` (Akka): `send(m)` is a word and the lifter already
  matches it.
- **Unicode** `→ ∧ ∨ ∈ ⟹`: unguessable on a keyboard, and `-feature` warnings aside, no editor
  completes them.
- **`disabled unless cond`** or `guard(cond) { ... }`: `if cond then ... else disabled` is plain Scala
  and lifts as `if`.

## Do not do

1. No new symbolic operator beyond `~>`, `->`, `:=`. Three is the whole legend, and it fits in one
   README line.
2. One symbol, one meaning, one position: `~>` binds an action to a function; `->` pairs a key with
   its value; `:=` gives a named slot a value. Never overload one of them with a second reading, and
   never spell one meaning two ways.
3. No `and`/`or`/`not`; keep `&&`/`||`/`!`. Only `implies` is a word, because it has no symbol
   programmers already own (Quint's rule).
4. No alphanumeric infix without `infix`, and no alphanumeric infix with more than one argument.
5. No `inline`, `transparent inline` or macro on the author surface; every operator is a `def` or
   `extension` whose call the lifter matches by name in TASTy (fn-113 R25, fn-112 "built once, in the
   lifter").
6. No operator that needs a sentence of explanation in a doc comment (style guide test). If the
   explanation is one English word, that word is the method name.
7. No temporal symbols (`[]`, `<>`, `~>`, `WF`); when temporal operators come (fn-120 decision 4),
   they are `always`/`eventually`/`weakFair` with TLA meaning.
8. No operator whose lowering needs a new IR node. Every accepted candidate above lowers to `or`,
   `not`, `OP_CONTAINS`, `field` or the existing assignment record.

## Which fn-112 task adopts what, and the spec text it implies

| Candidate | Task | Proposed edit (not applied) |
| --- | --- | --- |
| 1 `implies` | fn-112.3 | API Contracts, Step helpers: after `a implies b` add "an `infix` extension of `Boolean` with a by-name right side, lowered to `or(not a, b)` so the right side is read only when the left holds". fn-112.3 Approach: name the lowering. |
| 2 `in` | fn-112.3 | API Contracts: `phase.in(a, b, c)` → "`phase.in(a, b, c)` (dotted, at least one member; lowered to `OP_CONTAINS` over a list literal)". R5 unchanged. |
| 3 `records` on Step | fn-112.3 (define), fn-112.4 (composition form) | API Contracts, Step helpers: add `after.records(fact)  // the step records the fact; the composition form is after.records(_.member, fact)`. R5: "`enter`, `disabled`, `stay`, `in`, `implies` and `records`". R3's composition `records` then shares the definition. |
| 4 `:=` rules | fn-112.5 | Settled planning decisions, named-inputs bullet: add "`:=` carries `@targetName("set")` and means only 'this named slot gets this value'; no other operator takes that meaning and `:=` takes no other." |
| 5 `:=` in scripts | fn-112.9 | API Contracts, Realization script helpers: add one line to the `perform(start(...) -> startActivity(scheduleToStart = deadline))` example showing a typed assignment `_.namespace := environment(workerNamespace)` inside `rpc(role, method) { ... }`, and to R13 "…and writes request fields with `:=` against the method's request type". Owner decides; if declined, record in Decision Context that fn-117's `:=` sketch was superseded by `Assignment.typed`. |
| 6 words | fn-112.3, .9, fn-120.1 | No text change; fn-120.1 `choose(name -> steps)` already obeys the `->` rule. |
| Policy | fn-112 spec, Decision Context | One bullet: "Operators. The DSL has three symbols, `~>`, `->` and `:=`, each with one meaning; logic beyond `&& || !` is spelled in words (`implies`, `in`, `records`). Rationale and rejected candidates: `.plans/DSL_OPERATORS.md`." |
| README | fn-112.10 or fn-114.8 | "Writing a Model" gains one line listing the three symbols and their meaning, so the legend is in the place a new author reads first. |

## Conflicts with work already done

- **fn-117 (closed).** Its API Contracts sketched `_.taskQueue.name := Environment(b)`; the shipped
  surface is `Assignment.typed(Field[Req, V](_.x), Operand.…)`. Candidate 5 closes that gap in
  fn-112.9 without reopening fn-117; nothing fn-117 lifted changes. If the owner declines, the spec
  should say so, since the fn-117 text still reads as a promise.
- **fn-113.** R25 (no library) and the retired native evaluator are satisfied: every candidate is a
  plain `def`/`extension` with no runtime meaning. fn-113.9 (must-not-compile fixtures, still open)
  is where the compiler-level refusals for `in()` with no member and `:=` on a foreign token belong
  if fn-112 R16 does not take them.
- **fn-120.** `choose(name -> steps)` uses `->` as key→value, consistent with rule 2. Its decision
  that temporal operators take TLA meaning is compatible with Do-not-do 7 and the `~>` collision
  note.
- **`in` twice.** `find p in s` (Query) and `phase.in(a, b)` (membership) coexist; the spec need
  not rename either.
