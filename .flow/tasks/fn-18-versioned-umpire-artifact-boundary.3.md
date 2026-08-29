---
satisfies: [R1, R2, R6, R8]
---
# fn-18-versioned-umpire-artifact-boundary.3 Prove canonical v2 admission and hard rejection

## Description
Make fn-37's v2 DrivePlan and ExperimentSpec the sole persisted baseline and prove Lean/Go agreement before adding later families.


**Size:** M
**Files:** `tools/umpire/artifact/experiment.go`, focused tests and testdata, `model/Umpire/Artifact/Tests/Codecs.lean`
**Touches:** [tools/umpire/artifact/experiment.go, tools/umpire/artifact/experiment_test.go, tools/umpire/artifact/testdata/**, model/Umpire/Artifact/Tests/Codecs.lean]

### Approach
- Strictly decode every exact v2 field, re-encode the deterministic two-space pretty bytes, and
  independently recompute nested and outer Artifact Checksums from the exact pretty preimages with
  one terminal LF.
- Validate Definition IDs, Behavior Fingerprints, Limits, Known Gaps, occurrences, checkpoints, Properties, requirements, and provenance.
- Reject earlier formats before field validation, then reject legacy keys, unknown keys, malformed
  values, checksum drift, compact JSON, alternate whitespace/indentation, and missing or extra LF.
- Keep migrations reserved for a future reviewed post-v2 successor; implement no current migration registry.

### Investigation targets
**Required:** fn-37 v2 codecs, fixtures, and the parent early proof point.

## Acceptance
- [ ] The canonical pretty caller-closure and Switch v2 fixtures round-trip byte-for-byte with
  independently verified pretty-preimage checksums; compact spellings reject.
- [ ] One-at-a-time mutations cover every required hard-rejection class.
- [ ] Any earlier current-Artifact format reports unsupported format before field-level validation.
- [ ] No migration, best-effort normalization, alias, or fallback is present.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestExperimentV2`

## Done summary
Added the canonical v2 ExperimentSpec admission boundary over the task-.2 kernel: Switch and Nexus fixtures round-trip byte-for-byte, exact pretty-preimage checksums agree, and every required mutation class rejects with stable precedence. The public encoder now rejects invalid or stale values, retained Definition IDs use bounded ASCII namespaced validation, and nested DrivePlan versions are classified before field errors.

Baseline: green (`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestExperimentV2`). Final focused/full Go, internal artifactv2, Lean codec, exact regression, vet, changed-lines lint, race, and fuzz gates passed. The unittest receipt was not writable because the protected inherited `config/development.yaml` false symlink stat keeps the worktree dirty; the task gate itself passed. Review-fix memory capture was attempted but repository memory is not initialized.

stage: impl-review - ran [2026-08-29T01:58:29Z..2026-08-29T02:18:01Z]
## Evidence
- Commits: 5bbd9ff1349c6cecb93ab0a3d911baaa0832c8c5, 15dba9d00f1b4f5d5341bec5aa54fde7ee976398
- Tests: baseline: green (mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestExperimentV2), mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestExperimentV2, mise exec -- go test -count=1 ./tools/umpire/artifact/..., mise exec -- go test -count=1 ./tools/umpire/internal/artifactv2/..., cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs, make umpire-check-regression, mise exec -- go vet ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/..., .bin/golangci-lint-v2.13.1 run --config .github/.golangci.yml --new-from-rev 43a272e411be671aac2dbb2518a8e7195198695f ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/..., mise exec -- go test -race -count=1 ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/..., mise exec -- go test -count=1 ./tools/umpire/artifact/... -run '^$' -fuzz '^FuzzStrictJSONNoPanicOrPermissiveSuccess$' -fuzztime=5s, GATE_RECEIPT_NOT_WRITTEN:unittest - protected inherited config/development.yaml false symlink stat kept worktree dirty
- PRs:

stage: plan-sync - skipped(config: planSync.enabled != true)
