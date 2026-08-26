---
satisfies: [R2, R3, R7]
---
# fn-15-standalone-api-and-config-input-catalogs.3 Render API catalog facts inside the existing generated boundary

## Description
Render the complete checked API catalog into the existing generated API umbrella and pin publication/determinism behavior for R2/R3/R7.

**Size:** M
**Files:** `tools/umpire/internal/generate/api/render.go`, `tools/umpire/internal/generate/api/main_test.go`, `tools/umpire/internal/generate/api/fixture_test.go`, `tools/umpire/internal/generate/api/testdata/basic/expected/Fixture/API.lean`, `model/Temporal/API.lean`
**Touches:** [tools/umpire/internal/generate/api/render.go, tools/umpire/internal/generate/api/main_test.go, tools/umpire/internal/generate/api/fixture_test.go, tools/umpire/internal/generate/api/testdata/basic/expected/Fixture/API.lean, model/Temporal/API.lean]

### Approach
- Define generated catalog record/disposition structures and render the complete canonical fact list plus catalog identity into `API.lean`.
- Preserve all current typed method declarations, generated comments, child modules, and publication ordering.
- Extend the exact-tree fixture and integrated generation tests for byte stability and catalog validation-before-mutation.
- Keep catalog JSON outside the generator; only compiled Lean facts are generated.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/internal/generate/api/render.go:8-21,87-115` — three-artifact render boundary.
- `tools/umpire/internal/generate/api/main.go:22-109` — validate-before-publish orchestration.
- `tools/umpire/internal/generate/api/fixture_test.go:14-35` — exact generated tree fixture.
- `.flow/specs/fn-6-plan-simplify-lean-api-generator-output.md` — retained ownership and CLI contract.

### Quick command
`go test -count=1 -tags test_dep ./tools/umpire/internal/generate/api && make umpire-gen-api && cd model && lake build Temporal.API`

## Acceptance
- [ ] Generated output remains exactly `API.lean`, `API/Proto.lean`, and `API/Types.lean`.
- [ ] Fixture and production `API.lean` expose the complete canonical catalog while preserving existing RPC declarations and comments.
- [ ] Repeated and descriptor-reordered rendering is byte-identical.
- [ ] Catalog failure occurs before mutation and a rerun retains existing exclusive publication semantics.
- [ ] No fourth artifact, retained JSON, drift checker, check subcommand, or workflow is added.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
