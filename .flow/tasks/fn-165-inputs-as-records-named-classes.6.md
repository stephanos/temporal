---
satisfies: [R3, R6, R8]
---
# fn-165-inputs-as-records-named-classes.6 Remove the token form and := on classes; document the record form

## Description
**Note 2026-10-18 (one-language series):** Sweep proofs use the in-process IR (fn-201), fn-200's native lint and fn-203's Cases, not `model/ir` or `.lint.json`.

Remove the token form and document the record form (R3, R6, R8).

**Size:** S
**Files:** `model/framework/Action.scala`, `model/framework/Syntax.scala`, `model/framework/Machine.scala`, `model/README.md` (inputs paragraph), `model/framework/SEMANTICS.md` (step 3, the three-symbols note in step 21), `model/framework/Domain.scala` comments if they name tokens, the syntax lint's sugar list
**Touches:** [model/framework/*.scala, model/README.md, model/framework/SEMANTICS.md]

### Approach
- Delete `input[T]`, `Input`, `Slot`, `Assigned`, `:=` on classes, the `InputAt` overloads, `ActionDecl.tokens`, and any remaining arity-typed `~>`; a Model that writes a token form must fail to compile.
- Docs: inputs are a record; a class is empty, named or a value; a single-input action takes its value; a step reads `in.field`; a shared input is a structural bound; the DSL's symbols are `~>` and `->`.
- Run the step-table equivalence once more and the syntax lint.
## Acceptance
- [ ] The token form is gone from the framework; a fixture that writes `input[T]` or `:=` on a class fails to compile.
- [ ] README and SEMANTICS describe the record form and list two DSL symbols.
- [ ] The step-table equivalence and `make lint-model-syntax` pass.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: