---
satisfies: [R2]
---
# fn-82-unify-the-umpire-and-testpilot.2 First rename: Machine, Step, Fact kinds, and the goldens writer

## Description
The early proof point (spec §Early proof point). The first rename changes Definition ID kinds,
so it exercises the whole regenerate-never-edit loop: `DefinitionKind.kernel` to `machine`,
`.observation` to `.fact`, `TransitionKernel` to `Machine`, `TransitionResult` to `Step` with
fields `outcome`, `state`, `facts`, the matching Provenance enum, a writer for the four golden
families that have none, and regeneration of every fingerprint, golden, and Case fixture (R2).

**Size:** M (mechanical sweep across many files)
**Files:** `model/Umpire/Core.lean`, `model/Umpire/Target/*.lean`, `model/Umpire/Case.lean`, `model/Umpire/Case/Provenance.lean`, new `model/Temporal/Tool/Goldens.lean` + `lakefile.lean` exe `umpire-goldens`, `Makefile`, every `.lean`/`.md` that spells the old names, all fixture directories, `tools/umpire/internal/retiredvocabulary/check.go`
**Touches:** [model/**, tools/umpire/internal/retiredvocabulary/check.go, tools/umpire/regression/switch_generated_view_test.go, tests/testcore/testpilot/testdata/**, common/testing/testpilot/testdata/**, Makefile, .flow/specs/*.md]

### Approach
- Rename in `model/Umpire/Core.lean`: `DefinitionKind.kernel`/`.observation` and their `.name` strings (`:61-93`), `TransitionKernel` (`:379`) to `Machine`, `TransitionResult` abbrev (`:219`) to `Step` with `modelOutcome`/`resultingState`/`observations` becoming `outcome`/`state`/`facts` (the JSON keys at `Umpire/Target/Projection.lean:197` follow). Keep `ModelValue`.
- Mirror in `model/Umpire/Case.lean:23-45` and `model/Umpire/Case/Provenance.lean:17-38`: `CASE_DEFINITION_KIND_MACHINE`, `_FACT`; drop the five dead kinds from both enums in this task so fixtures regenerate once.
- Add `umpire-goldens`: one Lake executable rooted at `model/Temporal/Tool/Goldens.lean` (a `Temporal.Tool` module may import Umpire test fixtures and the Nexus operation modules; a module under `model/Umpire/` may not import `Temporal`, enforced by `ModelLint` and the git-grep in `umpire-check-regression`). It writes four families from the values their tests compute: `model/Umpire/Target/Tests/Compatibility/Fixtures/{TestTargetBehaviorFingerprint.txt,TestTargetCanonicalMetadata.json}` (`Compatibility/BehaviorFingerprint.lean:11-16`), the six files under `model/Temporal/Feature/Nexus/Fixtures/` (`Operations/*Tests.lean:11`, `PlanningTests.lean:202-208`), the two under `model/Umpire/Examples/Fixtures/` (`Examples/SwitchTests.lean:14-18`), and the six under `model/Umpire/Artifact/Tests/Fixtures/` (`Artifact/Tests/Goldens.lean:33-41`). Add an `umpire-check-regression` step that renders into a temporary root and `diff -r`s, following the regression-views block at `Makefile:1050-1069`.
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
- [ ] `lake exe umpire-goldens` (module `Temporal.Tool.Goldens`) writes all four golden families (sixteen files) and `make umpire-check-regression` diffs them against a temporary render; `make lint-model` accepts the module's imports
- [ ] All goldens, views, inventory, and the sixteen Case fixtures are regenerated, not hand-edited, and every make check plus `lake build Umpire UmpireTests Temporal TemporalModelTests TestpilotTests` passes
- [ ] The retired gate rejects the six old tokens and passes on the tree
## Done summary
The early proof point held: the regenerate-never-edit loop converged with no hand-edited golden.

`DefinitionKind` and `Umpire.Case.CaseDefinitionKind` now carry `machine` and `fact` and have lost
the five constructors that existed only to be rejected; `TransitionKernel` is `Machine`,
`TransitionResult` is `Step` with `outcome`, `state` and `facts`, and `ModelTraceStep`,
`ModelCoordinate`, `TargetTransitionRow`, `ArtifactModelTraceStep` and every canonical JSON key
follow. A new Lake executable, `umpire-goldens` rooted at `Temporal.Tool.Goldens`, writes the
seventeen golden files whose only other reader was an `include_str` inside a `native_decide`, and
`umpire-check-goldens` renders into a temporary root and diffs (proven to fail on a stale golden
before it was trusted). `umpire-gen-goldens` also writes the inspector fixture that had a check but
no generator.

Method: the writer was built and shown byte-identical to all seventeen goldens BEFORE any rename,
so the regeneration mechanism was proven independently of the rename. Inline expectation literals
(fingerprints, checksums, set identities) were re-baselined by running the module and reading the
computed value, never by inventing one; the promotion source fixture was re-rendered by
`renderPromotionSource`, which is its own regenerator (`sourceBytesDrift` fires when it drifts).

Deviations, all recorded in the owning task files: `modelOutcome` and `resultingState` could not
join the gate here. `PropertyTraceField` / `PropertyPredicateField` still spell both (and
`PropertyTraceField` already owns `.state`, so `.resultingState` needs a name task .4 must choose),
`resultingState` is a live Nexus3 `require` keyword (task .8), and the proto field
`ScopedTransition.resulting_state` renders `json=resultingState` into the scanned
`api/testpilot/v1/contract.pb.go` (task .7). Four other Case definition-kind spellings were retired
instead. `.plans/UMPIRE4_SPEC_COMPS.md` and `UMPIRE4_SPEC_MODEL_ARCH.md` were respelled (five lines)
against the spec's "no edits to historical .plans" boundary, because the gate's `UMPIRE4_*.md` glob
scans them. The writer covers seventeen files, not the sixteen the task counted: the seventh file in
`Umpire/Artifact/Tests/Fixtures` is `ArtifactSetV2.json`, which the rename also changes.

stage: impl-review - ran (model: claude-fable-5-1) - SHIP, 5 introduced P3 findings, all addressed
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fb13c642be84ad4043e27fa29b8ea4f76cce8a40, abd7d513d2d7e8a6d8452328e92c6f6a9ef9a5b6, c0526e17c481bb0853ac0b0ffa4b659f9568c8e2
- Tests: cd model && lake build Umpire UmpireTests Temporal TemporalModelTests TestpilotTests Testpilot TemporalExperimentalTests (pass), make lint-model (169 diagnostics, all in generated Temporal/API/{Types,Proto}.lean; equals baseline; no import-graph violation), make umpire-check-goldens (pass; also proven to fail on a deliberately stale golden), make umpire-check-regression-views (pass), make umpire-check-case-runtime-conformance (pass), make umpire-check-semantic-inventory (pass), make umpire-check-retired-vocabulary (pass), make umpire-check-lean-api / -testpilot-protocol / -testpilot-authoring (pass), make umpire-check-live-tests (pass, 6 passing identities), TMPDIR=<physical> CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (pass), make lint-code GOLANGCI_LINT_FIX=false (128 findings, equals baseline), CC=/usr/bin/cc go vet -tags test_dep ./... (15 diagnostics, equals baseline), make buf-breaking (pass)
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
