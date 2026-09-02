---
satisfies: [R2]
---
# fn-5-umpire-discovery-promotion-and-artifact.6 Compile the single duplicate-delivery proposal

## Description
Expose the fixed inert binding through one effect-thin executable and prove its exact review-only
proposal bytes compile without modifying the source tree.

**Size:** M
**Files:** `model/Temporal/Tool/Promote.lean`, `model/Temporal/Tool/PromoteTests.lean`, `model/TemporalExperimentalTests.lean`, `model/lakefile.toml`
**Touches:** [model/Temporal/Tool/Promote.lean, model/Temporal/Tool/PromoteTests.lean, model/TemporalExperimentalTests.lean, model/lakefile.toml]

### Approach

- Add `temporal-model-promote` with one accepted argument: the fixed candidate identity from task `.5`.
- Emit one canonical `umpire-promotion-proposal/v2` envelope that separately contains the unchanged
  base Query/PlannerRun/base-`ExperimentSpec` lineage, selected fault-bearing `ExperimentSpec`
  identity/checksum, promoted identities, compiled-source identity/SHA, and exact source bytes.
- Reject missing/extra/unknown arguments, either lineage drift, unsealed source, serialization drift,
  and elaboration drift with empty stdout, one structured diagnostic plus one LF, and status 1.
- State in the envelope contract and diagnostics that direct invocation is inert model compilation,
  not a runtime reproduction/minimization/Exact-Replay eligibility claim.
- Compile the emitted bytes in an isolated focused Lake fixture and assert the command never creates,
  overwrites, or edits a Lean source or generated file.

### Non-goals

- No broad stable regression set, general artifact evolution, destination path, automatic install, runtime eligibility claim, or multiple-candidate command surface.

### Investigation targets

**Required:**
- `model/Temporal/Tool/Inspect.lean` — current effect-thin result and diagnostic conventions.
- `model/Temporal/Tool/InspectTests.lean` — stdout/stderr/status fixture style.
- `model/Temporal/Tool/PromotionBinding.lean` — task `.5` exact binding.
- `model/lakefile.toml` — executable and aggregate target registration.
- `.plans/LEAN_GUIDELINES.md` — clean elaboration and deterministic source constraints.

### Quick command

`cd model && mise exec -- lake build Temporal.Tool.PromoteTests TemporalExperimentalTests temporal-model-promote`

## Acceptance
- [ ] The executable accepts only the one fixed candidate identity and emits one canonical inert proposal envelope plus one LF.
- [ ] The envelope separately binds every base-plan, fault-bearing ExperimentSpec, promoted-source identity, SHA-256, and exact source byte required by fn-22 validation.
- [ ] Missing, extra, unknown, stale/crossed base-or-fault lineage, unsealed, noncanonical, or non-elaborating input yields status 1, empty stdout, and one exact diagnostic.
- [ ] Direct invocation makes no reproduction, minimization, Exact Replay, or runtime eligibility claim; fn-22 owns those checks and the final runtime cross-binding.
- [ ] Emitted proposal source compiles in a clean focused Lake fixture and repeated invocations are byte-identical.
- [ ] The command performs no source-tree, fixture, documentation, or generated-file write.
- [ ] Existing comments in touched files are preserved.

## Done summary
Implemented the effect-thin fixed-candidate `temporal-model-promote` command with a sealed canonical v2 proposal, exact lineage/source bindings, structured failures, and no runtime-eligibility claim. Added a complete pinned golden and executable regression covering exact streams/status, byte stability, isolated source elaboration, and repository non-mutation; corrected the fn-5 Quick target typo under the user's standing replan authorization.

Verification passed for the corrected parent Lean Quick, task Quick plus executable regression, `make lint-model`, and `git diff --check`. `make umpire-check-regression` remains red only at the pre-edit `KnownGaps.lean:296` vocabulary finding, and `make lint-code` remains at the pre-existing 1373 repository findings.

stage: impl-review - ran [2026-09-02T16:55:48.233488Z..2026-09-02T17:09:52.829756Z]
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: tracker-sync - skipped(config: tracker inactive)
## Evidence
- Commits: dc73777924f1d51bcda5f19a335d7224a606f39c, 360564640827c87a0d256d0d91ab4baa15f78727, a7dd2fa3425ea6391dd1ea2ec3252d8ab55b2183
- Tests: baseline: red (cd model && mise exec -- lake build Temporal.Tool.NexusDiscoveryTests Temporal.Tool.PromotionTests TemporalExperimentalTests temporal-model-inspect temporal-model-promote failed pre-edit: future temporal-model-promote target absent; terminal run also exposed inherited nonexistent Temporal.Tool.PromotionTests target), cd model && mise exec -- lake build Temporal.Tool.NexusDiscoveryTests Temporal.Tool.PromotionBindingTests TemporalExperimentalTests temporal-model-inspect temporal-model-promote, cd model && mise exec -- lake build Temporal.Tool.PromoteTests TemporalExperimentalTests temporal-model-promote temporal-model-promote-tests && mise exec -- lake -q exe temporal-model-promote-tests, make lint-model, make umpire-check-regression (inherited failure: model/Umpire/SemanticInventory/KnownGaps.lean:296 forbidden Temporal-owned vocabulary; all preceding checks green), make lint-code (inherited failure: 1373 pre-existing repository findings), git diff --check
- PRs: