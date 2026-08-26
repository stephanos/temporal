---
satisfies: [R2, R4, R7]
---
# fn-15-standalone-api-and-config-input-catalogs.4 Add the bounded current-model API selection policy

## Description
Define and validate the explicit depth-four current-model selection policy over the complete generated catalog for R2/R4/R7.

**Size:** M
**Files:** `model/Temporal/System/API.lean`, `model/Temporal/System/API/Catalog.lean`, `model/Temporal/System/API/CatalogTests.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/System/API.lean, model/Temporal/System/API/Catalog.lean, model/Temporal/System/API/CatalogTests.lean, model/TemporalModelTests.lean]

### Approach
- Freeze `temporal-current-model/v1` to the exact nine canonical `temporal.api.history.v1.NexusOperation*EventAttributes` message roots named in the parent spec, the one-to-32 schema bound, and depth four.
- Validate root uniqueness, kind, and existence against generated facts.
- Apply the parent spec's exact propagation rules for method/service/request/response, message fields, real oneofs, enums/values, message-valued map values, and non-root methods.
- Traverse message edges breadth-first by full `(depth, root, edge path, message)` keys and retain the winning complete path provenance independent of descriptor order.
- Emit stopped-edge metadata for depth/cycle boundaries; mark boundary-only target messages and resolve multi-root/multi-path item conflicts using the specified root/dependency/boundary/outside precedence.
- Pin complete current-model identities/provenance and policy digest in focused tests.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/API/Proto.lean:6-21` — generated catalog/method support boundary after Task 3.
- `tools/umpire/internal/generate/api/message_graph.go:37-61` — analogous deterministic graph traversal.
- `model/Temporal/System/Configuration/Core.lean:86-101` — authored policy/check pattern.
- `.plans/UMPIRE4_COMPONENTS.md:141-171` — C1 current-model selection responsibility.
- `model/TemporalModelTests.lean:1-10` — aggregate test registration.

### Quick command
`cd model && lake env lean Temporal/System/API/CatalogTests.lean`

## Acceptance
- [ ] Tests pin the parent spec's exact nine fully qualified root identities, policy v1, the one-to-32 schema constraint, and depth four; adding, removing, renaming, or reordering roots cannot silently redefine the policy.
- [ ] Method roots select only their service/request/response closure; messages select all logical fields/real oneofs plus terminal referenced enums/values; message-valued fields/map values alone consume depth; unrelated methods remain outside.
- [ ] Root/fact ordering and equal-depth multi-root paths cannot change selected identities, full minimal provenance, dispositions, stopped-edge records, or policy identity.
- [ ] Cycle and depth edges stop through explicit owner-field/target/root/path/reason metadata; a boundary target reached admissibly elsewhere resolves to root/dependency while the stopped edge remains visible.
- [ ] Empty/duplicate/wrong-kind/unknown/overflow roots, wrong depth, missing provenance, and conflicting dispositions fail.
- [ ] The policy annotates the complete catalog without modifying or pruning generated declarations.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
