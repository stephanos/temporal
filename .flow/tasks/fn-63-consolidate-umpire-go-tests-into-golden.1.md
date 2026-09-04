---
satisfies: [R1, R2, R4, R5]
---
# fn-63-consolidate-umpire-go-tests-into-golden.1 Prove the shared golden scenario harness

## Description
Establish the post-`fn-61` baseline and prove the shared test-only scenario seam for R1/R2/R4/R5. This owns the harness contract; later package migrations consume it without adding competing fixture conventions.

**Size:** M
**Files:** `tools/umpire/internal/testscenario/**` (new), `tools/umpire/portableevaluation/parity_test.go`, `tools/umpire/portableevaluation/testdata/**`, and the surviving root executor test established by `fn-61`
**Touches:** [tools/umpire/internal/testscenario/**, tools/umpire/portableevaluation/parity_test.go, tools/umpire/portableevaluation/testdata/**, tools/umpire/umpire_test.go]

### Approach
- Wait for `fn-61` to complete, re-anchor its final execution/test owners, then record reproducible counts for handwritten `_test.go` lines and top-level `Test`/`Fuzz` functions in the task completion evidence.
- Extract one internal test-only loader/runner around the existing normal and duplicate-delivery fixture families; drive the surviving resident executor facade and normal admission path instead of rebuilding internal adapters.
- Keep the format closed: scenario identity, input members, independently authoritative expected output, typed expected status/error, provenance, and fixed exact/stable comparison mode.
- Compare deterministic Artifacts byte-for-byte. For runtime-assigned values, name the stable semantic projection and validate dynamic fields structurally; do not implement an arbitrary ignore/normalization facility.
- Add focused loader tests for incomplete, malformed, unexpected, or independently invalid fixtures, immutable parallel consumption, and path-scoped diagnostics.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-61-simplify-the-umpire-go-execution-surface.md` — final facade, deletions, and sequencing contract
- `tools/umpire/portableevaluation/parity_test.go:35-130` — current Lean-owned fixture generator
- `tools/umpire/portableevaluation/parity_test.go:482-555` — current Go-versus-Lean oracle comparison
- `tools/umpire/artifact/golden_test.go:23-90` — exact canonical byte comparison pattern
- `Makefile:1071-1083` — temporary regeneration and complete-tree diff gate

**Optional** (reference as needed):
- `tools/umpire/portableevaluation/README.md:193-242` — stable/dynamic comparison and runtime constraints

## Acceptance
- [ ] The task completion evidence records the exact post-`fn-61` handwritten test-line and top-level test/fuzz baseline with repeatable commands.
- [ ] One internal test-only harness loads the existing normal and duplicate-delivery families and drives the supported facade/admission path without restoring a removed package or exporting a production helper.
- [ ] The two proof scenarios match independently generated expected outputs and preserve exact typed outcomes, Artifact identities, and diagnostic categories.
- [ ] Deterministic outputs are byte-exact; dynamic fields are structurally validated with an explicit stable projection and no generic ignore list.
- [ ] Ordinary focused tests pass with `go test -count=1 -tags test_dep` and invoke neither Lean nor a rewrite path.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
