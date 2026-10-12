---
satisfies: [R3]
---
# fn-169-a-temporal-free-framework-the-lifter.2 Plug-in seam: layout conventions, realization-kit lifting and API descriptors supplied by temporal/export (after fn-141.8)

## Description
Runs after fn-141.8, which removes most realization-kit coupling from the lifter (owner, 2026-10-12).

1. **Plug-in seam.** The generic `framework/export` receives Temporal specifics only from `temporal/export`: layout conventions (where features, foundations, actors and capabilities live) as configuration, realization-kit lifting as a registered kit, and the target API's descriptors.
2. **Generic proto support (owner, 2026-10-12).** A system registers its API proto roots and descriptor sets once. The framework's proto-name lint (today `ProtoLiterals`, hard-coded to `temporal.api.` and `/temporal.api.`), the gate's proto stage (today `Gate`'s Temporal proto list) and realization lifting all read that registration. Temporal registers `temporal.api.*` and its descriptor sets from `temporal/export` and `temporal/check`.
3. Prove it: a fixture system with its own small proto package and no Temporal API builds, lints and exports through the framework alone; the mini lift is byte-identical; mini equivalence holds.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
