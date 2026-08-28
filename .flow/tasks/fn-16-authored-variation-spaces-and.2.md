---
satisfies: [R2, R5, R6, R8]
---
# fn-16-authored-variation-spaces-and.2 Add the checked artifact-intent planning seam

## Description
Add one checked intent projection seam that can populate the existing DrivePlan choice/variant/fault fields after ordinary target-owned selection for R2/R5/R6/R8. Keep the current `plan` entry point and its bytes unchanged.

**Size:** M
**Files:** `model/Umpire/Artifact.lean`, `model/Umpire/Planning/Engine.lean`, `model/Umpire/Space/Intent.lean`, `model/Umpire/Space/Tests/Intent.lean`, `model/Umpire/Planning/Tests/Artifacts.lean`
**Touches:** [model/Umpire/Artifact.lean, model/Umpire/Planning/Engine.lean, model/Umpire/Space/Intent.lean, model/Umpire/Space/Tests/Intent.lean, model/Umpire/Planning/Tests/Artifacts.lean]

### Approach
- Define the narrow checked `ArtifactIntent` representation for axis choices, role variants, fault-to-authored-occurrence references, and extra capabilities.
- Refactor artifact construction behind an intent-aware deep seam while retaining the existing empty-intent facade and comments.
- Resolve selected fault occurrence IDs against the planner-produced linear extension and reject missing/stale/mismatched occurrences.
- Recompute DrivePlan and ExperimentSpec semantic identities after canonical intent projection.
- Add byte-for-byte ordinary-plan regression tests and malformed-intent fixtures.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact.lean:36-80` — existing reserved fields and explicit semantic wall
- `model/Umpire/Artifact.lean:311-382` — single artifact construction seam
- `model/Umpire/Planning/Engine.lean:432-470` — unchanged pure planner facade
- `model/Umpire/Planning/Tests/Artifacts.lean:22-68` — artifact assertions
- `model/Umpire/Behavior/Language.lean:73-105` — named occurrence identity contract

### Acceptance
- [ ] Checked intent produces the exact parent-specified `SemanticValue` arrays and capability union.
- [ ] Ordinary `plan` artifacts remain byte-identical and continue to emit empty reserved arrays.
- [ ] Missing/mismatched occurrences, duplicate intent entries, invalid capability references, or identity drift fail with no projected artifact.
- [ ] Model actions/outcomes/states/checkpoints remain exactly those emitted by the ordinary planner.
- [ ] Existing comments remain intact.

## Acceptance
- [ ] Intent projection populates only existing v1 fields and recomputes identities.
- [ ] Ordinary planning bytes are unchanged.
- [ ] Invalid intent fails closed without changing target-owned semantics.

## Done summary
Added a checked, identity-bound ArtifactIntent seam that canonicalizes axis choices, role variants, fault occurrence references, and capability requirements; applies intent only after ordinary target-owned planning; validates source Artifact checksums; and recomputes both Artifact checksums. Added focused regressions for exact arrays, ordinary byte identity, repeated semantic values across distinct roles, malformed intent, duplicate bindings, identity drift, and stale checksums.

Baseline and verification passed for the implemented intent/artifact/validation/Switch targets, aggregate model suites, and regression smoke. The cumulative Compilation, Metadata, and Temporal variation-space targets remain inherited expected pre-feature missing targets assigned to tasks .4, .3, and .5; no scope-violating stubs were added. Review found and fixed repeated-value erasure and stale-checksum admission, then returned SHIP; memory capture was non-blockingly skipped because flow memory is not initialized.

stage: impl-review - ran [2026-08-28T00:44:11Z..2026-08-28T00:52:57.670368Z]
## Evidence
- Commits: 4f086627594b052c124e295f308acb398db6b7b2, aa8f40ddfe14c2c6696b917e2df35a155ceedeb6
- Tests: cd model && mise exec -- lake build Umpire.Space.Tests.Intent Umpire.Planning.Tests.Artifacts Umpire.Space.Tests.Validation Umpire.Examples.SwitchTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression
- PRs: