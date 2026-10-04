# Adopt what Quint does well: named choices, a model linter and an explorer

## Goal & Context
<!-- scope: business -->

A review of Quint, the specification language built on TLA, produced ten suggestions for Umpire on 2026-10-01. Its central lesson is that a specification should be a formally defined object that tools can read, and not code that happens to execute. Scala stays the authoring language; the question is how much of what an author means reaches the IR.

Umpire already has more of this than the review assumed. The lifter reads typed Scala trees and writes every step function, Property and monitor into the IR as an expression tree, and Go evaluates that tree. The IR is exported to Quint and P, and each export is held to Go's reading of the same Model. Bounded progress claims under weak fairness exist. `fn-113-clean-up-the-scala-model-layer-around` removes the last place where Scala code is the semantics.

So this spec adopts the three suggestions that add something Umpire lacks and that a model author would use next week, and it records a decision on the other seven so they are not reargued.

| # | Suggestion | Decision |
| --- | --- | --- |
| 1 | Effect discipline: keep pure, state-reading, transition and temporal expressions apart | Adopt in a small form (Part E). Function signatures and the lifter's subset already keep them apart; what is missing is that the levels are named and each violation has a test |
| 2 | Explicit nondeterminism | **Adopt (Part A)** |
| 3 | Reify the semantics instead of evaluating opaque Scala | Already the design. The lifter does it, and fn-113 retires the Scala evaluator |
| 4 | TLA-style temporal operators | Later. Bounded leads-to under weak fairness exists. The rule is recorded now: any temporal operator Umpire adds takes its meaning from TLA |
| 5 | Scenarios as constraints over executions (`anyOrder`, `repeat`) | Later, with its own spec, when a Query needs an ordering that is neither pinned nor free |
| 6 | A Query states the question, a backend answers it | Later, with its own spec |
| 7 | A standard trace format | Not adopted. The owner withdrew ITF interchange on 2026-10-04; Umpire's own trace stays the only witness format |
| 8 | Reproducible failure artifacts with seeds | Mostly covered. The Go search is deterministic and has no seed, recorded Runs replay, and fn-107 task 11 owns reproduction and minimization |
| 9 | A REPL over the model | **Adopt (Part C)**, as an explorer over the IR |
| 10 | A linter for model quality | **Adopt (Part B)** |

The reader this spec serves is the model author: the person who wants to know why an action is disabled, whether a branch they wrote is ever taken, and whether a Property they declared is ever asked.

## Architecture & Data Models
<!-- scope: technical -->

Everything here is read from the IR by the Go tooling. Part A adds to the DSL, the lifter and the IR; Parts B, C and D add Go only.

**Part A. Named choices.** Today a step function that can go two ways returns a list of two steps, and nothing says why there are two. A `choose` construct names each alternative. The IR records the name on each result. Each alternative is then something tools can count, export and report. Settle its behavior against fn-112's `accept` and `stay` helpers and captured-name rules before either the standalone feature or other Models rewrite branches. The number of named alternatives is not a factor in fn-112's Query.total: alternatives are results of one action class.

**Part B. Model lint.** One Go command reads the IR and reports findings about model quality. Several checks exist already as side effects of other checks (a stuck state, a Property a verify never exercised, a Query with no realization). They become one list of named findings with a location, and the gate fails on a new one.

Lint also prints a coverage summary, after Ivy's coverage check: a structural count, not a proof. Per IR file and per machine or composition, it gives counts with denominators, taken from the same reads that produce the findings. Each count is the population and the satisfied part of one finding kind, so the difference is that kind's findings; a count with no finding kind is a smell and is not printed. The summary is informational and deterministic. It does not restate the other reports: the Case manifest's standing says what lowering did with each Query, a Case's Known Gaps say what that Case cannot check, and the exploration ledger (`tools/umpire/explore/bridge.go`) says what one campaign covered. Lint reads the IR, the reader's `Check` and the linked API descriptors, and no Case, manifest or ledger. Where a finding overlaps one of them (a find Query with no realization, the manifest's `no-realization`), both come from one Go function.

One finding kind is about the API surface a realization reads. An enum value of a Temporal API field, or a member of its oneof, is in scope only where a realization's poll condition or Run Event guard tests that field. The value is unmodeled when no such test maps it to an evidence kind, and so to a Model fact. The field's full set of values comes from the descriptor linked into Go; the zero `*_UNSPECIFIED` value is not counted.

**Part C. Explorer.** One Go command steps through a machine from the IR: the classes enabled in a state, the results of taking one, and why a class is disabled. "Why" is the list of branch decisions the evaluator took in the step function, each with its Scala position. An interactive loop is a thin shell over the same commands.

Lint and the explorer are commands under `tools/umpire/cmd/`, the one place that may import both the reader and `tools/umpire/lower` (`tools/umpire/model/ownership_test.go`). The explorer needs only the reader. `tools/umpire/explore`, which fn-115's module map reserved for it, is the campaign-exploration package and is not used.

**Part E. Named semantic levels.** `SEMANTICS.md` names the levels an expression can be at (a pure computation, a reading of one state, a step from one state to the next, a reading of a step, a claim over a path) and which declaration takes which. The lifter refuses an expression at the wrong level at its line.

## API Contracts
<!-- scope: technical -->

The spelling fn-120.1 settled (model/umpire/Machine.scala; fixture model/lifter/testdata/lifts/Choices.scala):

```scala
val committed = choice
val redelivered = choice

def admitted(s: AdmissionState): List[AdmissionStep] = choose(
  committed -> accept(s.copy(phase = started, message = empty), statusStarted, attemptAdmitted),
  redelivered -> accept(s.copy(phase = started, message = redelivery), statusStarted, attemptAdmitted)
    .because("the channel may deliver the message again")
)
```

A name is a token of its own, declared once by the val that names it (`val committed = choice`), as an input token is; the IR name is the val's simple name. It is not an enum case: an enum used only for names would be lifted as an IR type the Model does not otherwise need, and the names of different step functions are not one closed set. `choose` is core (model/umpire/Machine.scala, lifted beside `because` in model/lifter/Expressions.scala); no sugar spelling over it was added. Each alternative is one step written out (`accept(...)`, `stay(s)`, `List(Step(...))`, optionally `.because(...)`); `choose` takes at least two alternatives, so a one-alternative choose does not compile, and the lifter refuses a name used twice, an alternative that is not one step and a token no val names, at their lines. The IR records each name on the step record's construct (`Construct.choice`, model/SEMANTICS.md "Named choices"). A step function with one result needs no `choose`. Fn-120.2 settled two more rules: an alternative may also call a function of the lifted sources that gives at most one step in each branch (`forged -> Protocol.completeStep(s, Resolution.succeeded)`), lifted as a call of the copy `<function>$<choice>` whose every step carries the name, and not taken where the function gives none; and the lifter refuses, at its line, a step list of several steps written without `choose` (`List(Step(...), Step(...))`, steps joined with `++`).

**Quint result contract.** The pure exported step function still returns an ordered list of every result, including inert choice names on their records. The checker action selects a class and then a result index nondeterministically at its action boundary. A `choose` does not select a branch inside the pure function or discard/reorder alternatives. Table rows and agreement checks compare the same ordered results; names are reportable metadata only.

The lint and explorer commands, by example. Final names follow the tooling's conventions.

```text
umpire lint model/ir/activity.json
  activityProtocol  unreachable-case   Phase.backingOff is never reached from the start   Model.scala:212
  activityProtocol  unasked-property   retryCompletes is named by no Query                Properties.scala:41
  activityProtocol  unmodeled-api-value  ActivityExecutionInfo.status RUNNING is mapped by no poll  Realization.scala:298
coverage activityProtocol
  properties          12 declared, 11 named by a Query
  verify Queries       4, 4 whose Property fired
  non-system actions   6, 6 performed by a realization
  facts               14, 14 with evidence
  named choices        5, 5 taken by a reachable state
  refinements          1 declared, 1 read through by a Query
  observations         2, 2 read
  API values           7 tested, 6 mapped to a fact

umpire explore model/ir/activity.json activityProtocol
> enabled
  attemptStart, control-pause, control-requestCancel, control-terminate, scheduleToStart
> why attemptResult-completed
  disabled: phase != started and phase != cancelRequested   Model.scala:263
```

## Edge Cases & Constraints
<!-- scope: technical -->

- **Named choices change no behavior.** Tables, Definition IDs, fingerprints, Query answers, search identity and Case bytes are the same before and after. The IR gains only inert choice-name metadata for existing alternatives; branch count/order and transition results remain exact. The metadata does not enter a Case or any semantic fingerprint. The IR bindings and linked API jar are regenerated, and historical descriptor/wire coverage proves new current fields without rewriting historical bytes.
- **Unnamed lists remain legal for one release of the spec.** A step function that returns several results without `choose` still lifts while the Models are converted. At the closing task the lifter refuses it, so that every branching in a Model is intentional and named.
- **`because` stays.** It is prose for a reader. A choice's name is an identifier for tools.
- **Lint findings need a way to be accepted.** Some findings are intended: a faulty control design has a violated Property on purpose. An accepted finding is recorded in a checked-in file beside the IR with its reason, and the gate fails on a finding that is neither fixed nor accepted, and on an acceptance that no longer matches anything.
- **Lint reads the state domain with care.** A machine's state type is a product of fields, and most combinations are unreachable by design. The check reports an enum case or a field value that no reachable state holds, and never a combination.
- **A count is not a threshold.** The gate fails on findings and stale acceptances, never on a count or its ratio. An accepted finding stays in its count's difference; the summary does not hide it.
- **API values are counted only where a realization tests them.** A request field a command writes, a message an observation keeps whole, and a field no condition tests are out of scope. A value a poll waits through, such as `ACTIVITY_EXECUTION_STATUS_RUNNING` in the standalone activity, is a finding; the author maps it or accepts it with a reason. A history read's `HistoryEvent` attributes oneof is not counted: the read sees every event of the workflow, and the IR does not record which event types a machine's operations can produce.
- **"Why" must be bounded.** A step function's evaluation trace can be long. The explorer prints the decisions that led to the empty result, and no more.
- **Order.** Part A's mechanism starts only after fn-112.11 finishes Query.total schema, binding and linked API jar regeneration; fn-112.11 itself follows fn-112.3, .4 and .5. This serializes the two schema edits and makes fn-120.1's choice compatibility check run against a descriptor that already contains Query.total. Fn-120.1 precedes fn-112.6 and fn-114 branching rewrites. The conductor checks these cross-spec completion gates because flowctl stores task dependencies only within one spec. Conversion to named choices is part of those rewrites. Only after fn-114 has migrated every consumer does Part A refuse unnamed branching. Lint inventories run after fn-114's Scala-owned roots are stable. Lint and explorer work may overlap fn-118's waiting changes once the schema and reader contracts are settled. Fn-119's example Model waits for this spec's full closure.

| Phase handoff | Completion gate | Next work |
| --- | --- | --- |
| Query.total foundation | fn-112.11 done after fn-112.3, .4 and .5 | fn-120.1 may add named-choice schema and final author syntax |
| Named-choice foundation | fn-120.1 done | fn-112.6 and fn-114 may convert branching declarations |
| Consumer rollout | fn-114 closed | fn-120.2 may refuse unnamed multi-result branches; lint uses stable roots |

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The DSL has a construct that names the alternatives of a nondeterministic step, the lifter lifts it, and the IR records the name on each result. Errors: two alternatives with one name, and a `choose` with one alternative, are refused at their line.
- **R2:** Every step function in the Models that can return more than one result uses the construct, and at the closing task the lifter refuses an unnamed list of several results at its line. Errors: a branching whose alternatives nobody can name is listed for the owner.
- **R3:** The baseline goldens (`tools/umpire/model/testdata/migration`, `tools/umpire/lower/testdata/migration`) pass across Part A. Errors: any change beyond inert names on existing result alternatives, including a branch count/order, table, Definition ID, fingerprint, Query answer, exploration identity or Case, stops the task.
- **R4:** Quint's pure step function retains every named alternative as an ordered result-list entry with its inert name; the checker action nondeterministically selects an index from that list. Agreement rows, branch count/order and behavior remain exact. Errors: if Quint cannot represent a name without selecting inside the pure function or changing the result list, the export reports the unsupported choice and its location.
- **R5:** A lint command reads an IR file and reports each finding with a stable kind, the machine, a message and a Scala position. It has at least these kinds: an enum case or field value no reachable state holds; an action class with no enabled row; an outcome or fact no step produces; a named choice no reachable state takes; a Property no Query names; a verify Query whose Property never fired; a fact with no evidence in a realization that covers its machine; a non-system action (party other than `system`) of a machine a realization covers that no performance binds and no activity script starts with; a find Query with no realization; a refinement no Query reads through (`Query.through`); an observation nothing reads; an enum value or oneof member of a Temporal API field that a realization's poll condition or Run Event guard tests, which no such test maps to an evidence kind (the zero value and the `HistoryEvent` attributes oneof excluded); and the specification-hole kinds of `.plans/MODALITIES.md`: H1 `disabled-by-default` (a state-class pair whose empty result comes from a wildcard `match` arm or a guard naming no state field), H2 `silent-rejection` (a disabled pair of a party action, neither `timer` nor `internal`, in a reachable non-end state), H3 `unconstrained-result` (an enabled pair of a class no same-step Property names, no transition Property or monitor of the machine or of its product through the refinement constrains, and no progress claim reaches), H4 `witness-only` (a same-step Property asked only by `find` Queries over pinned Scenarios) and, behind a flag and off by default, H5 `must-not-pinned` (a disabled pair of a system action no `never`/transition Property pins). Hole kinds aggregate by class and by the value of the state record's first enum-typed field (`phase` in the activity; by class alone in a machine without one), never per state, and are computed by joining the table, the R9 decision trace and the claim index; no second evaluator. Lint prints the per-operation modality table of a machine (the table grouped by class and by the named predicates the step function evaluated, each cell MAY with its results, MUST NOT with its guard, or `?` for H1/H2, with the Properties and laws that pin it). Errors: a malformed IR file is reported by the reader as today and produces no lint findings.
- **R6:** Each lint kind has a fixture that triggers it and one that does not. The unmodeled-API-value kind also has a fixture where a request field the realization only writes holds an unmapped value and is not reported. Errors: a kind with no fixture is not shipped.
- **R7:** The model gate runs lint over every checked-in IR file and fails on a finding that is neither fixed nor recorded as accepted with a reason, and on an acceptance that matches no finding. It never fails on an R15 count. The done summary lists the findings the first run produced and what was done about each, and the first run's coverage summary.
- **R8:** An explorer command lists a machine's start states, the classes enabled in a given state, and the results of taking a class, each with its choice name, outcome, facts and next state; `state <key>` prints the per-state modality report (each class as MAY with its results and the Properties that pin them, MUST NOT with the guard at its line, or `?` for an H1/H2 hole) and `rules <class>` the per-operation table grouped by the predicates the decision trace called and by the machine's capability parameters where fn-122 declares them, with gap, overlap and conflict lines for the guards' coverage of the state catalog. Both views are produced from the same table, trace and claim index lint uses. Errors: an unknown machine, state or class is refused with the nearest valid names; a fixture Model with a wildcard arm shows `?`.
- **R9:** The explorer answers why a class is disabled in a state with the branch decisions that produced the empty result, each at its Scala position, and says whether the last decision was a wildcard arm. Errors: a class that is enabled says so; a class disabled because the action is not bound says that.
- **R10:** The explorer runs as single commands and as an interactive session over the same commands, and both are covered by tests on a fixture Model.
- **R13:** `SEMANTICS.md` names the semantic levels and which declaration takes which, states that any temporal operator Umpire adds takes its meaning from TLA, and carries one paragraph "Modalities" under Machines: a row is permission with fixed results; a disabled pair is prohibition for a system action and silence for a party action; obligations are same-step Properties, progress claims and fairness; refinement narrows permission and does not by itself preserve obligation. The lifter has a refusal fixture for an expression at the wrong level for each declaration kind. Errors: a level the Scala types already make impossible to violate is listed as such instead of given a fixture.
- **R14:** The model gate, `make lint-model`, the Go tests of the Umpire tooling and `make lint-code-fast` pass at the closing task. The model's README mentions lint and the explorer where it describes how an author works (no error surface beyond the gates).
- **R15:** The lint command prints a coverage summary per IR file and machine or composition, with at least these counts: Properties declared and named by a Query; verify Queries and those whose Property fired; non-system actions of a realized machine and those a realization performs; facts and those with evidence; refinements declared and those a Query reads through; named choices and those a reachable state takes; observations and those read; tested API enum values and oneof members and those mapped to a fact. Each count is the population and satisfied part of one R5 kind, computed by the same function. Output is byte-stable for one IR file; a golden test pins the summary over the lint fixtures, and the model gate prints it for every checked-in IR file without comparing it to anything. Errors: a count whose difference disagrees with its kind's findings fails the test; a count with no R5 kind is not added.

## Boundaries
<!-- scope: business -->

- No new temporal operators, no strong fairness, no unbounded liveness.
- No Scenario combinators.
- No Query answered by Quint, Apalache or TLC. The exports keep checking agreement with Go.
- No effect types in the DSL. The levels are named and enforced by the lifter.
- No Quint syntax and no change of authoring language.
- No lint kind that needs mutation of the Model or a search per finding (a Property insensitive to mutations, a Scenario that pins an unnecessary ordering). They are candidates for later.
- No random simulation, and so no seeds.
- No trace interchange. Umpire witnesses are neither exported to nor imported from ITF, and no external checker's trace is lowered into a Case. The existing Quint agreement keeps reading Quint's own output as it does today.
- No fault-specific lint kind. Whether a fault is realizable or model-only belongs to the later spec that declares typed faults as environment actions, after fn-112 and fn-120.1. R15's non-system action count, which includes party `fault`, is its precursor.
- A choice's name is not carried into a Case or bound to a fault in a realization here. That link is the natural next step and is left for the spec that needs it.

## Decision Context
<!-- scope: both — conditionally substructured -->

**Why these three.** Named choices are the one place where an author's intent is lost between Scala and the IR today, and every later tool wants it: coverage, the Quint export, documentation, fault realization. Lint and the explorer are cheap, because the Go reader already computes everything they print, and they change the daily experience of writing a Model more than any backend would.

**Why not effect types.** Quint needs an effect system because one expression language covers everything. In Umpire a step function takes the state as an argument and returns the next ones, a Property takes a step and returns a Boolean, and the lifter refuses whatever is outside its subset. The signatures already do the work. Naming the levels and testing the refusals gives the clarity without new types in every Model.

**Why temporal operators, Scenario combinators and backend answering wait.** Each is a real extension of what a Model can say or how it is checked, each needs IR and search changes, and none has a Query waiting for it. Each gets its own spec when one does. The decision table keeps them on record.

**Why accepted findings live in a file.** Putting an acceptance into the Model would need another IR field and would mix quality bookkeeping into the specification. A file beside the IR is reviewable and is checked by the gate both ways.

- **Amended 2026-10-03 (owner).** The owner asked that CI show "how do we know we forgot something" as counts with denominators, and not only as findings. Ivy's coverage check is the model: it reports an assertion that no isolate guarantees, by structure and without a proof ([Ivy, Coverage](https://microsoft.github.io/ivy/examples/specification.html); `ivy/ivy_isolate.py` at kenmcmil/ivy 8858a02, lines 2010-2023). R15 adds the summary as a view of the R5 kinds, so it gets no gate and no second reporting system. R5 gains one kind for API values a realization tests but no Model fact maps. Fn-117 typed every field a realization reads, so the IR names the field and the value, and the Go descriptor supplies the rest; no IR field is added. A fault kind is left to the typed-fault spec.

- **Amended 2026-10-04 (owner).** ITF interchange is withdrawn: Part D, R11, R12 and their contracts are removed, and fn-120.5 is only the closing task. Part letters are kept, and R11 and R12 are retired, not reused, so references stay stable. No Query needs an external witness, and a lossless ITF extension would be a second witness format to keep in step with Umpire's trace.
- **Modalities, 2026-10-03.** `.plans/MODALITIES.md` found that MAY, MUST and MUST NOT are already in the IR (rows, non-rows and the safety claims that pin them, postconditions, progress and fairness) and that the valuable part is the report. No `may`/`must`/`mustNot` author words (fn-112's Decision Context records the rejection); the modalities are the vocabulary of the generated views: lint prints the per-operation table and the hole kinds H1-H5 (R5, task 3), the explorer the per-state view and `rules <class>` (R8, task 4), and SEMANTICS names them (R13). A rules table is a view, never authored; its guards are the named status sets fn-112 R10 gives the vocabulary objects and the capability parameters fn-122 declares, and fn-122's laws are rendered as the cells they pin. The activity IR's first lint run (measured 2026-10-03: 7 H1 lines, 7 H2 lines, 15 H3 classes, 8 H4 Properties) is fixed or accepted with a reason per fn-120 R7; fn-112.6 already replaces the wildcard arms, and the seven server-rejected pause/unpause pairs and the two witness-only Properties it leaves are accepted findings with fn-112's freeze as the reason until a later spec takes them.
- **Modal refinement (#ZOOM), deferred.** Machines 6 checks the may half of modal refinement (every protocol row is carried by a product row or is a stutter). The must half, that every product MAY is realized by some protocol row from every state mapping to its source, and that a product progress claim read through the refinement holds as a protocol progress claim under the protocol's own fairness, is not checked and today holds by luck. Both are checks over the two tables and the map, with no search, and belong to the #ZOOM spec, not to this one; fn-122's table view marks a product law the protocol has not been checked against as inherited until then.
- **`choose` is core, not sugar (fn-112's core/sugar file rule, 2026-10-03).** A `choose` names alternatives the IR records on each result, which no core form expresses, so it lives beside `steps` in the core DSL and its lifting beside the step lifting; any convenience spelling over it (an infix form, an abbreviation) goes to `model/umpire/Syntax.scala` with its matching in `model/lifter/Syntax.scala` and an IR-equality fixture, and task 1 records which it is if it adds one.

**Maintainability (plan review):** duplication - task 1 owns the single choice-name-only baseline allowance and harness; task 2 reuses it for retirement without defining a second allowance. Structure - none identified.

## Parked unknowns

- Whether a choice's name is declared as a value of its own or taken from an enum the machine already has. The first task settles it against what reads best in the admission and queue Models.
- Whether a history read's event types can be scoped to a machine, so the `HistoryEvent` attributes oneof can be counted. The IR does not record which event types a machine's operations produce; recording it would be an IR change outside Part B.

## Quick commands

```bash
make umpire-check-model
go test -tags test_dep ./tools/umpire/...
```

## Requirement coverage

| Req | Task(s) |
| --- | --- |
| R1 | fn-120.1 |
| R2 | fn-120.2 |
| R3 | fn-120.1, fn-120.2 |
| R4 | fn-120.1 |
| R5 | fn-120.3 |
| R6 | fn-120.3 |
| R7 | fn-120.3 |
| R8 | fn-120.4 |
| R9 | fn-120.3, fn-120.4 |
| R10 | fn-120.4 |
| R13 | fn-120.4 |
| R14 | fn-120.5 |
| R15 | fn-120.3 |

