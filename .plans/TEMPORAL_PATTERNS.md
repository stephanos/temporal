# Temporal patterns in the Umpire Scala DSL

Research note, 2026-10-03, for fn-112 (`.3`, `.4`) and fn-120 (decision 4). Question: which named
behavioral patterns (level 1) make the Models' claims readable without a logic course, how each lowers
to the IR that exists, and what an `always`/`eventually`/`until` DSL (level 2) would cost. Scope is the
phrasing only: a parallel note, `.plans/SEMANTIC_PROTOCOLS.md`, designs reusable capability laws
(`terminalStatesAreFinal`, `Pausable x Pollable`); those laws are instances of the patterns below,
and this note says which. Nothing here is edited into a spec; proposed edits are at the end.

The three levels, as the owner set them: (1) named patterns for most claims; (2) an
`always`/`eventually`/`until` DSL for the rest, with TLA meaning (fn-120 decision 4, words never
`[] <> ~>`, `.plans/DSL_OPERATORS.md` rule 7); (3) the IR itself (`ir.Property` with `transition`,
`ir.Monitor`, `ir.Progress`) and its Quint/P exports as the escape hatch and backend.

## 1. Inventory

Every claim under `model/temporal/**`, counted per declaration instance (a shared `def` such as
`designQueries(m)` counts once per machine it is applied to). 171 Properties, 6 monitors, 10 progress
claims. The IR files hold 172 Property entries because `activity-system.json` re-lists the two product
Properties it reads through a refinement. Dwyer/Avrunin/Corbett pattern and scope in the third column.

| Shape (as written today) | Where, count | Pattern, scope | Level-1 fit |
| --- | --- | --- | --- |
| `when <action> holds { s => phase == P && s.facts.contains(F) }` and `whenAction(...)` | activity 9, nexus 10, system 10, race 2, close policy 69: **100** | none: a step postcondition (Hoare triple on one row), not a temporal claim | stays `when … holds`; fn-112.3's `records`/`in`/`implies` are its words |
| `holds { after => … }`, no `when`: `atMostOneActive` (7), `outcomePreserved` (9), `knownIsTheHandlersOutcome` (5), `failedCommitKeepsTheMessage` (2) | **23** | universality, globally (state invariant) | `never(p)` for the 7 `active != two`; the rest read better as the predicate they name |
| `holdsAcross`: `!over(before) \|\| after.state.x == before.x`: `terminalIsFinal` (2), `terminalStays` (7), `handlerEffectIsIrreversible` (5) | **14** | absence (of a change) after P, inductive: P is read from the before-state, so no history | `once(over).keeps(_.x)` |
| `holdsAcross`: `closedHistoryIsFrozen` (5): both states closed implies `known`, `intent` unchanged | **5** | absence after P, with P also required of the after-state | `keeps` variant with a guard on both sides; write raw until a second use |
| `holdsAcross`: `knowledgeIsFinal` (5): once `known` is `original(r)`/`successor(r)`, after still knows `r` | **5** | absence after P, but "unchanged up to a relation" keyed by `r` | raw `holdsAcross` (level 0) |
| `holdsAcross`: `before.phase != paused \|\| after.phase != started`: `pausedIsNotDispatched` (1), `notAdmittedWhilePaused` (7); `noUnnecessaryWait` (3) | **11** | absence of one transition, globally (two-state) | `never(to).from(before)` |
| `holdsAcross`: `committedStays` (4): held before implies held after or `acknowledged` recorded | **4** | universality after P until R, inductive ("P until R" per step) | `stays(p).unless(release)` |
| `holdsAcross`: `ackOnlyWhenKept` (9): a report in flight before is kept or still owed after | **9** | same family as the row above, but the "kept" relation depends on `r` carried by the channel | raw `holdsAcross` |
| monitor `retainedOutcome`, `ownerAcknowledgment` (close policy) | **2** | universality, globally, with a sticky verdict (once broken, broken on every later read) | `sticky(name)(predicate)` |
| monitor `terminalFinality` (activity system) | 1 | absence after P, tracked (`open/closed/reopened`) | hand-written; the inductive form is `once(admissionOver).stays(admissionOver)` |
| monitor `atMostOneActiveAttempt` | 1 | bounded existence (admissions net of completions, read after each admission) | hand-written: counts facts, needs history |
| monitor `singleOutcome` | 1 | bounded existence of distinct recorded outcomes, globally | hand-written: needs history |
| monitor `cancelPrincipal` | 1 | absence of change after Q, where Q (`requested(p)`) is a past event | hand-written: needs history |
| `leadsTo(name)(from, to, within = n, under*)`: `outcomeReachesOwner` (7), `retainedReachesOwner` (2), `retainedWaitsWithoutRecovery` (1) | **10** | response, bounded (`within`), globally, under weak fairness | existing `leadsTo`; no pattern needed |

Three readings: 100 of 171 Properties are action postconditions, not temporal claims, and `when X
holds` plus fn-112.3's words is their phrasing; every transition Property is two-state and inductive,
so every level-1 pattern below is an `ir.Property` with `transition = true`; and the four hand-written
monitors are each one of a kind, so `sticky` is the only monitor pattern with two uses.

## 2. The level-1 pattern set

Four words, each used by at least two existing claims, each lowering to IR that exists. Spelling obeys
`.plans/DSL_OPERATORS.md` (words, dotted, no symbol, no `inline`, no combinator vocabulary). Every
predicate the author passes is a lambda or a `def` of the lifted sources, exactly what `holds` takes.

### 2.1 `once(over).keeps(projection)`: a value that does not change after a condition

```scala
// standaloneactivity/Claims.scala, terminalIsFinal: before
activityProduct.property("terminalIsFinal") holdsAcross { (before, after) =>
  !productTerminal(before) || after.state.phase == before.phase }
// after
activityProduct.property("terminalIsFinal").once(productTerminal).keeps(_.phase)

// closepolicy/Claims.scala, handlerEffectIsIrreversible: before
m.property("handlerEffectIsIrreversible") holdsAcross handlerEffectIsIrreversible   // working(before.handler) || after.state.handler == before.handler
// after
m.property("handlerEffectIsIrreversible").once(s => !working(s.handler)).keeps(_.handler)
```

Lowering: `transition = true`, `when` empty, `holds` = a function `(before, after) =>
or(not(over(before)), eq(field(after.state, x), field(before, x)))` where `over` is lifted by
`callee`/`stepFunction` as today and `x` is the field the projection selects. Same nodes the lambda
lifts to now (`or`, `not`, `==`, `field`, call); 14 uses. The composition form is
`c.property(...).once(s => productTerminal(s.activity)).keeps(_.activity.phase)`: the projection is any
field path, which is what lets R4's `terminalStays` be declared once over `_.activity`.

### 2.2 `never(to).from(before)`: a transition that must not happen

```scala
// standaloneactivity/Claims.scala, pausedIsNotDispatched: before
holdsAcross { (before, after) => before.phase != ProductPhase.paused || after.state.phase != ProductPhase.started }
// after
activityProduct.property("pausedIsNotDispatched").never(_.state.phase == ProductPhase.started).from(_.phase == ProductPhase.paused)

// closepolicy/Claims.scala, noUnnecessaryWait: before
before.known == Knowledge.expired || after.state.known != Knowledge.expired || !nothingOwed(before)
// after
m.property("noUnnecessaryWait").never(_.state.known == Knowledge.expired).from(b => b.known != Knowledge.expired && nothingOwed(b))
```

Lowering: `transition = true`, `holds` = `(before, after) => or(not(from(before)), not(to(after)))`;
11 uses. `never(p)` without `.from` is the same-step invariant `holds(not p)` (`transition = false`,
`when` empty): `never(_.state.active == Active.two)` for the 7 `atMostOneActive`. Read "never
[step into] started from paused". Not `while`: a Scala keyword.

### 2.3 `stays(p).unless(release)`: a condition that latches until a recorded release

```scala
// System.scala, committedStays: before
before.custody == Custody.nowhere || after.state.custody != Custody.nowhere || after.facts.contains(QueueFact.acknowledged)
// after
m.property("committedStays").stays(_.custody != Custody.nowhere).unless(_.records(QueueFact.acknowledged))
```

Lowering: `transition = true`, `holds` = `(before, after) => or(not(p(before)), or(p(after.state),
release(after)))`; `stays(p)` alone drops the last disjunct. 4 uses, and it is the inductive form of
`terminalFinality` (`stays(s => admissionOver(s.phase))`).

### 2.4 `sticky(name)(predicate)`: a promise that, once broken, stays broken

```scala
// closepolicy/Model.scala, retainedOutcome: before
monitor[CloseResetState, Answer, Fact, Boolean]("retainedOutcome", false)((lost, _, after) => lost || !outcomePreserved(after))(lost => lost)
// after
val retainedOutcome = sticky[CloseResetState, Answer, Fact]("retainedOutcome")(outcomePreserved)
val ownerAcknowledgment = stickyAcross[CloseResetState, Answer, Fact]("ownerAcknowledgment")(ackOnlyWhenKept)
```

Lowering: `ir.Monitor` with `state = bool`, `initial = false`, `next` = `or(m, not(call(p, after)))`
(or `call(p, before, after)`), `violated` = the identity, `evaluate = every_step`; the Definition ID is
the `val`'s full name as today, so the monitor-agreement exports and `MonitorExpectation` names are
unchanged. 2 uses, both in the close policy; it is the one monitor shape that recurs.

### What the capability laws build from

`terminalStatesAreFinal(terminal, phase)` is `once(terminal).keeps(phase)` applied to a capability's
projection, and its monitor twin is `stays(terminal)`. The `Pausable x Pollable` interaction law is
`never(started).from(paused)` applied to the product or record state. `stays(held).unless(acknowledged)`
is the queue's custody law. The capability layer supplies the projections and names; the patterns
supply the phrasing and the lowering, and the capability note should not define a second lowering.

### Lowering mechanics, and what stays identical

- **Where it runs.** The lifter inlines only functions of the lifted sources (`defs`,
  `model/lifter/Context.scala:19-51`; `callee` refuses the rest, `model/lifter/Expressions.scala:132`),
  so a word defined in `umpire` cannot carry its own body. Each word is one match case in
  `model/lifter/Claims.scala::fold` beside `holds`/`holdsAcross` (line 180), lifting the author's
  lambdas with `stepFunction` and synthesizing the Property function from existing `Expr` nodes: the
  cost `.plans/DSL_OPERATORS.md` principle 7 prices (one match case, one lifting and one refusal fixture).
- **Tables, IDs, fingerprints, answers, Case bytes: unchanged.** A Property enters no table row; the
  Behavior Fingerprint is a table's (`tools/umpire/model/checking.go:138`); a transition Property is
  never realized (SEMANTICS.md, Realizations, rule 5), and a same-step Property lowers to Contract
  clauses by evaluating it over the table, not by reading its tree. What changes is the lifted function
  body's shape (`or(not(eq))` where the author wrote `!=`), as fn-112.3's `in` and `implies` already
  change it; the fn-112 R1 projection compares tables, IDs, fingerprints, answers and Case bytes.
  `sticky` can be tree-identical, since it synthesizes the tree the lambda lifts to today.
- **Semantics check against SEMANTICS.md.** "A transition Property holds of the state before a step
  and the step record", on every step: each pattern is a two-state predicate, so its meaning is the
  lambda it replaces. Two rules become refusals: a transition Property takes no `when` (unsupported
  with `through`/`when`), so `once`, `never(...).from` and `stays` refuse a preceding `when` at the
  line; and a trigger not readable from the before-state (Dwyer's "after Q" for a past event,
  "between", "after-until") needs a monitor with history, which is what `terminalFinality`,
  `cancelPrincipal`, `singleOutcome` and `atMostOneActiveAttempt` are. No new IR schema, checker or
  export change: Quint and P read Property and monitor functions as trees of node kinds already encoded.
- **Not level 1.** A history-scoped `keepsOnce`/`atMostOnce` monitor pattern needs the lifter to
  synthesize a finite state type (`open/closed/reopened`): new IR content that would change the
  existing monitors' state types and IDs, for two uses with different state types. That is level 2.

## 3. What level 2 would need, and who needs it

An `always(p)` / `eventually(p)` / `p.until(q)` DSL over step records with TLA meaning:

- **IR.** A path-formula node kind (or a `Property` kind beside `transition`) carrying the formula
  and the fairness it is checked under; or a lifter that compiles the safety fragment (`always`,
  `until` without liveness) into synthesized monitors with generated state types. Either is a schema
  or type-synthesis change.
- **Checker.** Bounded trace semantics for `eventually`/`until` (a finite prefix cannot refute
  `eventually p`: an open prefix is `inconclusive`, as progress already reports) and fair-lasso
  detection generalized from `checker.progress` (`tools/umpire/model/checking.go:695`), which handles
  one `from`/`to` pair with a step bound.
- **Exports.** Quint has `always`/`eventually` but Apalache checks only a fragment; P has no temporal
  formulas (safety via spec machines, liveness via hot states). The export README already marks
  `query-agreement` and `progress-agreement` `unsupported`; level 2 widens that gap. SEMANTICS.md
  would gain a "Temporal claims" section at the level fn-120 R13 names.
- **Who needs it first.** Nobody: all 171 Properties are one- or two-state, and the 10 liveness
  claims are bounded responses `leadsTo` states (bounded leads-to under weak fairness is not TLA's
  unbounded `~>`, so do not rename it `eventually`). fn-118 declares visibility hints and derives
  waits, no claims. fn-119's workflow asks completion, retry and timeout as `when … holds`
  postconditions; a liveness Query there is a `leadsTo` within its limits. fn-120 bounds itself to "no
  new temporal operators, no strong fairness, no unbounded liveness". The trigger to record: a
  history-scoped claim recurring across two state types, which today's four unique monitors do not.

## 4. Where level 1 goes

**fn-112.4 (R4, R16), not .3.** Task .3 owns the step-level words (`implies`, `in`, `records`)
in `Expressions.scala`; task .4 owns `umpire/Claims.scala` and `lifter/Claims.scala`, and its R4
("the minimum typed abstraction that lets the three shared properties be declared once … with no new
IR") is exactly `never(_.activity.phase == started).from(_.activity.phase == paused)`,
`never(_.state.activity.active == Active.two)` and `once(over).keeps(_.activity.phase)` over a member
projection. `sticky` has both uses in the close policy, so it belongs to fn-114's Nexus rollout, with
its definition landing in .4 beside the others only if .4 is still open when fn-114 starts.

Proposed spec text (fn-112, API Contracts, after "Step helpers"):

> **Claim patterns** in `umpire`, on `PropertyBuilder`: `once(over).keeps(projection)` (a value
> unchanged after a condition), `never(to)` and `never(to).from(before)` (an invariant; a transition
> that must not happen), `stays(p).unless(release)` (a condition kept until a recorded release), and
> the monitor `sticky(name)(predicate)` (a promise that stays broken once broken). Each is a plain
> `def` the lifter matches by name and lowers to an existing `Property` (`transition` set for the
> two-state forms) or `Monitor`, with the function body built from `or`, `not`, `==`, `field` and
> calls; no new IR node. A pattern after `when` is refused at its line. Patterns that need history
> remain hand-written monitors; see `.plans/TEMPORAL_PATTERNS.md`.

R4, amended: "… declared on the machine and on both composition families from that one definition,
written as `never(...).from(...)`, `never(...)` and `once(...).keeps(...)` over the member projection."
R16: add "and each claim pattern" to the fixture list.

Task .4 Approach, one bullet: "Add `once/keeps`, `never/from`, `stays/unless` to `PropertyBuilder`
and `sticky` beside `monitor`; lift each in `fold` by synthesizing the Property or monitor function
from the author's lambdas; refuse a pattern chained after `when`, a `keeps` projection that is not a
field path, and `sticky` over a state type that is not the machine's." Acceptance: "`never`, `once`,
`keeps`, `stays`, `unless` and `sticky` lower to the existing Property and Monitor IR; tables, IDs,
fingerprints, answers and Case bytes equal the task-1 baseline; R4's three properties use them."

**Lifter fixtures required** (`model/lifter/testdata/lifts/`, compared with `expected/*.json` by
`Fixtures.test.scala`): one lifting fixture, say `Patterns.scala` with `expected/patterns.json`,
declaring each form once on a machine and once on a composition over a member projection, including
`never(p)` without `from`, `stays(p)` without `unless`, and `sticky` in both arities, so the expected
JSON shows the synthesized bodies; and refusal lines added to `Rejects.scala`/`expected/rejects.txt`
for `when … once(...).keeps`, `keeps(s => f(s))` with a non-field projection, and `sticky` whose
predicate takes the wrong state type (a compiler refusal, listed as such under fn-113.9 if R16 accepts
it). The migration goldens of fn-112.1 prove the frozen artifacts.

**Decision-context bullet** (fn-112, after "Operators, 2026-10-03"):

> **Temporal claims, 2026-10-03.** Claims are written at three levels. Level 1 is a small set of
> named patterns (`once/keeps`, `never/from`, `stays/unless`, `sticky`), each lowering to the existing
> transition Property or monitor IR with no new node, adopted in task .4 for R4. Level 2, an
> `always`/`eventually`/`until` DSL with TLA meaning (fn-120 decision 4, words only), is deferred:
> no existing or planned Query (fn-118, fn-119, fn-120) needs it, and it requires a formula node, a
> bounded-trace and fair-lasso checker beyond `leadsTo`, and export gaps in P. It gets its own spec
> when a history-scoped claim recurs across two state types. Level 3 is the IR and its exports, which
> stay the escape hatch. Action postconditions (100 of 171 claims) are not temporal and keep
> `when … holds`. Inventory and lowerings: `.plans/TEMPORAL_PATTERNS.md`.
