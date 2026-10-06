---
satisfies: [R1, R2, R12]
---
# fn-133-lean-typed-realizations.2 Kit evidence modules: described status, history evidence, request base

## Description
Part A, R1, R2, R12.

- **`describedStatus`.** One declaration of a describe method, its info field, its operation key and a fact→status table. It yields the evidence of each fact and an `await(fact)`. The activity and standalone Nexus realizations use it, and their `status`/`awaitStatus` helpers and per-fact await vals go.
- **History evidence.** One declaration of a workflow's history-event kinds: fact → attributes field, keyed by the field naming the operation (e.g. `scheduledEventId`). It yields the exhaustive kinds and the set the closing read `closes`. The Nexus caller's five `historyKind` calls go.
- **A request base.** A realization declares the fields every call on a role carries (namespace, operation id from `run`) once. `rpc` and `await` apply them, and an explicit assignment overrides them.

Refusal fixtures: a fact listed twice in a table; an await of an unlisted fact; a history entry with no operation key. Lint: an override equal to the base.

## Acceptance
- [ ] No feature file defines `status`/`awaitStatus`/`historyKind`, or assigns `namespace` or the operation id without overriding the base.
- [ ] Each refusal fixture is refused at its line; the lint reports a redundant override.
- [ ] A before/after projection is identical.
- [ ] The spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
