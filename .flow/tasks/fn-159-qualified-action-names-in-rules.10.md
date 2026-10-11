---
satisfies: [R17]
---
# fn-159-qualified-action-names-in-rules.10 Explicit input defaults: start() is every declared default, no first-value rule

## Description
Make an input's default explicit on its token, so that `start()` and a named call that omits inputs mean what they say rather than "the first value of each domain" (`model/SEMANTICS_FEEDBACK.md` 3.3).

**Today.** `start()` is `Class(decl, decl.domains.map(_.values.head))` (`framework/Action.scala`), and `start(maxAttempts := MaxAttempts.two)` fills every omitted input the same way (`Syntax.scala`, the `Assigned*` apply). That reads correctly only because the Models declare each input enum with its neutral case first (`Timeout.unset`, `MaxAttempts.unlimited`); the framework does not know that, and reordering an enum, or declaring one whose first case is not neutral, changes every empty and named class in every Scenario without a text change. The IR generator's `firstValue` (`irgen/Types.scala`) lowers the empty call the same way.

**Size:** S
**Files:** `model/framework/Action.scala` (`Input`, `input`, `apply()`), `model/framework/Syntax.scala` (the named `apply`), `model/irgen/Types.scala` (`firstValue` at empty and named calls), `model/irgen/Declarations.scala` (input token declarations), every `val x = input[T]` whose action is called empty or partially (`StandaloneActivity.scala`, the nexus features, the task queue), lift and reject fixtures, `model/README.md`, `model/SEMANTICS.md` step 3
**Touches:** [model/framework/Action.scala, model/framework/Syntax.scala, model/irgen/Types.scala, model/irgen/Declarations.scala, model/temporal/**/*.scala, model/irgen/testdata/**/*.scala, model/README.md, model/SEMANTICS.md]

### Approach
- **Framework.** `input[T](default = v)` records the default on the token (`Input[A]` gains `default: Option[A]`); `input[T]` without one declares an input that every class must supply. `apply()` and the named `apply` take each omitted input's declared default and refuse, with the input's name, an omitted input that has none. The refusal fires where the class is built, so a machine, Scenario or realization that holds such a call fails to initialize.
- **Lifter.** The empty and named calls lower by reading the token's default from its declaration instead of `firstValue`; an omitted input without a default is refused at the call. `firstValue` stays for its other uses, if any remain, or goes.
- **Models.** Add `default = Timeout.unset` and `default = MaxAttempts.unlimited` (and the equivalents in the nexus and task queue features) to every token that is ever omitted. Choose each default as the value the first-value rule gives today, so no class, Scenario or Case changes; an inventory of omitted inputs before the change is the check.
- **Fixtures.** One lift fixture with a declared default that is not the domain's first value, showing the class takes the default; one reject fixture for an omitted input with no default.
- **Docs.** The README's inputs paragraph and SEMANTICS step 3 state the rule: an omitted input is one with a declared default, `start()` is every default, and no form reads catalog order.
- **Out of scope.** The positional form and the `:=` symbol stay as they are; the owner is weighing a named-argument form for inputs (a parameter list or case class per action), which replaces `:=` and is captured as fn-165 (inputs as records). fn-165 R10 records whether this task lands first and is superseded, or is skipped in its favour.

### Batching
- Batch 1d (Scala DSL, immediate; see MILESTONES.md): Scala and lifter changes only, IR unchanged by construction (every declared default equals today's first value). Verify with the Scala build, the framework, Model and lifter tests, and the step-table equivalence.

### Quick commands
```sh
make umpire-check-model
make umpire-check-lint
```

## Acceptance
- [ ] `input[T](default = v)` declares a token's default; `start()` and a named call take each omitted input's declared default, and nothing in the framework or the IR generator reads a domain's first value to fill an input.
- [ ] An empty or named call that omits an input with no default is refused by the framework at construction and by the IR generator at the call, naming the input (reject fixture).
- [ ] A lift fixture whose declared default is not the domain's first value lifts the class with the declared default.
- [ ] Every token the Models omit somewhere declares the default the first-value rule gave it; `model/ir/*.json` and every existing fixture's expected IR are byte-identical; the step-table equivalence passes.
- [ ] README and SEMANTICS state the rule.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
