---
satisfies: [R2, R3, R6]
---

# fn-13-deterministic-go-regression-projections.2 Add the fixture-backed Go projection verifier
## Description
Add the isolated Go test helper consumed by generated wrappers for R2, R3, and R6. The helper validates every displayed projection field against a checked-in canonical fixture; it deliberately has no Temporal runtime, Lean invocation, or semantic procedure. This task is disjoint from rendering so it can be built in the first wave against the parent API contract.

**Size:** S
**Files:** `tools/umpire/regression/projection.go`, `tools/umpire/regression/projection_test.go`
**Touches:** [tools/umpire/regression/projection.go, tools/umpire/regression/projection_test.go]

## Approach
- Expose a narrow `Reference` plus `RequireProjection(testing.TB, Reference)` seam carrying only format, stable identity, repo-relative fixture, canonical model-root-relative sources, property identities, observation-requirement identities, and expected semantic fingerprint.
- Resolve the repository root independently of caller working directory by adapting the source-anchor pattern already used by repository tests; validate fixture containment under the repository root and source containment under its model root before reading.
- Decode only the fixture metadata needed for comparison, recompute SHA-256 over the decoded semantic identity, and compare the complete reference as whole values with `require` rather than per-field assertion noise.
- Keep the helper useful as an ordinary Go test contract while explicitly refusing execution/evidence semantics.

## Investigation targets
**Required** (read before coding):
- `tests/testutils/source_root.go:9-25` — source-anchored repository root pattern
- `model/Temporal/Feature/Nexus/testdata/nexus-caller-closure-experiment-spec.json` — fixture shape verified by the helper
- `model/Umpire/Artifact.lean:71-80` — portable envelope fields and no-runtime contract

**Optional** (reference as needed):
- `model/README.md:79-105` — current artifact and inspector truthfulness language

## Key context
Artifact provenance is model-root-relative, while the fixture is repository-relative. Preserve canonical source values for comparison and validate their real files under `<repo>/model`; repository-facing comments may add `model/`, but `Reference` comparisons do not. The helper validates a checked-in model projection, and its success must never be described as a Temporal execution or conformance result.

## Acceptance
- [ ] `RequireProjection` validates supported format, exact query identity, complete canonical Lean source provenance, property identities, observation-requirement identities, and the specified `sha256:` fingerprint from the fixture's decoded semantic identity.
- [ ] Repository-root discovery is independent of process working directory; fixture paths are non-empty repository-relative contained regular files, and provenance paths are non-empty model-root-relative contained Lean files.
- [ ] Missing/malformed/unsupported fixtures and identity, source, property, observation-requirement, or fingerprint mismatches fail the test with concise context and never skip.
- [ ] Tests mutate each displayed fixture field independently—including properties and observation requirements without updating semantic identity—to prove every mismatch is detected.
- [ ] Tests use `require` and cover working-directory independence plus absolute, traversal, symlink-escape, wrong-kind, and nonexistent path cases.
- [ ] The package imports no Umpire3, execution, evidence, or conformance implementation and adds no third-party dependency.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/regression` passes.

## Done summary
Added an isolated fixture-backed Go verifier that checks the complete generated projection reference, recomputes the semantic fingerprint, and enforces repository/model-root path containment without runtime, evidence, conformance, generator-internal, or Umpire3 dependencies. Added table-driven coverage for every projected field, strict fixture failures, working-directory independence, and unsafe fixture/source paths.

baseline: green (focused Go gate reused receipt 1b77b3c9; make umpire-check-regression passed pre-edit)
GATE_SKIPPED:unittest:green-receipt 1b77b3c9 - baseline reused from prior post-gate pass
stage: impl-review - ran [2026-08-26T05:02:19Z..2026-08-26T05:05:28Z] (SHIP)
## Evidence
- Commits: 959674eaf7eeef3c6243c250693e9ff22f51e017
- Tests: GATE_SKIPPED:unittest:green-receipt 1b77b3c9 - baseline reused from prior post-gate pass, go test -count=1 -tags test_dep ./tools/umpire/regression, go test -race -count=1 -tags test_dep ./tools/umpire/regression, go test -count=1 -tags test_dep ./tools/umpire/..., make umpire-check-regression
- PRs: