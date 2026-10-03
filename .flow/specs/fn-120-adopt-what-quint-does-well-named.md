# Adopt what Quint does well: named choices, a model linter, an explorer and trace interchange

## Goal & Context
<!-- scope: business -->

A review of Quint, the specification language built on TLA, produced ten suggestions for Umpire on 2026-10-01. Its central lesson is that a specification should be a formally defined object that tools can read, and not code that happens to execute. Scala stays the authoring language; the question is how much of what an author means reaches the IR.

Umpire already has more of this than the review assumed. The lifter reads typed Scala trees and writes every step function, Property and monitor into the IR as an expression tree, and Go evaluates that tree. The IR is exported to Quint and P, and Quint's output is read back in its trace format. Bounded progress claims under weak fairness exist. `fn-113-clean-up-the-scala-model-layer-around` removes the last place where Scala code is the semantics.

So this spec adopts the four suggestions that add something Umpire lacks and that a model author would use next week, and it records a decision on the other six so they are not reargued.

| # | Suggestion | Decision |
| --- | --- | --- |
| 1 | Effect discipline: keep pure, state-reading, transition and temporal expressions apart | Adopt in a small form (Part E). Function signatures and the lifter's subset already keep them apart; what is missing is that the levels are named and each violation has a test |
| 2 | Explicit nondeterminism | **Adopt (Part A)** |
| 3 | Reify the semantics instead of evaluating opaque Scala | Already the design. The lifter does it, and fn-113 retires the Scala evaluator |
| 4 | TLA-style temporal operators | Later. Bounded leads-to under weak fairness exists. The rule is recorded now: any temporal operator Umpire adds takes its meaning from TLA |
| 5 | Scenarios as constraints over executions (`anyOrder`, `repeat`) | Later, with its own spec, when a Query needs an ordering that is neither pinned nor free |
| 6 | A Query states the question, a backend answers it | Later, with its own spec. Part D removes the main obstacle |
| 7 | A standard trace format | **Adopt (Part D)** |
| 8 | Reproducible failure artifacts with seeds | Mostly covered. The Go search is deterministic and has no seed, recorded Runs replay, and fn-107 task 11 owns reproduction and minimization |
| 9 | A REPL over the model | **Adopt (Part C)**, as an explorer over the IR |
| 10 | A linter for model quality | **Adopt (Part B)** |

The reader this spec serves is the model author: the person who wants to know why an action is disabled, whether a branch they wrote is ever taken, and whether a Property they declared is ever asked.

## Architecture & Data Models
<!-- scope: technical -->

Everything here is read from the IR by the Go tooling. Part A adds to the DSL, the lifter and the IR; Parts B, C and D add Go only.

**Part A. Named choices.** Today a step function that can go two ways returns a list of two steps, and nothing says why there are two. A `choose` construct names each alternative. The IR records the name on each result. Each alternative is then something tools can count, export and report.

**Part B. Model lint.** One Go command reads the IR and reports findings about model quality. Several checks exist already as side effects of other checks (a stuck state, a Property a verify never exercised, a Query with no realization). They become one list of named findings with a location, and the gate fails on a new one.

**Part C. Explorer.** One Go command steps through a machine from the IR: the classes enabled in a state, the results of taking one, and why a class is disabled. "Why" is the list of branch decisions the evaluator took in the step function, each with its Scala position. An interactive loop is a thin shell over the same commands.

**Part D. Trace interchange.** An Umpire witness converts to and from ITF, the trace format Quint, Apalache and TLC tooling share. The export module already reads ITF from Quint. With both directions a trace found by another checker can be lowered through a realization into a Case, and an Umpire witness opens in existing trace viewers.

**Part E. Named semantic levels.** `SEMANTICS.md` names the levels an expression can be at (a pure computation, a reading of one state, a step from one state to the next, a reading of a step, a claim over a path) and which declaration takes which. The lifter refuses an expression at the wrong level at its line.

## API Contracts
<!-- scope: technical -->

A sketch of `choose`. The task that builds it settles the spelling and records it here.

```scala
def admitted(s: AdmissionState): Steps = choose(
  committed -> accept(s.copy(phase = started, answer = owed), statusStarted, attemptAdmitted),
  commitFailed -> stay(s).recording(admissionCommitFailed)
    .because("the durable update fails: nothing is admitted and the message stays deliverable")
)
```

The alternatives' names are values the author declares once and tools refer to. A step function with one result needs no `choose`.

The lint and explorer commands, by example. Final names follow the tooling's conventions.

```text
umpire lint model/ir/activity.json
  activityProtocol  unreachable-case   Phase.backingOff is never reached from the start   Model.scala:212
  activityProtocol  unasked-property   retryCompletes is named by no Query                Properties.scala:41

umpire explore model/ir/activity.json activityProtocol
> enabled
  attemptStart, control-pause, control-requestCancel, control-terminate, scheduleToStart
> why attemptResult-completed
  disabled: phase != started and phase != cancelRequested   Model.scala:263
```

## Edge Cases & Constraints
<!-- scope: technical -->

- **Named choices change no behavior.** Tables, Definition IDs, fingerprints and Query answers are the same before and after. The IR gains a field, which is a schema change, and Case bytes change only if a name is carried into a Case, which this spec does not do.
- **Unnamed lists remain legal for one release of the spec.** A step function that returns several results without `choose` still lifts while the Models are converted. At the closing task the lifter refuses it, so that every branching in a Model is intentional and named.
- **`because` stays.** It is prose for a reader. A choice's name is an identifier for tools.
- **Lint findings need a way to be accepted.** Some findings are intended: a faulty control design has a violated Property on purpose. An accepted finding is recorded in a checked-in file beside the IR with its reason, and the gate fails on a finding that is neither fixed nor accepted, and on an acceptance that no longer matches anything.
- **Lint reads the state domain with care.** A machine's state type is a product of fields, and most combinations are unreachable by design. The check reports an enum case or a field value that no reachable state holds, and never a combination.
- **"Why" must be bounded.** A step function's evaluation trace can be long. The explorer prints the decisions that led to the empty result, and no more.
- **ITF carries less than an Umpire trace.** It has states and no Definition IDs or outcomes. The conversion states what it drops on export and what it has to recompute on import, and an imported trace is replayed against the machine before it is accepted.
- **Order.** This spec follows fn-114, so that `choose` is added to a DSL that has settled and is rolled out to all Models once. The example of fn-119 is written after it and uses it.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The DSL has a construct that names the alternatives of a nondeterministic step, the lifter lifts it, and the IR records the name on each result. Errors: two alternatives with one name, and a `choose` with one alternative, are refused at their line.
- **R2:** Every step function in the Models that can return more than one result uses the construct, and at the closing task the lifter refuses an unnamed list of several results at its line. Errors: a branching whose alternatives nobody can name is listed for the owner.
- **R3:** The baseline goldens (`tools/umpire/model/testdata/migration`, `tools/umpire/lower/testdata/migration`) pass across Part A. Errors: any change to a table, a Definition ID, a fingerprint, a Query answer or a Case stops the task.
- **R4:** The Quint export writes a named choice as Quint's own nondeterministic choice with the names kept, and the agreement checks still pass. Errors: if Quint cannot express a choice the way the IR states it, the export says which and the finding is recorded.
- **R5:** A lint command reads an IR file and reports each finding with a stable kind, the machine, a message and a Scala position. It has at least these kinds: an enum case or field value no reachable state holds; an action class with no enabled row; an outcome or fact no step produces; a named choice no reachable state takes; a Property no Query names; a verify Query whose Property never fired; a fact with no evidence in a realization that covers its machine; an action no realization performs; a find Query with no realization; a refining machine whose refinement is not checked; an observation nothing reads. Errors: a malformed IR file is reported by the reader as today and produces no lint findings.
- **R6:** Each lint kind has a fixture that triggers it and one that does not. Errors: a kind with no fixture is not shipped.
- **R7:** The model gate runs lint over every checked-in IR file and fails on a finding that is neither fixed nor recorded as accepted with a reason, and on an acceptance that matches no finding. The done summary lists the findings the first run produced and what was done about each.
- **R8:** An explorer command lists a machine's start states, the classes enabled in a given state, and the results of taking a class, each with its choice name, outcome, facts and next state. Errors: an unknown machine, state or class is refused with the nearest valid names.
- **R9:** The explorer answers why a class is disabled in a state with the branch decisions that produced the empty result, each at its Scala position. Errors: a class that is enabled says so; a class disabled because the action is not bound says that.
- **R10:** The explorer runs as single commands and as an interactive session over the same commands, and both are covered by tests on a fixture Model.
- **R11:** An Umpire witness exports to ITF and an ITF trace imports to an Umpire trace. A witness exported and imported is equal to the original. An imported trace is replayed against its machine and refused at the first step the machine cannot take. Errors: the documentation lists what ITF does not carry and how import recomputes it.
- **R12:** One trace produced by Quint from an exported Model is imported and lowered through that Model's realization into a Case that Testpilot's preparation admits. Errors: if no realization can place the trace, the located reason is reported as for any Query.
- **R13:** `SEMANTICS.md` names the semantic levels and which declaration takes which, and states that any temporal operator Umpire adds takes its meaning from TLA. The lifter has a refusal fixture for an expression at the wrong level for each declaration kind. Errors: a level the Scala types already make impossible to violate is listed as such instead of given a fixture.
- **R14:** The model gate, `make lint-model`, the Go tests of the Umpire tooling and `make lint-code-fast` pass at the closing task. The model's README mentions lint and the explorer where it describes how an author works (no error surface beyond the gates).

## Boundaries
<!-- scope: business -->

- No new temporal operators, no strong fairness, no unbounded liveness.
- No Scenario combinators.
- No Query answered by Quint, Apalache or TLC. The exports keep checking agreement with Go.
- No effect types in the DSL. The levels are named and enforced by the lifter.
- No Quint syntax and no change of authoring language.
- No lint kind that needs mutation of the Model or a search per finding (a Property insensitive to mutations, a Scenario that pins an unnecessary ordering). They are candidates for later.
- No random simulation, and so no seeds.
- A choice's name is not carried into a Case or bound to a fault in a realization here. That link is the natural next step and is left for the spec that needs it.

## Decision Context
<!-- scope: both — conditionally substructured -->

**Why these four.** Named choices are the one place where an author's intent is lost between Scala and the IR today, and every later tool wants it: coverage, the Quint export, documentation, fault realization. Lint and the explorer are cheap, because the Go reader already computes everything they print, and they change the daily experience of writing a Model more than any backend would. Trace interchange is small and is the piece a later "backend answers a Query" spec cannot do without.

**Why not effect types.** Quint needs an effect system because one expression language covers everything. In Umpire a step function takes the state as an argument and returns the next ones, a Property takes a step and returns a Boolean, and the lifter refuses whatever is outside its subset. The signatures already do the work. Naming the levels and testing the refusals gives the clarity without new types in every Model.

**Why temporal operators, Scenario combinators and backend answering wait.** Each is a real extension of what a Model can say or how it is checked, each needs IR and search changes, and none has a Query waiting for it. Each gets its own spec when one does. The decision table keeps them on record.

**Why accepted findings live in a file.** Putting an acceptance into the Model would need another IR field and would mix quality bookkeeping into the specification. A file beside the IR is reviewable and is checked by the gate both ways.

## Parked unknowns

- Whether a choice's name is declared as a value of its own or taken from an enum the machine already has. The first task settles it against what reads best in the admission and queue Models.
- Whether the explorer is a subcommand of an existing Umpire tool or a command of its own. The module map of fn-115 decides.
