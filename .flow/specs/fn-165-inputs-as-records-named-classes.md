# Inputs as records: named classes, declared defaults, no `:=`

DSL changes ship a task-named Scalafix rule under `model/migrate/rules/`, prove it on copied `activity_mini` sources before the real-feature sweep, and follow `model/migrate/README.md`; production loading remains held in fn-166.10.

## Conversation Evidence

> user (2026-10-11): "the positiinal form seems difficult to read; since you need to know the positions."
> user: "is there a better more idtiomatic way in scala do to that rather than inventing :="
> user: "I like that empty form like `start()` means all defaults - why not?"
> user: "is there a design that uses the new case class but preserves the first loss. maybe named/opaque input types or sth. give a few options, you mentioned traits too"
> user (selected): "okay Option 1; write a flow spec/tasks as needed"

Source: `model/SEMANTICS_FEEDBACK.md` 3.3 and the design discussion of 2026-10-11 that followed it.

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 40% [user], 40% [paraphrase], 20% [inferred] -->

An action's inputs are declared as tokens (`val scheduleToStart = input[Timeout]`) chained onto the action (`.input(scheduleToStart)`), and a class of the action is written three ways: positionally (`start(unset, expires, unset, unset, unset, two)`), by name with the DSL's own symbol (`start(startToClose := expires, maxAttempts := MaxAttempts.two)`), or empty (`start()`). The positional form is hard to read because the positions are the order of six `.input` calls in another file [user]. The `:=` symbol exists only because an action is a value whose `apply` is generic over a tuple of input types and so cannot carry parameter names [paraphrase]. A step function that reads inputs takes them as extra positional parameters, typed by an arity ladder of `~>` extensions, which is the same positional reading in a second place [inferred].

The owner wants the idiomatic Scala form: a record of inputs per action, with named arguments and parameter defaults, so that `start(maxAttempts = MaxAttempts.two)` is plain Scala, `start()` stays "every input at its default" [user], and a step function reads `in.startToClose` as it reads `s.phase`. Inputs then use the same mechanism as the state: a case class, a derived catalog, field selection, structural equality. The DSL loses its one parallel mechanism for inputs (tokens, tuple typing, `:=`, the arity ladder) [paraphrase].

The one thing tokens give that records do not is identity across actions: `scheduleToStart` is one token shared by `start` and `schedule`, and the realization's `deadlines` names it by that token. The owner chose to preserve that fact structurally (option 1 of five weighed in Decision Context): a shared input is a field with the same name and type in each record, and a consumer that needs "any action with a `scheduleToStart`" states it as a structural bound the compiler checks [user].

## Architecture & Data Models
<!-- scope: technical -->

**The record.** [paraphrase] An action with inputs declares them as a final case class whose fields are finite domains, each with a default where the input may be omitted:

```scala
final case class Start(
    scheduleToClose: Timeout = Timeout.unset,
    scheduleToStart: Timeout = Timeout.unset,
    startToClose:    Timeout = Timeout.unset,
    heartbeat:       Timeout = Timeout.unset,
    startDelay:      Timeout = Timeout.unset,
    maxAttempts:     MaxAttempts = MaxAttempts.unlimited
)

object client extends Client:
  val start = action(this).inputs[Start].creates(activity)
```

A class is a record value: `client.start()` is every default, `client.start(maxAttempts = MaxAttempts.two)` names what differs, and `client.start(Start(...))` is the same class written as a value where a Scenario or a witness holds one. `on(client.start)` is still the whole action, every class; `on(client.start())` is the one all-defaults class.

**Single-input actions keep the value shape.** [inferred] `finish(Result.ok)` and `.example(Failure.fatal, text)` read well because the value is the class. An action declared `.input[Result]` takes the value directly; `Action[In]` is one type whose `In` is either a record or a single domain value, and the lifter tells the two apart by whether `In` is a case class the action declares as its record. A record with one field is allowed but never required.

**The step function reads a record.** [paraphrase] `~>` binds `S => Steps` for an action without inputs and `(S, In) => Steps` for one with; the arity ladder goes. Inside a step function, an input is read by field selection, `in.startToClose`, which the lifter lifts exactly as it lifts `s.phase`. No other operation on `in` is in the liftable subset: `in.copy`, pattern matching on the record, and methods a record declares are refused in function bodies, as they are for the state.

**Shared inputs are structural.** [user] A capability law or a realization helper that applies to any action with a given input states it as a bound, `[In <: { def scheduleToStart: Timeout }]`, and the framework reads the field by its label through the record's `Mirror`, with no reflection. Identity is "same name, same type"; nothing is declared. The realization's `deadlines` names the input by a selector, `_.scheduleToStart`, as compositions already name members by `_.member`.

**Catalog and IR unchanged.** [inferred] The classes of an action are `Finite[In].values`, derived like every other domain, in product order with the last field varying fastest; this is the order `Class` enumerates today, so class keys (`name-key(i1)-...`) and column order are unchanged when the record's field order is the token order. The IR's `Action.inputs` (`Param` name and type) is produced from the record's `MirroredElemLabels` and field types. `examples` and `schemas` are carried as today. No IR schema change, no Go change.

**Defaults.** [paraphrase] A parameter default is the input's default, replacing fn-159.10's `input[T](default = v)`, which this spec supersedes for every action it migrates. A default is an enum case or a literal of the field's domain; a computed default is refused by the lifter. An input without a default must be supplied in every named call, which the compiler enforces.

**Positional calls.** [inferred] Scala cannot forbid positional arguments, so the lifter refuses a class written positionally with two or more inputs (an argument that is not a `NamedArg`), at the call. A single-input action's `finish(Result.ok)` is not positional in this sense.

**Lifter.** [inferred] Before fn-141 Part B, the lifter lowers a class from the record's construction tree: `NamedArg` nodes supply the named fields, the case class's parameter defaults supply the rest, and `Start(...)` as a value is read the same way. After Part B, classes export by running (a class is a record value the framework holds) and the lifter keeps only the function-body read of `in.field` and the positional refusal. The token-identity sites (`inputTokens`, `tokens.indexOf(sym)` in `Realizations.scala`; the `Assigned` matcher in `Syntax.scala`) go.

## API Contracts
<!-- scope: technical -->

- `action(actor).inputs[In]` declares an action whose classes are the values of the record `In`; `.input[A]` declares a single-input action whose classes are the values of `A`. The chained `.input(token)` form and `input[T]` tokens are removed from the framework. [inferred]
- A class is written `a()` (every default), `a(field = value, ...)` (named, the rest at their defaults) or `a(In(...))`; a single-input class is `a(value)`. [user, paraphrase]
- `a ~> f` takes `f: S => Steps` or `f: (S, In) => Steps`; a step function written for another action's record does not compile. [paraphrase]
- Field selection `in.field` is the only input read in a function body. [inferred]
- A shared-input consumer is written with a structural bound on `In`; the framework provides `Inputs.field[In](label)` or an equivalent read by label. [user]
- A realization's `deadlines` and class presets name an input by selector, `_.field`, never by a token. [inferred]
- `:=` is removed from class construction. Its other use, the realization's request-scope assignment (`withFields { request.x := v }`), is outside this spec. [inferred]
- `Core form:` of every sugar this spec adds is documented in the framework's sugar file, as fn-141 requires. [paraphrase]

## Edge Cases & Constraints
<!-- scope: technical -->

- Field order is the class order and the `Param` order. A migration keeps each record's field order equal to its action's token order, and each default equal to the value the first-value rule gives today, so the IR is byte-identical. The step-table equivalence is the check. [inferred]
- Two actions that share an input today (`start` and `schedule` share `scheduleToStart`) become two records with a field of the same name and type. Nothing checks that two fields with one name are one concept; a Model that wants that fact checked writes the structural bound. [user]
- A multi-target rule `on(a, b)` whose actions have different records binds a step over the state alone, as today a step over different input tuples cannot be shared. [inferred]
- `.example(value, text)` on a single-input action is unchanged; on a record action an example is `.example(In(...), text)`. [inferred]
- A record field of a type with no catalog (`Int`, `String`, a list) is refused where `Finite[In]` is first needed, with the field named, as for a state record. [inferred]
- Strict equality: `in.startToClose == Timeout.expires` compiles through the derived catalog as any domain comparison does. [inferred]
- The two action shapes (record, single value) must lift to the same `Action` IR; a fixture holds one of each beside an action with no inputs. [inferred]
- fn-144 (tags on domains) and fn-125 (settings) add to the input surface; both are deferred, and a record field is the natural carrier for a tag. They adapt to this spec, not the reverse. [inferred]

## Acceptance Criteria
<!-- scope: both -->

- **R1:** An action with inputs declares them as a case class with `inputs[In]`, and a class is written as an empty call, a named call or a record value; a single-input action takes its value directly. [user, paraphrase] Errors: a named call that omits a field with no default does not compile; a positional call with two or more inputs is refused by the IR generator at the call, naming the action.
- **R2:** `a()` is every field at its declared default, and a named call fills every omitted field from its default; nothing in the framework or the IR generator reads a domain's first value to fill an input. [user] Errors: a computed default (anything but an enum case or a literal of the field's domain) is refused by the IR generator at the record, naming the field.
- **R3:** A step function takes `(s, in)` and reads inputs by field selection, and the arity-typed `~>` ladder is removed. [paraphrase] Errors: a step function bound to an action of another record does not compile; `copy`, a pattern match or a method call on an input record inside a function body is refused by the IR generator as outside the liftable subset, with the same text and position discipline as the state's.
- **R4:** The classes, class keys, `Param` names and types, examples and schemas of every migrated action are byte-identical in the IR; `model/ir/*.json`, `model/cases/*.json` and every existing fixture's expected IR do not change, and the step-table equivalence passes. [inferred] Errors: any byte difference stops the migration until it is traced to its cause.
- **R5:** A capability law or realization helper that applies to any action with a given input states it as a structural bound, the framework reads the field by label without reflection, and at least one existing shared-input consumer (the realization's `deadlines`, which today names a token) is migrated to a selector on the record. [user] Errors: a bound naming a field the record lacks does not compile.
- **R6:** The token form (`input[T]`, `.input(token)`, `Assigned`, `:=` on classes, `Slot`, the `InputAt` overloads) and the IR generator's token-identity sites are removed together with the migration, with no compatibility layer. [inferred] Errors: a Model that still writes a token form does not compile.
- **R7:** The lifter's fixtures hold a record action, a single-input action and an action with no inputs, a record whose default is not its domain's first value, a structural-bound consumer, a positional refusal, a computed-default refusal and a `copy`-in-body refusal. [inferred] Errors: each reject fixture refuses with the stated text at its position.
- **R8:** The README's inputs paragraph, SEMANTICS step 3 and the sugar file's `Core form:` comments describe records, defaults, the two action shapes and the structural bound; `:=` appears in neither as a class form. [inferred] Errors: none (documentation).
- **R9:** Before the framework changes, an inventory records every action with inputs, every `:=` class call, every effect that takes inputs, every consumer of token identity and the field order and default that keep each action's IR identical; the migration is checked against it. [inferred] Errors: a consumer of token identity found after the inventory stops the task that finds it until the inventory is amended.
- **R10:** fn-159.10 (explicit defaults on tokens) is either landed before this spec and superseded by it, or skipped in its favour; the decision is recorded on both. [inferred] Errors: none.

**Error cases (negative-cases discipline):** each criterion above states its error cases inline or records that it has none.

## Boundaries
<!-- scope: business -->

- Runs after fn-141 Part A (sugar defined once) and Part B (declarations export by running), so the class lowering is written once against the final lifter. Done earlier, about fifty lifter sites would be rewritten and then deleted by fn-141. [inferred]
- No IR schema change, no checked-in IR or Case change, no Go change. [inferred]
- Out of scope: per-input types (option 2), shared nested records (option 3), input traits (option 4), named tuples over tokens (option 5), the request-scope `:=`, tags on fields (fn-144) and settings (fn-125). [paraphrase]

## Decision Context
<!-- scope: both -->

Five ways to keep "one input across several actions" a fact the framework knows were weighed on 2026-10-11, and the owner chose the first. [user]

- **Option 1 (chosen): identity by name and type, structural bound.** No declaration; a consumer that needs the field says `[In <: { def scheduleToStart: Timeout }]` and the compiler checks it. Cheapest; for the Temporal API same name plus same type is the same concept.
- **Option 2: one type per shared input** (`opaque type ScheduleToStart = Timeout`). Strongest identity, but every value is wrapped and five timeouts become five types with the same two cases.
- **Option 3: a shared nested record** (`Timeouts` inside `Start` and `Schedule`). A real declaration, natural where a group of inputs is one API message, but calls gain a level and class keys gain a segment unless flattened.
- **Option 4: a trait per shared input** that records extend. Option 1's checks with an explicit declaration that repeats the field list.
- **Option 5: keep tokens and assemble a named tuple** from them. Most faithful to today, but the token must carry its name as a literal type, calls need double parentheses, and named-tuple selection lifts differently from a field read.

The record form was itself weighed against keeping tokens and making them read better (fn-159.10). Records won on symmetry with the state: one mechanism for "a step reads two records", one lifting rule, one catalog derivation, and parameter names and defaults that Scala already documents.

## Early proof point

Task 2 (framework) is the proof: an action declared from a record, called empty and by name, bound to a `(s, in)` step function, with `Finite[In]` enumerating the same classes in the same order as today's `Class` list for `start`. If the catalog order or the `Param` names cannot be made identical, the migration stops before any Model changes.

## Execution

Order: inventory (task 1), framework (2), lifter (3), realization kit (4), Models (5), removal and docs (6). Tasks 2 to 4 land behind the old form, both compiling, so each Model migrates in one step against a working framework; task 6 removes the old form last. The step-table equivalence runs after task 5 and again after task 6.

## Quick commands

```sh
mise exec -- scala-cli compile model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/irgen
make lint-model-syntax
make umpire-check-equivalence BASE=<baseline> MAP=<map> STEPS_ONLY=1
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Record declaration; empty, named and value classes; single-input value shape | 2, 3, 5 | |
| R2 | Declared defaults; no first-value rule; computed defaults refused | 2, 3 | |
| R3 | `(s, in)` step functions; field selection only; ladder removed | 2, 3, 6 | |
| R4 | IR byte-identical; equivalence | 1, 5, 6 | |
| R5 | Structural bound for shared inputs; `deadlines` by selector | 2, 4 | |
| R6 | Token form and lifter token sites removed | 6 | |
| R7 | Fixtures | 3 | |
| R8 | Docs | 6 | |
| R9 | Inventory | 1 | |
| R10 | Relation to fn-159.10 | 1 | |

## Scope: the mini feature only (owner, 2026-10-11)

Within the Scala DSL batch (`MILESTONES.md`), this spec changes only the framework, the lifter, the shared code and `features/activity_mini`. The real features (standalone activity, Nexus workflow, Nexus standalone and their legacy payloads) are not edited, migrated, lifted or regenerated, and they stay behind until the owner's sweep, which runs this spec's scalafix rules on them. Equivalence compares `FEATURES=activity_mini` against the mini baseline. This overrides any statement in this spec that edits, migrates or proves on a real feature. See fn-166's Scope. Task .5 migrates the mini feature and the shared code; the real Models migrate by the rule at the sweep.
