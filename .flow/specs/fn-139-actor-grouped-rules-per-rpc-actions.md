# Actor-grouped rules, per-RPC actions, shared rejections

> HTML render lens (local): open `.flow/artifacts/fn-139-actor-grouped-rules-per-rpc-actions/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (turn 1): "is this the clearest way we can express that?"
> user (turn 2): "can you propose a few alternatives"
> user (turn 3, part 1): "enum Failure derives Finite: case final, retryable / enum AttemptResult derives Finite: case completed; case failed(kind: Failure); case canceled" "I like that very much"
> user (turn 3, part 2): "> C. Group the blocks by what they mean, not by enum order" "yes!"
> user (turn 3, part 3): "apart from those; are there entirely new/different ways to express this. think outside the box"
> user (turn 5): "B2_StateFirst.scala reads the cleanist to me"
> user (turn 7, part 1): "I like this on(worker.poll) { } as it mirrors the planned `effect {}` sytnadx"
> user (turn 7, part 2): "and how we can have multiple in(started) ~> inside the block for grouping"
> user (turn 8, part 1): "> on(client.control(Control.pause)) still reads quite clunky, though"
> user (turn 8, part 2): "> something like answer, when or with" "I like when"
> user (turn 8, part 3): "also; are we able to represent when sth invalid happens; eg we should be able to express that a certain client action is invalid at certain states and what kind of error (loosing following the RPC error types that we can map to actual RPC types in realization - shared across all machines of course)"
> user (turn 9): "I like grouping it by \"actor\" where there's one on block for client and one for worker (and maybe more later)."
> user (turn 10): "why not use `from` to define the actor; and then `on` instead of when for the action."
> user (turn 11): "and then `when` instead of `in`"
> user (turn 12): "> on(respond(AttemptResult.completed))(when(states.held) ~> effects.complete) still looks confusing, too dense"
> user (turn 13): "yes 1 and 2"
> user (turn 14): "approved; let's write or update flow next spec"
> user (turn 14, selected): "Keep one spec"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 50% [paraphrase], 30% [user], 20% [inferred] -->

A Model's `rules` say, for each action, where it fires and what it does. Today each block is keyed by an action or one class of it, and the owner finds the result hard to read: `on(worker.respond(AttemptResult.failed(true)))` hides what the Boolean means, `on(client.control(Control.pause))` "still reads quite clunky", and the single-case form `on(x)(in(y) ~> z)` "still looks confusing, too dense". The rules for one actor are scattered across sibling blocks, and the meaning that groups them lives only in comments.

The owner chose, over several rounds of prototypes, a form that reads as a sentence: from an actor, on an action, when in these phases, do this. Rules are grouped by actor, "one on block for client and one for worker (and maybe more later)", each action's cases sit in one block "for grouping", and the block form "mirrors the planned `effect {}` syntax" (fn-135).

Two changes to the standalone activity's signature come with it. Each RPC becomes its own action, for the client's controls and the worker's answers alike, so no rule names a class of an action. And the worker's retryable flag becomes a named enum.

The owner also wants to "express that a certain client action is invalid at certain states and what kind of error", "loosely following the RPC error types that we can map to actual RPC types in realization", "shared across all machines".

## Architecture & Data Models
<!-- scope: technical -->

**Three levels in `rules`.** [paraphrase] `from(declarer) { … }` holds the rules of the actions one object declares (`client`, `worker`, the worker process, `timers`, `deadline`). Inside it, `on(action) { … }` holds the cases of one action or one class of it, as `on` does today. Inside that, each case is `when(phases…) ~> effect`, `when(set) ~> effect`, `when(…).where(g) ~> effect` or `always ~> effect`. [inferred] `from` only groups and scopes names: it adds nothing to what a rule means, and an `on` outside any `from` keeps working, so machines convert one at a time.

**How `from` scopes bare names.** Scala 3 has no receiver scoping, and a context function brings only givens into scope, so `from(declarer)` cannot put `declarer`'s members in scope by itself without a macro. The author writes `import declarer.*` as the first statement of the block: `from(client) { import client.*; on(pause) { … } }`, the import on its own line. A planning probe on Scala 3.9.0 settled it. The import compiles under the Models' options, `codeOf` names the action `…client.pause`, whose last segment the overlap message uses, and an inner wildcard import shadows same-package members declared in other files. A top-level definition of the same name in the same file makes the reference ambiguous, a compile error the author fixes by qualifying. `from` adds no `inline`, so the author-surface rule keeps `on` as its one exception. The lifter accepts the import as the block's first statement only.

**`when` replaces the rule-case `in`.** [user] The case form is renamed from `in(...)` to `when(...)`. [paraphrase] `in` then means membership alone (`p.in(a, b)` in a machine's `states`), where today it means both. [inferred] fn-136's role form for rule cases, `in[R]`, is renamed the same way to `when[R]`, whichever spec lands first.

**Several actions, one block.** [paraphrase] `on(a, b, …)` gives several actions the same cases, which is how the activity's four controls share "a control of an activity that is over is not found". [inferred] So one action, or one class, may now appear in more than one `on` block. The rule that an action or class has one block gives way to the overlap check, keyed by (class, phase) across every block that names the class.

**Every rule is a block.** [paraphrase] Every `on` is written as a block, one case per line. [inferred] The single-case parenthesized form `on(x)(case)` is the same Scala call, so the model lint enforces the block form.

**One action per RPC.** [paraphrase] The standalone activity's client actions are `start`, `pause`, `unpause`, `requestCancel` and `terminate`, each carrying its own request schema. The worker's are `poll`, `respondCompleted`, `respondFailed` and `respondCanceled`, each carrying its own request schema. `respondFailed` takes one input of a two-case enum `Failure`, whose cases say whether the failure may be retried. [inferred] The cases are `fatal` and `retryable` (`final` is a Scala keyword). The `Control` enum, its input, `AttemptResult` and its input go.

**Grouped by meaning.** [user] The worker's answers are grouped by what they mean, not by enum order. [paraphrase] A held attempt's final answer (completed, or a fatal failure) settles the activity. An answer that hands the attempt back (a retryable failure, or canceled) does what its phase says.

**Shared rejections.** [paraphrase] Every machine's outcome type is one shared type: accepted, or rejected with a shared `Rejection` that names why, loosely following the gRPC status codes. [inferred] The cases are `notFound`, `alreadyExists`, `failedPrecondition` and `invalidArgument`. [paraphrase] A rule writes a rejection as data: `when(phases) ~> rejects(r)`, optionally `.because(text)` with the server's message. It keeps the state, answers `rejected(r)` and lowers to the same rejecting row `reject(...)` gives today.

Where the shared types live: in the framework, but inside a namespace that `import umpire.*` does not open. Every Model and fixture wildcard-imports `umpire`, and many of them declare their own `Outcome`. A top-level `umpire.Outcome` would make every same-file `Outcome` ambiguous, and it would silently replace an `Outcome` that comes from the Model's package in another file. Adopters name the shared types with an explicit import, which outranks same-package definitions, and they delete their own enum in the same commit. The accepted outcome is found without a per-feature `given Ok`, and the lifter resolves it for `enter` and `stay`. `.because(text)` is a member of the value `rejects(r)` returns, not new top-level sugar, because a core `because` already exists.

**Rule layout.** A block is written brace, newline, one case per line, closing brace on its own line, including a block with a single case. The formatter must keep that layout. scalafmt's `RedundantBraces` with `parensForOneLineApply` turns a one-line brace block back into `on(x)(case)`.

**One mapping to RPC codes.** [paraphrase] The realization layer holds one table, shared by every Model, from each `Rejection` to the RPC status code a Run observes. [inferred] Conformance compares a rejecting step's observed status code against that table, which replaces each realization's own answer strings.

## API Contracts
<!-- scope: technical -->

- [paraphrase] **`from(declarer) { body }`**: inside `rules`; `body` holds `on` blocks for actions `declarer` declares, named by their bare names.
- [paraphrase] **`on(action) { cases }`**, **`on(action(class)) { cases }`**, **`on(a, b, …) { cases }`**: inside `rules`, inside or outside a `from`; `cases` are `when`/`always` cases, one per line.
- [paraphrase] **`when(p1, …)`**, **`when(set)`**, **`when[R]`**: a case that holds in the listed phases, in a named set of phases, or in the phases with role `R`, read through the rules' projection; `.where(g)` narrows it by the state, as `in(...)` does today.
- [paraphrase] **`case ~> rejects(r)`**, **`case ~> rejects(r).because(text)`**: a rejecting row with outcome `rejected(r)`, the state unchanged.
- [paraphrase] **`enum Rejection`** and **`enum Outcome { accepted; rejected(why: Rejection) }`**: shared by every machine.
- [paraphrase] **Rejection to RPC code**: one total mapping from `Rejection` to an RPC status code, in the realization layer.

## Edge Cases & Constraints
<!-- scope: technical -->

- [inferred] An `on` holds no `on`, and a `from` holds no `from`. Both are refused, naming the inner block.
- [inferred] An `on` inside `from(d)` naming an action `d` does not declare is refused, naming the action and `d`.
- [inferred] Two cases that both hold for one class in one phase are refused even when they sit in different `on` blocks. The refusal names the class, both rules and a state where both hold, as today's overlap refusal does.
- [inferred] `in(...)` written as a rule case is refused with a message naming `when`. `p.in(a, b)` as membership keeps working everywhere.
- A `from` block's statements are its declarer's wildcard import, first, then `on` blocks. Anything else in a `from`, including a second import, is refused naming the statement.
- `on(a, b, …)` takes two to four actions as fixed-arity overloads, because `codeOf` cannot name each element of an `inline` varargs list. The activity's four controls are the largest group.
- An action named twice in one `on(a, b, …)`, or an action that is `disabled(…)`, is refused naming the action.
- An Outcome is `Finite`, so `because(text)` text belongs to the step (the IR row's existing `because`) and never to the Outcome value.
- [paraphrase] The mapping from `Rejection` to RPC code is total. A `Rejection` case with no code does not compile.
- [inferred] A step whose observed status code differs from the code its `Rejection` maps to is a conformance failure, naming both codes.
- Today every machine's outcome is a flat enum, and the IR lists outcomes by case name (refinement matches product and System outcomes by name). A parameterized `rejected(why)` case is new to every consumer: the lifter, the Go reader, refinement, the lint acceptances and the capabilities' `rejected` parameter.
- R8's "every machine" covers the machines whose outcome is accepted or rejected: the activity's, Nexus's and the shared worker's. The task queue's `QueueOutcome` (committed, delivered, lost, …) and the close policy's `Answer` are delivery results, not answers to an RPC, so they keep their own types (planning decision, flagged for the owner).
- The rename to `when` changes no IR: rule headings are not written to `model/ir`. The renamed actions, classes, inputs and the shared outcome type are the only names R7 lets change.
- [paraphrase] The behavior freeze holds. Pairs that are disabled today (a pause of a paused, pause-requested or cancel-requested activity, an unpause of one not paused) stay disabled. Rejecting rows for them are not written in this spec.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** [paraphrase] `rules` accept `from(declarer) { import declarer.*; … }` holding `on` blocks for the actions `declarer` declares, named by their bare names; an `on` outside any `from` keeps its meaning. Errors: an `on` in an `on`, a `from` in a `from`, an `on` naming an action the enclosing `from`'s declarer does not declare, and a `from` statement that is neither the leading import nor an `on` are each refused with a message naming the block.
- **R2:** [user] The rule-case form is `when(...)`, including `when(set)`, `when(...).where(g)` and the role form, in place of `in(...)`; `in` keeps only its membership meaning. Errors: `in(...)` as a rule case is refused with a message naming `when`.
- **R3:** [paraphrase] `on(a, b, …) { cases }` gives every named action or class the same cases, and a class may appear in several `on` blocks; the overlap check runs per (class, phase) across all of them. Errors: an overlap is refused naming the class, both rules and a witness state.
- **R4:** [paraphrase] Every `on` in the Temporal Models is written as a block with one case per line. Errors: the model lint refuses the parenthesized single-case form, naming its position.
- **R5:** [paraphrase] The standalone activity's signature declares one action per RPC: the client's `pause`, `unpause`, `requestCancel` and `terminate`, and the worker's `respondCompleted`, `respondFailed` (one `Failure` input with a retryable and a non-retryable case) and `respondCanceled`. The control enum, the attempt-result enum and their inputs are removed. Errors: no error surface beyond compilation.
- **R6:** [paraphrase] The activity's product and System machines write their rules as `from(client)`, `from(worker)` and blocks for the process and the clocks; the four controls share one not-found block; the worker's answers are grouped as final answers that settle a held attempt, then answers that hand the attempt back. Errors: no error surface beyond R1–R3.
- **R7:** Every consumer of the old activity actions (the record, the task-queue composition, the realization, properties, scenarios and queries) uses the new actions. Every Query keeps its expected assessment. A recorded comparison of the generated IR and Cases shows differences only in source positions, in the renamed actions, classes and inputs, and in their declared structural consequences: a split action's step function no longer matches on the removed input, and each new action has its own step function with a shared arm (not-found) repeated in each; schema, result and example metadata move from the old action to the new ones; the shared outcome type replaces each machine's own, `notFound` reads `rejected(notFound)` and `alreadyCompleted` reads `rejected(failedPrecondition)` with its new `because` text; step bindings follow the new `from` grouping's order, and a Case whose witness changes only by that order is explained by it. Errors: any other difference fails the comparison.
- **R8:** [paraphrase] Every machine's outcome type is the shared `Outcome` with the shared `Rejection`. The activity's notFound becomes `rejected(notFound)`. Nexus's alreadyCompleted becomes `rejected(failedPrecondition)`, carrying the server's message as its `because`. Errors: no error surface beyond compilation.
- **R9:** [paraphrase] `when(...) ~> rejects(r)` and `.because(text)` write a rejecting row that lowers to the same IR as the equivalent `reject(Outcome.rejected(r), s)` effect. Errors: no error surface beyond R1–R3.
- **R10:** [paraphrase] The realization layer maps each `Rejection` to one RPC status code in one table shared by every Model, and conformance checks a rejecting step's observed code against it, in place of the per-realization answer strings. Errors: a mismatch fails the step's conformance, naming the expected and observed codes; a `Rejection` without a code does not compile.
- **R11:** Machines other than the activity's change only by the `in` to `when` rename and the shared `Outcome`; their Query assessments are unchanged. Errors: no error surface.

## Early proof point

Task fn-139-actor-grouped-rules-per-rpc-actions.1 validates the core approach: a shared, parameterized `Outcome` (`rejected(why: Rejection)`) lifts and is admitted by the Go IR reader, refinement's outcome-name check and the outcome catalogs, and `rejects(r)` lowers to the same row as `reject(Outcome.rejected(r), s)`, while `enter`/`stay` and a capability's `rejected` argument resolve the shared outcome, and the old per-machine outcomes still compile beside it. If the IR cannot carry a parameterized outcome without a schema or reader change, re-evaluate the shared-outcome representation with the owner before .2+ and before any Model adopts it.

## Boundaries
<!-- scope: business -->

- [paraphrase] New rejecting rows for pairs that are disabled today are out of scope while the behavior freeze holds.
- [inferred] Converting machines other than the activity's product and System to `from` blocks is out of scope. They take only the `in` to `when` rename and the shared outcome.
- [inferred] Splitting other features' actions by RPC (Nexus) is out of scope.
- [inferred] Deriving recorded facts from phases is not part of this spec. It is an amendment to fn-135.

## Decision Context
<!-- scope: both — conditionally substructured -->

[paraphrase] Alternatives prototyped and set aside, in the order considered:
- A decision table (phase × class grid). Every cell was visible, but it was wide, and it needed its own form.
- State-first `from(phase)` blocks. The owner found them the cleanest, but on the System they only worked as a hybrid with `on`, because deadlines and cross-phase answers are about one action, not one phase.
- Rules that name the phase they enter.
- Unions of answer classes ("settles", "yields"). These would have turned a canceled answer to a started attempt into a retry, which is a behavior change.
- A cancel request as a state flag.
- `when(class)` sub-blocks inside a per-action `on`.

[user] The owner then chose to group by actor, then `from` for the actor with `on` for the action, then `when` for the phases.

[paraphrase] The controls were one action only "because they share a result". A multi-action `on` says that in the rules, so the RPCs can be separate actions. [paraphrase] `Failure` replaces the Boolean because `failed(false)` says nothing at the call site. A named argument was considered, but nothing in the Models uses one and it was unverified with the lifter.

[inferred] This spec and fn-135 meet in effects: fn-135 makes effects `effect { }` blocks, and this spec makes rules blocks of the same shape. fn-136 adds `in[R]` for rule cases, which this spec renames to `when[R]`.

Bare-name scoping in `from`, settled by the planning probe:
- Rejected: `from(client) { on(_.pause) { … } }`. It compiles, through a leading `using Scope[D]` clause, but the action is not a bare name, and the lifter would have to read a lambda.
- Rejected: a lifter-recognized exception or macro that injects the declarer's members. It breaks the author-surface rule that admits no macros.
- Rejected: `inline` varargs for `on(a, b, …)`. `codeOf` renders the varargs list unreadably, so the overlap message could not name the action. Fixed arities two to four are used instead.

fn-139 closes with the DSL batch, after R10's export task before the batch regeneration and its
conformance task after it. The original task 8 was split on 2026-10-07 because investigation proved
the mapping needed a new realization IR field and Go reader path in addition to the conformance check.

`disabled(…)` stays outside any `from`: a `from` holds only its import and `on` blocks, and an empty `from(process)` would fail `-Wunused:imports`.

Maintainability (plan review): duplication - the declarer check and the refusal set exist in both the framework (reflection over the declarer's vals) and the lifter (symbol owner); keep the messages identical and test both; structure - the lifter's `ruleSteps` absorbs `from` walking, the import check, declarer membership and multi-target fan-out; extract a rule-block reader if it grows past one screen.

## Parked unknowns

- [inferred] `when` as the rule case versus `when` in Properties (`property when action holds { … }`). Both compile, in different scopes, but one file may show both meanings. Resolved by the owner keeping both or renaming one.
- [inferred] When the freeze lifts, rejecting rows will not refine as the product reads them. The System's `pauseRequested` reads as product `started`, yet a pause is rejected in `pauseRequested` and accepted in `started`, and an unpause is the reverse. Resolved by the owner deciding what `pauseRequested` reports. The same question is raised by fn-135's derived status facts.


## Quick commands

```bash
make lint-model
make MODEL_GATE_ARGS=--skip-go-checks umpire-check-model   # batch close only; tasks run the munit and fixture suites
mise exec -- scala-cli test model/irgen
```

## Resolved via Research
<!-- provenance: plan (docs-scout, practice-scout, docs-gap-scout, memory-scout) on 2026-10-06; plan writes the same section when its Step 1 ran docs-scout or practice-scout -->

### docs-scout
- **Scala 3.9.0 context functions**: a `T ?=> R` parameter is available only as a given, never as members in scope, so bare names need an import. Source: https://docs.scala-lang.org/scala3/reference/contextual/context-functions.html
- **Scala 3.9.0 export clauses**: allowed only in templates and at top level, not as block statements, so `export` cannot scope a `from` block. Source: https://docs.scala-lang.org/scala3/reference/other-new-features/export.html
- **Scala 3.9.0 keywords**: `final` is a hard keyword, hence `Failure.fatal`. `when`, `from` and `on` are plain identifiers. Source: https://docs.scala-lang.org/scala3/reference/syntax.html
- **-Wunused:imports with -Werror**: an `import declarer.*` that no rule uses fails the build, which is the right signal for an empty `from`. Source: model build options

### practice-scout
- **gRPC codes**: NOT_FOUND for an absent entity, ALREADY_EXISTS only for a create that collides, FAILED_PRECONDITION for an existing entity whose state forbids the call, INVALID_ARGUMENT for input bad in any state. Source: https://grpc.github.io/grpc/core/md_doc_statuscodes.html
- **Temporal server**: a control on an activity that is no longer pending answers NotFound, and a pause of an already-paused activity answers FailedPrecondition. Source: https://docs.temporal.io/activity-operations and the server's activity operator commands
- **Gotcha:** conformance compares status codes only, never message text. The order in which the server validates (argument, then existence, then state) must match the Model's choice of rejection. Source: https://grpc.github.io/grpc/core/md_doc_statuscodes.html

### docs-gap-scout
- **Docs that must change:** the authoring guide's rules section, its sugar list, its Store fixture (a single-case `on`) and its capability examples that name `client.control(Control.*)`. Source: model/README.md (rules section, implements examples)
- **Docs that must change:** the semantics document's Rules section: block forms, `when` lowering, disjointness per (class, phase) across blocks, `rejects` lowering. Source: model/SEMANTICS.md, Rules
- **Docs that must change:** the operator catalogue's rule-block row and its inline exception, rule 5. Source: .plans/DSL_OPERATORS.md
- **Docs that must change:** the rule doc comments in the framework syntax file, each new form with its `Core form:` line. Source: model/umpire/Syntax.scala

### memory-scout
- **Glossary renames can reintroduce names the vocabulary gate retired**: grep hand-written Go and Scala for the old and the new names after the action renames. Source: bug/build-errors/glossary-renames-can-reintroduce-names-2026-09-13
- **Package-only Model headers need normal compilation in every caller**: a new shared header for `Outcome` and `Rejection` must compile for every caller, including the Makefile's rewrite-mode scalafix. Source: bug/build-errors/package-only-model-headers-need-normal-2026-10-06
- **Consolidated extractor dropped a caller's nonempty-field rejection**: when per-machine outcomes merge into one, diff each machine's rejection set so none is lost. Source: bug/integration/consolidated-extractor-dropped-a-2026-09-27

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | [paraphrase] `rules` accept `from(declarer) { import declarer.*; … }` holding `on` blocks for the actions `declarer` declares, named by their bare names; an `on` outside any `from` keeps its meaning. Errors: an `on` in an `on`, a `from` in a `from`, an `on` naming an action the enclosing `from`'s declarer does not declare, and a `from` statement that is neither the leading import nor an `on` are each refused with a message naming the block. | fn-139-actor-grouped-rules-per-rpc-actions.2, fn-139-actor-grouped-rules-per-rpc-actions.3 | — |
| R2 | [user] The rule-case form is `when(...)`, including `when(set)`, `when(...).where(g)` and the role form, in place of `in(...)`; `in` keeps only its membership meaning. Errors: `in(...)` as a rule case is refused with a message naming `when`. | fn-139-actor-grouped-rules-per-rpc-actions.2, fn-139-actor-grouped-rules-per-rpc-actions.3, fn-139-actor-grouped-rules-per-rpc-actions.7 | — |
| R3 | [paraphrase] `on(a, b, …) { cases }` gives every named action or class the same cases, and a class may appear in several `on` blocks; the overlap check runs per (class, phase) across all of them. Errors: an overlap is refused naming the class, both rules and a witness state. | fn-139-actor-grouped-rules-per-rpc-actions.2, fn-139-actor-grouped-rules-per-rpc-actions.3 | — |
| R4 | [paraphrase] Every `on` in the Temporal Models is written as a block with one case per line. Errors: the model lint refuses the parenthesized single-case form, naming its position. | fn-139-actor-grouped-rules-per-rpc-actions.3, fn-139-actor-grouped-rules-per-rpc-actions.7 | — |
| R5 | [paraphrase] The standalone activity's signature declares one action per RPC: the client's `pause`, `unpause`, `requestCancel` and `terminate`, and the worker's `respondCompleted`, `respondFailed` (one `Failure` input with a retryable and a non-retryable case) and `respondCanceled`. The control enum, the attempt-result enum and their inputs are removed. Errors: no error surface beyond compilation. | fn-139-actor-grouped-rules-per-rpc-actions.4 | — |
| R6 | [paraphrase] The activity's product and System machines write their rules as `from(client)`, `from(worker)` and blocks for the process and the clocks; the four controls share one not-found block; the worker's answers are grouped as final answers that settle a held attempt, then answers that hand the attempt back. Errors: no error surface beyond R1–R3. | fn-139-actor-grouped-rules-per-rpc-actions.5 | — |
| R7 | Every consumer of the old activity actions (the record, the task-queue composition, the realization, properties, scenarios and queries) uses the new actions. Every Query keeps its expected assessment. A recorded comparison of the generated IR and Cases shows differences only in source positions, in the renamed actions, classes and inputs, and in their declared structural consequences: a split action's step function no longer matches on the removed input, and each new action has its own step function with a shared arm (not-found) repeated in each; schema, result and example metadata move from the old action to the new ones; the shared outcome type replaces each machine's own, `notFound` reads `rejected(notFound)` and `alreadyCompleted` reads `rejected(failedPrecondition)` with its new `because` text; step bindings follow the new `from` grouping's order, and a Case whose witness changes only by that order is explained by it. Errors: any other difference fails the comparison. | fn-139-actor-grouped-rules-per-rpc-actions.4, fn-139-actor-grouped-rules-per-rpc-actions.5 | — |
| R8 | [paraphrase] Every machine's outcome type is the shared `Outcome` with the shared `Rejection`. The activity's notFound becomes `rejected(notFound)`. Nexus's alreadyCompleted becomes `rejected(failedPrecondition)`, carrying the server's message as its `because`. Errors: no error surface beyond compilation. | fn-139-actor-grouped-rules-per-rpc-actions.1, fn-139-actor-grouped-rules-per-rpc-actions.5, fn-139-actor-grouped-rules-per-rpc-actions.6 | — |
| R9 | [paraphrase] `when(...) ~> rejects(r)` and `.because(text)` write a rejecting row that lowers to the same IR as the equivalent `reject(Outcome.rejected(r), s)` effect. Errors: no error surface beyond R1–R3. | fn-139-actor-grouped-rules-per-rpc-actions.1 | — |
| R10 | [paraphrase] The realization layer maps each `Rejection` to one RPC status code in one table shared by every Model, and conformance checks a rejecting step's observed code against it, in place of the per-realization answer strings. Errors: a mismatch fails the step's conformance, naming the expected and observed codes; a `Rejection` without a code does not compile. | fn-139-actor-grouped-rules-per-rpc-actions.8, fn-139-actor-grouped-rules-per-rpc-actions.9 | — |
| R11 | Machines other than the activity's change only by the `in` to `when` rename and the shared `Outcome`; their Query assessments are unchanged. Errors: no error surface. | fn-139-actor-grouped-rules-per-rpc-actions.6, fn-139-actor-grouped-rules-per-rpc-actions.7 | — |
