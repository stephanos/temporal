# Correlated obligations: `owes(fact).until(fact)`, a progress claim keyed by a fact's payload

## Conversation Evidence

> user (2026-10-17): "is any of this useful for Umpire?" (on a survey of FizzBee, P, statecharts, Alloy, Event-B, Ivy and UPPAAL notation; P's hot/cold monitor states were the one idea judged new)
> user (2026-10-17): "1. Correlated obligation monitors — yes, add flow spec"
> assistant, 2026-10-17: today the per-id obligation is a hand-written `Set[Id]` monitor read at ends; every model with a request/response pair rewrites the fold, and the fold is where the correlation mistake hides.

## Goal & Context

<!-- Goal & Context: 30% [user], 50% [paraphrase], 20% [inferred] -->

A progress claim today is `whenever(p): eventually(q)` over state predicates (fn-190 R13,
`model/design/progress-notation.md`). It cannot say "every request is answered by its own response":
`whenever(pending.nonEmpty): eventually(pending.isEmpty)` passes on a path where request 1 is never
answered because request 2 arrives and is answered each time. The claim that can say it is a
monitor whose state is the set of outstanding ids, advanced by the facts a step records and read at
ends, which is P's hot state parameterised by the id. It is expressible now in a dozen lines, and
every model with a request/response pair rewrites that fold; a fold that matches the discharging
fact without its payload discharges every request with one response, and nothing reports it.

This spec gives that claim one word: `owes(Fact.requested).until(Fact.responded)`, in `progress`,
named after its `val`, keyed by the payload the two facts share. It passes the test
`progress-notation.md` sets for a new word: it changes what is checked, since it reads later steps
and correlates them by payload, which no `if` and no `whenever` over state can say. Temporal is
where it earns its keep: Nexus result delivery, an activity's response per attempt, and a task
queue's delivery per task are all "one obligation per id" promises, and the mini feature's monitors
spell them as folds today. The samples do not need it; a fixture carries it until Temporal returns.

DSL only, in the owner's DSL completion phase (2026-10-17): the word, the Scala checker's
semantics (fn-196 makes that checker the oracle) and the documents. Lifting, export and conformance
over Runs are the later IR phase and are bounded below.

## Architecture & Data Models

<!-- Architecture: 20% [user], 50% [paraphrase], 30% [inferred] -->

- **The claim.** `owes(open)` names a fact constructor `K => F` whose recording opens an obligation
  for its payload; `.until(close)` names the constructor `K => F` whose recording for the same
  payload discharges it. The claim is a `LeadsTo`-shaped value: `within` optional on `until`, an
  environment after `in`, nothing fair unless the scenario says so. A zero-payload pair (two plain
  cases) is one obligation and is admitted.
- **Its state.** The checker explores the Machine's vars together with the set of owed payloads,
  `Set[K]` bounded by `K`'s `Finite` catalog. The set is part of explored-state identity: two paths
  that reach one var state while owing different payloads stay apart, as monitors do today
  (SEMANTICS §10). Recording `open(k)` adds `k`; recording `close(k)` removes it; a step records
  both in one step opens and closes nothing.
- **Its verdict.** Per payload, the progress semantics of SEMANTICS §11 with `from = owes k` and
  `to = !owes k`: violated by a deadlock or a path end while `k` is owed, a fair non-progress cycle
  on which `k` stays owed, or, with a bound, `within` steps from the opening step without the
  discharge. Each is reported with the payload, the opening step and the path. Verified on a
  complete exploration; a cut answers unknown. The longest open interval is reported where no path
  cycles.
- **What it is not.** A `close(k)` recorded while `k` is not owed is no violation: repeats and
  unsolicited discharges are the model's business, and a Model that wants them refused writes a
  postcondition. The word promises discharge and nothing else.
- **Where it lives.** `framework.dsl` (Assume.scala beside `whenever`, the Syntax macros that name
  the `val` and check `within`), the checker (fn-196's module; `NativeCheck` until it moves), the
  block-form lint, SEMANTICS §11, the framework README and `progress-notation.md`. Progress stays a
  model check only: the export lifts no `owes`, as it lifts no `whenever`.
- **Prototype first.** A probe under `model/design/` tries the spelling on a dispatcher fixture and
  on the mini feature's `attemptsNeverRewind`-style monitors as reading material, and the design
  note records the spelling the owner picks. Candidates: `owes(f).until(g)`; `whenever(recorded(f(id))):
  eventually(recorded(g(id)))` with `id` bound by the block (quantified `whenever`); `obliged(f, g)`.

## API Contracts

```scala
enum Fact:
  case requested(id: RequestId)
  case responded(id: RequestId)

object progress:
  // every request is answered with its own id
  val everyRequestAnswered =
    owes(Fact.requested).until(Fact.responded)
  // and within two steps, while the worker polls
  val answeredPromptly =
    owes(Fact.requested).until(Fact.responded, within = 2) in fair(worker.poll)
```

The checker reports, per claim: `verified` with the longest open interval; `refuted` with the
payload, the kind (deadlock, end while owed, fair non-progress cycle, missed deadline) and the path
from a start through the opening step to the counterexample; `unknown` on a cut.

Errors: `until` outside an `owes` is the compile error "until closes an owes claim"; constructors
whose payload types differ, or a constructor of another Machine's fact type, do not compile; a
constructor with more than one parameter is refused at compile time (one payload, a case class if
several fields correlate); `within` under one is the existing `whenever` compile error; an `owes`
outside `progress` is refused where `whenever` is; a payload type with no `Finite` catalog fails at
the Machine as any state type does.

## Edge Cases & Constraints

- A step that records `open(k)` twice owes `k` once; the set has no multiplicity. A Model that counts
  attempts writes a monitor.
- An obligation opened in a start state's first step and a start that already owes are the same
  thing: nothing is owed at a start, since no step has recorded a fact.
- The owed set enlarges the explored graph by at most the catalog of `Set[K]`; the fixture bounds `K`
  to a handful of ids, and the budget rules of fn-190 R14 apply unchanged.
- `in only(...)` or `without(...)` that drops the closing action turns every opened obligation into a
  refutation, which is the intended reading, as Kettle's `without(user.depressButton)` is.
- A composition's claim reads member facts as `Fact.<member>(f)` (fn-190 R17, fn-193); whether `owes`
  takes a member's constructor is decided by the prototype and is otherwise refused with a message.

## Acceptance Criteria

- **R1:** `owes(open).until(close)` is written in `progress`, named after its `val`, with `within`
  optional on `until` and a Scenario after `in`; the two constructors share one payload type, or
  both are plain cases. Errors: `until` outside `owes`, differing payload types, another Machine's
  fact type, a multi-parameter constructor, `within` under one and `owes` outside `progress` are
  compile errors with the messages in API Contracts. [paraphrase]
- **R2:** The checker explores the owed set with the vars, keeps two paths with different owed sets
  apart, and reports per payload a deadlock or end while owed, a fair non-progress cycle on which it
  stays owed, or a missed deadline, each with the payload, the opening step and the path; a complete
  exploration with none is `verified` with the longest open interval; a cut is `unknown`. Errors: a
  `close` while not owed changes nothing and is no finding. [paraphrase]
- **R3:** A fixture test (a dispatcher with two request ids and a worker that may answer one id
  forever) shows: the claim under `fair(worker.answer)` verified; the same claim refuted on the path
  that answers only one id, naming the other; `within = 1` missing its deadline; and a hand-written
  `whenever(pending.nonEmpty): eventually(pending.isEmpty)` on the same Machine passing where `owes`
  refutes, as the reason the word exists. Errors: none beyond the compile refusals of R1, each pinned
  by a test. [inferred]
- **R4:** A design note in `model/design` records the prototype's spelling candidates, the reading
  on the mini feature's monitors, and the owner's decision; the probe stays beside it with a README
  and `run.sh`. Errors: the note is not optional; a spelling other than `owes(...).until(...)`
  revises R1's words and this spec records the revision. [user]
- **R5:** SEMANTICS §11, the framework README's progress text and `progress-notation.md`'s "When a
  claim gets a word" table describe the word, its semantics and why it qualifies; the block-form
  lint covers `owes` as it covers `whenever`. Errors: none beyond review. [inferred]
- **R6:** *(2026-10-18, one-language series: recut: the lift (fn-201) is unchanged, read `lift` for `export`)* The export and the IR are unchanged; a Model with an `owes` claim lifts as it did, with the
  claim left out as `whenever` is. Errors: a lifter that trips on the new declaration is a finding
  in this spec. [paraphrase]

## Boundaries

- No IR node, no Quint, TLA or P export of the claim, no Case generation: progress is a model check
  only, as fn-190 R13 and `progress-notation.md` decided. [paraphrase]
- *(2026-10-18: waits for fn-204's descriptor, or the fn-209 oracle on go, not fn-196's conformance replay.)* Conformance of a recorded Run against an `owes` claim waits for fn-196's conformance replay in the
  Scala checker; the owed-set semantics are written so that replay needs no new meaning. [inferred]
- No safety half: an unsolicited or repeated discharge is not refused by `owes`. [paraphrase]
- No multiplicity: one payload is owed once. [inferred]
- The Temporal models stay `.scala.disabled`; the mini feature's monitors are read, not rewritten.
  [user]
- The six samples are not rewritten unless the prototype finds one that reads better with the word;
  ReliableBroadcast's `someoneDelivered(current)` claim is the only candidate. [inferred]

## Decision Context

The owner asked which ideas of a formal-methods notation survey were useful; P's hot/cold observer
states, correlated by request id, were the one judged new, since the rest is in the DSL or was
decided against in `.plans/UMPIRE4_INSPIRE.md`. [paraphrase] The owner chose the spec. [user]
A word rather than a documented fold because the fold's correlation is the error the word prevents,
and because `progress-notation.md`'s rule admits a word that changes what is checked. [paraphrase]
Keyed by a fact's payload rather than by a state predicate because a fact is the step's own record of
what it did, so the obligation is opened and discharged where the Model says so and not inferred from
a set the Model also has to keep. [inferred] Checked as progress rather than as a monitor read at
ends because an obligation owed forever on a fair cycle is a liveness failure, which only the
progress checker with fairness can judge; a monitor would see only the path end. [inferred]
Rejected: P's full message-queue execution model (fn-190 made one effect one step); a `hot`/`cold`
state annotation on a phase enum, which keys the obligation to the entity's one phase and not to a
payload; a safety half in the same word, which would make `owes` two promises. [inferred]

2026-10-18, one-language series: R6 reads `lift` for `export` (fn-201). Conformance of a recorded Run against an `owes` claim is judged from fn-204's descriptor (progress marks and fairness classes), or by the native oracle if fn-209 says go; fn-196's conformance replay no longer exists. No `owes` emission in fn-205/fn-206/fn-207.

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | `owes(open).until(close)` in `progress`, named after its `val`, `within` optional, Scenario after `in`, shared payload type; six compile refusals | fn-197.2 | — |
| R2 | Owed set explored with the vars and in state identity; per-payload deadlock, end while owed, fair non-progress cycle, missed deadline, with payload, opening step and path; verified with longest open interval; unknown on a cut | fn-197.2 | — |
| R3 | Fixture test: verified under fairness, refuted naming the unanswered id, missed deadline, and the `whenever(pending.nonEmpty)` contrast; one test per compile refusal | fn-197.2 | — |
| R4 | Design note and probe in `model/design` with the spelling candidates, the mini-feature readings and the owner's decision; R1 revised if the spelling changes | fn-197.1 | — |
| R5 | SEMANTICS §11, the framework README and `progress-notation.md` describe the word; the block-form lint covers it | fn-197.3 | — |
| R6 | Export and IR unchanged; a Model with an `owes` claim lifts as before | fn-197.2 | — |
