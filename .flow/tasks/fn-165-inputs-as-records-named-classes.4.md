---
satisfies: [R5]
---
# fn-165-inputs-as-records-named-classes.4 Realization kit names inputs by field selector

## Description
**Note 2026-10-18 (one-language series):** Sweep proofs use the in-process IR (fn-201), fn-200's native lint and fn-203's Cases, not `model/ir` or `.lint.json`.

The realization kit names inputs by field, not by token (R5).

**Size:** S
**Files:** `model/temporal/realize/*.scala` (`deadlines`, class presets, `withFields`), `model/irgen/Realizations.scala` (the preset and deadline readers around `inputTokens`), the activity and nexus realization kits
**Touches:** [model/temporal/realize/*.scala, model/irgen/Realizations.scala, model/temporal/features/*/realization/*.scala]

### Approach
- `deadlines` names an input by a selector on the action's record, `_.scheduleToStart`, and its reader resolves the selector to the field label and index; the "names its input by its token's val" refusal becomes "by a field of its record".
- Class presets are record values (`Start(...)`), read as task 3 reads them; the "must leave Timeout input at its default" check compares against the record's parameter defaults.
- A helper that applies to any action with a given input is written with the structural bound of task 2 and used at least once where the kit today repeats itself per action.
- `withFields { request.x := v }` is unchanged; note in the kit's syntax file that `:=` is the request-scope assignment only.
## Acceptance
- [ ] `deadlines` and class presets name inputs by field selector and record value; no realization file names an input token.
- [ ] The realization IR of every Model is byte-identical.
- [ ] At least one kit helper applies across actions through a structural bound.


## Done summary
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
