---
satisfies: [R1, R3, R6, R7]
---
# fn-5-umpire-discovery-promotion-and-artifact.2 Assemble the Temporal production catalog and catalog executable

## Description
Compose current reusable, Switch, and Nexus metadata into one checked production registry, bind stable projection metadata, and expose the pure catalog executable required by downstream generation for R1/R3/R6/R7.

**Size:** M
**Files:** `model/Temporal/Tool/Catalog.lean`, `model/Temporal/Tool/CatalogFixtures.lean`, `model/Temporal/Tool/CatalogCli.lean`, `model/Temporal/Tool/CatalogTests.lean`, `model/Temporal/Tool/CatalogCliTests.lean`, `model/Temporal/Tool/Inspect.lean`, `model/TemporalModelTests.lean`, `model/lakefile.toml`
**Touches:** [model/Temporal/Tool/Catalog.lean, model/Temporal/Tool/CatalogFixtures.lean, model/Temporal/Tool/CatalogCli.lean, model/Temporal/Tool/CatalogTests.lean, model/Temporal/Tool/CatalogCliTests.lean, model/Temporal/Tool/Inspect.lean, model/TemporalModelTests.lean, model/lakefile.toml]

### Approach

- Import and reuse `Temporal.Tool.Catalog.Core` from fn-15 for exact selector parsing, canonical list/explain/check envelopes, ordering, and command-result behavior. Keep semantic closure and entry validation in the fn-5 catalog; do not redeclare the generic query engine or merge semantic entries with the API/config input catalogs.
- Define the production seed registry as exactly two public Switch checked Queries (`switch.query.exact-action`, `switch.query.exact-trace`), six public Nexus checked Queries (`temporal.nexus.basic-lifecycle.query.async-start`, `temporal.nexus.basic-lifecycle.query.successful-completion`, `workflow-nexus.query.verify-caller-closure`, `workflow-nexus.query.explore-caller-closure`, `workflow-nexus.query.exact-action-caller-closure`, and `workflow-nexus.query.model-only-caller-closure`), and fn-16's `temporal.nexus.basic-lifecycle.space.fault-matrix` checked-space metadata.
- Compute the least typed metadata closure: Query to checked Behavior/form properties/target; Space to its axes/choices/faults/goals and checked base Query; target to its target/kernel declarations plus declarations, providers, connectors, and their capability/law/meaning references. Merge equal IDs only when kind/digest/source agree. Check the result against a golden canonical identity/kind set: the Nexus partition is exactly 61 entries (BasicLifecycle 10, BasicOperations 6, VariationSpace 15, CallerClosure 30), while the Switch partition is exhaustively named by the same golden fixture.
- Include internal semantic rows in the checked graph with `internal` disposition. Exclude nested roles, setup constraints, occurrences, Property clause IDs, authored/`Except` intermediates, PlannerRun/artifact outputs, proof-only definitions, test fixtures, and wrong-trace examples because they are not first-class checked declaration metadata.
- Mark the current Switch exact-action and Nexus caller-closure scenarios as the initial `stableRegression` set.
- Define a separately checked Temporal `CatalogProjectionBinding` registry keyed by stable entry identity. Each binding owns the canonical inspector selector, repository-relative fixture path, and per-entry projection key; its binding identity is projected to JSON but does not affect reusable catalog semantic identity. Aggregate output paths remain one set-level generator concern.
- Implement pure deterministic `listCatalog` and `explainCatalog` results plus the effect-thin `temporal-model-catalog list|explain|check` executable while preserving the existing inspector's exact single-scenario behavior.
- Keep Temporal vocabulary in this owner layer, not under `model/Umpire`.

### Investigation targets

**Required:**
- `model/Temporal/Tool/Inspect.lean:1-77` — current closed scenario registry and error shell.
- `model/Temporal/Tool/Catalog/Core.lean` — fn-15-owned generic query and response mechanics; consume without adding semantic vocabulary to the core.
- `model/lakefile.toml:1-20` — existing executable registration.
- `model/Umpire/Examples/Switch.lean:307-611` — reusable scenario metadata.
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean:234-245` — ten public checked declaration identities.
- `model/Temporal/Feature/Nexus/Examples/BasicOperations.lean:45-292` — two public Property/Behavior/Query roots.
- `model/Temporal/Feature/Nexus/Examples/VariationSpace.lean` — fn-16-owned exact BasicLifecycle space metadata seed.
- `model/Temporal/Feature/Nexus/CallerClosure.lean:510-658` — production checked Behavior/Query and artifact.
- `model/Temporal/Feature/Nexus/CallerClosureTests.lean:71-97` — independent semantic checks.
- `model/TemporalModelTests.lean:1-10` — aggregate registration.

### Quick command

`cd model && mise exec -- lake build Temporal.Tool.CatalogTests Temporal.Tool.CatalogCliTests temporal-model-catalog TemporalModelTests`

## Acceptance
- [ ] The checked production catalog is exactly the least typed closure of the eight named Query seeds plus the one named Space seed and matches the golden canonical identity/kind set.
- [ ] Nexus contributes exactly 61 unique entries partitioned 10 BasicLifecycle, 6 BasicOperations, 15 VariationSpace, and 30 CallerClosure; missing/extra seeds or closure rows fail.
- [ ] Internal semantic rows remain in the graph but are hidden by presentation disposition; explicitly excluded nested/proof/test/runtime values never become catalog entries.
- [ ] The initial stable set contains exactly Switch exact-action and Nexus caller-closure entries in canonical order.
- [ ] Every stable entry has exactly one validated projection binding with a safe fixture path and unique projection key, and binding identity changes do not alter catalog semantic identity.
- [ ] List/explain results are byte-stable under registry ordering changes.
- [ ] Semantic list/explain/check consume the fn-15 generic core and remain a distinct adapter/executable from API/config input catalogs.
- [ ] List/explain/check have canonical stdout/stderr/exit behavior; internal, unknown, ambiguous, and deprecated selectors return exact structured results without semantic redirection.
- [ ] The existing inspector scenarios and canonical outputs remain unchanged.
- [ ] No Temporal identity or import enters `model/Umpire`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
