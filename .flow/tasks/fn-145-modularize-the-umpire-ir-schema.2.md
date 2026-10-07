---
satisfies: [R1, R2, R3]
---
# fn-145-modularize-the-umpire-ir-schema.2 Split ir.proto into nine responsibility files

## Description
Move the existing declarations into the nine-file graph from the parent spec, using Task 1's closure-aware generation and descriptor harness. Keep the generated message packages and all message-level contracts stable.

**Size:** M
**Files:** `proto/internal/temporal/server/api/umpire/v1/*.proto`, `api/umpire/v1/*.pb.go`, `model/check/Gate.scala`
**Touches:** [proto/internal/temporal/server/api/umpire/v1/*.proto, api/umpire/v1/*.pb.go, model/check/Gate.scala]

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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
