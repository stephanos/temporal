---
satisfies: [R3, R4, R5, R7]
---
# fn-41-finitemachine-target-authoring.4 Pin expert and cross-layer compatibility

## Description
Add the integration regressions that distinguish the ordinary adapter from the expert kernel route and prove both Nexus migrations are semantically inert (R3, R4, R5, R7). This task follows both migrations so it can compare the complete checked graph.

**Size:** M
**Files:** `model/Umpire/Examples/SwitchTests.lean`, `model/Umpire/Tests/MigrationCompatibility.lean`, `model/Temporal/System/Nexus/ImplementationLinkTests.lean`, `model/Temporal/ImplementationLinkTests/Nexus.lean`
**Touches:** [model/Umpire/Examples/SwitchTests.lean, model/Umpire/Tests/MigrationCompatibility.lean, model/Temporal/System/Nexus/ImplementationLinkTests.lean, model/Temporal/ImplementationLinkTests/Nexus.lean]

### Approach
- Add an explicit expert-path regression around Switch's independently authored initial/step propositions and two enumerated results; keep the production Switch model on direct `TransitionKernel` authoring.
- Reuse the existing Switch golden Query, plan, behavior fingerprint, and compiled Artifact fixtures to prove the new public adapter did not narrow or replace the expert seam.
- Extend focused System-to-Feature checks to consume the existing named initial/step authority lemmas and compare checked target identities/fingerprints and correspondence results after both migrations.
- Exercise downstream Query/planning/artifact paths through existing golden tests rather than introducing a second compatibility format or regenerating expected data.
- Do not unfold `FiniteMachine` internals in cross-family consumer proofs; add a missing public local rewrite/authority theorem in the owning model task if integration exposes one.
- Preserve all existing test comments and declarations while making new assertions descriptively named where practical.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Examples/Switch.lean:113-230` — independent authoritative relations and two-result enumerator.
- `model/Umpire/Examples/Switch.lean:367-395` — direct expert kernel planning and Target construction.
- `model/Umpire/Examples/SwitchTests.lean:1-101` — existing golden Query, Artifact, and planning checks.
- `model/Umpire/Tests/MigrationCompatibility.lean:200-380` — executable Switch compatibility matrix and fixtures.
- `model/Temporal/System/Nexus/ImplementationLink.lean:312-365` — production correspondence witness.
- `model/Temporal/ImplementationLinkTests/Nexus.lean:18-40` — focused named-authority correspondence assertions.

**Optional** (reference as needed):
- `model/Temporal/ImplementationLinkTests/Nexus.lean:274-315` — checked evidence identity/fingerprint assertions.

## Acceptance
- [ ] Switch remains implemented through direct `TransitionKernel`, and tests pin its independent authority, two-result behavior, exact Query/planner results, fingerprint, and Artifact bytes.
- [ ] Feature and System checked targets retain their identities/fingerprints and the existing Implementation Link witness/result remains valid.
- [ ] Cross-family tests use model-owned public authority theorems rather than unfolding adapter internals.
- [ ] No compatibility fixture is regenerated and no production behavior is changed to satisfy a test.
- [ ] Existing comments and unrelated worktree changes are preserved.
- [ ] `cd model && mise exec -- lake build Umpire.Examples.SwitchTests Umpire.Tests.MigrationCompatibility Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests` passes.

## Done summary
Pinned Switch's direct `TransitionKernel` expert path, independent authority, ordered two-result behavior, exact planner result, literal Behavior Fingerprint, and existing pretty Query/Artifact golden bytes. Added cross-layer regressions for the migrated Feature and System checked target identities, fingerprints, public authority seams, and unchanged Implementation Link result without changing production code or fixtures.

baseline: green (`cd model && mise exec -- lake build Umpire.Target.Tests.FiniteMachine Umpire.TargetTests Temporal.Feature.Nexus.LifecycleTests Temporal.System.Nexus.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests`, `make umpire-check-regression`, and `make lint-model` passed pre-edit)

verification: green (focused 78-job acceptance build, exact parent aggregate after isolated Lake cache warmup, 186-job canonical regression, and 158-job model lint all passed; gate receipts were non-warrantable only because of the inherited false symlink stat at `config/development.yaml`)

stage: impl-review - ran [2026-08-28T20:28:29Z..2026-08-28T20:30:12Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2eee759b77274bb9e7ac191b249e7979186c1f4d
- Tests: cd model && mise exec -- lake build Umpire.Examples.SwitchTests Umpire.Tests.MigrationCompatibility Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests, cd model && mise exec -- lake build Umpire.Target.Tests.Compatibility, cd model && mise exec -- lake build Umpire.Target.Tests.FiniteMachine Umpire.TargetTests Temporal.Feature.Nexus.LifecycleTests Temporal.System.Nexus.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests, make umpire-check-regression, make lint-model
- PRs:
