---
satisfies: [R9, R10]
---
# fn-165-inputs-as-records-named-classes.1 Inventory the input surface and freeze field order, defaults and action shapes

## Description
**Note 2026-10-18 (one-language series):** Sweep proofs use the in-process IR (fn-201), fn-200's native lint and fn-203's Cases, not `model/ir` or `.lint.json`.

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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: