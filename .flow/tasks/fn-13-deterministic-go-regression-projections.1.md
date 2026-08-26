---
satisfies: [R1, R2, R3, R4, R6]
---

# fn-13-deterministic-go-regression-projections.1 Build the strict projection model and deterministic renderers
## Description
Build the pure C5 projection core for R1, R2, R3, R4, and R6. This task owns the closed one-entry mechanical manifest, strict extraction from the existing canonical `ExperimentSpec`, semantic-identity fingerprinting, and in-memory Go/Markdown rendering. It stops before subprocess execution or repository publication so the fundamental boundary is proven against fixtures first.

**Size:** M
**Files:** `tools/umpire/internal/generate/regression/catalog.go`, `tools/umpire/internal/generate/regression/projection.go`, `tools/umpire/internal/generate/regression/render.go`, `tools/umpire/internal/generate/regression/render_test.go`, `tools/umpire/internal/generate/regression/testdata/**`
**Touches:** [tools/umpire/internal/generate/regression/**]

## Approach
- Define one private production manifest entry containing only the caller-closure inspector identity and repository-relative fixture/output paths; keep all semantic fields out of the manifest.
- Strictly extract the supported format, query identity, semantic identity, canonical model-root-relative provenance sources, property identities, and observation requirements from canonical JSON. Validate every provenance path against the real model root, preserve it for fixture comparison, and derive the repository-facing `model/` path only for navigation output.
- Derive the lowercase `sha256:` fingerprint from decoded semantic-identity bytes. Render a thin generated Go test that carries all extracted metadata to the task-2 verifier and a generated Markdown index from the same projection record.
- Normalize Go with `go/format`, sort every rendered collection, and use exact UTF-8/LF bytes with no environmental data. Use injected/synthetic inputs for malformed and collision cases plus the production fixture for the early proof; do not implement a general JSON canonicalizer or decode procedural semantics beyond displayed metadata.

## Investigation targets
**Required** (read before coding):
- `model/Temporal/Tool/Inspect.lean:46-77` — authoritative registry lookup and canonical JSON output contract
- `model/Umpire/Artifact.lean:228-241` — semantic identity and canonical `ExperimentSpec` representation
- `model/Temporal/Feature/Nexus/testdata/nexus-caller-closure-experiment-spec.json` — production pilot fixture
- `tools/umpire/internal/generate/api/render.go:84-106` — deterministic renderer and generated-header convention

**Optional** (reference as needed):
- `.plans/UMPIRE_COMPONENTS.md:215-236` — C5 projection contract
- `tools/umpire/internal/generate/api/fixture_test.go:12-35` — deliberate golden-update pattern

## Key context
`ExperimentSpec.semanticIdentity` is a large canonical value, not a short digest. The projection fingerprint is SHA-256 of that decoded string, never of raw JSON or rendered documentation. Artifact source paths omit the repository's `model/` prefix by contract; validate them below the model root and add the prefix only when rendering repository navigation. The exact Go generated marker must appear before the package clause.

## Acceptance
- [ ] The production manifest contains exactly the caller-closure ID plus fixture and two output paths, with no behavioral description or procedure.
- [ ] Supported production input yields one validated projection with exact ID, format, canonical sorted sources, properties, requirements, repository-facing source paths, and the specified `sha256:` fingerprint.
- [ ] Model-root-relative provenance resolves to real contained Lean files; a production-path test proves `Temporal/Feature/Nexus/CallerClosure.lean` renders as `model/Temporal/Feature/Nexus/CallerClosure.lean` without changing the canonical comparison value.
- [ ] Generated Go parses/formats, has the standard marker and one collision-checked `TestXxx`, carries properties and observation requirements to the shared verifier, and delegates only to it; Markdown carries matching metadata and a no-runtime notice.
- [ ] Repeated rendering and semantically equivalent JSON object ordering produce identical bytes without timestamps, absolute paths, or full semantic identity content.
- [ ] Focused tests reject malformed/empty JSON, unsupported version, identity mismatch, missing/duplicate/unsafe or nonexistent provenance, empty identity, unsafe manifest paths, invalid/colliding Go names, and Go/Markdown metadata divergence.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/internal/generate/regression` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
