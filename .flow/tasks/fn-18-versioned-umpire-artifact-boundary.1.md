---
satisfies: [R2, R8]
---
# fn-18-versioned-umpire-artifact-boundary.1 Adopt the v2 Artifact baseline and vertical package

## Description
Place fn-37's canonical v2 DrivePlan and ExperimentSpec behind the vertical Artifact facade without changing bytes or introducing a second format.


**Size:** M
**Files:** `model/Umpire/Artifact.lean`, `model/Umpire/Artifact/**`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Artifact.lean, model/Umpire/Artifact/**, model/UmpireTests.lean]

### Approach
- Preserve the existing declarations and comments while moving implementation behind focused modules.
- Reuse Definition IDs, Behavior Fingerprints, Artifact Checksums, Limits, and Known Gaps exactly as fn-37 defines them.
- Keep `umpire-drive-plan/v2` and `umpire-experiment/v2` as the sole supported current formats.
- Add no reader, migration, alias, or fallback for an earlier prototype format.

### Investigation targets
**Required:** fn-37 Artifact modules and the parent v2-baseline contract.

## Acceptance
- [ ] Existing v2 canonical bytes and Artifact Checksums remain byte-identical.
- [ ] Public imports expose one vertical Artifact package with comments preserved.
- [ ] No earlier-format reader, alternate writer, compatibility alias, or inferred missing intent exists.

### Quick command

`cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs`

## Done summary
Established the vertical v2 Artifact package, removed the alternate v3 writer/domain, and restored strict v2 generator admission. Compact fn-37 wire bytes remain exact while separately pretty fixtures are preserved and checked by JSON semantics; final focused, direct, and aggregate builds are green.

Phase5 gate receipt minting was non-blockingly unavailable because protected external config/development.yaml status keeps the shared worktree dirty; the exact Quick command itself exited zero.

stage: impl-review - ran [2026-08-28T14:36:38Z..2026-08-28T14:54:32Z] (NEEDS_WORK -> NEEDS_WORK -> SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: adb54774e08e769325bd1f99b0a61188cc16242f, fd84945b84e5ede63c22a1b18f27d689a39ab129, 665c9bd4f0151b02094661313156e4cdc5b83be3, b19fe6c2e9272cb15f769a563193b865ef40ff71
- Tests: baseline: red (cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs failed pre-edit because the task-owned target did not yet exist), cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs, cd model && mise exec -- lake build Umpire.ExecutionHandoffTests Umpire.ImportTests Temporal.Tool.GenerateTestsTests Umpire.Planning.Tests.Artifacts, cd model && mise exec -- lake build Umpire.Artifact.Tests.Codecs Temporal.Tool.GenerateTestsTests Umpire.ExecutionHandoffTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests Umpire.Examples.SwitchTests Umpire.Tests.MigrationCompatibility, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests umpire-gen-tests-tests, cmp <(git show f9df288a3adf253b6432f2fdf8e1a4479866c468:model/Umpire/Examples/testdata/switch-experiment-spec.json) model/Umpire/Artifact/Tests/Fixtures/SwitchExperimentSpecV2CanonicalBytes.json
- PRs:
