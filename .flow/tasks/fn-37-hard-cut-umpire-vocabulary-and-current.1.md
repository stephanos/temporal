---
satisfies: [R2]
---
# fn-37-hard-cut-umpire-vocabulary-and-current.1 Establish typed fingerprint and checksum primitives

## Description
Build the isolated hashing foundation required by R2 before any public rename depends on it. Keep the interface small and pure so every model language and artifact encoder shares one implementation.

**Size:** M
**Files:** `model/Umpire/Fingerprint.lean`, `model/Umpire/FingerprintTests.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Fingerprint.lean, model/Umpire/FingerprintTests.lean, model/UmpireTests.lean]

### Approach
- Add a pure Lean SHA-256 implementation or an existing pure library dependency only if it does not pull IO or runtime concerns into Umpire.
- Expose distinct `BehaviorFingerprint` and `ArtifactChecksum` value types with validated `sha256:` lowercase-hex rendering.
- Centralize fixed domain tags and derivation operations; callers provide already-canonical content and cannot interchange result types.
- Test published SHA-256 vectors, UTF-8 input, empty input, malformed rendering, domain separation, and repeatability.
- Add a golden value that Go can independently recompute in Task `.6`.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:222-235` — current inline digest derivation to replace.
- `model/Umpire/Artifact.lean:285-355` — current artifact identity call sites.
- `tools/umpire/regression/projection.go:145-175` — current Go SHA-256 precedent.
- `tools/umpire/cmd/umpire-gen-lean-dynamic-config-catalog/project.go:810-830` — established Go checksum rendering.
- `model/Umpire/CoreTests.lean` — focused pure-model test style.

### Key context
The two values intentionally share a wire encoding but not a Lean type or derivation domain. Artifact Checksum input includes all canonical artifact content except its own field; Behavior Fingerprint input is supplied by owner modules after excluding non-behavioral source and documentation data.

## Acceptance
- [ ] Standard SHA-256 vectors and the cross-language golden value pass.
- [ ] Behavior Fingerprint and Artifact Checksum cannot be mixed without an explicit conversion, and no such conversion is public.
- [ ] Malformed checksum text and wrong-domain expectations fail focused tests.
- [ ] The module remains pure and independently buildable.

## Done summary
Implemented a pure, independently buildable SHA-256 foundation with sealed `BehaviorFingerprint` and `ArtifactChecksum` types, validated lowercase `sha256:` rendering, and fixed derivation domains for behavior, DrivePlan, and ExperimentSpec content. Focused tests cover published empty/single-block/multi-block vectors, UTF-8, malformed input, repeatability, type/domain separation, and cross-language golden values.

baseline: green for all currently available gates; `umpire-check-regression-views` is an inherited sequencing gap because fn37.6 owns creation of that renamed target, while its current projection equivalent passed through `umpire-check-regression`.

stage: impl-review - ran | verdict: SHIP | session: 01a043af-24eb-72f2-a3dd-4ad6c522a3ec

stage: plan-sync - skipped(config: planSync.enabled != true)

GATE_SKIPPED:build:green-receipt 87205aa9 - current task diff already passed the full Lean build
GATE_SKIPPED:unittest:green-receipt 87205aa9 - current task diff already passed pinned Go tests
GATE_SKIPPED:smoke:green-receipt 87205aa9 - current task diff already passed the regression check
GATE_SKIPPED:generated-view-smoke:inherited-sequencing-gap - fn37.6 creates the final `umpire-check-regression-views` target; the current projection check passed through `umpire-check-regression`
## Evidence
- Commits: 87205aa9206a122369f34e9a34c364677b0f2589
- Tests: cd model && mise exec -- lake build Umpire.FingerprintTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect, mise exec -- go test ./tools/umpire/..., mise exec -- make umpire-check-regression, GATE_SKIPPED:build:green-receipt 87205aa9 - current task diff already passed the full Lean build, GATE_SKIPPED:unittest:green-receipt 87205aa9 - current task diff already passed pinned Go tests, GATE_SKIPPED:smoke:green-receipt 87205aa9 - current task diff already passed the regression check, GATE_SKIPPED:generated-view-smoke:inherited-sequencing-gap - fn37.6 creates the final umpire-check-regression-views target; the current projection check passed through umpire-check-regression
- PRs:
