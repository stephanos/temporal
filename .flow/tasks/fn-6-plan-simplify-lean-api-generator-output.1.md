# fn-6-plan-simplify-lean-api-generator-output.1 Implement Plan: Simplify Lean API Generator Output

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
Simplified Lean API generation to a deterministic three-file API layout with path-only descriptor inputs, merged-descriptor diagnostics, safe replacement publishing, updated fixtures/model/docs, and removal of obsolete Generated/schema/manifest/check/inspect surfaces. Added focused coverage for the CLI contract, semantic merging, exact module output, replacement and preservation behavior, error wrapping, and determinism.

stage: impl-review - ran [2026-08-24T23:10:44Z..2026-08-24T23:16:05Z]
## Evidence
- Commits: b911ec334a7ca0841b953fa37f2ad1241dce2697
- Tests: gofmt -w tools/umpire/internal/generate/api/*.go tools/umpire/cmd/umpire-gen-api/*.go, go vet -tags test_dep ./tools/umpire/..., go test -count=1 -tags test_dep ./tools/umpire/internal/generate/api, make umpire-gen-api-fixture, go test -count=1 -tags test_dep ./tools/umpire/internal/generate/api, make umpire-gen-api, cd model && mise exec -- lake build, find model/Temporal -maxdepth 4 -type f | sort, rg 'Temporal\.Generated|Temporal\.Proto\.Core|Generated/Catalog|Generated/GRPC|Generated/(schema|manifest)\.json|--source|--default-source' model tools/umpire Makefile docs/superpowers (no matches), make umpire-check-regression
- PRs: