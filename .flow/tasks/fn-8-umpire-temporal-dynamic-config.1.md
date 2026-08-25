---
satisfies: [R1, R3]
---
# fn-8-umpire-temporal-dynamic-config.1 Expose immutable dynamic-config registry metadata

## Description
Add the registration-time metadata and snapshot boundary required by R1 and the Go-owned portion of R3. Keep the registry as the authority and preserve all existing setting behavior and comments.

**Size:** M
**Files:** `common/dynamicconfig/metadata.go`, `common/dynamicconfig/metadata_test.go`, `common/dynamicconfig/registry.go`, `common/dynamicconfig/setting.go`, `common/dynamicconfig/setting_gen.go`, `cmd/tools/gendynamicconfig/main.go`, `cmd/tools/gendynamicconfig/dynamic_config.tmpl`
**Touches:** [common/dynamicconfig/metadata.go, common/dynamicconfig/metadata_test.go, common/dynamicconfig/registry.go, common/dynamicconfig/setting.go, common/dynamicconfig/setting_gen.go, cmd/tools/gendynamicconfig/main.go, cmd/tools/gendynamicconfig/dynamic_config.tmpl]

### Approach
- Define one narrow metadata vocabulary for canonical key, description, precedence, result-type identity, codec class, and concrete/constrained/opaque default inputs; do not expose converter functions or mutable registry internals.
- Capture metadata through the existing constructor families so runtime registration and validation remain single-sourced. Update both the `gendynamicconfig` driver and its authoritative constructor template before regenerating `setting_gen.go`; preserve all template/generated comments and ordering.
- Return deeply copied snapshots after static initialization and retain the existing post-query registration invariant. Normalize and validate keys at the boundary.
- Cover scalar aliases, structural typed settings, custom converters, constrained defaults, all constraint dimensions, and all eight precedence policies with table-driven `require` assertions; prefer aggregate equality over field-by-field checks.
- Prove callers cannot mutate registry maps, slices, defaults, constraints, or descriptions through a returned snapshot.

### Investigation targets
**Required** (read before coding):
- `common/dynamicconfig/registry.go:19-37` — registration/query invariant and authoritative map
- `common/dynamicconfig/setting.go:16-40` — generic setting boundary
- `common/dynamicconfig/setting_gen.go:24-84` — generated constructor families
- `cmd/tools/gendynamicconfig/main.go:31-118` — source driver for types and precedence orders
- `cmd/tools/gendynamicconfig/dynamic_config.tmpl` — authoritative generated constructor bodies

**Optional** (reference as needed):
- `common/dynamicconfig/client.go:47-96` — constraint vocabulary and exact-shape comments
- `common/dynamicconfig/key.go:9-20` — lowercase key normalization

### Key context
`setting_gen.go` is generated output: metadata wiring belongs in `dynamic_config.tmpl`, with the driver updated only for the data the template requires. Regeneration must prove the emitted constructors contain the wiring and preserve existing comments.

### Quick commands
```bash
go test -count=1 -tags test_dep ./common/dynamicconfig
make fmt-imports
```
## Acceptance
- [ ] Every existing constructor family records complete metadata without changing setting lookup, validation, registration timing, or comments.
- [ ] The authoritative `dynamic_config.tmpl` and its driver contain the metadata generation logic, and regenerated `setting_gen.go` contains the expected wiring without manual-only edits.
- [ ] Snapshots are deeply immutable and deterministic, with canonical normalized keys and exact constraint/default shapes.
- [ ] Tests cover all eight policies plus scalar, structural, custom-converter, and constrained-default metadata using `require`.
- [ ] Missing metadata, empty registry, post-query registration, and normalized-key collisions fail without returning partial state.
- [ ] Generated constructor output is regenerated from the authoritative template and focused tests pass.
## Done summary
Added constructor-time setting metadata and deterministic, deeply copied registry snapshots with normalized keys, codec/type identity, and concrete, constrained, or opaque defaults. Updated the authoritative generator driver/template and regenerated all constructor families while preserving existing behavior and comments.

Baseline: green (`go test -count=1 -tags test_dep ./common/dynamicconfig`; `make fmt-imports`). R1/R3 error coverage: `TestRegisteredSettingMetadataRejectsIncompleteRegistry`, `TestRegisteredSettingMetadataFreezesRegistry`, and `TestRegisteredSettingMetadataRejectsNormalizedKeyCollision`; immutability and constructor-family coverage live in the remaining focused metadata tests.

stage: impl-review - ran [2026-08-25T14:49:47Z..2026-08-25T14:56:02Z]
## Evidence
- Commits: 8b5b98a9a8dafcbb9b135a54bbf4ce0a5a39681c
- Tests: go test -count=1 -tags test_dep ./common/dynamicconfig, make fmt-imports
- PRs: