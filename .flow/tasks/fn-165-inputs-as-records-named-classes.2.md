---
satisfies: [R1, R2, R3, R5]
---
# fn-165-inputs-as-records-named-classes.2 Framework: actions from a record of inputs, (s, in) step functions, structural shared inputs

## Description
**Note 2026-10-18 (one-language series):** Sweep proofs use the in-process IR (fn-201), fn-200's native lint and fn-203's Cases, not `model/ir` or `.lint.json`.

The framework side: an action declared from a record of inputs, called empty, by name or by value, bound to a `(s, in)` step function, with the catalog, class order and `Param` names unchanged (R1, R2, R3, R5). This is the early proof point: if the class order or the `Param` names cannot be made identical to today's, stop here.

**Size:** M
**Files:** `model/framework/Action.scala` (`Action[I]`, `ActionDecl.tokens`/`domains`, the `InputAt` overloads, `apply()`), `model/framework/Syntax.scala` (`Assigned`, `:=`, the named `apply`), `model/framework/Machine.scala` (the `~>` arity ladder, `Rule.fires`, `run(s, inputs)`), framework tests (`Inputs.test.scala`, `Rules.test.scala`)
**Touches:** [model/framework/Action.scala, model/framework/Syntax.scala, model/framework/Machine.scala, model/framework/*.test.scala]

### Approach
- `Action[In]`: `action(actor).inputs[In](using Finite[In], Mirror.ProductOf[In])` for a record, `.input[A](using Finite[A])` for a single value; `ActionDecl.inputs` (the `Param` names) from `MirroredElemLabels` for a record and from the `val` name for a single input, as the lifter derives it today; `domains` from the record's field catalogs.
- Classes: `apply()` is `In`'s all-defaults value, which Scala builds from the parameter defaults; a named call is Scala's; `apply(in: In)` wraps a value. `Class.values` stays the positional list so keys and column order do not change; it is `in.productIterator`.
- `~>`: two extensions, `S => Steps` and `(S, In) => Steps`; `Rule.fires(in)` and `run(s, in)` take the record. Remove the six-arity ladder.
- Shared inputs: `Inputs.field[In](label)` reads a field by label through the `Mirror`; a structural bound `[In <: { def scheduleToStart: Timeout }]` is what a consumer writes. One framework test shows a law applied to two records with the field and refused for one without.
- Keep the token form compiling beside the new one until task 6, so Models migrate one at a time.
- Proof: a test enumerates `Finite[Start].values` and compares, element by element, with the `Class` list the token form produces for the same action.
## Acceptance
- [ ] `action(actor).inputs[In]` and `.input[A]` declare actions whose classes are the record's or the value's catalog, in the order the token form produced; a framework test compares the two lists for `start`.
- [ ] `a()`, `a(field = v)` and `a(In(...))` build classes; a named call that omits a field without a default does not compile.
- [ ] `a ~> f` accepts `S => Steps` and `(S, In) => Steps` only; the arity ladder is gone.
- [ ] A structural bound on `In` reads a shared field by label without reflection; a bound on a missing field does not compile.
- [ ] `Param` names of a record action equal its field names; of a single-input action, the `val` name.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: