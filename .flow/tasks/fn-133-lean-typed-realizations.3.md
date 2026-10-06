---
satisfies: [R4, R5, R14]
---
# fn-133-lean-typed-realizations.3 One instruction convention, own names, named evidence arguments, ids from facts

## Description
Part B, R4, R5, R14 (ids).

- **Lower-case instruction forms.** Feature files write only `fault`, `hold`, `release`, `finish` and `command`. The upper-case case classes stay core forms. Lint: an upper-case instruction form in a feature file.
- **No borrowed names.** `Command("release-dispatch", …)` (activity lost race) and `Command("start-nexus-operation", …)` (Nexus caller schedule) are the two today. Decide between evidence naming both commands and an explicit kit alias that the evidence line shows, and record the decision in the spec. The lifter refuses two commands of one script with one name unless the alias declares it.
- **`withFields` matching documented.** The kit and README say that evidence naming a call matches all its `withFields` variants.
- **Named arguments.** `delivered(…, attempt = 1, after = startActivity, …)` and `answeredAs`. No evidence kind is a free string; ids come from facts or declared constants (`evidenceId("scheduled")`, `sourceId("history")` go).

## Acceptance
- [ ] The lint finds no upper-case instruction form in a feature file, and no feature file builds a `Command` with an explicit id outside the alias.
- [ ] The alias decision is recorded in the spec's Decision Context.
- [ ] No feature file passes a string literal to `evidenceId`/`sourceId`/`answeredAs`.
- [ ] A before/after projection is identical apart from command ids the alias decision changes, which are listed.
- [ ] The spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
