---
satisfies: [R2, R3, R7]
---
# fn-8-umpire-temporal-dynamic-config.2 Project the initialized production registry

## Description
Build the discovery and canonical projection core for R2, R3, and the Go-produced half of R7. This task stops at a validated in-memory catalog and fixtures; rendering/publication belongs to the next task.

**Size:** M
**Files:** `cmd/tools/genleandynamicconfig/main.go`, `cmd/tools/genleandynamicconfig/run.go`, `cmd/tools/genleandynamicconfig/discover.go`, `cmd/tools/genleandynamicconfig/project.go`, `cmd/tools/genleandynamicconfig/project_test.go`, `cmd/tools/genleandynamicconfig/testdata/**`
**Touches:** [cmd/tools/genleandynamicconfig/**]

### Approach
- Use the already-established `golang.org/x/tools/go/packages` dependency with the minimum typed/compiled-file load mode needed to locate production registration sites; exclude test files and fail package errors or ill-typed discoveries.
- Create and clean a temporary helper inside the module so Go `internal` imports remain valid. Blank-import every discovered production package, then read the frozen runtime snapshot once; package analysis discovers initializers but never reconstructs defaults.
- Project metadata into a closed canonical Go model for normalized keys, schemas, codec classes, exact constraints, all eight precedence policies, defaults, descriptions, provenance, and mechanical identity. Sort all maps/sets explicitly and reject non-canonical encodings.
- Compute bounded expected resolution fixtures through the real Go collection/property functions using real settings for every precedence policy. Record canonical inputs, selected source/constraint, and result; do not claim converter/YAML parity.
- Make stage/package/key-specific diagnostics deterministic and guarantee discovery/helper/projection errors produce no renderable partial result.

### Investigation targets
**Required** (read before coding):
- `cmd/tools/genleanmodeldescriptors/main.go:105-190` — package inventory, helper, and runtime registry pattern
- `tools/gomad2/internal/translate/main.go:71-90` — existing `go/packages` loading conventions
- `common/dynamicconfig/collection.go:314-377` — real precedence/default interleaving
- `cmd/tools/gendynamicconfig/main.go:60-118` — current policy order authority
- `chasm/lib/callback/config.go:17-83` — representative namespace, destination, and custom settings

**Optional** (reference as needed):
- `common/dynamicconfig/constants.go:1292-1308` — real task-queue constrained defaults
- `common/dynamicconfig/constants.go:3216-3221` — namespace callback feature setting

### Key context
Go initializes blank imports before the helper and exactly once. Keep the helper module-local, request compiled production files, and treat any load/initialization error as catalog incompleteness rather than silently skipping a package.

### Quick commands
```bash
go test -count=1 -tags test_dep ./cmd/tools/genleandynamicconfig
go test -count=1 -tags test_dep ./common/dynamicconfig
```

## Acceptance
- [ ] Typed discovery finds every production registration package, excludes tests, and fails unloadable, ill-typed, discovered-but-unregistered, or zero-catalog cases.
- [ ] The module-local helper initializes the authoritative registry and is cleaned on success and every failure path.
- [ ] Projection is complete, normalized, canonical, order-independent, and rejects duplicate keys/constraints, unknown policy, incoherent defaults, and nondeterministic structured values.
- [ ] Unprojectable faithful defaults remain present as stable opaque metadata rather than causing omission.
- [ ] Go-computed fixtures cover all eight policies, exact unset dimensions, fallback/specific matching, and constrained-default interleaving.

## Done summary
Implemented typed production registration discovery, module-local runtime registry initialization, closed canonical catalog projection, and real Go resolution fixtures for all eight precedence policies. The review fix makes fixture results identify and validate the selected source and exact constraint, including negative coverage for incorrect and ambiguous selections.

Focused generator/common tests, the combined Go gate, model build, and task-scoped lint pass. The parent integration target remains absent as it was at baseline and belongs to fn-8.4; canonical `make lint-code` remains inherited-red with 1848 branch-wide findings while the changed package reports zero issues.

stage: impl-review - ran [2026-08-25T15:32:54Z..2026-08-25T15:45:11Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f27dc41ae3a0438aca150c3b5836e5f704cf0982, 765d139f80ddc19a484e2465e10529bddb20115f
- Tests: go test -count=1 -tags test_dep ./cmd/tools/genleandynamicconfig, go test -count=1 -tags test_dep ./common/dynamicconfig, go test -count=1 -tags test_dep ./common/dynamicconfig ./cmd/tools/genleandynamicconfig, cd model && mise exec -- lake build, .bin/golangci-lint-v2.13.1 run --verbose --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --config=.github/.golangci.yml ./cmd/tools/genleandynamicconfig/..., baseline: red (go test -count=1 -tags test_dep ./common/dynamicconfig ./cmd/tools/genleandynamicconfig failed pre-edit because the generator package did not exist), baseline: red (make umpire-gen-dynamic-config failed pre-edit because the integration target belongs to fn-8.4 and does not exist yet), baseline: green (cd model && mise exec -- lake build), baseline: red (make lint-code failed pre-edit due exhausted temporary storage); inherited post-verify red (1848 branch-wide findings); task-scoped lint passed
- PRs:
