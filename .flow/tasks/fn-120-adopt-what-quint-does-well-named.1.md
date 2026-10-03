---
satisfies: [R1, R3, R4]
---
# fn-120-adopt-what-quint-does-well-named.1 Add named-choice declarations, inert IR names and Quint export

Touches: [model/umpire/**, model/lifter/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, model/gate/**, tools/umpire/model/**, tools/umpire/export/**, model/ir/**]

## Description
Start only after fn-112.11 completes Query.total schema, binding and linked API jar regeneration. Its same-spec prerequisites are fn-112.3, .4 and .5; the conductor checks fn-112.11 completion as the cross-spec entry gate. Finish before fn-112.6 rewrites branches. Settle Part A syntax and semantics against accept/stay and captured val names. The construct labels existing alternatives without adding or reordering results. Keep unnamed multi-result lists readable during migration.

**Size:** M
**Files:** model/umpire step declarations, model/lifter step lifting, IR schema and generation, Go reader/goldens, Quint export.

### Approach
- Use the fn-112.1 original-baseline harness: permit only inert choice names on existing result alternatives; prove exact tables, Definition IDs, fingerprints, Query answers, exploration identities and Case bytes.
- Add choice metadata to the descriptor that already contains Query.total, then regenerate both IR bindings and the linked API jar. Extend historical descriptor/wire coverage in tools/umpire/model/schema_test.go for both current fields without replacing captured historical bytes.
- Specify that named alternatives do not multiply Query.total; test alongside fn-112.11's static formula.
- Export the same ordered result list to Quint with each alternative's inert name on its record. Keep nondeterministic result-index selection in the checker action, after the pure step function returns; preserve all alternatives, order and agreement rows.

### Investigation targets
- model/umpire step declarations, model/lifter/Declarations.scala, tools/umpire/model/schema_test.go, tools/umpire/export/quint.go, model/gate/Gate.scala.

## Acceptance
- [ ] R1 names existing alternatives once and refuses duplicate or singleton choices at the Scala line, while unnamed branching still lifts during rollout.
- [ ] fn-112.11 is complete before this task changes the schema; the regenerated descriptor and linked API jar contain both Query.total and choice-name metadata, with historical wire compatibility retained.
- [ ] R3 original-baseline comparisons permit only choice-name metadata; result count/order, behavior, IDs, tables, fingerprints, answers, exploration identity and Case bytes stay exact.
- [ ] R4 Quint's pure step function returns the full ordered named-result list; the checker action selects a result nondeterministically by index, and agreement rows stay exact. An unexpressible name is reported at its source.
- [ ] A choice fixture proves branch count is excluded from Query.total; both current descriptor fields have compatibility coverage and the linked API jar is regenerated.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
