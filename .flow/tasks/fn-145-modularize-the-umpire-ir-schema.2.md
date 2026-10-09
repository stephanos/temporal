---
satisfies: [R1, R2, R3]
---
# fn-145-modularize-the-umpire-ir-schema.2 Split ir.proto into nine responsibility files

## Description
Move the existing declarations into the nine-file graph from the parent spec, using Task 1's closure-aware generation and descriptor harness. Keep the generated message packages and all message-level contracts stable.

**Size:** M
**Files:** `proto/internal/temporal/server/api/umpire/v1/*.proto`, `api/umpire/v1/*.pb.go`, `model/check/Gate.scala`, `tools/umpire/ir/schema_test.go`
**Touches:** [proto/internal/temporal/server/api/umpire/v1/*.proto, api/umpire/v1/*.pb.go, model/check/Gate.scala, tools/umpire/ir/schema_test.go]

### Approach
- Extract the model chain and realization chain without changing declaration bodies.
- Keep `Model` in `ir.proto` and retain the existing protobuf, Go and JVM packages.
- Regenerate through the repository targets and classify descriptor file-name changes separately from message and wire equality.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:1-44` - root and package contract
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:45-320` - common values and expressions
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:321-711` - machines and claims
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:712-1326` - realization chain
- `tools/umpire/ir/schema_test.go:125-210` - relocation equivalence harness

### Key context
Messages with no current production instances remain supported because fixtures and admission tests use them. The Umpire IR must not import Testpilot schemas.


### Quick commands

```bash
make protoc
go test -tags test_dep ./tools/umpire/ir/...
```

## Acceptance
- [ ] R1's exact nine-file graph is generated and linted.
- [ ] R2's closure checks cover each imported file and generated output.
- [ ] R3's descriptor, JSON, Model, Query, Case and identity comparisons have no semantic delta.
- [ ] `make protoc` and focused Umpire schema tests pass.

## Done summary
Umpire's IR schema now compiles as nine acyclic responsibility files while preserving all 123 protobuf declarations and the regenerated Go API surface. The schema test compares the complete descriptor closure; Model IR and Case regeneration remain assigned to fn-145.4.

Tier: session (jev-unavailable(no_key))
stage: impl-review - ran [2026-10-09T04:17:26Z..2026-10-09T04:23:53Z] (codex, SHIP)

Baseline and verification:

- PASS `make protoc` with the repository's Mise-managed protoc 29.5 toolchain.
- PASS `go test -count=1 -tags test_dep ./tools/umpire/ir -run '^TestSchema'`.
- PASS `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check --test-only umpire.check.GateSuite`.
- PASS `go test -count=1 -tags test_dep ./cmd/tools/getproto -run '^TestLinkedModelDescriptors$'`.
- EXPECTED BATCH RED `go test -tags test_dep ./tools/umpire/ir/...`. The same 25 stale `umpire.Step` failures occurred before and after this task; fn-145.4 owns production Model IR and Case regeneration.
- INHERITED RED `make lint-api` and `make lint-protos`. Existing declaration-layout, comment, and zero-enum conventions conflict with the byte-preserving split and remain unchanged.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 86a256162e6297c01f100bb9998b9da28cbc2bb3
- Tests: make protoc, go test -count=1 -tags test_dep ./tools/umpire/ir -run '^TestSchema', mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check --test-only umpire.check.GateSuite, go test -count=1 -tags test_dep ./cmd/tools/getproto -run '^TestLinkedModelDescriptors$', EXPECTED BATCH RED (fn-145.4 regeneration deferred; identical 25 stale umpire.Step failures pre/post): go test -tags test_dep ./tools/umpire/ir/..., INHERITED RED (byte-preserving declaration conventions unchanged): make lint-api; make lint-protos
- PRs: