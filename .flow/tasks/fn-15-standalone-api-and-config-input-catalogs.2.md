---
satisfies: [R2, R3, R7]
---
# fn-15-standalone-api-and-config-input-catalogs.2 Derive the complete API catalog and prove field coverage

## Description
Derive complete API catalog facts from the existing neutral descriptor projection and completed Lean plan, and establish the independent completeness proof required by R2/R3/R7.

**Size:** M
**Files:** `tools/umpire/internal/generate/api/model.go`, `tools/umpire/internal/generate/api/lean_plan.go`, `tools/umpire/internal/generate/api/catalog.go`, `tools/umpire/internal/generate/api/catalog_test.go`
**Touches:** [tools/umpire/internal/generate/api/model.go, tools/umpire/internal/generate/api/lean_plan.go, tools/umpire/internal/generate/api/catalog.go, tools/umpire/internal/generate/api/catalog_test.go]

### Approach
- Enrich existing projection/plan records only with catalog completeness and source-provenance facts they already own.
- Derive canonical enum/value/message/oneof/logical-field/service/method facts after the Lean plan completes.
- Walk extensions explicitly and assign the closed unsupported disposition without fabricating a Lean mapping.
- Map every supported logical field to exactly one planned structure field or real-oneof constructor.
- Build the early-proof tests with an independent raw-descriptor logical-field walker, the integrated fixture, and an inline extension descriptor.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/internal/generate/api/model.go:30-100,162-203,251-280` — neutral projection facts.
- `tools/umpire/internal/generate/api/lean_plan.go:35-42,506-539,612-663,741-813` — completed names/types/oneofs.
- `tools/umpire/internal/generate/api/message_graph.go:17-61` — recursive dependency handling.
- `tools/umpire/internal/generate/api/descriptors.go:38-72` — merged descriptor ordering.
- `tools/umpire/internal/generate/api/main_test.go:16-98` — integrated fixture contract.

### Quick command
`go test -count=1 -tags test_dep ./tools/umpire/internal/generate/api -run 'TestCatalog|TestGenerateArtifacts'`

## Acceptance
- [ ] Independent raw logical-field count and identity set exactly equal derived catalog fields in normal and reversed descriptor order.
- [ ] Map internals and synthetic optional oneofs do not leak; real oneofs, recursion, nested/imported declarations, proto2 facts, and extensions retain their expected records.
- [ ] Every supported field maps to exactly one planned Lean identity/type and every extension is explicitly unsupported.
- [ ] Duplicate/unmatched/silently omitted fields and unsupported ordinary fields fail before rendering or publication.
- [ ] Existing generated comments, type planning, command arguments, and exactly-three-artifact ownership remain unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
