---
satisfies: [R3, R6, R8]
---
# fn-165-inputs-as-records-named-classes.6 Remove the token form and := on classes; document the record form

## Description
Remove the token form and document the record form (R3, R6, R8).

**Size:** S
**Files:** `model/framework/Action.scala`, `model/framework/Syntax.scala`, `model/framework/Machine.scala`, `model/README.md` (inputs paragraph), `model/SEMANTICS.md` (step 3, the three-symbols note in step 21), `model/framework/Domain.scala` comments if they name tokens, the syntax lint's sugar list
**Touches:** [model/framework/*.scala, model/README.md, model/SEMANTICS.md]

### Approach
- Delete `input[T]`, `Input`, `Slot`, `Assigned`, `:=` on classes, the `InputAt` overloads, `ActionDecl.tokens`, and any remaining arity-typed `~>`; a Model that writes a token form must fail to compile.
- Docs: inputs are a record; a class is empty, named or a value; a single-input action takes its value; a step reads `in.field`; a shared input is a structural bound; the DSL's symbols are `~>` and `->`.
- Run the step-table equivalence once more and the syntax lint.

## Acceptance
- [ ] The token form is gone from the framework; a fixture that writes `input[T]` or `:=` on a class fails to compile.
- [ ] README and SEMANTICS describe the record form and list two DSL symbols.
- [ ] The step-table equivalence and `make lint-model-syntax` pass.


## Done summary
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
