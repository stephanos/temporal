---
satisfies: [R2, R5, R6]
---
# fn-37-hard-cut-umpire-vocabulary-and-current.6 Cut Go consumers and regression generation to v2 Generated Views

## Description
Implement the Go half of R5 and R6. Rename the regression Projection surface to Generated View, admit only exact v2 artifacts, independently verify Artifact Checksums, and regenerate the checked-in Go and Markdown views atomically.

**Size:** M
**Files:** `tools/umpire/regression/**`, renamed regression-view generator command, generated catalog test/Markdown, `Makefile`
**Touches:** [tools/umpire/regression/**, tools/umpire/cmd/umpire-gen-regression-projections/**, tools/umpire/cmd/umpire-gen-regression-views/**, model/Temporal/Tool/Generated/Regressions.md, Makefile]

### Approach
- Rename the command directory, internal record/functions, exported verifier, diagnostics, and Make targets from Projection to Generated View; remove the old command path and target aliases.
- Replace Go wire structs and strict-key tables with the exact v2 field names and format constants.
- Independently recompute Behavior Fingerprint and Artifact Checksum golden values using Go SHA-256 and reject mismatches before rendering.
- Return one stable unsupported-format classification for v1 before field-level validation; do not add translation or fallback structs.
- Preserve duplicate/case-collision/trailing-data, path-containment, provenance, deterministic ordering, and atomic publish guards.
- Regenerate the checked-in Go catalog and Markdown through the renamed authoritative command.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/regression/projection.go` — exported verification API and strict fixture parser.
- `tools/umpire/cmd/umpire-gen-regression-projections/projection.go` — generator-side schema projection.
- `tools/umpire/cmd/umpire-gen-regression-projections/generate.go` — validation and publication flow.
- `tools/umpire/cmd/umpire-gen-regression-projections/render.go` — checked-in view rendering.
- `tools/umpire/cmd/umpire-gen-regression-projections/render_test.go` — mutation and deterministic generation tests.
- `tools/umpire/regression/catalog_generated_test.go` — generated consumer shape.
- `Makefile:1010-1040` — generation/check target wiring.

### Key context
Updating the existing checked-in Generated Views is required by the hard schema cut. The previously declined work remains declined: do not add a generic generated-API drift verifier or a new CI workflow.

## Acceptance
- [ ] The renamed Generated View command and verifier accept exact v2 artifacts and reproduce the checked-in Go/Markdown output.
- [ ] V1, legacy/unknown/case-colliding/duplicate keys, trailing data, malformed values, and checksum mismatches reject deterministically.
- [ ] Go recomputation matches Lean fingerprint/checksum golden values.
- [ ] The old Projection command path, exported API, diagnostics, and Make target aliases are absent.
- [ ] Generation remains deterministic and atomically publishes its complete output set.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
