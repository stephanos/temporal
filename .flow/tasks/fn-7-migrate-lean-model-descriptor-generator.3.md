---
satisfies: [R1, R2, R4]
---
# fn-7-migrate-lean-model-descriptor-generator.3 Co-locate descriptor generation with genleanmodeldescriptors

## Description
Move, rather than copy, descriptor orchestration, tests, and fixtures from tools/common/godescriptors directly into cmd/tools/genleanmodeldescriptors as package main. Merge the implementation into main.go and its tests into main_test.go. Keep artifactio and protofile shared, preserve behavior and comments, and update all live documentation, Flow references, and testdata ignore rules. Touches: cmd/tools/genleanmodeldescriptors/**, tools/common/godescriptors/**, .gitignore, docs/superpowers/specs/2026-08-24-genleanmodeldescriptors-migration-design.md, .turbo/plans/descriptor-driven-lean-generator.md, .flow/specs/fn-7-migrate-lean-model-descriptor-generator.md.

## Acceptance
The command directory owns Run, its tests, and all fixtures; tools/common/godescriptors is absent; imports and fixture package paths use the command location; behavior, diagnostics, deterministic descriptors, and checked-in artifact bytes are unchanged; focused tests, build, lint, stale-reference scan, and artifact comparison pass.

## Done summary
Moved, rather than copied, descriptor orchestration, tests, and fixtures from `tools/common/godescriptors` directly into `cmd/tools/genleanmodeldescriptors` as `package main`. Merged the implementation into `main.go` and its tests into `main_test.go`; shared `artifactio` and `protofile` helpers remain under `tools/common`. Updated the testdata ignore exception, current migration design, implementation plan, and Flow spec to reflect command ownership.

TDD evidence: after moving the descriptor tests and fixtures first, the command package failed to build with undefined `Run`, `removeDescriptorHelper`, `exportDescriptors`, and `listDescriptorPackages`. Moving the implementation into `package main` made the same package test pass; the final implementation and test files were then consolidated into `main.go` and `main_test.go`.

No files were staged or committed. Unrelated active fn-3 changes were preserved.
## Evidence
- Commits:
- Tests: TDD RED: go test -count=1 -tags test_dep ./cmd/tools/genleanmodeldescriptors (undefined local Run and helpers before production move), go test -count=1 -tags test_dep ./tools/common/artifactio ./tools/common/protofile ./tools/umpire/internal/generate/api ./cmd/tools/genleanmodeldescriptors, go build -tags test_dep -o <temp>/genleanmodeldescriptors ./cmd/tools/genleanmodeldescriptors, go run -tags test_dep ./cmd/tools/genleanmodeldescriptors --package-pattern go.temporal.io/api/... --file-prefix temporal/api/ --output <temp>/umpire-public.binpb && cmp <temp>/umpire-public.binpb proto/umpire-public.binpb, .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --config=.github/.golangci.yml ./tools/common/artifactio ./tools/common/protofile ./cmd/tools/genleanmodeldescriptors, git diff --check, stale live reference and removed-directory scan
- PRs:
