---
satisfies: [R2]
---
# fn-5-umpire-discovery-promotion-and-artifact.6 Compile the single duplicate-delivery proposal

## Description
Expose the eligibility-gated fixed binding through one effect-thin executable and prove its exact
review-only proposal bytes compile without modifying the source tree.

**Size:** M
**Files:** `model/Temporal/Tool/Promote.lean`, `model/Temporal/Tool/PromoteTests.lean`, `model/TemporalExperimentalTests.lean`, `model/lakefile.toml`
**Touches:** [model/Temporal/Tool/Promote.lean, model/Temporal/Tool/PromoteTests.lean, model/TemporalExperimentalTests.lean, model/lakefile.toml]

### Approach

- Add `temporal-model-promote` with no arguments. Read exactly one canonical
  `umpire-reviewed-promotion-eligibility/v1` handoff from stdin, validate it through task `.5`, and
  make `CheckedPromotionEligibility` the only route to the fixed binding.
- Emit one canonical `umpire-promotion-proposal/v2` envelope containing original lineage, promoted
  identities, compiled-source identity/SHA, and exact source bytes, followed by one LF.
- Reject any argument, empty/bare-ID/malformed/noncanonical handoff, failed/incomplete/crossed gate,
  unsealed source, serialization drift, and elaboration drift with empty stdout, one structured
  diagnostic plus one LF, and status 1.
- Compile the emitted bytes in an isolated focused Lake fixture and assert the command never creates,
  overwrites, or edits a Lean source or generated file.

### Non-goals

- No broad stable regression set, general artifact evolution, destination path, automatic install, unchecked-ID mode, or multiple-candidate command surface.

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
- [ ] The executable accepts no arguments and emits one canonical proposal envelope plus one LF only from a valid checked eligibility handoff on stdin.
- [ ] The envelope binds every fixed original/promoted/source identity, SHA-256, and exact source byte required by fn-22 validation.
- [ ] Bare candidate identity, missing/extra arguments, malformed/noncanonical handoff, failed/incomplete/crossed receipts, stale lineage, unsealed source, or non-elaborating input yields status 1, empty stdout, and one exact diagnostic.
- [ ] Emitted proposal source compiles in a clean focused Lake fixture and repeated invocations are byte-identical.
- [ ] The command performs no source-tree, fixture, documentation, or generated-file write.
- [ ] Existing comments in touched files are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
