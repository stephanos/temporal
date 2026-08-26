---
satisfies: [R2, R6]
---
# fn-13-deterministic-go-regression-projections.2 Add the fixture-backed Go projection verifier

## Description
Add the isolated Go test helper consumed by generated wrappers for R2 and R6. The helper validates projection identity and provenance against a checked-in canonical fixture; it deliberately has no Temporal runtime, Lean invocation, or semantic procedure. This task is disjoint from rendering so it can be built in the first wave against the parent API contract.

**Size:** S
**Files:** `tools/umpire/regression/projection.go`, `tools/umpire/regression/projection_test.go`
**Touches:** [tools/umpire/regression/projection.go, tools/umpire/regression/projection_test.go]

### Approach
- Expose a narrow `Reference` plus `RequireProjection(testing.TB, Reference)` seam carrying only format, stable identity, repo-relative fixture, Lean sources, and expected semantic fingerprint.
- Resolve the repository root independently of caller working directory by adapting the source-anchor pattern already used by repository tests; validate containment before reading the fixture.
- Decode only the fixture metadata needed for comparison, recompute SHA-256 over the decoded semantic identity, and report failures with `require`-style whole-value comparisons rather than per-field assertion noise.
- Keep the helper useful as an ordinary Go test contract while explicitly refusing execution/evidence semantics.

### Investigation targets
**Required** (read before coding):
- `tests/testutils/source_root.go:9-25` — source-anchored repository root pattern
- `model/Temporal/Feature/Nexus/testdata/nexus-caller-closure-experiment-spec.json` — fixture shape verified by the helper
- `model/Umpire/Artifact.lean:71-80` — portable envelope fields and no-runtime contract

**Optional** (reference as needed):
- `model/README.md:79-105` — current artifact and inspector truthfulness language

### Key context
The helper validates a checked-in model projection; its success must never be described as a Temporal execution or conformance result. Preserve existing comments in any touched source.

### Acceptance

## Acceptance
- [ ] `RequireProjection` validates supported format, exact query identity, complete Lean source provenance, and the specified `sha256:` fingerprint from the fixture's decoded semantic identity.
- [ ] Repository-root discovery is independent of process working directory, and fixture paths must be non-empty, repository-relative, contained, non-symlink escapes, and regular files.
- [ ] Missing/malformed/unsupported fixtures and identity, source, or fingerprint mismatches fail the test with concise context and never skip.
- [ ] Tests use `require` and cover working-directory independence plus all invalid path and fixture cases.
- [ ] The package imports no Umpire3, execution, evidence, or conformance implementation and adds no third-party dependency.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/regression` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
