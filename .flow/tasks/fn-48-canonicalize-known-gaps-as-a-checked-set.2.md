---
satisfies: [R2, R4]
---
# fn-48-canonicalize-known-gaps-as-a-checked-set.2 Admit checked Known Gaps at artifact boundaries

## Description
Move shared Lean Artifact types, encoders, and set checks onto the checked boundary while retaining raw negative cases at `KnownGapSet.checkCanonical` and the Go persisted decoder (R2, R4).

**Size:** M
**Files:** `model/Umpire/Artifact/Types.lean`, `model/Umpire/Artifact/Codecs.lean`, `model/Umpire/Artifact/Set.lean`, `model/Umpire/Planning/Tests/Artifacts.lean`, `tools/umpire/internal/artifactv2/artifact_test.go`
**Touches:** [model/Umpire/Artifact/Types.lean, model/Umpire/Artifact/Codecs.lean, model/Umpire/Artifact/Set.lean, model/Umpire/Planning/Tests/Artifacts.lean, tools/umpire/internal/artifactv2/artifact_test.go]

### Approach
- Store only checked sets in Lean semantic artifacts and render them through canonical projection; do not invent a Lean JSON decoder.
- Move malformed-list Lean assertions to strict set-admission tests where opacity makes invalid semantic Artifacts unconstructible.
- Retain the Go decoder's persisted malformed/order/duplicate/conflict/checksum matrix as the independent wire boundary.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact/Types.lean:60-75` — planning Artifact Known Gap field
- `model/Umpire/Artifact/Codecs.lean:120-150` — canonical encoder/checksum path
- `model/Umpire/Artifact/Set.lean:104-115` — repeated Lean set validation
- `model/Umpire/Planning/Tests/Artifacts.lean:139-182` — semantic Artifact mutations that opacity changes
- `tools/umpire/internal/artifactv2/artifact.go:580-610` — actual persisted-array admission
- `tools/umpire/internal/artifactv2/artifact_test.go` — Go negative-wire coverage
## Acceptance
- [ ] Lean semantic artifact models cannot carry an unchecked Known Gap list and no Lean decoder subsystem is added.
- [ ] Invalid Lean rows/order/duplicates/conflicts are covered at strict set admission; Go still rejects malformed, noncanonical, duplicate, conflicting, stale, and checksum-invalid persisted input.
- [ ] Valid planning Artifact bytes and checksums are unchanged.
- [ ] Focused Lean planning/set tests and Go artifactv2 tests pass.
## Done summary
Migrated planning `DrivePlan` artifacts to carry opaque checked `KnownGapSet` values, projected canonical rows only at encoding/list boundaries, removed redundant Lean set validation, and retained Go persisted-input rejection coverage. Canonical artifact bytes/checksums, aggregate builds, regression checks, and focused Lean/Go tests remain unchanged and pass.

baseline: focused gates green via handoff (verified at 34cabcc9 by fn-48-canonicalize-known-gaps-as-a-checked-set.1); `make lint-code` red pre-edit with the approved 1,379 inherited findings. Final `make lint-code` reproduced the exact category counts with zero findings in the task-touched Go file; its unrelated auto-fix side effect was restored.

verification environment: the exact Go gate passed after selecting `/usr/bin/clang` and a physical macOS `TMPDIR`, avoiding the inherited Lean-bundled Clang header lookup and `/var` symlink mismatch.

stage: impl-review - ran [2026-09-03T06:26:32Z..2026-09-03T06:31:59Z] (Codex SHIP after one NEEDS_WORK fix loop)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 82c97e4bd98d0cbf234d35871c591acafd18082c, f9556add4cfd06f82b041c2ea976465b5821c6dc
- Tests: cd model && mise exec -- lake build Umpire.Planning.Tests.KnownGaps Umpire.Planning.Tests.Artifacts Umpire.Artifact.Tests.Set, cd model && mise exec -- lake build Umpire.Planning.Tests.KnownGaps Umpire.Artifact.Tests.Codecs Umpire.Artifact.Tests.Runtime Umpire.Artifact.Tests.Evidence Umpire.Artifact.Tests.Result Temporal.Tool.RunEvaluationTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests, go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/runevaluation, make umpire-check-regression, make lint-model, make lint-code (waived inherited failure: exact 1,379 baseline findings; zero findings in tools/umpire/internal/artifactv2/artifact_test.go)
- PRs:
