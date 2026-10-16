# Expect blocks for Properties

## Native DSL task order (owner override)

Execute .2 and independent .11 in parallel isolated workspaces, with serialized compiler access and integration. Task .11 owns native colon/block lint enforcement; task .2 owns expectation DSL and its SyntaxRule boundary. After both committed checkpoints, .10 completes native caller/API/colon migration and formatter idempotence, consuming the native portions of .7-.9. Tasks .3 and .7-.9 remain held for complete capture/exporter fixture migration and exact positioned IR proofs. Tasks .1/.4-.6 retain their original broader migration obligations. Native completion does not close these tasks or their full R-IDs.

After committed .10, release fn-165.7's task admission hold. Flowctl forbids cross-spec task edges; the conductor releases this explicit hold instead of requiring whole-spec closure. Native implementation and checks exclude Temporal entirely. Broad export verification and all IR/Go/backend work follow the framework-native DSL queue.

## Framework-only execution amendment

The owner requires native DSL work before IR, exporter, Go and backend work, and excludes `model/temporal` entirely. Native task .2 starts from committed framework source checkpoint `437016b7d1` with its passing native core/check suites. Fn-188.2's broad export verification remains incomplete and owner-held; it does not gate native DSL work. Run focused native compile/runtime and syntax checks rooted only in `model/framework`. Do not run exporter/capture/IR parity or broad `make check-framework` as this native task's gate. Never consume `model/project.scala`, Temporal Model/API jars or disabled Temporal sources.

The first native proofs preserve explicit contextual parameter names for later exact IR comparison, compiler refusals for discarded Boolean/path/step expressions, and the exact composition field plus machine identity, including repeated same-machine members. A plain Step does not establish composition ownership; owner-free functions must retain enough evidence to validate their eventual attachment. These are implementation proof obligations, not inferred passes.

Original tasks .4/.5 and the full-scope .6/.1 seals remain held for Temporal migration and later reconciliation. Task .3 and the IR-coupled .7/.8/.9 acceptance remain open for the later IR phase. Split native caller migration, public lambda finisher removal and colon-style cleanup into native checkpoints before execution rather than waiting for .3 or claiming full IR-coupled tasks complete. The final native DSL surface removes the old finishers; disabled Temporal callers and deferred exporter fixtures do not justify compatibility methods. Full Temporal R6/R8 remain open. Later generic lifting still owes exact function/declaration inventories, expression shapes, eager lets and parameter identities. No Go/schema/backend work or regeneration is authorized. Native completion does not complete the whole spec.

| Framework task | Requirement contribution | Dependency |
| --- | --- | --- |
| .7 caller migration | R2, R4, R5, R6, framework only | .3 |
| .8 public finisher removal and docs | R4, R5, R6, R7, framework only | .7 |
| .9 colon enforcement and final framework check | R5, R7, R8, framework only | .8 |

DSL changes ship a task-named Scalafix rule under `model/migrate/rules/`, prove it on copied `activity_mini` sources before the real-feature sweep, and follow `model/migrate/README.md`; production loading remains held in fn-166.10.

> HTML render lens (local): open `.flow/artifacts/fn-164-expect-blocks-for-properties/spec.html` - regenerable, markdown is the record. <!-- flow-next:artifact-link -->

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
- `oneOf: <branches>`: at least one branch's requirements hold. Each alternative is a `branch:` block containing conjunctive requirements; `branch` is a structural delimiter only inside `oneOf`, not a fifth assertion kind. Most disjunctions become an `if` or a phase set (`.in(...)`).

**What a field name reads.** [inferred] In a same-step block, a field read by name is the step's resulting state. A transition block also names the state before the step as `before`, and every read of the earlier state goes through `before`. This differs from `effect { }`, where a bare field is the state before the step, so the API docs must say it explicitly.

**Lowering.** [paraphrase] A block lowers to the core Boolean expression it stands for: requirements combine with AND, `if c then r` lowers to `!c || r` (the core form of `implies`), `if/else` to the corresponding conditional, and `oneOf` to OR. The lowering adds no IR construct, so the Go consumers and the Quint export see the same expressions as today. Conjunction and alternatives preserve short-circuit evaluation: failure stops the enclosing requirement block, alternatives isolate requirement failures and stop after the first success, and specification holes or unrelated exceptions propagate. A conditional-only nonempty block holds when no requirement applies; it is not an empty declaration.

Preservation includes Boolean association, conditional shape, eager local-let placement, named-predicate forwarding and core function parameter identity. Explicitly named contextual bindings provide the old state/Step parameter names where a migrated declaration needs them; ordinary blocks can use implicit contexts and the standard accessors. The early proof must show this is readable and liftable, including synthetic parameter names, before caller migration. A generic alpha-normalization or wrapper-function exemption is not an equivalence proof.

**Data from the current Models.** [paraphrase] Of 166 Property bodies in the Temporal Models, 22 use `||` and 5 use `implies`. Most of the `||` uses are hidden implications (`!a || b`) or phase sets. Only about two Properties, the activity reset settling Property and the Nexus closed-run Property, are genuine "one of these outcomes" disjunctions. That is why `oneOf` is a rare tool and not part of the everyday vocabulary.

## API Contracts
<!-- scope: technical -->

- A same-step Property uses `property after(<class>) expect:`, or `property expect:` without a restriction. The head does not reuse `when`, which means "in this phase" in rules. `after` retains exact-class restriction; a separate `afterAction` retains the existing whole-action restriction rather than silently treating the two alike.
- A transition Property uses the same builder with `.across expect:`, providing typed `before: S` as well as typed `after: Step[S, O, F]`. A same-step block has `after` but no `before` evidence. Bare fields read `after.state` in both forms. The contextual binding naming form must preserve existing lifted parameter names when migrating them.
- `records`, `check`, `if` and `oneOf` are the requirement vocabulary; `branch:` structurally groups an alternative inside `oneOf`. Requirements and structural blocks return `Unit`, without warning exemptions. Immutable Boolean helper locals are permitted where preserving existing predicate lets requires them; arbitrary declarations are not requirements.
- A fact test in a condition uses the typed step's Boolean query, `after.records(fact)`, distinct from the Unit requirement `records(fact)`. Composition requirements use `.records(_.member -> MemberMachine, fact)` to retain the selected machine's fact type and exact field identity.
- Reusable same-step and transition expectation predicates have ordinary core function types and can be passed inline or by name to outer combinators. They work under generic `Declares` capability constructors and compositions, not just machine-only `Owner` contexts.
- Every new sugar definition documents its `Core form:` like the existing sugar, and it lives in the framework's sugar file; core declaration storage remains independent of these contexts.

## Edge Cases & Constraints
<!-- scope: technical -->

- [paraphrase] A block that contains a temporal operator or reads a claim over a path fails to compile, which enforces the existing semantics rule by types instead of convention.
- [paraphrase] Statements are not sequential in time: every line is about the same step. Docs and the `expect` name must not suggest "first this, then that".
- An empty expectation, an empty branch and `oneOf` with fewer than two branches are refused. A discarded final Boolean expression is refused too: Scala Unit adaptation is not an assertion.
- Composition fact requirements use the typed field-to-machine form from the API contract; same-named facts on different members cannot satisfy each other's requirements.
- The predicate-scope compile protection covers locally bound and discarded path/temporal declarations and step-making expressions, not only mismatched arguments to `check`. Outer machine givens cannot accidentally make such declarations legal inside the block.
- [paraphrase] fn-141 makes a sugar's definition its only lowering, resolved by generic expansion. The block's definition must fit that model, as `effect { }` and `is { }` blocks must, or fn-141 must carry it explicitly.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** [paraphrase] A same-step Property can be declared with an `expect` block whose requirements are `records(fact)`, `check(condition)`, `if … then … [else …]` and `oneOf` with explicit `branch` blocks. The Property holds of a step exactly where every applicable requirement holds, preserving short-circuit behavior and propagating holes. Errors: an empty block or branch, `oneOf` with fewer than two branches, and a non-`Unit` statement (including discarded final Boolean expressions) are refused at compile or lift time, naming the position.
- **R2:** [paraphrase] A transition Property can be declared with the same block, with the state before the step named `before`. Bare field reads in either form read the step's resulting state. Errors: a reference to `before` in a same-step block fails to compile.
- **R3:** [user] The block vocabulary has no `implies` and no assignment-shaped assertion (`phase = x`). A conditional requirement is written with `if`, and a phase requirement as `check(phase == x)`. Errors: no error surface beyond the compile.
- **R4:** [paraphrase] Temporal forms stay outside the block: Queries, the claim patterns, `leadsTo` and monitors keep their meaning, and their predicate arguments accept `is { }` / `expect { }` blocks where they take a state or step predicate. Errors: a temporal operator or path claim inside a block fails to compile.
- **R5:** [paraphrase] The IR generator lowers a block to the core Boolean expression it stands for, adding no IR construct. Each converted Property's IR equals its pre-conversion IR apart from source positions, and every machine's step table is unchanged. Errors: any other IR difference is explained before it is accepted.
- **R6:** Every Property of the Temporal Models written as a `holds`/`holdsAcross` lambda is migrated to `expect`, and the lambda forms are removed from the authoring surface together, with no compatibility layer. Errors: a Property that cannot be expressed is listed with its reason before the old form is removed. Planning confirmed this inference against the current Property/capability inventory and the requested complete Scala authoring migration.
- **R7:** The model README's authoring guidance and each sugar definition's `Core form:` describe `expect`, the field-reading rule of R2, and when to use `oneOf`/`branch`. Planning confirmed these documentation owners in the existing authoring/sugar guidance. Errors: no error surface beyond documentation consistency.
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

## Early proof point

The first runtime/API task proves requirement short-circuiting, explicit alternative grouping, typed before/after contexts and contextual binding names. The lifting task then proves exact core-IR preservation for heartbeat and reset-shaped paired fixtures, local lets and named predicate forwarding before caller migration. If either proof fails, resolve the API/lowering conflict before migrating or removing the old surface; do not accept a generic equivalence exemption.

## Execution

The fn-140 source foundation now includes primary `.records` or `.excludes`, ordered conditional `.recordsWhen` clauses and additional absent-fact guards. Consume their actual committed native/core/helper/source-span receipts at admission, not the older positive-only witness API. Preserve those witnesses and their generated ordinary Properties; expectation migration must not recreate deleted private triples, drop forbidden facts, merge same-spelled composed facts or normalize conditional helper bodies/parameters. The readable heartbeat/reset fixture proofs remain mandatory even when those production Queries have become witnesses.

Consume the committed fn-140 witness source seal and fn-149 grouping source seal, then implement expectation runtime, lifting/bridges, foundation migration, feature migration and final surface/docs cleanup. The existing colon-style task depends on that final cleanup and completes the pre-fault meaning-preserving seal. fn-123 Scala fault work starts only after this seal is committed.

No whole-spec dependency on fn-140 or fn-149 is added: their final tasks still wait for owner-deferred shared non-Scala closure. These committed source gates are explicit conductor admission requirements, not full-spec completion. fn-141 later consumes this settled expect surface and must lower its sugar through the generic definition-expansion contract. No publication is required for local source admission. Go, full gates, managed regeneration, independent review and live runs remain deferred.

## Quick commands

```sh
mise exec -- scala-cli compile model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/irgen
mise exec -- scala-cli test model/check
make lint-model-syntax
make umpire-check-equivalence BASE=.flow/tmp/fn160/baseline-ae3f36ee51 MAP=model/testdata/equivalence/batch-1d.map STEPS_ONLY=1
```

Every migration compares scratch lifts against its frozen predecessor with exact declaration/function coverage and explicit source-span mapping. Run heavy commands serially. Checked-in IR/Cases remain unchanged.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | [paraphrase] A same-step Property can be declared with an `expect` block whose requirements are `records(fact)`, `check(condition)`, `if … then … [else …]` and `oneOf` with explicit `branch` blocks. The Property holds of a step exactly where every applicable requirement holds, preserving short-circuit behavior and propagating holes. Errors: an empty block or branch, `oneOf` with fewer than two branches, and a non-`Unit` statement (including discarded final Boolean expressions) are refused at compile or lift time, naming the position. | fn-164-expect-blocks-for-properties.2, fn-164-expect-blocks-for-properties.3 | — |
| R2 | [paraphrase] A transition Property can be declared with the same block, with the state before the step named `before`. Bare field reads in either form read the step's resulting state. Errors: a reference to `before` in a same-step block fails to compile. | fn-164-expect-blocks-for-properties.10, fn-164-expect-blocks-for-properties.2, fn-164-expect-blocks-for-properties.3, fn-164-expect-blocks-for-properties.5, fn-164-expect-blocks-for-properties.7 | — |
| R3 | [user] The block vocabulary has no `implies` and no assignment-shaped assertion (`phase = x`). A conditional requirement is written with `if`, and a phase requirement as `check(phase == x)`. Errors: no error surface beyond the compile. | fn-164-expect-blocks-for-properties.10, fn-164-expect-blocks-for-properties.2, fn-164-expect-blocks-for-properties.3 | — |
| R4 | [paraphrase] Temporal forms stay outside the block: Queries, the claim patterns, `leadsTo` and monitors keep their meaning, and their predicate arguments accept `is { }` / `expect { }` blocks where they take a state or step predicate. Errors: a temporal operator or path claim inside a block fails to compile. | fn-164-expect-blocks-for-properties.10, fn-164-expect-blocks-for-properties.2, fn-164-expect-blocks-for-properties.3, fn-164-expect-blocks-for-properties.5, fn-164-expect-blocks-for-properties.6, fn-164-expect-blocks-for-properties.7, fn-164-expect-blocks-for-properties.8 | — |
| R5 | [paraphrase] The IR generator lowers a block to the core Boolean expression it stands for, adding no IR construct. Each converted Property's IR equals its pre-conversion IR apart from source positions, and every machine's step table is unchanged. Errors: any other IR difference is explained before it is accepted. | fn-164-expect-blocks-for-properties.3, fn-164-expect-blocks-for-properties.4, fn-164-expect-blocks-for-properties.5, fn-164-expect-blocks-for-properties.6, fn-164-expect-blocks-for-properties.7, fn-164-expect-blocks-for-properties.8, fn-164-expect-blocks-for-properties.9 | — |
| R6 | Every Property of the Temporal Models written as a `holds`/`holdsAcross` lambda is migrated to `expect`, and the lambda forms are removed from the authoring surface together, with no compatibility layer. Errors: a Property that cannot be expressed is listed with its reason before the old form is removed. Planning confirmed this inference against the current Property/capability inventory and the requested complete Scala authoring migration. | fn-164-expect-blocks-for-properties.10, fn-164-expect-blocks-for-properties.4, fn-164-expect-blocks-for-properties.5, fn-164-expect-blocks-for-properties.6, fn-164-expect-blocks-for-properties.7, fn-164-expect-blocks-for-properties.8 | — |
| R7 | The model README's authoring guidance and each sugar definition's `Core form:` describe `expect`, the field-reading rule of R2, and when to use `oneOf`/`branch`. Planning confirmed these documentation owners in the existing authoring/sugar guidance. Errors: no error surface beyond documentation consistency. | fn-164-expect-blocks-for-properties.10, fn-164-expect-blocks-for-properties.2, fn-164-expect-blocks-for-properties.6, fn-164-expect-blocks-for-properties.8, fn-164-expect-blocks-for-properties.9 | — |
| R8 | [user, 2026-10-10] Use Scala 3 colon-and-indentation blocks consistently across the Scala DSL, including `rules`, `on`, `from`, `effect`, `is`, `expect` and `oneOf`, wherever Scala admits that spelling. Migrate the Models, framework examples, fixtures and author documentation together, and enforce the style through the existing formatter/check workflow. Prefer the pinned Scalafmt's built-in optional-braces rewrite; add a focused existing gate or Scalafix check only for gaps the formatter cannot enforce. Required Scala braces (such as import selectors) remain. Errors: formatting/lint must reject a reintroduced optional brace block in the covered authoring surface, accept nested colon blocks, and distinguish DSL blocks from comments, strings and unrelated calls. Compilation, lifting and step-table equivalence must prove that the migration changes no behavior; lifted changes are limited to source positions. | fn-164-expect-blocks-for-properties.1, fn-164-expect-blocks-for-properties.10, fn-164-expect-blocks-for-properties.11, fn-164-expect-blocks-for-properties.9 | — |

Native tasks contribute only their stated portions. Held tasks retain the complete IR/integration requirements; coverage is assignment, not completion.

## Scope: the mini feature only (owner, 2026-10-11)

Within the Scala DSL batch (`MILESTONES.md`), this spec changes only the framework, the lifter, the shared code and `features/activity_mini`. The real features (standalone activity, Nexus workflow, Nexus standalone and their legacy payloads) are not edited, migrated, lifted or regenerated, and they stay behind until the owner's sweep, which runs this spec's scalafix rules on them. Equivalence compares `FEATURES=activity_mini` against the mini baseline. This overrides any statement in this spec that edits, migrates or proves on a real feature. See fn-166's Scope. Task .4 migrates the shared capability and foundation predicates and task .5 the mini feature's Properties. The real features' Properties migrate by the rule at the sweep.

## Redesign impact (fn-173, 2026-10-13)

Owner-approved redesign `fn-173` (reachable exploration; `model/design/reachable-exploration.md`) drops full catalog tables, Scala step tables and equivalence sealing, and Query totals; the Behavior Fingerprint becomes an IR structural hash. For this spec:

- **R5 evidence:** "every machine's step table is unchanged" and the `STEPS_ONLY` commands become "IR equal apart from positions" (an unchanged structural hash).
