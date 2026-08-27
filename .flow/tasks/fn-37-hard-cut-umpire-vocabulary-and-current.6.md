---
satisfies: [R2, R5, R6]
---
# fn-37-hard-cut-umpire-vocabulary-and-current.6 Cut Go consumers and regression generation to v2 Generated Views

## Description
Implement the Go half of R5 and R6. Rename the regression Projection surface to Generated View, admit only exact canonical v2 bytes, independently verify Artifact Checksums, and generate a complete Switch-and-Nexus Go/Markdown output set atomically.

**Size:** L
**Files:** `tools/umpire/regression/**`, renamed Generated View command, four generated outputs, `Makefile`
**Touches:** [tools/umpire/regression/**, tools/umpire/cmd/umpire-gen-regression-projections/**, tools/umpire/cmd/umpire-gen-regression-views/**, tools/umpire/regression/catalog_generated_test.go, tools/umpire/regression/switch_generated_view_test.go, model/Temporal/Tool/Generated/Regressions.md, model/Umpire/Examples/Generated/Switch.md, Makefile]

### Approach
- Rename the command directory, internal record/functions, exported verifier, diagnostics, and Make targets from Projection to Generated View; remove the old command path and target aliases.
- Replace Go wire structs and strict-key tables with the exact v2 field names, Known Gap record, and format constants.
- Preserve a duplicate-aware top-level preflight that classifies `umpire-experiment/v1` as unsupported before any v2 field validation. It never translates or falls back to v1 structs.
- For v2, strictly decode, validate, recompute nested and outer Artifact Checksums, re-encode through one canonical v2 encoder, append exactly one LF, and require byte-for-byte equality with the input fixture.
- Add rejection tests for reordered object fields, leading/trailing/pretty whitespace, missing or extra LF, alternate valid JSON string escaping, alternate numeric representations such as exponent notation, legacy/unknown/case-colliding/duplicate keys, trailing data, malformed values, and checksum mismatches.
- Independently recompute the Task `.1` Behavior Fingerprint/Artifact Checksum golden values using Go SHA-256.
- Expand the closed production manifest to exactly two entries: existing Nexus caller closure and Switch `model/Umpire/Examples/testdata/switch-experiment-spec.json`. Bind each record and generated Go/Markdown view to the fixture's verified Artifact Checksum.
- Keep Nexus outputs at `tools/umpire/regression/catalog_generated_test.go` and `model/Temporal/Tool/Generated/Regressions.md`; add Switch outputs at `tools/umpire/regression/switch_generated_view_test.go` and `model/Umpire/Examples/Generated/Switch.md`.
- Preserve path-containment, provenance, deterministic ordering, and atomic publication. Test that the published map is exactly the four manifest-owned outputs and rejects missing, extra, stale, or partially written output.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/regression/projection.go` — exported verification API and strict fixture parser.
- `tools/umpire/cmd/umpire-gen-regression-projections/catalog.go` — Nexus-only production manifest to close over Switch too.
- `tools/umpire/cmd/umpire-gen-regression-projections/projection.go` — generator-side schema decoding.
- `tools/umpire/cmd/umpire-gen-regression-projections/generate.go` — validation and publication flow.
- `tools/umpire/cmd/umpire-gen-regression-projections/render.go` and `render_test.go` — checked-in view rendering and completeness tests.
- `tools/umpire/regression/catalog_generated_test.go` and `model/Temporal/Tool/Generated/Regressions.md` — existing Nexus outputs.
- `Makefile:1010-1040` — generation/check target wiring.

### Key context
Canonical validity is a byte contract, not merely “decodes to the same Go value.” Updating the existing Generated Views and adding the Switch entry are required by the hard schema cut. Do not add a generic generated-API drift verifier or a new CI workflow.
## Acceptance
- [ ] The renamed Generated View command and verifier accept only byte-canonical v2 Artifacts and reproduce the checked-in output.
- [ ] The top-level preflight classifies v1 as unsupported before field-level validation and no v1 reader, translation, or fallback struct exists.
- [ ] Reordered fields, whitespace/LF variations, alternate string escaping/numeric encodings, legacy/unknown/case-colliding/duplicate keys, trailing data, malformed values, and checksum mismatches reject deterministically.
- [ ] Go recomputation matches Lean Behavior Fingerprint and Artifact Checksum golden values.
- [ ] The closed production manifest contains exactly Nexus and Switch; every entry binds the verified Artifact Checksum into its Go and Markdown Generated Views.
- [ ] Generation deterministically and atomically publishes exactly four outputs—the Nexus Go/Markdown pair and Switch Go/Markdown pair—and completeness tests reject missing, extra, stale, or partial output.
- [ ] The old Projection command path, exported API, diagnostics, and Make target aliases are absent.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
