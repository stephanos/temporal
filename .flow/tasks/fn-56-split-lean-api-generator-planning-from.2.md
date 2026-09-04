---
satisfies: [R2, R3, R4, R5]
---
# fn-56-split-lean-api-generator-planning-from.2 Make rendering a pure validated-plan consumer

## Description
Move generation coordination beside the command runner and make one renderer emit the exact artifact map from the completed plan only. Preserve direct builder emission, exact bytes, and validation-before-publication.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-gen-lean-api/generate.go`, `tools/umpire/cmd/umpire-gen-lean-api/render.go`, `tools/umpire/cmd/umpire-gen-lean-api/main_test.go`, `tools/umpire/cmd/umpire-gen-lean-api/fixture_test.go`
**Touches:** [tools/umpire/cmd/umpire-gen-lean-api/generate.go, tools/umpire/cmd/umpire-gen-lean-api/render.go, tools/umpire/cmd/umpire-gen-lean-api/main_test.go, tools/umpire/cmd/umpire-gen-lean-api/fixture_test.go]

### Approach
- Move `generateArtifacts` phase coordination beside `run`, explicitly sequencing plan construction, rendering, artifact-map validation, and existing publication.
- Add one private artifact-rendering entry point that accepts only `leanPlan`.
- Move recursive Lean type printing into `render.go`; consume planned paths, imports, names, and support namespace rather than projections or general configuration.
- Keep the current direct `strings.Builder` and `fmt.Fprintf` style, generated headers and docs, artifact-map validation, and publication owner.
- Extend integrated tests to pin plan-to-render sequencing, the exact three-artifact tree, the normalized empty-plan output, reversed-input determinism, and no artifact or publication on planner failure.
- Treat checked-in expected files as read-only byte oracles; do not run a fixture rewrite.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/cmd/umpire-gen-lean-api/render.go:26-140` — mixed coordination and source emission
- `tools/umpire/cmd/umpire-gen-lean-api/generate.go:14-48` — command pipeline and pre-publication ordering
- `tools/umpire/cmd/umpire-gen-lean-api/main_test.go:16-98` — structural output and determinism coverage
- `tools/umpire/cmd/umpire-gen-lean-api/main_test.go:220-233` — failure-before-output coverage
- `tools/umpire/cmd/umpire-gen-lean-api/fixture_test.go:14-38` — exact three-file golden contract
- `tools/umpire/cmd/umpire-gen-lean-api/testdata/basic/expected/Fixture/API/Types.lean` — exact type grammar output

**Optional** (reference as needed):
- `docs/superpowers/specs/2026-08-24-descriptor-driven-lean-generator-design.md:41-55` — documented stage separation and error behavior

### Key context
- Rendering is mechanical implementation, not a second semantic authority.
- Preserve the exact output bytes and current filesystem/publication error surface. Broad drift checks and CI workflow work remain out of scope.

## Acceptance
- [ ] R2-R3 are satisfied by a renderer that accepts only `leanPlan` and contains no projection traversal, name allocation, reference resolution, type reconstruction, or general configuration interpretation.
- [ ] The CLI flags, three-artifact membership, headers, docs, imports, declaration ordering, qualification, formatting, validation order, and publication behavior remain exact.
- [ ] Planning failure yields no artifact map and cannot reach publication; malformed inputs and unsafe/removal/publication failures retain current diagnostics.
- [ ] Exact-tree fixtures pass without rewrite, and reversed descriptor inputs remain byte-identical.
- [ ] Rendering a valid normalized empty plan produces exactly the current three artifacts with their current headers, module docs, imports, newlines, and bytes.
- [ ] No generated file, public package, exported internal, template engine, dependency, drift target, regeneration comparison, or CI workflow is added.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api` passes.
- [ ] `cd model && mise exec -- lake build Temporal.API` passes.
- [ ] `make lint-code` passes.

## Done summary
Moved generation coordination beside the command runner and made a private renderer consume only the validated Lean plan, preserving exact artifacts, byte output, validation order, diagnostics, and publication behavior.
## Evidence
- Commits: 949447ebe
- Tests: go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api, cd model && mise exec -- lake build Temporal.API, scoped golangci-lint against be2e670c4 (0 issues)
- PRs: