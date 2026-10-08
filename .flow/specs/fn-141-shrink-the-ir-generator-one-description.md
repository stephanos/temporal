# Shrink the IR generator: one description of each DSL construct

## Conversation Evidence

> user (turn 1, part 1): "/Users/stephan/Workspace/skunkworks/umpire/temporal/model/irgen has so much code."
> user (turn 1, part 2): "we need to investigate ways to reduce it."
> user (turn 1, part 3): "can we adjust the scala DSL to support that?"
> user (turn 1, part 4): "but we don't want to sacrifice any expressiveness."
> user (turn 1, part 5): "we can make reasonable changes to the DSL, though, if they support this goal."
> user (turn 2): "for example, is there a way to integrate them better? would that help?"
> user (turn 3): ">  \"no inline, no macro, no runtime semantics\" where does that cme from?"
> user (turn 4, part 1): "1. Define sugar once (small change, roughly 500–700 lines). 2. Export declarations from the running Models; lift only function bodies (large change). 3. Move the lints out of irgen. 4. Retire spellings nothing uses."
> user (turn 4, part 2): "yes to all."
> user (turn 4, part 3): "write flow next spec (but mark as deferred for now); and update MILESTONES.md"
> user (turn 5): "> - Any Scala becomes legal at declaration level, such as a for that yields Queries. as long as we put rstirictions on where it matters ie we must still  be able to produce Quint/TLA exports etc and the Umpire IR cleanl"
> user (turn 6): "is it helpful to focus the lifter on the underlying concepts/concsutrcts and types - and have the syntactic sugar be sth it doesn't kow about as it's resolved automatically before it looks at the model?"
> user (turn 7): "are tasks broken up?"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 30% [user], 40% [paraphrase], 30% [inferred] -->

**Ready.** The owner approved executing this spec last, after the activity batch, fn-142/fn-143 preparation, fn-140, fn-123 and the schema chain fn-145 through fn-148. Its tasks were planned on 2026-10-06 against the earlier tree, so each task re-reads its files and recounts against that settled schema baseline before changing it. The byte-identity and expressiveness contracts below apply to that baseline.

A Model author and a framework maintainer both pay for the IR generator's size. `model/irgen` is 8,242 source lines against 2,958 for the framework it reads (`model/umpire`). Every new DSL word costs a tree matcher, a lowering, a refusal and two fixtures in the lifter. The owner wants that code reduced, accepts reasonable DSL changes to get there, and sets two limits. The DSL must lose no expressiveness. The Umpire IR must stay clean, so that the Quint export and later exports such as TLA+ keep working from it.

A full read of the lifter on 2026-10-06 gave this split. Line shares are estimates within about 20%. The lint total and the site counts are exact.

| What the code does | Lines | Share |
| --- | --- | --- |
| Refusals and validation (about 450 sites) | ~2,200 | 27% |
| Recognizing which DSL call a tree is | ~1,200 | 15% |
| Assembling IR messages | ~900 | 11% |
| Lifting function bodies and types | ~800 | 10% |
| Lowering sugar to core forms | ~750 | 9% |
| Names, IDs and positions | ~600 | 7% |
| Evaluating Scala at lift time | ~550 | 7% |
| Plumbing | ~1,200 | 15% |

Two facts follow from the split. About 10% of the lifter does work only a tree reader can do, which is lifting function bodies and types. The declaration-order lint, the structure lint and the marker checks total 1,479 lines and emit no IR.

The cause is that each construct is described twice, and each sugar three times. The Models already run as Scala. `Rules` registers rules and checks their overlap, the gate constructs every root of every IR file, and the builders build Properties, Scenarios and Queries. The lifter then derives the same declarations again from typed trees. Its `fold` is a hand-written evaluator for builder chains, lists, string interpolation and helper defs. Its rule lowering restates the framework's own `stepFunction`. Its sugar file re-implements each definition of the framework's sugar file. About a third of its refusals say "this spelling cannot be read", a limit that exists only because the lifter reads source where the Scala program holds values.

This spec removes the second description in four parts the owner approved together. It moves the lints out, retires unused spellings, defines sugar once, and exports declarations from the constructed Models so that the lifter lifts function bodies and types only.

## Architecture & Data Models
<!-- scope: technical -->

**Two levels.** [paraphrase] A Model has a declaration level and a function level. The declaration level says what exists: actions, machines, rules, compositions, Properties, Scenarios, Queries, capabilities and realizations. The function level says what a step, guard or predicate computes. Scala evaluates the declaration level when the Model is constructed. The function level is never executed for the IR. The lifter reads it from the typed tree and holds it to the liftable subset.

**The restriction sits at the function level.** [paraphrase] Declaration-level Scala is evaluated away before any IR exists, so an author may compute declarations with any Scala, such as a comprehension that yields Queries. Nothing of that computation reaches the IR. Function bodies stay in today's liftable subset, so the IR has the same shape and every reader of it, the Go interpreter, the lowering and the Quint export, is unaffected.

**The lifter and exporter know core constructs only.** [paraphrase] Sugar is resolved before either reads a Model. Declaration-level sugar (the claim patterns, `sticky`, named inputs, the realization helpers) resolves by running. It is plain Scala that evaluates to core values. A sugar call inside a function body (`enter`, `stay`, `reject`, `in`, `implies`, `records`) cannot resolve by running, and the compiler's `inline` does not help because TASTy is written before inlining. One generic expansion replaces such a call with its definition's body, the call's arguments bound, before the lifter matches anything. The expansion names no sugar.

**Capture points.** [inferred] The framework records two things where an author writes them. A declaration records its name, its Definition ID and its position, through a context parameter the framework supplies. A function passed to a declaration records its source span and the values it closes over. These are the only places the framework uses `inline` or a macro.

**The exporter.** [inferred] One exporter in the framework walks the constructed roots of an IR file, as the gate's construction test walks them today, and emits each declaration's IR. Where a declaration holds a function, the exporter asks the lifter for the function at the recorded span, with the captured values bound.

**What the lifter keeps.** [paraphrase] Lifting a function body to an IR expression, lifting the types it reaches, the generic sugar expansion, and the lookup of a function by span.

**Evidence the mechanism works.** [inferred] A spike on 2026-10-06 with Scala 3.9.0 recorded, at run time, the name, qualified path and line of a declaration from its `val` or object, including through a helper def and inside a comprehension. It also recorded the line span of a lambda and the values it closed over. A captured plain function stayed opaque, which is why a helper's function-valued parameter takes a recorded-function type. The spike did not test finding a tree by its span inside the TASTy inspector.

**Order of delivery.** [paraphrase] Part C (lints out) and Part D (unused spellings) come first and change no behavior. Part A (sugar once) follows. Part B (export) goes one declaration kind at a time and starts with realizations, which hold no step functions and whose lifter file is 1,373 lines.

## API Contracts
<!-- scope: technical -->

- [paraphrase] **Unchanged:** the IR schema, the `lift` command line and the gate's commands, the text and position of every refusal that survives, and every file under `model/ir` and `model/cases`.
- [inferred] **Helper def that declares:** takes the framework's naming context as a context parameter, so the declaration it returns is named after the `val` that calls it.
- [inferred] **Function-valued parameter of a helper def:** takes the framework's recorded-function type in place of a plain Scala function type. A def reference or a lambda converts to it at the call site.
- [paraphrase] **Sugar definition:** a plain `def` or `extension` whose body is written in core forms. It is the sugar's only lowering. A kit may define its own.
- [paraphrase] **Factory:** every framework and kit factory returns a value that holds everything its arguments said.

## Edge Cases & Constraints
<!-- scope: technical -->

- [inferred] Two declarations made under one `val`, such as two Queries yielded by one comprehension, derive the same name. Each needs a name of its own, and the export refuses the pair.
- [inferred] A sugar's `Core form:` doc comment and its body can disagree once the body is the lowering. The body is authoritative, and the syntax lint's doc rule keeps the comment present.
- [inferred] Export depends on constructed values, so a `val` read before it is declared yields a wrong IR where today it yields only a missed overlap check. The initialization rules of the declaration-order lint keep running first.
- [inferred] A refusal raised while a Model is constructed has a stack trace and no tree. It must still name the author's file and line, which the capture points supply.
- [inferred] fn-131 moves source positions beside the tree. Whichever spec lands second adapts to the other. The byte-identity checks here compare against the baseline of their own step.
- [inferred] The lifter and the Models must agree on one build of the Models. The export constructs the same compiled classes whose TASTy the lifter reads.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** [paraphrase] `model/irgen` holds only code that produces IR. The declaration-order lint, the structure lint and the marker checks run as their own step of the model gate, before any IR file is written, and a refusal of theirs still stops the lift. Errors: a source one of them refuses writes no IR file; each of their reject fixtures refuses with the same text at the same position from its new home.
- **R2:** [paraphrase] Every spelling of a construct that no Model and no kit file uses is removed from the framework, the lifter and the lifter's fixtures. The work recounts uses at its start and lists each candidate with its count. The candidates on 2026-10-06 were hand-bound `Bindings`, the hand-written `Typed*` realization constructors, `Instruction.readUntil`, the receiver-less `capabilities(limits)(…)`, the chained `.overriding(…)` and `.total(n)`. Errors: a spelling with one or more uses under `model/temporal` stays; a spelling that a kept sugar's definition is written in stays; a removed spelling no longer compiles, and its fixtures are deleted with it.
- **R3:** [paraphrase] The lifter and the exporter read core constructs and types only. Sugar is resolved before either reads a Model: declaration-level sugar by running, and a sugar call inside a function body by one generic expansion that replaces the call with its definition's body. The syntax lint holds that no file of the lifter or the exporter names a sugar definition. Errors: the lint fails naming the file, the line and the sugar; a sugar definition whose body cannot be lifted is refused naming the sugar and the position of its definition.
- **R4:** [paraphrase] A sugar's definition is its only lowering. The hand-written lowerings of `enter`, `stay`, `reject`, `disabled`, `in`, `implies`, `records`, the claim patterns, `sticky`, `stickyAcross`, named inputs and the request-scope assignment are gone. Adding a sugar definition to the framework or a kit needs no lifter or exporter change, which a fixture kit with a sugar of its own shows. IR nodes from an expanded body carry the position of the call. Errors: a sugar body that calls itself, directly or through another sugar, is refused naming the cycle.
- **R5:** [paraphrase] Every framework and kit factory builds, at run time, the complete value it declares. Nothing an author wrote is discarded or left empty for the lifter to fill: a request scope keeps its assignments, an evidence value keeps its operation and fields, a condition keeps its operator, and a `sync` and a `replaces` record what they pair. Errors: a test constructs one value per factory and fails naming the factory and the argument it cannot read back.
- **R6:** [paraphrase] The IR of every declaration comes from the constructed Models. The gate constructs each IR file's roots, and one exporter emits their actions and inputs, channels, assumptions, holes, monitors, machines with their rules and derivations, compositions with their members and syncs, Properties, Scenarios, Queries, progress claims, capability expansions and realizations. The lifter evaluates no declaration: it folds no builder chain, binds no helper def's arguments for a declaration and follows no `val` to one. Errors: a root that throws while it is constructed is refused naming the root and the declaration being constructed; nothing is written once any root failed.
- **R7:** [paraphrase] Each declaration records, where the author writes it, its name, its Definition ID and its file and line, with the values the lifter derives today from the `val` or object that declares it. An action's run-time name is its name, so the rules' overlap message needs no `codeOf`. Errors: a declaration with no `val` or object to name it and no name given is refused at its position; two declarations that derive one ID are refused naming both positions.
- **R8:** [paraphrase] Every function a declaration takes records the source span of the function and the values it closes over. This covers guards, effects, Property and progress predicates, monitor functions, `end`, refinement maps, evidence and a capability's function fields. The lifter lifts the function found at that span and binds each captured value: a value of a finite Model type becomes the IR value it is, and a recorded function becomes a call of it. Errors: a captured value that is neither is refused at the function's position naming the captured name and its type; a span at which the lifter finds no function is refused naming the declaration.
- **R9:** [paraphrase] Only the declaration level is free. A function body is held to the liftable subset as today: every function-level reject fixture refuses with the same text at the same position. No IR node, field or name records how a declaration was computed, and the IR schema does not change. Errors: a comprehension, a loop or a call outside the Models inside a function body is refused as outside the liftable subset.
- **R10:** [paraphrase] Export is deterministic, and the IR's readers run unchanged. The same sources give byte-identical IR on every run: no output reads object identity, hash order, initialization order, time or environment. The Quint export needs no change, and `make umpire-check-backends` gives the answers it gave before. Errors: [inferred] the lifter's fixture test exports each fixture in two separate JVM runs and fails on any byte difference, naming the first differing line.
- **R11:** [inferred] Part B moves one declaration kind at a time, realizations first. After each step every file under `model/ir` and `model/cases` and every expected IR of the lifter's fixtures is byte-identical to that step's baseline, and no kind is produced by both the tree path and the exporter. Errors: any byte difference stops the step until it is traced to its cause.
- **R12:** [inferred] Each refusal kind the lifter has when the work starts gets one recorded outcome. Deleted: the spelling it refused now exports, and its reject fixture becomes a lift fixture. Kept: enforced in one place, by the compiler, by the framework at construction or by the exporter, with the author's file and line. Left to Go: where the Go reader already repeats it. Errors: a kept refusal whose reject fixture no longer refuses at its line fails the fixture test; a refusal raised at construction without an author position is a defect.
- **R13:** [paraphrase] No Model and no kit file loses a spelling it uses. The only edits to Models are the two mechanical ones named under API Contracts. Declaration-level Scala the lifter refuses today exports, which a fixture shows by yielding Queries from a comprehension, each with its own name. Errors: a Model edit of any other kind is put to the owner before it is made.
- **R14:** [inferred] The initialization rules of the declaration-order lint, a `val` read before it is declared and a cycle of owners, run before export. Errors: a `null` or half-made declaration reached during export is refused naming the declaration that holds it.
- **R15:** [paraphrase] The rules of record say what holds afterwards. `.plans/DSL_OPERATORS.md` rule 5, the note on fn-113's R15 and the model README state that the framework uses `inline` and macros only at its capture points, that Models and sugar use neither, and that every declaration has a run-time value (no error surface beyond the docs being updated).
- **R16:** [inferred] Each part records the source lines of `model/irgen` before and after, with the lines it adds to the framework and the gate. The realization step is the proof point: if it removes fewer than half of the realization lifter's lines, net of the lines it adds, work stops and the owner decides whether Part B continues. Errors: no error surface beyond the stop.

**Error cases (negative-cases discipline):** each criterion above states its error cases inline or records that it has none.

## Early proof point

Task fn-141-shrink-the-ir-generator-one-description.5 validates the core approach of Part B: the framework records a declaration's name and position and a function's span and captured values, and the lifter lifts the function found at that span to the IR the tree path gives. If it fails, Part B stops there. Parts C, D and A (tasks 1 to 4) stand without it.

Task fn-141-shrink-the-ir-generator-one-description.8 is the size proof (R16). If exporting realizations removes fewer than half of the realization lifter's lines, net of the lines tasks 7 and 8 add, work stops and the owner decides whether tasks 9 to 13 run.

## Quick commands

```bash
scala-cli test model/irgen
make lint-model
make umpire-check-model
make umpire-gen-model   # the diff of model/ir and model/cases must be empty
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/...
```

## Boundaries
<!-- scope: business -->

- [paraphrase] The IR schema and the Go consumers do not change. The Quint export is not edited.
- [paraphrase] Function bodies are still read from typed trees after compilation. This spec adds no macro that lifts a function body.
- [paraphrase] The reading-order and layout rules of the two lints keep their content. Whether about 1,300 lines of layout policy are worth keeping is a separate decision for the owner.
- [inferred] No new sugar, no renamed construct and no new IR node.
- [inferred] A third-party library for name capture is adopted only by fn-113 R25's weighing, with its line counts recorded.
- [inferred] The line-level repetition inside the lifter (wrapper stripping, parallel evidence arms) is cleaned only where a part already rewrites the code.

## Decision Context
<!-- scope: both -->

### Motivation

[user] The owner asked whether the DSL and the lifter could "integrate" better and whether that would help. [paraphrase] The inventory answers yes: the lifter's largest categories, refusals, call recognition, sugar lowering and lift-time evaluation, exist because the lifter derives from source what the running Scala program already holds. [user] The owner approved all four parts with "yes to all". [paraphrase] The owner then bounded the freedom Part B gives: restrictions stay "where it matters", so that the IR stays clean and the Quint and TLA+ exports remain producible. R9 and R10 carry that bound. [paraphrase] The owner also proposed that the lifter focus on the underlying constructs and types and never see sugar. R3 states it as an invariant with a check.

### Implementation Tradeoffs

[paraphrase] Where "no inline, no macro, no runtime semantics" came from. The inline rule is rule 5 of the research note `.plans/DSL_OPERATORS.md`, whose stated reason is that the lifter matches calls by name. It cites fn-113 R25, which is about weighing libraries and names neither. The run-time rule is fn-113 R15, which retired the native evaluator. fn-126 R16 has since put a run-time check back: rule disjointness runs when a machine is constructed. `Rules.on` is already `inline` so that `codeOf` can name an action whose run-time name is empty. fn-113 dismissed `sourcecode` because "nothing reads a name at runtime", which stopped being true with fn-126.

[inferred] Three alternatives were rejected. Lifting function bodies with a macro fails on bodies defined in the same compilation run, which `.plans/SCALA.md` records. Building expressions from a typed expression type removes native `match` and `copy` from step functions, which costs expressiveness. Recovering a lambda's identity from its serialized form ties the IR to the compiler's synthetic names.

[inferred] Realizations go first in Part B because their declarations are data, they hold no step functions, and their lifter file is the largest. The projection for the whole spec is that the non-lint lifter falls from about 6,800 lines to between 3,000 and 3,500, counting the new exporter. It is a projection from the inventory. R16 makes the first step test it.

[inferred] The checked-in IR and Cases are the oracle for every step. Each part must leave them byte-identical, so a difference is a defect of the step and never an accepted change.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | [paraphrase] `model/irgen` holds only code that produces IR. The declaration-order lint, the structure lint and the marker checks run as their own step of the model gate, before any IR file is written, and a refusal of theirs still stops the lift. Errors: a source one of them refuses writes no IR file; each of their reject fixtures refuses with the same text at the same position from its new home. | fn-141-shrink-the-ir-generator-one-description.1 | — |
| R2 | [paraphrase] Every spelling of a construct that no Model and no kit file uses is removed from the framework, the lifter and the lifter's fixtures. The work recounts uses at its start and lists each candidate with its count. The candidates on 2026-10-06 were hand-bound `Bindings`, the hand-written `Typed*` realization constructors, `Instruction.readUntil`, the receiver-less `capabilities(limits)(…)`, the chained `.overriding(…)` and `.total(n)`. Errors: a spelling with one or more uses under `model/temporal` stays; a spelling that a kept sugar's definition is written in stays; a removed spelling no longer compiles, and its fixtures are deleted with it. | fn-141-shrink-the-ir-generator-one-description.2 | — |
| R3 | [paraphrase] The lifter and the exporter read core constructs and types only. Sugar is resolved before either reads a Model: declaration-level sugar by running, and a sugar call inside a function body by one generic expansion that replaces the call with its definition's body. The syntax lint holds that no file of the lifter or the exporter names a sugar definition. Errors: the lint fails naming the file, the line and the sugar; a sugar definition whose body cannot be lifted is refused naming the sugar and the position of its definition. | fn-141-shrink-the-ir-generator-one-description.3, fn-141-shrink-the-ir-generator-one-description.4 | — |
| R4 | [paraphrase] A sugar's definition is its only lowering. The hand-written lowerings of `enter`, `stay`, `reject`, `disabled`, `in`, `implies`, `records`, the claim patterns, `sticky`, `stickyAcross`, named inputs and the request-scope assignment are gone. Adding a sugar definition to the framework or a kit needs no lifter or exporter change, which a fixture kit with a sugar of its own shows. IR nodes from an expanded body carry the position of the call. Errors: a sugar body that calls itself, directly or through another sugar, is refused naming the cycle. | fn-141-shrink-the-ir-generator-one-description.3, fn-141-shrink-the-ir-generator-one-description.4, fn-141-shrink-the-ir-generator-one-description.8, fn-141-shrink-the-ir-generator-one-description.9 | — |
| R5 | [paraphrase] Every framework and kit factory builds, at run time, the complete value it declares. Nothing an author wrote is discarded or left empty for the lifter to fill: a request scope keeps its assignments, an evidence value keeps its operation and fields, a condition keeps its operator, and a `sync` and a `replaces` record what they pair. Errors: a test constructs one value per factory and fails naming the factory and the argument it cannot read back. | fn-141-shrink-the-ir-generator-one-description.11, fn-141-shrink-the-ir-generator-one-description.7 | — |
| R6 | [paraphrase] The IR of every declaration comes from the constructed Models. The gate constructs each IR file's roots, and one exporter emits their actions and inputs, channels, assumptions, holes, monitors, machines with their rules and derivations, compositions with their members and syncs, Properties, Scenarios, Queries, progress claims, capability expansions and realizations. The lifter evaluates no declaration: it folds no builder chain, binds no helper def's arguments for a declaration and follows no `val` to one. Errors: a root that throws while it is constructed is refused naming the root and the declaration being constructed; nothing is written once any root failed. | fn-141-shrink-the-ir-generator-one-description.10, fn-141-shrink-the-ir-generator-one-description.11, fn-141-shrink-the-ir-generator-one-description.12, fn-141-shrink-the-ir-generator-one-description.13, fn-141-shrink-the-ir-generator-one-description.8, fn-141-shrink-the-ir-generator-one-description.9 | — |
| R7 | [paraphrase] Each declaration records, where the author writes it, its name, its Definition ID and its file and line, with the values the lifter derives today from the `val` or object that declares it. An action's run-time name is its name, so the rules' overlap message needs no `codeOf`. Errors: a declaration with no `val` or object to name it and no name given is refused at its position; two declarations that derive one ID are refused naming both positions. | fn-141-shrink-the-ir-generator-one-description.5, fn-141-shrink-the-ir-generator-one-description.9 | — |
| R8 | [paraphrase] Every function a declaration takes records the source span of the function and the values it closes over. This covers guards, effects, Property and progress predicates, monitor functions, `end`, refinement maps, evidence and a capability's function fields. The lifter lifts the function found at that span and binds each captured value: a value of a finite Model type becomes the IR value it is, and a recorded function becomes a call of it. Errors: a captured value that is neither is refused at the function's position naming the captured name and its type; a span at which the lifter finds no function is refused naming the declaration. | fn-141-shrink-the-ir-generator-one-description.10, fn-141-shrink-the-ir-generator-one-description.12, fn-141-shrink-the-ir-generator-one-description.5 | — |
| R9 | [paraphrase] Only the declaration level is free. A function body is held to the liftable subset as today: every function-level reject fixture refuses with the same text at the same position. No IR node, field or name records how a declaration was computed, and the IR schema does not change. Errors: a comprehension, a loop or a call outside the Models inside a function body is refused as outside the liftable subset. | fn-141-shrink-the-ir-generator-one-description.12, fn-141-shrink-the-ir-generator-one-description.14, fn-141-shrink-the-ir-generator-one-description.5 | — |
| R10 | [paraphrase] Export is deterministic, and the IR's readers run unchanged. The same sources give byte-identical IR on every run: no output reads object identity, hash order, initialization order, time or environment. The Quint export needs no change, and `make umpire-check-backends` gives the answers it gave before. Errors: [inferred] the lifter's fixture test exports each fixture in two separate JVM runs and fails on any byte difference, naming the first differing line. | fn-141-shrink-the-ir-generator-one-description.14, fn-141-shrink-the-ir-generator-one-description.8 | — |
| R11 | [inferred] Part B moves one declaration kind at a time, realizations first. After each step every file under `model/ir` and `model/cases` and every expected IR of the lifter's fixtures is byte-identical to that step's baseline, and no kind is produced by both the tree path and the exporter. Errors: any byte difference stops the step until it is traced to its cause. | fn-141-shrink-the-ir-generator-one-description.10, fn-141-shrink-the-ir-generator-one-description.11, fn-141-shrink-the-ir-generator-one-description.12, fn-141-shrink-the-ir-generator-one-description.13, fn-141-shrink-the-ir-generator-one-description.8, fn-141-shrink-the-ir-generator-one-description.9 | — |
| R12 | [inferred] Each refusal kind the lifter has when the work starts gets one recorded outcome. Deleted: the spelling it refused now exports, and its reject fixture becomes a lift fixture. Kept: enforced in one place, by the compiler, by the framework at construction or by the exporter, with the author's file and line. Left to Go: where the Go reader already repeats it. Errors: a kept refusal whose reject fixture no longer refuses at its line fails the fixture test; a refusal raised at construction without an author position is a defect. | fn-141-shrink-the-ir-generator-one-description.10, fn-141-shrink-the-ir-generator-one-description.11, fn-141-shrink-the-ir-generator-one-description.12, fn-141-shrink-the-ir-generator-one-description.13, fn-141-shrink-the-ir-generator-one-description.6, fn-141-shrink-the-ir-generator-one-description.8, fn-141-shrink-the-ir-generator-one-description.9 | — |
| R13 | [paraphrase] No Model and no kit file loses a spelling it uses. The only edits to Models are the two mechanical ones named under API Contracts. Declaration-level Scala the lifter refuses today exports, which a fixture shows by yielding Queries from a comprehension, each with its own name. Errors: a Model edit of any other kind is put to the owner before it is made. | fn-141-shrink-the-ir-generator-one-description.12, fn-141-shrink-the-ir-generator-one-description.14 | — |
| R14 | [inferred] The initialization rules of the declaration-order lint, a `val` read before it is declared and a cycle of owners, run before export. Errors: a `null` or half-made declaration reached during export is refused naming the declaration that holds it. | fn-141-shrink-the-ir-generator-one-description.8 | — |
| R15 | [paraphrase] The rules of record say what holds afterwards. `.plans/DSL_OPERATORS.md` rule 5, the note on fn-113's R15 and the model README state that the framework uses `inline` and macros only at its capture points, that Models and sugar use neither, and that every declaration has a run-time value (no error surface beyond the docs being updated). | fn-141-shrink-the-ir-generator-one-description.14, fn-141-shrink-the-ir-generator-one-description.5 | — |
| R16 | [inferred] Each part records the source lines of `model/irgen` before and after, with the lines it adds to the framework and the gate. The realization step is the proof point: if it removes fewer than half of the realization lifter's lines, net of the lines it adds, work stops and the owner decides whether Part B continues. Errors: no error surface beyond the stop. | fn-141-shrink-the-ir-generator-one-description.1, fn-141-shrink-the-ir-generator-one-description.14, fn-141-shrink-the-ir-generator-one-description.2, fn-141-shrink-the-ir-generator-one-description.4, fn-141-shrink-the-ir-generator-one-description.8 | — |
