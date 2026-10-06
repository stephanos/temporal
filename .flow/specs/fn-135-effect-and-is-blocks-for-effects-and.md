# effect { } and is { } blocks for effects and predicates

> HTML render lens: [.flow/artifacts/fn-135-effect-and-is-blocks-for-effects-and/spec.html](../artifacts/fn-135-effect-and-is-blocks-for-effects-and/spec.html) — local-only (gitignored): open it from the working tree; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (turn 1, part 1): ">   def over(s: State) = terminal(s.phase)"
> user (turn 1, part 2): ">     def startAttempt(s: State) = enter(s.copy(phase = started), statusStarted)"
> user (turn 1, part 3): "why do we even need `s: State`? isn't that part of the object ActivityProduct?"
> user (turn 1, part 4): "can't we refer to it inside the method directly. droppong the param would remove a lot of visual noise"
> user (turn 2): "sth very sleak like `effect { }` maybe?"
> user (turn 3, part 1): "any chance we can do"
> user (turn 3, part 2): "val startAttempt = effect { phase = started; return statusStarted } or sth like it?"
> user (turn 3, part 3): "> holds { } "
> user (turn 3, part 4): "why \"holds\"?"
> user (turn 4): "the impicit return `statusStarted` is a bit odd; can we make that clearer to a reader? it's okay if it's longer"
> user (turn 5): "> val paused = is { phase == Phase.paused } - can we use it for all `states`? for consistency?"
> user (turn 6, part 1): "yes for this"
> user (turn 6, part 2): "val pause = effect {"
> user (turn 6, part 3): "  phase = Phase.paused"
> user (turn 6, part 4): "  record(statusPaused)"
> user (turn 6, part 5): "}"
> user (turn 6, part 6): "yes for `is` as descrined here"
> user (turn 6, part 7): "let's create a flow next spec"
> user (turn 7): "one more thing; I think Temporal usually calls what we call \"Phase\" a \"State\"? should we align this better? but then we'd need a new word for \"State\", huh?"
> user (turn 8, selected): "Parked note only"
> user (amendment, part 1): "val startAttempt = effect { phase = started; record(statusStarted) }"
> user (amendment, part 2): "why do we need the record(statusStarted) etc?"
> user (amendment, part 3): "can't that be declrated declartively somehow; maybe on the phase itself? what is it good for anyway?"
> user (amendment, selected): "Amend fn-135"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 60% [paraphrase], 40% [inferred] -->

A Model author writes every effect and every state predicate of a machine as a method that takes the state as a parameter: `def startAttempt(s: State) = enter(s.copy(phase = started), statusStarted)`, `def over(s: State) = terminal(s.phase)`. The owner reads the `s: State` parameter, and the `s.` and `s.copy(...)` that follow from it, as visual noise. They asked why the state is not simply in scope inside the method.

It cannot be a member of the machine object: a machine is a stateless description of a transition function, and the state is a value the checker threads from step to step. The rules pass effects and predicates around as functions of the state. What can change is how an author writes those functions.

This spec adds two block forms. An effect is written as `effect { … }`: the block assigns the state's fields by name and records facts or rejects with explicit statements. A predicate over the state is written as `is { … }`: the block reads the state's fields by name. The owner chose explicit `record(...)` statements over a block whose last expression is the recorded fact, because an implicit result "is a bit odd" for a reader, and accepted a longer form for clarity. They chose `is` over `holds`, and want it used for every predicate in a machine's `states` section, for consistency.

The activity's product machine (`ActivityProduct`) is the first machine written this way.

[amendment] The owner then asked why an effect records its status at all, and whether that could be declared on the phase instead. A recorded fact is what a step claims a Run can observe: evidence confirms each step by recording one of its facts, Properties read facts, and refinement compares them. In the product every fact an effect records is the status of the phase it enters, so writing it in every effect repeats what the phase already says. The amendment declares each phase's status once, on the phase enum, and derives the status fact.

## Architecture & Data Models
<!-- scope: technical -->

**The state stays a value.** [paraphrase] A machine object holds no state. `effect { … }` and `is { … }` each produce a function of the state, the same type the rules and capabilities take today, so rule bindings (`~> effects.pause`, `where(states.held)`) and capability declarations keep their shape.

**Effect blocks.** [paraphrase] Inside `effect { … }`, each field of the machine's state type is readable and assignable by its name (`phase`, `phase = started`). The block starts from the current state; an assignment replaces that field in the state the effect enters. `record(f, …)` records one or more facts of the step. `reject(o)` answers the step with outcome `o` and leaves the state as it was. A block that neither records nor rejects enters the assigned state with no fact. Outside the block nothing is mutable: the effect, as seen by the rules, the checker and the IR, is the same pure function of the state that the method form produces.

**Predicate blocks.** [paraphrase] Inside `is { … }`, each field of the state type is readable by name, and the block's Boolean expression is the predicate's answer. An `is` block receives a read-only view of the state; the effect draft extends that view, so getters work in both blocks while setters, `record` and the one-argument `reject` require the draft and do not compile inside `is { }`.

**Which `states` members become `is { }`.** [paraphrase] Every member of a `states` section that answers a yes/no question about the state is written as `is { … }`. Three kinds of member keep their own form, because their types differ:
- a predicate over the projected status (in the activity, `terminal`), which `in(...)` in the rules and `Closable(terminal = …)` take as a status predicate, stays a function of the status;
- the status projection, which `Closable(status = …)` takes, stays a function of the state to its status. In the activity it is renamed from `phase` to `status`, because a `states.phase` member would shadow the `phase` field the blocks read;
- a constant (in the activity, `notFoundCode`) stays a constant.

**Field accessors.** Each machine's state type provides the by-name field getters and setters the blocks use, hand-written in its level file after the file's types and before its machine object (the feature-file order), in a scope that does not clash with section members. Each accessor has one fixed one-line shape over the draft API: a getter reads one field through the block's view of the state, and a setter replaces one field of the draft's state with a copy. The lifter recognizes exactly that shape, lifting a getter call as a read of that field of the state and a setter call as an update of that field, and refuses an accessor of any other shape, naming it. So an accessor cannot read or write a different field from the one the IR records. They are not derived: the DSL's author surface admits no macros or `inline` (the DSL operator rules), and Scala cannot synthesize named members from a `Mirror`. A converted machine whose state gains a field without accessors does not compile where a block names that field.

**Derived status facts.** [amendment] [paraphrase] Each case of a machine's phase enum declares the fact its status is recorded as, on the case itself. A step that changes the declared status records that fact, after any facts its effect records explicitly; a step that keeps the status records no status fact. An effect block writes `record(...)` only for facts the phase does not determine (an attempt count, which deadline fired). [inferred] The derivation runs where the effect's step is built, so the effect seen by the rules, the checker and the IR is the same pure function of the state that writing the fact by hand gives, and it lifts to the same IR. A machine whose phase enum declares no status keeps today's behavior: it records only what its effects record.

**Each call starts from a fresh draft.** An effect's draft is created per call and never escapes the block, so effects stay pure and re-entrant as seen from outside.

**The IR generator lifts both forms.** [paraphrase] The generator lifts an `effect { … }` block to the same step function it lifts from the equivalent method form, and an `is { … }` block to the same predicate, with the same parameter name and type the method form's lifted function carries: the block has no state parameter of its own, so the lifter supplies one named `s` of the machine's state type. Field updates are emitted in field-declaration order, each lifted with the field's type as its expected type, as a `copy` call's are. It resolves references to section members declared as `val`s the way it resolves `def` members today, wherever a rule binding, a rule condition, a capability declaration or a claim names a member, through one recognizer that every such site consults. Every path that handles `def` members stays as it is. The declaration-placement lint classifies an `effect { }` val as the step function it is, so the rule that step functions belong in `effects` still applies to converted effects (the lint has no kind for predicates, `def` or `val`).

## API Contracts
<!-- scope: technical -->

- [paraphrase] **`effect { body }`**: available inside a machine object, typed by the machine's state, outcome and fact types; returns a function from the state to the machine's steps. `body` is a sequence of field assignments, `record(...)` and `reject(...)` statements.
- [paraphrase] **`is { body }`**: available inside a machine object; returns a function from the state to `Boolean`. `body` is a Boolean expression over the state's fields.
- [paraphrase] **`record(facts*)`**: valid only inside `effect { }`; records the given facts of the step, in order.
- [paraphrase] **`reject(outcome)`**: valid only inside `effect { }`; answers the step with `outcome` and the unchanged state.
- [paraphrase] **Field access**: inside either block, `<field>` reads the current value of a state field; inside `effect { }`, `<field> = v` assigns it.
- [amendment] [paraphrase] **Status declaration**: each case of a phase enum names the fact its status is recorded as. When a step changes the status, that fact is recorded without a `record(...)`.

## Edge Cases & Constraints
<!-- scope: technical -->

- An effect block's statements are straight-line: assignments, `record` and `reject` do not appear inside a branch (`if`, `match`) or a loop. The generator refuses a block that breaks this, naming the statement. The generator lifts `if` and `match` generically today, so this refusal is new.
- A block that calls `reject` contains that one `reject` and nothing else; the generator refuses one that also assigns, records or rejects again.
- A field is assigned at most once per block, and is not read after its assignment in the same block. The IR's copy of the state has no statement order, so a read after an assignment would mean one thing at run time and another in the IR. The generator refuses both, naming the field.
- `record` takes at least one fact; several `record` calls record their facts in call order, as one `record` with all of them would.
- A block with assignments and no `record` enters the assigned state with no fact; an empty block enters the unchanged state with no fact. Each lifts as the equivalent `enter(...)` does.
- An `effect` or `is` nested inside another block is refused by the generator.
- An effect block holds only field assignments, `record` and `reject`; any other statement (a local `val`, a bare expression such as `phase == started`, any other call) is refused, naming it.
- Inside `object states`, a section member shadows a same-named enum case brought in by a wildcard import (`paused`). A block body names such a case qualified (`Phase.paused`), as the method form already does.
- `record` and the one-argument `reject` outside an effect block, and a field assignment outside an effect block (including inside `is { }`), do not compile. The two-argument `reject(outcome, s)` stays available to method-form effects.
- The method forms (`def f(s: State) = …`) stay accepted; machines not converted by this spec keep compiling and lifting unchanged.
- Section `val`s initialize with their section object. No section member reads `rules` or `init` while the machine initializes.
- [amendment] [inferred] Every case of an enum that declares statuses declares one. A case without one does not compile.
- [amendment] [inferred] An effect that explicitly records the status fact the derivation would record is refused, naming the effect and the fact, so a step never records one fact twice.
- [amendment] [inferred] A rejecting effect keeps the state, so it records no derived fact.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** [paraphrase] `ActivityProduct` declares every effect as `effect { … }`, assigning state fields by name and recording facts with `record(...)` (or answering with `reject(...)`); no effect in it takes a state parameter or calls `copy`. Errors: no error surface beyond R5 and R6.
- **R2:** [paraphrase] Every member of `ActivityProduct`'s `states` section that is a yes/no question about the state is declared as `is { … }`; the status predicate, the status projection and the constant keep their forms, and the projection is named `status`. Errors: no error surface beyond R5.
- **R3:** [paraphrase] The rules, capability declarations and `end` of `ActivityProduct` reference the converted members without wrapping, e.g. `~> effects.pause`, `where(states.held)`, `Closable(status = states.status, …)`. Errors: no error surface beyond compilation.
- **R4:** The IR and Cases generated for the activity model after the conversion are identical to those generated before it, apart from source positions and the function name of the renamed status projection (`…states.phase` → `…states.status`) wherever that name appears in the IR, its laws and lint files. A recorded comparison that parses the JSON, strips positions, renames that function and compares functions keyed by name (the IR sorts functions by name, so the rename moves it) shows this. Errors: any other difference fails the comparison.
- **R5:** The IR generator lifts `effect { … }` and `is { … }` blocks declared as section `val`s, and resolves those members wherever rules, capability declarations and claims name them. Errors: an effect block with an assignment, `record` or `reject` inside a branch or loop; a rejecting block with any other statement; a field assigned twice or read after its assignment; any statement other than an assignment, `record` or `reject`; a nested block; and a field accessor not of the fixed shape are each refused with a message that names the statement and its position.
- **R6:** Effect and predicate blocks behave as their method equivalents: from a given state, `effect { phase = started; record(statusStarted) }` yields the same steps as `s => enter(s.copy(phase = started), statusStarted)`, `effect { reject(o) }` the same as `s => reject(o, s)`, and an `is { … }` block answers as the equivalent predicate. A fixture machine with a two-field state shows the same for a partial assignment and for several `record` calls, both at run time and in lifted IR. Errors: no error surface beyond R5.
- **R7:** Machines still written in the method form compile and lift to unchanged IR (every existing lifter fixture and every IR file other than the activity's is unchanged). Errors: no error surface.
- **R8:** [amendment] [paraphrase] `ActivityProduct`'s phase enum declares, on each case, the status fact it is recorded as, and no effect of `ActivityProduct` records a status fact explicitly: `startAttempt` is `effect { phase = started }`. Errors: a case without a status, and an effect recording the derived fact explicitly, are refused as the Edge Cases state.
- **R9:** [amendment] [inferred] A step that changes the declared status records that status's fact after the effect's explicit facts, and a step that keeps it records no status fact. A fixture machine shows both, including an effect that also records a fact the phase does not determine, at run time and in lifted IR. Errors: no error surface beyond R8.
- **R10:** [amendment] [inferred] R4's comparison holds after R8: the activity's IR and Cases are unchanged by deriving the facts, apart from what R4 already allows. Errors: any other difference fails the comparison.

## Early proof point

Task fn-135-effect-and-is-blocks-for-effects-and.2 validates the core approach (a `val` section member built by `is { }` lifts through rule conditions and capability declarations to the same IR as its `def` form, with every `def` path unchanged). If it fails, re-evaluate keeping members as `def`s whose body is the block (`def held = is { … }`) before continuing with .3+.

## Quick commands

```bash
# From the repo root: model tests and lifter fixtures, then the checked-in IR and Cases
make umpire-check-model && make umpire-check-cases
# DSL syntax and lint rules (sugar names, Core form docs, scalafix)
make lint-model-syntax lint-model
```

## Boundaries
<!-- scope: business -->

- Converting machines other than `ActivityProduct` is out of scope; this spec proves the form on one machine.
- Branching inside an effect block (conditional assignments or records) is not supported.
- Effects that take action parameters (`(S, A) => …`) keep the method form; `effect { }` covers state-only effects.
- An `is { }` block calling another `is` member, and `effect { }` or `is { }` used anywhere but as a section `val`, are not supported.
- The rules' status projection (`Rules(_.phase)`) is unchanged.
- [paraphrase] An effect whose last expression is the recorded fact, and a `return` of the fact, are not offered; recording is always an explicit `record(...)`.
- [amendment] [inferred] Deriving facts in machines other than `ActivityProduct`, and deriving the System's refinement mapping or `Closable`'s status projection from the status declaration, are out of scope. They are candidates once the declaration exists.

## Decision Context
<!-- scope: both — conditionally substructured -->

[paraphrase] Keeping the state as a machine member was rejected: a machine is a stateless description, and the rules need effects and predicates as functions of the state. Rejected forms, in the order considered: helper constructors such as `to(started, statusStarted)` (terse, but each new shape needs a new helper); extension methods on the state (moves the noise to the call sites, `where(_.held)`); an effect block returning its fact as the last expression (the owner found the implicit result unclear); a fact named outside the block, `effect { … } records f` (reads well for one fact but gives rejections a different shape from every other effect). [paraphrase] `is` was preferred over `holds`, `condition` and `when` for predicates. A prototype on Scala 3.9.0 confirmed that by-name field assignment inside a context-function block compiles (`phase = started` resolves to a `phase_=` setter in scope) and that `record`/`reject` statements compose.

[amendment] [paraphrase] Status facts are derived rather than written because, in the product, every one of them restates the phase being entered. Facts stay, because conformance confirms steps through them and Properties and refinement read them. Only status facts are derived. Facts the phase does not determine stay explicit `record(...)` calls. The System shows why it can't be all of them: `attemptCount` is recorded on a backoff and on an attempt start, and `statusTimedOut` takes which deadline fired. The amendment narrows the earlier "recording is always an explicit `record(...)`" to facts the phase does not determine. The rule against an implicit last-expression result still stands.

Maintainability (plan review): duplication - each `isFunction`/`forwardedDef` caller deciding separately whether a member is a block val, instead of one recognizer; structure - none identified.
Maintainability (plan review): duplication - none identified; structure - the rule-effect lowering gains a `val` branch beside its lambda/`def` branch (advisory).
Maintainability (plan review): duplication - the batch no-regeneration line is pasted into tasks .2-.5 while .2 and .3 keep `make umpire-check-model` acceptance items; structure - the .3 effect-block walker also takes on status-fact derivation and two refusals in .5 (advisory).

## Parked unknowns

- [paraphrase] Whether to rename `Phase` to `Status` across every machine, keeping `State` for the machine's whole state. Temporal's server names the activity's enum `ActivityExecutionStatus`, with the same values as the activity's `Phase`, and `Closable` already calls the projection `status`. The rename spans all machines, not just the one this spec converts. If it happens, the field read inside the blocks becomes `status`, and the status projection this spec names `status` needs another name, or it shadows that field. Resolved by the owner deciding whether to capture the rename as its own spec.
- [amendment] [inferred] What the System's `pauseRequested` reports. Its pause request records `statusPaused`, yet its refinement reads `pauseRequested` as the product's `started`. A status declared on the phase forces one answer when the System adopts derived facts. fn-139 raises the same question for rejecting rows. Resolved by the owner deciding what DescribeActivityExecution shows while a pause request waits on the worker.
- [amendment] [inferred] What "changes the declared status" (R9) means. Read as a comparison of old and new values, it conflicts with R10: `ActivityProduct`'s `requestCancel` fires from `cancelRequested` too and records `statusCancelRequested` there today, and a value comparison would also put a conditional in the lifted fact list. Planning (fn-135.5) implements the static reading instead: an effect block that assigns the status field records the assigned case's fact, and one that does not assign it records none. Resolved by the owner confirming that reading or amending R9/R10 (or narrowing `requestCancel`'s rule) before fn-135.5 starts.


## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | [paraphrase] `ActivityProduct` declares every effect as `effect { … }`, assigning state fields by name and recording facts with `record(...)` (or answering with `reject(...)`); no effect in it takes a state parameter or calls `copy`. Errors: no error surface beyond R5 and R6. | fn-135-effect-and-is-blocks-for-effects-and.4 | — |
| R2 | [paraphrase] Every member of `ActivityProduct`'s `states` section that is a yes/no question about the state is declared as `is { … }`; the status predicate, the status projection and the constant keep their forms, and the projection is named `status`. Errors: no error surface beyond R5. | fn-135-effect-and-is-blocks-for-effects-and.4 | — |
| R3 | [paraphrase] The rules, capability declarations and `end` of `ActivityProduct` reference the converted members without wrapping, e.g. `~> effects.pause`, `where(states.held)`, `Closable(status = states.status, …)`. Errors: no error surface beyond compilation. | fn-135-effect-and-is-blocks-for-effects-and.4 | — |
| R4 | The IR and Cases generated for the activity model after the conversion are identical to those generated before it, apart from source positions and the function name of the renamed status projection (`…states.phase` → `…states.status`) wherever that name appears in the IR, its laws and lint files. A recorded comparison that parses the JSON, strips positions, renames that function and compares functions keyed by name (the IR sorts functions by name, so the rename moves it) shows this. Errors: any other difference fails the comparison. | fn-135-effect-and-is-blocks-for-effects-and.4 | — |
| R5 | The IR generator lifts `effect { … }` and `is { … }` blocks declared as section `val`s, and resolves those members wherever rules, capability declarations and claims name them. Errors: an effect block with an assignment, `record` or `reject` inside a branch or loop; a rejecting block with any other statement; a field assigned twice or read after its assignment; any statement other than an assignment, `record` or `reject`; a nested block; and a field accessor not of the fixed shape are each refused with a message that names the statement and its position. | fn-135-effect-and-is-blocks-for-effects-and.2, fn-135-effect-and-is-blocks-for-effects-and.3 | — |
| R6 | Effect and predicate blocks behave as their method equivalents: from a given state, `effect { phase = started; record(statusStarted) }` yields the same steps as `s => enter(s.copy(phase = started), statusStarted)`, `effect { reject(o) }` the same as `s => reject(o, s)`, and an `is { … }` block answers as the equivalent predicate. A fixture machine with a two-field state shows the same for a partial assignment and for several `record` calls, both at run time and in lifted IR. Errors: no error surface beyond R5. | fn-135-effect-and-is-blocks-for-effects-and.1, fn-135-effect-and-is-blocks-for-effects-and.3 | — |
| R7 | Machines still written in the method form compile and lift to unchanged IR (every existing lifter fixture and every IR file other than the activity's is unchanged). Errors: no error surface. | fn-135-effect-and-is-blocks-for-effects-and.2, fn-135-effect-and-is-blocks-for-effects-and.3 | — |
| R8 | [amendment] [paraphrase] `ActivityProduct`'s phase enum declares, on each case, the status fact it is recorded as, and no effect of `ActivityProduct` records a status fact explicitly: `startAttempt` is `effect { phase = started }`. Errors: a case without a status, and an effect recording the derived fact explicitly, are refused as the Edge Cases state. | fn-135-effect-and-is-blocks-for-effects-and.4, fn-135-effect-and-is-blocks-for-effects-and.5 | — |
| R9 | [amendment] [inferred] A step that changes the declared status records that status's fact after the effect's explicit facts, and a step that keeps it records no status fact. A fixture machine shows both, including an effect that also records a fact the phase does not determine, at run time and in lifted IR. Errors: no error surface beyond R8. | fn-135-effect-and-is-blocks-for-effects-and.5 | — |
| R10 | [amendment] [inferred] R4's comparison holds after R8: the activity's IR and Cases are unchanged by deriving the facts, apart from what R4 already allows. Errors: any other difference fails the comparison. | fn-135-effect-and-is-blocks-for-effects-and.4 | — |

