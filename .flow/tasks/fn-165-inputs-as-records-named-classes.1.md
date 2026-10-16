---
satisfies: [R9, R10]
---
# fn-165-inputs-as-records-named-classes.1 Inventory the input surface and freeze field order, defaults and action shapes

## Description
Freeze what the migration must preserve before any framework change (R9, R10).

**Size:** S
**Files:** `.flow/specs/fn-165-*.md` (inventory appended under Execution), `model/temporal/**/*.scala` (read only)
**Touches:** [.flow/specs/fn-165-inputs-as-records-named-classes.md]

### Approach
- List every action with inputs (today about six: `client.start`, `client.schedule`, the nexus `respond`/`start` forms, the task queue's), its tokens in order, each token's domain and the value the first-value rule gives it today. That table is each record's field order and defaults.
- List every class call: positional, `:=` (52 on 2026-10-11: `maxAttempts` 22, `startToClose` 9, `scheduleToStart` 9, `heartbeat` 6, `scheduleToClose` 4, `startDelay` 2), and empty (`start()` 31, `heartbeat()` 16, `schedule()` 8, ...), by file.
- List every step function that takes inputs (bound through the arity-typed `~>`), with its parameter list.
- List every consumer of token identity: the realization's `deadlines` and class presets (`irgen/Realizations.scala` around `inputTokens` / `tokens.indexOf`), `Syntax.scala`'s `Assigned` matcher, capability laws that name an input, `.example` uses.
- Decide and record: which actions are single-input (value shape) and which take a record; whether fn-159.10 lands first and is superseded, or is skipped in favour of this spec (R10), and write the decision on fn-159.10 too.

## Acceptance
- [ ] The spec's Execution section holds the inventory: per action its record field order and defaults; every class call by form and file; every input-reading step function; every consumer of token identity.
- [ ] The single-input/record decision per action and the fn-159.10 decision are recorded on this spec and on fn-159.10.


## Done summary
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
