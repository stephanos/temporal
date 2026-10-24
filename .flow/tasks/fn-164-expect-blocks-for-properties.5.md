---
satisfies: [R2, R4, R5, R6]
---
# fn-164-expect-blocks-for-properties.5 Migrate feature and actor Properties with a complete parity inventory

## Description
**Note 2026-10-18 (one-language series):** IR parity proofs use the in-process internal IR (fn-201); no checked-in IR.

**Size:** M
**Files:** feature/actor Property and predicate callers with their adjacent Scala tests.
**Touches:** [model/temporal/features/**, model/temporal/actors/**]

Convert the production feature/actor side of R6 after the foundation migration, retaining each owning facet's fn-149 safety/liveness placement and fn-140 witness/Query identities.

### Approach
- Freeze exact scratch IR plus independent Property/function inventory from the committed predecessor. Begin with heartbeatExhausts and activity resetSettles: preserve the fact/implication AND tree and the reset's eager restarted let, six/three alternatives and conditional shape. Use readable explicit branch groups, not a giant Boolean expression hidden in one check unless a small grouped expression is required to retain association.
- Migrate all remaining holds/holdsAcross lambda Properties, including shared claim constructors, compositions and monitor/pattern predicate users. Use after/afterAction restrictions without changing exact-class versus all-input semantics. Keep names, Query readers, realization bindings, safety/liveness classification and named helper identities.
- Preserve before/after parameter spellings through the proven contextual binding form; bare fields read resulting state. A same-step block never obtains before from unrelated lexical evidence.
- For each owner, compare complete scratch IR against the frozen predecessor with explicit source-span mappings and a comprehensive inventory, and run its Scala regression tests. No alpha-normalization, wrapper deletion or Boolean reassociation in the comparator.
- Inventory unsupported bodies before removing any old authoring API. Resolve them inside the approved predicate vocabulary without dropping promises, moving evaluation or weakening scope.

### Investigation targets
**Required:** `model/temporal/features/activity/standalone/protocol/Heartbeat.scala:77`; `model/temporal/features/activity/standalone/protocol/Protocol.scala:227-253`; `model/temporal/features/activity/standalone/protocol/Reset.scala:100-135`; production `properties.safety/liveness` and shared claim functions; composition/monitor callers discovered by the source inventory.
**Optional:** capability migration evidence/source maps and existing feature Scala regression suites.

### Quick commands
```sh
mise exec -- scala-cli compile model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/project.scala model/framework model/temporal
make lint-model-syntax
make umpire-check-equivalence BASE=.flow/tmp/fn160/baseline-ae3f36ee51 MAP=model/testdata/equivalence/batch-1d.map STEPS_ONLY=1
```

Scratch lifts and evidence only; no production managed regeneration/full gates/Go/review/live.
## Acceptance
- [ ] Every production feature/actor lambda Property and required shared predicate caller migrates, proved by independent complete inventories (R6).
- [ ] Heartbeat/reset remain readable and preserve exact expression/let/conditional/parameter identity; complete owner scratch lifts differ only at mapped positions (R5).
- [ ] Class/action restrictions, composed facts, outer patterns/monitors, Query identities, groups/facets and realization bindings remain intact (R2/R4/R5).
- [ ] Scala compile/regressions, syntax and step-table equivalence pass; unsupported-body inventory is empty before handoff to removal.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: