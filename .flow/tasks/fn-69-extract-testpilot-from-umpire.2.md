---
satisfies: [R1, R6]
---
# fn-69-extract-testpilot-from-umpire.2 Establish the Testpilot protobuf contract

## Description
Add the Testpilot-owned proto source and generated Go package beside the old package, proving structural compatibility before runtime extraction (R1, R6). The old package remains temporarily for buildable migration and has no translation path.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/*.proto`, `api/testpilot/v1/*.pb.go`, `api/testpilot/v1/*.go-helpers.pb.go`, `proto/image.bin`, `common/testing/testpilot/protocol_compatibility_test.go`, migration ledger
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**, proto/image.bin, common/testing/testpilot/protocol_compatibility_test.go, .flow/artifacts/fn-69-extract-testpilot-from-umpire/migration-ledger.md]

### Approach
- Mirror the five established messages under `temporal.server.api.testpilot.v1` and Go package `api/testpilot/v1`, preserving comments, fields, enums, numbers, defaults, and cardinalities.
- Regenerate through `make proto`; never hand-edit generated Go or `proto/image.bin`.
- Add a persistent descriptor-structure test over the Testpilot package using the bounded reconstruction pattern already used by the Case schema tests.
- Compare the old and new descriptors while ignoring only explicitly enumerated file/package/full-name changes; record generated diff allowlists and hashes.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/umpire/v1/*.proto` — exact source contract to mirror
- `Makefile:508-529` — canonical proto generation
- `tools/umpire/cmd/umpire-gen-lean-api/case_schema_test.go` — structural descriptor comparison pattern
- `.flow/memory/bug/integration/validate-protobuf-descriptor-structure-2026-09-05.md` — bounded descriptor parity lesson

### Acceptance

## Acceptance
- [ ] Five Testpilot proto sources and ten generated Go/helper files use the approved source path, protobuf namespace, and Go package; field/enum shapes and numbers equal the frozen Umpire baseline.
- [ ] `make proto`, focused descriptor tests, `make lint-protos`, and `make lint-api` pass or match verified inherited failures, with every generated change allowlisted.
- [ ] Structural tests reject changed numbers, kinds, cardinalities, defaults, or missing descriptors and accept only the explicit namespace/file identity substitution.
- [ ] Old and new packages coexist without aliases, converters, registries, fallback, or active consumer migration; the ledger marks the old package temporary and task 8-owned.

## Done summary
Added the five Testpilot protobuf sources and canonical generated Go package alongside Umpire, with a full descriptor-structure compatibility test and migration-ledger hashes/allowlist. The old package remains temporary through consumer reconciliation, with no aliases, converters, registries, fallbacks, or consumer migration.

baseline: red (`go test -count=1 -tags test_dep ./common/testing/testpilot/...` failed before implementation because the task-2 destination did not exist); `make proto` passed before implementation and generated no drift.

verification: `make proto`, `go test -count=1 -tags test_dep ./common/testing/testpilot/...`, `make lint-protos`, and `make lint-api` passed. Final gate classification was forced full by the user's pre-existing `.plans/UMPIRE4_ORDER.md` change; task-scoped focused gates remained green and no unrelated broad suite was rerun.

stage: impl-review - ran [2026-09-06T17:50Z..2026-09-06T17:56:28Z]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: make proto, go test -count=1 -tags test_dep ./common/testing/testpilot/..., make lint-protos, make lint-api
- PRs:
