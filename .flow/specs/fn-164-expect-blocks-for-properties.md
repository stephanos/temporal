# Expect blocks for Properties

> HTML render lens: [.flow/artifacts/fn-164-expect-blocks-for-properties/spec.html](../artifacts/fn-164-expect-blocks-for-properties/spec.html) — local-only (gitignored): open it from the working tree; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (turn 1): "this is hideous / we need to be able to express this in a super readable way" (about `heartbeatExhausts`)
> user (turn 9): "I find this very hard to parse"
> user (turn 10): "the property syntax doesn't click for me. can you suggest alterantives?"
> user (turn 12): "how does that expect {} snyrax scale to the temporal logic support we need"
> user (turn 13): "I like expect {} / but help me understand how `is` and `record` are used. and compare to quint and fizzbee."
> user (turn 14): "\"implies\" is confusing to me / I like expect {} looks like effect {} / but I undestsnd the bool adavnatege / would we ever have `any` ?"
> user (turn 15): "yes, write a new spec. attach the Scala DSL changes to the existing \"s\" batch (see milestones.md)"
> user (turn 15, aside): "phase = timedOut is too clever"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 50% [user], 40% [paraphrase], 10% [inferred] -->

A Model author should be able to read a Property as a short list of what a step does. Today a same-step Property is `property when <action> holds { after => <one Boolean expression> }`. Every clause has to fit into one expression of `&&`, `||` and `implies`, so a two-clause promise such as `heartbeatExhausts` becomes nested parentheses that the owner found "hideous" and "very hard to parse". The owner said the property syntax "doesn't click", and that `implies` in particular is confusing. [user]

The fix is an `expect { }` block that reads like the `effect { }` block it checks: one requirement per line, all of which hold of the step, with `if` choosing which requirements apply. [paraphrase] The owner chose it because it "looks like effect {}", accepting the statement form over a Boolean `all(...)`/`any(...)` form despite the Boolean form's advantages. [user]

## Architecture & Data Models
<!-- scope: technical -->

**Two layers.** [paraphrase] The framework already separates temporal quantification from predicates: a Query's `verify`/`find`, the claim patterns (`once … keeps`, `stays … unless`, `never … from`), bounded `leadsTo` and monitors quantify over paths, and the predicate inside reads one step (or the state before it and the step). No predicate may read a claim over a path. `expect { }` replaces only the inner predicate language for steps, as `is { }` already does for states. Temporal operators stay outer combinators and take these blocks as arguments. A future TLA-named operator such as `eventually` is one more combinator, never new block vocabulary.

**The block.** [paraphrase] Each statement of an `expect { }` block is a requirement on the step. The block holds where every requirement that applies holds. The vocabulary:

- `records(fact)`: the step records the fact (the reading counterpart of an effect's `record(fact)`).
- `check(condition)`: any other Boolean requirement, including a phase comparison such as `check(phase == timedOut)`.
- `if condition then <requirements> [else <requirements>]`: the requirements apply only where the condition holds. This replaces `implies` and the `!a || b` idiom.
- `oneOf { <branches> }`: at least one branch's requirements hold. It is the rare escape hatch for genuine "one of these outcomes" Properties, and most disjunctions become an `if` or a phase set (`.in(...)`).

**What a field name reads.** [inferred] In a same-step block, a field read by name is the step's resulting state. A transition block also names the state before the step as `before`, and every read of the earlier state goes through `before`. This differs from `effect { }`, where a bare field is the state before the step, so the API docs must say it explicitly.

**Lowering.** [paraphrase] A block lowers to the core Boolean expression it stands for: requirements combine with AND, `if c then r` lowers to `!c || r` (the core form of `implies`), `if/else` to the corresponding conditional, and `oneOf` to OR. The lowering adds no IR construct, so the Go consumers and the Quint export see the same expressions as today.

**Data from the current Models.** [paraphrase] Of 166 Property bodies in the Temporal Models, 22 use `||` and 5 use `implies`. Most of the `||` uses are hidden implications (`!a || b`) or phase sets. Only about two Properties, the activity reset settling Property and the Nexus closed-run Property, are genuine "one of these outcomes" disjunctions. That is why `oneOf` is a rare tool and not part of the everyday vocabulary.

## API Contracts
<!-- scope: technical -->

- [paraphrase] A same-step Property is declared on an action and finished by an `expect { }` block, e.g. `after(<action>) expect { … }`. The exact head word is settled at plan time; it must not reuse `when`, which means "in this phase" in rules.
- [paraphrase] A transition Property is declared the same way, with `before` in scope inside the block.
- [paraphrase] `records`, `check`, `if` and `oneOf` are the only statements. Each returns `Unit`, so the block compiles under `-Wnonunit-statement` without exemptions.
- [inferred] A step fact used inside an `if` condition (a Boolean, not a requirement) has a distinct query name from the `records` requirement, so that one word does not mean two things.
- [inferred] Every new sugar definition documents its `Core form:` like the existing sugar, and it lives in the framework's sugar file.

## Edge Cases & Constraints
<!-- scope: technical -->

- [paraphrase] A block that contains a temporal operator or reads a claim over a path fails to compile, which enforces the existing semantics rule by types instead of convention.
- [paraphrase] Statements are not sequential in time: every line is about the same step. Docs and the `expect` name must not suggest "first this, then that".
- [inferred] An empty `expect { }` block is refused: a Property that requires nothing is an authoring mistake.
- [inferred] `oneOf` with fewer than two branches is refused.
- [inferred] Composition Properties use the composition fact form (`records(_.member, fact)`) inside the block.
- [paraphrase] fn-141 makes a sugar's definition its only lowering, resolved by generic expansion. The block's definition must fit that model, as `effect { }` and `is { }` blocks must, or fn-141 must carry it explicitly.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** [paraphrase] A same-step Property can be declared with an `expect { }` block whose statements are `records(fact)`, `check(condition)`, `if … then … [else …]` and `oneOf { … }`. The Property holds of a step exactly where every applicable requirement holds. Errors: an empty block, a `oneOf` with fewer than two branches, and a non-`Unit` statement are refused at compile or lift time, naming the position.
- **R2:** [paraphrase] A transition Property can be declared with the same block, with the state before the step named `before`. Bare field reads in either form read the step's resulting state. Errors: a reference to `before` in a same-step block fails to compile.
- **R3:** [user] The block vocabulary has no `implies` and no assignment-shaped assertion (`phase = x`). A conditional requirement is written with `if`, and a phase requirement as `check(phase == x)`. Errors: no error surface beyond the compile.
- **R4:** [paraphrase] Temporal forms stay outside the block: Queries, the claim patterns, `leadsTo` and monitors keep their meaning, and their predicate arguments accept `is { }` / `expect { }` blocks where they take a state or step predicate. Errors: a temporal operator or path claim inside a block fails to compile.
- **R5:** [paraphrase] The IR generator lowers a block to the core Boolean expression it stands for, adding no IR construct. Each converted Property's IR equals its pre-conversion IR apart from source positions, and every machine's step table is unchanged. Errors: any other IR difference is explained before it is accepted.
- **R6:** [inferred] Every Property of the Temporal Models written as a `holds`/`holdsAcross` lambda is migrated to `expect { }`, and the lambda forms are removed from the authoring surface together, with no compatibility layer. Errors: a Property that cannot be expressed is listed with its reason before the old form is removed.
- **R7:** [inferred] The model README's authoring guidance and each sugar definition's `Core form:` describe `expect { }`, the field-reading rule of R2, and when to use `oneOf`.
- **R8:** [user, 2026-10-10] Use Scala 3 colon-and-indentation blocks consistently across the Scala DSL, including `rules`, `on`, `from`, `effect`, `is`, `expect` and `oneOf`, wherever Scala admits that spelling. Migrate the Models, framework examples, fixtures and author documentation together, and enforce the style through the existing formatter/check workflow. Prefer the pinned Scalafmt's built-in optional-braces rewrite; add a focused existing gate or Scalafix check only for gaps the formatter cannot enforce. Required Scala braces (such as import selectors) remain. Errors: formatting/lint must reject a reintroduced optional brace block in the covered authoring surface, accept nested colon blocks, and distinguish DSL blocks from comments, strings and unrelated calls. Compilation, lifting and step-table equivalence must prove that the migration changes no behavior; lifted changes are limited to source positions.

## Boundaries
<!-- scope: business -->

- [user] No Boolean `all(...)`/`any(...)` predicate form: the owner chose the statement form that mirrors `effect { }`.
- [user] No `phase = x` assertion syntax ("too clever").
- [paraphrase] No new temporal operators. Adding `eventually` or similar is separate work that would take these blocks as arguments.
- [inferred] Query, Scenario and realization syntax stay with fn-140 and fn-162.
- [user, 2026-10-10] Colon-block consistency is part of this Scala spec and runs after the `expect` implementation/migration, before the meaning-preserving seal. It adds no Go work or production regeneration to the current Scala-only mode.

## Decision Context
<!-- scope: both -->

### Motivation

- [user] Readability is the goal. The owner rejected the current lambda form ("hideous", "very hard to parse") and a helper-based rewrite of `heartbeatExhausts` that still combined `&&` and `implies`.
- [paraphrase] Statement form over Boolean `all`/`any`: Quint's `all { }`/`any { }` keeps everything Boolean, which avoids the requirement/condition split and makes disjunction first-class. But the data shows disjunction is rare, and the owner prefers the `effect { }` mirror. FizzBee puts the temporal modality as a keyword before an ordinary function body (`always assertion`, `transition assertion`), which confirms the two-layer split this spec keeps.
- [paraphrase] Umpire's step facts and action-scoped claims are richer than either Quint or FizzBee, which model events as state. `records(fact)` stays first-class in the block for that reason.
- [inferred] Scheduled in batch S as a meaning-preserving migration, so it runs before the seal's step-table equivalence and the batch G regeneration absorbs the source-position changes.

## Parked unknowns

- The head word for declaring a Property on an action (`after(action) expect`, or something else). Resolved at plan time against fn-140's and fn-162's Query and Property forms.
- Whether `oneOf` or a `match` on the phase reads better for the two genuine disjunctions. Resolved by converting the activity reset settling Property first.

## Requirement coverage

| R-ID | Task(s) |
| --- | --- |
| R1 | TBD - populate via /flow-next:plan |
| R2 | TBD - populate via /flow-next:plan |
| R3 | TBD - populate via /flow-next:plan |
| R4 | TBD - populate via /flow-next:plan |
| R5 | TBD - populate via /flow-next:plan |
| R6 | TBD - populate via /flow-next:plan |
| R7 | TBD - populate via /flow-next:plan |
| R8 | fn-164-expect-blocks-for-properties.1 |
