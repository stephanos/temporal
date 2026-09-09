---
satisfies: [R2]
---
# fn-82-unify-the-umpire-and-testpilot.2 First rename: Machine, Step, Fact kinds, and the goldens writer

## Description
The early proof point (spec §Early proof point). The first rename changes Definition ID kinds,
so it exercises the whole regenerate-never-edit loop: `DefinitionKind.kernel` to `machine`,
`.observation` to `.fact`, `TransitionKernel` to `Machine`, `TransitionResult` to `Step` with
fields `outcome`, `state`, `facts`, the matching Provenance enum, a writer for the two golden
families that have none, and regeneration of every fingerprint, golden, and Case fixture (R2).

**Size:** M (mechanical sweep across many files)
**Files:** `model/Umpire/Core.lean`, `model/Umpire/Target/*.lean`, `model/Umpire/Case.lean`, `model/Umpire/Case/Provenance.lean`, new `model/Umpire/Goldens.lean` + `lakefile.lean` exe `umpire-goldens`, `Makefile`, every `.lean`/`.md` that spells the old names, all fixture directories, `tools/umpire/internal/retiredvocabulary/check.go`
**Touches:** [model/**, tools/umpire/internal/retiredvocabulary/check.go, tools/umpire/regression/switch_generated_view_test.go, tests/testcore/testpilot/testdata/**, common/testing/testpilot/testdata/**, Makefile, .flow/specs/*.md]

### Approach
- Rename in `model/Umpire/Core.lean`: `DefinitionKind.kernel`/`.observation` and their `.name` strings (`:61-93`), `TransitionKernel` (`:379`) to `Machine`, `TransitionResult` abbrev (`:219`) to `Step` with `modelOutcome`/`resultingState`/`observations` becoming `outcome`/`state`/`facts` (the JSON keys at `Umpire/Target/Projection.lean:197` follow). Keep `ModelValue`.
- Mirror in `model/Umpire/Case.lean:23-45` and `model/Umpire/Case/Provenance.lean:17-38`: `CASE_DEFINITION_KIND_MACHINE`, `_FACT`; drop the five dead kinds from both enums in this task so fixtures regenerate once.
- Add `umpire-goldens`: one Lake executable that writes `model/Umpire/Target/Tests/Compatibility/Fixtures/{TestTargetBehaviorFingerprint.txt,TestTargetCanonicalMetadata.json}` and the six files under `model/Temporal/Feature/Nexus/Fixtures/` from the same values their tests compute (`Compatibility/BehaviorFingerprint.lean:11-16`, `Operations/*Tests.lean:11`, `PlanningTests.lean:202-208`). Add an `umpire-check-regression` step that renders into a temporary root and `diff -r`s, following the regression-views block at `Makefile:1050-1069`.
- Regenerate in order: `lake exe umpire-goldens`, `make umpire-gen-regression-views`, `make umpire-gen-case-runtime-conformance`, `make umpire-gen-semantic-inventory`. Re-baseline `#guard_msgs` blocks by running the modules, never by editing expected text.
- Add `TransitionKernel`, `TransitionResult`, `modelOutcome`, `resultingState`, `CASE_DEFINITION_KIND_KERNEL`, `CASE_DEFINITION_KIND_OBSERVATION` to the gate; respell every scanned `.md` and open spec in the same commit.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:61-93,156-310,379-430` — kinds, trace step, kernel
- `model/Umpire/Case/Provenance.lean:17-38` — encoded kind strings (opaque to Go)
- `model/Umpire/Target/Tests/Compatibility/BehaviorFingerprint.lean:11-16` — how the golden is consumed
- `Makefile:1050-1069,1172-1180` — regression views diff block and the aggregate check
- `model/Temporal/System/Nexus/Core.lean:85-89,478-493` — `canonicalBehavior` literals that must survive unchanged here

**Optional** (reference as needed):
- `model/Umpire/Examples/Switch.lean` — the reference authored model; every rename must keep it building
- `tools/umpire/cmd/umpire-gen-regression-views/catalog.go:10` — `switchIdentity` (a `query` kind, unchanged)

### Key context
- `Umpire.Value` and `Umpire.Operation` are fn-77's; do not rename anything there.
- Provenance bytes are opaque to Go, so the Case fixtures change bytes but no Go code changes.
- If a golden cannot be regenerated from semantic inputs, stop and report per the spec's early proof point.

## Acceptance
- [ ] `DefinitionKind` and `Umpire.Provenance`-side kinds have `machine` and `fact`, no `kernel`/`observation`, and no `experimentSpace`/`variationAxis`/`choice`/`fault`/`coverageGoal`
- [ ] `Machine` and `Step` replace `TransitionKernel` and `TransitionResult`; `Step` fields are `outcome`, `state`, `facts`
- [ ] `lake exe umpire-goldens` writes both golden families and `make umpire-check-regression` diffs them against a temporary render
- [ ] All goldens, views, inventory, and the sixteen Case fixtures are regenerated, not hand-edited, and every make check plus `lake build Umpire UmpireTests Temporal TemporalModelTests TestpilotTests` passes
- [ ] The retired gate rejects the six old tokens and passes on the tree


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
